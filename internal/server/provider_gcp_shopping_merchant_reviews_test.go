package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShoppingMerchantReviewsRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReviewsContractServer(t)

	assertGCPShoppingMerchantReviewsSuccess(t, ts, http.MethodGet, "/gcp/reviews/v1beta/accounts/123456/merchantReviews/merchant-review-1001", nil, "merchantReviewId")
	assertGCPShoppingMerchantReviewsSuccess(t, ts, http.MethodGet, "/gcp/reviews/v1beta/accounts/123456/merchantReviews?pageSize=1", nil, "merchantReviews")
	assertGCPShoppingMerchantReviewsSuccess(t, ts, http.MethodPost, "/gcp/reviews/v1beta/accounts/123456/merchantReviews:insert?dataSource=accounts/123456/dataSources/104628", []byte(`{
		"merchantReview":{
			"merchantReviewId":"merchant-review-3001",
			"merchantReviewAttributes":{
				"title":"Fast delivery",
				"content":"Shipped ahead of schedule."
			}
		}
	}`), "merchantReviewId")
	assertGCPShoppingMerchantReviewsSuccess(t, ts, http.MethodDelete, "/gcp/reviews/v1beta/accounts/123456/merchantReviews/merchant-review-1001", nil, "{}")

	assertGCPShoppingMerchantReviewsSuccess(t, ts, http.MethodGet, "/gcp/reviews/v1beta/accounts/123456/productReviews/product-review-1001", nil, "productReviewId")
	assertGCPShoppingMerchantReviewsSuccess(t, ts, http.MethodGet, "/gcp/reviews/v1beta/accounts/123456/productReviews?pageSize=1", nil, "productReviews")
	assertGCPShoppingMerchantReviewsSuccess(t, ts, http.MethodPost, "/gcp/reviews/v1beta/accounts/123456/productReviews:insert?dataSource=accounts/123456/dataSources/104628", []byte(`{
		"productReview":{
			"productReviewId":"product-review-3001",
			"productReviewAttributes":{
				"title":"Great fit",
				"content":"Very comfortable and true to size."
			}
		}
	}`), "productReviewId")
	assertGCPShoppingMerchantReviewsSuccess(t, ts, http.MethodDelete, "/gcp/reviews/v1beta/accounts/123456/productReviews/product-review-1001", nil, "{}")
}

func TestGCPShoppingMerchantReviewsRouter_InsertRequiresDataSource(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReviewsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/reviews/v1beta/accounts/123456/merchantReviews:insert", []byte(`{
		"merchantReview":{"merchantReviewId":"merchant-review-1001"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-reviews",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from merchant reviews insert missing dataSource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument in response")
	}
}

func TestGCPShoppingMerchantReviewsRouter_InsertRejectsDataSourceAccountMismatch(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReviewsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/reviews/v1beta/accounts/123456/productReviews:insert?dataSource=accounts/999999/dataSources/104628", []byte(`{
		"productReview":{"productReviewId":"product-review-1001"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-reviews",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from product reviews insert account mismatch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"FailedPrecondition"`) {
		t.Fatalf("expected FailedPrecondition in response")
	}
}

func TestGCPShoppingMerchantReviewsRouter_InsertRejectsMissingBody(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReviewsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/reviews/v1beta/accounts/123456/merchantReviews:insert?dataSource=accounts/123456/dataSources/104628", nil, map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-reviews",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from merchant reviews insert missing body, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument in response")
	}
}

func TestGCPShoppingMerchantReviewsRouter_InsertRejectsMissingReviewID(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReviewsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/reviews/v1beta/accounts/123456/productReviews:insert?dataSource=accounts/123456/dataSources/104628", []byte(`{
		"productReview":{"productReviewAttributes":{"title":"good"}}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-reviews",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from product reviews insert missing productReviewId, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument in response")
	}
}

func TestGCPShoppingMerchantReviewsRouter_ListRejectsInvalidPaging(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReviewsContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/reviews/v1beta/accounts/123456/merchantReviews?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-reviews",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from merchant reviews list invalid pageSize, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument in response")
	}

	resp = providerContractRequest(t, ts, http.MethodGet, "/gcp/reviews/v1beta/accounts/123456/productReviews?pageToken=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-reviews",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from product reviews list invalid pageToken, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument in response")
	}
}

