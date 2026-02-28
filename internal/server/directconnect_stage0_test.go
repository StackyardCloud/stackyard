package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func directConnectRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "OvertureService." + action,
		},
		"directconnect",
	)
}

func TestDirectConnectStage0CatalogCoverage(t *testing.T) {
	if len(directConnectOperations) != 63 {
		t.Fatalf("expected 63 Direct Connect operations from docs, got %d", len(directConnectOperations))
	}
	if len(directConnectOperationByName) != len(directConnectOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateConnection",
		"DescribeConnections",
		"CreateDirectConnectGateway",
		"DescribeDirectConnectGateways",
		"CreatePrivateVirtualInterface",
		"DescribeVirtualInterfaces",
	}
	for _, action := range requiredActions {
		if _, ok := directConnectOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(directConnectDataTypes) != 29 {
		t.Fatalf("expected 29 Direct Connect data types from docs, got %d", len(directConnectDataTypes))
	}
	if len(directConnectDataTypeByName) != len(directConnectDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Connection",
		"DirectConnectGateway",
		"VirtualInterface",
		"Lag",
		"Location",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := directConnectDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestDirectConnectStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := directConnectRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestDirectConnectKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := directConnectRequest(t, ts, "DescribeConnections", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "connections") {
		t.Fatalf("expected DescribeConnections response body to include connections, got %q", body)
	}
}

func TestDirectConnectAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range directConnectOperations {
		resp := directConnectRequest(t, ts, op.Name, `{}`)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
