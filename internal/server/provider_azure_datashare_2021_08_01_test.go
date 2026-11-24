package server

import (
	"net/http"
	"testing"
)

const azureDSHRTestAPI = "?api-version=2021-08-01"

func TestAzureDataShare20210801ProviderConsumerFlow(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-ds")
	accounts := sub + "/resourceGroups/rg-ds/providers/Microsoft.DataShare/accounts/"
	provider, consumer := accounts+"provider", accounts+"consumer"
	for _, account := range []string{provider, consumer} {
		azureResourcesExpectStatus(t, do(http.MethodPut, account+azureDSHRTestAPI, `{"location":"eastus","identity":{"type":"SystemAssigned"}}`), http.StatusCreated)
	}
	share := provider + "/shares/s1"
	azureResourcesExpectStatus(t, do(http.MethodPut, share+azureDSHRTestAPI, `{"properties":{"shareKind":"CopyBased","terms":"t"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, share+"/dataSets/d1"+azureDSHRTestAPI, `{"kind":"Blob","properties":{"resourceGroup":"rg","storageAccountName":"sa","subscriptionId":"x","containerName":"c","filePath":"f.csv"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, share+"/synchronizationSettings/sync"+azureDSHRTestAPI, `{"kind":"ScheduleBased","properties":{"recurrenceInterval":"Day","synchronizationTime":"2026-01-01T00:00:00Z"}}`), http.StatusCreated)
	invitation := providerContractJSONMap(t, do(http.MethodPut, share+"/invitations/i1"+azureDSHRTestAPI, `{"properties":{"targetEmail":"c@example.com"}}`))
	invitationID := invitation["properties"].(map[string]any)["invitationId"].(string)

	pending := providerContractJSONMap(t, do(http.MethodGet, "/azure/providers/Microsoft.DataShare/listInvitations"+azureDSHRTestAPI, ""))["value"].([]any)
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending invitation, got %d", len(pending))
	}
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/providers/Microsoft.DataShare/locations/eastus/consumerInvitations/"+invitationID+azureDSHRTestAPI, ""), http.StatusOK)

	subscription := consumer + "/shareSubscriptions/ss1"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, subscription+azureDSHRTestAPI, `{"properties":{"invitationId":"nope","sourceShareLocation":"eastus"}}`), http.StatusBadRequest, "InvalidRequestContent")
	resp := do(http.MethodPut, subscription+azureDSHRTestAPI, `{"properties":{"invitationId":"`+invitationID+`","sourceShareLocation":"eastus"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	objectID := providerContractJSONMap(t, resp)["properties"].(map[string]any)["shareSubscriptionObjectId"].(string)
	if status := providerContractJSONMap(t, do(http.MethodGet, share+"/invitations/i1"+azureDSHRTestAPI, ""))["properties"].(map[string]any)["invitationStatus"]; status != "Accepted" {
		t.Fatalf("invitation status = %v, want Accepted", status)
	}

	sources := providerContractJSONMap(t, do(http.MethodGet, subscription+"/consumerSourceDataSets"+azureDSHRTestAPI, ""))["value"].([]any)
	if len(sources) != 1 {
		t.Fatalf("expected 1 consumer source data set, got %d", len(sources))
	}
	dataSetID := sources[0].(map[string]any)["properties"].(map[string]any)["dataSetId"].(string)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, subscription+"/dataSetMappings/m1"+azureDSHRTestAPI, `{"kind":"Blob","properties":{"dataSetId":"other"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, subscription+"/dataSetMappings/m1"+azureDSHRTestAPI, `{"kind":"Blob","properties":{"dataSetId":"`+dataSetID+`"}}`), http.StatusCreated)
	settings := providerContractJSONMap(t, do(http.MethodPost, subscription+"/listSourceShareSynchronizationSettings"+azureDSHRTestAPI, ""))["value"].([]any)
	if len(settings) != 1 {
		t.Fatalf("expected the source share's synchronization setting, got %d", len(settings))
	}

	sync := providerContractJSONMap(t, do(http.MethodPost, subscription+"/synchronize"+azureDSHRTestAPI, `{"synchronizationMode":"Incremental"}`))
	syncID := sync["synchronizationId"].(string)
	details := providerContractJSONMap(t, do(http.MethodPost, subscription+"/listSynchronizationDetails"+azureDSHRTestAPI, `{"synchronizationId":"`+syncID+`"}`))["value"].([]any)
	if len(details) != 1 {
		t.Fatalf("expected 1 synchronization detail, got %d", len(details))
	}
	if n := len(providerContractJSONMap(t, do(http.MethodPost, share+"/listSynchronizations"+azureDSHRTestAPI, ""))["value"].([]any)); n != 1 {
		t.Fatalf("provider should see 1 synchronization, got %d", n)
	}

	pss := share + "/providerShareSubscriptions/" + objectID
	azureResourcesExpectStatus(t, do(http.MethodGet, pss+azureDSHRTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, pss+"/reinstate"+azureDSHRTestAPI, ""), http.StatusConflict)
	azureResourcesExpectStatus(t, do(http.MethodPost, pss+"/revoke"+azureDSHRTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, subscription+"/synchronize"+azureDSHRTestAPI, `{"synchronizationMode":"FullSync"}`), http.StatusConflict)
	azureResourcesExpectStatus(t, do(http.MethodPost, pss+"/reinstate"+azureDSHRTestAPI, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, share+"/providerShareSubscriptions/missing/revoke"+azureDSHRTestAPI, ""), http.StatusNotFound)
}

func TestAzureDataShare20210801Validation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-ds")
	account := sub + "/resourceGroups/rg-ds/providers/Microsoft.DataShare/accounts/a1"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, account+azureDSHRTestAPI, `{"location":"eastus"}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, account+azureDSHRTestAPI, `{"location":"eastus","identity":{"type":"SystemAssigned"}}`), http.StatusCreated)
	share := account + "/shares/s1"
	azureResourcesExpectStatus(t, do(http.MethodPut, share+azureDSHRTestAPI, `{"properties":{"shareKind":"CopyBased"}}`), http.StatusCreated)
	for _, tc := range []struct{ name, path, body string }{
		{"share kind", share, `{"properties":{"shareKind":"Magic"}}`},
		{"invitation target", share + "/invitations/i", `{"properties":{}}`},
		{"invitation email", share + "/invitations/i", `{"properties":{"targetEmail":"nope"}}`},
		{"data set kind", share + "/dataSets/d", `{"kind":"Floppy","properties":{}}`},
		{"blob data set fields", share + "/dataSets/d", `{"kind":"Blob","properties":{"containerName":"c"}}`},
		{"schedule interval", share + "/synchronizationSettings/s", `{"kind":"ScheduleBased","properties":{"recurrenceInterval":"Week","synchronizationTime":"2026-01-01T00:00:00Z"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, do(http.MethodPut, tc.path+azureDSHRTestAPI, tc.body), http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, "/azure/providers/Microsoft.DataShare/locations/eastus/rejectInvitation"+azureDSHRTestAPI, `{"properties":{"invitationId":"missing"}}`), http.StatusNotFound)
	azureResourcesExpectStatus(t, do(http.MethodPost, "/azure/providers/Microsoft.DataShare/locations/eastus/registerEmail"+azureDSHRTestAPI, ""), http.StatusOK)
}
