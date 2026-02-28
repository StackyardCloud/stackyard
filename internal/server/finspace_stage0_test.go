package server

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func finspaceRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "finspace-api")
}

func TestFinSpaceStage0CatalogCoverage(t *testing.T) {
	if len(finspaceOperations) != 31 {
		t.Fatalf("expected 31 FinSpace operations from docs, got %d", len(finspaceOperations))
	}
	if len(finspaceOperationByName) != len(finspaceOperations) {
		t.Fatalf("expected unique FinSpace operation names")
	}

	requiredActions := []string{
		"CreateDataset",
		"GetDataset",
		"ListDatasets",
		"CreateUser",
		"GetUser",
		"ListUsers",
		"CreatePermissionGroup",
		"ListPermissionGroups",
		"AssociateUserToPermissionGroup",
		"GetProgrammaticAccessCredentials",
	}
	for _, action := range requiredActions {
		if _, ok := finspaceOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(finspaceDataTypes) != 20 {
		t.Fatalf("expected 20 FinSpace data types from docs, got %d", len(finspaceDataTypes))
	}
	if len(finspaceDataTypeByName) != len(finspaceDataTypes) {
		t.Fatalf("expected unique FinSpace data type names")
	}

	requiredTypes := []string{
		"Dataset",
		"DataViewSummary",
		"ChangesetSummary",
		"User",
		"PermissionGroup",
	}
	for _, typeName := range requiredTypes {
		if _, ok := finspaceDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestFinSpaceStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := finspaceRequest(t, ts, http.MethodPost, "/finspace-unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestFinSpaceStage0KnownActionReturnsListDatasets(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := finspaceRequest(t, ts, http.MethodGet, "/datasetsv2", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "datasets") {
		t.Fatalf("expected ListDatasets response body to include datasets, got %q", body)
	}
}

func TestFinSpaceStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"datasetId":         "dataset-000001",
		"dataviewId":        "dataview-000001",
		"dataViewId":        "dataview-000001",
		"changesetId":       "changeset-000001",
		"permissionGroupId": "permission-group-000001",
		"userId":            "user-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range finspaceOperations {
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

		resp := finspaceRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
