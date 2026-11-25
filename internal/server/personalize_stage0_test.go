package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func newPersonalizeTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func pszDecodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("decode %q: %v", raw, err)
		}
	}
	return out
}

func pszCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": pszTargetPrefix + op,
	}, "personalize")
	return resp.StatusCode, pszDecodeBody(t, resp)
}

func pszOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := pszCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func pszErrCode(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := pszCall(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s: want 400, got %d %v", op, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

// pszREST calls a Personalize Events or Runtime path.
func pszREST(t *testing.T, ts *httptest.Server, path, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+path, []byte(body), map[string]string{"Content-Type": "application/json"}, "personalize")
	return resp.StatusCode, pszDecodeBody(t, resp)
}

func TestPersonalizeStage0CatalogsMatchModels(t *testing.T) {
	check := func(name string, model *awsmodels.Model, ops []string, handled func(string) bool) {
		if len(ops) != len(model.Operations) {
			t.Fatalf("%s: catalog has %d operations, model %d", name, len(ops), len(model.Operations))
		}
		for _, op := range ops {
			if _, ok := model.Operations[op]; !ok {
				t.Errorf("%s: %s is not in the model", name, op)
			}
			if !handled(op) {
				t.Errorf("%s: %s has no handler", name, op)
			}
		}
	}
	var ops []string
	for _, op := range personalizeOperations {
		ops = append(ops, op.Name)
	}
	check("personalize", awsmodels.PersonalizeModel(), ops, func(op string) bool { return pszHandlers[op] != nil })
	ops = nil
	for _, op := range personalizeEventsOperations {
		ops = append(ops, op.Name)
	}
	routed := func(paths map[string]string) func(string) bool {
		return func(op string) bool {
			for _, v := range paths {
				if v == op {
					return true
				}
			}
			return false
		}
	}
	check("personalize-events", awsmodels.PersonalizeEventsModel(), ops, routed(pszevPaths))
	ops = nil
	for _, op := range personalizeRuntimeOperations {
		ops = append(ops, op.Name)
	}
	check("personalize-runtime", awsmodels.PersonalizeRuntimeModel(), ops, routed(pszrtPaths))
}

// The three Personalize APIs share a signing name; each claims only its own
// traffic, and none takes requests signed for other services.
func TestPersonalizeStage0Routing(t *testing.T) {
	_, ts := newPersonalizeTestServer(t)
	if out := pszOK(t, ts, "ListDatasetGroups", `{}`); out["datasetGroups"] == nil {
		t.Fatalf("ListDatasetGroups: %v", out)
	}
	// A personalize-signed request to an unrelated path isn't claimed as Events/Runtime.
	if status, out := pszREST(t, ts, "/events", `{"trackingId":"nope","sessionId":"s","eventList":[{"eventType":"click","sentAt":1}]}`); status != http.StatusBadRequest || out["__type"] != "InvalidInputException" {
		t.Fatalf("PutEvents with unknown tracker: %d %v", status, out)
	}
	if status, out := pszREST(t, ts, "/recommendations", `{"campaignArn":"arn:aws:personalize:us-east-1:123456789012:campaign/none","userId":"u"}`); status != http.StatusNotFound || out["__type"] != "ResourceNotFoundException" {
		t.Fatalf("GetRecommendations on a missing campaign: %d %v", status, out)
	}
	// Signed for another service: /events and /users stay with their owners.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/users", []byte(`{}`), map[string]string{"Content-Type": "application/json"}, "cognito-idp")
	if body := string(mustBody(t, resp)); strings.Contains(body, "InvalidInputException") {
		t.Fatalf("Personalize Events claimed a cognito-idp request: %s", body)
	}
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": pszTargetPrefix + "ListDatasetGroups",
	}, "s3")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("s3-signed request: %d", resp.StatusCode)
	}
	resp.Body.Close()
	// Migration Hub's JSON target is untouched.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": mghTargetPrefix + "ListProgressUpdateStreams",
	}, "mgh")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Migration Hub request: %d %s", resp.StatusCode, mustBody(t, resp))
	}
	resp.Body.Close()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("X-Amz-Target", pszTargetPrefix+"ListDatasetGroups")
	chunked, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer chunked.Body.Close()
	if chunked.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", chunked.StatusCode)
	}
	if code := pszErrCode(t, ts, "NoSuchOperation", `{}`); code != "UnknownOperationException" {
		t.Fatalf("unknown op: %s", code)
	}
	if code := pszErrCode(t, ts, "CreateDatasetGroup", `{"name":"-bad"}`); code != "InvalidInputException" {
		t.Fatalf("bad name: %s", code)
	}
}
