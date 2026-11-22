package server

import (
	"fmt"
	"net/http"
	"testing"
)

func TestApplicationAutoScalingStage1ScalableTargets(t *testing.T) {
	srv, ts := newAASTestServer(t)

	// The DynamoDB table must exist, and a new target needs both capacities.
	if code := aasErrorCode(t, ts, "RegisterScalableTarget", `{`+aasTableKey+`,"MinCapacity":1,"MaxCapacity":10}`); code != "ValidationException" {
		t.Fatalf("missing table: %s", code)
	}
	aasCreateTable(t, srv, "orders")
	if code := aasErrorCode(t, ts, "RegisterScalableTarget", `{`+aasTableKey+`,"MinCapacity":1}`); code != "ValidationException" {
		t.Fatalf("missing MaxCapacity: %s", code)
	}
	for name, body := range map[string]string{
		"min above max":      `{` + aasTableKey + `,"MinCapacity":11,"MaxCapacity":10}`,
		"wrong dimension":    `{"ServiceNamespace":"dynamodb","ResourceId":"table/orders","ScalableDimension":"ecs:service:DesiredCount","MinCapacity":1,"MaxCapacity":2}`,
		"bad resource id":    `{"ServiceNamespace":"dynamodb","ResourceId":"orders","ScalableDimension":"dynamodb:table:ReadCapacityUnits","MinCapacity":1,"MaxCapacity":2}`,
		"bad namespace":      `{"ServiceNamespace":"nope","ResourceId":"table/orders","ScalableDimension":"dynamodb:table:ReadCapacityUnits","MinCapacity":1,"MaxCapacity":2}`,
		"bad role":           `{` + aasTableKey + `,"MinCapacity":1,"MaxCapacity":2,"RoleARN":"role"}`,
		"negative capacity":  `{` + aasTableKey + `,"MinCapacity":-1,"MaxCapacity":2}`,
		"aws: tag":           `{` + aasTableKey + `,"MinCapacity":1,"MaxCapacity":2,"Tags":{"aws:x":"y"}}`,
		"capacity overflows": `{` + aasTableKey + `,"MinCapacity":1,"MaxCapacity":4294967296}`,
	} {
		if code := aasErrorCode(t, ts, "RegisterScalableTarget", body); code != "ValidationException" && code != "SerializationException" {
			t.Fatalf("%s: %s", name, code)
		}
	}

	arn := aasCall(t, ts, http.StatusOK, "RegisterScalableTarget", `{`+aasTableKey+`,"MinCapacity":1,"MaxCapacity":10,"Tags":{"team":"a"}}`)["ScalableTargetARN"].(string)
	// An update may change one capacity, and keeps the ARN.
	if again := aasCall(t, ts, http.StatusOK, "RegisterScalableTarget", `{`+aasTableKey+`,"MaxCapacity":20,"SuspendedState":{"ScheduledScalingSuspended":true}}`)["ScalableTargetARN"]; again != arn {
		t.Fatalf("update changed the ARN: %v vs %v", again, arn)
	}
	if code := aasErrorCode(t, ts, "RegisterScalableTarget", `{`+aasTableKey+`,"MinCapacity":21}`); code != "ValidationException" {
		t.Fatalf("update min above max: %s", code)
	}
	targets := aasCall(t, ts, http.StatusOK, "DescribeScalableTargets", `{"ServiceNamespace":"dynamodb"}`)["ScalableTargets"].([]any)
	if len(targets) != 1 {
		t.Fatalf("targets: %v", targets)
	}
	got := targets[0].(map[string]any)
	suspended := got["SuspendedState"].(map[string]any)
	if got["MinCapacity"] != float64(1) || got["MaxCapacity"] != float64(20) || suspended["ScheduledScalingSuspended"] != true ||
		got["RoleARN"] != aasDefaultRole("dynamodb") || got["ScalableTargetARN"] != arn {
		t.Fatalf("target: %v", got)
	}
	if n := len(aasCall(t, ts, http.StatusOK, "DescribeScalableTargets", `{"ServiceNamespace":"ecs"}`)["ScalableTargets"].([]any)); n != 0 {
		t.Fatalf("ecs filter returned %d targets", n)
	}
	// Registrations that set capacity are recorded as activities.
	acts := aasCall(t, ts, http.StatusOK, "DescribeScalingActivities", `{"ServiceNamespace":"dynamodb","ResourceId":"table/orders"}`)["ScalingActivities"].([]any)
	if len(acts) != 2 || acts[0].(map[string]any)["Description"] != "Setting min capacity to 1 and max capacity to 20" {
		t.Fatalf("activities: %v", acts)
	}

	// Tags.
	aasCall(t, ts, http.StatusOK, "TagResource", fmt.Sprintf(`{"ResourceARN":%q,"Tags":{"env":"dev"}}`, arn))
	aasCall(t, ts, http.StatusOK, "UntagResource", fmt.Sprintf(`{"ResourceARN":%q,"TagKeys":["team"]}`, arn))
	tags := aasCall(t, ts, http.StatusOK, "ListTagsForResource", fmt.Sprintf(`{"ResourceARN":%q}`, arn))["Tags"].(map[string]any)
	if len(tags) != 1 || tags["env"] != "dev" {
		t.Fatalf("tags: %v", tags)
	}
	missing := "arn:aws:application-autoscaling:us-east-1:123456789012:scalable-target/0000"
	if code := aasErrorCode(t, ts, "ListTagsForResource", fmt.Sprintf(`{"ResourceARN":%q}`, missing)); code != "ResourceNotFoundException" {
		t.Fatalf("unknown ARN: %s", code)
	}
	many := `{"ResourceARN":"` + arn + `","Tags":{`
	for i := 0; i < 50; i++ {
		many += fmt.Sprintf(`"k%d":"v",`, i)
	}
	if code := aasErrorCode(t, ts, "TagResource", many[:len(many)-1]+"}}"); code != "TooManyTagsException" {
		t.Fatalf("51 tags: %s", code)
	}

	aasCall(t, ts, http.StatusOK, "DeregisterScalableTarget", `{`+aasTableKey+`}`)
	if code := aasErrorCode(t, ts, "DeregisterScalableTarget", `{`+aasTableKey+`}`); code != "ObjectNotFoundException" {
		t.Fatalf("second deregister: %s", code)
	}
}

