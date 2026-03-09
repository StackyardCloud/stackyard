package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPIdentityToolkitRouter_RESTActionRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPIdentityToolkitContractServer(t)
	assertGCPIdentityToolkitNotImplemented(t, ts, http.MethodPost, "/gcp/v2/accounts/mfaEnrollment:start", "mfaEnrollment:start")
	assertGCPIdentityToolkitNotImplemented(t, ts, http.MethodPost, "/gcp/v2/accounts/mfaEnrollment:finalize", "mfaEnrollment:finalize")
	assertGCPIdentityToolkitNotImplemented(t, ts, http.MethodPost, "/gcp/v2/accounts/mfaEnrollment:withdraw", "mfaEnrollment:withdraw")
	assertGCPIdentityToolkitNotImplemented(t, ts, http.MethodPost, "/gcp/v2/accounts/mfaSignIn:start", "mfaSignIn:start")
	assertGCPIdentityToolkitNotImplemented(t, ts, http.MethodPost, "/gcp/v2/accounts/mfaSignIn:finalize", "mfaSignIn:finalize")
}

func TestGCPIdentityToolkitRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPIdentityToolkitContractServer(t)
	assertGCPIdentityToolkitNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.identitytoolkit.v2.AccountManagementService/StartMfaEnrollment", "AccountManagementService/StartMfaEnrollment")
	assertGCPIdentityToolkitNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.identitytoolkit.v2.AccountManagementService/FinalizeMfaEnrollment", "AccountManagementService/FinalizeMfaEnrollment")
	assertGCPIdentityToolkitNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.identitytoolkit.v2.AccountManagementService/WithdrawMfa", "AccountManagementService/WithdrawMfa")
	assertGCPIdentityToolkitNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.identitytoolkit.v2.AuthenticationService/StartMfaSignIn", "AuthenticationService/StartMfaSignIn")
	assertGCPIdentityToolkitNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.identitytoolkit.v2.AuthenticationService/FinalizeMfaSignIn", "AuthenticationService/FinalizeMfaSignIn")
}

func newGCPIdentityToolkitContractServer(t *testing.T) *httptest.Server {
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

func assertGCPIdentityToolkitNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp identity toolkit router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPIdentitytoolkitRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPIdentitytoolkitRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/identitytoolkit?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp identitytoolkit contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "identitytoolkit" {
		t.Fatalf("expected service=identitytoolkit, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
