package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stackyard/stackyard/internal/services/ecs"
)

func TestApplicationAutoScalingStage2ScalingPolicies(t *testing.T) {
	srv, ts := newAASTestServer(t)
	aasCreateTable(t, srv, "orders")
	policy := func(name, extra string) string {
		return fmt.Sprintf(`{%s,"PolicyName":%q%s}`, aasTableKey, name, extra)
	}
	tracking := `,"PolicyType":"TargetTrackingScaling","TargetTrackingScalingPolicyConfiguration":{"TargetValue":70,"PredefinedMetricSpecification":{"PredefinedMetricType":"DynamoDBReadCapacityUtilization"}}`
	// The target has to exist first.
	if code := aasErrorCode(t, ts, "PutScalingPolicy", policy("read", tracking)); code != "ObjectNotFoundException" {
		t.Fatalf("no target: %s", code)
	}
	aasCall(t, ts, http.StatusOK, "RegisterScalableTarget", `{`+aasTableKey+`,"MinCapacity":1,"MaxCapacity":10}`)

	for name, extra := range map[string]string{
		"write metric on read dimension": `,"PolicyType":"TargetTrackingScaling","TargetTrackingScalingPolicyConfiguration":{"TargetValue":70,"PredefinedMetricSpecification":{"PredefinedMetricType":"DynamoDBWriteCapacityUtilization"}}`,
		"ECS metric on DynamoDB":         `,"PolicyType":"TargetTrackingScaling","TargetTrackingScalingPolicyConfiguration":{"TargetValue":70,"PredefinedMetricSpecification":{"PredefinedMetricType":"ECSServiceAverageCPUUtilization"}}`,
		"no target value":                `,"PolicyType":"TargetTrackingScaling","TargetTrackingScalingPolicyConfiguration":{"PredefinedMetricSpecification":{"PredefinedMetricType":"DynamoDBReadCapacityUtilization"}}`,
		"two metric sources":             `,"PolicyType":"TargetTrackingScaling","TargetTrackingScalingPolicyConfiguration":{"TargetValue":70,"PredefinedMetricSpecification":{"PredefinedMetricType":"DynamoDBReadCapacityUtilization"},"CustomizedMetricSpecification":{"MetricName":"m","Namespace":"n","Statistic":"Average"}}`,
		"two returning queries":          `,"PolicyType":"TargetTrackingScaling","TargetTrackingScalingPolicyConfiguration":{"TargetValue":70,"CustomizedMetricSpecification":{"Metrics":[{"Id":"a","Expression":"1"},{"Id":"b","Expression":"2"}]}}`,
		"step without config":            `,"PolicyType":"StepScaling"`,
		"mismatched config":              `,"PolicyType":"StepScaling","TargetTrackingScalingPolicyConfiguration":{"TargetValue":70}`,
		"step gap":                       `,"StepScalingPolicyConfiguration":{"AdjustmentType":"ChangeInCapacity","StepAdjustments":[{"MetricIntervalUpperBound":0,"ScalingAdjustment":-1},{"MetricIntervalLowerBound":10,"ScalingAdjustment":1}]}`,
		"magnitude without percent":      `,"StepScalingPolicyConfiguration":{"AdjustmentType":"ChangeInCapacity","MinAdjustmentMagnitude":2,"StepAdjustments":[{"ScalingAdjustment":1}]}`,
		"predictive outside ECS":         `,"PolicyType":"PredictiveScaling","PredictiveScalingPolicyConfiguration":{"MetricSpecifications":[{"TargetValue":40,"PredefinedMetricPairSpecification":{"PredefinedMetricType":"ECSServiceCPUUtilization"}}]}`,
	} {
		if code := aasErrorCode(t, ts, "PutScalingPolicy", policy("bad", extra)); code != "ValidationException" {
			t.Fatalf("%s: %s", name, code)
		}
	}

	put := aasCall(t, ts, http.StatusOK, "PutScalingPolicy", policy("read", tracking))
	arn, _ := put["PolicyARN"].(string)
	alarms, _ := put["Alarms"].([]any)
	if !strings.Contains(arn, ":scalingPolicy:") || !strings.HasSuffix(arn, "policyName/read") || len(alarms) != 2 {
		t.Fatalf("put tracking policy: %v", put)
	}
	alarmName := alarms[0].(map[string]any)["AlarmName"].(string)
	if _, ok := srv.cloudwatch.alarms[alarmName]; !ok {
		t.Fatalf("alarm %s wasn't created in CloudWatch", alarmName)
	}
	step := `,"StepScalingPolicyConfiguration":{"AdjustmentType":"PercentChangeInCapacity","MinAdjustmentMagnitude":1,"Cooldown":60,"StepAdjustments":[{"MetricIntervalUpperBound":0,"ScalingAdjustment":-10},{"MetricIntervalLowerBound":0,"ScalingAdjustment":10}]}`
	aasCall(t, ts, http.StatusOK, "PutScalingPolicy", policy("steps", step))

	// Replacing the tracking policy with a step policy keeps its ARN and
	// drops its alarms.
	again := aasCall(t, ts, http.StatusOK, "PutScalingPolicy", policy("read", step))
	if again["PolicyARN"] != arn || len(again["Alarms"].([]any)) != 0 {
		t.Fatalf("update: %v", again)
	}
	if _, ok := srv.cloudwatch.alarms[alarmName]; ok {
		t.Fatal("replaced policy's alarm is still in CloudWatch")
	}
	policies := aasCall(t, ts, http.StatusOK, "DescribeScalingPolicies", `{"ServiceNamespace":"dynamodb","PolicyNames":["steps"]}`)["ScalingPolicies"].([]any)
	if len(policies) != 1 {
		t.Fatalf("policies: %v", policies)
	}
	got := policies[0].(map[string]any)
	cfg := got["StepScalingPolicyConfiguration"].(map[string]any)
	if got["PolicyType"] != "StepScaling" || cfg["Cooldown"] != float64(60) || len(cfg["StepAdjustments"].([]any)) != 2 {
		t.Fatalf("described policy: %v", got)
	}

	aasCall(t, ts, http.StatusOK, "DeleteScalingPolicy", policy("steps", ""))
	if code := aasErrorCode(t, ts, "DeleteScalingPolicy", policy("steps", "")); code != "ObjectNotFoundException" {
		t.Fatalf("second delete: %s", code)
	}
	// Deregistering the target deletes its remaining policies.
	aasCall(t, ts, http.StatusOK, "DeregisterScalableTarget", `{`+aasTableKey+`}`)
	if n := len(aasCall(t, ts, http.StatusOK, "DescribeScalingPolicies", `{"ServiceNamespace":"dynamodb"}`)["ScalingPolicies"].([]any)); n != 0 {
		t.Fatalf("%d policies left after deregister", n)
	}
}

