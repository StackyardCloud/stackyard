package server

import (
	"net/http"
	"testing"
)

func TestAzureDataCatalogCustomDataSourceLifecycle(t *testing.T) {
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

	path := "/azure/catalogs/DefaultCatalog/dataSourceProtocols?api-version=2016-03-30"

	resp := providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for enumerate, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	payload := providerContractJSONMap(t, resp)
	items, ok := payload["value"].([]any)
	if !ok || len(items) != 0 {
		t.Fatalf("expected empty value array, got %#v", payload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, path, []byte(`{
		"namespace":"Test.MyTestProtocols",
		"name":"abc-def",
		"identityProperties":[
			{"name":"prop1","type":"string","ignoreCase":true},
			{"name":"prop2","type":"int"}
		],
		"identitySets":[
			{"properties":["prop1"]},
			{"properties":["prop1","prop2"]}
		]
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for register, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	location := resp.Header.Get("Location")
	if location == "" {
		t.Fatalf("expected Location header on register")
	}
	created := providerContractJSONMap(t, resp)
	if created["namespace"] != "Test.MyTestProtocols" || created["name"] != "abc-def" {
		t.Fatalf("unexpected register response payload: %#v", created)
	}
	if created["id"] != location {
		t.Fatalf("expected id to match Location header, id=%#v location=%q", created["id"], location)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, path, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for enumerate after register, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	afterCreate := providerContractJSONMap(t, resp)
	items, ok = afterCreate["value"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one registered protocol, got %#v", afterCreate)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, path, []byte(`{
		"namespace":"Test.MyTestProtocols",
		"name":"abc-def",
		"identityProperties":[{"name":"prop1","type":"string"}],
		"identitySets":[{"properties":["prop1"]}]
	}`), headers)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for duplicate register, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	duplicate := providerContractJSONMap(t, resp)
	errorObject, ok := duplicate["error"].(map[string]any)
	if !ok || errorObject["code"] != "ImmutableViewItem" {
		t.Fatalf("expected ImmutableViewItem error, got %#v", duplicate)
	}
}

func TestAzureDataCatalogCustomDataSourceValidation(t *testing.T) {
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
		wantStatus int
		wantCode   string
	}{
		{
			name:       "missing api version",
			method:     http.MethodGet,
			path:       "/azure/catalogs/DefaultCatalog/dataSourceProtocols",
			wantStatus: http.StatusBadRequest,
			wantCode:   "BadRequest",
		},
		{
			name:       "invalid api version",
			method:     http.MethodGet,
			path:       "/azure/catalogs/DefaultCatalog/dataSourceProtocols?api-version=bad",
			wantStatus: http.StatusBadRequest,
			wantCode:   "BadRequest",
		},
		{
			name:       "unsupported method",
			method:     http.MethodPut,
			path:       "/azure/catalogs/DefaultCatalog/dataSourceProtocols?api-version=2016-03-30",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "missing namespace",
			method: http.MethodPost,
			path:   "/azure/catalogs/DefaultCatalog/dataSourceProtocols?api-version=2016-03-30",
			body: []byte(`{
				"name":"abc-def",
				"identityProperties":[{"name":"prop1","type":"string"}],
				"identitySets":[{"properties":["prop1"]}]
			}`),
			wantStatus: http.StatusBadRequest,
			wantCode:   "BadRequest",
		},
		{
			name:   "identity set unknown property",
			method: http.MethodPost,
			path:   "/azure/catalogs/DefaultCatalog/dataSourceProtocols?api-version=2016-03-30",
			body: []byte(`{
				"namespace":"Test.MyTestProtocols",
				"name":"abc-def",
				"identityProperties":[{"name":"prop1","type":"string"}],
				"identitySets":[{"properties":["prop2"]}]
			}`),
			wantStatus: http.StatusBadRequest,
			wantCode:   "BadRequest",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
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
