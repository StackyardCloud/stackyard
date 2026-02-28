package server

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func amplifyAdminUIRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "amplifybackend")
}

func TestAmplifyAdminUIStage0CatalogCoverage(t *testing.T) {
	if len(amplifyAdminUIOperations) != 31 {
		t.Fatalf("expected 31 Amplify Admin UI operations from docs, got %d", len(amplifyAdminUIOperations))
	}
	if len(amplifyAdminUIOperationByName) != len(amplifyAdminUIOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateBackend",
		"GetBackend",
		"CreateBackendAPI",
		"GetBackendAuth",
		"CreateToken",
		"GetToken",
		"ListS3Buckets",
	}
	for _, action := range requiredActions {
		if _, ok := amplifyAdminUIOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(amplifyAdminUIResources) != 30 {
		t.Fatalf("expected 30 Amplify Admin UI resources from docs, got %d", len(amplifyAdminUIResources))
	}
	if len(amplifyAdminUIResourceByName) != len(amplifyAdminUIResources) {
		t.Fatalf("expected unique resource names")
	}

	requiredResources := []string{
		"Backend",
		"Backend appId Api",
		"Backend appId Auth",
		"Backend appId Storage",
		"S3Buckets",
	}
	for _, resourceName := range requiredResources {
		if _, ok := amplifyAdminUIResourceByName[resourceName]; !ok {
			t.Fatalf("missing documented resource %s", resourceName)
		}
	}
}

func TestAmplifyAdminUIStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := amplifyAdminUIRequest(t, ts, http.MethodPost, "/prod/amplifyadminui-unknown-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestAmplifyAdminUIStage0KnownActionReturnsS3Buckets(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := amplifyAdminUIRequest(t, ts, http.MethodPost, "/prod/s3Buckets", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "buckets") {
		t.Fatalf("expected ListS3Buckets response body to include buckets, got %q", body)
	}
}

func TestAmplifyAdminUIStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"appId":                  "d1234567890",
		"backendEnvironmentName": "dev",
		"sessionId":              "session-000001",
		"jobId":                  "job-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range amplifyAdminUIOperations {
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

		resp := amplifyAdminUIRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
