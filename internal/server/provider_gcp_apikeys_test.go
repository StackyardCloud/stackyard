package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPAPIKeysRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPAPIKeysContractServer(t)

	assertGCPAPIKeysSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/global/keys?pageSize=1", nil, "keys")
	assertGCPAPIKeysSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/global/keys/team-key", nil, "keys/team-key")
	assertGCPAPIKeysSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/global/keys/team-key/keyString", nil, "stackyard-demo-key")
	assertGCPAPIKeysSuccess(t, ts, http.MethodPost, "/gcp/v2/keys:lookupKey", []byte(`{"keyString":"stackyard-demo-key"}`), "projects/stackyard/locations/global/keys/team-key")
}

func TestGCPAPIKeysRouter_ListKeysInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPAPIKeysContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/global/keys?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp apikeys router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPAPIKeysRouter_LookupKeyInvalidJSON(t *testing.T) {
	t.Parallel()

	ts := newGCPAPIKeysContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/keys:lookupKey", []byte(`{"keyString"`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp apikeys router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPAPIKeysRouter_LookupKeyRequiresKeyString(t *testing.T) {
	t.Parallel()

	ts := newGCPAPIKeysContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/keys:lookupKey", []byte(`{"name":"projects/stackyard/locations/global/keys/team-key"}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp apikeys router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPAPIKeysContractServer(t *testing.T) *httptest.Server {
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

func assertGCPAPIKeysSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()
	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp apikeys router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
