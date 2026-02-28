package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func globalAcceleratorRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	if strings.TrimSpace(payload) == "" {
		payload = `{}`
	}
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "GlobalAccelerator_V20180706." + action,
		},
		"globalaccelerator",
	)
}

func TestGlobalAcceleratorStage0CatalogCoverage(t *testing.T) {
	if len(globalAcceleratorOperations) != 56 {
		t.Fatalf("expected 56 Global Accelerator operations from docs, got %d", len(globalAcceleratorOperations))
	}
	if len(globalAcceleratorOperationByName) != len(globalAcceleratorOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateAccelerator",
		"DescribeAccelerator",
		"ListAccelerators",
		"CreateListener",
		"CreateEndpointGroup",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := globalAcceleratorOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(globalAcceleratorDataTypes) != 29 {
		t.Fatalf("expected 29 Global Accelerator data types from docs, got %d", len(globalAcceleratorDataTypes))
	}
	if len(globalAcceleratorDataTypeByName) != len(globalAcceleratorDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Accelerator",
		"Listener",
		"EndpointGroup",
		"CustomRoutingAccelerator",
		"ByoipCidr",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := globalAcceleratorDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestGlobalAcceleratorStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := globalAcceleratorRequest(t, ts, "UnknownGlobalAcceleratorAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestGlobalAcceleratorKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := globalAcceleratorRequest(t, ts, "ListAccelerators", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Accelerators") {
		t.Fatalf("expected ListAccelerators response body to include Accelerators, got %q", body)
	}
}

func TestGlobalAcceleratorStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range globalAcceleratorOperations {
		resp := globalAcceleratorRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
