package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAppConfigStage0CatalogCoverage(t *testing.T) {
	if len(appConfigOperations) != 45 {
		t.Fatalf("expected 45 AppConfig operations from docs, got %d", len(appConfigOperations))
	}
	if len(appConfigOperationByName) != len(appConfigOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateApplication",
		"GetApplication",
		"ListApplications",
		"StartDeployment",
		"StopDeployment",
		"ValidateConfiguration",
	}
	for _, action := range requiredActions {
		if _, ok := appConfigOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(appConfigDataTypes) != 18 {
		t.Fatalf("expected 18 AppConfig data types from docs, got %d", len(appConfigDataTypes))
	}
	if len(appConfigDataTypeByName) != len(appConfigDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Application",
		"Environment",
		"DeploymentStrategy",
		"ConfigurationProfileSummary",
		"HostedConfigurationVersionSummary",
		"Validator",
	}
	for _, typeName := range requiredTypes {
		if _, ok := appConfigDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func appConfigRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
	}
	headers := map[string]string{}
	if method == http.MethodPost || method == http.MethodPatch {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "appconfig")
}

func TestAppConfigStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := appConfigRequest(t, ts, http.MethodGet, "/unknown-appconfig-route", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestAppConfigKnownRouteReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := appConfigRequest(t, ts, http.MethodGet, "/applications", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "Items") {
		t.Fatalf("expected ListApplications response body to include Items, got %q", body)
	}
}

func TestAppConfigStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacer := strings.NewReplacer(
		"{ApplicationId}", "app-000001",
		"{ConfigurationProfileId}", "cp-000001",
		"{DeploymentStrategyId}", "ds-000001",
		"{EnvironmentId}", "env-000001",
		"{ExtensionIdentifier}", "ext-000001",
		"{ExtensionAssociationId}", "exa-000001",
		"{VersionNumber}", "1",
		"{Application}", "app-000001",
		"{Environment}", "env-000001",
		"{Configuration}", "cfg-000001",
		"{DeploymentNumber}", "1",
		"{ResourceArn}", "arn:aws:appconfig:us-east-1:123456789012:application/app-000001",
	)

	for _, op := range appConfigOperations {
		path := replacer.Replace(op.URI)
		payload := ""
		if op.Method == http.MethodPost || op.Method == http.MethodPatch || op.Method == http.MethodPut {
			payload = `{}`
		}
		resp := appConfigRequest(t, ts, op.Method, path, payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
