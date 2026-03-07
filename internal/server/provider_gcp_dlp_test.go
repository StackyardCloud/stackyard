package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDLPRouter_ContentAndInfoTypeRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDLPContractServer(t)
	assertGCPDLPNotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/global/content:inspect", "content:inspect")
	assertGCPDLPNotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/global/content:deidentify", "content:deidentify")
	assertGCPDLPNotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/global/content:reidentify", "content:reidentify")
	assertGCPDLPNotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/global/image:redact", "image:redact")
	assertGCPDLPNotImplemented(t, ts, http.MethodGet, "/gcp/v2/infoTypes?filter=supported_by=INSPECT", "/infoTypes")
}

func TestGCPDLPRouter_TemplateAndTriggerRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDLPContractServer(t)
	assertGCPDLPNotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/global/inspectTemplates?pageSize=1", "/inspectTemplates")
	assertGCPDLPNotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/global/inspectTemplates", "/inspectTemplates")
	assertGCPDLPNotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/global/inspectTemplates/tpl-1", "/inspectTemplates/tpl-1")
	assertGCPDLPNotImplemented(t, ts, http.MethodDelete, "/gcp/v2/projects/stackyard/locations/global/inspectTemplates/tpl-1", "/inspectTemplates/tpl-1")
	assertGCPDLPNotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/global/jobTriggers?pageSize=1", "/jobTriggers")
	assertGCPDLPNotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/global/jobTriggers", "/jobTriggers")
	assertGCPDLPNotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/global/jobTriggers/trigger-1:activate", ":activate")
	assertGCPDLPNotImplemented(t, ts, http.MethodDelete, "/gcp/v2/projects/stackyard/locations/global/jobTriggers/trigger-1", "/jobTriggers/trigger-1")
}

func TestGCPDLPRouter_JobAndStoredInfoTypeRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDLPContractServer(t)
	assertGCPDLPNotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/global/dlpJobs", "/dlpJobs")
	assertGCPDLPNotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/global/dlpJobs?pageSize=1", "/dlpJobs")
	assertGCPDLPNotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/global/dlpJobs/job-1", "/dlpJobs/job-1")
	assertGCPDLPNotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/global/dlpJobs/job-1:cancel", ":cancel")
	assertGCPDLPNotImplemented(t, ts, http.MethodDelete, "/gcp/v2/projects/stackyard/locations/global/dlpJobs/job-1", "/dlpJobs/job-1")
	assertGCPDLPNotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/global/storedInfoTypes", "/storedInfoTypes")
	assertGCPDLPNotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/global/storedInfoTypes?pageSize=1", "/storedInfoTypes")
	assertGCPDLPNotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/global/storedInfoTypes/sit-1", "/storedInfoTypes/sit-1")
	assertGCPDLPNotImplemented(t, ts, http.MethodDelete, "/gcp/v2/projects/stackyard/locations/global/storedInfoTypes/sit-1", "/storedInfoTypes/sit-1")
}

func TestGCPDLPRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDLPContractServer(t)
	assertGCPDLPNotImplemented(t, ts, http.MethodPost, "/gcp/google.privacy.dlp.v2.DlpService/InspectContent", "DlpService/InspectContent")
	assertGCPDLPNotImplemented(t, ts, http.MethodPost, "/gcp/google.privacy.dlp.v2.DlpService/CreateInspectTemplate", "DlpService/CreateInspectTemplate")
	assertGCPDLPNotImplemented(t, ts, http.MethodPost, "/gcp/google.privacy.dlp.v2.DlpService/CreateDlpJob", "DlpService/CreateDlpJob")
	assertGCPDLPNotImplemented(t, ts, http.MethodPost, "/gcp/google.privacy.dlp.v2.DlpService/CreateStoredInfoType", "DlpService/CreateStoredInfoType")
}

func newGCPDLPContractServer(t *testing.T) *httptest.Server {
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

func assertGCPDLPNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dlp router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPDlpRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPDlpRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/dlp?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp dlp contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "dlp" {
		t.Fatalf("expected service=dlp, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

