package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func newGlueTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func glueRequest(t *testing.T, ts *httptest.Server, op, body string) *http.Response {
	t.Helper()
	return signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"X-Amz-Target": "AWSGlue." + op,
		"Content-Type": "application/x-amz-json-1.1",
	}, "glue")
}

// glueCall sends op and decodes the JSON response, failing unless the status
// is want.
func glueCall(t *testing.T, ts *httptest.Server, want int, op, body string) map[string]any {
	t.Helper()
	resp := glueRequest(t, ts, op, body)
	raw := mustBody(t, resp)
	if resp.StatusCode != want {
		t.Fatalf("%s %s: status %d, want %d: %s", op, body, resp.StatusCode, want, raw)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s: invalid JSON %q: %v", op, raw, err)
	}
	return out
}

func glueErrorType(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	out := glueCall(t, ts, http.StatusBadRequest, op, body)
	code, _ := out["__type"].(string)
	return code
}

func TestGlueStage0OperationsMatchModel(t *testing.T) {
	model := awsmodels.GlueModel()
	if len(glueOperations) != len(model.Operations) || len(glueOperationByName) != len(glueOperations) {
		t.Fatalf("ops file has %d (%d unique), model has %d", len(glueOperations), len(glueOperationByName), len(model.Operations))
	}
	for name := range model.Operations {
		if _, ok := glueOperationByName[name]; !ok {
			t.Errorf("model operation %s missing from glue_ops.go", name)
		}
		_, custom := glueHandlers[name]
		_, generic := glueOps[name]
		if !custom && !generic {
			t.Errorf("operation %s has no handler", name)
		}
	}
	for op, spec := range glueOps {
		if _, ok := model.Operations[op]; !ok {
			t.Errorf("glueOps names unknown operation %s", op)
		}
		if _, ok := glueKinds[spec.kind]; !ok {
			t.Errorf("%s uses unknown kind %s", op, spec.kind)
		}
	}
	for op := range glueHandlers {
		if _, ok := model.Operations[op]; !ok {
			t.Errorf("handler registered for unknown operation %s", op)
		}
	}
}

func TestGlueStage0RoutingAndValidation(t *testing.T) {
	_, ts := newGlueTestServer(t)

	if code := glueErrorType(t, ts, "NoSuchOperation", `{}`); code != "UnknownOperationException" {
		t.Fatalf("unknown operation: %s", code)
	}
	if code := glueErrorType(t, ts, "GetDatabase", `{`); code != "SerializationException" {
		t.Fatalf("bad JSON: %s", code)
	}
	if code := glueErrorType(t, ts, "GetDatabase", `{}`); code != "InvalidInputException" {
		t.Fatalf("missing required member: %s", code)
	}
	if code := glueErrorType(t, ts, "GetDatabase", `{"Name":7}`); code != "InvalidInputException" {
		t.Fatalf("wrong type: %s", code)
	}
	if code := glueErrorType(t, ts, "CreateTrigger", `{"Name":"t","Type":"HOURLY","Actions":[{"JobName":"j"}]}`); code != "InvalidInputException" {
		t.Fatalf("bad enum: %s", code)
	}
	if code := glueErrorType(t, ts, "GetDatabase", `{"Name":"`+strings.Repeat("d", 300)+`"}`); code != "InvalidInputException" {
		t.Fatalf("over max length: %s", code)
	}

	// Signed for another service.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"X-Amz-Target": "AWSGlue.GetDatabases", "Content-Type": "application/x-amz-json-1.1",
	}, "athena")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong signing service: %d", resp.StatusCode)
	}
	_ = mustBody(t, resp)

	// The response content type is constant.
	resp = glueRequest(t, ts, "GetDatabases", `{}`)
	if ct := resp.Header.Get("Content-Type"); ct != "application/x-amz-json-1.1" {
		t.Fatalf("content type %q", ct)
	}
	_ = mustBody(t, resp)

	// Other JSON services keep their traffic.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"X-Amz-Target": "AmazonAthena.ListWorkGroups", "Content-Type": "application/x-amz-json-1.1",
	}, "athena")
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "WorkGroups") {
		t.Fatalf("athena response: %s", body)
	}
}

// glueDefaultValue builds a minimal value for a shape: required members only,
// enums at their first value, strings "stackyard".
func glueDefaultValue(name string, depth int) any {
	sh := glueShape(name)
	if sh == nil || depth > 12 {
		return "stackyard"
	}
	switch sh.Type {
	case "structure":
		out := map[string]any{}
		for _, r := range sh.Required {
			out[r] = glueDefaultValue(sh.Members[r], depth+1)
		}
		return out
	case "list":
		return []any{glueDefaultValue(sh.Member, depth+1)}
	case "map":
		return map[string]any{"stackyard": glueDefaultValue(sh.Value, depth+1)}
	case "string":
		if len(sh.Enum) > 0 {
			return sh.Enum[0]
		}
		return "stackyard"
	case "integer", "long":
		if sh.Min != nil && *sh.Min > 1 {
			return *sh.Min
		}
		return 1.0
	case "double", "float":
		return 1.0
	case "boolean":
		return true
	case "timestamp":
		return 1700000000.0
	}
	return "c3RhY2t5YXJk"
}

// Every operation answers a model-minimal request with JSON: either a 200
// whose body matches the output shape, or a 400 with an error type.
func TestGlueStage0EveryOperationAnswersMinimalRequests(t *testing.T) {
	_, ts := newGlueTestServer(t)
	model := awsmodels.GlueModel()
	names := make([]string, 0, len(model.Operations))
	for n := range model.Operations {
		names = append(names, n)
	}
	sort.Strings(names)
	ok := 0
	for _, op := range names {
		body, _ := json.Marshal(glueDefaultValue(model.Operations[op].Input, 0))
		resp := glueRequest(t, ts, op, string(body))
		raw := mustBody(t, resp)
		var out map[string]any
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: non-JSON response %d %q", op, resp.StatusCode, raw)
		}
		switch resp.StatusCode {
		case http.StatusOK:
			ok++
			if out["__type"] != nil {
				t.Fatalf("%s: 200 with an error body %s", op, raw)
			}
			for _, req := range glueShape(model.Operations[op].Output).Required {
				if _, present := out[req]; !present {
					t.Fatalf("%s: response lacks required %s: %s", op, req, raw)
				}
			}
		case http.StatusBadRequest:
			if out["__type"] == nil {
				t.Fatalf("%s: 400 without an error type: %s", op, raw)
			}
		default:
			t.Fatalf("%s: status %d: %s", op, resp.StatusCode, raw)
		}
	}
	if ok < 40 {
		t.Fatalf("only %d operations succeeded on minimal input", ok)
	}
}
