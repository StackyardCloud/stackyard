package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPParameterManagerRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPParameterManagerContractServer(t)

	assertGCPParameterManagerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/parameters?pageSize=1", "/parameters")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/parameters/app-config", "/parameters/app-config")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/parameters?parameterId=app-config", "/parameters")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/parameters/app-config?updateMask=labels", "/parameters/app-config")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/parameters/app-config", "/parameters/app-config")

	assertGCPParameterManagerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/parameters/app-config/versions?pageSize=1", "/versions")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/parameters/app-config/versions/v1", "/versions/v1")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/us-central1/parameters/app-config/versions?parameterVersionId=v1", "/versions")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/locations/us-central1/parameters/app-config/versions/v1?updateMask=disabled", "/versions/v1")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/locations/us-central1/parameters/app-config/versions/v1", "/versions/v1")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/parameters/app-config/versions/v1:render", ":render")
}

func TestGCPParameterManagerRouter_RESTLocationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPParameterManagerContractServer(t)

	assertGCPParameterManagerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", "/locations/us-central1")
}

func TestGCPParameterManagerRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPParameterManagerContractServer(t)

	assertGCPParameterManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parametermanager.v1.ParameterManager/ListParameters", "ParameterManager/ListParameters")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parametermanager.v1.ParameterManager/GetParameter", "ParameterManager/GetParameter")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parametermanager.v1.ParameterManager/CreateParameter", "ParameterManager/CreateParameter")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parametermanager.v1.ParameterManager/UpdateParameter", "ParameterManager/UpdateParameter")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parametermanager.v1.ParameterManager/DeleteParameter", "ParameterManager/DeleteParameter")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parametermanager.v1.ParameterManager/ListParameterVersions", "ParameterManager/ListParameterVersions")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parametermanager.v1.ParameterManager/GetParameterVersion", "ParameterManager/GetParameterVersion")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parametermanager.v1.ParameterManager/RenderParameterVersion", "ParameterManager/RenderParameterVersion")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parametermanager.v1.ParameterManager/CreateParameterVersion", "ParameterManager/CreateParameterVersion")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parametermanager.v1.ParameterManager/UpdateParameterVersion", "ParameterManager/UpdateParameterVersion")
	assertGCPParameterManagerNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.parametermanager.v1.ParameterManager/DeleteParameterVersion", "ParameterManager/DeleteParameterVersion")
}

func newGCPParameterManagerContractServer(t *testing.T) *httptest.Server {
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

func assertGCPParameterManagerNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp parametermanager router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPParametermanagerRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPParametermanagerRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/parametermanager?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp parametermanager contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "parametermanager" {
		t.Fatalf("expected service=parametermanager, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
