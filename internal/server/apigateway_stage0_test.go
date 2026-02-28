package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func apiGatewayRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		method,
		ts.URL+path,
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/json",
		},
		"apigateway",
	)
}

func TestAPIGatewayStage0CatalogCoverage(t *testing.T) {
	if len(apiGatewayOperations) != 124 {
		t.Fatalf("expected 124 API Gateway operations from docs, got %d", len(apiGatewayOperations))
	}
	if len(apiGatewayOperationByName) != len(apiGatewayOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{"CreateRestApi", "GetRestApis", "GetRestApi", "DeleteRestApi", "TagResource", "UntagResource"}
	for _, action := range requiredActions {
		if _, ok := apiGatewayOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(apiGatewayDataTypes) != 39 {
		t.Fatalf("expected 39 API Gateway data types from docs, got %d", len(apiGatewayDataTypes))
	}
	if len(apiGatewayDataTypeByName) != len(apiGatewayDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{"RestApi", "Stage", "Method", "Integration", "UsagePlan"}
	for _, typeName := range requiredTypes {
		if _, ok := apiGatewayDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestAPIGatewayStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := apiGatewayRequest(t, ts, http.MethodPost, "/apigateway/not-a-real-action", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "BadRequestException") {
		t.Fatalf("expected BadRequestException response body, got %q", body)
	}
}

func TestAPIGatewayKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := apiGatewayRequest(t, ts, http.MethodPost, "/apigateway/get-rest-apis", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "items") {
		t.Fatalf("expected GetRestApis response body to include items, got %q", body)
	}
}

func TestAPIGatewayStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range apiGatewayOperations {
		resp := apiGatewayRequest(t, ts, op.Method, op.URI, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
