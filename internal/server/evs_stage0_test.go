package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func evsRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "evs")
}

func evsPathForOperation(op evsOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{environmentId}": "evs-env-00000001",
		"{hostId}":        "h-00000001",
		"{vlanId}":        "vlan-0001",
		"{allocationId}":  "eipalloc-00000001",
		"{resourceArn}":   url.PathEscape("arn:aws:evs:us-east-1:123456789012:environment/evs-env-00000001"),
	}
	for k, v := range replacements {
		path = strings.ReplaceAll(path, k, v)
	}
	return path
}

func TestEVSStage0CatalogCoverage(t *testing.T) {
	if len(evsOperations) != 14 {
		t.Fatalf("expected 14 EVS operations from docs, got %d", len(evsOperations))
	}
	if len(evsOperationByName) != len(evsOperations) {
		t.Fatalf("expected unique EVS operation names")
	}

	requiredActions := []string{
		"CreateEnvironment",
		"GetEnvironment",
		"ListEnvironments",
		"CreateEnvironmentHost",
		"ListEnvironmentHosts",
		"ListEnvironmentVlans",
		"GetVersions",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := evsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(evsDataTypes) != 18 {
		t.Fatalf("expected 18 EVS data types from docs, got %d", len(evsDataTypes))
	}
	if len(evsDataTypeByName) != len(evsDataTypes) {
		t.Fatalf("expected unique EVS data type names")
	}

	requiredTypes := []string{
		"Environment",
		"EnvironmentSummary",
		"Host",
		"Vlan",
		"ConnectivityInfo",
		"VcfVersionInfo",
	}
	for _, typeName := range requiredTypes {
		if _, ok := evsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestEVSStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := evsRequest(t, ts, http.MethodPost, "/unknown", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestEVSStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := evsRequest(t, ts, http.MethodGet, "/environments", ``)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "environments") {
		t.Fatalf("expected ListEnvironments response body to include environments, got %q", body)
	}
}

func TestEVSStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range evsOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) {
			payload = ""
		}
		resp := evsRequest(t, ts, op.Method, evsPathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
