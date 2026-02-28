package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func wickrRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "wickr")
}

func TestWickrStage0CatalogCoverage(t *testing.T) {
	if len(wickrOperations) != 42 {
		t.Fatalf("expected 42 Wickr actions from docs, got %d", len(wickrOperations))
	}
	if len(wickrOperationByName) != len(wickrOperations) {
		t.Fatalf("expected unique Wickr action names")
	}

	requiredActions := []string{
		"CreateNetwork",
		"GetNetwork",
		"ListNetworks",
		"UpdateNetwork",
		"DeleteNetwork",
		"CreateBot",
		"GetBot",
		"ListBots",
		"UpdateBot",
		"DeleteBot",
		"GetUser",
		"ListUsers",
		"GetUsersCount",
		"CreateSecurityGroup",
		"GetSecurityGroup",
		"ListSecurityGroups",
		"UpdateSecurityGroup",
		"DeleteSecurityGroup",
		"GetOidcInfo",
		"RegisterOidcConfig",
	}
	for _, action := range requiredActions {
		if _, ok := wickrOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(wickrDataTypes) != 30 {
		t.Fatalf("expected 30 Wickr data types from docs, got %d", len(wickrDataTypes))
	}
	if len(wickrDataTypeByName) != len(wickrDataTypes) {
		t.Fatalf("expected unique Wickr data type names")
	}

	requiredTypes := []string{
		"Network",
		"NetworkSettings",
		"User",
		"Bot",
		"SecurityGroup",
		"GuestUser",
		"OidcConfigInfo",
	}
	for _, typeName := range requiredTypes {
		if _, ok := wickrDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestWickrStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := wickrRequest(t, ts, http.MethodGet, "/wickr/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestWickrStage0KnownActionReturnsListNetworks(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := wickrRequest(t, ts, http.MethodGet, "/networks?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Networks") {
		t.Fatalf("expected ListNetworks response body to include Networks, got %q", body)
	}
}

func TestWickrStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"networkId":     "n-000001",
		"userId":        "u-000001",
		"botId":         "b-000001",
		"groupId":       "g-000001",
		"usernameHash":  "uh-000001",
		"suspend":       "false",
		"certificate":   "stackyard-cert",
		"clientId":      "stackyard-client-id",
		"clientSecret":  "stackyard-client-secret",
		"code":          "stackyard-code",
		"codeVerifier":  "stackyard-verifier",
		"grantType":     "authorization_code",
		"redirectUri":   "https://localhost/callback",
		"url":           "https://idp.example.com",
		"endTime":       "2026-01-01T00:00:00Z",
		"startTime":     "2025-01-01T00:00:00Z",
		"admin":         "true",
		"maxResults":    "10",
		"nextToken":     "token-000001",
		"sortDirection": "ASCENDING",
		"sortFields":    "CreateDate",
		"username":      "stackyard-user",
		"displayName":   "stackyard-bot",
		"status":        "ACTIVE",
		"billingPeriod": "LAST_30_DAYS",
		"firstName":     "Stack",
		"lastName":      "Yard",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range wickrOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.TrimSpace(strings.Trim(token, "{}"))
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			return url.PathEscape(value)
		})

		var body []byte
		switch op.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			body = []byte(`{}`)
		default:
			body = nil
		}

		resp := wickrRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
