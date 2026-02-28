package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cloudFrontRequest(t *testing.T, ts *httptest.Server, action string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{
		"X-Stackyard-Operation": action,
	}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, http.MethodPost, ts.URL+"/2020-05-31/cloudfront", body, headers, "cloudfront")
}

func TestCloudFrontStage0CatalogCoverage(t *testing.T) {
	if len(cloudFrontOperations) != 168 {
		t.Fatalf("expected 168 CloudFront operations from docs, got %d", len(cloudFrontOperations))
	}
	if len(cloudFrontOperationByName) != len(cloudFrontOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredOps := []string{
		"CreateDistribution",
		"GetDistribution",
		"ListDistributions",
		"CreateInvalidation",
		"TagResource",
		"UpdateDistribution",
	}
	for _, name := range requiredOps {
		if _, ok := cloudFrontOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}

	if len(cloudFrontDataTypes) != 208 {
		t.Fatalf("expected 208 CloudFront data types from docs, got %d", len(cloudFrontDataTypes))
	}
	if len(cloudFrontDataTypeByName) != len(cloudFrontDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Distribution",
		"DistributionConfig",
		"Invalidation",
		"PublicKey",
		"Tag",
		"VpcOrigin",
	}
	for _, name := range requiredTypes {
		if _, ok := cloudFrontDataTypeByName[name]; !ok {
			t.Fatalf("missing documented data type %s", name)
		}
	}
}

func TestCloudFrontStage0KnownRouteDoesNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudFrontRequest(t, ts, "ListDistributions", []byte(`{}`))
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("expected CloudFront route to be implemented")
	}
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("expected non-NotImplemented response body, got %q", body)
	}
}

func TestCloudFrontAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range cloudFrontOperations {
		resp := cloudFrontRequest(t, ts, op.Name, []byte(`{}`))
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
