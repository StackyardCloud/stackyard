package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func odbRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, []byte(payload), headers, "odb")
}

func TestODBStage0CatalogCoverage(t *testing.T) {
	if len(odbOperations) != 43 {
		t.Fatalf("expected 43 ODB actions from docs, got %d", len(odbOperations))
	}
	if len(odbOperationByName) != len(odbOperations) {
		t.Fatalf("expected unique ODB action names")
	}

	requiredActions := []string{
		"InitializeService",
		"CreateOdbNetwork",
		"CreateCloudExadataInfrastructure",
		"CreateCloudVmCluster",
		"CreateCloudAutonomousVmCluster",
		"ListDbNodes",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := odbOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(odbDataTypes) != 41 {
		t.Fatalf("expected 41 ODB data types from docs, got %d", len(odbDataTypes))
	}
	if len(odbDataTypeByName) != len(odbDataTypes) {
		t.Fatalf("expected unique ODB data type names")
	}

	requiredTypes := []string{
		"OdbNetwork",
		"OdbPeeringConnection",
		"CloudExadataInfrastructure",
		"CloudVmCluster",
		"CloudAutonomousVmCluster",
		"DbNode",
		"DbServer",
	}
	for _, typeName := range requiredTypes {
		if _, ok := odbDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestODBStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := odbRequest(t, ts, http.MethodPost, "/odb/unknown", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestODBStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := odbRequest(t, ts, http.MethodPost, "/ListOdbNetworks", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "odbNetworks") {
		t.Fatalf("expected ListOdbNetworks response body to include odbNetworks, got %q", body)
	}
}

func TestODBStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range odbOperations {
		resp := odbRequest(t, ts, op.Method, op.URI, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