func TestApplicationAutoScalingStage1Pagination(t *testing.T) {
	srv, ts := newAASTestServer(t)
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("page%d", i)
		aasCreateTable(t, srv, name)
		aasCall(t, ts, http.StatusOK, "RegisterScalableTarget", fmt.Sprintf(`{"ServiceNamespace":"dynamodb","ResourceId":"table/%s","ScalableDimension":"dynamodb:table:WriteCapacityUnits","MinCapacity":1,"MaxCapacity":5}`, name))
	}
	page := aasCall(t, ts, http.StatusOK, "DescribeScalableTargets", `{"ServiceNamespace":"dynamodb","MaxResults":2}`)
	next, _ := page["NextToken"].(string)
	if len(page["ScalableTargets"].([]any)) != 2 || next == "" {
		t.Fatalf("first page: %v", page)
	}
	page = aasCall(t, ts, http.StatusOK, "DescribeScalableTargets", fmt.Sprintf(`{"ServiceNamespace":"dynamodb","MaxResults":2,"NextToken":%q}`, next))
	if len(page["ScalableTargets"].([]any)) != 1 || page["NextToken"] != nil {
		t.Fatalf("last page: %v", page)
	}
	if code := aasErrorCode(t, ts, "DescribeScalableTargets", `{"ServiceNamespace":"dynamodb","NextToken":"bogus"}`); code != "InvalidNextTokenException" {
		t.Fatalf("bad token: %s", code)
	}
	if code := aasErrorCode(t, ts, "DescribeScalableTargets", `{"ServiceNamespace":"dynamodb","MaxResults":51}`); code != "ValidationException" {
		t.Fatalf("MaxResults 51: %s", code)
	}
	ids := `{"ServiceNamespace":"dynamodb","ResourceIds":["table/page1"]}`
	if n := len(aasCall(t, ts, http.StatusOK, "DescribeScalableTargets", ids)["ScalableTargets"].([]any)); n != 1 {
		t.Fatalf("ResourceIds filter: %d", n)
	}
}
