package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPIAMRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMContractServer(t)

	assertGCPIAMSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:getIamPolicy", []byte(`{"resource":"projects/stackyard"}`), "bindings")
	assertGCPIAMSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:setIamPolicy", []byte(`{"policy":{"bindings":[{"role":"roles/viewer","members":["user:alice@example.com"]}]}}`), "bindings")
	assertGCPIAMSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:testIamPermissions", []byte(`{"permissions":["resourcemanager.projects.get"]}`), "permissions")
}

func TestGCPIAMRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMContractServer(t)
	assertGCPIAMNotImplemented(t, ts, http.MethodPost, "/gcp/google.iam.v1.IAMPolicy/GetIamPolicy", "IAMPolicy/GetIamPolicy")
}

func TestGCPIAMRouter_SetIamPolicyRequiresPolicy(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:setIamPolicy", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp iam router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func TestGCPIAMRouter_TestPermissionsRequiresPermissions(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:testIamPermissions", []byte(`{"permissions":[]}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp iam router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func newGCPIAMContractServer(t *testing.T) *httptest.Server {
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

func assertGCPIAMNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp iam router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPIAMSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp iam router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, string(providerContractBody(t, resp)))
	}
}
