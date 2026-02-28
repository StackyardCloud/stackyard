package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCloudWatchInvestigationsStage0CatalogCoverage(t *testing.T) {
	if len(cloudWatchInvestigationsOperations) != 11 {
		t.Fatalf("expected 11 CloudWatch Investigations operations from docs, got %d", len(cloudWatchInvestigationsOperations))
	}
	if len(cloudWatchInvestigationsOperationByName) != len(cloudWatchInvestigationsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateInvestigationGroup",
		"GetInvestigationGroup",
		"UpdateInvestigationGroup",
		"PutInvestigationGroupPolicy",
		"ListInvestigationGroups",
	}
	for _, action := range requiredActions {
		if _, ok := cloudWatchInvestigationsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(cloudWatchInvestigationsDataTypes) != 3 {
		t.Fatalf("expected 3 CloudWatch Investigations data types from docs, got %d", len(cloudWatchInvestigationsDataTypes))
	}
	if len(cloudWatchInvestigationsDataTypeByName) != len(cloudWatchInvestigationsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"CrossAccountConfiguration",
		"EncryptionConfiguration",
		"ListInvestigationGroupsModel",
	}
	for _, typeName := range requiredTypes {
		if _, ok := cloudWatchInvestigationsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func cloudWatchInvestigationsRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "aiops")
}

func cloudWatchInvestigationsPathForTest(template string) string {
	out := template
	out = strings.ReplaceAll(out, "{identifier}", "stackyard-investigation-group")
	out = strings.ReplaceAll(out, "{resourceArn}", url.PathEscape("arn:aws:aiops:us-east-1:123456789012:investigation-group/stackyard-investigation-group"))
	return out
}

func TestCloudWatchInvestigationsStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudWatchInvestigationsRequest(t, ts, http.MethodGet, "/investigationGroups/stackyard-investigation-group/unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCloudWatchInvestigationsKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudWatchInvestigationsRequest(t, ts, http.MethodGet, "/investigationGroups", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "investigationGroups") {
		t.Fatalf("expected response body to include investigationGroups, got %q", body)
	}
}

func TestCloudWatchInvestigationsAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range cloudWatchInvestigationsOperations {
		path := cloudWatchInvestigationsPathForTest(op.URI)
		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}
		resp := cloudWatchInvestigationsRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
