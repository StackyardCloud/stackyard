package server

import (
	"net/http"
	"testing"
)

func TestAzureVideoIndexerLifecycle(t *testing.T) {
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

	subscriptionID := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "rg-videoindexer"
	accountName := "video-account-a"
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	accountBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.VideoIndexer/accounts/" + accountName

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.VideoIndexer/operations?api-version="+azureVideoIndexerAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected non-empty operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, subscriptionBase+"/providers/Microsoft.VideoIndexer/checkNameAvailability?api-version="+azureVideoIndexerAPIVersion, []byte(`{
		"name":"video-account-a",
		"type":"Microsoft.VideoIndexer/accounts"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 checking name availability, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	nameAvailability := providerContractJSONMap(t, resp)
	if nameAvailability["nameAvailable"] != true {
		t.Fatalf("expected available name, got %#v", nameAvailability)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, accountBase+"?api-version="+azureVideoIndexerAPIVersion, []byte(`{
		"location":"eastus",
		"identity":{"type":"SystemAssigned"},
		"tags":{"env":"dev"},
		"properties":{"description":"Local Video Indexer account"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, accountBase+"?api-version="+azureVideoIndexerAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading account, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"?api-version="+azureVideoIndexerAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.VideoIndexer/accounts?api-version="+azureVideoIndexerAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing accounts by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, accountBase+"?api-version="+azureVideoIndexerAPIVersion, []byte(`{
		"tags":{"env":"stage"},
		"properties":{"description":"Updated description"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	tokenTests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "generate access token",
			path: accountBase + "/generateAccessToken?api-version=" + azureVideoIndexerAPIVersion,
			body: `{"permissionType":"Reader","scope":"Account","tokenLifetimeInSeconds":120}`,
		},
		{
			name: "generate restricted viewer access token",
			path: accountBase + "/generateRestrictedViewerAccessToken?api-version=" + azureVideoIndexerAPIVersion,
			body: `{"scope":"Video","videoId":"video-1","tokenLifetimeInSeconds":300}`,
		},
		{
			name: "generate extension access token",
			path: accountBase + "/generateExtensionAccessToken?api-version=" + azureVideoIndexerAPIVersion,
			body: `{"permissionType":"Contributor","scope":"Project","projectId":"project-1","extensionId":"ext-1","tokenLifetimeInSeconds":600}`,
		},
		{
			name: "generate extension restricted viewer access token",
			path: accountBase + "/generateExtensionRestrictedViewerAccessToken?api-version=" + azureVideoIndexerAPIVersion,
			body: `{"scope":"Account","extensionId":"ext-1","tokenLifetimeInSeconds":900}`,
		},
	}
	for _, tt := range tokenTests {
		resp = providerContractRequest(t, ts, http.MethodPost, tt.path, []byte(tt.body), headers)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d body=%s", tt.name, resp.StatusCode, string(providerContractBody(t, resp)))
		}
		payload := providerContractJSONMap(t, resp)
		if payload["accessToken"] == "" {
			t.Fatalf("expected accessToken for %s, got %#v", tt.name, payload)
		}
	}

	privateEndpointBase := accountBase + "/privateEndpointConnections/pec-a"
	resp = providerContractRequest(t, ts, http.MethodPut, privateEndpointBase+"?api-version="+azureVideoIndexerAPIVersion, []byte(`{
		"properties":{
			"privateEndpoint":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-network/providers/Microsoft.Network/privateEndpoints/pe-a"},
			"groupIds":["avi"],
			"privateLinkServiceConnectionState":{"status":"Approved","description":"Approved by test","actionsRequired":"None"}
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating private endpoint connection, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/privateEndpointConnections?api-version="+azureVideoIndexerAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing private endpoint connections, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, privateEndpointBase+"?api-version="+azureVideoIndexerAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting private endpoint connection, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/privateLinkResources?api-version="+azureVideoIndexerAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing private link resources, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/privateLinkResources/avi?api-version="+azureVideoIndexerAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting private link resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureVideoIndexerValidationAndNotFound(t *testing.T) {
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

	subscriptionID := "00000000-0000-0000-0000-000000000000"
	resourceGroup := "rg-videoindexer"
	base := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.VideoIndexer/accounts"

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
	}{
		{
			name:       "invalid account name",
			method:     http.MethodPut,
			path:       base + "/bad_name?api-version=" + azureVideoIndexerAPIVersion,
			body:       []byte(`{"location":"eastus"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing location",
			method:     http.MethodPut,
			path:       base + "/good-name?api-version=" + azureVideoIndexerAPIVersion,
			body:       []byte(`{"properties":{"description":"missing location"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid check name type",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.VideoIndexer/checkNameAvailability?api-version=" + azureVideoIndexerAPIVersion,
			body:       []byte(`{"name":"good-name","type":"wrong/type"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing account",
			method:     http.MethodGet,
			path:       base + "/missing?api-version=" + azureVideoIndexerAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing permission type for access token",
			method:     http.MethodPost,
			path:       base + "/seed-account/generateAccessToken?api-version=" + azureVideoIndexerAPIVersion,
			body:       []byte(`{"scope":"Account"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "extension restricted viewer missing extension id",
			method:     http.MethodPost,
			path:       base + "/seed-account/generateExtensionRestrictedViewerAccessToken?api-version=" + azureVideoIndexerAPIVersion,
			body:       []byte(`{"scope":"Account"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid lifetime",
			method:     http.MethodPost,
			path:       base + "/seed-account/generateRestrictedViewerAccessToken?api-version=" + azureVideoIndexerAPIVersion,
			body:       []byte(`{"scope":"Account","tokenLifetimeInSeconds":50000}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing private endpoint connection",
			method:     http.MethodGet,
			path:       base + "/seed-account/privateEndpointConnections/missing?api-version=" + azureVideoIndexerAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing private link resource",
			method:     http.MethodGet,
			path:       base + "/seed-account/privateLinkResources/missing?api-version=" + azureVideoIndexerAPIVersion,
			wantStatus: http.StatusNotFound,
		},
	}

	// Seed account for token and child-resource validation cases.
	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/subscriptions/"+subscriptionID+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.VideoIndexer/accounts?api-version="+azureVideoIndexerAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 seeding account list, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, headers)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(providerContractBody(t, resp)))
			}
		})
	}
}
