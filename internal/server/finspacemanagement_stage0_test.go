package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func finspaceManagementRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "finspace")
}

func TestFinSpaceManagementStage0CatalogCoverage(t *testing.T) {
	if len(finspaceManagementOperations) != 50 {
		t.Fatalf("expected 50 FinSpace Management operations from docs, got %d", len(finspaceManagementOperations))
	}
	if len(finspaceManagementOperationByName) != len(finspaceManagementOperations) {
		t.Fatalf("expected unique FinSpace Management operation names")
	}

	requiredActions := []string{
		"CreateEnvironment",
		"GetEnvironment",
		"ListEnvironments",
		"CreateKxCluster",
		"GetKxCluster",
		"ListKxClusters",
		"TagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := finspaceManagementOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(finspaceManagementDataTypes) != 39 {
		t.Fatalf("expected 39 FinSpace Management data types from docs, got %d", len(finspaceManagementDataTypes))
	}
	if len(finspaceManagementDataTypeByName) != len(finspaceManagementDataTypes) {
		t.Fatalf("expected unique FinSpace Management data type names")
	}

	requiredTypes := []string{
		"Environment",
		"KxEnvironment",
		"KxCluster",
		"KxDatabaseConfiguration",
		"KxVolume",
	}
	for _, typeName := range requiredTypes {
		if _, ok := finspaceManagementDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestFinSpaceManagementStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := finspaceManagementRequest(t, ts, http.MethodPost, "/finspace-mgmt-unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestFinSpaceManagementStage0KnownActionReturnsListEnvironments(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := finspaceManagementRequest(t, ts, http.MethodGet, "/environment?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "environments") {
		t.Fatalf("expected ListEnvironments response body to include environments, got %q", body)
	}
}

func TestFinSpaceManagementStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"environmentId":    "env-000001",
		"databaseName":     "db-000001",
		"clusterName":      "cluster-000001",
		"nodeId":           "node-000001",
		"dataviewName":     "dataview-000001",
		"scalingGroupName": "scaling-group-000001",
		"userName":         "user-000001",
		"volumeName":       "volume-000001",
		"changesetId":      "changeset-000001",
		"resourceArn":      url.PathEscape("arn:aws:finspace:us-east-1:123456789012:environment/env-000001"),
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range finspaceManagementOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			if value := replacements[name]; value != "" {
				return value
			}
			return "stackyard"
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := finspaceManagementRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
