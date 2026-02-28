package server

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func rolesAnywhereRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "rolesanywhere")
}

func rolesAnywherePathForOperation(op rolesAnywhereOperation) string {
	path := strings.TrimSpace(strings.SplitN(op.URI, "?", 2)[0])
	re := regexp.MustCompile(`\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(path, func(match string) string {
		name := strings.Trim(match, "{}")
		switch name {
		case "profileId":
			return "profile-000001"
		case "trustAnchorId":
			return "ta-000001"
		case "crlId":
			return "crl-000001"
		case "subjectId":
			return "subject-000001"
		case "resourceArn":
			return "arn:aws:rolesanywhere:us-east-1:123456789012:profile/profile-000001"
		default:
			return "stackyard"
		}
	})
}

func TestRolesAnywhereStage0CatalogCoverage(t *testing.T) {
	if len(rolesAnywhereOperations) != 30 {
		t.Fatalf("expected 30 Roles Anywhere operations from docs, got %d", len(rolesAnywhereOperations))
	}
	if len(rolesAnywhereOperationByName) != len(rolesAnywhereOperations) {
		t.Fatalf("expected unique Roles Anywhere operation names")
	}

	requiredActions := []string{
		"ListProfiles",
		"CreateProfile",
		"GetProfile",
		"ListTrustAnchors",
		"ImportCrl",
		"ListSubjects",
		"TagResource",
		"PutNotificationSettings",
	}
	for _, action := range requiredActions {
		if _, ok := rolesAnywhereOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(rolesAnywhereDataTypes) != 15 {
		t.Fatalf("expected 15 Roles Anywhere data types from docs, got %d", len(rolesAnywhereDataTypes))
	}
	if len(rolesAnywhereDataTypeByName) != len(rolesAnywhereDataTypes) {
		t.Fatalf("expected unique Roles Anywhere data type names")
	}
}

func TestRolesAnywhereStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := rolesAnywhereRequest(t, ts, http.MethodGet, "/definitely-not-rolesanywhere-route", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestRolesAnywhereStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := rolesAnywhereRequest(t, ts, http.MethodGet, "/profiles?pageSize=10", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "profiles") {
		t.Fatalf("expected ListProfiles response body to include profiles, got %q", body)
	}
}

func TestRolesAnywhereStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range rolesAnywhereOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := rolesAnywhereRequest(t, ts, op.Method, rolesAnywherePathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
