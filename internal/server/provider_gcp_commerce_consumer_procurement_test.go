package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPCommerceConsumerProcurementRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPCommerceConsumerProcurementContractServer(t)

	base := "/gcp/v1/billingAccounts/0123456789/orders"
	assertGCPCommerceConsumerProcurementSuccess(t, ts, http.MethodGet, base+"?pageSize=1", nil, "orders")
	assertGCPCommerceConsumerProcurementSuccess(t, ts, http.MethodGet, base+"/order-1", nil, "order-1")
	assertGCPCommerceConsumerProcurementSuccess(t, ts, http.MethodPost, base+":place", []byte(`{"displayName":"Team commitment order"}`), "operations/placeOrder")
	assertGCPCommerceConsumerProcurementSuccess(t, ts, http.MethodPost, base+"/order-1:modify", []byte(`{"displayName":"Updated","modifications":[{"lineItemId":"line-item-1"}]}`), "operations/modifyOrder")
	assertGCPCommerceConsumerProcurementSuccess(t, ts, http.MethodPost, base+"/order-1:cancel", []byte(`{"cancellationPolicy":"CANCEL_AT_TERM_END"}`), "operations/cancelOrder")
	assertGCPCommerceConsumerProcurementSuccess(t, ts, http.MethodGet, base+"/order-1/operations/op-1", nil, `"done":true`)
}

func TestGCPCommerceConsumerProcurementRouter_ListOrdersInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPCommerceConsumerProcurementContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/billingAccounts/0123456789/orders?pageSize=bad", nil, nil)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp commerce consumer procurement router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCommerceConsumerProcurementRouter_PlaceOrderRequiresDisplayName(t *testing.T) {
	t.Parallel()

	ts := newGCPCommerceConsumerProcurementContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/billingAccounts/0123456789/orders:place", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp commerce consumer procurement router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPCommerceConsumerProcurementRouter_ModifyOrderRequiresModifications(t *testing.T) {
	t.Parallel()

	ts := newGCPCommerceConsumerProcurementContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/billingAccounts/0123456789/orders/order-1:modify", []byte(`{"displayName":"Updated","modifications":[]}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp commerce consumer procurement router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func newGCPCommerceConsumerProcurementContractServer(t *testing.T) *httptest.Server {
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

func assertGCPCommerceConsumerProcurementSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp commerce consumer procurement router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}

func TestGCPCommerceConsumerProcurementRouter_CancelOrderRequiresCancellationPolicy(t *testing.T) {
	t.Parallel()

	ts := newGCPCommerceConsumerProcurementContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/billingAccounts/0123456789/orders/order-1:cancel", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp commerce consumer procurement router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}
