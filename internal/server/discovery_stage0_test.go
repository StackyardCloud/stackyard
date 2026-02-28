package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func discoveryRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AWSPoseidonService_V2015_11_01." + action,
		},
		"discovery",
	)
}

func TestDiscoveryStage0CatalogCoverage(t *testing.T) {
	if len(discoveryOperations) != 28 {
		t.Fatalf("expected 28 Discovery operations from docs, got %d", len(discoveryOperations))
	}
	if len(discoveryOperationByName) != len(discoveryOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateApplication",
		"DescribeAgents",
		"GetDiscoverySummary",
		"StartExportTask",
		"StartImportTask",
		"StartBatchDeleteConfigurationTask",
		"ListConfigurations",
	}
	for _, action := range requiredActions {
		if _, ok := discoveryOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(discoveryDataTypes) != 29 {
		t.Fatalf("expected 29 Discovery data types from docs, got %d", len(discoveryDataTypes))
	}
	if len(discoveryDataTypeByName) != len(discoveryDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"AgentInfo",
		"AgentConfigurationStatus",
		"BatchDeleteConfigurationTask",
		"ConfigurationTag",
		"ImportTask",
		"Tag",
		"UsageMetricBasis",
	}
	for _, typeName := range requiredTypes {
		if _, ok := discoveryDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestDiscoveryStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := discoveryRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestDiscoveryStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := discoveryRequest(t, ts, "DescribeAgents", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "agentsInfo") {
		t.Fatalf("expected DescribeAgents response body to include agentsInfo, got %q", body)
	}
}

func TestDiscoveryStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range discoveryOperations {
		resp := discoveryRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
