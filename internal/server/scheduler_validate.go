package server

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // ScheduleExpressionTimezone must resolve even in images without zoneinfo.
)

var (
	schedulerNamePattern   = regexp.MustCompile(`^[0-9a-zA-Z\-_.]{1,64}$`)
	schedulerTokenPattern  = regexp.MustCompile(`^[a-zA-Z0-9\-_]{1,64}$`)
	schedulerRolePattern   = regexp.MustCompile(`^arn:aws[a-z-]*:iam::\d{12}:role/[\w+=,.@/-]+$`)
	schedulerDLQPattern    = regexp.MustCompile(`^arn:aws[a-z-]*:sqs:[a-z0-9\-]+:\d{12}:[a-zA-Z0-9\-_]+$`)
	schedulerKMSPattern    = regexp.MustCompile(`^arn:aws[a-z-]*:kms:[a-z0-9\-]+:\d{12}:(key|alias)/[0-9a-zA-Z\-_]*$`)
	schedulerUniversalARN  = regexp.MustCompile(`^arn:aws[a-z-]*:scheduler:::aws-sdk:[a-z0-9\-]+:[a-zA-Z0-9]+$`)
	schedulerTemplatedARN  = regexp.MustCompile(`^arn:aws[a-z-]*:([a-z0-9\-]+):[a-z0-9\-]*:\d{12}:(.+)$`)
	schedulerRateExpr      = regexp.MustCompile(`^rate\((\d+) (minute|minutes|hour|hours|day|days)\)$`)
	schedulerAtExpr        = regexp.MustCompile(`^at\((\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\)$`)
	schedulerCronExpr      = regexp.MustCompile(`^cron\((.+)\)$`)
	schedulerCronFieldPart = regexp.MustCompile(`^(\*|\?|L|LW|[0-9A-Z]+(-[0-9A-Z]+)?|\*/\d+|[0-9A-Z]+(-[0-9A-Z]+)?/\d+|\d+W|\d*L|\d+#\d+)$`)
	// schedulerCronSpecial matches the ?, L, W and # forms allowed only in
	// day-of-month and day-of-week.
	schedulerCronSpecial = regexp.MustCompile(`^(\?|L|LW|\d+W|\d*L|\d+#\d+)$`)
	schedulerCronToken   = regexp.MustCompile(`[0-9A-Z]+`)
)

// schedulerTemplatedServices are the services a templated target ARN may name.
var schedulerTemplatedServices = map[string]bool{
	"codebuild": true, "codepipeline": true, "ecs": true, "events": true, "firehose": true,
	"inspector": true, "kinesis": true, "lambda": true, "sagemaker": true, "sns": true,
	"sqs": true, "states": true,
}

// schedulerTarget is the Target structure; unknown members are rejected when
// decoding, so nothing but model members is ever stored.
type schedulerTarget struct {
	Arn              *string `json:"Arn"`
	RoleArn          *string `json:"RoleArn"`
	DeadLetterConfig *struct {
		Arn *string `json:"Arn,omitempty"`
	} `json:"DeadLetterConfig,omitempty"`
	RetryPolicy           *schedulerRetryPolicy      `json:"RetryPolicy,omitempty"`
	Input                 *string                    `json:"Input,omitempty"`
	EcsParameters         map[string]json.RawMessage `json:"EcsParameters,omitempty"`
	EventBridgeParameters *schedulerEventParams      `json:"EventBridgeParameters,omitempty"`
	// The V2 PutEvents parameters share the DetailType/Source shape.
	EventBridgePutEventsV2Parameters  *schedulerEventParams      `json:"EventBridgePutEventsV2Parameters,omitempty"`
	EventBridgePutRawEventsParameters map[string]json.RawMessage `json:"EventBridgePutRawEventsParameters,omitempty"`
	KinesisParameters                 *struct {
		PartitionKey *string `json:"PartitionKey"`
	} `json:"KinesisParameters,omitempty"`
	SageMakerPipelineParameters map[string]json.RawMessage `json:"SageMakerPipelineParameters,omitempty"`
	SqsParameters               *struct {
		MessageGroupId *string `json:"MessageGroupId,omitempty"`
	} `json:"SqsParameters,omitempty"`
}

