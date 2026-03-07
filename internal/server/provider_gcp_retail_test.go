package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPRetailRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPRetailContractServer(t)

	catalogParent := "/gcp/v2/projects/stackyard/locations/global/catalogs/default_catalog"
	branchParent := catalogParent + "/branches/default_branch"
	productName := branchParent + "/products/product-1"
	servingConfigName := catalogParent + "/servingConfigs/default_config"
	modelName := catalogParent + "/models/model-1"
	placementName := catalogParent + "/placements/default_search"

	assertGCPRetailSuccess(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/global/catalogs?pageSize=1", nil, "catalogs")
	assertGCPRetailSuccess(t, ts, http.MethodPatch, catalogParent+"?updateMask=displayName", []byte(`{"name":"projects/stackyard/locations/global/catalogs/default_catalog","displayName":"Retail Catalog Updated"}`), "Retail Catalog Updated")
	assertGCPRetailSuccess(t, ts, http.MethodPost, catalogParent+":setDefaultBranch", []byte(`{"branch":"projects/stackyard/locations/global/catalogs/default_catalog/branches/default_branch","note":"switch"}`), "{}")
	assertGCPRetailSuccess(t, ts, http.MethodGet, catalogParent+":getDefaultBranch", nil, "default_branch")

	assertGCPRetailSuccess(t, ts, http.MethodGet, catalogParent+"/completionConfig", nil, "matchingOrder")
	assertGCPRetailSuccess(t, ts, http.MethodPatch, catalogParent+"/completionConfig?updateMask=matchingOrder", []byte(`{"name":"projects/stackyard/locations/global/catalogs/default_catalog/completionConfig","matchingOrder":"exact-prefix"}`), "exact-prefix")
	assertGCPRetailSuccess(t, ts, http.MethodGet, catalogParent+"/attributesConfig", nil, "catalogAttributes")
	assertGCPRetailSuccess(t, ts, http.MethodPost, catalogParent+"/attributesConfig:addCatalogAttribute", []byte(`{"catalogAttribute":{"key":"size","searchableOption":"SEARCHABLE_ENABLED"}}`), "size")

	assertGCPRetailSuccess(t, ts, http.MethodPost, branchParent+"/products?productId=product-1", []byte(`{"title":"Stackyard Hoodie"}`), "Stackyard Hoodie")
	assertGCPRetailSuccess(t, ts, http.MethodGet, productName, nil, "product-1")
	assertGCPRetailSuccess(t, ts, http.MethodGet, branchParent+"/products?pageSize=1", nil, "products")
	assertGCPRetailSuccess(t, ts, http.MethodPatch, productName+"?updateMask=title", []byte(`{"name":"projects/stackyard/locations/global/catalogs/default_catalog/branches/default_branch/products/product-1","title":"Updated Hoodie"}`), "Updated Hoodie")
	assertGCPRetailSuccess(t, ts, http.MethodPost, branchParent+"/products:import", []byte(`{"inputConfig":{"productInlineSource":{"products":[{"id":"product-1","title":"Inline Product"}]}}}`), "operations/import-products")
	assertGCPRetailSuccess(t, ts, http.MethodPost, branchParent+"/products:purge", []byte(`{"filter":"availability:IN_STOCK","force":false}`), "operations/purge-products")
	assertGCPRetailSuccess(t, ts, http.MethodPost, productName+":setInventory", []byte(`{"inventory":{"name":"projects/stackyard/locations/global/catalogs/default_catalog/branches/default_branch/products/product-1"}}`), "operations/set-inventory")
	assertGCPRetailSuccess(t, ts, http.MethodPost, productName+":addFulfillmentPlaces", []byte(`{"product":"projects/stackyard/locations/global/catalogs/default_catalog/branches/default_branch/products/product-1","placeIds":["store-1"]}`), "operations/add-fulfillment")

	assertGCPRetailSuccess(t, ts, http.MethodPost, placementName+":search", []byte(`{"query":"hoodie","pageSize":1}`), "attributionToken")
	assertGCPRetailSuccess(t, ts, http.MethodPost, placementName+":predict", []byte(`{"userEvent":{"eventType":"search","visitorId":"visitor-1"}}`), "results")
	assertGCPRetailSuccess(t, ts, http.MethodPost, catalogParent+":completeQuery", []byte(`{"query":"hoo"}`), "completionResults")
	assertGCPRetailSuccess(t, ts, http.MethodPost, catalogParent+"/completionData:import", []byte(`{"inputConfig":{"bigQuerySource":{"projectId":"stackyard","datasetId":"retail","tableId":"completion"}}}`), "operations/import-completion-data")

	assertGCPRetailSuccess(t, ts, http.MethodPost, catalogParent+"/servingConfigs?servingConfigId=default_config", []byte(`{"displayName":"Default Config"}`), "Default Config")
	assertGCPRetailSuccess(t, ts, http.MethodGet, servingConfigName, nil, "servingConfigs/default_config")
	assertGCPRetailSuccess(t, ts, http.MethodGet, catalogParent+"/servingConfigs?pageSize=1", nil, "servingConfigs")
	assertGCPRetailSuccess(t, ts, http.MethodPost, servingConfigName+":addControl", []byte(`{"servingConfig":"projects/stackyard/locations/global/catalogs/default_catalog/servingConfigs/default_config","controlId":"control-1"}`), "control-1")

	assertGCPRetailSuccess(t, ts, http.MethodPost, catalogParent+"/userEvents:write", []byte(`{"userEvent":{"eventType":"detail-page-view","visitorId":"visitor-1"}}`), "detail-page-view")
	assertGCPRetailSuccess(t, ts, http.MethodPost, catalogParent+"/userEvents:collect", []byte(`{"userEvent":"eventType=detail-page-view&visitorId=visitor-1"}`), "contentType")
	assertGCPRetailSuccess(t, ts, http.MethodPost, catalogParent+"/userEvents:import", []byte(`{"inputConfig":{"userEventInlineSource":{"userEvents":[{"eventType":"detail-page-view","visitorId":"visitor-1"}]}}}`), "operations/import-user-events")

	assertGCPRetailSuccess(t, ts, http.MethodPost, catalogParent+"/models?modelId=model-1", []byte(`{"displayName":"Recommendations Model","type":"recommended-for-you"}`), "operations/create-model")
	assertGCPRetailSuccess(t, ts, http.MethodGet, modelName, nil, "models/model-1")
	assertGCPRetailSuccess(t, ts, http.MethodGet, catalogParent+"/models?pageSize=1", nil, "models")
	assertGCPRetailSuccess(t, ts, http.MethodPost, modelName+":tune", []byte(`{}`), "operations/tune-model")

	assertGCPRetailSuccess(t, ts, http.MethodPost, catalogParent+":exportAnalyticsMetrics", []byte(`{"outputConfig":{"gcsDestination":{"outputUriPrefix":"gs://stackyard-exports/metrics"}}}`), "operations/export-analytics")

	assertGCPRetailSuccess(t, ts, http.MethodPatch, catalogParent+"/generativeQuestionFeature", []byte(`{"catalog":"projects/stackyard/locations/global/catalogs/default_catalog","featureEnabled":true}`), "featureEnabled")
	assertGCPRetailSuccess(t, ts, http.MethodGet, catalogParent+"/generativeQuestionFeature", nil, "minimumProducts")
	assertGCPRetailSuccess(t, ts, http.MethodGet, catalogParent+"/generativeQuestions?pageSize=1", nil, "generativeQuestionConfigs")
	assertGCPRetailSuccess(t, ts, http.MethodPost, catalogParent+"/generativeQuestion:batchUpdate", []byte(`{"requests":[{"catalog":"projects/stackyard/locations/global/catalogs/default_catalog","facet":"brand"}]}`), "generativeQuestionConfigs")

	assertGCPRetailSuccess(t, ts, http.MethodGet, catalogParent+"/operations?pageSize=1", nil, "operations")
	assertGCPRetailSuccess(t, ts, http.MethodGet, catalogParent+"/operations/op-1", nil, "operations/op-1")
}