func TestApplicationAutoScalingStage2ScheduledActions(t *testing.T) {
	srv, ts := newAASTestServer(t)
	aasCreateTable(t, srv, "orders")
	aasCall(t, ts, http.StatusOK, "RegisterScalableTarget", `{`+aasTableKey+`,"MinCapacity":1,"MaxCapacity":10}`)
	action := func(name, extra string) string {
		return fmt.Sprintf(`{%s,"ScheduledActionName":%q%s}`, aasTableKey, name, extra)
	}
	for name, extra := range map[string]string{
		"no schedule":       `,"ScalableTargetAction":{"MinCapacity":2}`,
		"no target action":  `,"Schedule":"rate(1 hour)"`,
		"bad rate unit":     `,"Schedule":"rate(1 hours)","ScalableTargetAction":{"MinCapacity":2}`,
		"cron needs ?":      `,"Schedule":"cron(0 6 * * * *)","ScalableTargetAction":{"MinCapacity":2}`,
		"cron field count":  `,"Schedule":"cron(0 6 * * ?)","ScalableTargetAction":{"MinCapacity":2}`,
		"bad at":            `,"Schedule":"at(2026-13-01T00:00:00)","ScalableTargetAction":{"MinCapacity":2}`,
		"bad time zone":     `,"Schedule":"rate(5 minutes)","Timezone":"Mars/Olympus","ScalableTargetAction":{"MinCapacity":2}`,
		"min above max":     `,"Schedule":"rate(5 minutes)","ScalableTargetAction":{"MinCapacity":5,"MaxCapacity":2}`,
		"end before start":  `,"Schedule":"rate(5 minutes)","StartTime":2000,"EndTime":1000,"ScalableTargetAction":{"MinCapacity":2}`,
		"name with a slash": `,"Schedule":"rate(5 minutes)","ScalableTargetAction":{"MinCapacity":2}`,
	} {
		n := "bad"
		if name == "name with a slash" {
			n = "a/b"
		}
		if code := aasErrorCode(t, ts, "PutScheduledAction", action(n, extra)); code != "ValidationException" {
			t.Fatalf("%s: %s", name, code)
		}
	}
	aasCall(t, ts, http.StatusOK, "PutScheduledAction", action("morning", `,"Schedule":"cron(0 6 ? * MON-FRI *)","Timezone":"America/New_York","ScalableTargetAction":{"MinCapacity":4,"MaxCapacity":8}`))
	// An update merges into the existing action.
	aasCall(t, ts, http.StatusOK, "PutScheduledAction", action("morning", `,"ScalableTargetAction":{"MaxCapacity":9}`))
	actions := aasCall(t, ts, http.StatusOK, "DescribeScheduledActions", `{"ServiceNamespace":"dynamodb"}`)["ScheduledActions"].([]any)
	if len(actions) != 1 {
		t.Fatalf("actions: %v", actions)
	}
	got := actions[0].(map[string]any)
	sta := got["ScalableTargetAction"].(map[string]any)
	if got["Schedule"] != "cron(0 6 ? * MON-FRI *)" || got["Timezone"] != "America/New_York" || sta["MinCapacity"] != float64(4) || sta["MaxCapacity"] != float64(9) ||
		!strings.Contains(got["ScheduledActionARN"].(string), "scheduledActionName/morning") {
		t.Fatalf("action: %v", got)
	}
	aasCall(t, ts, http.StatusOK, "DeleteScheduledAction", action("morning", ""))
	if code := aasErrorCode(t, ts, "DeleteScheduledAction", action("morning", "")); code != "ObjectNotFoundException" {
		t.Fatalf("second delete: %s", code)
	}
}

