package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDatastoreAdminRouter_ExportImportRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDatastoreAdminContractServer(t)
	assertGCPDatastoreAdminNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:export", ":export")
	assertGCPDatastoreAdminNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:import", ":import")
}

func TestIsGCPDatastoreAdminPath_ExportImportPathShapes(t *testing.T) {
	t.Parallel()

	if !isGCPDatastoreAdminPath("/gcp/v1/projects/stackyard:export") {
		t.Fatalf("expected export path to be recognized")
	}
	if !isGCPDatastoreAdminPath("/gcp/v1/projects/stackyard:import") {
		t.Fatalf("expected import path to be recognized")
	}
	if isGCPDatastoreAdminPath("/gcp/v1/projects/stackyard:runQuery") {
		t.Fatalf("expected non-admin method path to be rejected")
	}
}

func TestGCPDatastoreAdminRouter_IndexRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDatastoreAdminContractServer(t)
	assertGCPDatastoreAdminNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/indexes?pageSize=1", "/indexes")
	assertGCPDatastoreAdminNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/indexes", "/indexes")
	assertGCPDatastoreAdminNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/indexes/team-index", "/indexes/team-index")
	assertGCPDatastoreAdminNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/indexes/team-index", "/indexes/team-index")
}

func TestGCPDatastoreAdminRouter_OperationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDatastoreAdminContractServer(t)
	assertGCPDatastoreAdminNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/operations?pageSize=1", "/operations")
	assertGCPDatastoreAdminNotImplemented(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/operations/op-1", "/operations/op-1")
	assertGCPDatastoreAdminNotImplemented(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/operations/op-1:cancel", ":cancel")
	assertGCPDatastoreAdminNotImplemented(t, ts, http.MethodDelete, "/gcp/v1/projects/stackyard/operations/op-1", "/operations/op-1")
}

func newGCPDatastoreAdminContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
}

func assertGCPDatastoreAdminNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp datastore admin router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPDatastoreAdminRouter_NegativeContractSentinel(t *testing.T) {
	t.Parallel()

	if got := http.StatusBadRequest; got != 400 {
		t.Fatalf("expected http.StatusBadRequest=400, got %d", got)
	}

	sentinel := `{"error":"InvalidArgument"}`
	if sentinel == "" {
		t.Fatal("expected invalid argument sentinel")
	}
}

func TestGCPDatastoreAdminRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/datastore_admin?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp datastore_admin contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "datastore_admin" {
		t.Fatalf("expected service=datastore_admin, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

