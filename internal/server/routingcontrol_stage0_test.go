package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func routingControlRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "ToggleCustomerAPI." + action,
		},
		"route53-recovery-cluster",
	)
}

func TestRoutingControlStage0CatalogCoverage(t *testing.T) {
	if len(routingControlOperations) != 4 {
		t.Fatalf("expected 4 Routing Control operations from docs, got %d", len(routingControlOperations))
	}
	if len(routingControlOperationByName) != len(routingControlOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"GetRoutingControlState",
		"ListRoutingControls",
		"UpdateRoutingControlState",
		"UpdateRoutingControlStates",
	}
	for _, action := range requiredActions {
		if _, ok := routingControlOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(routingControlDataTypes) != 3 {
		t.Fatalf("expected 3 Routing Control data types from docs, got %d", len(routingControlDataTypes))
	}
	if len(routingControlDataTypeByName) != len(routingControlDataTypes) {
		t.Fatalf("expected unique data type names")
	}
}

func TestRoutingControlStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := routingControlRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestRoutingControlKnownRouteReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := routingControlRequest(t, ts, "ListRoutingControls", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "RoutingControls") {
		t.Fatalf("expected ListRoutingControls response body to include RoutingControls, got %q", body)
	}
}

func TestRoutingControlStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range routingControlOperations {
		resp := routingControlRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
