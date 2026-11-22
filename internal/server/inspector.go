package server

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
	"github.com/stackyard/stackyard/internal/services/ec2"
)

// Amazon Inspector Classic (JSON 1.1, X-Amz-Target InspectorService.*,
// SigV4 name "inspector"). Inspector v2 is a separate REST service
// (inspectorv2.go, signing name "inspector2").

const (
	iscTargetPrefix = "InspectorService."
	iscMaxBody      = 1 << 20
	iscAccount      = "123456789012"
	iscRegion       = "us-east-1"
)

type iscError struct {
	code      string
	msg       string
	errorCode string
	extra     map[string]any
}

func (e *iscError) status() int {
	if e.code == "InternalException" {
		return http.StatusInternalServerError
	}
	return http.StatusBadRequest
}

func iscInvalid(errorCode, format string, args ...any) *iscError {
	return &iscError{code: "InvalidInputException", errorCode: errorCode, msg: fmt.Sprintf(format, args...)}
}

func iscNoSuch(errorCode, format string, args ...any) *iscError {
	return &iscError{code: "NoSuchEntityException", errorCode: errorCode, msg: fmt.Sprintf(format, args...)}
}

func iscLimit(errorCode, format string, args ...any) *iscError {
	return &iscError{code: "LimitExceededException", errorCode: errorCode, msg: fmt.Sprintf(format, args...)}
}

func iscInProgress(arns []string) *iscError {
	return &iscError{code: "AssessmentRunInProgressException", msg: "one or more assessment runs are still in progress",
		extra: map[string]any{"assessmentRunArns": arns, "assessmentRunArnsTruncated": false}}
}

// iscCall carries one request into the store. Inputs from other stores (EC2
// instances, SNS topic existence) are read before the store lock is taken.
type iscCall struct {
	in          map[string]any
	instances   []ec2.Instance
	topicExists bool
	baseURL     string
}

func (s *Server) handleInspectorJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if r.Method != http.MethodPost || !strings.HasPrefix(target, iscTargetPrefix) {
		return false
	}
	if hint := strings.TrimSpace(sigV4ServiceHint(r)); hint != "" && hint != "inspector" {
		return false
	}
	// A declared length bounds what net/http reads, so require one.
	if r.ContentLength < 0 || r.ContentLength > iscMaxBody {
		respondInspectorError(w, iscInvalid("", "request body must declare a Content-Length of at most 1 MiB"))
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "inspector"); !ok {
		respondInspectorJSON(w, status, map[string]any{"__type": code, "message": msg})
		return true
	}
	op := strings.TrimPrefix(target, iscTargetPrefix)
	if _, ok := inspectorOperationByName[op]; !ok {
		respondInspectorJSON(w, http.StatusBadRequest, map[string]any{"__type": "UnknownOperationException", "message": "unknown operation " + op})
		return true
	}
	raw, err := readBodyBytes(r)
	in := map[string]any{}
	if err == nil && len(bytes.TrimSpace(raw)) > 0 {
		err = json.Unmarshal(raw, &in)
	}
	if err != nil {
		respondInspectorJSON(w, http.StatusBadRequest, map[string]any{"__type": "SerializationException", "message": "request body is not a JSON object"})
		return true
	}
	model := awsmodels.InspectorModel()
	shapes := model.Operations[op]
	if problems := model.Validate(shapes.Input, in); len(problems) > 0 {
		respondInspectorError(w, iscInvalid(iscErrorCodeFor(problems[0]), "%s", problems[0]))
		return true
	}
	call := &iscCall{in: in, baseURL: iscBaseURL(r)}
	switch op {
	case "StartAssessmentRun", "PreviewAgents", "CreateExclusionsPreview":
		for _, res := range s.ec2.DescribeInstances(nil) {
			call.instances = append(call.instances, res.Instances...)
		}
	case "SubscribeToEvent":
		_, terr := s.sns.GetTopic(iscStr(in, "topicArn"))
		call.topicExists = terr == nil
	}
	out, ierr := s.inspector.dispatch(op, shapes.Output, call)
	if ierr != nil {
		respondInspectorError(w, ierr)
		return true
	}
	respondInspectorJSON(w, http.StatusOK, out)
	return true
}

