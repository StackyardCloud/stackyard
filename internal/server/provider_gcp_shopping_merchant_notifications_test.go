package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShoppingMerchantNotificationsRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantNotificationsContractServer(t)
	parent := "accounts/123456"
	resource := parent + "/notificationsubscriptions/all-managed-product-status-change"

	assertGCPShoppingMerchantNotificationsSuccess(t, ts, http.MethodPost, "/gcp/notifications/v1/"+parent+"/notificationsubscriptions", []byte(`{
		"registeredEvent":"PRODUCT_STATUS_CHANGE",
		"callBackUri":"https://example.com/hooks/merchant-notifications",
		"allManagedAccounts":true
	}`), "/notificationsubscriptions/all-managed-product-status-change")
	assertGCPShoppingMerchantNotificationsSuccess(t, ts, http.MethodGet, "/gcp/notifications/v1/"+resource, nil, resource)
	assertGCPShoppingMerchantNotificationsSuccess(t, ts, http.MethodGet, "/gcp/notifications/v1/"+parent+"/notificationsubscriptions?pageSize=1", nil, `"notificationSubscriptions"`)
	assertGCPShoppingMerchantNotificationsSuccess(t, ts, http.MethodPatch, "/gcp/notifications/v1/"+resource+"?updateMask=callBackUri", []byte(`{
		"name":"accounts/123456/notificationsubscriptions/all-managed-product-status-change",
		"registeredEvent":"PRODUCT_STATUS_CHANGE",
		"callBackUri":"https://example.com/hooks/merchant-notifications-updated",
		"allManagedAccounts":true
	}`), "merchant-notifications-updated")
	assertGCPShoppingMerchantNotificationsSuccess(t, ts, http.MethodGet, "/gcp/notifications/v1/"+resource+":getHealth", nil, `"acknowledgedMessagesCount"`)
	assertGCPShoppingMerchantNotificationsSuccess(t, ts, http.MethodDelete, "/gcp/notifications/v1/"+resource, nil, "{}")
}

func TestGCPShoppingMerchantNotificationsRouter_ListRejectsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantNotificationsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/notifications/v1/accounts/123456/notificationsubscriptions?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-notifications",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant notifications list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantNotificationsRouter_CreateRejectsInvalidInterestedIn(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantNotificationsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/notifications/v1/accounts/123456/notificationsubscriptions", []byte(`{
		"registeredEvent":"PRODUCT_STATUS_CHANGE",
		"callBackUri":"https://example.com/hooks/merchant-notifications"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-notifications",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant notifications create, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantNotificationsRouter_CreateRejectsInvalidCallbackURI(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantNotificationsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/notifications/v1/accounts/123456/notificationsubscriptions", []byte(`{
		"registeredEvent":"PRODUCT_STATUS_CHANGE",
		"callBackUri":"notaurl",
		"allManagedAccounts":true
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-notifications",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant notifications create callback validation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantNotificationsRouter_UpdateRequiresMask(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantNotificationsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/notifications/v1/accounts/123456/notificationsubscriptions/all-managed-product-status-change", []byte(`{
		"name":"accounts/123456/notificationsubscriptions/all-managed-product-status-change",
		"registeredEvent":"PRODUCT_STATUS_CHANGE",
		"callBackUri":"https://example.com/hooks/merchant-notifications-updated",
		"allManagedAccounts":true
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-notifications",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant notifications update, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantNotificationsRouter_UpdateRejectsUnsupportedMask(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantNotificationsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/notifications/v1/accounts/123456/notificationsubscriptions/all-managed-product-status-change?updateMask=name", []byte(`{
		"name":"accounts/123456/notificationsubscriptions/all-managed-product-status-change",
		"registeredEvent":"PRODUCT_STATUS_CHANGE",
		"callBackUri":"https://example.com/hooks/merchant-notifications-updated",
		"allManagedAccounts":true
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-notifications",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant notifications update unsupported mask, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"FailedPrecondition"`) {
		t.Fatalf("expected FailedPrecondition error in response")
	}
}

func TestGCPShoppingMerchantNotificationsRouter_GetMissingNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantNotificationsContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/notifications/v1/accounts/123456/notificationsubscriptions/missing-sub", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-notifications",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp shopping merchant notifications get, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"NotFound"`) {
		t.Fatalf("expected NotFound error in response")
	}
}

func TestGCPShoppingMerchantNotificationsRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantNotificationsContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-notifications",
	}

	createResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/notifications/v1/accounts/123456/notificationsubscriptions", []byte(`{
		"registeredEvent":"PRODUCT_STATUS_CHANGE",
		"callBackUri":"https://example.com/hooks/merchant-notifications",
		"allManagedAccounts":true
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-notifications",
	})
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant notifications create, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	createBody := providerContractJSONMap(t, createResp)
	if _, ok := createBody["name"].(string); !ok {
		t.Fatalf("expected subscription name string, got %#v", createBody["name"])
	}
	if _, ok := createBody["registeredEvent"].(float64); !ok {
		t.Fatalf("expected registeredEvent numeric, got %#v", createBody["registeredEvent"])
	}
	if _, ok := createBody["callBackUri"].(string); !ok {
		t.Fatalf("expected callBackUri string, got %#v", createBody["callBackUri"])
	}
	if _, ok := createBody["allManagedAccounts"].(bool); !ok {
		t.Fatalf("expected allManagedAccounts bool, got %#v", createBody["allManagedAccounts"])
	}

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/notifications/v1/accounts/123456/notificationsubscriptions?pageSize=1", nil, headers)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant notifications list, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	items, ok := listBody["notificationSubscriptions"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected notificationSubscriptions array, got %#v", listBody["notificationSubscriptions"])
	}
	firstItem, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first notification subscription object, got %#v", items[0])
	}
	if _, ok := firstItem["name"].(string); !ok {
		t.Fatalf("expected listed subscription name string, got %#v", firstItem["name"])
	}

	healthResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/notifications/v1/accounts/123456/notificationsubscriptions/all-managed-product-status-change:getHealth", nil, headers)
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant notifications health, got %d body=%s", healthResp.StatusCode, string(providerContractBody(t, healthResp)))
	}
	healthBody := providerContractJSONMap(t, healthResp)
	if _, ok := healthBody["name"].(string); !ok {
		t.Fatalf("expected health name string, got %#v", healthBody["name"])
	}
	if _, ok := healthBody["acknowledgedMessagesCount"].(string); !ok {
		t.Fatalf("expected acknowledgedMessagesCount string, got %#v", healthBody["acknowledgedMessagesCount"])
	}
	if _, ok := healthBody["undeliveredMessagesCount"].(string); !ok {
		t.Fatalf("expected undeliveredMessagesCount string, got %#v", healthBody["undeliveredMessagesCount"])
	}
	if _, ok := healthBody["oldestUnacknowledgedMessageWaitingTime"].(string); !ok {
		t.Fatalf("expected oldestUnacknowledgedMessageWaitingTime string, got %#v", healthBody["oldestUnacknowledgedMessageWaitingTime"])
	}
}

func TestGCPShoppingMerchantNotificationsRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/shopping_merchant_notifications/sample?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant notifications contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shopping_merchant_notifications" {
		t.Fatalf("expected service=shopping_merchant_notifications, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShoppingMerchantNotificationsContractServer(t *testing.T) *httptest.Server {
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

func assertGCPShoppingMerchantNotificationsSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-notifications",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant notifications router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if expectBodyFragment != "" && !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("expected response body for %s %s to contain %q, got %s", method, path, expectBodyFragment, string(providerContractBody(t, resp)))
	}
}
