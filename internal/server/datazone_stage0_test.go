package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func datazoneRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "datazone")
}

func datazonePathForOperation(op datazoneOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{domainIdentifier}":      "dzd-000001",
		"{identifier}":            "id-000001",
		"{entityType}":            "asset",
		"{entityIdentifier}":      "asset-000001",
		"{projectIdentifier}":     "project-000001",
		"{environmentIdentifier}": "env-000001",
		"{dataSourceIdentifier}":  "ds-000001",
		"{resourceArn}":           "arn:aws:datazone:us-east-1:123456789012:domain/dzd-000001",
	}
	for key, value := range replacements {
		path = strings.ReplaceAll(path, key, value)
	}
	return path
}

func TestDataZoneStage0CatalogCoverage(t *testing.T) {
	if len(datazoneOperations) != 175 {
		t.Fatalf("expected 175 DataZone operations from docs, got %d", len(datazoneOperations))
	}
	if len(datazoneOperationByName) != len(datazoneOperations) {
		t.Fatalf("expected unique DataZone operation names")
	}

	requiredActions := []string{
		"CreateDomain",
		"GetDomain",
		"ListDomains",
		"CreateProject",
		"ListProjects",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := datazoneOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(datazoneDataTypes) != 306 {
		t.Fatalf("expected 306 DataZone data types from docs, got %d", len(datazoneDataTypes))
	}
	if len(datazoneDataTypeByName) != len(datazoneDataTypes) {
		t.Fatalf("expected unique DataZone data type names")
	}

	requiredTypes := []string{
		"DomainSummary",
		"ProjectSummary",
		"DataSourceSummary",
		"EnvironmentSummary",
		"ResourceTag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := datazoneDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestDataZoneStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := datazoneRequest(t, ts, http.MethodPost, "/unknown", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestDataZoneStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := datazoneRequest(t, ts, http.MethodGet, "/v2/domains", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
}

func TestDataZoneStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range datazoneOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := datazoneRequest(t, ts, op.Method, datazonePathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
