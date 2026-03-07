package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPRecommendationengineRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPRecommendationengineContractServer(t)

	catalogParent := "/gcp/v1beta1/projects/stackyard/locations/global/catalogs/default_catalog"
	eventStoreParent := catalogParent + "/eventStores/default_event_store"
	itemPath := catalogParent + "/catalogItems/item-1"
	registrationPath := eventStoreParent + "/predictionApiKeyRegistrations/stackyard-api-key"

	assertGCPRecommendationengineSuccess(t, ts, http.MethodPost, catalogParent+"/catalogItems", []byte(`{"id":"item-1","title":"Stackyard Item 1"}`), "\"id\":\"item-1\"")
	assertGCPRecommendationengineSuccess(t, ts, http.MethodGet, catalogParent+"/catalogItems?pageSize=1", nil, "catalogItems")
	assertGCPRecommendationengineSuccess(t, ts, http.MethodGet, itemPath, nil, "\"id\":\"item-1\"")
	assertGCPRecommendationengineSuccess(t, ts, http.MethodPatch, itemPath+"?updateMask=title", []byte(`{"id":"item-1","title":"Stackyard Item Updated"}`), "Updated")
	assertGCPRecommendationengineSuccess(t, ts, http.MethodDelete, itemPath, nil, "{}")
	assertGCPRecommendationengineSuccess(t, ts, http.MethodPost, catalogParent+"/catalogItems:import", []byte(`{"inputConfig":{"catalogInlineSource":{"catalogItems":[{"id":"item-1","title":"Stackyard Item 1"}]}}}`), "operations/importCatalogItems")

	assertGCPRecommendationengineSuccess(t, ts, http.MethodPost, eventStoreParent+"/userEvents:write", []byte(`{"eventType":"detail-page-view","userInfo":{"visitorId":"visitor-1"}}`), "detail-page-view")
	assertGCPRecommendationengineSuccess(t, ts, http.MethodGet, eventStoreParent+"/userEvents:collect?userEvent=test-event", nil, "\"status\":\"ok\"")
	assertGCPRecommendationengineSuccess(t, ts, http.MethodGet, eventStoreParent+"/userEvents?pageSize=1", nil, "userEvents")
	assertGCPRecommendationengineSuccess(t, ts, http.MethodPost, eventStoreParent+"/userEvents:purge", []byte(`{"filter":"eventType = detail-page-view"}`), "operations/purgeUserEvents")
	assertGCPRecommendationengineSuccess(t, ts, http.MethodPost, eventStoreParent+"/userEvents:import", []byte(`{"inputConfig":{"userEventInlineSource":{"userEvents":[{"eventType":"detail-page-view","userInfo":{"visitorId":"visitor-1"}}]}}}`), "operations/importUserEvents")

	assertGCPRecommendationengineSuccess(t, ts, http.MethodPost, eventStoreParent+"/placements/home_page:predict", []byte(`{"userEvent":{"eventType":"detail-page-view","userInfo":{"visitorId":"visitor-1"}},"pageSize":1}`), "recommendationToken")

	assertGCPRecommendationengineSuccess(t, ts, http.MethodPost, eventStoreParent+"/predictionApiKeyRegistrations", []byte(`{"predictionApiKeyRegistration":{"apiKey":"stackyard-api-key"}}`), "stackyard-api-key")
	assertGCPRecommendationengineSuccess(t, ts, http.MethodGet, eventStoreParent+"/predictionApiKeyRegistrations?pageSize=1", nil, "predictionApiKeyRegistrations")
	assertGCPRecommendationengineSuccess(t, ts, http.MethodDelete, registrationPath, nil, "{}")

	assertGCPRecommendationengineSuccess(t, ts, http.MethodGet, catalogParent+"/operations/importCatalogItems-1", nil, "operations/importCatalogItems-1")
}

func TestGCPRecommendationengineRouter_ListCatalogItemsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPRecommendationengineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1beta1/projects/stackyard/locations/global/catalogs/default_catalog/catalogItems?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "recommendationengine",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp recommendationengine router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRecommendationengineRouter_CreateCatalogItemRequiresIDAndTitle(t *testing.T) {
	t.Parallel()

	ts := newGCPRecommendationengineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/catalogs/default_catalog/catalogItems", []byte(`{"id":""}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "recommendationengine",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp recommendationengine router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRecommendationengineRouter_UpdateCatalogItemIDMustMatchPath(t *testing.T) {
	t.Parallel()

	ts := newGCPRecommendationengineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1beta1/projects/stackyard/locations/global/catalogs/default_catalog/catalogItems/item-1?updateMask=title", []byte(`{"id":"item-2","title":"bad"}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "recommendationengine",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp recommendationengine router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRecommendationengineRouter_WriteUserEventRequiresVisitorID(t *testing.T) {
	t.Parallel()

	ts := newGCPRecommendationengineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/catalogs/default_catalog/eventStores/default_event_store/userEvents:write", []byte(`{"eventType":"detail-page-view","userInfo":{}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "recommendationengine",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp recommendationengine router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRecommendationengineRouter_PredictRequiresUserEvent(t *testing.T) {
	t.Parallel()

	ts := newGCPRecommendationengineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/catalogs/default_catalog/eventStores/default_event_store/placements/home_page:predict", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "recommendationengine",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp recommendationengine router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRecommendationengineRouter_CreatePredictionAPIKeyRequiresAPIKey(t *testing.T) {
	t.Parallel()

	ts := newGCPRecommendationengineContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1beta1/projects/stackyard/locations/global/catalogs/default_catalog/eventStores/default_event_store/predictionApiKeyRegistrations", []byte(`{"predictionApiKeyRegistration":{}}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "recommendationengine",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp recommendationengine router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPRecommendationengineRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/recommendationengine?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp recommendationengine contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "recommendationengine" {
		t.Fatalf("expected service=recommendationengine, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPRecommendationengineContractServer(t *testing.T) *httptest.Server {
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

func assertGCPRecommendationengineSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "recommendationengine",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp recommendationengine router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
