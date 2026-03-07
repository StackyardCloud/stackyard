package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShoppingMerchantConversionsRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantConversionsContractServer(t)
	parent := "accounts/123456"
	collection := "/gcp/conversions/v1/" + parent + "/conversionSources"
	mcdName := parent + "/conversionSources/mcdn:1001"
	gaName := parent + "/conversionSources/galk:2001"

	assertGCPShoppingMerchantConversionsSuccess(t, ts, http.MethodPost, collection, []byte(`{
		"merchantCenterDestination":{
			"displayName":"Primary Destination",
			"currencyCode":"USD",
			"attributionSettings":{
				"attributionLookbackWindowDays":30,
				"attributionModel":"CROSS_CHANNEL_LAST_CLICK"
			}
		}
	}`), "mcdn:1001")

	assertGCPShoppingMerchantConversionsSuccess(t, ts, http.MethodPost, collection, []byte(`{
		"googleAnalyticsLink":{"propertyId":"2001"}
	}`), "galk:2001")

	assertGCPShoppingMerchantConversionsSuccess(t, ts, http.MethodGet, collection+"?pageSize=1", nil, "conversionSources")
	assertGCPShoppingMerchantConversionsSuccess(t, ts, http.MethodGet, "/gcp/conversions/v1/"+mcdName, nil, mcdName)
	assertGCPShoppingMerchantConversionsSuccess(t, ts, http.MethodPatch, "/gcp/conversions/v1/"+mcdName+"?updateMask=merchantCenterDestination.displayName", []byte(`{
		"name":"accounts/123456/conversionSources/mcdn:1001",
		"merchantCenterDestination":{"displayName":"Primary Destination Updated"}
	}`), "Primary Destination Updated")
	assertGCPShoppingMerchantConversionsSuccess(t, ts, http.MethodDelete, "/gcp/conversions/v1/"+mcdName, nil, "{}")
	assertGCPShoppingMerchantConversionsSuccess(t, ts, http.MethodPost, "/gcp/conversions/v1/"+mcdName+":undelete", []byte(`{
		"name":"accounts/123456/conversionSources/mcdn:1001"
	}`), `"state":"ACTIVE"`)
	assertGCPShoppingMerchantConversionsSuccess(t, ts, http.MethodDelete, "/gcp/conversions/v1/"+gaName, nil, "{}")
}

func TestGCPShoppingMerchantConversionsRouter_ListRejectsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantConversionsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/conversions/v1/accounts/123456/conversionSources?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-conversions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant conversions list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantConversionsRouter_CreateRequiresSingleSourceType(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantConversionsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/conversions/v1/accounts/123456/conversionSources", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-conversions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant conversions create, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantConversionsRouter_UpdateRequiresUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantConversionsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/conversions/v1/accounts/123456/conversionSources/mcdn:1001", []byte(`{
		"name":"accounts/123456/conversionSources/mcdn:1001",
		"merchantCenterDestination":{"displayName":"Primary Destination Updated"}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-conversions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant conversions update, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantConversionsRouter_UndeleteGALinkFailsPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantConversionsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/conversions/v1/accounts/123456/conversionSources/galk:2001:undelete", []byte(`{
		"name":"accounts/123456/conversionSources/galk:2001"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-conversions",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant conversions undelete, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"FailedPrecondition"`) {
		t.Fatalf("expected FailedPrecondition error in response")
	}
}

func TestGCPShoppingMerchantConversionsRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantConversionsContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-conversions",
	}

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/conversions/v1/accounts/123456/conversionSources?pageSize=2", nil, headers)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant conversions list, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	sources, ok := listBody["conversionSources"].([]any)
	if !ok || len(sources) == 0 {
		t.Fatalf("expected conversionSources array, got %#v", listBody["conversionSources"])
	}
	first, ok := sources[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first conversion source object, got %#v", sources[0])
	}
	if _, ok := first["name"].(string); !ok {
		t.Fatalf("expected conversion source name string, got %#v", first["name"])
	}
	if _, ok := first["state"].(string); !ok {
		t.Fatalf("expected conversion source state string, got %#v", first["state"])
	}
	if _, ok := first["merchantCenterDestination"].(map[string]any); !ok {
		t.Fatalf("expected merchantCenterDestination object, got %#v", first["merchantCenterDestination"])
	}

	getResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/conversions/v1/accounts/123456/conversionSources/galk:2001", nil, headers)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant conversions get, got %d body=%s", getResp.StatusCode, string(providerContractBody(t, getResp)))
	}
	getBody := providerContractJSONMap(t, getResp)
	gaLink, ok := getBody["googleAnalyticsLink"].(map[string]any)
	if !ok {
		t.Fatalf("expected googleAnalyticsLink object, got %#v", getBody["googleAnalyticsLink"])
	}
	if _, ok := gaLink["propertyId"].(string); !ok {
		t.Fatalf("expected googleAnalyticsLink.propertyId string, got %#v", gaLink["propertyId"])
	}
}

func TestGCPShoppingMerchantConversionsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/shopping_merchant_conversions/sample?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant conversions contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shopping_merchant_conversions" {
		t.Fatalf("expected service=shopping_merchant_conversions, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShoppingMerchantConversionsContractServer(t *testing.T) *httptest.Server {
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

func assertGCPShoppingMerchantConversionsSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-conversions",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant conversions router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if expectBodyFragment != "" && !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("expected response body for %s %s to contain %q, got %s", method, path, expectBodyFragment, string(providerContractBody(t, resp)))
	}
}
