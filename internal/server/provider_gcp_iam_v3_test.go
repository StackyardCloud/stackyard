package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPIAMV3Router_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMV3ContractServer(t)
	bindingParent := "/gcp/v3/projects/stackyard/locations/global"
	bindingName := bindingParent + "/policyBindings/binding-a"
	pabParent := "/gcp/v3/organizations/123456789012/locations/global"
	pabName := pabParent + "/principalAccessBoundaryPolicies/pab-a"

	assertGCPIAMV3Success(t, ts, http.MethodGet, bindingParent+"/policyBindings?pageSize=1", nil, "policyBindings")
	assertGCPIAMV3Success(t, ts, http.MethodPost, bindingParent+"/policyBindings", []byte(`{"policyBinding":{"displayName":"binding"},"policyBindingId":"binding-a"}`), "operations")
	assertGCPIAMV3Success(t, ts, http.MethodGet, bindingName, nil, "binding-a")
	assertGCPIAMV3Success(t, ts, http.MethodPatch, bindingName, []byte(`{"policyBinding":{"displayName":"updated"}}`), "operations")
	assertGCPIAMV3Success(t, ts, http.MethodDelete, bindingName, nil, "operations")
	assertGCPIAMV3Success(t, ts, http.MethodGet, bindingParent+"/policyBindings:searchTargetPolicyBindings?target=%2F%2Fcloudresourcemanager.googleapis.com%2Fprojects%2Fstackyard&pageSize=1", nil, "policyBindings")

	assertGCPIAMV3Success(t, ts, http.MethodGet, pabParent+"/principalAccessBoundaryPolicies?pageSize=1", nil, "principalAccessBoundaryPolicies")
	assertGCPIAMV3Success(t, ts, http.MethodPost, pabParent+"/principalAccessBoundaryPolicies", []byte(`{"principalAccessBoundaryPolicy":{"displayName":"pab"},"principalAccessBoundaryPolicyId":"pab-a"}`), "operations")
	assertGCPIAMV3Success(t, ts, http.MethodGet, pabName, nil, "pab-a")
	assertGCPIAMV3Success(t, ts, http.MethodPatch, pabName, []byte(`{"principalAccessBoundaryPolicy":{"displayName":"updated"}}`), "operations")
	assertGCPIAMV3Success(t, ts, http.MethodDelete, pabName, nil, "operations")
	assertGCPIAMV3Success(t, ts, http.MethodGet, pabName+":searchPolicyBindings?pageSize=1", nil, "policyBindings")

	assertGCPIAMV3Success(t, ts, http.MethodGet, "/gcp/v3/operations/op-1", nil, "operations/op-1")
}

func TestGCPIAMV3Router_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMV3ContractServer(t)
	assertGCPIAMV3NotImplemented(t, ts, http.MethodPost, "/gcp/google.iam.v3.PolicyBindings/CreatePolicyBinding", "PolicyBindings/CreatePolicyBinding")
}

func TestGCPIAMV3Router_SearchTargetRequiresTarget(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMV3ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v3/projects/stackyard/locations/global/policyBindings:searchTargetPolicyBindings", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp iam v3 router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func TestGCPIAMV3Router_CreatePABRequiresPolicyBody(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMV3ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v3/organizations/123456789012/locations/global/principalAccessBoundaryPolicies", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp iam v3 router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func newGCPIAMV3ContractServer(t *testing.T) *httptest.Server {
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

func assertGCPIAMV3NotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp iam v3 router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPIAMV3Success(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp iam v3 router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, string(providerContractBody(t, resp)))
	}
}