func TestGCPShoppingMerchantReviewsRouter_MissingResourceNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReviewsContractServer(t)

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/reviews/v1beta/accounts/123456/merchantReviews/missing-review", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-reviews",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from merchant reviews get missing resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound in response")
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, "/gcp/reviews/v1beta/accounts/missing/productReviews/product-review-1001", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-reviews",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from product reviews delete missing account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound in response")
	}
}

func TestGCPShoppingMerchantReviewsRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantReviewsContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-reviews",
		"Content-Type":            "application/json",
	}

	getMerchantResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/reviews/v1beta/accounts/123456/merchantReviews/merchant-review-1001", nil, headers)
	if getMerchantResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from merchant reviews get, got %d body=%s", getMerchantResp.StatusCode, string(providerContractBody(t, getMerchantResp)))
	}
	getMerchantBody := providerContractJSONMap(t, getMerchantResp)
	if _, ok := getMerchantBody["name"].(string); !ok {
		t.Fatalf("expected merchant review name string, got %#v", getMerchantBody["name"])
	}
	if _, ok := getMerchantBody["merchantReviewId"].(string); !ok {
		t.Fatalf("expected merchantReviewId string, got %#v", getMerchantBody["merchantReviewId"])
	}
	merchantAttrs, ok := getMerchantBody["merchantReviewAttributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected merchantReviewAttributes object, got %#v", getMerchantBody["merchantReviewAttributes"])
	}
	if _, ok := merchantAttrs["title"].(string); !ok {
		t.Fatalf("expected merchantReviewAttributes.title string, got %#v", merchantAttrs["title"])
	}
	if _, ok := merchantAttrs["content"].(string); !ok {
		t.Fatalf("expected merchantReviewAttributes.content string, got %#v", merchantAttrs["content"])
	}

	listProductResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/reviews/v1beta/accounts/123456/productReviews?pageSize=1", nil, headers)
	if listProductResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from product reviews list, got %d body=%s", listProductResp.StatusCode, string(providerContractBody(t, listProductResp)))
	}
	listProductBody := providerContractJSONMap(t, listProductResp)
	productReviews, ok := listProductBody["productReviews"].([]any)
	if !ok || len(productReviews) == 0 {
		t.Fatalf("expected productReviews list, got %#v", listProductBody["productReviews"])
	}
	first, ok := productReviews[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first product review object, got %#v", productReviews[0])
	}
	if _, ok := first["productReviewId"].(string); !ok {
		t.Fatalf("expected productReviewId string, got %#v", first["productReviewId"])
	}
	productAttrs, ok := first["productReviewAttributes"].(map[string]any)
	if !ok {
		t.Fatalf("expected productReviewAttributes object, got %#v", first["productReviewAttributes"])
	}
	if _, ok := productAttrs["title"].(string); !ok {
		t.Fatalf("expected productReviewAttributes.title string, got %#v", productAttrs["title"])
	}
	reviewLink, ok := productAttrs["reviewLink"].(map[string]any)
	if !ok {
		t.Fatalf("expected productReviewAttributes.reviewLink object, got %#v", productAttrs["reviewLink"])
	}
	if _, ok := reviewLink["link"].(string); !ok {
		t.Fatalf("expected productReviewAttributes.reviewLink.link string, got %#v", reviewLink["link"])
	}
}

func TestGCPShoppingMerchantReviewsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/shopping_merchant_reviews/sample?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant reviews contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shopping_merchant_reviews" {
		t.Fatalf("expected service=shopping_merchant_reviews, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShoppingMerchantReviewsContractServer(t *testing.T) *httptest.Server {
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

func assertGCPShoppingMerchantReviewsSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-reviews",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant reviews router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if expectBodyFragment != "" && !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("expected response body for %s %s to contain %q, got %s", method, path, expectBodyFragment, string(providerContractBody(t, resp)))
	}
}
