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

func newMigrationHubTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func mghCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": mghTargetPrefix + op,
	}, "mgh")
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %q: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func mghOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := mghCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func mghErrCode(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := mghCall(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s: want 400, got %d %v", op, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func TestMigrationHubStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.MigrationHubModel()
	if len(migrationHubOperations) != len(model.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(migrationHubOperations), len(model.Operations))
	}
	for _, op := range migrationHubOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Errorf("%s is not in the model", op.Name)
		}
		if mghHandlers[op.Name] == nil {
			t.Errorf("%s has no handler", op.Name)
		}
	}
}

// Migration Hub must not take the other Migration Hub services' traffic, and
// it refuses requests signed for another service and chunked bodies.
func TestMigrationHubStage0Routing(t *testing.T) {
	_, ts := newMigrationHubTestServer(t)
	if out := mghOK(t, ts, "ListProgressUpdateStreams", `{}`); out["ProgressUpdateStreamSummaryList"] == nil {
		t.Fatalf("ListProgressUpdateStreams: %v", out)
	}
	// A look-alike target prefix (no "." right after AWSMigrationHub) isn't claimed.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AWSMigrationHubStrategyService.GetPortfolioSummary",
	}, "migrationhub-strategy")
	if body := string(mustBody(t, resp)); strings.Contains(body, "unknown operation") {
		t.Fatalf("Migration Hub claimed another service's target: %s", body)
	}
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": mghTargetPrefix + "ListProgressUpdateStreams",
	}, "s3")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("s3-signed request: %d", resp.StatusCode)
	}
	resp.Body.Close()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("X-Amz-Target", mghTargetPrefix+"ListProgressUpdateStreams")
	chunked, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer chunked.Body.Close()
	if chunked.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", chunked.StatusCode)
	}
	if code := mghErrCode(t, ts, "NoSuchOperation", `{}`); code != "UnknownOperationException" {
		t.Fatalf("unknown op: %s", code)
	}
}