type schedulerRetryPolicy struct {
	MaximumEventAgeInSeconds *int64 `json:"MaximumEventAgeInSeconds,omitempty"`
	MaximumRetryAttempts     *int64 `json:"MaximumRetryAttempts,omitempty"`
}

type schedulerEventParams struct {
	DetailType *string `json:"DetailType"`
	Source     *string `json:"Source"`
}

type schedulerFlexibleWindow struct {
	Mode                   *string `json:"Mode"`
	MaximumWindowInMinutes *int64  `json:"MaximumWindowInMinutes,omitempty"`
}

func schedulerValidName(field, v string) *schedulerError {
	if !schedulerNamePattern.MatchString(v) {
		return schedulerValidation("%s must be 1-64 characters from [0-9a-zA-Z-_.]", field)
	}
	return nil
}

// schedulerValidateExpression checks at(), rate() and cron() expressions and
// the optional IANA timezone they are evaluated in.
func schedulerValidateExpression(expr, tz string) *schedulerError {
	if len(expr) == 0 || len(expr) > 256 {
		return schedulerValidation("ScheduleExpression must be 1-256 characters")
	}
	if tz != "" {
		if len(tz) > 50 {
			return schedulerValidation("ScheduleExpressionTimezone must be at most 50 characters")
		}
		if _, err := time.LoadLocation(tz); err != nil {
			return schedulerValidation("Invalid ScheduleExpressionTimezone %q", tz)
		}
	}
	switch {
	case strings.HasPrefix(expr, "rate("):
		m := schedulerRateExpr.FindStringSubmatch(expr)
		if m == nil {
			return schedulerValidation("Invalid Schedule Expression %s.", expr)
		}
		if n, err := strconv.ParseInt(m[1], 10, 32); err != nil || n < 1 {
			return schedulerValidation("Invalid Schedule Expression %s: the rate value must be a positive integer.", expr)
		}
	case strings.HasPrefix(expr, "at("):
		m := schedulerAtExpr.FindStringSubmatch(expr)
		if m == nil {
			return schedulerValidation("Invalid Schedule Expression %s.", expr)
		}
		if _, err := time.Parse("2006-01-02T15:04:05", m[1]); err != nil {
			return schedulerValidation("Invalid Schedule Expression %s: %s is not a valid date and time.", expr, m[1])
		}
	case strings.HasPrefix(expr, "cron("):
		m := schedulerCronExpr.FindStringSubmatch(expr)
		if m == nil {
			return schedulerValidation("Invalid Schedule Expression %s.", expr)
		}
		return schedulerValidateCron(expr, strings.Fields(m[1]))
	default:
		return schedulerValidation("Invalid Schedule Expression %s: use at(), rate() or cron().", expr)
	}
	return nil
}

