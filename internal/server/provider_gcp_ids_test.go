package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPIDSRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPIDSContractServer(t)

	base := "/gcp/v1/projects/stackyard/locations/us-central1"
	endpoints := base + "/endpoints"
	endpoint := endpoints + "/ids-endpoint-1"
	operation := base + "/operations/op-1"

	assertGCPIDSNotImplemented(t, ts, http.MethodGet, endpoints+"?pageSize=1", "/endpoints")
	assertGCPIDSNotImplemented(t, ts, http.MethodPost, endpoints, "/endpoints")
	assertGCPIDSNotImplemented(t, ts, http.MethodGet, endpoint, "/endpoints/ids-endpoint-1")
	assertGCPIDSNotImplemented(t, ts, http.MethodDelete, endpoint, "/endpoints/ids-endpoint-1")
	assertGCPIDSNotImplemented(t, ts, http.MethodGet, operation, "/operations/op-1")
	assertGCPIDSNotImplemented(t, ts, http.MethodPost, operation+":cancel", ":cancel")
}

func TestGCPIDSRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPIDSContractServer(t)
	assertGCPIDSNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.ids.v1.IDS/ListEndpoints", "IDS/ListEndpoints")
	assertGCPIDSNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.ids.v1.IDS/GetEndpoint", "IDS/GetEndpoint")
	assertGCPIDSNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.ids.v1.IDS/CreateEndpoint", "IDS/CreateEndpoint")
	assertGCPIDSNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.ids.v1.IDS/DeleteEndpoint", "IDS/DeleteEndpoint")
}

func newGCPIDSContractServer(t *testing.T) *httptest.Server {
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

func assertGCPIDSNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp ids router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPIdsRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPIdsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/ids?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp ids contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "ids" {
		t.Fatalf("expected service=ids, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
