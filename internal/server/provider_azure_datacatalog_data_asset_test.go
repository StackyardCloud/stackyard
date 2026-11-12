package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestAzureDataCatalogDataAssetLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	registerPath := "/azure/catalogs/DefaultCatalog/views/tables?api-version=2016-03-30"
	registerBody := []byte(`{
		"roles":[
			{
				"role":"Contributor",
				"members":[{"objectId":"00000000-0000-0000-0000-000000000201"}]
			}
		],
		"properties":{
			"fromSourceSystem":true,
			"name":"Orders",
			"dataSource":{"sourceType":"SQL Server","objectType":"Table"},
			"dsl":{
				"protocol":"tds",
				"authentication":"windows",
				"address":{
					"server":"MyServer.contoso.com",
					"database":"NORTHWND",
					"schema":"dbo",
					"object":"Orders"
				}
			}
		},
		"annotations":{
			"schema":{
				"properties":{
					"fromSourceSystem":true,
					"columns":[{"name":"OrderID","type":"int","isNullable":false}]
				}
			}
		}
	}`)

	resp := providerContractRequest(t, ts, http.MethodPost, registerPath, registerBody, headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for register create, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	registerPayload := providerContractJSONMap(t, resp)
	assetID := strings.TrimSpace(providerContractString(registerPayload["id"]))
	if assetID == "" {
		t.Fatalf("register response did not include id: %#v", registerPayload)
	}
	location := strings.TrimSpace(resp.Header.Get("Location"))
	if location == "" || location != assetID {
		t.Fatalf("expected Location header to equal id, location=%q id=%q", location, assetID)
	}

	assetPathPrefix := "/azure/catalogs/DefaultCatalog/views/tables/"
	assetPath := strings.Replace(assetID, "https://stackyard.local/catalogs/DefaultCatalog/views/tables/", assetPathPrefix, 1) + "?api-version=2016-03-30"

	getHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Accept":                    "application/json;adc.metadata=full",
	}

	resp = providerContractRequest(t, ts, http.MethodGet, assetPath, nil, getHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	getPayload := providerContractJSONMap(t, resp)
	if providerContractString(getPayload["id"]) != assetID {
		t.Fatalf("get returned unexpected id: %#v", getPayload["id"])
	}
	assetETag := strings.TrimSpace(providerContractString(getPayload["etag"]))
	if assetETag == "" {
		t.Fatalf("get response did not include etag: %#v", getPayload)
	}
	annotations, ok := getPayload["annotations"].(map[string]any)
	if !ok {
		t.Fatalf("expected annotations object, got %#v", getPayload["annotations"])
	}
	schemaAnnotation, ok := annotations["schema"].(map[string]any)
	if !ok || providerContractString(schemaAnnotation["id"]) == "" {
		t.Fatalf("expected schema annotation in get response: %#v", annotations)
	}

	expertsPath := strings.TrimSuffix(assetPath, "?api-version=2016-03-30") + "/experts?api-version=2016-03-30"
	expertBody := []byte(`{
		"properties":{
			"fromSourceSystem":false,
			"key":"22c3fa019b3945dc97143ebc3ad74cbf--1111fa019b3945dc97143ebc3ad74cbf",
			"expert":{"upn":"expert1@contoso.com","objectId":"1111fa019b3945dc97143ebc3ad74cbf"}
		}
	}`)

	resp = providerContractRequest(t, ts, http.MethodPost, expertsPath, expertBody, headers)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 for annotate create, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	expertLocation := strings.TrimSpace(resp.Header.Get("Location"))
	if expertLocation == "" {
		t.Fatalf("expected Location header on annotate create")
	}

	resp = providerContractRequest(t, ts, http.MethodPost, expertsPath, expertBody, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for annotate update, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, assetPath, nil, getHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get after annotate, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	getPayload = providerContractJSONMap(t, resp)
	annotations, _ = getPayload["annotations"].(map[string]any)
	experts, ok := annotations["experts"].([]any)
	if !ok || len(experts) != 1 {
		t.Fatalf("expected one experts annotation, got %#v", annotations["experts"])
	}
	expertItem, ok := experts[0].(map[string]any)
	if !ok {
		t.Fatalf("expected expert item object, got %#v", experts[0])
	}
	expertETag := strings.TrimSpace(providerContractString(expertItem["etag"]))
	if expertETag == "" {
		t.Fatalf("expected expert etag, got %#v", expertItem)
	}
	expertID := strings.TrimSpace(providerContractString(expertItem["id"]))
	if expertID == "" {
		t.Fatalf("expected expert id, got %#v", expertItem)
	}

	updateExpertPath := strings.Replace(expertID, "https://stackyard.local/catalogs/DefaultCatalog/views/tables/", assetPathPrefix, 1) + "?api-version=2016-03-30"

	resp = providerContractRequest(t, ts, http.MethodPut, updateExpertPath, []byte(`{
		"etag":"stale-value",
		"properties":{
			"fromSourceSystem":false,
			"key":"22c3fa019b3945dc97143ebc3ad74cbf--1111fa019b3945dc97143ebc3ad74cbf",
			"expert":{"upn":"expert1@contoso.com","objectId":"1111fa019b3945dc97143ebc3ad74cbf"}
		}
	}`), headers)
	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected 412 for update annotation with stale etag, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPut, updateExpertPath, []byte(`{
		"etag":"`+expertETag+`",
		"properties":{
			"fromSourceSystem":false,
			"key":"22c3fa019b3945dc97143ebc3ad74cbf--1111fa019b3945dc97143ebc3ad74cbf",
			"expert":{"upn":"expert2@contoso.com","objectId":"1111fa019b3945dc97143ebc3ad74cbf"}
		}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for update annotation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, updateExpertPath, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"If-Match":                  `W/"mismatch"`,
	})
	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected 412 for delete annotation with stale if-match, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, assetPath, nil, getHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get before delete annotation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	getPayload = providerContractJSONMap(t, resp)
	annotations, _ = getPayload["annotations"].(map[string]any)
	experts, _ = annotations["experts"].([]any)
	expertItem, _ = experts[0].(map[string]any)
	expertETag = strings.TrimSpace(providerContractString(expertItem["etag"]))

	resp = providerContractRequest(t, ts, http.MethodDelete, updateExpertPath, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"If-Match":                  `W/"` + expertETag + `"`,
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete annotation, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	searchPath := "/azure/catalogs/DefaultCatalog/search/search?searchTerms=Orders&count=10&startPage=1&api-version=2016-03-30"
	resp = providerContractRequest(t, ts, http.MethodGet, searchPath, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for search, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	searchPayload := providerContractJSONMap(t, resp)
	query, ok := searchPayload["query"].(map[string]any)
	if !ok {
		t.Fatalf("expected query object in search response, got %#v", searchPayload)
	}
	if providerContractFloat64(query["totalResults"]) < 1 {
		t.Fatalf("expected search totalResults >= 1, got %#v", query["totalResults"])
	}

	resp = providerContractRequest(t, ts, http.MethodGet, assetPath, nil, getHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get before delete asset, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	getPayload = providerContractJSONMap(t, resp)
	assetETag = strings.TrimSpace(providerContractString(getPayload["etag"]))

	resp = providerContractRequest(t, ts, http.MethodDelete, assetPath, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"If-Match":                  `W/"mismatch"`,
	})
	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected 412 for delete asset with stale if-match, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, assetPath, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"If-Match":                  `W/"` + assetETag + `"`,
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete asset, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, assetPath, nil, map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	})
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete-if-exists, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureDataCatalogDataAssetValidation(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		headers    map[string]string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "missing api version",
			method:     http.MethodPost,
			path:       "/azure/catalogs/DefaultCatalog/views/tables",
			body:       []byte(`{"properties":{"name":"Orders"}}`),
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BadRequest",
		},
		{
			name:       "unsupported method on search",
			method:     http.MethodPost,
			path:       "/azure/catalogs/DefaultCatalog/search/search?searchTerms=Orders&api-version=2016-03-30",
			headers:    headers,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "search missing search terms parameter",
			method:     http.MethodGet,
			path:       "/azure/catalogs/DefaultCatalog/search/search?api-version=2016-03-30",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BadRequest",
		},
		{
			name:       "search invalid start page",
			method:     http.MethodGet,
			path:       "/azure/catalogs/DefaultCatalog/search/search?searchTerms=Orders&startPage=0&api-version=2016-03-30",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BadRequest",
		},
		{
			name:       "search invalid count",
			method:     http.MethodGet,
			path:       "/azure/catalogs/DefaultCatalog/search/search?searchTerms=Orders&count=101&api-version=2016-03-30",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BadRequest",
		},
		{
			name:       "search invalid view",
			method:     http.MethodGet,
			path:       "/azure/catalogs/DefaultCatalog/search/search?searchTerms=Orders&view=Tables&api-version=2016-03-30",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BadRequest",
		},
		{
			name:   "invalid adc metadata mode",
			method: http.MethodGet,
			path:   "/azure/catalogs/DefaultCatalog/views/tables/abc?api-version=2016-03-30",
			headers: map[string]string{
				"Authorization":             "SharedKey devstoreaccount1:signature",
				"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
				"Accept":                    "application/json;adc.metadata=invalid",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "BadRequest",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, tt.headers)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d for %s, got %d body=%s", tt.wantStatus, tt.name, resp.StatusCode, string(providerContractBody(t, resp)))
			}
			if tt.wantCode == "" {
				return
			}
			payload := providerContractJSONMap(t, resp)
			errorObject, ok := payload["error"].(map[string]any)
			if !ok || errorObject["code"] != tt.wantCode {
				t.Fatalf("expected error code %q, got %#v", tt.wantCode, payload)
			}
		})
	}
}

func providerContractString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func providerContractFloat64(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	default:
		return 0
	}
}
