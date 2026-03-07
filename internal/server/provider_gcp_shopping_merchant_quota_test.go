package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShoppingMerchantQuotaRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantQuotaContractServer(t)
	assertGCPShoppingMerchantQuotaSuccess(t, ts, http.MethodGet, "/gcp/quota/v1/accounts/123456/quotas?pageSize=1", nil, "quotaGroups")
}

func TestGCPShoppingMerchantQuotaRouter_ListRejectsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantQuotaContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/quota/v1/accounts/123456/quotas?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-quota",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant quota list invalid pageSize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantQuotaRouter_ListRejectsInvalidPageToken(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantQuotaContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/quota/v1/accounts/123456/quotas?pageToken=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-quota",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant quota list invalid pageToken, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantQuotaRouter_ListMissingAccountNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantQuotaContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/quota/v1/accounts/missing/quotas", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-quota",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp shopping merchant quota list missing account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound error in response")
	}
}

func TestGCPShoppingMerchantQuotaRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantQuotaContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-quota",
	}

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/quota/v1/accounts/123456/quotas?pageSize=1", nil, headers)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant quota list, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	quotaGroups, ok := listBody["quotaGroups"].([]any)
	if !ok || len(quotaGroups) == 0 {
		t.Fatalf("expected quotaGroups array, got %#v", listBody["quotaGroups"])
	}
	first, ok := quotaGroups[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first quota group object, got %#v", quotaGroups[0])
	}
	if _, ok := first["name"].(string); !ok {
		t.Fatalf("expected quota group name string, got %#v", first["name"])
	}
	if _, ok := first["quotaUsage"].(string); !ok {
		t.Fatalf("expected quotaUsage string, got %#v", first["quotaUsage"])
	}
	if _, ok := first["quotaLimit"].(string); !ok {
		t.Fatalf("expected quotaLimit string, got %#v", first["quotaLimit"])
	}
	if _, ok := first["quotaMinuteLimit"].(string); !ok {
		t.Fatalf("expected quotaMinuteLimit string, got %#v", first["quotaMinuteLimit"])
	}
	methodDetails, ok := first["methodDetails"].([]any)
	if !ok || len(methodDetails) == 0 {
		t.Fatalf("expected methodDetails array, got %#v", first["methodDetails"])
	}
	firstMethod, ok := methodDetails[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first methodDetails object, got %#v", methodDetails[0])
	}
	if _, ok := firstMethod["method"].(string); !ok {
		t.Fatalf("expected methodDetails.method string, got %#v", firstMethod["method"])
	}
	if _, ok := firstMethod["path"].(string); !ok {
		t.Fatalf("expected methodDetails.path string, got %#v", firstMethod["path"])
	}
}

func TestGCPShoppingMerchantQuotaRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/shopping_merchant_quota/sample?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant quota contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shopping_merchant_quota" {
		t.Fatalf("expected service=shopping_merchant_quota, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShoppingMerchantQuotaContractServer(t *testing.T) *httptest.Server {
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

func assertGCPShoppingMerchantQuotaSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-quota",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant quota router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if expectBodyFragment != "" && !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("expected response body for %s %s to contain %q, got %s", method, path, expectBodyFragment, string(providerContractBody(t, resp)))
	}
}
