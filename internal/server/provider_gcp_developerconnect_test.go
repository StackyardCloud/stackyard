package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDeveloperConnectRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDeveloperConnectContractServer(t)

	assertGCPDeveloperConnectSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connections?pageSize=1", nil, "connections")
	assertGCPDeveloperConnectSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connections/team-connection", nil, "team-connection")
	assertGCPDeveloperConnectSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connections?connectionId=team-connection", []byte(`{"connection":{}}`), "operations")
	assertGCPDeveloperConnectSuccess(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/connections/team-connection", nil, "operations")
	assertGCPDeveloperConnectSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connections/team-connection/gitRepositoryLinks?pageSize=1", nil, "gitRepositoryLinks")
	assertGCPDeveloperConnectSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connections/team-connection/gitRepositoryLinks/orders", nil, "orders")
	assertGCPDeveloperConnectSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connections/team-connection/gitRepositoryLinks?gitRepositoryLinkId=orders", []byte(`{"gitRepositoryLink":{}}`), "operations")
	assertGCPDeveloperConnectSuccess(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/connections/team-connection/gitRepositoryLinks/orders", nil, "operations")
	assertGCPDeveloperConnectSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connections/team-connection/gitRepositoryLinks/orders:fetchReadToken", []byte(`{}`), "read-token")
	assertGCPDeveloperConnectSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connections/team-connection/gitRepositoryLinks/orders:fetchReadWriteToken", []byte(`{}`), "read-write-token")
	assertGCPDeveloperConnectSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connections/team-connection:fetchLinkableGitRepositories?pageSize=1", nil, "linkableGitRepositories")
	assertGCPDeveloperConnectSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connections/team-connection:fetchGitHubInstallations", nil, "installations")
	assertGCPDeveloperConnectSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connections/team-connection/gitRepositoryLinks/orders:fetchGitRefs?refType=BRANCH&pageSize=1", nil, "refNames")
	operationHeaders := map[string]string{"X-Stackyard-GCP-Service": "developerconnect"}
	assertGCPDeveloperConnectSuccessWithHeaders(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", nil, operationHeaders, "operations")
	assertGCPDeveloperConnectSuccessWithHeaders(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", nil, operationHeaders, "op-1")
	assertGCPDeveloperConnectSuccessWithHeaders(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", []byte(`{}`), operationHeaders, "{}")
	assertGCPDeveloperConnectSuccessWithHeaders(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", nil, operationHeaders, "{}")
}

func TestGCPDeveloperConnectRouter_ListConnectionsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPDeveloperConnectContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connections?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp developerconnect router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPDeveloperConnectRouter_CreateConnectionRequiresID(t *testing.T) {
	t.Parallel()

	ts := newGCPDeveloperConnectContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connections", []byte(`{"connection":{}}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp developerconnect router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPDeveloperConnectRouter_CreateGitRepositoryLinkRequiresID(t *testing.T) {
	t.Parallel()

	ts := newGCPDeveloperConnectContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/connections/team-connection/gitRepositoryLinks", []byte(`{"gitRepositoryLink":{}}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp developerconnect router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPDeveloperConnectRouter_FetchGitRefsRejectsInvalidRefType(t *testing.T) {
	t.Parallel()

	ts := newGCPDeveloperConnectContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/connections/team-connection/gitRepositoryLinks/orders:fetchGitRefs?refType=BAD", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp developerconnect router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPDeveloperConnectContractServer(t *testing.T) *httptest.Server {
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

func assertGCPDeveloperConnectSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	assertGCPDeveloperConnectSuccessWithHeaders(t, ts, method, path, payload, nil, expectBodyFragment)
}

func assertGCPDeveloperConnectSuccessWithHeaders(t *testing.T, ts *httptest.Server, method, path string, payload []byte, extraHeaders map[string]string, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	for k, v := range extraHeaders {
		headers[k] = v
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp developerconnect router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
