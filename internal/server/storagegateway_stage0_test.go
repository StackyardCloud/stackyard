package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func storageGatewayRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	body := []byte(payload)
	headers := map[string]string{
		"Content-Type": "application/x-amz-json-1.1",
		"X-Amz-Target": "StorageGateway_20130630." + action,
	}
	return signedRequestWithService(t, http.MethodPost, ts.URL+"/", body, headers, "storagegateway")
}

func TestStorageGatewayStage0CatalogCoverage(t *testing.T) {
	if len(storageGatewayOperations) != 96 {
		t.Fatalf("expected 96 Storage Gateway operations from docs, got %d", len(storageGatewayOperations))
	}
	if len(storageGatewayOperationByName) != len(storageGatewayOperations) {
		t.Fatalf("expected unique Storage Gateway operation names")
	}

	requiredActions := []string{
		"ActivateGateway",
		"CreateNFSFileShare",
		"CreateSMBFileShare",
		"ListGateways",
		"DescribeGatewayInformation",
		"UpdateGatewayInformation",
	}
	for _, action := range requiredActions {
		if _, ok := storageGatewayOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(storageGatewayDataTypes) != 35 {
		t.Fatalf("expected 35 Storage Gateway data types from docs, got %d", len(storageGatewayDataTypes))
	}
	if len(storageGatewayDataTypeByName) != len(storageGatewayDataTypes) {
		t.Fatalf("expected unique Storage Gateway data type names")
	}

	requiredTypes := []string{
		"GatewayInfo",
		"FileShareInfo",
		"VolumeInfo",
		"TapeInfo",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := storageGatewayDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestStorageGatewayStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := storageGatewayRequest(t, ts, "UnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestStorageGatewayStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := storageGatewayRequest(t, ts, "ListGateways", `{"Limit":10}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Gateways") {
		t.Fatalf("expected ListGateways response body to include Gateways, got %q", body)
	}
}

func TestStorageGatewayStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range storageGatewayOperations {
		resp := storageGatewayRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
