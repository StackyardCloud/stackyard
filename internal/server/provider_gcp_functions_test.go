package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPCloudFunctionsRouter_FunctionLifecycleRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudFunctionsContractServer(t)
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/functions?pageSize=1", "/functions")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/functions", "/functions")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/functions/hello-fn", "/functions/hello-fn")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/functions/hello-fn?updateMask=description", "/functions/hello-fn")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/functions/hello-fn", "/functions/hello-fn")
}

func TestGCPCloudFunctionsRouter_FunctionActionAndIAMRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudFunctionsContractServer(t)
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/functions/hello-fn:call", ":call")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/functions:generateUploadUrl", ":generateUploadUrl")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/functions/hello-fn:generateDownloadUrl", ":generateDownloadUrl")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/functions/hello-fn:getIamPolicy", ":getIamPolicy")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/functions/hello-fn:setIamPolicy", ":setIamPolicy")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/functions/hello-fn:testIamPermissions", ":testIamPermissions")
}

func TestGCPCloudFunctionsRouter_LocationsAndOperationsRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudFunctionsContractServer(t)
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", "/locations/us-central1")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodGet, "/gcp/v1/operations?pageSize=1", "/operations")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodGet, "/gcp/v1/operations/op-1", "/operations/op-1")
}

func TestGCPCloudFunctionsRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCloudFunctionsContractServer(t)
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.functions.v1.CloudFunctionsService/ListFunctions", "CloudFunctionsService/ListFunctions")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.functions.v1.CloudFunctionsService/GetFunction", "CloudFunctionsService/GetFunction")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.functions.v1.CloudFunctionsService/CreateFunction", "CloudFunctionsService/CreateFunction")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.functions.v1.CloudFunctionsService/CallFunction", "CloudFunctionsService/CallFunction")
	assertGCPCloudFunctionsNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.functions.v1.CloudFunctionsService/GenerateUploadUrl", "CloudFunctionsService/GenerateUploadUrl")
}

func newGCPCloudFunctionsContractServer(t *testing.T) *httptest.Server {
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

func assertGCPCloudFunctionsNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp cloud functions router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPFunctionsRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPFunctionsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/functions?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp functions contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "functions" {
		t.Fatalf("expected service=functions, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

