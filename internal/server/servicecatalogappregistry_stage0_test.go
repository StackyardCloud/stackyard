package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newAppRegistryTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// aprCall sends a servicecatalog-signed REST call and decodes the reply.
func aprCall(t *testing.T, ts *httptest.Server, want int, method, path, body string) map[string]any {
	t.Helper()
	var raw []byte
	headers := map[string]string{}
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "servicecatalog")
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		t.Fatalf("%s %s: %d (want %d) %s", method, path, resp.StatusCode, want, data)
	}
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("decode %s: %v", data, err)
		}
	}
	if want >= 400 && resp.Header.Get("X-Amzn-ErrorType") == "" {
		t.Fatalf("%s %s: error without X-Amzn-ErrorType", method, path)
	}
	return out
}

// aprErrCode returns the X-Amzn-ErrorType of a call expected to fail.
func aprErrCode(t *testing.T, ts *httptest.Server, method, path, body string) string {
	t.Helper()
	resp := signedRequestWithService(t, method, ts.URL+path, []byte(body), map[string]string{"Content-Type": "application/json"}, "servicecatalog")
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	if resp.StatusCode < 400 {
		t.Fatalf("%s %s: status %d, want an error", method, path, resp.StatusCode)
	}
	return resp.Header.Get("X-Amzn-ErrorType")
}

func TestAppRegistryStage0Catalog(t *testing.T) {
	if len(servicecatalogappregistryOperations) != 24 || len(servicecatalogappregistryOperationByName) != 24 {
		t.Fatalf("expected 24 unique operations, got %d/%d", len(servicecatalogappregistryOperations), len(servicecatalogappregistryOperationByName))
	}
}

func TestAppRegistryStage0Routing(t *testing.T) {
	_, ts := newAppRegistryTestServer(t)
	aprCall(t, ts, http.StatusNotFound, http.MethodPut, "/applications", "")

	// Service Catalog (same signing name, JSON 1.1 with X-Amz-Target) isn't claimed.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/applications", []byte(`{}`),
		map[string]string{"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AWS242ServiceCatalogService.ListPortfolios"}, "servicecatalog")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "applications") || resp.Header.Get("X-Amzn-ErrorType") == "UnknownOperationException" {
		t.Fatalf("an X-Amz-Target request reached AppRegistry: %d %s", resp.StatusCode, body)
	}
	// An AppRegistry path signed for another service isn't claimed.
	resp = signedRequestWithService(t, http.MethodGet, ts.URL+"/attribute-groups", nil, nil, "s3")
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "attributeGroups") {
		t.Fatalf("an s3-signed request reached AppRegistry: %s", body)
	}
	// A chunked body of unknown length is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/applications", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/servicecatalog/aws4_request, SignedHeaders=host, Signature=00")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", resp.StatusCode)
	}
}
