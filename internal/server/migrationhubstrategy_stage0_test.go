package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func migrationHubStrategyRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "migrationhub-strategy")
}

func migrationHubStrategyPathForOperation(op migrationHubStrategyOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{applicationComponentId}": "appcomp-00000001",
		"{id}":                     "assessment-00000001",
		"{serverId}":               "server-00000001",
	}
	for key, value := range replacements {
		path = strings.ReplaceAll(path, key, value)
	}
	return path
}

func TestMigrationHubStrategyStage0CatalogCoverage(t *testing.T) {
	if len(migrationHubStrategyOperations) != 22 {
		t.Fatalf("expected 22 Migration Hub Strategy operations from docs, got %d", len(migrationHubStrategyOperations))
	}
	if len(migrationHubStrategyOperationByName) != len(migrationHubStrategyOperations) {
		t.Fatalf("expected unique Migration Hub Strategy operation names")
	}

	requiredActions := []string{
		"GetAssessment",
		"ListServers",
		"StartAssessment",
		"StopAssessment",
		"PutPortfolioPreferences",
		"UpdateServerConfig",
	}
	for _, action := range requiredActions {
		if _, ok := migrationHubStrategyOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(migrationHubStrategyDataTypes) != 53 {
		t.Fatalf("expected 53 Migration Hub Strategy data types from docs, got %d", len(migrationHubStrategyDataTypes))
	}
	if len(migrationHubStrategyDataTypeByName) != len(migrationHubStrategyDataTypes) {
		t.Fatalf("expected unique Migration Hub Strategy data type names")
	}
}

func TestMigrationHubStrategyStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := migrationHubStrategyRequest(t, ts, http.MethodGet, "/get-unknown", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestMigrationHubStrategyStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := migrationHubStrategyRequest(t, ts, http.MethodPost, "/list-servers", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "serverInfos") {
		t.Fatalf("expected ListServers response body to include serverInfos, got %q", body)
	}
}

func TestMigrationHubStrategyStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range migrationHubStrategyOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) {
			payload = ""
		}
		resp := migrationHubStrategyRequest(t, ts, op.Method, migrationHubStrategyPathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
