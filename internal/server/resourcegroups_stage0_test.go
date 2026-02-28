package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func resourceGroupsRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "resource-groups")
}

func resourceGroupsPathForTest(template string) string {
	resourceARN := "arn:aws:resource-groups:us-east-1:123456789012:group/stackyard-group"
	out := template
	out = strings.ReplaceAll(out, "{Arn}", url.PathEscape(resourceARN))
	return out
}

func TestResourceGroupsStage0CatalogCoverage(t *testing.T) {
	if len(resourceGroupsOperations) != 23 {
		t.Fatalf("expected 23 Resource Groups operations from docs, got %d", len(resourceGroupsOperations))
	}
	if len(resourceGroupsOperationByName) != len(resourceGroupsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateGroup",
		"ListGroups",
		"SearchResources",
		"Tag",
		"Untag",
	}
	for _, action := range requiredActions {
		if _, ok := resourceGroupsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(resourceGroupsDataTypes) != 21 {
		t.Fatalf("expected 21 Resource Groups data types from docs, got %d", len(resourceGroupsDataTypes))
	}
	if len(resourceGroupsDataTypeByName) != len(resourceGroupsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"AccountSettings",
		"Group",
		"GroupQuery",
		"ResourceIdentifier",
		"TagSyncTaskItem",
	}
	for _, typeName := range requiredTypes {
		if _, ok := resourceGroupsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestResourceGroupsUnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := resourceGroupsRequest(t, ts, http.MethodPost, "/resource-groups-unknown-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestResourceGroupsAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range resourceGroupsOperations {
		path := resourceGroupsPathForTest(op.URI)
		body := []byte{}
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			switch op.Name {
			case "Tag":
				body = []byte(`{"Tags":{"stackyard":"true"}}`)
			case "Untag":
				body = []byte(`{"Keys":["stackyard"]}`)
			default:
				body = []byte(`{}`)
			}
		}

		resp := resourceGroupsRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
