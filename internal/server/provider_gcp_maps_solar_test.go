package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPMapsSolarRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsSolarContractServer(t)

	assertGCPMapsSolarSuccess(t, ts, http.MethodGet, "/gcp/v1/buildingInsights:findClosest?location.latitude=37.7937&location.longitude=-122.3965", nil, "buildingInsights/stackyard-building-1")
	assertGCPMapsSolarSuccess(t, ts, http.MethodGet, "/gcp/v1/dataLayers:get?location.latitude=37.7937&location.longitude=-122.3965&radiusMeters=100", nil, "dsmUrl")
	assertGCPMapsSolarSuccess(t, ts, http.MethodGet, "/gcp/v1/geoTiff:get?id=asset-id", nil, `"contentType":"image/tiff"`)
}

func TestGCPMapsSolarRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsSolarContractServer(t)

	assertGCPMapsSolarNotImplemented(t, ts, http.MethodPost, "/gcp/google.maps.solar.v1.Solar/FindClosestBuildingInsights", "Solar/FindClosestBuildingInsights")
	assertGCPMapsSolarNotImplemented(t, ts, http.MethodPost, "/gcp/google.maps.solar.v1.Solar/GetDataLayers", "Solar/GetDataLayers")
	assertGCPMapsSolarNotImplemented(t, ts, http.MethodPost, "/gcp/google.maps.solar.v1.Solar/GetGeoTiff", "Solar/GetGeoTiff")
}

func TestGCPMapsSolarRouter_FindClosestRequiresLatitude(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsSolarContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/buildingInsights:findClosest?location.longitude=-122.3965", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps solar router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPMapsSolarRouter_GetDataLayersInvalidRadius(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsSolarContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/dataLayers:get?location.latitude=37.7937&location.longitude=-122.3965&radiusMeters=not-a-number", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps solar router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPMapsSolarRouter_GetGeoTiffRequiresID(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsSolarContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/geoTiff:get", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps solar router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPMapsSolarContractServer(t *testing.T) *httptest.Server {
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

func assertGCPMapsSolarNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp maps solar router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPMapsSolarSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp maps solar router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
