package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func workspacesThinClientRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "thinclient")
}

func TestWorkSpacesThinClientStage0CatalogCoverage(t *testing.T) {
	if len(workspacesThinClientOperations) != 16 {
		t.Fatalf("expected 16 WorkSpaces Thin Client actions from docs, got %d", len(workspacesThinClientOperations))
	}
	if len(workspacesThinClientOperationByName) != len(workspacesThinClientOperations) {
		t.Fatalf("expected unique WorkSpaces Thin Client action names")
	}

	requiredActions := []string{
		"CreateEnvironment",
		"DeleteDevice",
		"DeleteEnvironment",
		"DeregisterDevice",
		"GetDevice",
		"GetEnvironment",
		"GetSoftwareSet",
		"ListDevices",
		"ListEnvironments",
		"ListSoftwareSets",
		"ListTagsForResource",
		"TagResource",
		"UntagResource",
		"UpdateDevice",
		"UpdateEnvironment",
		"UpdateSoftwareSet",
	}
	for _, action := range requiredActions {
		if _, ok := workspacesThinClientOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(workspacesThinClientTypes) != 9 {
		t.Fatalf("expected 9 WorkSpaces Thin Client data types from docs, got %d", len(workspacesThinClientTypes))
	}
	if len(workspacesThinClientTypeByName) != len(workspacesThinClientTypes) {
		t.Fatalf("expected unique WorkSpaces Thin Client data type names")
	}

	requiredTypes := []string{
		"Device",
		"Environment",
		"MaintenanceWindow",
		"Software",
		"SoftwareSet",
		"ValidationExceptionField",
	}
	for _, typeName := range requiredTypes {
		if _, ok := workspacesThinClientTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestWorkSpacesThinClientStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesThinClientRequest(t, ts, http.MethodPost, "/workspaces-thin-client/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestWorkSpacesThinClientStage0KnownActionReturnsListEnvironments(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesThinClientRequest(t, ts, http.MethodGet, "/environments?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "environments") {
		t.Fatalf("expected ListEnvironments response body to include environments, got %q", body)
	}
}

func TestWorkSpacesThinClientStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"id":          "abcdefghijklmnopqrstuvwx",
		"clientToken": "stackyard-thin-client-token-000001",
		"resourceArn": "arn:aws:thinclient:us-east-1:123456789012:environment/abcdefghijklmnopqrstuvwx",
		"tagKeys":     "env",
		"maxResults":  "10",
		"nextToken":   "token-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range workspacesThinClientOperations {
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

		resp := workspacesThinClientRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
