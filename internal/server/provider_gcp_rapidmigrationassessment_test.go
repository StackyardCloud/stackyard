package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPRapidMigrationAssessmentRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPRapidMigrationAssessmentContractServer(t)

	baseLocation := "/gcp/v1/projects/stackyard/locations/us-central1"
	collectorBase := baseLocation + "/collectors/collector-1"
	annotationBase := baseLocation + "/annotations/annotation-1"

	assertGCPRapidMigrationAssessmentSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", nil, "locations")
	assertGCPRapidMigrationAssessmentSuccess(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", nil, "us-central1")

	assertGCPRapidMigrationAssessmentSuccess(t, ts, http.MethodGet, baseLocation+"/collectors?pageSize=1", nil, "collectors")
	assertGCPRapidMigrationAssessmentSuccess(t, ts, http.MethodGet, collectorBase, nil, "collectors/collector-1")
	assertGCPRapidMigrationAssessmentSuccess(t, ts, http.MethodPost, baseLocation+"/collectors?collectorId=collector-1", []byte(`{"collector":{"displayName":"Collector One"}}`), "operations/createCollector.collector-1")
	assertGCPRapidMigrationAssessmentSuccess(t, ts, http.MethodPatch, collectorBase+"?updateMask=display_name", []byte(`{"collector":{"name":"projects/stackyard/locations/us-central1/collectors/collector-1","displayName":"Collector One Updated"}}`), "operations/updateCollector.collector-1")
	assertGCPRapidMigrationAssessmentSuccess(t, ts, http.MethodDelete, collectorBase, nil, "operations/deleteCollector.collector-1")

	assertGCPRapidMigrationAssessmentSuccess(t, ts, http.MethodPost, collectorBase+":pause", []byte(`{}`), "pauseCollector.collector-1")
	assertGCPRapidMigrationAssessmentSuccess(t, ts, http.MethodPost, baseLocation+"/collectors/collector-1-paused:resume", []byte(`{}`), "resumeCollector.collector-1-paused")
	assertGCPRapidMigrationAssessmentSuccess(t, ts, http.MethodPost, collectorBase+":register", []byte(`{}`), "registerCollector.collector-1")

	assertGCPRapidMigrationAssessmentSuccess(t, ts, http.MethodPost, baseLocation+"/annotations", []byte(`{"annotation":{"type":"Annotation_TYPE_LEGACY_EXPORT_CONSENT"}}`), "createAnnotation.annotation-1")
	assertGCPRapidMigrationAssessmentSuccess(t, ts, http.MethodGet, annotationBase, nil, "annotations/annotation-1")

	assertGCPRapidMigrationAssessmentSuccess(t, ts, http.MethodGet, baseLocation+"/operations?pageSize=1", nil, "operations")
	assertGCPRapidMigrationAssessmentSuccess(t, ts, http.MethodGet, baseLocation+"/operations/createCollector.collector-1", nil, "createCollector.collector-1")
}

func TestGCPRapidMigrationAssessmentRouter_ListCollectorsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPRapidMigrationAssessmentContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/collectors?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "rapidmigrationassessment",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp rapidmigrationassessment router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRapidMigrationAssessmentRouter_CreateCollectorRequiresCollectorID(t *testing.T) {
	t.Parallel()

	ts := newGCPRapidMigrationAssessmentContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/collectors", []byte(`{"collector":{"displayName":"Collector One"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "rapidmigrationassessment",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp rapidmigrationassessment router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRapidMigrationAssessmentRouter_UpdateCollectorNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPRapidMigrationAssessmentContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/collectors/collector-1?updateMask=display_name", []byte(`{"collector":{"name":"projects/stackyard/locations/us-central1/collectors/collector-2","displayName":"Collector One Updated"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "rapidmigrationassessment",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp rapidmigrationassessment router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRapidMigrationAssessmentRouter_CreateAnnotationRejectsInvalidType(t *testing.T) {
	t.Parallel()

	ts := newGCPRapidMigrationAssessmentContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/annotations", []byte(`{"annotation":{"type":"Annotation_TYPE_UNSPECIFIED"}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "rapidmigrationassessment",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp rapidmigrationassessment router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRapidMigrationAssessmentRouter_PauseCollectorAlreadyPaused(t *testing.T) {
	t.Parallel()

	ts := newGCPRapidMigrationAssessmentContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/collectors/collector-1-paused:pause", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "rapidmigrationassessment",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp rapidmigrationassessment router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPRapidMigrationAssessmentContractServer(t *testing.T) *httptest.Server {
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

func assertGCPRapidMigrationAssessmentSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "rapidmigrationassessment",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp rapidmigrationassessment router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
