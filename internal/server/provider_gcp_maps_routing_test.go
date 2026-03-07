package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPMapsRoutingRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsRoutingContractServer(t)

	assertGCPMapsRoutingSuccess(t, ts, http.MethodPost, "/gcp/directions/v2:computeRoutes", []byte(`{
		"origin":{"location":{"latLng":{"latitude":37.7937,"longitude":-122.3965}}},
		"destination":{"location":{"latLng":{"latitude":37.7793,"longitude":-122.4193}}},
		"travelMode":"DRIVE"
	}`), "routes")
	assertGCPMapsRoutingSuccess(t, ts, http.MethodPost, "/gcp/distanceMatrix/v2:computeRouteMatrix", []byte(`{
		"origins":[{"waypoint":{"location":{"latLng":{"latitude":37.7937,"longitude":-122.3965}}}}],
		"destinations":[{"waypoint":{"location":{"latLng":{"latitude":37.7793,"longitude":-122.4193}}}}],
		"travelMode":"DRIVE"
	}`), "distanceMeters")
}

func TestGCPMapsRoutingRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsRoutingContractServer(t)

	assertGCPMapsRoutingSuccess(t, ts, http.MethodPost, "/gcp/google.maps.routing.v2.Routes/ComputeRoutes", []byte(`{
		"origin":{"location":{"latLng":{"latitude":37.7937,"longitude":-122.3965}}},
		"destination":{"location":{"latLng":{"latitude":37.7793,"longitude":-122.4193}}},
		"travelMode":"DRIVE"
	}`), "routes")
	assertGCPMapsRoutingSuccess(t, ts, http.MethodPost, "/gcp/google.maps.routing.v2.Routes/ComputeRouteMatrix", []byte(`{
		"origins":[{"waypoint":{"location":{"latLng":{"latitude":37.7937,"longitude":-122.3965}}}}],
		"destinations":[{"waypoint":{"location":{"latLng":{"latitude":37.7793,"longitude":-122.4193}}}}],
		"travelMode":"DRIVE"
	}`), "distanceMeters")
}

func TestGCPMapsRoutingRouter_InvalidJSON(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsRoutingContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/directions/v2:computeRoutes", []byte(`{"origin"`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps routing router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPMapsRoutingRouter_ValidateRequiredFields(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsRoutingContractServer(t)

	missingDestination := providerContractRequest(t, ts, http.MethodPost, "/gcp/directions/v2:computeRoutes", []byte(`{
		"origin":{"location":{"latLng":{"latitude":37.7937,"longitude":-122.3965}}}
	}`), map[string]string{
		"Content-Type": "application/json",
	})
	if missingDestination.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing destination, got %d body=%s", missingDestination.StatusCode, string(providerContractBody(t, missingDestination)))
	}
	if body := string(providerContractBody(t, missingDestination)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}

	missingOrigins := providerContractRequest(t, ts, http.MethodPost, "/gcp/distanceMatrix/v2:computeRouteMatrix", []byte(`{
		"origins":[],
		"destinations":[{"waypoint":{"location":{"latLng":{"latitude":37.7793,"longitude":-122.4193}}}}]
	}`), map[string]string{
		"Content-Type": "application/json",
	})
	if missingOrigins.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty origins, got %d body=%s", missingOrigins.StatusCode, string(providerContractBody(t, missingOrigins)))
	}
	if body := string(providerContractBody(t, missingOrigins)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPMapsRoutingContractServer(t *testing.T) *httptest.Server {
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

func assertGCPMapsRoutingSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, payload, map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp maps routing router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	bodyBytes := providerContractBody(t, resp)
	body := string(bodyBytes)
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
	if strings.Contains(path, "computeRouteMatrix") {
		var streamPayload []map[string]any
		if err := json.Unmarshal(bodyBytes, &streamPayload); err != nil {
			t.Fatalf("decode matrix stream payload: %v", err)
		}
		if len(streamPayload) == 0 {
			t.Fatalf("expected at least one matrix element in stream payload")
		}
	}
}
