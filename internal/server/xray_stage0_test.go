package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func xrayRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, []byte(payload), headers, "xray")
}

func TestXRayStage0CatalogCoverage(t *testing.T) {
	if len(xrayOperations) != 38 {
		t.Fatalf("expected 38 X-Ray actions from docs, got %d", len(xrayOperations))
	}
	if len(xrayOperationByName) != len(xrayOperations) {
		t.Fatalf("expected unique X-Ray action names")
	}

	requiredActions := []string{
		"PutTraceSegments",
		"BatchGetTraces",
		"GetTraceSummaries",
		"CreateGroup",
		"GetSamplingRules",
		"StartTraceRetrieval",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := xrayOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(xrayDataTypes) != 68 {
		t.Fatalf("expected 68 X-Ray data types from docs, got %d", len(xrayDataTypes))
	}
	if len(xrayDataTypeByName) != len(xrayDataTypes) {
		t.Fatalf("expected unique X-Ray data type names")
	}

	requiredTypes := []string{
		"Group",
		"SamplingRule",
		"Trace",
		"TraceSummary",
		"Service",
		"ResourcePolicy",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := xrayDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestXRayStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := xrayRequest(t, ts, http.MethodPost, "/xray/unknown", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestXRayStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := xrayRequest(t, ts, http.MethodPost, "/Groups", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Groups") {
		t.Fatalf("expected GetGroups response body to include Groups, got %q", body)
	}
}

func TestXRayStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range xrayOperations {
		resp := xrayRequest(t, ts, op.Method, op.URI, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
