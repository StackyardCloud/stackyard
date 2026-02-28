package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func apiGatewayV2Request(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		method,
		ts.URL+path,
		[]byte(payload),
		map[string]string{"Content-Type": "application/json"},
		"apigateway",
	)
}

func TestAPIGatewayV2Stage0CatalogCoverage(t *testing.T) {
	if len(apiGatewayV2Operations) != 94 {
		t.Fatalf("expected 94 API Gateway v2 operations from docs, got %d", len(apiGatewayV2Operations))
	}
	if len(apiGatewayV2OperationByName) != len(apiGatewayV2Operations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{"CreateApi", "GetApi", "GetApis", "DeleteApi", "TagResource", "GetTags"}
	for _, action := range requiredActions {
		if _, ok := apiGatewayV2OperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(apiGatewayV2DataTypes) != 45 {
		t.Fatalf("expected 45 API Gateway v2 data types from docs, got %d", len(apiGatewayV2DataTypes))
	}
	if len(apiGatewayV2DataTypeByName) != len(apiGatewayV2DataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{"Api", "Route", "Stage", "Tags", "VPCLink"}
	for _, typeName := range requiredTypes {
		if _, ok := apiGatewayV2DataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestAPIGatewayV2Stage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := apiGatewayV2Request(t, ts, http.MethodPost, "/apigatewayv2/not-a-real-action", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "BadRequestException") {
		t.Fatalf("expected BadRequestException response body, got %q", body)
	}
}

func TestAPIGatewayV2KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := apiGatewayV2Request(t, ts, http.MethodPost, "/apigatewayv2/get-apis", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Items") {
		t.Fatalf("expected GetApis response body to include Items, got %q", body)
	}
}

func TestAPIGatewayV2Stage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range apiGatewayV2Operations {
		resp := apiGatewayV2Request(t, ts, op.Method, op.URI, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
