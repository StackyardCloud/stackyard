package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPMapsFleetEngineDeliveryPath_BatchCreateMatches(t *testing.T) {
	t.Parallel()

	if !isGCPMapsFleetEngineDeliveryPath("/gcp/v1/providers/stackyard/tasks:batchCreate") {
		t.Fatalf("expected tasks:batchCreate path to match maps fleetengine delivery router")
	}
}

func TestGCPMapsFleetEngineDeliveryRouter_RESTDeliveryVehicleRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsFleetEngineDeliveryContractServer(t)

	assertGCPMapsFleetEngineDeliverySuccess(t, ts, http.MethodPost, "/gcp/v1/providers/stackyard/deliveryVehicles?deliveryVehicleId=dv-1", []byte(`{"type":1}`), "deliveryVehicles/dv-1")
	assertGCPMapsFleetEngineDeliverySuccess(t, ts, http.MethodGet, "/gcp/v1/providers/stackyard/deliveryVehicles?pageSize=1", nil, "deliveryVehicles")
	assertGCPMapsFleetEngineDeliverySuccess(t, ts, http.MethodGet, "/gcp/v1/providers/stackyard/deliveryVehicles/dv-1", nil, "deliveryVehicles/dv-1")
	assertGCPMapsFleetEngineDeliverySuccess(t, ts, http.MethodPatch, "/gcp/v1/providers/stackyard/deliveryVehicles/dv-1?updateMask=last_location", []byte(`{"name":"providers/stackyard/deliveryVehicles/dv-1","type":1}`), "deliveryVehicles/dv-1")
	assertGCPMapsFleetEngineDeliverySuccess(t, ts, http.MethodDelete, "/gcp/v1/providers/stackyard/deliveryVehicles/dv-1", nil, `"deleted":true`)
}

func TestGCPMapsFleetEngineDeliveryRouter_RESTTaskRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsFleetEngineDeliveryContractServer(t)

	assertGCPMapsFleetEngineDeliverySuccess(t, ts, http.MethodPost, "/gcp/v1/providers/stackyard/tasks?taskId=task-1", []byte(`{"trackingId":"tracking-1","type":1,"state":1}`), "tasks/task-1")
	assertGCPMapsFleetEngineDeliverySuccess(t, ts, http.MethodPost, "/gcp/v1/providers/stackyard/tasks:batchCreate", []byte(`{"requests":[{"taskId":"task-batch-1","task":{"trackingId":"tracking-batch-1"}}]}`), "task-batch-1")
	assertGCPMapsFleetEngineDeliverySuccess(t, ts, http.MethodGet, "/gcp/v1/providers/stackyard/tasks?pageSize=1", nil, "tasks")
	assertGCPMapsFleetEngineDeliverySuccess(t, ts, http.MethodGet, "/gcp/v1/providers/stackyard/tasks/task-1", nil, "tasks/task-1")
	assertGCPMapsFleetEngineDeliverySuccess(t, ts, http.MethodPatch, "/gcp/v1/providers/stackyard/tasks/task-1?updateMask=state", []byte(`{"name":"providers/stackyard/tasks/task-1","trackingId":"tracking-1","state":2}`), "tasks/task-1")
	assertGCPMapsFleetEngineDeliverySuccess(t, ts, http.MethodDelete, "/gcp/v1/providers/stackyard/tasks/task-1", nil, `"deleted":true`)
	assertGCPMapsFleetEngineDeliverySuccess(t, ts, http.MethodGet, "/gcp/v1/providers/stackyard/taskTrackingInfo/trk-1", nil, "taskTrackingInfo/trk-1")
}

func TestGCPMapsFleetEngineDeliveryRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsFleetEngineDeliveryContractServer(t)

	assertGCPMapsFleetEngineDeliveryNotImplemented(t, ts, http.MethodPost, "/gcp/maps.fleetengine.delivery.v1.DeliveryService/CreateDeliveryVehicle", "CreateDeliveryVehicle")
	assertGCPMapsFleetEngineDeliveryNotImplemented(t, ts, http.MethodPost, "/gcp/maps.fleetengine.delivery.v1.DeliveryService/GetDeliveryVehicle", "GetDeliveryVehicle")
	assertGCPMapsFleetEngineDeliveryNotImplemented(t, ts, http.MethodPost, "/gcp/maps.fleetengine.delivery.v1.DeliveryService/CreateTask", "CreateTask")
	assertGCPMapsFleetEngineDeliveryNotImplemented(t, ts, http.MethodPost, "/gcp/maps.fleetengine.delivery.v1.DeliveryService/GetTask", "GetTask")
	assertGCPMapsFleetEngineDeliveryNotImplemented(t, ts, http.MethodPost, "/gcp/maps.fleetengine.delivery.v1.DeliveryService/GetTaskTrackingInfo", "GetTaskTrackingInfo")
}

func TestGCPMapsFleetEngineDeliveryRouter_ListDeliveryVehiclesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsFleetEngineDeliveryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/providers/stackyard/deliveryVehicles?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps fleetengine delivery router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPMapsFleetEngineDeliveryRouter_CreateTaskRequiresTaskID(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsFleetEngineDeliveryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/providers/stackyard/tasks", []byte(`{"trackingId":"tracking-1"}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps fleetengine delivery router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPMapsFleetEngineDeliveryRouter_BatchCreateTasksRequiresRequests(t *testing.T) {
	t.Parallel()

	ts := newGCPMapsFleetEngineDeliveryContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/providers/stackyard/tasks:batchCreate", []byte(`{"requests":[]}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp maps fleetengine delivery router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPMapsFleetEngineDeliveryContractServer(t *testing.T) *httptest.Server {
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

func assertGCPMapsFleetEngineDeliveryNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp maps fleetengine delivery router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPMapsFleetEngineDeliverySuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp maps fleetengine delivery router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
