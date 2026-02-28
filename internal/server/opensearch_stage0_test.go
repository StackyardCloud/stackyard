package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func opensearchRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "es")
}

func opensearchPathForTest(template string) string {
	replacements := map[string]string{
		"{ConnectionId}":   "cc-1234567890abcdef0",
		"{DomainName}":     "stackyard-opensearch-domain",
		"{PackageID}":      "pkg-1234567890abcdef",
		"{DataSourceName}": "stackyard-ds",
		"{EngineVersion}":  "OpenSearch_2.13",
		"{InstanceType}":   "m5.large.search",
		"{id}":             "app-1234567890abcdef",
		"{IndexName}":      "stackyard-index",
		"{VpcEndpointId}":  "vpce-1234567890abcdef0",
	}
	out := template
	for key, value := range replacements {
		out = strings.ReplaceAll(out, key, url.PathEscape(value))
	}
	return out
}

func TestOpenSearchStage0OperationCoverage(t *testing.T) {
	if len(opensearchOperations) != 82 {
		t.Fatalf("expected 82 OpenSearch Service operations from docs, got %d", len(opensearchOperations))
	}
	if len(opensearchOperationByName) != len(opensearchOperations) {
		t.Fatalf("expected unique operation names")
	}
	required := []string{
		"CreateDomain",
		"DescribeDomain",
		"UpdateDomainConfig",
		"ListDomainNames",
		"DeleteDomain",
		"UpgradeDomain",
	}
	for _, name := range required {
		if _, ok := opensearchOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
	if _, ok := opensearchOperationByName["CreatePipeline"]; ok {
		t.Fatalf("OpenSearch Ingestion operation leaked into OpenSearch Service list")
	}
}

func TestOpenSearchUnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := opensearchRequest(t, ts, http.MethodPost, "/2021-01-01/opensearch/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestOpenSearchKnownRouteReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := opensearchRequest(
		t,
		ts,
		http.MethodGet,
		"/2021-01-01/opensearch/domain/stackyard-opensearch-domain",
		nil,
	)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "DomainStatus") {
		t.Fatalf("expected DescribeDomain response body to include DomainStatus, got %q", body)
	}
}

func TestOpenSearchAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range opensearchOperations {
		if strings.TrimSpace(op.Method) == "" || strings.TrimSpace(op.URI) == "" {
			continue
		}
		path := opensearchPathForTest(op.URI)
		var body []byte
		switch op.Method {
		case http.MethodPost, http.MethodPut:
			body = []byte(`{}`)
		}

		resp := opensearchRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