func respondInspectorJSON(w http.ResponseWriter, status int, body any) {
	raw, _ := json.Marshal(body)
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func respondInspectorError(w http.ResponseWriter, e *iscError) {
	body := map[string]any{"__type": e.code, "message": e.msg, "canRetry": false}
	if e.errorCode != "" {
		body["errorCode"] = e.errorCode
	}
	for k, v := range e.extra {
		body[k] = v
	}
	w.Header().Set("X-Amzn-ErrorType", e.code)
	respondInspectorJSON(w, e.status(), body)
}

func iscBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// iscErrorCodeFor maps a model validation problem ("path: reason") to the
// closest InvalidInputErrorCode, keyed on the top-level member.
func iscErrorCodeFor(problem string) string {
	path, reason, _ := strings.Cut(problem, ": ")
	member := path
	if strings.HasPrefix(reason, "missing required member ") {
		member = strings.TrimPrefix(reason, "missing required member ")
	}
	if i := strings.IndexAny(member, ".["); i >= 0 {
		member = member[:i]
	}
	codes := map[string]string{
		"assessmentTargetArn": "INVALID_ASSESSMENT_TARGET_ARN", "assessmentTargetArns": "INVALID_ASSESSMENT_TARGET_ARN",
		"assessmentTemplateArn": "INVALID_ASSESSMENT_TEMPLATE_ARN", "assessmentTemplateArns": "INVALID_ASSESSMENT_TEMPLATE_ARN",
		"assessmentRunArn": "INVALID_ASSESSMENT_RUN_ARN", "assessmentRunArns": "INVALID_ASSESSMENT_RUN_ARN",
		"findingArns": "INVALID_FINDING_ARN", "resourceGroupArn": "INVALID_RESOURCE_GROUP_ARN", "resourceGroupArns": "INVALID_RESOURCE_GROUP_ARN",
		"rulesPackageArns": "INVALID_RULES_PACKAGE_ARN", "resourceArn": "INVALID_RESOURCE_ARN", "previewAgentsArn": "INVALID_ASSESSMENT_TARGET_ARN",
		"topicArn": "INVALID_SNS_TOPIC_ARN", "roleArn": "INVALID_IAM_ROLE_ARN", "assessmentTargetName": "INVALID_ASSESSMENT_TARGET_NAME",
		"assessmentTemplateName": "INVALID_ASSESSMENT_TEMPLATE_NAME", "assessmentRunName": "INVALID_ASSESSMENT_TEMPLATE_NAME",
		"durationInSeconds": "INVALID_ASSESSMENT_TEMPLATE_DURATION", "tags": "INVALID_TAG", "resourceGroupTags": "INVALID_RESOURCE_GROUP_TAG_KEY",
		"userAttributesForFindings": "INVALID_USER_ATTRIBUTE", "attributes": "INVALID_USER_ATTRIBUTE", "attributeKeys": "INVALID_USER_ATTRIBUTE_KEY",
		"nextToken": "INVALID_PAGINATION_TOKEN", "maxResults": "INVALID_MAX_RESULTS", "locale": "INVALID_LOCALE", "event": "INVALID_EVENT",
		"exclusionArns": "INVALID_RESOURCE_ARN", "previewToken": "INVALID_RESOURCE_ARN", "filter": "INVALID_ASSESSMENT_RUN_STATE",
	}
	return codes[member]
}

// --- small input/output helpers ---

func iscStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func iscStrList(in map[string]any, key string) []string {
	var out []string
	items, _ := in[key].([]any)
	for _, it := range items {
		if s, ok := it.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func iscInt(in map[string]any, key string, def int) int {
	if n, ok := in[key].(float64); ok && n >= -1<<31 && n < 1<<31 {
		return int(n)
	}
	return def
}

func iscTime(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

// iscParseTime reads a model timestamp (epoch seconds or ISO 8601).
func iscParseTime(v any) (time.Time, bool) {
	switch x := v.(type) {
	case float64:
		return time.UnixMilli(int64(x * 1000)).UTC(), true
	case string:
		if t, err := time.Parse(time.RFC3339, x); err == nil {
			return t, true
		}
		if f, err := strconv.ParseFloat(x, 64); err == nil {
			return time.UnixMilli(int64(f * 1000)).UTC(), true
		}
	}
	return time.Time{}, false
}

func iscInRange(rng any, t time.Time) bool {
	m, ok := rng.(map[string]any)
	if !ok {
		return true
	}
	if b, ok := iscParseTime(m["beginDate"]); ok && t.Before(b) {
		return false
	}
	if e, ok := iscParseTime(m["endDate"]); ok && t.After(e) {
		return false
	}
	return true
}

const iscIDChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// iscID returns an Inspector-style resource id, "0-" plus 8 base62 chars.
func iscID() string {
	b := make([]byte, 8)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(iscIDChars))))
		b[i] = iscIDChars[n.Int64()]
	}
	return "0-" + string(b)
}

// iscGlob matches Inspector name patterns (* and ? wildcards). Both sides
// are model-bounded to 140 characters, so the O(n*m) table is small.
func iscGlob(pattern, name string) bool {
	p, n := []rune(pattern), []rune(name)
	prev := make([]bool, len(n)+1)
	prev[0] = true
	for i := 1; i <= len(p); i++ {
		cur := make([]bool, len(n)+1)
		cur[0] = prev[0] && p[i-1] == '*'
		for j := 1; j <= len(n); j++ {
			switch p[i-1] {
			case '*':
				cur[j] = prev[j] || cur[j-1]
			case '?':
				cur[j] = prev[j-1]
			default:
				cur[j] = prev[j-1] && p[i-1] == n[j-1]
			}
		}
		prev = cur
	}
	return prev[len(n)]
}

// iscPage pages an ordered ARN list with an offset token.
func iscPage(items []string, in map[string]any, maxAllowed int) ([]string, string, *iscError) {
	limit := iscInt(in, "maxResults", 10)
	if limit < 1 || limit > maxAllowed {
		return nil, "", iscInvalid("INVALID_MAX_RESULTS", "maxResults must be between 1 and %d", maxAllowed)
	}
	start := 0
	if tok := iscStr(in, "nextToken"); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, "", iscInvalid("INVALID_PAGINATION_TOKEN", "invalid nextToken")
		}
		start = int(n)
	}
	end := min(start+limit, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return append([]string{}, items[start:end]...), next, nil
}
