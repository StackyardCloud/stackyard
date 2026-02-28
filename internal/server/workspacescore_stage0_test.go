package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func workspacesCoreRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
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

func TestWorkSpacesCoreStage0CatalogCoverage(t *testing.T) {
	if len(workspacesCoreOperations) != 34 {
		t.Fatalf("expected 34 WorkSpaces Core actions from docs, got %d", len(workspacesCoreOperations))
	}
	if len(workspacesCoreOperationByName) != len(workspacesCoreOperations) {
		t.Fatalf("expected unique Core action names")
	}

	requiredActions := []string{
		"CreateWorkspaces",
		"DescribeWorkspaces",
		"CreateWorkspaceBundle",
		"CreateWorkspaceImage",
		"CreateTags",
		"DescribeTags",
		"DeleteTags",
		"TerminateWorkspaces",
	}
	for _, action := range requiredActions {
		if _, ok := workspacesCoreOperationByName[action]; !ok {
			t.Fatalf("missing documented Core action %s", action)
		}
		if _, ok := workspacesOperationByName[action]; !ok {
			t.Fatalf("Core action %s must be implemented by WorkSpaces router", action)
		}
	}
}

func TestWorkSpacesCoreStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesCoreRequest(t, ts, "TotallyUnknownCoreAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestWorkSpacesCoreStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := workspacesCoreRequest(t, ts, "DescribeAccount", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "DedicatedTenancySupport") {
		t.Fatalf("expected DescribeAccount response body to include DedicatedTenancySupport, got %q", body)
	}
}

func TestWorkSpacesCoreStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range workspacesCoreOperations {
		resp := workspacesCoreRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