// schedulerValidateCron checks the six AWS cron fields (minutes hours
// day-of-month month day-of-week year). Exactly one of day-of-month and
// day-of-week must be '?'.
// ponytail: checks syntax and numeric bounds per field, not every
// combination AWS rejects (e.g. W with ranges).
func schedulerValidateCron(expr string, fields []string) *schedulerError {
	bad := func(why string) *schedulerError {
		return schedulerValidation("Invalid Schedule Expression %s: %s", expr, why)
	}
	if len(fields) != 6 {
		return bad("cron expressions have six fields (minutes hours day-of-month month day-of-week year).")
	}
	if (fields[2] == "?") == (fields[4] == "?") {
		return bad("exactly one of day-of-month and day-of-week must be '?'.")
	}
	limits := []struct {
		lo, hi int64
		names  []string
	}{
		{0, 59, nil}, {0, 23, nil}, {1, 31, nil},
		{1, 12, []string{"JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC"}},
		{1, 7, []string{"SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"}},
		{1970, 2199, nil},
	}
	for i, field := range fields {
		for _, part := range strings.Split(field, ",") {
			if !schedulerCronFieldPart.MatchString(part) {
				return bad(fmt.Sprintf("field %d has an invalid value %q.", i+1, part))
			}
			if schedulerCronSpecial.MatchString(part) && i != 2 && i != 4 {
				return bad(fmt.Sprintf("field %d does not allow %q.", i+1, part))
			}
			for _, tok := range schedulerCronToken.FindAllString(strings.Split(part, "/")[0], -1) {
				if tok == "L" || tok == "LW" || tok == "W" {
					continue
				}
				// 15W and 5L carry a number before the suffix; names like JUL don't.
				if len(tok) > 1 && tok[0] >= '0' && tok[0] <= '9' {
					tok = strings.TrimRight(tok, "WL")
				}
				if n, err := strconv.ParseInt(tok, 10, 32); err == nil {
					if n < limits[i].lo || n > limits[i].hi {
						return bad(fmt.Sprintf("field %d value %d is outside %d-%d.", i+1, n, limits[i].lo, limits[i].hi))
					}
					continue
				}
				found := false
				for _, name := range limits[i].names {
					found = found || name == tok
				}
				if !found {
					return bad(fmt.Sprintf("field %d has an invalid value %q.", i+1, tok))
				}
			}
		}
	}
	return nil
}

func schedulerValidateWindow(w *schedulerFlexibleWindow) *schedulerError {
	if w == nil || w.Mode == nil {
		return schedulerValidation("FlexibleTimeWindow.Mode is required")
	}
	switch *w.Mode {
	case "OFF":
		if w.MaximumWindowInMinutes != nil {
			return schedulerValidation("MaximumWindowInMinutes must not be set when FlexibleTimeWindow.Mode is OFF")
		}
	case "FLEXIBLE":
		if w.MaximumWindowInMinutes == nil || *w.MaximumWindowInMinutes < 1 || *w.MaximumWindowInMinutes > 1440 {
			return schedulerValidation("MaximumWindowInMinutes must be 1-1440 when FlexibleTimeWindow.Mode is FLEXIBLE")
		}
	default:
		return schedulerValidation("FlexibleTimeWindow.Mode must be OFF or FLEXIBLE")
	}
	return nil
}

