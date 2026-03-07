package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDatastoreRouter_QueryRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDatastoreContractServer(t)
	assertGCPDatastoreSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:lookup", "found")
	assertGCPDatastoreSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:runQuery", "entityResults")
	assertGCPDatastoreSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:runAggregationQuery", "aggregationResults")
}

func TestGCPDatastoreRouter_TransactionRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDatastoreContractServer(t)
	assertGCPDatastoreSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:beginTransaction", "transaction")
	assertGCPDatastoreSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:commit", "commitTime")
	assertGCPDatastoreSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:rollback", "rolledBack")
}

func TestGCPDatastoreRouter_IDAllocationRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDatastoreContractServer(t)
	assertGCPDatastoreSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:allocateIds", "keys")
	assertGCPDatastoreSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:reserveIds", "reserved")
}

func TestGCPDatastoreRouter_DatabaseScopedRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDatastoreContractServer(t)
	assertGCPDatastoreSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/databases/analytics:lookup", "analytics")
	assertGCPDatastoreSuccess(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/databases/analytics:commit", "commitTime")
}

func TestGCPDatastoreRouter_InvalidJSONBodyReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ts := newGCPDatastoreContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard:lookup", []byte("{"), nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp datastore router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPDatastoreContractServer(t *testing.T) *httptest.Server {
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

func assertGCPDatastoreSuccess(t *testing.T, ts *httptest.Server, method, path, expectBodyFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp datastore router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
