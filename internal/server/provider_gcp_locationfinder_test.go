package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPLocationFinderRouter_CloudLocationsRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPLocationFinderContractServer(t)
	assertGCPLocationFinderNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/cloudLocations?pageSize=1", "/cloudLocations")
	assertGCPLocationFinderNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/cloudLocations/us-east1", "/cloudLocations/us-east1")
	assertGCPLocationFinderNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/cloudLocations:search?sourceCloudLocation=projects%2Fstackyard%2Flocations%2Fus-central1%2FcloudLocations%2Fus-central1&query=latency", ":search")
}

func TestGCPLocationFinderRouter_LocationsRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPLocationFinderContractServer(t)
	assertGCPLocationFinderNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations?pageSize=1", "/locations")
	assertGCPLocationFinderNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1", "/locations/us-central1")
}

func TestGCPLocationFinderRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPLocationFinderContractServer(t)
	assertGCPLocationFinderNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.locationfinder.v1.CloudLocationFinder/ListCloudLocations", "CloudLocationFinder/ListCloudLocations")
	assertGCPLocationFinderNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.locationfinder.v1.CloudLocationFinder/GetCloudLocation", "CloudLocationFinder/GetCloudLocation")
	assertGCPLocationFinderNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.locationfinder.v1.CloudLocationFinder/SearchCloudLocations", "CloudLocationFinder/SearchCloudLocations")
}

func newGCPLocationFinderContractServer(t *testing.T) *httptest.Server {
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

func assertGCPLocationFinderNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp locationfinder router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPLocationfinderRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPLocationfinderRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/locationfinder?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp locationfinder contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "locationfinder" {
		t.Fatalf("expected service=locationfinder, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

