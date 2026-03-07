package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPShoppingMerchantDatasourcesRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantDatasourcesContractServer(t)
	parent := "accounts/123456"
	collection := "/gcp/datasources/v1/" + parent + "/dataSources"
	name := parent + "/dataSources/1001"

	assertGCPShoppingMerchantDatasourcesSuccess(t, ts, http.MethodPost, collection, []byte(`{
		"displayName":"Stackyard Created Data Source",
		"fileInput":{"fileName":"products.csv"},
		"primaryProductDataSource":{
			"feedLabel":"US",
			"contentLanguage":"en",
			"countries":["US"]
		}
	}`), "dataSources/1001")
	assertGCPShoppingMerchantDatasourcesSuccess(t, ts, http.MethodGet, collection+"?pageSize=1", nil, "dataSources")
	assertGCPShoppingMerchantDatasourcesSuccess(t, ts, http.MethodGet, "/gcp/datasources/v1/"+name, nil, name)
	assertGCPShoppingMerchantDatasourcesSuccess(t, ts, http.MethodPatch, "/gcp/datasources/v1/"+name+"?updateMask=display_name", []byte(`{
		"name":"accounts/123456/dataSources/1001",
		"displayName":"Stackyard Data Source Updated"
	}`), "Stackyard Data Source 1001")
	assertGCPShoppingMerchantDatasourcesSuccess(t, ts, http.MethodPost, "/gcp/datasources/v1/"+name+":fetch", []byte(`{
		"name":"accounts/123456/dataSources/1001"
	}`), "{}")
	assertGCPShoppingMerchantDatasourcesSuccess(t, ts, http.MethodGet, "/gcp/datasources/v1/"+name+"/fileUploads/latest", nil, "processingState")
	assertGCPShoppingMerchantDatasourcesSuccess(t, ts, http.MethodDelete, "/gcp/datasources/v1/"+name, nil, "{}")
}

func TestGCPShoppingMerchantDatasourcesRouter_ListRejectsInvalidPageSize(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantDatasourcesContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/datasources/v1/accounts/123456/dataSources?pageSize=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-datasources",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant datasources list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantDatasourcesRouter_CreateRequiresSingleType(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantDatasourcesContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/datasources/v1/accounts/123456/dataSources", []byte(`{
		"displayName":"Stackyard Created Data Source"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-datasources",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant datasources create, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantDatasourcesRouter_UpdateRequiresUpdateMask(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantDatasourcesContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/datasources/v1/accounts/123456/dataSources/1001", []byte(`{
		"name":"accounts/123456/dataSources/1001",
		"displayName":"Stackyard Data Source Updated"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-datasources",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant datasources update, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"InvalidArgument"`) {
		t.Fatalf("expected InvalidArgument error in response")
	}
}

func TestGCPShoppingMerchantDatasourcesRouter_FetchNoFileFailsPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantDatasourcesContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/datasources/v1/accounts/123456/dataSources/nofile:fetch", []byte(`{
		"name":"accounts/123456/dataSources/nofile"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "shopping-merchant-datasources",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp shopping merchant datasources fetch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if !strings.Contains(string(providerContractBody(t, resp)), `"error":"FailedPrecondition"`) {
		t.Fatalf("expected FailedPrecondition error in response")
	}
}

func TestGCPShoppingMerchantDatasourcesRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPShoppingMerchantDatasourcesContractServer(t)
	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-datasources",
	}

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/datasources/v1/accounts/123456/dataSources?pageSize=1", nil, headers)
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant datasources list, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	sources, ok := listBody["dataSources"].([]any)
	if !ok || len(sources) == 0 {
		t.Fatalf("expected dataSources array, got %#v", listBody["dataSources"])
	}
	first, ok := sources[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first data source object, got %#v", sources[0])
	}
	if _, ok := first["name"].(string); !ok {
		t.Fatalf("expected data source name string, got %#v", first["name"])
	}
	if _, ok := first["displayName"].(string); !ok {
		t.Fatalf("expected data source displayName string, got %#v", first["displayName"])
	}
	if _, ok := first["fileInput"].(map[string]any); !ok {
		t.Fatalf("expected data source fileInput object, got %#v", first["fileInput"])
	}
	if _, ok := first["primaryProductDataSource"].(map[string]any); !ok {
		t.Fatalf("expected data source primaryProductDataSource object, got %#v", first["primaryProductDataSource"])
	}

	fileResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/datasources/v1/accounts/123456/dataSources/1001/fileUploads/latest", nil, headers)
	if fileResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant datasources get file upload, got %d body=%s", fileResp.StatusCode, string(providerContractBody(t, fileResp)))
	}
	fileBody := providerContractJSONMap(t, fileResp)
	if _, ok := fileBody["name"].(string); !ok {
		t.Fatalf("expected file upload name string, got %#v", fileBody["name"])
	}
	if _, ok := fileBody["processingState"].(string); !ok {
		t.Fatalf("expected file upload processingState string, got %#v", fileBody["processingState"])
	}
	issues, ok := fileBody["issues"].([]any)
	if !ok || len(issues) == 0 {
		t.Fatalf("expected file upload issues array, got %#v", fileBody["issues"])
	}
	if _, ok := fileBody["itemsTotal"].(string); !ok {
		t.Fatalf("expected file upload itemsTotal string, got %#v", fileBody["itemsTotal"])
	}
}

func TestGCPShoppingMerchantDatasourcesRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/shopping_merchant_datasources/sample?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant datasources contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "shopping_merchant_datasources" {
		t.Fatalf("expected service=shopping_merchant_datasources, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected typed name field in contract probe response, got %#v", body["name"])
	}
}

func newGCPShoppingMerchantDatasourcesContractServer(t *testing.T) *httptest.Server {
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

func assertGCPShoppingMerchantDatasourcesSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "shopping-merchant-datasources",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp shopping merchant datasources router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if expectBodyFragment != "" && !strings.Contains(string(providerContractBody(t, resp)), expectBodyFragment) {
		t.Fatalf("expected response body for %s %s to contain %q, got %s", method, path, expectBodyFragment, string(providerContractBody(t, resp)))
	}
}
