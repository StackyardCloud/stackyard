package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPAutoMLRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPAutoMLContractServer(t)

	assertGCPAutoMLSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/datasets?pageSize=1", nil, "dataset")
	assertGCPAutoMLSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/datasets/team-dataset", nil, "datasets/team-dataset")
	assertGCPAutoMLSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/models?pageSize=1", nil, "model")
	assertGCPAutoMLSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/models/team-model", nil, "models/team-model")
	assertGCPAutoMLSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/models/team-model/modelEvaluations?filter=annotation_spec_id:*&pageSize=1", nil, "modelEvaluation")
	assertGCPAutoMLSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/models/team-model/modelEvaluations/eval-1", nil, "modelEvaluations/eval-1")
	assertGCPAutoMLSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/models/team-model:predict", []byte(`{"payload":{"textSnippet":{"content":"hello","mimeType":"text/plain"}}}`), "payload")
	assertGCPAutoMLSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/models/team-model:batchPredict", []byte(`{"inputConfig":{"gcsSource":{"inputUris":["gs://in/predict.csv"]}},"outputConfig":{"gcsDestination":{"outputUriPrefix":"gs://out/pred"}}}`), "operations/automl.batchPredict")
}

func TestGCPAutoMLRouter_ListDatasetsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPAutoMLContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/datasets?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp automl router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPAutoMLRouter_ListModelsPageTokenOutOfRange(t *testing.T) {
	t.Parallel()

	ts := newGCPAutoMLContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/models?pageToken=99", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp automl router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPAutoMLRouter_PredictInvalidJSON(t *testing.T) {
	t.Parallel()

	ts := newGCPAutoMLContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/models/team-model:predict", []byte(`{"payload"`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp automl router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPAutoMLRouter_BatchPredictRequiresOutputConfig(t *testing.T) {
	t.Parallel()

	ts := newGCPAutoMLContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/models/team-model:batchPredict", []byte(`{"inputConfig":{"gcsSource":{"inputUris":["gs://in/predict.csv"]}}}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp automl router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPAutoMLContractServer(t *testing.T) *httptest.Server {
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

func assertGCPAutoMLSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp automl router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
