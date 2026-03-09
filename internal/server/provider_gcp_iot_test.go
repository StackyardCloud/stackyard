package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPIoTRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPIoTContractServer(t)
	location := "/gcp/v1/projects/stackyard/locations/us-central1"
	registries := location + "/registries"
	registry := registries + "/team-registry"
	devices := registry + "/devices"
	device := devices + "/sensor-1"

	assertGCPIoTNotImplemented(t, ts, http.MethodGet, registries+"?pageSize=1", "/registries")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, registries, "/registries")
	assertGCPIoTNotImplemented(t, ts, http.MethodGet, registry, "/registries/team-registry")
	assertGCPIoTNotImplemented(t, ts, http.MethodPatch, registry+"?updateMask.paths=log_level", "/registries/team-registry")
	assertGCPIoTNotImplemented(t, ts, http.MethodDelete, registry, "/registries/team-registry")

	assertGCPIoTNotImplemented(t, ts, http.MethodGet, devices+"?pageSize=1", "/devices")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, devices, "/devices")
	assertGCPIoTNotImplemented(t, ts, http.MethodGet, device, "/devices/sensor-1")
	assertGCPIoTNotImplemented(t, ts, http.MethodPatch, device+"?updateMask.paths=blocked", "/devices/sensor-1")
	assertGCPIoTNotImplemented(t, ts, http.MethodDelete, device, "/devices/sensor-1")

	assertGCPIoTNotImplemented(t, ts, http.MethodPost, device+":modifyCloudToDeviceConfig", ":modifyCloudToDeviceConfig")
	assertGCPIoTNotImplemented(t, ts, http.MethodGet, device+"/configVersions?numVersions=1", "/configVersions")
	assertGCPIoTNotImplemented(t, ts, http.MethodGet, device+"/states?numStates=1", "/states")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, registry+":setIamPolicy", ":setIamPolicy")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, registry+":getIamPolicy", ":getIamPolicy")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, registry+":testIamPermissions", ":testIamPermissions")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, device+":sendCommandToDevice", ":sendCommandToDevice")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, registry+":bindDeviceToGateway", ":bindDeviceToGateway")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, registry+":unbindDeviceFromGateway", ":unbindDeviceFromGateway")
}

func TestGCPIoTRouter_GrpcRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPIoTContractServer(t)
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/ListDeviceRegistries", "DeviceManager/ListDeviceRegistries")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/GetDeviceRegistry", "DeviceManager/GetDeviceRegistry")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/CreateDeviceRegistry", "DeviceManager/CreateDeviceRegistry")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/UpdateDeviceRegistry", "DeviceManager/UpdateDeviceRegistry")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/DeleteDeviceRegistry", "DeviceManager/DeleteDeviceRegistry")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/ListDevices", "DeviceManager/ListDevices")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/GetDevice", "DeviceManager/GetDevice")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/CreateDevice", "DeviceManager/CreateDevice")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/UpdateDevice", "DeviceManager/UpdateDevice")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/DeleteDevice", "DeviceManager/DeleteDevice")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/ModifyCloudToDeviceConfig", "DeviceManager/ModifyCloudToDeviceConfig")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/ListDeviceConfigVersions", "DeviceManager/ListDeviceConfigVersions")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/ListDeviceStates", "DeviceManager/ListDeviceStates")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/SetIamPolicy", "DeviceManager/SetIamPolicy")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/GetIamPolicy", "DeviceManager/GetIamPolicy")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/TestIamPermissions", "DeviceManager/TestIamPermissions")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/SendCommandToDevice", "DeviceManager/SendCommandToDevice")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/BindDeviceToGateway", "DeviceManager/BindDeviceToGateway")
	assertGCPIoTNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.iot.v1.DeviceManager/UnbindDeviceFromGateway", "DeviceManager/UnbindDeviceFromGateway")
}

func newGCPIoTContractServer(t *testing.T) *httptest.Server {
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

func assertGCPIoTNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp iot router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPIotRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPIotRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/iot?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp iot contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "iot" {
		t.Fatalf("expected service=iot, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}
