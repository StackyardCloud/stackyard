package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPArtifactRegistryRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPArtifactRegistryContractServer(t)

	assertGCPArtifactRegistrySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories?pageSize=1", nil, "repositories")
	assertGCPArtifactRegistrySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo", nil, "repositories/team-repo")
	assertGCPArtifactRegistrySuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/repositories?repositoryId=team-repo", []byte(`{"repository":{"format":"DOCKER"}}`), "operations/artifactregistry.createRepository")
	assertGCPArtifactRegistrySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo/packages?pageSize=1", nil, "packages")
	assertGCPArtifactRegistrySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo/packages/orders", nil, "packages/orders")
	assertGCPArtifactRegistrySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo/packages/orders/versions?pageSize=1", nil, "versions")
	assertGCPArtifactRegistrySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo/packages/orders/versions/1.0.0", nil, "versions/1.0.0")
	assertGCPArtifactRegistrySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo/packages/orders/tags?pageSize=1", nil, "tags")
	assertGCPArtifactRegistrySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo/packages/orders/tags/latest", nil, "tags/latest")
	assertGCPArtifactRegistrySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo/dockerImages?pageSize=1", nil, "dockerImages")
	assertGCPArtifactRegistrySuccess(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo", nil, "operations/artifactregistry.deleteRepository")
}

func TestGCPArtifactRegistryRouter_ListRepositoriesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPArtifactRegistryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "artifactregistry",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp artifactregistry router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPArtifactRegistryRouter_CreateRepositoryInvalidJSON(t *testing.T) {
	t.Parallel()

	ts := newGCPArtifactRegistryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/repositories?repositoryId=team-repo", []byte(`{"repository"`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "artifactregistry",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp artifactregistry router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPArtifactRegistryRouter_CreateRepositoryRequiresRepositoryID(t *testing.T) {
	t.Parallel()

	ts := newGCPArtifactRegistryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/repositories", []byte(`{"repository":{"format":"DOCKER"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "artifactregistry",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp artifactregistry router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPArtifactRegistryRouter_ListTagsPageTokenOutOfRange(t *testing.T) {
	t.Parallel()

	ts := newGCPArtifactRegistryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/repositories/team-repo/packages/orders/tags?pageToken=99", nil, map[string]string{
		"X-Stackyard-GCP-Service": "artifactregistry",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp artifactregistry router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPArtifactRegistryContractServer(t *testing.T) *httptest.Server {
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

func assertGCPArtifactRegistrySuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "artifactregistry",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp artifactregistry router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
