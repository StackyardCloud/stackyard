package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func migrationHubRefactorSpacesRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "refactor-spaces")
}

func migrationHubRefactorSpacesPathForOperation(op migrationHubRefactorSpacesOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{environmentIdentifier}": "env-00000001",
		"{applicationIdentifier}": "app-00000001",
		"{serviceIdentifier}":     "service-00000001",
		"{routeIdentifier}":       "route-00000001",
		"{identifier}":            "env-00000001",
		"{resourceArn}":           url.PathEscape("arn:aws:refactor-spaces:us-east-1:123456789012:environment/env-00000001"),
	}
	for key, value := range replacements {
		path = strings.ReplaceAll(path, key, value)
	}
	return path
}

func TestMigrationHubRefactorSpacesStage0CatalogCoverage(t *testing.T) {
	if len(migrationHubRefactorSpacesOperations) != 24 {
		t.Fatalf("expected 24 Migration Hub Refactor Spaces operations from docs, got %d", len(migrationHubRefactorSpacesOperations))
	}
	if len(migrationHubRefactorSpacesOperationByName) != len(migrationHubRefactorSpacesOperations) {
		t.Fatalf("expected unique Migration Hub Refactor Spaces operation names")
	}

	requiredActions := []string{
		"CreateEnvironment",
		"GetEnvironment",
		"ListEnvironments",
		"CreateApplication",
		"ListApplications",
		"CreateService",
		"CreateRoute",
		"PutResourcePolicy",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := migrationHubRefactorSpacesOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(migrationHubRefactorSpacesDataTypes) != 18 {
		t.Fatalf("expected 18 Migration Hub Refactor Spaces data types from docs, got %d", len(migrationHubRefactorSpacesDataTypes))
	}
	if len(migrationHubRefactorSpacesDataTypeByName) != len(migrationHubRefactorSpacesDataTypes) {
		t.Fatalf("expected unique Migration Hub Refactor Spaces data type names")
	}

	requiredTypes := []string{
		"ApplicationSummary",
		"EnvironmentSummary",
		"EnvironmentVpc",
		"RouteSummary",
		"ServiceSummary",
		"ErrorResponse",
	}
	for _, typeName := range requiredTypes {
		if _, ok := migrationHubRefactorSpacesDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestMigrationHubRefactorSpacesStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := migrationHubRefactorSpacesRequest(t, ts, http.MethodGet, "/environments/env-00000001/unknown", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestMigrationHubRefactorSpacesStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := migrationHubRefactorSpacesRequest(t, ts, http.MethodGet, "/environments", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "environmentSummaryList") {
		t.Fatalf("expected ListEnvironments response body to include environmentSummaryList, got %q", body)
	}
}

func TestMigrationHubRefactorSpacesStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range migrationHubRefactorSpacesOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := migrationHubRefactorSpacesRequest(t, ts, op.Method, migrationHubRefactorSpacesPathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
