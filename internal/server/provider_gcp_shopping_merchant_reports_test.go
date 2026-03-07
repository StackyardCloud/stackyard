package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShoppingMerchantReportsRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReportsContractServer(t)
	assertGCPShoppingMerchantReportsSuccess(t, ts, http.MethodPost, "/gcp/reports/v1/accounts/123456/reports:search", []byte(`{
		"parent":"accounts/123456",
		"query":"SELECT product_view.id, product_view.title FROM product_view",
		"pageSize":1
	}`), "results")
}

func TestGCPShoppingMerchantReportsRouter_SearchRequiresQuery(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReportsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/reports/v1/accounts/123456/reports:search", []byte(`{
		"parent":"accounts/123456",
		"pageSize":1
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-reports",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant reports search missing query, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantReportsRouter_SearchRejectsInvalidParent(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReportsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/reports/v1/accounts/123456/reports:search", []byte(`{
		"parent":"accounts/not valid",
		"query":"SELECT product_view.id FROM product_view"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-reports",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant reports search invalid parent, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantReportsRouter_SearchRejectsParentMismatch(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReportsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/reports/v1/accounts/123456/reports:search", []byte(`{
		"parent":"accounts/999999",
		"query":"SELECT product_view.id FROM product_view"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-reports",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant reports search parent mismatch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"FailedPrecondition"`) {
		t.Fatalf("expected FailedPrecondition error in response")
	}
}

func TestGCPShoppingMerchantReportsRouter_SearchRejectsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReportsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/reports/v1/accounts/123456/reports:search", []byte(`{
		"parent":"accounts/123456",
		"query":"SELECT product_view.id FROM product_view",
		"pageSize":"bad"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-reports",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant reports search invalid page size, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantReportsRouter_SearchRejectsInvalidPageToken(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReportsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/reports/v1/accounts/123456/reports:search", []byte(`{
		"parent":"accounts/123456",
		"query":"SELECT product_view.id FROM product_view",
		"pageToken":"bad"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-reports",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant reports search invalid page token, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantReportsRouter_SearchMissingAccountNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReportsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/reports/v1/accounts/missing/reports:search", []byte(`{
		"parent":"accounts/missing",
		"query":"SELECT product_view.id FROM product_view"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-reports",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp shopping merchant reports search missing account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound error in response")
	}
}

func TestGCPShoppingMerchantReportsRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReportsContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-reports",
		"Content-Type":            "application/json",
	}

	searchResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/reports/v1/accounts/123456/reports:search", []byte(`{
		"parent":"accounts/123456",
		"query":"SELECT product_view.id, product_view.title FROM product_view",
		"pageSize":1
	}`), headers)
	if searchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant reports search, got %d body=%s", searchResp.StatusCode, string(providerContractBody(t, searchResp)))
	}
	body := providerContractJSONMap(t, searchResp)
	results, ok := body["results"].([]any)
	if !ok || len(results) == 0 {
		t.Fatalf("expected results array, got %#v", body["results"])
	}
	first, ok := results[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first result object, got %#v", results[0])
	}
	productView, ok := first["productView"].(map[string]any)
	if !ok {
		t.Fatalf("expected productView object, got %#v", first["productView"])
	}
	if _, ok := productView["id"].(string); !ok {
		t.Fatalf("expected productView.id string, got %#v", productView["id"])
	}
	if _, ok := productView["title"].(string); !ok {
		t.Fatalf("expected productView.title string, got %#v", productView["title"])
	}
	perfView, ok := first["productPerformanceView"].(map[string]any)
	if !ok {
		t.Fatalf("expected productPerformanceView object, got %#v", first["productPerformanceView"])
	}
	if _, ok := perfView["clicks"].(string); !ok {
		t.Fatalf("expected productPerformanceView.clicks string, got %#v", perfView["clicks"])
	}
	if _, ok := perfView["impressions"].(string); !ok {
		t.Fatalf("expected productPerformanceView.impressions string, got %#v", perfView["impressions"])
	}
}

func TestGCPShoppingMerchantReportsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/shopping_merchant_reports/sample?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant reports contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shopping_merchant_reports" {
		t.Fatalf("expected service=shopping_merchant_reports, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShoppingMerchantReportsContractServer(t *testing.T) *httptest.Server {
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

func assertGCPShoppingMerchantReportsSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-reports",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant reports router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if expectBodyFragment != "" && !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("expected response body for %s %s to contain %q, got %s", method, path, expectBodyFragment, string(providerContractBody(t, resp)))
	}
}
