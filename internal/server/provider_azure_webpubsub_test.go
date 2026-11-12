package server

import (
	"net/http"
	"testing"
)

func TestAzureWebPubSubLifecycle(t *testing.T) {
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
	resourceGroup := "rg-webpubsub"
	location := "eastus"
	accountName := "webpubsub-a"
	subscriptionBase := "/azure/subscriptions/" + subscriptionID
	accountBase := subscriptionBase + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.SignalRService/webPubSub/" + accountName

	resp := providerContractRequest(t, ts, http.MethodGet, "/azure/providers/Microsoft.SignalRService/operations?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing operations, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	operations := providerContractJSONMap(t, resp)
	if values, ok := operations["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected non-empty operations payload, got %#v", operations)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/providers/Microsoft.SignalRService/locations/"+location+"/usages?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing usages, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	usages := providerContractJSONMap(t, resp)
	if values, ok := usages["value"].([]any); !ok || len(values) == 0 {
		t.Fatalf("expected non-empty usages payload, got %#v", usages)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, subscriptionBase+"/providers/Microsoft.SignalRService/locations/"+location+"/checkNameAvailability?api-version="+azureWebPubSubAPIVersion, []byte(`{
		"name":"webpubsub-a",
		"type":"Microsoft.SignalRService/webPubSub"
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 checking name availability, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	availability := providerContractJSONMap(t, resp)
	if availability["nameAvailable"] != true {
		t.Fatalf("expected name available, got %#v", availability)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, accountBase+"?api-version="+azureWebPubSubAPIVersion, []byte(`{
		"location":"eastus",
		"identity":{"type":"SystemAssigned"},
		"tags":{"env":"dev"},
		"properties":{"publicNetworkAccess":"Enabled"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodHead, accountBase+"?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 heading account, got %d", resp.StatusCode)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	account := providerContractJSONMap(t, resp)
	if account["location"] != location {
		t.Fatalf("expected eastus account location, got %#v", account)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/resourceGroups/"+resourceGroup+"/providers/Microsoft.SignalRService/webPubSub?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing by resource group, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, subscriptionBase+"/providers/Microsoft.SignalRService/webPubSub?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing by subscription, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, accountBase+"/listKeys?api-version="+azureWebPubSubAPIVersion, []byte(`{}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing keys, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	keys := providerContractJSONMap(t, resp)
	if keys["primaryKey"] == "" || keys["secondaryKey"] == "" {
		t.Fatalf("expected primaryKey and secondaryKey, got %#v", keys)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, accountBase+"/regenerateKey?api-version="+azureWebPubSubAPIVersion, []byte(`{"keyType":"Primary"}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 regenerating key, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/skus?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing skus, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	customCertificateBase := accountBase + "/customCertificates/cert-a"
	resp = providerContractRequest(t, ts, http.MethodPut, customCertificateBase+"?api-version="+azureWebPubSubAPIVersion, []byte(`{
		"properties":{"keyVaultBaseUri":"https://stackyard.vault.azure.net/"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating custom certificate, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/customCertificates?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing custom certificates, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, customCertificateBase+"?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting custom certificate, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	customDomainBase := accountBase + "/customDomains/domain-a"
	resp = providerContractRequest(t, ts, http.MethodPut, customDomainBase+"?api-version="+azureWebPubSubAPIVersion, []byte(`{
		"properties":{"domainName":"domain-a.example.com"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating custom domain, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/customDomains?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing custom domains, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	hubBase := accountBase + "/hubs/hub-a"
	resp = providerContractRequest(t, ts, http.MethodPut, hubBase+"?api-version="+azureWebPubSubAPIVersion, []byte(`{
		"properties":{"anonymousConnectPolicy":"deny"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating hub, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPatch, hubBase+"?api-version="+azureWebPubSubAPIVersion, []byte(`{
		"properties":{"anonymousConnectPolicy":"allow"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching hub, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/hubs?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing hubs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	privateEndpointBase := accountBase + "/privateEndpointConnections/pec-a"
	resp = providerContractRequest(t, ts, http.MethodPut, privateEndpointBase+"?api-version="+azureWebPubSubAPIVersion, []byte(`{
		"properties":{
			"privateEndpoint":{"id":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-network/providers/Microsoft.Network/privateEndpoints/pe-a"},
			"groupIds":["webpubsub"]
		}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating private endpoint connection, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/privateEndpointConnections?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing private endpoint connections, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/privateLinkResources?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing private link resources, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/privateLinkResources/webpubsub?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 getting private link resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	sharedPrivateLinkBase := accountBase + "/sharedPrivateLinkResources/splr-a"
	resp = providerContractRequest(t, ts, http.MethodPut, sharedPrivateLinkBase+"?api-version="+azureWebPubSubAPIVersion, []byte(`{
		"properties":{"groupId":"blob","privateLinkResourceId":"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-storage/providers/Microsoft.Storage/storageAccounts/sa1"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating shared private link resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPatch, sharedPrivateLinkBase+"?api-version="+azureWebPubSubAPIVersion, []byte(`{
		"properties":{"status":"Approved"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching shared private link resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/sharedPrivateLinkResources?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing shared private link resources, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	replicaBase := accountBase + "/replicas/replica-a"
	resp = providerContractRequest(t, ts, http.MethodPut, replicaBase+"?api-version="+azureWebPubSubAPIVersion, []byte(`{
		"location":"westus2",
		"properties":{"regionEndpointEnabled":true}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating replica, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPatch, replicaBase+"?api-version="+azureWebPubSubAPIVersion, []byte(`{
		"properties":{"regionEndpointType":"Secondary"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching replica, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, accountBase+"/replicas?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing replicas, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, replicaBase+"/skus?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing replica skus, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	replicaSharedPrivateLinkBase := replicaBase + "/sharedPrivateLinkResources/splr-replica"
	resp = providerContractRequest(t, ts, http.MethodPut, replicaSharedPrivateLinkBase+"?api-version="+azureWebPubSubAPIVersion, []byte(`{
		"properties":{"groupId":"sql"}
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 creating replica shared private link resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodPatch, replicaSharedPrivateLinkBase+"?api-version="+azureWebPubSubAPIVersion, []byte(`{
		"properties":{"status":"Approved"}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching replica shared private link resource, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	resp = providerContractRequest(t, ts, http.MethodGet, replicaBase+"/sharedPrivateLinkResources?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 listing replica shared private link resources, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, accountBase+"?api-version="+azureWebPubSubAPIVersion, []byte(`{
		"tags":{"env":"stage"},
		"properties":{"resourceStopped":true}
	}`), headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 patching account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	for _, path := range []string{
		replicaSharedPrivateLinkBase + "?api-version=" + azureWebPubSubAPIVersion,
		replicaBase + "?api-version=" + azureWebPubSubAPIVersion,
		sharedPrivateLinkBase + "?api-version=" + azureWebPubSubAPIVersion,
		privateEndpointBase + "?api-version=" + azureWebPubSubAPIVersion,
		hubBase + "?api-version=" + azureWebPubSubAPIVersion,
		customDomainBase + "?api-version=" + azureWebPubSubAPIVersion,
		customCertificateBase + "?api-version=" + azureWebPubSubAPIVersion,
	} {
		resp = providerContractRequest(t, ts, http.MethodDelete, path, nil, headers)
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("expected 204 deleting child resource %s, got %d body=%s", path, resp.StatusCode, string(providerContractBody(t, resp)))
		}
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, accountBase+"?api-version="+azureWebPubSubAPIVersion, nil, headers)
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected 202 deleting account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureWebPubSubValidationAndNotFound(t *testing.T) {
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
	resourceGroup := "rg-webpubsub"
	base := "/azure/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.SignalRService/webPubSub"

	// Seed an account for child-resource validation.
	resp := providerContractRequest(t, ts, http.MethodPut, base+"/seed-webpubsub?api-version="+azureWebPubSubAPIVersion, []byte(`{
		"location":"eastus"
	}`), headers)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 201/200 seeding account, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

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
			path:       base + "/1bad?api-version=" + azureWebPubSubAPIVersion,
			body:       []byte(`{"location":"eastus"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing location on create",
			method:     http.MethodPut,
			path:       base + "/good-name?api-version=" + azureWebPubSubAPIVersion,
			body:       []byte(`{"tags":{"env":"dev"}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid name availability type",
			method:     http.MethodPost,
			path:       "/azure/subscriptions/" + subscriptionID + "/providers/Microsoft.SignalRService/locations/eastus/checkNameAvailability?api-version=" + azureWebPubSubAPIVersion,
			body:       []byte(`{"name":"good-name","type":"wrong/type"}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing account",
			method:     http.MethodGet,
			path:       base + "/missing?api-version=" + azureWebPubSubAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing key type",
			method:     http.MethodPost,
			path:       base + "/seed-webpubsub/regenerateKey?api-version=" + azureWebPubSubAPIVersion,
			body:       []byte(`{}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing private endpoint id",
			method:     http.MethodPut,
			path:       base + "/seed-webpubsub/privateEndpointConnections/pec-b?api-version=" + azureWebPubSubAPIVersion,
			body:       []byte(`{"properties":{"groupIds":["webpubsub"]}}`),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing private link resource",
			method:     http.MethodGet,
			path:       base + "/seed-webpubsub/privateLinkResources/missing?api-version=" + azureWebPubSubAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "missing replica",
			method:     http.MethodGet,
			path:       base + "/seed-webpubsub/replicas/missing?api-version=" + azureWebPubSubAPIVersion,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "empty child patch",
			method:     http.MethodPatch,
			path:       base + "/seed-webpubsub/hubs/missing-hub?api-version=" + azureWebPubSubAPIVersion,
			body:       []byte(`{}`),
			wantStatus: http.StatusBadRequest,
		},
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
