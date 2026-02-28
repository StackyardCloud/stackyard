package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func recycleBinRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "rbin")
}

func recycleBinPathForOperation(op recycleBinOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{identifier}":  "rbin-00000001",
		"{resourceArn}": url.PathEscape("arn:aws:rbin:us-east-1:123456789012:rule/rbin-00000001"),
	}
	for k, v := range replacements {
		path = strings.ReplaceAll(path, k, v)
	}
	return path
}

func recycleBinPayloadForOperation(op recycleBinOperation) string {
	switch op.Name {
	case "CreateRule":
		return `{"ResourceType":"EBS_SNAPSHOT","RetentionPeriod":{"RetentionPeriodValue":30,"RetentionPeriodUnit":"DAYS"},"Description":"stackyard rule"}`
	case "ListRules":
		return `{"MaxResults":10}`
	case "LockRule":
		return `{"LockConfiguration":{"UnlockDelay":{"UnlockDelayValue":7,"UnlockDelayUnit":"DAYS"}}}`
	case "TagResource":
		return `{"Tags":[{"Key":"env","Value":"dev"}]}`
	case "UntagResource":
		return `{"TagKeys":["env"]}`
	case "UpdateRule":
		return `{"Description":"updated rule"}`
	}
	if strings.EqualFold(op.Method, http.MethodPost) || strings.EqualFold(op.Method, http.MethodPatch) {
		return `{}`
	}
	return ""
}

func TestRecycleBinStage0CatalogCoverage(t *testing.T) {
	if len(recycleBinOperations) != 10 {
		t.Fatalf("expected 10 Recycle Bin operations from docs, got %d", len(recycleBinOperations))
	}
	if len(recycleBinOperationByName) != len(recycleBinOperations) {
		t.Fatalf("expected unique Recycle Bin operation names")
	}

	requiredActions := []string{
		"CreateRule",
		"GetRule",
		"ListRules",
		"LockRule",
		"UnlockRule",
		"TagResource",
		"ListTagsForResource",
		"UpdateRule",
		"DeleteRule",
	}
	for _, action := range requiredActions {
		if _, ok := recycleBinOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(recycleBinDataTypes) != 31 {
		t.Fatalf("expected 31 Recycle Bin data types from docs, got %d", len(recycleBinDataTypes))
	}
	if len(recycleBinDataTypeByName) != len(recycleBinDataTypes) {
		t.Fatalf("expected unique Recycle Bin data type names")
	}

	requiredTypes := []string{
		"RetentionPeriod",
		"RuleSummary",
		"LockConfiguration",
		"UnlockDelay",
		"Tag",
		"ResourceTag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := recycleBinDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestRecycleBinStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := recycleBinRequest(t, ts, http.MethodGet, "/not-a-real-recycle-bin-route", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestRecycleBinStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := recycleBinRequest(t, ts, http.MethodPost, "/list-rules", `{"MaxResults":10}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Rules") {
		t.Fatalf("expected ListRules response body to include Rules, got %q", body)
	}
}

func TestRecycleBinStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range recycleBinOperations {
		resp := recycleBinRequest(t, ts, op.Method, recycleBinPathForOperation(op), recycleBinPayloadForOperation(op))
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
