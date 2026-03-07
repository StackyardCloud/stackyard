package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPCloudDeployRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudDeployContractServer(t)

	location := "/gcp/v1/projects/stackyard/locations/us-central1"
	pipeline := location + "/deliveryPipelines/team-pipeline"
	target := location + "/targets/team-target"
	release := pipeline + "/releases/team-release"
	rollout := release + "/rollouts/team-rollout"
	jobRun := rollout + "/jobRuns/job-a"

	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, location+"/deliveryPipelines?pageSize=1", nil, nil, "deliveryPipelines")
	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, pipeline, nil, nil, "team-pipeline")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPost, location+"/deliveryPipelines?deliveryPipelineId=team-pipeline", []byte(`{"deliveryPipeline":{"name":"projects/stackyard/locations/us-central1/deliveryPipelines/team-pipeline"}}`), nil, "operations")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPatch, pipeline+"?updateMask=description", []byte(`{"deliveryPipeline":{"name":"projects/stackyard/locations/us-central1/deliveryPipelines/team-pipeline"}}`), nil, "operations")
	assertGCPCloudDeploySuccess(t, ts, http.MethodDelete, pipeline, nil, nil, "operations")

	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, location+"/targets?pageSize=1", nil, nil, "targets")
	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, target, nil, nil, "team-target")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPost, location+"/targets?targetId=team-target", []byte(`{"target":{"name":"projects/stackyard/locations/us-central1/targets/team-target"}}`), nil, "operations")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPost, target+":rollbackTarget", []byte(`{}`), nil, "{}")
	assertGCPCloudDeploySuccess(t, ts, http.MethodDelete, target, nil, nil, "operations")

	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, pipeline+"/releases?pageSize=1", nil, nil, "releases")
	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, release, nil, nil, "team-release")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPost, pipeline+"/releases?releaseId=team-release", []byte(`{"release":{"name":"projects/stackyard/locations/us-central1/deliveryPipelines/team-pipeline/releases/team-release"}}`), nil, "operations")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPost, release+":abandon", []byte(`{}`), nil, "{}")

	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, release+"/rollouts?pageSize=1", nil, nil, "rollouts")
	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, rollout, nil, nil, "team-rollout")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPost, release+"/rollouts?rolloutId=team-rollout", []byte(`{"rollout":{"name":"projects/stackyard/locations/us-central1/deliveryPipelines/team-pipeline/releases/team-release/rollouts/team-rollout"}}`), nil, "operations")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPost, rollout+":approve", []byte(`{"approved":true}`), nil, "{}")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPost, rollout+":advance", []byte(`{"phaseId":"phase-1"}`), nil, "{}")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPost, rollout+":cancel", []byte(`{}`), nil, "{}")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPost, rollout+":ignoreJob", []byte(`{"phaseId":"phase-1","jobId":"job-a"}`), nil, "{}")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPost, rollout+":retryJob", []byte(`{"phaseId":"phase-1","jobId":"job-a"}`), nil, "{}")

	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, rollout+"/jobRuns?pageSize=1", nil, nil, "jobRuns")
	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, jobRun, nil, nil, "job-a")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPost, jobRun+":terminate", []byte(`{}`), nil, "{}")

	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, location+"/deployPolicies?pageSize=1", nil, nil, "deployPolicies")
	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, location+"/deployPolicies/policy-1", nil, nil, "policy-1")
	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, location+"/config", nil, nil, "defaultSkaffoldVersion")

	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, pipeline+":getIamPolicy", nil, nil, "bindings")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPost, pipeline+":setIamPolicy", []byte(`{"policy":{"etag":"AQ=="}}`), nil, "etag")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPost, pipeline+":testIamPermissions", []byte(`{"permissions":["clouddeploy.deliveryPipelines.get"]}`), nil, "permissions")
}

func TestGCPCloudDeployRouter_OperationRoutesRecognizedWithServiceHint(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudDeployContractServer(t)
	headers := map[string]string{"X-Stackyard-GCP-Service": "deploy"}

	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", nil, headers, "operations")
	assertGCPCloudDeploySuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", nil, headers, "op-1")
	assertGCPCloudDeploySuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", []byte(`{}`), headers, "{}")
	assertGCPCloudDeploySuccess(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", nil, headers, "{}")
}

func TestGCPCloudDeployRouter_ListDeliveryPipelinesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudDeployContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/deliveryPipelines?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloud deploy router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCloudDeployRouter_CreateDeliveryPipelineRequiresID(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudDeployContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/deliveryPipelines", []byte(`{"deliveryPipeline":{}}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp cloud deploy router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPCloudDeployContractServer(t *testing.T) *httptest.Server {
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

func assertGCPCloudDeploySuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, headers map[string]string, expectBodyFragment string) {
	t.Helper()

	reqHeaders := map[string]string{
		"X-Stackyard-GCP-Service": "deploy",
	}
	for k, v := range headers {
		reqHeaders[k] = v
	}
	if payload != nil {
		reqHeaders["Content-Type"] = "application/json"
	}

	resp := providerContractRequest(t, ts, method, path, payload, reqHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp cloud deploy router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