func TestGCPRetailRouter_ListProductsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPRetailContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v2/projects/stackyard/locations/global/catalogs/default_catalog/branches/default_branch/products?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "retail",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp retail router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRetailRouter_CreateProductRequiresProductID(t *testing.T) {
	t.Parallel()

	ts := newGCPRetailContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/global/catalogs/default_catalog/branches/default_branch/products", []byte(`{"title":"Stackyard"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "retail",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp retail router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRetailRouter_UpdateProductNameMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPRetailContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v2/projects/stackyard/locations/global/catalogs/default_catalog/branches/default_branch/products/product-1?updateMask=title", []byte(`{"name":"projects/stackyard/locations/global/catalogs/default_catalog/branches/default_branch/products/product-2","title":"Mismatch"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "retail",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp retail router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRetailRouter_SearchRequiresQuery(t *testing.T) {
	t.Parallel()

	ts := newGCPRetailContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/global/catalogs/default_catalog/placements/default_search:search", []byte(`{"pageSize":1}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "retail",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp retail router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRetailRouter_SetDefaultBranchRequiresCatalogBranch(t *testing.T) {
	t.Parallel()

	ts := newGCPRetailContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/locations/global/catalogs/default_catalog:setDefaultBranch", []byte(`{"branch":"projects/stackyard/locations/global/catalogs/other_catalog/branches/default_branch"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "retail",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp retail router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRetailRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/retail?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp retail contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "retail" {
		t.Fatalf("expected service=retail, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPRetailContractServer(t *testing.T) *httptest.Server {
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

func assertGCPRetailSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "retail",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp retail router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
