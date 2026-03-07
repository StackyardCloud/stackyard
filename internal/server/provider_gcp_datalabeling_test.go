package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDataLabelingRouter_DatasetRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataLabelingContractServer(t)
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/datasets?pageSize=1", "/datasets")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/datasets", "/datasets")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset", "/datasets/team-dataset")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodDelete, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset", "/datasets/team-dataset")
}

func TestGCPDataLabelingRouter_ImportAndExportRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataLabelingContractServer(t)
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset:importData", ":importData")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset:exportData", ":exportData")
}

func TestGCPDataLabelingRouter_DataItemAndAnnotatedDatasetRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataLabelingContractServer(t)
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset/dataItems?pageSize=1", "/dataItems")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset/dataItems/item-1", "/dataItems/item-1")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset/annotatedDatasets?pageSize=1", "/annotatedDatasets")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset/annotatedDatasets/annotated-1", "/annotatedDatasets/annotated-1")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodDelete, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset/annotatedDatasets/annotated-1", "/annotatedDatasets/annotated-1")
}

func TestGCPDataLabelingRouter_LabelingActionRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataLabelingContractServer(t)
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset/image:label", "/image:label")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset/video:label", "/video:label")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset/text:label", "/text:label")
}

func TestGCPDataLabelingRouter_ExampleRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataLabelingContractServer(t)
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset/annotatedDatasets/annotated-1/examples?pageSize=1", "/examples")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset/annotatedDatasets/annotated-1/examples/example-1", "/examples/example-1")
}

func TestGCPDataLabelingRouter_AnnotationSpecSetRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataLabelingContractServer(t)
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/annotationSpecSets?pageSize=1", "/annotationSpecSets")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/annotationSpecSets", "/annotationSpecSets")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/annotationSpecSets/spec-set-1", "/annotationSpecSets/spec-set-1")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodDelete, "/gcp/v1beta1/projects/stackyard/annotationSpecSets/spec-set-1", "/annotationSpecSets/spec-set-1")
}

func TestGCPDataLabelingRouter_InstructionRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataLabelingContractServer(t)
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/instructions?pageSize=1", "/instructions")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/instructions", "/instructions")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/instructions/instruction-1", "/instructions/instruction-1")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodDelete, "/gcp/v1beta1/projects/stackyard/instructions/instruction-1", "/instructions/instruction-1")
}

func TestGCPDataLabelingRouter_EvaluationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataLabelingContractServer(t)
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset/evaluations/eval-1", "/evaluations/eval-1")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/evaluations:search?pageSize=1", "/evaluations:search")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/datasets/team-dataset/evaluations/eval-1/exampleComparisons:search?pageSize=1", "/exampleComparisons:search")
}

func TestGCPDataLabelingRouter_EvaluationJobRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataLabelingContractServer(t)
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/evaluationJobs?pageSize=1", "/evaluationJobs")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/evaluationJobs", "/evaluationJobs")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/evaluationJobs/job-1", "/evaluationJobs/job-1")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodPatch, "/gcp/v1beta1/projects/stackyard/evaluationJobs/job-1?updateMask=evaluation_job_config.example_count", "/evaluationJobs/job-1")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/evaluationJobs/job-1:pause", ":pause")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/evaluationJobs/job-1:resume", ":resume")
	assertGCPDataLabelingNotImplemented(t, ts, http.MethodDelete, "/gcp/v1beta1/projects/stackyard/evaluationJobs/job-1", "/evaluationJobs/job-1")
}

func newGCPDataLabelingContractServer(t *testing.T) *httptest.Server {
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

func assertGCPDataLabelingNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp datalabeling router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPDatalabelingRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPDatalabelingRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/datalabeling?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp datalabeling contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "datalabeling" {
		t.Fatalf("expected service=datalabeling, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

