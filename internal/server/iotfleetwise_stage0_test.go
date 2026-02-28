package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIoTFleetWiseStage0CatalogCoverage(t *testing.T) {
	if len(iotFleetWiseOperations) != 57 {
		t.Fatalf("expected 57 IoT FleetWise actions from docs, got %d", len(iotFleetWiseOperations))
	}
	if len(iotFleetWiseOperationByName) != len(iotFleetWiseOperations) {
		t.Fatalf("expected unique IoT FleetWise action names")
	}

	requiredActions := []string{
		"RegisterAccount",
		"CreateSignalCatalog",
		"CreateModelManifest",
		"CreateDecoderManifest",
		"CreateFleet",
		"CreateVehicle",
		"CreateCampaign",
		"CreateStateTemplate",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := iotFleetWiseOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(iotFleetWiseDataTypes) != 72 {
		t.Fatalf("expected 72 IoT FleetWise data types from docs, got %d", len(iotFleetWiseDataTypes))
	}
	if len(iotFleetWiseDataTypeByName) != len(iotFleetWiseDataTypes) {
		t.Fatalf("expected unique IoT FleetWise data type names")
	}

	requiredTypes := []string{
		"CampaignSummary",
		"DecoderManifestSummary",
		"FleetSummary",
		"ModelManifestSummary",
		"SignalCatalogSummary",
		"StateTemplateSummary",
		"VehicleStatus",
		"ValidationExceptionField",
	}
	for _, typeName := range requiredTypes {
		if _, ok := iotFleetWiseDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func iotFleetWiseRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "IoTAutobahnControlPlane." + action,
		},
		"iotfleetwise",
	)
}

func TestIoTFleetWiseStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotFleetWiseRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestIoTFleetWiseStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotFleetWiseRequest(t, ts, "ListFleets", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "fleetSummaries") {
		t.Fatalf("expected ListFleets response to include fleetSummaries, got %q", body)
	}
}

func TestIoTFleetWiseStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range iotFleetWiseOperations {
		resp := iotFleetWiseRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
