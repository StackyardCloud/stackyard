package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func amplifyUIBuilderRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "amplifyuibuilder")
}

func TestAmplifyUIBuilderStage0CatalogCoverage(t *testing.T) {
	if len(amplifyUIBuilderOperations) != 28 {
		t.Fatalf("expected 28 Amplify UI Builder operations from docs, got %d", len(amplifyUIBuilderOperations))
	}
	if len(amplifyUIBuilderOperationByName) != len(amplifyUIBuilderOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateComponent",
		"GetComponent",
		"UpdateComponent",
		"CreateForm",
		"GetForm",
		"CreateTheme",
		"GetTheme",
		"StartCodegenJob",
		"GetMetadata",
		"ExchangeCodeForToken",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := amplifyUIBuilderOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(amplifyUIBuilderResources) != 66 {
		t.Fatalf("expected 66 Amplify UI Builder resources from docs, got %d", len(amplifyUIBuilderResources))
	}
	if len(amplifyUIBuilderResourceByName) != len(amplifyUIBuilderResources) {
		t.Fatalf("expected unique resource names")
	}

	requiredResources := []string{
		"Component",
		"Form",
		"Theme",
		"CodegenJob",
		"PutMetadataFlagBody",
	}
	for _, resourceName := range requiredResources {
		if _, ok := amplifyUIBuilderResourceByName[resourceName]; !ok {
			t.Fatalf("missing documented resource %s", resourceName)
		}
	}
}

func TestAmplifyUIBuilderStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := amplifyUIBuilderRequest(t, ts, http.MethodPost, "/amplifyuibuilder-unknown-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestAmplifyUIBuilderStage0KnownActionReturnsMetadata(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := amplifyUIBuilderRequest(t, ts, http.MethodGet, "/app/d1234567890/environment/dev/metadata", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "features") {
		t.Fatalf("expected GetMetadata response body to include features, got %q", body)
	}
}

func TestAmplifyUIBuilderStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"appId":           "d1234567890",
		"environmentName": "dev",
		"id":              "component-000001",
		"provider":        "figma",
		"resourceArn":     url.PathEscape("arn:aws:amplify:us-east-1:123456789012:apps/d1234567890"),
		"featureName":     "isRelationshipSupported",
		"clientToken":     "token-000001",
		"tagKeys":         "env",
		"maxResults":      "10",
		"nextToken":       "token-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range amplifyUIBuilderOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			return value
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := amplifyUIBuilderRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
