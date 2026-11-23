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

func newAOSSTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// aossCall sends one awsJson1_0 call signed for aoss and decodes the reply.
func aossCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": aossTargetPrefix + op,
	}, "aoss")
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %q: %v", op, raw, err)
		}
	}
	return resp.StatusCode, out
}

func aossOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := aossCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func aossErrCode(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := aossCall(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s: want 400, got %d %v", op, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func TestOpenSearchServerlessStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.OpenSearchServerlessModel()
	if len(openSearchServerlessOperations) != len(model.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(openSearchServerlessOperations), len(model.Operations))
	}
	for _, op := range openSearchServerlessOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Fatalf("%s is not in the model", op.Name)
		}
		if aossHandlers[op.Name] == nil {
			t.Fatalf("%s has no handler", op.Name)
		}
	}
}

func TestOpenSearchServerlessStage0Routing(t *testing.T) {
	_, ts := newAOSSTestServer(t)
	// Our traffic is claimed.
	aossOK(t, ts, "GetAccountSettings", `{}`)
	// The OpenSearch Service (es) API still reaches its own router.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/2021-01-01/domain", nil, nil, "es")
	body := mustBody(t, resp)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "DomainNames") {
		t.Fatalf("OpenSearch ListDomainNames: %d %s", resp.StatusCode, body)
	}
	// A request signed for another service with our target isn't claimed.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": aossTargetPrefix + "GetAccountSettings",
	}, "s3")
	if strings.Contains(string(mustBody(t, resp)), "capacityLimits") {
		t.Fatal("an s3-signed request reached OpenSearch Serverless")
	}
}

func TestOpenSearchServerlessStage0RefusesUndeclaredBodies(t *testing.T) {
	_, ts := newAOSSTestServer(t)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Amz-Target", aossTargetPrefix+"GetAccountSettings")
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/aoss/aws4_request, SignedHeaders=host, Signature=00")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}

func TestOpenSearchServerlessStage0ModelDocuments(t *testing.T) {
	model := awsmodels.OpenSearchServerlessModel()
	// A document member accepts any JSON and is copied, not shared, on output.
	doc := map[string]any{"mappings": map[string]any{"properties": map[string]any{"title": map[string]any{"type": "text"}}}}
	if p := model.Validate("CreateIndexRequest", map[string]any{"id": "abc", "indexName": "books", "indexSchema": doc}); len(p) != 0 {
		t.Fatalf("document rejected: %v", p)
	}
	out := model.Conform("GetIndexResponse", map[string]any{"indexSchema": doc}).(map[string]any)
	out["indexSchema"].(map[string]any)["mappings"].(map[string]any)["x"] = 1
	if _, shared := doc["mappings"].(map[string]any)["x"]; shared {
		t.Fatal("conformed document shares memory with its source")
	}
}
