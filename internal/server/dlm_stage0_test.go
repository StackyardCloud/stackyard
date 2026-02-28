package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func dlmRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "dlm")
}

func dlmPathForOperation(op dlmOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{policyId}":          "policy-00000001",
		"{DefaultPolicyType}": "EBS_SNAPSHOT_MANAGEMENT",
		"{PolicyIds}":         "policy-00000001",
		"{ResourceTypes}":     "VOLUME",
		"{State}":             "ENABLED",
		"{TagsToAdd}":         "env=dev",
		"{TargetTags}":        "Name=stackyard",
		"{TagKeys}":           "seed",
		"{resourceArn}":       url.PathEscape("arn:aws:dlm:us-east-1:123456789012:policy/policy-00000001"),
	}
	for key, value := range replacements {
		path = strings.ReplaceAll(path, key, value)
	}
	return path
}

func TestDLMStage0CatalogCoverage(t *testing.T) {
	if len(dlmOperations) != 8 {
		t.Fatalf("expected 8 DLM operations from docs, got %d", len(dlmOperations))
	}
	if len(dlmOperationByName) != len(dlmOperations) {
		t.Fatalf("expected unique DLM operation names")
	}

	requiredActions := []string{
		"CreateLifecyclePolicy",
		"DeleteLifecyclePolicy",
		"GetLifecyclePolicies",
		"GetLifecyclePolicy",
		"ListTagsForResource",
		"TagResource",
		"UntagResource",
		"UpdateLifecyclePolicy",
	}
	for _, action := range requiredActions {
		if _, ok := dlmOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(dlmDataTypes) != 26 {
		t.Fatalf("expected 26 DLM data types from docs, got %d", len(dlmDataTypes))
	}
	if len(dlmDataTypeByName) != len(dlmDataTypes) {
		t.Fatalf("expected unique DLM data type names")
	}

	requiredTypes := []string{
		"LifecyclePolicy",
		"LifecyclePolicySummary",
		"PolicyDetails",
		"Schedule",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := dlmDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestDLMStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dlmRequest(t, ts, http.MethodPost, "/unknown", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestDLMStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dlmRequest(t, ts, http.MethodGet, "/policies", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Policies") {
		t.Fatalf("expected GetLifecyclePolicies response body to include Policies, got %q", body)
	}
}

func TestDLMStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range dlmOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := dlmRequest(t, ts, op.Method, dlmPathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
