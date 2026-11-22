package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/services/dynamodb"
)

func newAASTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func aasRequest(t *testing.T, ts *httptest.Server, op, body string) *http.Response {
	t.Helper()
	return signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.1",
		"X-Amz-Target": aasTargetPrefix + op,
	}, "application-autoscaling")
}

// aasCall sends op and decodes the reply, failing unless the status is want.
func aasCall(t *testing.T, ts *httptest.Server, want int, op, body string) map[string]any {
	t.Helper()
	resp := aasRequest(t, ts, op, body)
	raw := mustBody(t, resp)
	if resp.StatusCode != want {
		t.Fatalf("%s: status %d, want %d: %s", op, resp.StatusCode, want, raw)
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s: decode %s: %v", op, raw, err)
	}
	return out
}

// aasErrorCode sends op, expects a 400 and returns the error type.
func aasErrorCode(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	out := aasCall(t, ts, http.StatusBadRequest, op, body)
	code, _ := out["__type"].(string)
	return code
}

func aasCreateTable(t *testing.T, srv *Server, name string) {
	t.Helper()
	if _, err := srv.dynamodb.CreateTable(name, []dynamodb.AttributeDefinition{{AttributeName: "id", AttributeType: "S"}},
		[]dynamodb.KeySchemaElement{{AttributeName: "id", KeyType: "HASH"}}, "PROVISIONED", nil); err != nil {
		t.Fatalf("create table: %v", err)
	}
}

const aasTableKey = `"ServiceNamespace":"dynamodb","ResourceId":"table/orders","ScalableDimension":"dynamodb:table:ReadCapacityUnits"`

func TestApplicationAutoScalingStage0Catalog(t *testing.T) {
	if len(applicationautoscalingOperations) != 14 || len(applicationautoscalingOperationByName) != 14 {
		t.Fatalf("expected 14 unique operations, got %d/%d", len(applicationautoscalingOperations), len(applicationautoscalingOperationByName))
	}
	for _, op := range []string{"RegisterScalableTarget", "PutScalingPolicy", "PutScheduledAction", "GetPredictiveScalingForecast"} {
		if _, ok := applicationautoscalingOperationByName[op]; !ok {
			t.Fatalf("missing %s", op)
		}
	}
}

func TestApplicationAutoScalingStage0Routing(t *testing.T) {
	_, ts := newAASTestServer(t)
	if code := aasErrorCode(t, ts, "NoSuchOperation", `{}`); code != "UnknownOperationException" {
		t.Fatalf("unknown op: %s", code)
	}
	if code := aasErrorCode(t, ts, "DescribeScalableTargets", `{"ServiceNamespace":`); code != "SerializationException" {
		t.Fatalf("bad JSON: %s", code)
	}
	// Signed for another service: rejected, not silently served.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{"ServiceNamespace":"ecs"}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": aasTargetPrefix + "DescribeScalableTargets",
	}, "autoscaling")
	if body := string(mustBody(t, resp)); resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("wrong signing service: %d %s", resp.StatusCode, body)
	}
	// Auto Scaling Plans (AnyScaleScalingPlannerFrontendService) still routes to its own handler.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AnyScaleScalingPlannerFrontendService.DescribeScalingPlans",
	}, "autoscaling-plans")
	if body := string(mustBody(t, resp)); strings.Contains(body, "UnknownOperationException") {
		t.Fatalf("autoscaling plans was claimed by application auto scaling: %s", body)
	}
}
