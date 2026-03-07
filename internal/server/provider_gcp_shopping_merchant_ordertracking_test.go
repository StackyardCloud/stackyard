package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShoppingMerchantOrdertrackingRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantOrdertrackingContractServer(t)

	assertGCPShoppingMerchantOrdertrackingSuccess(t, ts, http.MethodPost, "/gcp/ordertracking/v1/accounts/123456/orderTrackingSignals", []byte(`{
		"orderTrackingSignal":{
			"orderCreatedTime":"2026-01-02T03:04:05Z",
			"orderId":"ORDER-1001",
			"shippingInfo":[{
				"shipmentId":"SHIP-1001",
				"shippingStatus":"SHIPPED",
				"originPostalCode":"94043",
				"originRegionCode":"US",
				"trackingId":"TRACK-1001",
				"carrier":"UPS"
			}],
			"lineItems":[{
				"lineItemId":"line-1",
				"productId":"online:en:US:offer-1001",
				"quantity":"2"
			}]
		}
	}`), `"orderTrackingSignalId"`)
}

func TestGCPShoppingMerchantOrdertrackingRouter_CreateRequiresBody(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantOrdertrackingContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/ordertracking/v1/accounts/123456/orderTrackingSignals", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-ordertracking",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant ordertracking create without body, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantOrdertrackingRouter_CreateRequiresOrderTrackingSignal(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantOrdertrackingContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/ordertracking/v1/accounts/123456/orderTrackingSignals", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-ordertracking",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant ordertracking create missing signal, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantOrdertrackingRouter_CreateRejectsInvalidShippingStatus(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantOrdertrackingContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/ordertracking/v1/accounts/123456/orderTrackingSignals", []byte(`{
		"orderTrackingSignal":{
			"orderCreatedTime":"2026-01-02T03:04:05Z",
			"orderId":"ORDER-1001",
			"shippingInfo":[{
				"shipmentId":"SHIP-1001",
				"shippingStatus":"INVALID",
				"originPostalCode":"94043",
				"originRegionCode":"US",
				"trackingId":"TRACK-1001",
				"carrier":"UPS"
			}],
			"lineItems":[{
				"lineItemId":"line-1",
				"productId":"online:en:US:offer-1001",
				"quantity":"2"
			}]
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-ordertracking",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant ordertracking invalid shipping status, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantOrdertrackingRouter_CreateRejectsInvalidMappingReference(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantOrdertrackingContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/ordertracking/v1/accounts/123456/orderTrackingSignals", []byte(`{
		"orderTrackingSignal":{
			"orderCreatedTime":"2026-01-02T03:04:05Z",
			"orderId":"ORDER-1001",
			"shippingInfo":[{
				"shipmentId":"SHIP-1001",
				"shippingStatus":"SHIPPED",
				"originPostalCode":"94043",
				"originRegionCode":"US",
				"trackingId":"TRACK-1001",
				"carrier":"UPS"
			}],
			"lineItems":[{
				"lineItemId":"line-1",
				"productId":"online:en:US:offer-1001",
				"quantity":"2"
			}],
			"shipmentLineItemMapping":[{
				"shipmentId":"unknown-shipment",
				"lineItemId":"line-1",
				"quantity":"1"
			}]
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-ordertracking",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant ordertracking invalid mapping, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"FailedPrecondition"`) {
		t.Fatalf("expected FailedPrecondition error in response")
	}
}

func TestGCPShoppingMerchantOrdertrackingRouter_CreateDuplicateConflict(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantOrdertrackingContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/ordertracking/v1/accounts/123456/orderTrackingSignals", []byte(`{
		"orderTrackingSignal":{
			"orderCreatedTime":"2026-01-02T03:04:05Z",
			"orderId":"duplicate-order",
			"shippingInfo":[{
				"shipmentId":"SHIP-1001",
				"shippingStatus":"SHIPPED",
				"originPostalCode":"94043",
				"originRegionCode":"US",
				"trackingId":"TRACK-1001",
				"carrier":"UPS"
			}],
			"lineItems":[{
				"lineItemId":"line-1",
				"productId":"online:en:US:offer-1001",
				"quantity":"2"
			}]
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-ordertracking",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 from gcp shopping merchant ordertracking duplicate create, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"AlreadyExists"`) {
		t.Fatalf("expected AlreadyExists error in response")
	}
}

func TestGCPShoppingMerchantOrdertrackingRouter_CreateMissingAccountNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantOrdertrackingContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/ordertracking/v1/accounts/missing/orderTrackingSignals", []byte(`{
		"orderTrackingSignal":{
			"orderCreatedTime":"2026-01-02T03:04:05Z",
			"orderId":"ORDER-1001",
			"shippingInfo":[{
				"shipmentId":"SHIP-1001",
				"shippingStatus":"SHIPPED",
				"originPostalCode":"94043",
				"originRegionCode":"US",
				"trackingId":"TRACK-1001",
				"carrier":"UPS"
			}],
			"lineItems":[{
				"lineItemId":"line-1",
				"productId":"online:en:US:offer-1001",
				"quantity":"2"
			}]
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-ordertracking",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp shopping merchant ordertracking missing account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound error in response")
	}
}

func TestGCPShoppingMerchantOrdertrackingRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantOrdertrackingContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/ordertracking/v1/accounts/123456/orderTrackingSignals", []byte(`{
		"orderTrackingSignal":{
			"orderCreatedTime":"2026-01-02T03:04:05Z",
			"orderId":"ORDER-1001",
			"shippingInfo":[{
				"shipmentId":"SHIP-1001",
				"shippingStatus":"SHIPPED",
				"originPostalCode":"94043",
				"originRegionCode":"US",
				"trackingId":"TRACK-1001",
				"carrier":"UPS"
			}],
			"lineItems":[{
				"lineItemId":"line-1",
				"productId":"online:en:US:offer-1001",
				"quantity":"2",
				"gtins":["00012345678905"]
			}],
			"shipmentLineItemMapping":[{
				"shipmentId":"SHIP-1001",
				"lineItemId":"line-1",
				"quantity":"2"
			}],
			"customerShippingFee":{"currencyCode":"USD","amountMicros":"1299000"},
			"deliveryPostalCode":"94043",
			"deliveryRegionCode":"US"
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-ordertracking",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant ordertracking typed create, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if _, ok := body["orderTrackingSignalId"].(string); !ok {
		t.Fatalf("expected orderTrackingSignalId string, got %#v", body["orderTrackingSignalId"])
	}
	if _, ok := body["merchantId"].(string); !ok {
		t.Fatalf("expected merchantId string, got %#v", body["merchantId"])
	}
	if _, ok := body["orderId"].(string); !ok {
		t.Fatalf("expected orderId string, got %#v", body["orderId"])
	}
	orderCreatedTime, ok := body["orderCreatedTime"].(map[string]any)
	if !ok {
		t.Fatalf("expected orderCreatedTime object, got %#v", body["orderCreatedTime"])
	}
	if _, ok := orderCreatedTime["year"].(float64); !ok {
		t.Fatalf("expected orderCreatedTime.year number, got %#v", orderCreatedTime["year"])
	}

	shippingInfo, ok := body["shippingInfo"].([]any)
	if !ok || len(shippingInfo) == 0 {
		t.Fatalf("expected shippingInfo array, got %#v", body["shippingInfo"])
	}
	firstShipment, ok := shippingInfo[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first shipment object, got %#v", shippingInfo[0])
	}
	if _, ok := firstShipment["shipmentId"].(string); !ok {
		t.Fatalf("expected shipmentId string, got %#v", firstShipment["shipmentId"])
	}
	if _, ok := firstShipment["shippingStatus"].(float64); !ok {
		t.Fatalf("expected shippingStatus number, got %#v", firstShipment["shippingStatus"])
	}

	lineItems, ok := body["lineItems"].([]any)
	if !ok || len(lineItems) == 0 {
		t.Fatalf("expected lineItems array, got %#v", body["lineItems"])
	}
	firstLineItem, ok := lineItems[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first line item object, got %#v", lineItems[0])
	}
	if _, ok := firstLineItem["lineItemId"].(string); !ok {
		t.Fatalf("expected lineItemId string, got %#v", firstLineItem["lineItemId"])
	}
	if _, ok := firstLineItem["quantity"].(string); !ok {
		t.Fatalf("expected quantity string, got %#v", firstLineItem["quantity"])
	}

	if fee, ok := body["customerShippingFee"].(map[string]any); !ok {
		t.Fatalf("expected customerShippingFee object, got %#v", body["customerShippingFee"])
	} else {
		if _, ok := fee["currencyCode"].(string); !ok {
			t.Fatalf("expected customerShippingFee.currencyCode string, got %#v", fee["currencyCode"])
		}
		if _, ok := fee["amountMicros"].(string); !ok {
			t.Fatalf("expected customerShippingFee.amountMicros string, got %#v", fee["amountMicros"])
		}
	}
}

func TestGCPShoppingMerchantOrdertrackingRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/shopping_merchant_ordertracking/sample?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant ordertracking contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shopping_merchant_ordertracking" {
		t.Fatalf("expected service=shopping_merchant_ordertracking, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShoppingMerchantOrdertrackingContractServer(t *testing.T) *httptest.Server {
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

func assertGCPShoppingMerchantOrdertrackingSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-ordertracking",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant ordertracking router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if expectBodyFragment != "" && !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("expected response body for %s %s to contain %q, got %s", method, path, expectBodyFragment, string(providerContractBody(t, resp)))
	}
}
