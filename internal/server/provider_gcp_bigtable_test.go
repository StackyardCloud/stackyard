package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPBigtableRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPBigtableContractServer(t)

	assertGCPBigtableSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/instances?pageSize=1", nil, "instances")
	assertGCPBigtableSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/instances/dev-instance", nil, "instances/dev-instance")
	assertGCPBigtableSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/instances/dev-instance/tables?pageSize=1", nil, "tables")
	assertGCPBigtableSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/instances/dev-instance/tables/orders", nil, "tables/orders")
	assertGCPBigtableSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/instances/dev-instance/tables?tableId=orders", []byte(`{"columnFamilies":{"cf1":{}}}`), "createTable")
	assertGCPBigtableSuccess(t, ts, http.MethodDelete, "/gcp/v2/projects/stackyard/instances/dev-instance/tables/orders", nil, "deleteTable")
	assertGCPBigtableSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/instances/dev-instance/tables/orders:dropRowRange", []byte(`{"deleteAllDataFromTable":true}`), "dropped")
	assertGCPBigtableSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/instances/dev-instance/tables/orders:readRows", []byte(`{"rowsLimit":1}`), "chunks")
	assertGCPBigtableSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/instances/dev-instance/tables/orders:sampleRowKeys", nil, "sampleRowKeys")
	assertGCPBigtableSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/instances/dev-instance/tables/orders:mutateRow", []byte(`{"mutations":[{"setCell":{"familyName":"cf1"}}]}`), "status")
}

func TestGCPBigtableRouter_ListTablesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPBigtableContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/instances/dev-instance/tables?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp bigtable router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPBigtableRouter_CreateTableRequiresTableID(t *testing.T) {
	t.Parallel()

	ts := newGCPBigtableContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/instances/dev-instance/tables", []byte(`{"columnFamilies":{"cf1":{}}}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp bigtable router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPBigtableRouter_MutateRowRequiresMutations(t *testing.T) {
	t.Parallel()

	ts := newGCPBigtableContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/instances/dev-instance/tables/orders:mutateRow", []byte(`{"rowKey":"order-1"}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp bigtable router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPBigtableContractServer(t *testing.T) *httptest.Server {
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

func assertGCPBigtableSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp bigtable router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
