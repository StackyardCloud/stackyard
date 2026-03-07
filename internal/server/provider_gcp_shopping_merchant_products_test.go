package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShoppingMerchantProductsRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductsContractServer(t)
	parent := "accounts/123456"
	productName := parent + "/products/en~US~sku-1001"
	productInputName := parent + "/productInputs/en~US~sku-1001"

	assertGCPShoppingMerchantProductsSuccess(t, ts, http.MethodGet, "/gcp/products/v1/"+productName, nil, `"name":"`+productName+`"`)
	assertGCPShoppingMerchantProductsSuccess(t, ts, http.MethodGet, "/gcp/products/v1/"+parent+"/products?pageSize=1", nil, `"products"`)
	assertGCPShoppingMerchantProductsSuccess(t, ts, http.MethodPost, "/gcp/products/v1/"+parent+"/productInputs:insert?dataSource=accounts/123456/dataSources/104628", []byte(`{
		"offerId":"sku-1001",
		"contentLanguage":"en",
		"feedLabel":"US",
		"productAttributes":{"title":"Stackyard SKU 1001"}
	}`), `"name":"`+productInputName+`"`)
	assertGCPShoppingMerchantProductsSuccess(t, ts, http.MethodPatch, "/gcp/products/v1/"+productInputName+"?dataSource=accounts/123456/dataSources/104628&updateMask=productAttributes.title", []byte(`{
		"name":"accounts/123456/productInputs/en~US~sku-1001",
		"offerId":"sku-1001",
		"contentLanguage":"en",
		"feedLabel":"US",
		"productAttributes":{"title":"Stackyard SKU 1001 Updated"}
	}`), "Updated")
	assertGCPShoppingMerchantProductsSuccess(t, ts, http.MethodDelete, "/gcp/products/v1/"+productInputName+"?dataSource=accounts/123456/dataSources/104628", nil, "{}")
}

func TestGCPShoppingMerchantProductsRouter_ListRejectsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/products/v1/accounts/123456/products?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-products",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant products list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantProductsRouter_GetMissingNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/products/v1/accounts/123456/products/en~US~missing-sku", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-products",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp shopping merchant products get, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound error in response")
	}
}

func TestGCPShoppingMerchantProductsRouter_InsertRequiresDataSource(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/products/v1/accounts/123456/productInputs:insert", []byte(`{
		"offerId":"sku-1001",
		"contentLanguage":"en",
		"feedLabel":"US"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-products",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant products insert without dataSource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantProductsRouter_InsertRequiresCoreFields(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/products/v1/accounts/123456/productInputs:insert?dataSource=accounts/123456/dataSources/104628", []byte(`{
		"offerId":"sku-1001"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-products",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant products insert missing fields, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantProductsRouter_UpdateRejectsUnsupportedMask(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/products/v1/accounts/123456/productInputs/en~US~sku-1001?dataSource=accounts/123456/dataSources/104628&updateMask=offerId", []byte(`{
		"name":"accounts/123456/productInputs/en~US~sku-1001",
		"offerId":"sku-1001",
		"contentLanguage":"en",
		"feedLabel":"US"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-products",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant products update unsupported mask, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantProductsRouter_InsertVersionConflictAborted(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/products/v1/accounts/123456/productInputs:insert?dataSource=accounts/123456/dataSources/104628", []byte(`{
		"offerId":"sku-1001",
		"contentLanguage":"en",
		"feedLabel":"US",
		"versionNumber":"1"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-products",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 from gcp shopping merchant products insert stale version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"Aborted"`) {
		t.Fatalf("expected Aborted error in response")
	}
}

func TestGCPShoppingMerchantProductsRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantProductsContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-products",
	}

	getResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/products/v1/accounts/123456/products/en~US~sku-1001", nil, headers)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant products get, got %d body=%s", getResp.StatusCode, string(providerContractBody(t, getResp)))
	}
	getBody := providerContractJSONMap(t, getResp)
	if _, ok := getBody["name"].(string); !ok {
		t.Fatalf("expected product name string, got %#v", getBody["name"])
	}
	if _, ok := getBody["offerId"].(string); !ok {
		t.Fatalf("expected offerId string, got %#v", getBody["offerId"])
	}
	if _, ok := getBody["legacyLocal"].(bool); !ok {
		t.Fatalf("expected legacyLocal bool, got %#v", getBody["legacyLocal"])
	}
	if _, ok := getBody["productStatus"].(map[string]any); !ok {
		t.Fatalf("expected productStatus object, got %#v", getBody["productStatus"])
	}

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/products/v1/accounts/123456/products?pageSize=1", nil, headers)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant products list, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	products, ok := listBody["products"].([]any)
	if !ok || len(products) == 0 {
		t.Fatalf("expected products array, got %#v", listBody["products"])
	}
	firstProduct, ok := products[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first product object, got %#v", products[0])
	}
	if _, ok := firstProduct["name"].(string); !ok {
		t.Fatalf("expected listed product name string, got %#v", firstProduct["name"])
	}

	insertResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/products/v1/accounts/123456/productInputs:insert?dataSource=accounts/123456/dataSources/104628", []byte(`{
		"offerId":"sku-typed",
		"contentLanguage":"en",
		"feedLabel":"US",
		"productAttributes":{"title":"Typed Product"},
		"customAttributes":[{"name":"material","value":"cotton"}]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-products",
	})
	if insertResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant products insert, got %d body=%s", insertResp.StatusCode, string(providerContractBody(t, insertResp)))
	}
	insertBody := providerContractJSONMap(t, insertResp)
	if _, ok := insertBody["name"].(string); !ok {
		t.Fatalf("expected productInput name string, got %#v", insertBody["name"])
	}
	if _, ok := insertBody["product"].(string); !ok {
		t.Fatalf("expected productInput product string, got %#v", insertBody["product"])
	}
	if _, ok := insertBody["productAttributes"].(map[string]any); !ok {
		t.Fatalf("expected productAttributes object, got %#v", insertBody["productAttributes"])
	}
	if _, ok := insertBody["customAttributes"].([]any); !ok {
		t.Fatalf("expected customAttributes array, got %#v", insertBody["customAttributes"])
	}
}

func TestGCPShoppingMerchantProductsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/shopping_merchant_products/sample?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant products contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shopping_merchant_products" {
		t.Fatalf("expected service=shopping_merchant_products, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShoppingMerchantProductsContractServer(t *testing.T) *httptest.Server {
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

func assertGCPShoppingMerchantProductsSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-products",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant products router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if expectBodyFragment != "" && !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("expected response body for %s %s to contain %q, got %s", method, path, expectBodyFragment, string(providerContractBody(t, resp)))
	}
}