func TestApplicationAutoScalingStage2PredictiveForecast(t *testing.T) {
	srv, ts := newAASTestServer(t)
	if _, err := srv.ecs.CreateCluster("web", nil, nil); err != nil {
		t.Fatal(err)
	}
	td, err := srv.ecs.RegisterTaskDefinition(ecs.TaskDefinitionInput{Family: "app", ContainerDefinitions: []ecs.ContainerDefinition{{Name: "app", Image: "nginx"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.ecs.CreateService("web", "api", td.Arn, "", 1, true, nil); err != nil {
		t.Fatal(err)
	}
	key := `"ServiceNamespace":"ecs","ResourceId":"service/web/api","ScalableDimension":"ecs:service:DesiredCount"`
	if code := aasErrorCode(t, ts, "RegisterScalableTarget", `{"ServiceNamespace":"ecs","ResourceId":"service/web/nope","ScalableDimension":"ecs:service:DesiredCount","MinCapacity":1,"MaxCapacity":4}`); code != "ValidationException" {
		t.Fatalf("missing ECS service: %s", code)
	}
	aasCall(t, ts, http.StatusOK, "RegisterScalableTarget", `{`+key+`,"MinCapacity":1,"MaxCapacity":4}`)
	if code := aasErrorCode(t, ts, "PutScalingPolicy", `{`+key+`,"PolicyName":"p","PolicyType":"PredictiveScaling","PredictiveScalingPolicyConfiguration":{"MetricSpecifications":[{"TargetValue":40,"PredefinedScalingMetricSpecification":{"PredefinedMetricType":"ECSServiceAverageCPUUtilization"}}]}}`); code != "ValidationException" {
		t.Fatalf("scaling metric without load metric: %s", code)
	}
	aasCall(t, ts, http.StatusOK, "PutScalingPolicy", `{`+key+`,"PolicyName":"p","PolicyType":"PredictiveScaling","PredictiveScalingPolicyConfiguration":{"Mode":"ForecastOnly","MetricSpecifications":[{"TargetValue":40,"PredefinedMetricPairSpecification":{"PredefinedMetricType":"ECSServiceCPUUtilization"}}]}}`)

	start := float64(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix())
	window := fmt.Sprintf(`"StartTime":%v,"EndTime":%v`, start, start+3*3600)
	fc := aasCall(t, ts, http.StatusOK, "GetPredictiveScalingForecast", `{`+key+`,"PolicyName":"p",`+window+`}`)
	capacity := fc["CapacityForecast"].(map[string]any)
	if len(capacity["Timestamps"].([]any)) != 3 || capacity["Values"].([]any)[0] != float64(2) {
		t.Fatalf("forecast: %v", fc)
	}
	if code := aasErrorCode(t, ts, "GetPredictiveScalingForecast", `{`+key+`,"PolicyName":"nope",`+window+`}`); code != "ValidationException" {
		t.Fatalf("unknown policy: %s", code)
	}
	tooLong := fmt.Sprintf(`"StartTime":%v,"EndTime":%v`, start, start+31*24*3600)
	if code := aasErrorCode(t, ts, "GetPredictiveScalingForecast", `{`+key+`,"PolicyName":"p",`+tooLong+`}`); code != "ValidationException" {
		t.Fatalf("31-day window: %s", code)
	}
}

func TestApplicationAutoScalingStage2RefusesBodiesWithoutDeclaredLength(t *testing.T) {
	_, ts := newAASTestServer(t)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", strings.NewReader(`{}`))
	req.ContentLength = -1 // sent chunked
	req.Header.Set("X-Amz-Target", aasTargetPrefix+"DescribeScalableTargets")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if body := string(mustBody(t, resp)); resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "Content-Length") {
		t.Fatalf("chunked body: %d %s", resp.StatusCode, body)
	}
}
