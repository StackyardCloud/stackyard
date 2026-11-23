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

func newSchemasTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// schRaw sends a schemas-signed REST call and returns the status, headers and body.
func schRaw(t *testing.T, ts *httptest.Server, method, path, body string) (int, http.Header, []byte) {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "schemas")
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, data
}

// schCall sends a schemas-signed REST call and decodes the JSON reply.
func schCall(t *testing.T, ts *httptest.Server, want int, method, path, body string) map[string]any {
	t.Helper()
	status, hdr, data := schRaw(t, ts, method, path, body)
	if status != want {
		t.Fatalf("%s %s: %d (want %d) %s", method, path, status, want, data)
	}
	if want >= 400 && hdr.Get("X-Amzn-ErrorType") == "" {
		t.Fatalf("%s %s: error without X-Amzn-ErrorType", method, path)
	}
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("decode %s: %v", data, err)
		}
	}
	return out
}

// schErrCode returns the X-Amzn-ErrorType of a call expected to fail.
func schErrCode(t *testing.T, ts *httptest.Server, method, path, body string) string {
	t.Helper()
	status, hdr, data := schRaw(t, ts, method, path, body)
	if status < 400 {
		t.Fatalf("%s %s: status %d, want an error: %s", method, path, status, data)
	}
	return hdr.Get("X-Amzn-ErrorType")
}

func TestSchemasStage0CatalogMatchesModel(t *testing.T) {
	model := awsmodels.SchemasModel()
	if len(schemasOperations) != len(model.Operations) || len(schemasOperationByName) != len(schemasOperations) {
		t.Fatalf("catalog has %d operations, model has %d", len(schemasOperations), len(model.Operations))
	}
	for _, op := range schemasOperations {
		if _, ok := model.Operations[op.Name]; !ok {
			t.Fatalf("%s isn't in the model", op.Name)
		}
	}
}

func TestSchemasStage0Routing(t *testing.T) {
	_, ts := newSchemasTestServer(t)
	schCall(t, ts, http.StatusOK, http.MethodGet, "/v1/registries", "")
	if code := schErrCode(t, ts, http.MethodPatch, "/v1/registries/name/x", ""); code != "NotFoundException" {
		t.Fatalf("unknown route: %s", code)
	}
	// A schemas path signed for another service isn't claimed.
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/v1/registries", nil, nil, "s3")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "Registries") {
		t.Fatalf("an s3-signed request reached EventBridge Schemas: %s", body)
	}
	// EventBridge (JSON 1.1, signing name events) keeps its traffic.
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AWSEvents.ListEventBuses"}, "events")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "EventBuses") {
		t.Fatalf("EventBridge ListEventBuses: %d %s", resp.StatusCode, body)
	}
	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/registries/name/x", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/schemas/aws4_request, SignedHeaders=host, Signature=00")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", r.StatusCode)
	}
}
