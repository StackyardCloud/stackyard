package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPAPIHubRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPAPIHubContractServer(t)

	assertGCPAPIHubSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/attributes?pageSize=1", nil, "attributes")
	assertGCPAPIHubSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/deployments?pageSize=1", nil, "deployments")
	assertGCPAPIHubSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/externalApis?pageSize=1", nil, "externalApis")
	assertGCPAPIHubSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/apis/team-api/versions?pageSize=1", nil, "versions")
	assertGCPAPIHubSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/apis/team-api/versions/v1/definitions/openapi", nil, "definitions/openapi")
	assertGCPAPIHubSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/apis/team-api/versions/v1/operations/get-orders", nil, "operations/get-orders")
	assertGCPAPIHubSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:searchResources", []byte(`{"query":"orders","pageSize":1}`), "results")
}

func TestGCPAPIHubRouter_ListAttributesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPAPIHubContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/attributes?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp apihub router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPAPIHubRouter_SearchResourcesInvalidJSON(t *testing.T) {
	t.Parallel()

	ts := newGCPAPIHubContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:searchResources", []byte(`{"query"`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp apihub router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPAPIHubRouter_SearchResourcesRequiresQuery(t *testing.T) {
	t.Parallel()

	ts := newGCPAPIHubContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1:searchResources", []byte(`{"pageSize":1}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp apihub router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPAPIHubContractServer(t *testing.T) *httptest.Server {
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

func assertGCPAPIHubSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()
	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp apihub router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
