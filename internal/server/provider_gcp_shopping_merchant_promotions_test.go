package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShoppingMerchantPromotionsRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantPromotionsContractServer(t)
	parent := "accounts/123456"
	promotionName := parent + "/promotions/en~US~promo-1001"

	assertGCPShoppingMerchantPromotionsSuccess(t, ts, http.MethodGet, "/gcp/promotions/v1/"+promotionName, nil, `"name":"`+promotionName+`"`)
	assertGCPShoppingMerchantPromotionsSuccess(t, ts, http.MethodGet, "/gcp/promotions/v1/"+parent+"/promotions?pageSize=1", nil, `"promotions"`)
	assertGCPShoppingMerchantPromotionsSuccess(t, ts, http.MethodPost, "/gcp/promotions/v1/"+parent+"/promotions:insert", []byte(`{
		"parent":"accounts/123456",
		"dataSource":"accounts/123456/dataSources/104628",
		"promotion":{
			"promotionId":"promo-1001",
			"contentLanguage":"en",
			"targetCountry":"US",
			"redemptionChannel":["ONLINE"],
			"attributes":{
				"productApplicability":"ALL_PRODUCTS",
				"offerType":"NO_CODE",
				"longTitle":"Stackyard Promotion 1001",
				"couponValueType":"MONEY_OFF",
				"promotionEffectiveTimePeriod":{
					"startTime":"2026-01-01T00:00:00Z",
					"endTime":"2026-01-31T23:59:59Z"
				}
			}
		}
	}`), `"name":"`+promotionName+`"`)
}

func TestGCPShoppingMerchantPromotionsRouter_ListRejectsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantPromotionsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/promotions/v1/accounts/123456/promotions?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-promotions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant promotions list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantPromotionsRouter_GetMissingNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantPromotionsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/promotions/v1/accounts/123456/promotions/en~US~missing-promo", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-promotions",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp shopping merchant promotions get missing, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound error in response")
	}
}

