package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPCloudBuildRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudBuildContractServer(t)

	assertGCPCloudBuildSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/connections?pageSize=1", nil, "connections")
	assertGCPCloudBuildSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection", nil, "connections/team-connection")
	assertGCPCloudBuildSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/connections?connectionId=team-connection", []byte(`{"name":"projects/stackyard/locations/us-central1/connections/team-connection"}`), "createConnection")
	assertGCPCloudBuildSuccess(t, ts, http.MethodPatch, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection", []byte(`{"name":"projects/stackyard/locations/us-central1/connections/team-connection"}`), "updateConnection")
	assertGCPCloudBuildSuccess(t, ts, http.MethodDelete, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection", nil, "deleteConnection")
	assertGCPCloudBuildSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection/repositories?pageSize=1", nil, "repositories")
	assertGCPCloudBuildSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection/repositories/orders", nil, "repositories/orders")
	assertGCPCloudBuildSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection/repositories?repositoryId=orders", []byte(`{"name":"projects/stackyard/locations/us-central1/connections/team-connection/repositories/orders"}`), "createRepository")
	assertGCPCloudBuildSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection/repositories:batchCreate", []byte(`{"requests":[{"repository":{"name":"projects/stackyard/locations/us-central1/connections/team-connection/repositories/orders-batch"},"repositoryId":"orders-batch"}]}`), "batchCreateRepositories")
	assertGCPCloudBuildSuccess(t, ts, http.MethodDelete, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection/repositories/orders", nil, "deleteRepository")
	assertGCPCloudBuildSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection/repositories/orders:accessReadToken", nil, "token")
	assertGCPCloudBuildSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection/repositories/orders:accessReadWriteToken", nil, "token")
	assertGCPCloudBuildSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection:fetchLinkableRepositories", []byte(`{"pageSize":1}`), "repositories")
	assertGCPCloudBuildSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection/repositories/orders:fetchGitRefs", []byte(`{"refType":"BRANCH"}`), "refNames")
	assertGCPCloudBuildSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection:getIamPolicy", []byte(`{}`), "bindings")
	assertGCPCloudBuildSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection:setIamPolicy", []byte(`{"policy":{}}`), "etag")
	assertGCPCloudBuildSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection:testIamPermissions", []byte(`{"permissions":["cloudbuild.connections.get"]}`), "permissions")
	assertGCPCloudBuildSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/operations/op-1", nil, "operations/op-1")
	assertGCPCloudBuildSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/operations/op-1:cancel", []byte(`{}`), "{}")
}

func TestGCPCloudBuildRouter_ListConnectionsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudBuildContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/connections?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "cloudbuild",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloudbuild router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCloudBuildRouter_CreateConnectionRequiresConnectionID(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudBuildContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/connections", []byte(`{"name":"projects/stackyard/locations/us-central1/connections/team-connection"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "cloudbuild",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloudbuild router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCloudBuildRouter_BatchCreateRequiresRequests(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudBuildContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection/repositories:batchCreate", []byte(`{"requests":[]}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "cloudbuild",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloudbuild router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCloudBuildRouter_TestIAMPermissionsRequiresPermissions(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudBuildContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/connections/team-connection:testIamPermissions", []byte(`{"permissions":[]}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "cloudbuild",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloudbuild router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPCloudBuildContractServer(t *testing.T) *httptest.Server {
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

func assertGCPCloudBuildSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "cloudbuild",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp cloudbuild router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
