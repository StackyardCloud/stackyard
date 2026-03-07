package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPMapsFleetEngineRouter_RESTTripRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsFleetEngineContractServer(t)

	assertGCPMapsFleetEngineSuccess(t, ts, http.MethodPost, "/gcp/v1/providers/stackyard/trips?tripId=trip-1", []byte(`{"tripType":1}`), "trips/trip-1")
	assertGCPMapsFleetEngineSuccess(t, ts, http.MethodGet, "/gcp/v1/providers/stackyard/trips?pageSize=1", nil, "trips")
	assertGCPMapsFleetEngineSuccess(t, ts, http.MethodGet, "/gcp/v1/providers/stackyard/trips/trip-1", nil, "trips/trip-1")
	assertGCPMapsFleetEngineSuccess(t, ts, http.MethodPatch, "/gcp/v1/providers/stackyard/trips/trip-1?updateMask=vehicle_id", []byte(`{"name":"providers/stackyard/trips/trip-1","vehicleId":"vehicle-1"}`), "trips/trip-1")
	assertGCPMapsFleetEngineSuccess(t, ts, http.MethodDelete, "/gcp/v1/providers/stackyard/trips/trip-1", nil, `"deleted":true`)
	assertGCPMapsFleetEngineSuccess(t, ts, http.MethodPost, "/gcp/v1/providers/stackyard/trips:search", []byte(`{"vehicleId":"vehicle-1"}`), "trips")
	assertGCPMapsFleetEngineSuccess(t, ts, http.MethodPost, "/gcp/v1/providers/stackyard/trips/trip-1:reportBillable", []byte(`{"countryCode":"US"}`), `"reported":true`)
}

func TestGCPMapsFleetEngineRouter_RESTVehicleRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsFleetEngineContractServer(t)

	assertGCPMapsFleetEngineSuccess(t, ts, http.MethodPost, "/gcp/v1/providers/stackyard/vehicles?vehicleId=vehicle-1", []byte(`{"vehicleState":1}`), "vehicles/vehicle-1")
	assertGCPMapsFleetEngineSuccess(t, ts, http.MethodGet, "/gcp/v1/providers/stackyard/vehicles?pageSize=1", nil, "vehicles")
	assertGCPMapsFleetEngineSuccess(t, ts, http.MethodGet, "/gcp/v1/providers/stackyard/vehicles/vehicle-1", nil, "vehicles/vehicle-1")
	assertGCPMapsFleetEngineSuccess(t, ts, http.MethodPatch, "/gcp/v1/providers/stackyard/vehicles/vehicle-1?updateMask=vehicle_state", []byte(`{"name":"providers/stackyard/vehicles/vehicle-1","vehicleState":1}`), "vehicles/vehicle-1")
	assertGCPMapsFleetEngineSuccess(t, ts, http.MethodDelete, "/gcp/v1/providers/stackyard/vehicles/vehicle-1", nil, `"deleted":true`)
	assertGCPMapsFleetEngineSuccess(t, ts, http.MethodPost, "/gcp/v1/providers/stackyard/vehicles:search", []byte(`{"pickupPoint":{"point":{"latitude":37.77,"longitude":-122.41}}}`), "matches")
	assertGCPMapsFleetEngineSuccess(t, ts, http.MethodPost, "/gcp/v1/providers/stackyard/vehicles/vehicle-1:updateAttributes", []byte(`{"attributes":[{"key":"env","value":"local"}]}`), "attributes")
}

func TestGCPMapsFleetEngineRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsFleetEngineContractServer(t)

	assertGCPMapsFleetEngineNotImplemented(t, ts, http.MethodPost, "/gcp/maps.fleetengine.v1.TripService/CreateTrip", "TripService/CreateTrip")
	assertGCPMapsFleetEngineNotImplemented(t, ts, http.MethodPost, "/gcp/maps.fleetengine.v1.TripService/SearchTrips", "TripService/SearchTrips")
	assertGCPMapsFleetEngineNotImplemented(t, ts, http.MethodPost, "/gcp/maps.fleetengine.v1.VehicleService/CreateVehicle", "VehicleService/CreateVehicle")
	assertGCPMapsFleetEngineNotImplemented(t, ts, http.MethodPost, "/gcp/maps.fleetengine.v1.VehicleService/SearchVehicles", "VehicleService/SearchVehicles")
}

func TestGCPMapsFleetEngineRouter_ListVehiclesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsFleetEngineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/providers/stackyard/vehicles?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps fleetengine router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPMapsFleetEngineRouter_CreateTripInvalidJSON(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsFleetEngineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/providers/stackyard/trips?tripId=trip-1", []byte(`{"tripType"`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps fleetengine router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPMapsFleetEngineRouter_SearchVehiclesRequiresPickupPoint(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsFleetEngineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/providers/stackyard/vehicles:search", []byte(`{"tripId":"trip-1"}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps fleetengine router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPMapsFleetEngineContractServer(t *testing.T) *httptest.Server {
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

func assertGCPMapsFleetEngineNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp maps fleetengine router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPMapsFleetEngineSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp maps fleetengine router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