func TestGCPShoppingMerchantPromotionsRouter_InsertRequiresDataSource(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantPromotionsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/promotions/v1/accounts/123456/promotions:insert", []byte(`{
		"parent":"accounts/123456",
		"promotion":{
			"promotionId":"promo-1001",
			"contentLanguage":"en",
			"targetCountry":"US",
			"redemptionChannel":["ONLINE"],
			"attributes":{
				"productApplicability":"ALL_PRODUCTS",
				"offerType":"NO_CODE",
				"longTitle":"Stackyard Promotion 1001",
				"couponValueType":"MONEY_OFF",
				"promotionEffectiveTimePeriod":{
					"startTime":"2026-01-01T00:00:00Z",
					"endTime":"2026-01-31T23:59:59Z"
				}
			}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-promotions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant promotions insert missing dataSource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantPromotionsRouter_InsertRequiresAttributes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantPromotionsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/promotions/v1/accounts/123456/promotions:insert", []byte(`{
		"parent":"accounts/123456",
		"dataSource":"accounts/123456/dataSources/104628",
		"promotion":{
			"promotionId":"promo-1001",
			"contentLanguage":"en",
			"targetCountry":"US",
			"redemptionChannel":["ONLINE"]
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-promotions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant promotions insert missing attributes, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantPromotionsRouter_InsertAccountMismatchFailedPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantPromotionsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/promotions/v1/accounts/123456/promotions:insert", []byte(`{
		"parent":"accounts/123456",
		"dataSource":"accounts/999999/dataSources/104628",
		"promotion":{
			"promotionId":"promo-1001",
			"contentLanguage":"en",
			"targetCountry":"US",
			"redemptionChannel":["ONLINE"],
			"attributes":{
				"productApplicability":"ALL_PRODUCTS",
				"offerType":"NO_CODE",
				"longTitle":"Stackyard Promotion 1001",
				"couponValueType":"MONEY_OFF",
				"promotionEffectiveTimePeriod":{
					"startTime":"2026-01-01T00:00:00Z",
					"endTime":"2026-01-31T23:59:59Z"
				}
			}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-promotions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant promotions insert account mismatch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"FailedPrecondition"`) {
		t.Fatalf("expected FailedPrecondition error in response")
	}
}

func TestGCPShoppingMerchantPromotionsRouter_InsertVersionConflictAborted(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantPromotionsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/promotions/v1/accounts/123456/promotions:insert", []byte(`{
		"parent":"accounts/123456",
		"dataSource":"accounts/123456/dataSources/104628",
		"promotion":{
			"promotionId":"promo-1001",
			"contentLanguage":"en",
			"targetCountry":"US",
			"redemptionChannel":["ONLINE"],
			"versionNumber":"1",
			"attributes":{
				"productApplicability":"ALL_PRODUCTS",
				"offerType":"NO_CODE",
				"longTitle":"Stackyard Promotion 1001",
				"couponValueType":"MONEY_OFF",
				"promotionEffectiveTimePeriod":{
					"startTime":"2026-01-01T00:00:00Z",
					"endTime":"2026-01-31T23:59:59Z"
				}
			}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-promotions",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 from gcp shopping merchant promotions insert stale version, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"Aborted"`) {
		t.Fatalf("expected Aborted error in response")
	}
}

func TestGCPShoppingMerchantPromotionsRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantPromotionsContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-promotions",
	}

	getResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/promotions/v1/accounts/123456/promotions/en~US~promo-1001", nil, headers)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant promotions get, got %d body=%s", getResp.StatusCode, string(providerContractBody(t, getResp)))
	}
	getBody := providerContractJSONMap(t, getResp)
	if _, ok := getBody["name"].(string); !ok {
		t.Fatalf("expected name string, got %#v", getBody["name"])
	}
	if _, ok := getBody["promotionId"].(string); !ok {
		t.Fatalf("expected promotionId string, got %#v", getBody["promotionId"])
	}
	if _, ok := getBody["redemptionChannel"].([]any); !ok {
		t.Fatalf("expected redemptionChannel array, got %#v", getBody["redemptionChannel"])
	}
	if _, ok := getBody["attributes"].(map[string]any); !ok {
		t.Fatalf("expected attributes object, got %#v", getBody["attributes"])
	}
	if _, ok := getBody["promotionStatus"].(map[string]any); !ok {
		t.Fatalf("expected promotionStatus object, got %#v", getBody["promotionStatus"])
	}

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/promotions/v1/accounts/123456/promotions?pageSize=1", nil, headers)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant promotions list, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	promotions, ok := listBody["promotions"].([]any)
	if !ok || len(promotions) == 0 {
		t.Fatalf("expected promotions array, got %#v", listBody["promotions"])
	}
	firstPromotion, ok := promotions[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first promotion object, got %#v", promotions[0])
	}
	if _, ok := firstPromotion["contentLanguage"].(string); !ok {
		t.Fatalf("expected contentLanguage string, got %#v", firstPromotion["contentLanguage"])
	}
	if _, ok := firstPromotion["targetCountry"].(string); !ok {
		t.Fatalf("expected targetCountry string, got %#v", firstPromotion["targetCountry"])
	}

	insertResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/promotions/v1/accounts/123456/promotions:insert", []byte(`{
		"parent":"accounts/123456",
		"dataSource":"accounts/123456/dataSources/104628",
		"promotion":{
			"promotionId":"promo-typed",
			"contentLanguage":"en",
			"targetCountry":"US",
			"redemptionChannel":["ONLINE"],
			"attributes":{
				"productApplicability":"ALL_PRODUCTS",
				"offerType":"NO_CODE",
				"longTitle":"Typed Stackyard Promotion",
				"couponValueType":"MONEY_OFF",
				"promotionEffectiveTimePeriod":{
					"startTime":"2026-01-01T00:00:00Z",
					"endTime":"2026-01-31T23:59:59Z"
				}
			}
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-promotions",
	})
	if insertResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant promotions insert, got %d body=%s", insertResp.StatusCode, string(providerContractBody(t, insertResp)))
	}
	insertBody := providerContractJSONMap(t, insertResp)
	if _, ok := insertBody["dataSource"].(string); !ok {
		t.Fatalf("expected dataSource string, got %#v", insertBody["dataSource"])
	}
	if _, ok := insertBody["versionNumber"].(string); !ok {
		t.Fatalf("expected versionNumber string, got %#v", insertBody["versionNumber"])
	}
}

func TestGCPShoppingMerchantPromotionsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/shopping_merchant_promotions/sample?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant promotions contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shopping_merchant_promotions" {
		t.Fatalf("expected service=shopping_merchant_promotions, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShoppingMerchantPromotionsContractServer(t *testing.T) *httptest.Server {
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

func assertGCPShoppingMerchantPromotionsSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-promotions",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant promotions router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if expectBodyFragment != "" && !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("expected response body for %s %s to contain %q, got %s", method, path, expectBodyFragment, string(providerContractBody(t, resp)))
	}
}
