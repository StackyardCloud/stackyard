package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPIAMCredentialsRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMCredentialsContractServer(t)
	base := "/gcp/v1/projects/stackyard/serviceAccounts/stackyard@example.iam.gserviceaccount.com"

	assertGCPIAMCredentialsSuccess(t, ts, http.MethodPost, base+":generateAccessToken", []byte(`{"scope":["https://www.googleapis.com/auth/cloud-platform"]}`), "accessToken")
	assertGCPIAMCredentialsSuccess(t, ts, http.MethodPost, base+":generateIdToken", []byte(`{"audience":"https://example.com","includeEmail":true}`), "token")
	assertGCPIAMCredentialsSuccess(t, ts, http.MethodPost, base+":signBlob", []byte(`{"payload":"c3RhY2t5YXJkLWJsb2I="}`), "signedBlob")
	assertGCPIAMCredentialsSuccess(t, ts, http.MethodPost, base+":signJwt", []byte(`{"payload":"{\"sub\":\"stackyard@example.com\"}"}`), "signedJwt")
}

func TestGCPIAMCredentialsRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMCredentialsContractServer(t)
	assertGCPIAMCredentialsNotImplemented(t, ts, http.MethodPost, "/gcp/google.iam.credentials.v1.IAMCredentials/GenerateAccessToken", "IAMCredentials/GenerateAccessToken")
}

func TestGCPIAMCredentialsRouter_GenerateAccessTokenRequiresScope(t *testing.T) {
	t.Parallel()

	ts := newGCPIAMCredentialsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/serviceAccounts/stackyard@example.iam.gserviceaccount.com:generateAccessToken", []byte(`{"scope":[]}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp iam credentials router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func newGCPIAMCredentialsContractServer(t *testing.T) *httptest.Server {
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

func assertGCPIAMCredentialsNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp iam credentials router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPIAMCredentialsSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp iam credentials router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, string(providerContractBody(t, resp)))
	}
}
