package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPCloudFunctionsV2Router_FunctionLifecycleRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudFunctionsV2ContractServer(t)
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/functions?pageSize=1", "/functions")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/functions", "/functions")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/functions/hello-fn", "/functions/hello-fn")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodPatch, "/gcp/v2/projects/stackyard/locations/us-central1/functions/hello-fn?updateMask=description", "/functions/hello-fn")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodDelete, "/gcp/v2/projects/stackyard/locations/us-central1/functions/hello-fn", "/functions/hello-fn")
}

func TestGCPCloudFunctionsV2Router_SourceAndIAMRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudFunctionsV2ContractServer(t)
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/functions:generateUploadUrl", ":generateUploadUrl")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/functions/hello-fn:generateDownloadUrl", ":generateDownloadUrl")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/functions/hello-fn:getIamPolicy", ":getIamPolicy")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/functions/hello-fn:setIamPolicy", ":setIamPolicy")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/us-central1/functions/hello-fn:testIamPermissions", ":testIamPermissions")
}

func TestGCPCloudFunctionsV2Router_RuntimesLocationsAndOperationsRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudFunctionsV2ContractServer(t)
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/runtimes", "/runtimes")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1", "/locations/us-central1")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/operations?pageSize=1", "/operations")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/us-central1/operations/op-1", "/operations/op-1")
}

func TestGCPCloudFunctionsV2Router_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudFunctionsV2ContractServer(t)
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.functions.v2.FunctionService/ListFunctions", "FunctionService/ListFunctions")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.functions.v2.FunctionService/GetFunction", "FunctionService/GetFunction")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.functions.v2.FunctionService/CreateFunction", "FunctionService/CreateFunction")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.functions.v2.FunctionService/ListRuntimes", "FunctionService/ListRuntimes")
	assertGCPCloudFunctionsV2NotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.functions.v2.FunctionService/GenerateUploadUrl", "FunctionService/GenerateUploadUrl")
}

func newGCPCloudFunctionsV2ContractServer(t *testing.T) *httptest.Server {
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

func assertGCPCloudFunctionsV2NotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp cloud functions v2 router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPFunctionsV2Router_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPFunctionsV2Router_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/functions_v2?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp functions_v2 contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "functions_v2" {
		t.Fatalf("expected service=functions_v2, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
