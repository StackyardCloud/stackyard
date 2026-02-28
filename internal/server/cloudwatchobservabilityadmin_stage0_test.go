package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func cloudWatchObservabilityAdminRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "observabilityadmin")
}

func TestCloudWatchObservabilityAdminStage0CatalogCoverage(t *testing.T) {
	if len(cloudWatchObservabilityAdminOperations) != 40 {
		t.Fatalf("expected 40 CloudWatch Observability Admin operations from docs, got %d", len(cloudWatchObservabilityAdminOperations))
	}
	if len(cloudWatchObservabilityAdminOperationByName) != len(cloudWatchObservabilityAdminOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateTelemetryPipeline",
		"CreateTelemetryRule",
		"CreateTelemetryRuleForOrganization",
		"CreateCentralizationRuleForOrganization",
		"CreateS3TableIntegration",
		"ListResourceTelemetryForOrganization",
		"ValidateTelemetryPipelineConfiguration",
	}
	for _, action := range requiredActions {
		if _, ok := cloudWatchObservabilityAdminOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(cloudWatchObservabilityAdminDataTypes) != 40 {
		t.Fatalf("expected 40 CloudWatch Observability Admin data types from docs, got %d", len(cloudWatchObservabilityAdminDataTypes))
	}
	if len(cloudWatchObservabilityAdminDataTypeByName) != len(cloudWatchObservabilityAdminDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"CentralizationRule",
		"TelemetryPipeline",
		"TelemetryRule",
		"TelemetryConfiguration",
		"ValidateTelemetryPipelineConfiguration",
	}
	for _, typeName := range requiredTypes {
		if _, ok := cloudWatchObservabilityAdminDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestCloudWatchObservabilityAdminStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudWatchObservabilityAdminRequest(t, ts, http.MethodPost, "/UnknownAction", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCloudWatchObservabilityAdminKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudWatchObservabilityAdminRequest(t, ts, http.MethodPost, "/ListTelemetryPipelines", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "TelemetryPipelines") {
		t.Fatalf("expected ListTelemetryPipelines response body to include TelemetryPipelines, got %q", body)
	}
}

func TestCloudWatchObservabilityAdminAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range cloudWatchObservabilityAdminOperations {
		body := []byte(`{}`)
		resp := cloudWatchObservabilityAdminRequest(t, ts, op.Method, op.URI, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
