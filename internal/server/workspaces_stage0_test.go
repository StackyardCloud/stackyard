package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func workspacesRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "WorkspacesService." + action,
		},
		"workspaces",
	)
}

func TestWorkSpacesStage0CatalogCoverage(t *testing.T) {
	if len(workspacesOperations) != 91 {
		t.Fatalf("expected 91 WorkSpaces operations from docs, got %d", len(workspacesOperations))
	}
	if len(workspacesOperationByName) != len(workspacesOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateWorkspaces",
		"DescribeWorkspaces",
		"CreateWorkspaceBundle",
		"CreateWorkspaceImage",
		"CreateWorkspacesPool",
		"CreateTags",
		"DescribeTags",
	}
	for _, action := range requiredActions {
		if _, ok := workspacesOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(workspacesDataTypes) != 80 {
		t.Fatalf("expected 80 WorkSpaces data types from docs, got %d", len(workspacesDataTypes))
	}
	if len(workspacesDataTypeByName) != len(workspacesDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Workspace",
		"WorkspaceBundle",
		"WorkspaceImage",
		"WorkspacesPool",
		"WorkspacesPoolSession",
		"WorkspacesIpGroup",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := workspacesDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestWorkSpacesStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestWorkSpacesStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesRequest(t, ts, "DescribeWorkspaces", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Workspaces") {
		t.Fatalf("expected DescribeWorkspaces response body to include Workspaces, got %q", body)
	}
}

func TestWorkSpacesStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range workspacesOperations {
		resp := workspacesRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
