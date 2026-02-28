package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func workspacesWebRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "workspaces-web")
}

func TestWorkSpacesWebStage0CatalogCoverage(t *testing.T) {
	if len(workspacesWebOperations) != 75 {
		t.Fatalf("expected 75 WorkSpaces Web actions from docs, got %d", len(workspacesWebOperations))
	}
	if len(workspacesWebOperationByName) != len(workspacesWebOperations) {
		t.Fatalf("expected unique WorkSpaces Web action names")
	}

	requiredActions := []string{
		"CreatePortal",
		"UpdatePortal",
		"DeletePortal",
		"GetPortal",
		"ListPortals",
		"AssociateBrowserSettings",
		"ListSessions",
		"ExpireSession",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := workspacesWebOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(workspacesWebTypes) != 45 {
		t.Fatalf("expected 45 WorkSpaces Web data types from docs, got %d", len(workspacesWebTypes))
	}
	if len(workspacesWebTypeByName) != len(workspacesWebTypes) {
		t.Fatalf("expected unique WorkSpaces Web data type names")
	}

	requiredTypes := []string{
		"Portal",
		"PortalSummary",
		"Session",
		"SessionSummary",
		"BrowserSettings",
		"TrustStore",
		"Tag",
		"ValidationExceptionField",
	}
	for _, typeName := range requiredTypes {
		if _, ok := workspacesWebTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestWorkSpacesWebStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesWebRequest(t, ts, http.MethodPost, "/workspaces-web/unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestWorkSpacesWebStage0KnownActionReturnsListPortals(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesWebRequest(t, ts, http.MethodGet, "/portals?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "portals") {
		t.Fatalf("expected ListPortals response body to include portals, got %q", body)
	}
}

func TestWorkSpacesWebStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"portalArn":                     "arn:aws:workspaces-web:us-east-1:123456789012:portal/p-000001",
		"portalArn+":                    "arn:aws:workspaces-web:us-east-1:123456789012:portal/p-000001",
		"browserSettingsArn":            "arn:aws:workspaces-web:us-east-1:123456789012:browserSettings/bs-000001",
		"browserSettingsArn+":           "arn:aws:workspaces-web:us-east-1:123456789012:browserSettings/bs-000001",
		"dataProtectionSettingsArn":     "arn:aws:workspaces-web:us-east-1:123456789012:dataProtectionSettings/dps-000001",
		"dataProtectionSettingsArn+":    "arn:aws:workspaces-web:us-east-1:123456789012:dataProtectionSettings/dps-000001",
		"identityProviderArn":           "arn:aws:workspaces-web:us-east-1:123456789012:identityProvider/idp-000001",
		"identityProviderArn+":          "arn:aws:workspaces-web:us-east-1:123456789012:identityProvider/idp-000001",
		"ipAccessSettingsArn":           "arn:aws:workspaces-web:us-east-1:123456789012:ipAccessSettings/ipa-000001",
		"ipAccessSettingsArn+":          "arn:aws:workspaces-web:us-east-1:123456789012:ipAccessSettings/ipa-000001",
		"networkSettingsArn":            "arn:aws:workspaces-web:us-east-1:123456789012:networkSettings/ns-000001",
		"networkSettingsArn+":           "arn:aws:workspaces-web:us-east-1:123456789012:networkSettings/ns-000001",
		"sessionLoggerArn":              "arn:aws:workspaces-web:us-east-1:123456789012:sessionLogger/sl-000001",
		"sessionLoggerArn+":             "arn:aws:workspaces-web:us-east-1:123456789012:sessionLogger/sl-000001",
		"trustStoreArn":                 "arn:aws:workspaces-web:us-east-1:123456789012:trustStore/ts-000001",
		"trustStoreArn+":                "arn:aws:workspaces-web:us-east-1:123456789012:trustStore/ts-000001",
		"userAccessLoggingSettingsArn":  "arn:aws:workspaces-web:us-east-1:123456789012:userAccessLoggingSettings/ual-000001",
		"userAccessLoggingSettingsArn+": "arn:aws:workspaces-web:us-east-1:123456789012:userAccessLoggingSettings/ual-000001",
		"userSettingsArn":               "arn:aws:workspaces-web:us-east-1:123456789012:userSettings/us-000001",
		"userSettingsArn+":              "arn:aws:workspaces-web:us-east-1:123456789012:userSettings/us-000001",
		"resourceArn":                   "arn:aws:workspaces-web:us-east-1:123456789012:portal/p-000001",
		"resourceArn+":                  "arn:aws:workspaces-web:us-east-1:123456789012:portal/p-000001",
		"portalId":                      "p-000001",
		"sessionId":                     "s-000001",
		"thumbprint":                    "aa",
		"maxResults":                    "10",
		"nextToken":                     "token-000001",
		"status":                        "ACTIVE",
		"sortBy":                        "StartTime",
		"username":                      "stackyard-user",
		"tagKeys":                       "env",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range workspacesWebOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = replacements[strings.TrimSuffix(name, "+")]
			}
			if value == "" {
				value = "stackyard"
			}
			return url.PathEscape(value)
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := workspacesWebRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
