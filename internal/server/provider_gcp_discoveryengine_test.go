package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPDiscoveryEngineRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPDiscoveryEngineContractServer(t)
	base := "/gcp/v1/projects/stackyard/locations/global/collections/default_collection"

	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodGet, base+"/engines?pageSize=1", nil, "engines")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodPost, base+"/engines?engineId=orders-engine", []byte(`{"engine":{"displayName":"orders-engine"}}`), "operations")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodGet, base+"/engines/orders-engine", nil, "orders-engine")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodPatch, base+"/engines/orders-engine", []byte(`{"engine":{"displayName":"orders-engine"}}`), "orders-engine")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodDelete, base+"/engines/orders-engine", nil, "operations")

	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodGet, base+"/dataStores?pageSize=1", nil, "dataStores")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodPost, base+"/dataStores?dataStoreId=orders-store", []byte(`{"dataStore":{"displayName":"orders-store"}}`), "operations")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodGet, base+"/dataStores/orders-store", nil, "orders-store")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodPatch, base+"/dataStores/orders-store", []byte(`{"dataStore":{"displayName":"orders-store"}}`), "orders-store")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodDelete, base+"/dataStores/orders-store", nil, "operations")

	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodPost, base+"/engines/orders-engine/servingConfigs/default_serving_config:search", []byte(`{"query":"latest order status"}`), "results")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodPost, base+"/engines/orders-engine/servingConfigs/default_serving_config:searchLite", []byte(`{"query":"lite"}`), "results")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodPost, base+"/dataStores/orders-store:completeQuery", []byte(`{"query":"order"}`), "querySuggestions")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodPost, base+"/engines/orders-engine/servingConfigs/default_serving_config:answer", []byte(`{"query":{"text":"status"}}`), "answer")

	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodGet, base+"/dataStores/orders-store/conversations?pageSize=1", nil, "conversations")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodPost, base+"/dataStores/orders-store/conversations", []byte(`{"conversation":{"name":"conv-1"}}`), "conversations/conv-1")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodGet, base+"/dataStores/orders-store/conversations/conv-1", nil, "conv-1")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodPatch, base+"/dataStores/orders-store/conversations/conv-1", []byte(`{"conversation":{"userPseudoId":"user-1"}}`), "conv-1")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodPost, base+"/dataStores/orders-store/conversations/conv-1:converse", []byte(`{"query":{"input":"orders"}}`), "reply")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodDelete, base+"/dataStores/orders-store/conversations/conv-1", nil, "{}")

	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodGet, base+"/dataStores/orders-store/sessions?pageSize=1", nil, "sessions")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodPost, base+"/dataStores/orders-store/sessions", []byte(`{"session":{"name":"session-1"}}`), "sessions/session-1")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodGet, base+"/dataStores/orders-store/sessions/session-1", nil, "session-1")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodPatch, base+"/dataStores/orders-store/sessions/session-1", []byte(`{"session":{"displayName":"session-1"}}`), "session-1")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodDelete, base+"/dataStores/orders-store/sessions/session-1", nil, "{}")

	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodGet, base+"/operations?pageSize=1", nil, "operations")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodGet, base+"/operations/op-1", nil, "op-1")
	assertGCPDiscoveryEngineSuccess(t, ts, http.MethodPost, base+"/operations/op-1:cancel", []byte(`{}`), "{}")
}

func TestGCPDiscoveryEngineRouter_GrpcRoutesStillNotImplemented(t *testing.T) {
	t.Parallel()

	ts := newGCPDiscoveryEngineContractServer(t)
	assertGCPDiscoveryEngineNotImplemented(t, ts, http.MethodPost, "/gcp/google.cloud.discoveryengine.v1.SearchService/Search", "google.cloud.discoveryengine.v1.SearchService/Search")
}

func TestGCPDiscoveryEngineRouter_ListEnginesInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPDiscoveryEngineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/global/collections/default_collection/engines?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp discoveryengine router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func TestGCPDiscoveryEngineRouter_SearchRequiresQuery(t *testing.T) {
	t.Parallel()

	ts := newGCPDiscoveryEngineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/projects/stackyard/locations/global/collections/default_collection/engines/orders-engine/servingConfigs/default_serving_config:search", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp discoveryengine router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", string(providerContractBody(t, resp)))
	}
}

func newGCPDiscoveryEngineContractServer(t *testing.T) *httptest.Server {
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

func assertGCPDiscoveryEngineNotImplemented(t *testing.T, ts *httptest.Server, method, path, expectPathFragment string) {
	t.Helper()

	resp := providerContractRequest(t, ts, method, path, nil, nil)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected 501 from gcp discoveryengine router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"provider":"gcp"`) || !strings.Contains(body, expectPathFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func assertGCPDiscoveryEngineSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp discoveryengine router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, string(providerContractBody(t, resp)))
	}
}
