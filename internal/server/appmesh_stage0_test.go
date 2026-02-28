package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func appMeshRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
	}
	headers := map[string]string{}
	if method == http.MethodPut {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "appmesh")
}

func TestAppMeshStage0CatalogCoverage(t *testing.T) {
	if len(appMeshOperations) != 38 {
		t.Fatalf("expected 38 App Mesh operations from docs, got %d", len(appMeshOperations))
	}
	if len(appMeshOperationByName) != len(appMeshOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateMesh",
		"DescribeMesh",
		"ListMeshes",
		"CreateVirtualNode",
		"CreateRoute",
		"CreateVirtualService",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := appMeshOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(appMeshDataTypes) != 142 {
		t.Fatalf("expected 142 App Mesh data types from docs, got %d", len(appMeshDataTypes))
	}
	if len(appMeshDataTypeByName) != len(appMeshDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"MeshData",
		"VirtualNodeData",
		"VirtualRouterData",
		"RouteData",
		"VirtualServiceData",
		"GatewayRouteData",
	}
	for _, typeName := range requiredTypes {
		if _, ok := appMeshDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestAppMeshStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := appMeshRequest(t, ts, http.MethodGet, "/v20190125/unknown", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestAppMeshKnownRouteReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := appMeshRequest(t, ts, http.MethodGet, "/v20190125/meshes", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "meshes") {
		t.Fatalf("expected ListMeshes response body to include meshes, got %q", body)
	}
}

func TestAppMeshStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacer := strings.NewReplacer(
		"{meshName}", "apps",
		"{virtualGatewayName}", "stackyard-gateway",
		"{gatewayRouteName}", "stackyard-gateway-route",
		"{virtualNodeName}", "stackyard-node",
		"{virtualRouterName}", "stackyard-router",
		"{routeName}", "stackyard-route",
		"{virtualServiceName}", "stackyard.local",
	)

	for _, op := range appMeshOperations {
		path := replacer.Replace(op.URI)
		payload := ""
		if op.Method == http.MethodPut {
			payload = `{}`
		}
		resp := appMeshRequest(t, ts, op.Method, path, payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
