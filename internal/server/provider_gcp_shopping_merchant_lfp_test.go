package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShoppingMerchantLFPRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantLFPContractServer(t)
	parent := "accounts/123456"
	storeName := parent + "/lfpStores/567890~store-nyc"

	assertGCPShoppingMerchantLFPSuccess(t, ts, http.MethodPost, "/gcp/lfp/v1/"+parent+"/lfpStores:insert", []byte(`{
		"targetAccount":"567890",
		"storeCode":"store-nyc",
		"storeAddress":"1600 Amphitheatre Pkwy, Mountain View, CA 94043, USA"
	}`), "/lfpStores/567890~store-nyc")
	assertGCPShoppingMerchantLFPSuccess(t, ts, http.MethodGet, "/gcp/lfp/v1/"+storeName, nil, storeName)
	assertGCPShoppingMerchantLFPSuccess(t, ts, http.MethodGet, "/gcp/lfp/v1/"+parent+"/lfpStores?targetAccount=567890&pageSize=1", nil, `"lfpStores"`)

	assertGCPShoppingMerchantLFPSuccess(t, ts, http.MethodPost, "/gcp/lfp/v1/"+parent+"/lfpInventories:insert", []byte(`{
		"targetAccount":"567890",
		"storeCode":"store-nyc",
		"offerId":"offer-1001",
		"regionCode":"US",
		"contentLanguage":"en",
		"availability":"in stock",
		"price":{"currencyCode":"USD","amountMicros":"12990000"},
		"quantity":"7"
	}`), "/lfpInventories/567890~store-nyc~offer-1001")

	assertGCPShoppingMerchantLFPSuccess(t, ts, http.MethodPost, "/gcp/lfp/v1/"+parent+"/lfpSales:insert", []byte(`{
		"targetAccount":"567890",
		"storeCode":"store-nyc",
		"offerId":"offer-1001",
		"regionCode":"US",
		"contentLanguage":"en",
		"gtin":"00012345678905",
		"price":{"currencyCode":"USD","amountMicros":"14990000"},
		"quantity":"1",
		"saleTime":"2026-01-01T12:34:56Z"
	}`), "/lfpSales/567890~store-nyc~offer-1001")

	assertGCPShoppingMerchantLFPSuccess(t, ts, http.MethodGet, "/gcp/lfp/v1/"+parent+"/lfpMerchantStates/567890", nil, "/lfpMerchantStates/567890")
	assertGCPShoppingMerchantLFPSuccess(t, ts, http.MethodDelete, "/gcp/lfp/v1/"+storeName, nil, "{}")
}

func TestGCPShoppingMerchantLFPRouter_ListRequiresTargetAccount(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantLFPContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/lfp/v1/accounts/123456/lfpStores?pageSize=1", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-lfp",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant lfp list stores, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantLFPRouter_InsertStoreRequiresFields(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantLFPContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/lfp/v1/accounts/123456/lfpStores:insert", []byte(`{
		"storeCode":"store-nyc"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-lfp",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant lfp insert store, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantLFPRouter_InsertInventoryUnknownStoreFailedPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantLFPContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/lfp/v1/accounts/123456/lfpInventories:insert", []byte(`{
		"targetAccount":"567890",
		"storeCode":"missing-store",
		"offerId":"offer-1001",
		"regionCode":"US",
		"contentLanguage":"en",
		"availability":"in stock"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-lfp",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant lfp insert inventory, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"FailedPrecondition"`) {
		t.Fatalf("expected FailedPrecondition error in response")
	}
}

func TestGCPShoppingMerchantLFPRouter_InsertSaleRequiresPriceAndSaleTime(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantLFPContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/lfp/v1/accounts/123456/lfpSales:insert", []byte(`{
		"targetAccount":"567890",
		"storeCode":"store-nyc",
		"offerId":"offer-1001",
		"regionCode":"US",
		"contentLanguage":"en",
		"gtin":"00012345678905",
		"quantity":"1"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-lfp",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant lfp insert sale, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantLFPRouter_GetMissingStoreNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantLFPContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/lfp/v1/accounts/123456/lfpStores/567890~missing-store", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-lfp",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp shopping merchant lfp get store, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound error in response")
	}
}

func TestGCPShoppingMerchantLFPRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantLFPContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-lfp",
	}

	storeResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/lfp/v1/accounts/123456/lfpStores:insert", []byte(`{
		"targetAccount":"567890",
		"storeCode":"store-nyc",
		"storeAddress":"1600 Amphitheatre Pkwy, Mountain View, CA 94043, USA"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-lfp",
	})
	if storeResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant lfp insert store, got %d body=%s", storeResp.StatusCode, string(providerContractBody(t, storeResp)))
	}
	storeBody := providerContractJSONMap(t, storeResp)
	if _, ok := storeBody["name"].(string); !ok {
		t.Fatalf("expected store name string, got %#v", storeBody["name"])
	}
	if _, ok := storeBody["targetAccount"].(string); !ok {
		t.Fatalf("expected targetAccount string, got %#v", storeBody["targetAccount"])
	}
	if _, ok := storeBody["storeCode"].(string); !ok {
		t.Fatalf("expected storeCode string, got %#v", storeBody["storeCode"])
	}

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/lfp/v1/accounts/123456/lfpStores?targetAccount=567890&pageSize=1", nil, headers)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant lfp list stores, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	stores, ok := listBody["lfpStores"].([]any)
	if !ok || len(stores) == 0 {
		t.Fatalf("expected lfpStores array, got %#v", listBody["lfpStores"])
	}
	firstStore, ok := stores[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first store object, got %#v", stores[0])
	}
	if _, ok := firstStore["storeAddress"].(string); !ok {
		t.Fatalf("expected storeAddress string, got %#v", firstStore["storeAddress"])
	}

	inventoryResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/lfp/v1/accounts/123456/lfpInventories:insert", []byte(`{
		"targetAccount":"567890",
		"storeCode":"store-nyc",
		"offerId":"offer-1001",
		"regionCode":"US",
		"contentLanguage":"en",
		"availability":"in stock",
		"price":{"currencyCode":"USD","amountMicros":"12990000"},
		"quantity":"7"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-lfp",
	})
	if inventoryResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant lfp insert inventory, got %d body=%s", inventoryResp.StatusCode, string(providerContractBody(t, inventoryResp)))
	}
	inventoryBody := providerContractJSONMap(t, inventoryResp)
	if _, ok := inventoryBody["offerId"].(string); !ok {
		t.Fatalf("expected offerId string, got %#v", inventoryBody["offerId"])
	}
	price, ok := inventoryBody["price"].(map[string]any)
	if !ok {
		t.Fatalf("expected price object, got %#v", inventoryBody["price"])
	}
	if _, ok := price["amountMicros"].(string); !ok {
		t.Fatalf("expected price.amountMicros string, got %#v", price["amountMicros"])
	}

	saleResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/lfp/v1/accounts/123456/lfpSales:insert", []byte(`{
		"targetAccount":"567890",
		"storeCode":"store-nyc",
		"offerId":"offer-1001",
		"regionCode":"US",
		"contentLanguage":"en",
		"gtin":"00012345678905",
		"price":{"currencyCode":"USD","amountMicros":"14990000"},
		"quantity":"1",
		"saleTime":"2026-01-01T12:34:56Z"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-lfp",
	})
	if saleResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant lfp insert sale, got %d body=%s", saleResp.StatusCode, string(providerContractBody(t, saleResp)))
	}
	saleBody := providerContractJSONMap(t, saleResp)
	if _, ok := saleBody["uid"].(string); !ok {
		t.Fatalf("expected uid string, got %#v", saleBody["uid"])
	}
	if _, ok := saleBody["saleTime"].(string); !ok {
		t.Fatalf("expected saleTime string, got %#v", saleBody["saleTime"])
	}

	stateResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/lfp/v1/accounts/123456/lfpMerchantStates/567890", nil, headers)
	if stateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant lfp get merchant state, got %d body=%s", stateResp.StatusCode, string(providerContractBody(t, stateResp)))
	}
	stateBody := providerContractJSONMap(t, stateResp)
	if _, ok := stateBody["linkedGbps"].(string); !ok {
		t.Fatalf("expected linkedGbps string, got %#v", stateBody["linkedGbps"])
	}
	storeStates, ok := stateBody["storeStates"].([]any)
	if !ok || len(storeStates) == 0 {
		t.Fatalf("expected storeStates array, got %#v", stateBody["storeStates"])
	}
	inventoryStats, ok := stateBody["inventoryStats"].(map[string]any)
	if !ok {
		t.Fatalf("expected inventoryStats object, got %#v", stateBody["inventoryStats"])
	}
	if _, ok := inventoryStats["submittedEntries"].(string); !ok {
		t.Fatalf("expected inventoryStats.submittedEntries string, got %#v", inventoryStats["submittedEntries"])
	}
}

func TestGCPShoppingMerchantLFPRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/shopping_merchant_lfp/sample?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant lfp contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shopping_merchant_lfp" {
		t.Fatalf("expected service=shopping_merchant_lfp, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShoppingMerchantLFPContractServer(t *testing.T) *httptest.Server {
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

func assertGCPShoppingMerchantLFPSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-lfp",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant lfp router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if expectBodyFragment != "" && !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("expected response body for %s %s to contain %q, got %s", method, path, expectBodyFragment, string(providerContractBody(t, resp)))
	}
}
