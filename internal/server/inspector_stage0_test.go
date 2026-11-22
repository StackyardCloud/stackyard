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

func newInspectorTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// iscCallAPI sends one signed Inspector Classic call and decodes the reply.
func iscCallAPI(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "InspectorService." + op,
	}, "inspector")
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %s: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func iscOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := iscCallAPI(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %v", op, body, status, out)
	}
	return out
}

// iscFail expects a 400 and returns "Type/errorCode".
func iscFail(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := iscCallAPI(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s %s: status %d (want 400) %v", op, body, status, out)
	}
	code, _ := out["errorCode"].(string)
	return out["__type"].(string) + "/" + code
}

func TestInspectorStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.InspectorModel()
	if len(inspectorOperations) != len(model.Operations) || len(inspectorOperations) != 37 {
		t.Fatalf("%d operations, model has %d", len(inspectorOperations), len(model.Operations))
	}
	for _, op := range inspectorOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Fatalf("%s is not in the model", op.Name)
		}
		if iscHandlers[op.Name] == nil {
			t.Fatalf("%s has no handler", op.Name)
		}
	}
	if len(inspectorScanOperations) != 1 || inspectorScanOperations[0].Name != "ScanSbom" {
		t.Fatalf("scan operations: %v", inspectorScanOperations)
	}
}

func TestInspectorStage0Routing(t *testing.T) {
	_, ts := newInspectorTestServer(t)
	// Unknown action and bad input.
	if status, out := iscCallAPI(t, ts, "NoSuchThing", `{}`); status != http.StatusBadRequest || out["__type"] != "UnknownOperationException" {
		t.Fatalf("unknown op: %d %v", status, out)
	}
	if code := iscFail(t, ts, "CreateAssessmentTarget", `{}`); code != "InvalidInputException/INVALID_ASSESSMENT_TARGET_NAME" {
		t.Fatalf("missing name: %s", code)
	}
	// A classic call signed for another service is refused.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "InspectorService.ListRulesPackages",
	}, "inspector2")
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("inspector2-signed classic call: %d", resp.StatusCode)
	}
	resp.Body.Close()
	// Inspector v2 keeps its traffic.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/findings/list", []byte(`{}`), map[string]string{"Content-Type": "application/json"}, "inspector2")
	body := mustBody(t, resp)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "findings") {
		t.Fatalf("inspector2 ListFindings: %d %s", resp.StatusCode, body)
	}
	// Chunked bodies of unknown length are refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("X-Amz-Target", "InspectorService.ListRulesPackages")
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/inspector/aws4_request, SignedHeaders=host, Signature=00")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}

func TestInspectorStage0GlobAndPaging(t *testing.T) {
	for pattern, want := range map[string]bool{"web*": true, "w?b-1": true, "*-1": true, "db*": false, "": false, "*": true} {
		if got := iscGlob(pattern, "web-1"); got != want {
			t.Fatalf("glob %q: %v", pattern, got)
		}
	}
	items := []string{"a", "b", "c"}
	page, next, err := iscPage(items, map[string]any{"maxResults": float64(2)}, 500)
	if err != nil || len(page) != 2 || next != "2" {
		t.Fatalf("page 1: %v %q %v", page, next, err)
	}
	page, next, err = iscPage(items, map[string]any{"maxResults": float64(2), "nextToken": next}, 500)
	if err != nil || len(page) != 1 || next != "" {
		t.Fatalf("page 2: %v %q %v", page, next, err)
	}
	if _, _, err := iscPage(items, map[string]any{"nextToken": "99"}, 500); err == nil || err.errorCode != "INVALID_PAGINATION_TOKEN" {
		t.Fatalf("bad token: %v", err)
	}
	if _, _, err := iscPage(items, map[string]any{"maxResults": float64(0)}, 500); err == nil || err.errorCode != "INVALID_MAX_RESULTS" {
		t.Fatalf("bad max: %v", err)
	}
}
