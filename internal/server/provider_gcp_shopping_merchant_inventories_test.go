package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShoppingMerchantInventoriesRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantInventoriesContractServer(t)
	parent := "accounts/123456/products/sku-1001"

	assertGCPShoppingMerchantInventoriesSuccess(t, ts, http.MethodGet, "/gcp/inventories/v1/"+parent+"/localInventories?pageSize=1", nil, "localInventories")
	assertGCPShoppingMerchantInventoriesSuccess(t, ts, http.MethodPost, "/gcp/inventories/v1/"+parent+"/localInventories:insert", []byte(`{
		"storeCode":"store-nyc",
		"localInventoryAttributes":{
			"quantity":"8",
			"pickupMethod":"PICKUP_METHOD_UNSPECIFIED",
			"pickupSla":"PICKUP_SLA_UNSPECIFIED"
		}
	}`), "localInventories/store-nyc")
	assertGCPShoppingMerchantInventoriesSuccess(t, ts, http.MethodDelete, "/gcp/inventories/v1/"+parent+"/localInventories/store-nyc", nil, "{}")

	assertGCPShoppingMerchantInventoriesSuccess(t, ts, http.MethodGet, "/gcp/inventories/v1/"+parent+"/regionalInventories?pageSize=1", nil, "regionalInventories")
	assertGCPShoppingMerchantInventoriesSuccess(t, ts, http.MethodPost, "/gcp/inventories/v1/"+parent+"/regionalInventories:insert", []byte(`{
		"region":"us-east1",
		"regionalInventoryAttributes":{
			"availability":"REGIONAL_INVENTORY_AVAILABILITY_UNSPECIFIED"
		}
	}`), "regionalInventories/us-east1")
	assertGCPShoppingMerchantInventoriesSuccess(t, ts, http.MethodDelete, "/gcp/inventories/v1/"+parent+"/regionalInventories/us-east1", nil, "{}")
}

func TestGCPShoppingMerchantInventoriesRouter_ListRejectsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantInventoriesContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/inventories/v1/accounts/123456/products/sku-1001/localInventories?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-inventories",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant inventories local list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantInventoriesRouter_InsertLocalRequiresStoreCode(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantInventoriesContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/inventories/v1/accounts/123456/products/sku-1001/localInventories:insert", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-inventories",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant inventories local insert, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantInventoriesRouter_InsertLocalPickupPairValidation(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantInventoriesContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/inventories/v1/accounts/123456/products/sku-1001/localInventories:insert", []byte(`{
		"storeCode":"store-nyc",
		"localInventoryAttributes":{"pickupMethod":"PICKUP_METHOD_UNSPECIFIED"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-inventories",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant inventories local insert pickup validation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"FailedPrecondition"`) {
		t.Fatalf("expected FailedPrecondition error in response")
	}
}

func TestGCPShoppingMerchantInventoriesRouter_InsertRegionalRequiresRegion(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantInventoriesContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/inventories/v1/accounts/123456/products/sku-1001/regionalInventories:insert", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-inventories",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant inventories regional insert, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantInventoriesRouter_InsertRegionalSalePriceValidation(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantInventoriesContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/inventories/v1/accounts/123456/products/sku-1001/regionalInventories:insert", []byte(`{
		"region":"us-east1",
		"regionalInventoryAttributes":{
			"salePriceEffectiveDate":{"startTime":"2026-01-01T00:00:00Z","endTime":"2026-01-02T00:00:00Z"}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-inventories",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant inventories regional insert sale-price validation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"FailedPrecondition"`) {
		t.Fatalf("expected FailedPrecondition error in response")
	}
}

func TestGCPShoppingMerchantInventoriesRouter_DeleteMissingNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantInventoriesContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodDelete, "/gcp/inventories/v1/accounts/123456/products/sku-1001/localInventories/missing-store", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-inventories",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp shopping merchant inventories delete local, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound error in response")
	}
}

func TestGCPShoppingMerchantInventoriesRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantInventoriesContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-inventories",
	}

	localResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/inventories/v1/accounts/123456/products/sku-1001/localInventories?pageSize=1", nil, headers)
	if localResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant inventories local list, got %d body=%s", localResp.StatusCode, string(providerContractBody(t, localResp)))
	}
	localBody := providerContractJSONMap(t, localResp)
	locals, ok := localBody["localInventories"].([]any)
	if !ok || len(locals) == 0 {
		t.Fatalf("expected localInventories array, got %#v", localBody["localInventories"])
	}
	firstLocal, ok := locals[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first local inventory object, got %#v", locals[0])
	}
	if _, ok := firstLocal["name"].(string); !ok {
		t.Fatalf("expected local inventory name string, got %#v", firstLocal["name"])
	}
	if _, ok := firstLocal["account"].(string); !ok {
		t.Fatalf("expected local inventory account string, got %#v", firstLocal["account"])
	}
	if _, ok := firstLocal["storeCode"].(string); !ok {
		t.Fatalf("expected local inventory storeCode string, got %#v", firstLocal["storeCode"])
	}
	if _, ok := firstLocal["localInventoryAttributes"].(map[string]any); !ok {
		t.Fatalf("expected localInventoryAttributes object, got %#v", firstLocal["localInventoryAttributes"])
	}

	regionalResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/inventories/v1/accounts/123456/products/sku-1001/regionalInventories?pageSize=1", nil, headers)
	if regionalResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant inventories regional list, got %d body=%s", regionalResp.StatusCode, string(providerContractBody(t, regionalResp)))
	}
	regionalBody := providerContractJSONMap(t, regionalResp)
	regionalItems, ok := regionalBody["regionalInventories"].([]any)
	if !ok || len(regionalItems) == 0 {
		t.Fatalf("expected regionalInventories array, got %#v", regionalBody["regionalInventories"])
	}
	firstRegional, ok := regionalItems[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first regional inventory object, got %#v", regionalItems[0])
	}
	if _, ok := firstRegional["name"].(string); !ok {
		t.Fatalf("expected regional inventory name string, got %#v", firstRegional["name"])
	}
	if _, ok := firstRegional["region"].(string); !ok {
		t.Fatalf("expected regional inventory region string, got %#v", firstRegional["region"])
	}
	if _, ok := firstRegional["regionalInventoryAttributes"].(map[string]any); !ok {
		t.Fatalf("expected regionalInventoryAttributes object, got %#v", firstRegional["regionalInventoryAttributes"])
	}
}

func TestGCPShoppingMerchantInventoriesRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/shopping_merchant_inventories/sample?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant inventories contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shopping_merchant_inventories" {
		t.Fatalf("expected service=shopping_merchant_inventories, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShoppingMerchantInventoriesContractServer(t *testing.T) *httptest.Server {
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

func assertGCPShoppingMerchantInventoriesSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-inventories",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant inventories router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if expectBodyFragment != "" && !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("expected response body for %s %s to contain %q, got %s", method, path, expectBodyFragment, string(providerContractBody(t, resp)))
	}
}
