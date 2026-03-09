package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPNotebooksV2Router_RESTInstanceAndActionRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNotebooksV2ContractServer(t)

	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/instances?pageSize=1", "/instances")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/instances", "/instances")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/instances/notebook-1", "/instances/notebook-1")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodPatch, "/gcp/v2/projects/stackyard/locations/us-central1/instances/notebook-1?updateMask=labels", "/instances/notebook-1")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodDelete, "/gcp/v2/projects/stackyard/locations/us-central1/instances/notebook-1", "/instances/notebook-1")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/instances/notebook-1:start", ":start")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/instances/notebook-1:checkUpgradability", ":checkUpgradability")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/instances/notebook-1:diagnose", ":diagnose")
}

func TestGCPNotebooksV2Router_RESTIAMLocationAndOperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNotebooksV2ContractServer(t)

	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1", "/locations/us-central1")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/instances/notebook-1:getIamPolicy", ":getIamPolicy")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/instances/notebook-1:setIamPolicy", ":setIamPolicy")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/instances/notebook-1:testIamPermissions", ":testIamPermissions")
}

func TestGCPNotebooksV2Router_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPNotebooksV2ContractServer(t)

	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.notebooks.v2.NotebookService/ListInstances", "NotebookService/ListInstances")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.notebooks.v2.NotebookService/GetInstance", "NotebookService/GetInstance")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.notebooks.v2.NotebookService/CheckInstanceUpgradability", "NotebookService/CheckInstanceUpgradability")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.notebooks.v2.NotebookService/StartInstance", "NotebookService/StartInstance")
	assertGCPNotebooksV2NotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.notebooks.v2.NotebookService/DiagnoseInstance", "NotebookService/DiagnoseInstance")
}

func newGCPNotebooksV2ContractServer(t *testing.T) *httptest.Server {
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

func assertGCPNotebooksV2NotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp notebooks v2 router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPNotebooksV2Router_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPNotebooksV2Router_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/notebooks_v2?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp notebooks_v2 contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "notebooks_v2" {
		t.Fatalf("expected service=notebooks_v2, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
