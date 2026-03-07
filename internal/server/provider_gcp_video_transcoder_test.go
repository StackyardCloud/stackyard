package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPVideoTranscoderRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoTranscoderContractServer(t)

	base := "/gcp/v1/projects/stackyard/locations/us-central1"
	jobName := base + "/jobs/job-1"
	jobTemplateName := base + "/jobTemplates/template-1"

	assertGCPVideoTranscoderSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, `locations`)
	assertGCPVideoTranscoderSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, `us-central1`)

	assertGCPVideoTranscoderSuccess(t, ts, http.MethodPost, base+"/jobs", []byte(`{"job":{"name":"projects/stackyard/locations/us-central1/jobs/job-1","inputUri":"gs://stackyard-inputs/job-1.mp4","outputUri":"gs://stackyard-outputs/job-1/","templateId":"preset/web-hd"}}`), `jobs/job-1`)
	assertGCPVideoTranscoderSuccess(t, ts, http.MethodGet, base+"/jobs?pageSize=1", nil, `jobs`)
	assertGCPVideoTranscoderSuccess(t, ts, http.MethodGet, jobName, nil, `jobs/job-1`)
	assertGCPVideoTranscoderSuccess(t, ts, http.MethodDelete, jobName, nil, `{}`)

	assertGCPVideoTranscoderSuccess(t, ts, http.MethodPost, base+"/jobTemplates?jobTemplateId=template-1", []byte(`{"jobTemplate":{"name":"projects/stackyard/locations/us-central1/jobTemplates/template-1","config":{"output":{"uri":"gs://stackyard-outputs/templates/template-1/"}}}}`), `jobTemplates/template-1`)
	assertGCPVideoTranscoderSuccess(t, ts, http.MethodGet, base+"/jobTemplates?pageSize=1", nil, `jobTemplates`)
	assertGCPVideoTranscoderSuccess(t, ts, http.MethodGet, jobTemplateName, nil, `jobTemplates/template-1`)
	assertGCPVideoTranscoderSuccess(t, ts, http.MethodDelete, jobTemplateName, nil, `{}`)

	assertGCPVideoTranscoderSuccess(t, ts, http.MethodPost, gcpVideoTranscoderGRPCPathPrefix+"CreateJob", []byte(`{"parent":"projects/stackyard/locations/us-central1","job":{"name":"projects/stackyard/locations/us-central1/jobs/job-2","inputUri":"gs://stackyard-inputs/job-2.mp4","outputUri":"gs://stackyard-outputs/job-2/","templateId":"preset/web-hd"}}`), `jobs/job-2`)
}

func TestGCPVideoTranscoderRouter_InvalidJSONReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoTranscoderContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/jobs", []byte(`{"job"`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-transcoder",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video transcoder invalid json, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoTranscoderRouter_CreateJobRequiresOutputURI(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoTranscoderContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/jobs", []byte(`{"job":{"name":"projects/stackyard/locations/us-central1/jobs/job-1","inputUri":"gs://stackyard-inputs/job-1.mp4","templateId":"preset/web-hd"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-transcoder",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video transcoder create job missing output, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoTranscoderRouter_GetJobMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoTranscoderContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/jobs/missing-job", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-transcoder",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video transcoder get missing job, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoTranscoderRouter_CreateJobTemplateRequiresID(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoTranscoderContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/jobTemplates", []byte(`{"jobTemplate":{"config":{"output":{"uri":"gs://stackyard-outputs/templates/template-1/"}}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-transcoder",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video transcoder create template missing id, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoTranscoderRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoTranscoderContractServer(t)

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/jobs?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-transcoder",
	})
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video transcoder list jobs, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	jobs, ok := listBody["jobs"].([]any)
	if !ok || len(jobs) == 0 {
		t.Fatalf("expected jobs array, got %#v", listBody["jobs"])
	}
	firstJob, _ := jobs[0].(map[string]any)
	if _, ok := firstJob["name"].(string); !ok {
		t.Fatalf("expected job name string, got %#v", firstJob["name"])
	}

	createTemplateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/jobTemplates?jobTemplateId=template-1", []byte(`{"jobTemplate":{"name":"projects/stackyard/locations/us-central1/jobTemplates/template-1","config":{"output":{"uri":"gs://stackyard-outputs/templates/template-1/"}}}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "video-transcoder",
	})
	if createTemplateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video transcoder create template, got %d body=%s", createTemplateResp.StatusCode, string(providerContractBody(t, createTemplateResp)))
	}
	createTemplateBody := providerContractJSONMap(t, createTemplateResp)
	if _, ok := createTemplateBody["name"].(string); !ok {
		t.Fatalf("expected template name string, got %#v", createTemplateBody["name"])
	}
	if _, ok := createTemplateBody["config"].(map[string]any); !ok {
		t.Fatalf("expected config object, got %#v", createTemplateBody["config"])
	}
}

func TestGCPVideoTranscoderRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoTranscoderContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/video_transcoder?stackyard_contract_probe=1&typedSuccess=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-transcoder",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video transcoder contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "video_transcoder" {
		t.Fatalf("expected service=video_transcoder, got %#v", body["service"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func TestGCPVideoTranscoderRouter_ContractProbeInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoTranscoderContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/video_transcoder?stackyard_contract_probe=1&pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "video-transcoder",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video transcoder contract probe invalid pageSize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func assertGCPVideoTranscoderSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, contains string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "video-transcoder",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video transcoder %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if contains != "" {
		body := string(providerContractBody(t, resp))
		if !strings.Contains(body, contains) {
			t.Fatalf("expected body to contain %q, got %s", contains, body)
		}
	}
}

func newGCPVideoTranscoderContractServer(t *testing.T) *httptest.Server {
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
