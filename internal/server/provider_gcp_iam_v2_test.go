package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPIAMV2Router_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMV2ContractServer(t)
	parent := "/gcp/v2/policies/cloudresourcemanager.googleapis.com%2Fprojects%2Fstackyard/denypolicies"
	policy := parent + "/deny-read-access"

	assertGCPIAMV2Success(t, ts, http.MethodGet, parent+"?pageSize=1", nil, "policies")
	assertGCPIAMV2Success(t, ts, http.MethodGet, policy, nil, "deny-read-access")
	assertGCPIAMV2Success(t, ts, http.MethodPost, parent+"?policyId=deny-read-access", []byte(`{"policy":{"displayName":"stackyard deny policy"}}`), "operations")
	assertGCPIAMV2Success(t, ts, http.MethodPut, policy, []byte(`{"policy":{"displayName":"updated"}}`), "operations")
	assertGCPIAMV2Success(t, ts, http.MethodDelete, policy, nil, "operations")
	assertGCPIAMV2Success(t, ts, http.MethodGet, "/gcp/v2/operations/op-1", nil, "operations/op-1")
}

func TestGCPIAMV2Router_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMV2ContractServer(t)
	assertGCPIAMV2NotImplemented(t, ts, http.MethodPost, "/gcp/google.iam.v2.Policies/CreatePolicy", "Policies/CreatePolicy")
}

func TestGCPIAMV2Router_ListPoliciesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMV2ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/policies/cloudresourcemanager.googleapis.com%2Fprojects%2Fstackyard/denypolicies?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp iam v2 router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func TestGCPIAMV2Router_CreatePolicyRequiresPolicyID(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMV2ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/policies/cloudresourcemanager.googleapis.com%2Fprojects%2Fstackyard/denypolicies", []byte(`{"policy":{"displayName":"deny"}}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp iam v2 router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func newGCPIAMV2ContractServer(t *testing.T) *httptest.Server {
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

func assertGCPIAMV2NotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp iam v2 router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPIAMV2Success(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp iam v2 router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, string(providerContractBody(t, resp)))
	}
}
