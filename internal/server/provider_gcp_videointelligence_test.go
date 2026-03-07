package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPVideoIntelligenceRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoIntelligenceContractServer(t)

	annotatePayload := []byte(`{
		"inputUri":"gs://stackyard-inputs/video-1.mp4",
		"features":["SHOT_CHANGE_DETECTION"],
		"locationId":"us-east1"
	}`)
	operationPath := "/gcp/v1/projects/stackyard/locations/us-east1/operations/annotateVideo.video-1"

	assertGCPVideoIntelligenceSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPVideoIntelligenceSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-east1", nil, "us-east1")

	assertGCPVideoIntelligenceSuccess(t, ts, http.MethodPost, "/gcp/v1/videos:annotate", annotatePayload, "annotateVideo.video-1")
	assertGCPVideoIntelligenceSuccess(t, ts, http.MethodPost, "/gcp/google.cloud.videointelligence.v1.VideoIntelligenceService/AnnotateVideo", annotatePayload, "annotateVideo.video-1")

	assertGCPVideoIntelligenceSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-east1/operations?pageSize=1", nil, "operations")
	assertGCPVideoIntelligenceSuccess(t, ts, http.MethodGet, operationPath, nil, "annotateVideo.video-1")
	assertGCPVideoIntelligenceSuccess(t, ts, http.MethodPost, operationPath+":cancel", []byte(`{}`), "{}")
	assertGCPVideoIntelligenceSuccess(t, ts, http.MethodDelete, operationPath, nil, "{}")
}

func TestGCPVideoIntelligenceRouter_InvalidJSONReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoIntelligenceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/videos:annotate", []byte(`{"inputUri"`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "videointelligence",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video intelligence invalid json, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoIntelligenceRouter_AnnotateRequiresFeatures(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoIntelligenceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/videos:annotate", []byte(`{"inputUri":"gs://stackyard-inputs/video.mp4"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "videointelligence",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video intelligence annotate missing features, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoIntelligenceRouter_AnnotateRejectsMutuallyExclusiveInputs(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoIntelligenceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/videos:annotate", []byte(`{
		"inputUri":"gs://stackyard-inputs/video.mp4",
		"inputContent":"c3RhY2t5YXJk",
		"features":["SHOT_CHANGE_DETECTION"]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "videointelligence",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video intelligence annotate mutually exclusive input, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoIntelligenceRouter_AnnotateRejectsInvalidLocation(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoIntelligenceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/videos:annotate", []byte(`{
		"inputUri":"gs://stackyard-inputs/video.mp4",
		"features":["SHOT_CHANGE_DETECTION"],
		"locationId":"us-central1"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "videointelligence",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video intelligence annotate invalid location, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoIntelligenceRouter_GetOperationMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoIntelligenceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-east1/operations/missing-operation", nil, map[string]string{
		"X-Stackyard-GCP-Service": "videointelligence",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video intelligence get missing operation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPVideoIntelligenceRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoIntelligenceContractServer(t)
	headers := map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "videointelligence",
	}

	annotateResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/videos:annotate", []byte(`{
		"inputUri":"gs://stackyard-inputs/video-1.mp4",
		"features":["SHOT_CHANGE_DETECTION"],
		"locationId":"us-east1"
	}`), headers)
	if annotateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video intelligence annotate, got %d body=%s", annotateResp.StatusCode, string(providerContractBody(t, annotateResp)))
	}
	annotateBody := providerContractJSONMap(t, annotateResp)
	if _, ok := annotateBody["name"].(string); !ok {
		t.Fatalf("expected operation name string, got %#v", annotateBody["name"])
	}
	if _, ok := annotateBody["done"].(bool); !ok {
		t.Fatalf("expected operation done bool, got %#v", annotateBody["done"])
	}
	metadata, ok := annotateBody["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata object, got %#v", annotateBody["metadata"])
	}
	if _, ok := metadata["@type"].(string); !ok {
		t.Fatalf("expected metadata @type string, got %#v", metadata["@type"])
	}
	progress, ok := metadata["annotationProgress"].([]any)
	if !ok || len(progress) == 0 {
		t.Fatalf("expected metadata annotationProgress array, got %#v", metadata["annotationProgress"])
	}
	response, ok := annotateBody["response"].(map[string]any)
	if !ok {
		t.Fatalf("expected response object, got %#v", annotateBody["response"])
	}
	if _, ok := response["@type"].(string); !ok {
		t.Fatalf("expected response @type string, got %#v", response["@type"])
	}
	annotationResults, ok := response["annotationResults"].([]any)
	if !ok || len(annotationResults) == 0 {
		t.Fatalf("expected annotationResults array, got %#v", response["annotationResults"])
	}

	locationResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-east1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "videointelligence",
	})
	if locationResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video intelligence get location, got %d body=%s", locationResp.StatusCode, string(providerContractBody(t, locationResp)))
	}
	locationBody := providerContractJSONMap(t, locationResp)
	if _, ok := locationBody["name"].(string); !ok {
		t.Fatalf("expected location name string, got %#v", locationBody["name"])
	}
	if _, ok := locationBody["metadata"].(map[string]any); !ok {
		t.Fatalf("expected location metadata object, got %#v", locationBody["metadata"])
	}
}

func TestGCPVideoIntelligenceRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoIntelligenceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-east1/videointelligence?stackyard_contract_probe=1&typedSuccess=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "videointelligence",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from video intelligence contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "videointelligence" {
		t.Fatalf("expected service=videointelligence, got %#v", body["service"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func TestGCPVideoIntelligenceRouter_ContractProbeInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPVideoIntelligenceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-east1/videointelligence?stackyard_contract_probe=1&pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "videointelligence",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from video intelligence contract probe invalid pageSize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPVideoIntelligenceContractServer(t *testing.T) *httptest.Server {
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

func assertGCPVideoIntelligenceSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "videointelligence",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp video intelligence router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
