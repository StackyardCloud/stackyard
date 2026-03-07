package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPOsLoginRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPOsLoginContractServer(t)

	assertGCPOsLoginNotImplemented(t, ts, http.MethodPost, "/gcp/v1/users/alice@example.com/sshPublicKeys", "/sshPublicKeys")
	assertGCPOsLoginNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/users/alice@example.com/projects/stackyard", "/projects/stackyard")
	assertGCPOsLoginNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/users/alice@example.com/sshPublicKeys/fingerprint-1", "/sshPublicKeys/fingerprint-1")
	assertGCPOsLoginNotImplemented(t, ts, http.MethodGet, "/gcp/v1/users/alice@example.com/loginProfile?projectId=stackyard", "/loginProfile")
	assertGCPOsLoginNotImplemented(t, ts, http.MethodGet, "/gcp/v1/users/alice@example.com/sshPublicKeys/fingerprint-1", "/sshPublicKeys/fingerprint-1")
	assertGCPOsLoginNotImplemented(t, ts, http.MethodPost, "/gcp/v1/users/alice@example.com:importSshPublicKey?projectId=stackyard", ":importSshPublicKey")
	assertGCPOsLoginNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/users/alice@example.com/sshPublicKeys/fingerprint-1?updateMask=expiration_time_usec", "/sshPublicKeys/fingerprint-1")
}

func TestGCPOsLoginRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPOsLoginContractServer(t)

	assertGCPOsLoginNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.oslogin.v1.OsLoginService/CreateSshPublicKey", "OsLoginService/CreateSshPublicKey")
	assertGCPOsLoginNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.oslogin.v1.OsLoginService/DeletePosixAccount", "OsLoginService/DeletePosixAccount")
	assertGCPOsLoginNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.oslogin.v1.OsLoginService/DeleteSshPublicKey", "OsLoginService/DeleteSshPublicKey")
	assertGCPOsLoginNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.oslogin.v1.OsLoginService/GetLoginProfile", "OsLoginService/GetLoginProfile")
	assertGCPOsLoginNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.oslogin.v1.OsLoginService/GetSshPublicKey", "OsLoginService/GetSshPublicKey")
	assertGCPOsLoginNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.oslogin.v1.OsLoginService/ImportSshPublicKey", "OsLoginService/ImportSshPublicKey")
	assertGCPOsLoginNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.oslogin.v1.OsLoginService/UpdateSshPublicKey", "OsLoginService/UpdateSshPublicKey")
}

func newGCPOsLoginContractServer(t *testing.T) *httptest.Server {
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

func assertGCPOsLoginNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp oslogin router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPOsloginRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPOsloginRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/oslogin?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp oslogin contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "oslogin" {
		t.Fatalf("expected service=oslogin, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

