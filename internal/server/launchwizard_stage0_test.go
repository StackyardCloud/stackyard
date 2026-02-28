package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLaunchWizardStage0CatalogCoverage(t *testing.T) {
	if len(launchWizardOperations) != 15 {
		t.Fatalf("expected 15 Launch Wizard actions from docs, got %d", len(launchWizardOperations))
	}
	if len(launchWizardOperationByName) != len(launchWizardOperations) {
		t.Fatalf("expected unique Launch Wizard action names")
	}

	requiredActions := []string{
		"CreateDeployment",
		"GetDeployment",
		"ListDeployments",
		"ListDeploymentEvents",
		"ListWorkloads",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := launchWizardOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(launchWizardDataTypes) != 13 {
		t.Fatalf("expected 13 Launch Wizard data types from docs, got %d", len(launchWizardDataTypes))
	}
	if len(launchWizardDataTypeByName) != len(launchWizardDataTypes) {
		t.Fatalf("expected unique Launch Wizard data type names")
	}

	requiredTypes := []string{
		"DeploymentData",
		"DeploymentDataSummary",
		"DeploymentEventDataSummary",
		"WorkloadData",
		"WorkloadDataSummary",
		"WorkloadDeploymentPatternDataSummary",
	}
	for _, typeName := range requiredTypes {
		if _, ok := launchWizardDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func launchWizardRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	body := []byte(payload)
	if payload == "" {
		body = nil
	}
	return signedRequestWithService(
		t,
		method,
		ts.URL+path,
		body,
		map[string]string{"Content-Type": "application/json"},
		"launchwizard",
	)
}

func TestLaunchWizardStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := launchWizardRequest(t, ts, http.MethodPost, "/totallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestLaunchWizardStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := launchWizardRequest(t, ts, http.MethodPost, "/listWorkloads", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "workloads") {
		t.Fatalf("expected ListWorkloads response to include workloads, got %q", body)
	}
}

func TestLaunchWizardStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range launchWizardOperations {
		payload := `{}`
		if op.Method == http.MethodGet || op.Method == http.MethodDelete {
			payload = ""
		}
		resp := launchWizardRequest(t, ts, op.Method, op.URI, payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
