package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func resourceGroupsTaggingAPIRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "ResourceGroupsTaggingAPI_20170126." + action,
		},
		"tagging",
	)
}

func TestResourceGroupsTaggingAPIStage0CatalogCoverage(t *testing.T) {
	if len(resourceGroupsTaggingAPIOperations) != 8 {
		t.Fatalf("expected 8 Resource Groups Tagging API operations from docs, got %d", len(resourceGroupsTaggingAPIOperations))
	}
	if len(resourceGroupsTaggingAPIOperationByName) != len(resourceGroupsTaggingAPIOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"GetResources",
		"GetTagKeys",
		"GetTagValues",
		"TagResources",
		"UntagResources",
	}
	for _, action := range requiredActions {
		if _, ok := resourceGroupsTaggingAPIOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(resourceGroupsTaggingAPIDataTypes) != 6 {
		t.Fatalf("expected 6 Resource Groups Tagging API data types from docs, got %d", len(resourceGroupsTaggingAPIDataTypes))
	}
	if len(resourceGroupsTaggingAPIDataTypeByName) != len(resourceGroupsTaggingAPIDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"ComplianceDetails",
		"FailureInfo",
		"ResourceTagMapping",
		"Summary",
		"Tag",
		"TagFilter",
	}
	for _, typeName := range requiredTypes {
		if _, ok := resourceGroupsTaggingAPIDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestResourceGroupsTaggingAPIStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := resourceGroupsTaggingAPIRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestResourceGroupsTaggingAPIKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := resourceGroupsTaggingAPIRequest(t, ts, "GetResources", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "ResourceTagMappingList") {
		t.Fatalf("expected GetResources response body to include ResourceTagMappingList, got %q", body)
	}
}

func TestResourceGroupsTaggingAPIStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range resourceGroupsTaggingAPIOperations {
		resp := resourceGroupsTaggingAPIRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
