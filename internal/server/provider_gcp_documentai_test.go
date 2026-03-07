package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDocumentAIRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDocumentAIContractServer(t)
	base := "/gcp/v1/projects/stackyard/locations/us"

	assertGCPDocumentAISuccess(t, ts, http.MethodPost, base+":fetchProcessorTypes", []byte(`{"parent":"projects/stackyard/locations/us"}`), "processorTypes")
	assertGCPDocumentAISuccess(t, ts, http.MethodGet, base+"/processorTypes?pageSize=1", nil, "processorTypes")
	assertGCPDocumentAISuccess(t, ts, http.MethodGet, base+"/processorTypes/FORM_PARSER_PROCESSOR", nil, "FORM_PARSER_PROCESSOR")

	assertGCPDocumentAISuccess(t, ts, http.MethodGet, base+"/processors?pageSize=1", nil, "processors")
	assertGCPDocumentAISuccess(t, ts, http.MethodPost, base+"/processors?processorId=proc-1", []byte(`{"processor":{"displayName":"stackyard-processor","type":"FORM_PARSER_PROCESSOR"}}`), "proc-1")
	assertGCPDocumentAISuccess(t, ts, http.MethodGet, base+"/processors/proc-1", nil, "proc-1")
	assertGCPDocumentAISuccess(t, ts, http.MethodPost, base+"/processors/proc-1:process", []byte(`{"name":"projects/stackyard/locations/us/processors/proc-1"}`), "document")
	assertGCPDocumentAISuccess(t, ts, http.MethodPost, base+"/processors/proc-1:batchProcess", []byte(`{"name":"projects/stackyard/locations/us/processors/proc-1"}`), "operations")

	assertGCPDocumentAISuccess(t, ts, http.MethodGet, base+"/processors/proc-1/processorVersions?pageSize=1", nil, "processorVersions")
	assertGCPDocumentAISuccess(t, ts, http.MethodGet, base+"/processors/proc-1/processorVersions/ver-1", nil, "ver-1")
	assertGCPDocumentAISuccess(t, ts, http.MethodPost, base+"/processors/proc-1/processorVersions:train", []byte(`{}`), "operations")
	assertGCPDocumentAISuccess(t, ts, http.MethodPost, base+"/processors/proc-1/processorVersions/ver-1:deploy", []byte(`{}`), "operations")
	assertGCPDocumentAISuccess(t, ts, http.MethodPost, base+"/processors/proc-1/processorVersions/ver-1:undeploy", []byte(`{}`), "operations")
	assertGCPDocumentAISuccess(t, ts, http.MethodDelete, base+"/processors/proc-1/processorVersions/ver-1", nil, "{}")

	assertGCPDocumentAISuccess(t, ts, http.MethodPost, base+"/processors/proc-1:enable", []byte(`{}`), "operations")
	assertGCPDocumentAISuccess(t, ts, http.MethodPost, base+"/processors/proc-1:disable", []byte(`{}`), "operations")
	assertGCPDocumentAISuccess(t, ts, http.MethodPost, base+"/processors/proc-1:setDefaultProcessorVersion", []byte(`{"defaultProcessorVersion":"projects/stackyard/locations/us/processors/proc-1/processorVersions/ver-1"}`), "operations")
	assertGCPDocumentAISuccess(t, ts, http.MethodPost, base+"/processors/proc-1/humanReviewConfig:reviewDocument", []byte(`{"humanReviewConfig":"projects/stackyard/locations/us/processors/proc-1/humanReviewConfig"}`), "operations")
	assertGCPDocumentAISuccess(t, ts, http.MethodPost, base+"/processors/proc-1/processorVersions/ver-1:evaluateProcessorVersion", []byte(`{}`), "operations")

	assertGCPDocumentAISuccess(t, ts, http.MethodGet, base+"/processors/proc-1/processorVersions/ver-1/evaluations?pageSize=1", nil, "evaluations")
	assertGCPDocumentAISuccess(t, ts, http.MethodGet, base+"/processors/proc-1/processorVersions/ver-1/evaluations/eval-1", nil, "eval-1")

	assertGCPDocumentAISuccess(t, ts, http.MethodGet, base, nil, "locationId")
	assertGCPDocumentAISuccess(t, ts, http.MethodGet, base+"/locations?pageSize=1", nil, "locations")
	assertGCPDocumentAISuccess(t, ts, http.MethodGet, base+"/operations?pageSize=1", nil, "operations")
	assertGCPDocumentAISuccess(t, ts, http.MethodGet, base+"/operations/op-1", nil, "op-1")
	assertGCPDocumentAISuccess(t, ts, http.MethodPost, base+"/operations/op-1:cancel", []byte(`{}`), "{}")
	assertGCPDocumentAISuccess(t, ts, http.MethodDelete, base+"/processors/proc-1", nil, "{}")
}

func TestGCPDocumentAIRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPDocumentAIContractServer(t)
	assertGCPDocumentAINotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.documentai.v1.DocumentProcessorService/ProcessDocument", "DocumentProcessorService/ProcessDocument")
}

func TestGCPDocumentAIRouter_ListProcessorsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPDocumentAIContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us/processors?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp documentai router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func TestGCPDocumentAIRouter_CreateProcessorRequiresBody(t *testing.T) {
	t.Parallel()

	ts := newGCPDocumentAIContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us/processors", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp documentai router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func newGCPDocumentAIContractServer(t *testing.T) *httptest.Server {
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

func assertGCPDocumentAINotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp documentai router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPDocumentAISuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp documentai router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, string(providerContractBody(t, resp)))
	}
}
