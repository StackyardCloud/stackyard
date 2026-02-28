package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func amplifyRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "amplify")
}

func TestAmplifyStage0CatalogCoverage(t *testing.T) {
	if len(amplifyOperations) != 37 {
		t.Fatalf("expected 37 Amplify operations from docs, got %d", len(amplifyOperations))
	}
	if len(amplifyOperationByName) != len(amplifyOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateApp",
		"GetApp",
		"ListApps",
		"CreateBranch",
		"ListBranches",
		"CreateDeployment",
		"StartJob",
		"TagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := amplifyOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(amplifyDataTypes) != 20 {
		t.Fatalf("expected 20 Amplify data types from docs, got %d", len(amplifyDataTypes))
	}
	if len(amplifyDataTypeByName) != len(amplifyDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"App",
		"Branch",
		"BackendEnvironment",
		"DomainAssociation",
		"Job",
		"Webhook",
	}
	for _, typeName := range requiredTypes {
		if _, ok := amplifyDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestAmplifyStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := amplifyRequest(t, ts, http.MethodPost, "/amplify-unknown-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestAmplifyStage0KnownActionReturnsListApps(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := amplifyRequest(t, ts, http.MethodGet, "/apps", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "apps") {
		t.Fatalf("expected ListApps response body to include apps, got %q", body)
	}
}

func TestAmplifyStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"appId":           "d1234567890",
		"branchName":      "main",
		"environmentName": "dev",
		"domainName":      "example.com",
		"jobId":           "0000001",
		"webhookId":       "webhook-000001",
		"artifactId":      "artifact-000001",
		"resourceArn":     "arn:aws:amplify:us-east-1:123456789012:apps/d1234567890",
		"tagKeys":         "env",
		"maxResults":      "10",
		"nextToken":       "token-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range amplifyOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			if strings.EqualFold(name, "resourceArn") {
				return url.PathEscape(value)
			}
			return value
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := amplifyRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
