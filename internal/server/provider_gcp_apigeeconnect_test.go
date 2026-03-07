package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPApigeeConnectRouter_ListConnectionsRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPApigeeConnectContractServer(t)
	assertGCPApigeeConnectSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/endpoints/local/connections?pageSize=1", nil, "connections")
}

func TestGCPApigeeConnectRouter_EgressRouteRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPApigeeConnectContractServer(t)
	assertGCPApigeeConnectSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/endpoints/local:egress", []byte(`{"id":"egress-bootstrap"}`), "httpResponse")
}

func TestGCPApigeeConnectRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPApigeeConnectContractServer(t)
	assertGCPApigeeConnectNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.apigeeconnect.v1.ConnectionService/ListConnections", "ConnectionService/ListConnections")
	assertGCPApigeeConnectNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.apigeeconnect.v1.Tether/Egress", "Tether/Egress")
}

func TestGCPApigeeConnectRouter_ListConnectionsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPApigeeConnectContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/endpoints/local/connections?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp apigeeconnect router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPApigeeConnectRouter_EgressInvalidJSON(t *testing.T) {
	t.Parallel()

	ts := newGCPApigeeConnectContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/endpoints/local:egress", []byte(`{"id"`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp apigeeconnect router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPApigeeConnectRouter_EgressRequiresID(t *testing.T) {
	t.Parallel()

	ts := newGCPApigeeConnectContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/endpoints/local:egress", []byte(`{"name":"projects/stackyard/endpoints/local"}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp apigeeconnect router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPApigeeConnectContractServer(t *testing.T) *httptest.Server {
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

func assertGCPApigeeConnectNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp apigeeconnect router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPApigeeConnectSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp apigeeconnect router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
