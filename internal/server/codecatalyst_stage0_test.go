package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func codeCatalystRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "codecatalyst")
}

func TestCodeCatalystStage0CatalogCoverage(t *testing.T) {
	if len(codeCatalystOperations) != 38 {
		t.Fatalf("expected 38 CodeCatalyst operations from docs, got %d", len(codeCatalystOperations))
	}
	if len(codeCatalystOperationByName) != len(codeCatalystOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateProject",
		"GetProject",
		"ListProjects",
		"CreateDevEnvironment",
		"ListWorkflows",
		"CreateAccessToken",
		"VerifySession",
	}
	for _, action := range requiredActions {
		if _, ok := codeCatalystOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(codeCatalystTypes) != 31 {
		t.Fatalf("expected 31 CodeCatalyst data types from docs, got %d", len(codeCatalystTypes))
	}
	if len(codeCatalystTypeByName) != len(codeCatalystTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"AccessTokenSummary",
		"DevEnvironmentSummary",
		"ProjectSummary",
		"SpaceSummary",
		"WorkflowSummary",
	}
	for _, typeName := range requiredTypes {
		if _, ok := codeCatalystTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestCodeCatalystStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeCatalystRequest(t, ts, http.MethodGet, "/v1/codecatalyst-unknown-action", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCodeCatalystStage0KnownActionReturnsListSpaces(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeCatalystRequest(t, ts, http.MethodPost, "/v1/spaces", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "items") {
		t.Fatalf("expected ListSpaces response body to include items, got %q", body)
	}
}

func TestCodeCatalystStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"spaceName":            "stackyard-space",
		"projectName":          "stackyard-project",
		"name":                 "stackyard-resource",
		"sourceRepositoryName": "stackyard-repository",
		"id":                   "resource-000001",
		"devEnvironmentId":     "dev-env-000001",
		"sessionId":            "session-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range codeCatalystOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			return url.PathEscape(value)
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := codeCatalystRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