// schedulerValidateTarget checks the target ARN kind and that only the
// parameter block matching it is set (and is set when the target needs it).
func schedulerValidateTarget(t *schedulerTarget) *schedulerError {
	if t == nil || t.Arn == nil || t.RoleArn == nil {
		return schedulerValidation("Target.Arn and Target.RoleArn are required")
	}
	arn := *t.Arn
	if len(arn) == 0 || len(arn) > 1600 {
		return schedulerValidation("Target.Arn must be 1-1600 characters")
	}
	if !schedulerRolePattern.MatchString(*t.RoleArn) || len(*t.RoleArn) > 1600 {
		return schedulerValidation("Target.RoleArn must be an IAM role ARN")
	}
	if t.DeadLetterConfig != nil && t.DeadLetterConfig.Arn != nil && !schedulerDLQPattern.MatchString(*t.DeadLetterConfig.Arn) {
		return schedulerValidation("DeadLetterConfig.Arn must be an SQS queue ARN")
	}
	if p := t.RetryPolicy; p != nil {
		if v := p.MaximumEventAgeInSeconds; v != nil && (*v < 60 || *v > 86400) {
			return schedulerValidation("MaximumEventAgeInSeconds must be 60-86400")
		}
		if v := p.MaximumRetryAttempts; v != nil && (*v < 0 || *v > 185) {
			return schedulerValidation("MaximumRetryAttempts must be 0-185")
		}
	}
	if t.Input != nil && *t.Input == "" {
		return schedulerValidation("Target.Input must not be empty")
	}
	set := map[string]bool{
		"ecs": t.EcsParameters != nil, "events": t.EventBridgeParameters != nil || t.EventBridgePutEventsV2Parameters != nil || t.EventBridgePutRawEventsParameters != nil,
		"kinesis": t.KinesisParameters != nil, "sagemaker": t.SageMakerPipelineParameters != nil, "sqs": t.SqsParameters != nil,
	}
	service := ""
	if schedulerUniversalARN.MatchString(arn) {
		if t.Input == nil || !json.Valid([]byte(*t.Input)) {
			return schedulerValidation("Universal targets require a JSON Input")
		}
	} else {
		m := schedulerTemplatedARN.FindStringSubmatch(arn)
		if m == nil || !schedulerTemplatedServices[m[1]] {
			return schedulerValidation("Invalid target ARN %s: use a templated target or arn:aws:scheduler:::aws-sdk:<service>:<apiAction>", arn)
		}
		service = m[1]
	}
	for svc, has := range set {
		if has && svc != service {
			return schedulerValidation("Target parameters for %s can't be used with target %s", svc, arn)
		}
	}
	switch service {
	case "ecs":
		return schedulerValidateECS(t.EcsParameters)
	case "events":
		n := 0
		for _, p := range []*schedulerEventParams{t.EventBridgeParameters, t.EventBridgePutEventsV2Parameters} {
			if p == nil {
				continue
			}
			n++
			if p.DetailType == nil || p.Source == nil || *p.DetailType == "" || len(*p.DetailType) > 128 || *p.Source == "" || len(*p.Source) > 256 {
				return schedulerValidation("EventBridge parameters need DetailType (1-128) and Source (1-256)")
			}
			if strings.HasPrefix(*p.Source, "aws.") {
				return schedulerValidation("EventBridge Source must not start with aws.")
			}
		}
		if t.EventBridgePutRawEventsParameters != nil {
			n++
		}
		if n != 1 {
			return schedulerValidation("EventBridge targets need exactly one of EventBridgeParameters, EventBridgePutEventsV2Parameters and EventBridgePutRawEventsParameters")
		}
	case "kinesis":
		if t.KinesisParameters == nil || t.KinesisParameters.PartitionKey == nil || *t.KinesisParameters.PartitionKey == "" || len(*t.KinesisParameters.PartitionKey) > 256 {
			return schedulerValidation("Kinesis targets need KinesisParameters.PartitionKey (1-256)")
		}
	case "sqs":
		fifo := strings.HasSuffix(arn, ".fifo")
		var group *string
		if t.SqsParameters != nil {
			group = t.SqsParameters.MessageGroupId
		}
		if fifo && (group == nil || *group == "") {
			return schedulerValidation("FIFO queue targets need SqsParameters.MessageGroupId")
		}
		if group != nil && (len(*group) == 0 || len(*group) > 128) {
			return schedulerValidation("MessageGroupId must be 1-128 characters")
		}
	}
	return nil
}

func schedulerValidateECS(p map[string]json.RawMessage) *schedulerError {
	var def string
	if err := json.Unmarshal(p["TaskDefinitionArn"], &def); err != nil || def == "" || len(def) > 1600 {
		return schedulerValidation("EcsParameters.TaskDefinitionArn is required")
	}
	if raw, ok := p["TaskCount"]; ok {
		var n int64
		if err := json.Unmarshal(raw, &n); err != nil || n < 1 || n > 10 {
			return schedulerValidation("EcsParameters.TaskCount must be 1-10")
		}
	}
	if raw, ok := p["LaunchType"]; ok {
		var lt string
		if err := json.Unmarshal(raw, &lt); err != nil || (lt != "EC2" && lt != "FARGATE" && lt != "EXTERNAL") {
			return schedulerValidation("EcsParameters.LaunchType must be EC2, FARGATE or EXTERNAL")
		}
	}
	return nil
}
