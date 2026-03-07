package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDataprocRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataprocContractServer(t)
	region := "/gcp/v1/projects/stackyard/regions/us-central1"

	assertGCPDataprocSuccess(t, ts, http.MethodGet, region+"/clusters?pageSize=1", nil, "clusters")
	assertGCPDataprocSuccess(t, ts, http.MethodPost, region+"/clusters", []byte(`{"cluster":{"clusterName":"team-cluster"}}`), "operations")
	assertGCPDataprocSuccess(t, ts, http.MethodGet, region+"/clusters/team-cluster", nil, "clusterName")
	assertGCPDataprocSuccess(t, ts, http.MethodPatch, region+"/clusters/team-cluster?updateMask=labels", []byte(`{"cluster":{"labels":{"env":"local"}}}`), "operations")
	assertGCPDataprocSuccess(t, ts, http.MethodPost, region+"/clusters/team-cluster:start", []byte(`{}`), "operations")
	assertGCPDataprocSuccess(t, ts, http.MethodPost, region+"/clusters/team-cluster:stop", []byte(`{}`), "operations")
	assertGCPDataprocSuccess(t, ts, http.MethodPost, region+"/clusters/team-cluster:diagnose", []byte(`{}`), "operations")
	assertGCPDataprocSuccess(t, ts, http.MethodDelete, region+"/clusters/team-cluster", nil, "operations")

	assertGCPDataprocSuccess(t, ts, http.MethodPost, region+"/jobs:submit", []byte(`{"job":{"reference":{"jobId":"team-job"}}}`), "team-job")
	assertGCPDataprocSuccess(t, ts, http.MethodPost, region+"/jobs:submitAsOperation", []byte(`{"job":{"reference":{"jobId":"team-job-op"}}}`), "operations")
	assertGCPDataprocSuccess(t, ts, http.MethodGet, region+"/jobs?pageSize=1", nil, "jobs")
	assertGCPDataprocSuccess(t, ts, http.MethodGet, region+"/jobs/team-job", nil, "team-job")
	assertGCPDataprocSuccess(t, ts, http.MethodPatch, region+"/jobs/team-job?updateMask=labels", []byte(`{"job":{"labels":{"env":"local"}}}`), "team-job")
	assertGCPDataprocSuccess(t, ts, http.MethodPost, region+"/jobs/team-job:cancel", []byte(`{}`), "CANCEL_PENDING")
	assertGCPDataprocSuccess(t, ts, http.MethodDelete, region+"/jobs/team-job", nil, "{}")

	assertGCPDataprocSuccess(t, ts, http.MethodGet, region+"/workflowTemplates?pageSize=1", nil, "templates")
	assertGCPDataprocSuccess(t, ts, http.MethodPost, region+"/workflowTemplates", []byte(`{"template":{"id":"team-template"}}`), "team-template")
	assertGCPDataprocSuccess(t, ts, http.MethodGet, region+"/workflowTemplates/team-template", nil, "team-template")
	assertGCPDataprocSuccess(t, ts, http.MethodPut, region+"/workflowTemplates/team-template", []byte(`{"template":{"id":"team-template"}}`), "team-template")
	assertGCPDataprocSuccess(t, ts, http.MethodPost, region+"/workflowTemplates/team-template:instantiate", []byte(`{}`), "operations")
	assertGCPDataprocSuccess(t, ts, http.MethodPost, region+"/workflowTemplates:instantiateInline", []byte(`{"template":{"id":"inline-template"}}`), "operations")
	assertGCPDataprocSuccess(t, ts, http.MethodDelete, region+"/workflowTemplates/team-template", nil, "{}")

	assertGCPDataprocSuccess(t, ts, http.MethodGet, region+"/autoscalingPolicies?pageSize=1", nil, "policies")
	assertGCPDataprocSuccess(t, ts, http.MethodPost, region+"/autoscalingPolicies", []byte(`{"policy":{"id":"team-policy"}}`), "team-policy")
	assertGCPDataprocSuccess(t, ts, http.MethodGet, region+"/autoscalingPolicies/team-policy", nil, "team-policy")
	assertGCPDataprocSuccess(t, ts, http.MethodPut, region+"/autoscalingPolicies/team-policy", []byte(`{"policy":{"id":"team-policy"}}`), "team-policy")
	assertGCPDataprocSuccess(t, ts, http.MethodDelete, region+"/autoscalingPolicies/team-policy", nil, "{}")
}

func TestGCPDataprocRouter_ListClustersInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPDataprocContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/regions/us-central1/clusters?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp dataproc router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPDataprocRouter_CreateClusterRequiresBody(t *testing.T) {
	t.Parallel()

	ts := newGCPDataprocContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/regions/us-central1/clusters", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp dataproc router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPDataprocContractServer(t *testing.T) *httptest.Server {
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

func assertGCPDataprocSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "dataproc",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp dataproc router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
