package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func iotWirelessRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		method,
		ts.URL+path,
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/json",
		},
		"iotwireless",
	)
}

func TestIoTWirelessStage0CatalogCoverage(t *testing.T) {
	if len(iotWirelessOperations) != 112 {
		t.Fatalf("expected 112 IoT Wireless operations from docs, got %d", len(iotWirelessOperations))
	}
	if len(iotWirelessOperationByName) != len(iotWirelessOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateWirelessDevice",
		"CreateWirelessGateway",
		"ListWirelessDevices",
		"GetWirelessDevice",
		"SendDataToWirelessDevice",
		"StartWirelessDeviceImportTask",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := iotWirelessOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(iotWirelessDataTypes) != 131 {
		t.Fatalf("expected 131 IoT Wireless data types from docs, got %d", len(iotWirelessDataTypes))
	}
	if len(iotWirelessDataTypeByName) != len(iotWirelessDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"WirelessDeviceStatistics",
		"WirelessGatewayStatistics",
		"Destinations",
		"DeviceProfile",
		"ServiceProfile",
		"FuotaTask",
	}
	for _, typeName := range requiredTypes {
		if _, ok := iotWirelessDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestIoTWirelessStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotWirelessRequest(t, ts, http.MethodPost, "/unknown-iotwireless-route", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestIoTWirelessKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotWirelessRequest(t, ts, http.MethodGet, "/wireless-devices?MaxResults=1", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "WirelessDeviceList") {
		t.Fatalf("expected ListWirelessDevices response body to include WirelessDeviceList, got %q", body)
	}
}
