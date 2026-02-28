package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func iotGreengrassRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		method,
		ts.URL+path,
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/json",
		},
		"greengrass",
	)
}

func TestIoTGreengrassStage0CatalogCoverage(t *testing.T) {
	if len(iotGreengrassOperations) != 29 {
		t.Fatalf("expected 29 IoT Greengrass operations from docs, got %d", len(iotGreengrassOperations))
	}
	if len(iotGreengrassOperationByName) != len(iotGreengrassOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateDeployment",
		"GetDeployment",
		"ListDeployments",
		"GetComponent",
		"ListCoreDevices",
		"UpdateConnectivityInfo",
	}
	for _, action := range requiredActions {
		if _, ok := iotGreengrassOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(iotGreengrassDataTypes) != 41 {
		t.Fatalf("expected 41 IoT Greengrass data types from docs, got %d", len(iotGreengrassDataTypes))
	}
	if len(iotGreengrassDataTypeByName) != len(iotGreengrassDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Component",
		"CoreDevice",
		"Deployment",
		"EffectiveDeployment",
		"InstalledComponent",
		"ConnectivityInfo",
	}
	for _, typeName := range requiredTypes {
		if _, ok := iotGreengrassDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestIoTGreengrassStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotGreengrassRequest(t, ts, http.MethodPost, "/greengrass/v2/unknown-route", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestIoTGreengrassKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotGreengrassRequest(t, ts, http.MethodGet, "/greengrass/v2/components?maxResults=1", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "components") {
		t.Fatalf("expected ListComponents response body to include components, got %q", body)
	}
}
