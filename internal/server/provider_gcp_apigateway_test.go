package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPAPIGatewayRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPAPIGatewayContractServer(t)

	assertGCPAPIGatewaySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/apis?pageSize=1", nil, "apis")
	assertGCPAPIGatewaySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/apis/team-api", nil, "apis/team-api")
	assertGCPAPIGatewaySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/apis/team-api/configs?pageSize=1", nil, "apiConfigs")
	assertGCPAPIGatewaySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/apis/team-api/configs/team-config", nil, "configs/team-config")
	assertGCPAPIGatewaySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/gateways?pageSize=1", nil, "gateways")
	assertGCPAPIGatewaySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/gateways/team-gateway", nil, "gateways/team-gateway")
}

func TestGCPAPIGatewayRouter_ListApisInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPAPIGatewayContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/apis?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp apigateway router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPAPIGatewayRouter_ListGatewaysPageTokenOutOfRange(t *testing.T) {
	t.Parallel()

	ts := newGCPAPIGatewayContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/gateways?pageToken=99", nil, map[string]string{
		"X-Stackyard-GCP-Service": "apigateway",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp apigateway router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPAPIGatewayContractServer(t *testing.T) *httptest.Server {
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

func assertGCPAPIGatewaySuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "apigateway",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp apigateway router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
