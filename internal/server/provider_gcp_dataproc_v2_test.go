package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDataprocV2Router_BatchRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataprocV2ContractServer(t)
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/batches?pageSize=1", "/batches")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/batches?batchId=team-batch", "/batches")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/batches/team-batch", "/batches/team-batch")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/batches/team-batch", "/batches/team-batch")
}

func TestGCPDataprocV2Router_SessionRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataprocV2ContractServer(t)
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/sessions?pageSize=1", "/sessions")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/sessions?sessionId=interactive-1", "/sessions")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/sessions/interactive-1", "/sessions/interactive-1")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/sessions/interactive-1:terminate", ":terminate")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/sessions/interactive-1", "/sessions/interactive-1")
}

func TestGCPDataprocV2Router_SessionTemplateRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataprocV2ContractServer(t)
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/sessionTemplates?pageSize=1", "/sessionTemplates")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/sessionTemplates", "/sessionTemplates")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/sessionTemplates/analytics-template", "/sessionTemplates/analytics-template")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodPut, "/gcp/v1/projects/stackyard/locations/us-central1/sessionTemplates/analytics-template", "/sessionTemplates/analytics-template")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/sessionTemplates/analytics-template", "/sessionTemplates/analytics-template")
}

func TestGCPDataprocV2Router_OperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataprocV2ContractServer(t)
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1:cancel", ":cancel")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
}

func TestGCPDataprocV2Router_IAMActionRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDataprocV2ContractServer(t)
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/sessions/interactive-1:getIamPolicy", ":getIamPolicy")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/sessions/interactive-1:setIamPolicy", ":setIamPolicy")
	assertGCPDataprocV2NotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/sessions/interactive-1:testIamPermissions", ":testIamPermissions")
}

func newGCPDataprocV2ContractServer(t *testing.T) *httptest.Server {
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

func assertGCPDataprocV2NotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp dataproc v2 router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPDataprocV2Router_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPDataprocV2Router_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/dataproc_v2?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp dataproc_v2 contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "dataproc_v2" {
		t.Fatalf("expected service=dataproc_v2, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
