package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPIAMAdminRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMAdminContractServer(t)
	project := "/gcp/v1/projects/stackyard"
	account := project + "/serviceAccounts/stackyard@example.iam.gserviceaccount.com"
	role := project + "/roles/customViewer"

	assertGCPIAMAdminSuccess(t, ts, http.MethodGet, project+"/serviceAccounts?pageSize=1", nil, "accounts")
	assertGCPIAMAdminSuccess(t, ts, http.MethodPost, project+"/serviceAccounts", []byte(`{"accountId":"stackyard-sa","serviceAccount":{"displayName":"Stackyard IAM Admin SA"}}`), "serviceAccounts")
	assertGCPIAMAdminSuccess(t, ts, http.MethodGet, account, nil, "serviceAccounts")
	assertGCPIAMAdminSuccess(t, ts, http.MethodPatch, account, []byte(`{"serviceAccount":{"displayName":"Updated"}}`), "serviceAccounts")
	assertGCPIAMAdminSuccess(t, ts, http.MethodPost, account+":disable", []byte(`{}`), "{}")
	assertGCPIAMAdminSuccess(t, ts, http.MethodPost, account+":enable", []byte(`{}`), "{}")
	assertGCPIAMAdminSuccess(t, ts, http.MethodPost, account+":undelete", []byte(`{}`), "serviceAccounts")
	assertGCPIAMAdminSuccess(t, ts, http.MethodDelete, account, nil, "{}")

	assertGCPIAMAdminSuccess(t, ts, http.MethodGet, "/gcp/v1/roles?pageSize=1", nil, "roles")
	assertGCPIAMAdminSuccess(t, ts, http.MethodGet, role, nil, "customViewer")
	assertGCPIAMAdminSuccess(t, ts, http.MethodPost, project+"/roles?roleId=customViewer", []byte(`{"role":{"title":"Stackyard Custom Viewer"}}`), "customViewer")
	assertGCPIAMAdminSuccess(t, ts, http.MethodPatch, role, []byte(`{"role":{"title":"Updated"}}`), "customViewer")
	assertGCPIAMAdminSuccess(t, ts, http.MethodDelete, role, nil, "{}")

	assertGCPIAMAdminSuccess(t, ts, http.MethodPost, "/gcp/v1/roles:queryGrantableRoles", []byte(`{"fullResourceName":"//cloudresourcemanager.googleapis.com/projects/stackyard"}`), "roles")
	assertGCPIAMAdminSuccess(t, ts, http.MethodPost, "/gcp/v1/permissions:queryTestablePermissions", []byte(`{"fullResourceName":"//cloudresourcemanager.googleapis.com/projects/stackyard"}`), "permissions")
	assertGCPIAMAdminSuccess(t, ts, http.MethodPost, "/gcp/v1/iamPolicies:queryAuditableServices", []byte(`{"fullResourceName":"//cloudresourcemanager.googleapis.com/projects/stackyard"}`), "services")
	assertGCPIAMAdminSuccess(t, ts, http.MethodPost, "/gcp/v1/iamPolicies:lintPolicy", []byte(`{"fullResourceName":"//cloudresourcemanager.googleapis.com/projects/stackyard"}`), "lintResults")
}

func TestGCPIAMAdminRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMAdminContractServer(t)
	assertGCPIAMAdminNotImplemented(t, ts, http.MethodPost, "/gcp/google.iam.admin.v1.IAM/ListServiceAccounts", "IAM/ListServiceAccounts")
}

func TestGCPIAMAdminRouter_ListServiceAccountsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMAdminContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/serviceAccounts?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp iam admin router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func TestGCPIAMAdminRouter_CreateRoleRequiresRole(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMAdminContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/roles?roleId=customViewer", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp iam admin router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func newGCPIAMAdminContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
}

func assertGCPIAMAdminNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp iam admin router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPIAMAdminSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp iam admin router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, string(providerContractBody(t, resp)))
	}
}
