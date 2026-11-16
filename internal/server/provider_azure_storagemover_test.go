package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureStorageMoverTestAPIVersion = "?api-version=2024-07-01"

func TestAzureStorageMoverLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-move")
	mover := sub + "/resourceGroups/rg-move/providers/Microsoft.StorageMover/storageMovers/migration"
	ids := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-move/providers"

	azureResourcesExpectStatus(t, do(http.MethodPut, mover+azureStorageMoverTestAPIVersion, `{"location":"eastus","properties":{"description":"NAS migration"}}`), http.StatusCreated)
	resp := do(http.MethodPut, mover+"/agents/agent1"+azureStorageMoverTestAPIVersion, `{"properties":{"arcResourceId":"`+ids+`/Microsoft.HybridCompute/machines/agent1","arcVmUuid":"3bb2c024-eba9-4d18-9e7a-1d772fcc5fe9"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["agentStatus"]; got != "Online" {
		t.Fatalf("expected an online agent, got %v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, mover+"/endpoints/nas"+azureStorageMoverTestAPIVersion, `{"properties":{"endpointType":"NfsMount","host":"10.0.0.1","export":"/data","nfsVersion":"NFSv4"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, mover+"/endpoints/blob"+azureStorageMoverTestAPIVersion, `{"properties":{"endpointType":"AzureStorageBlobContainer","storageAccountResourceId":"`+ids+`/Microsoft.Storage/storageAccounts/target","blobContainerName":"data"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, mover+"/endpoints/smb"+azureStorageMoverTestAPIVersion, `{"properties":{"endpointType":"SmbMount","host":"fs01","shareName":"data","credentials":{"type":"AzureKeyVaultSmb","usernameUri":"https://vault.vault.azure.net/secrets/user","passwordUri":"https://vault.vault.azure.net/secrets/password"}}}`), http.StatusCreated)

	project := mover + "/projects/wave1"
	azureResourcesExpectStatus(t, do(http.MethodPut, project+azureStorageMoverTestAPIVersion, `{"properties":{"description":"First wave"}}`), http.StatusCreated)
	job := project + "/jobDefinitions/data"
	resp = do(http.MethodPut, job+azureStorageMoverTestAPIVersion, `{"properties":{"copyMode":"Additive","sourceName":"nas","targetName":"blob","agentName":"agent1"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if !strings.HasSuffix(props["sourceResourceId"].(string), "/storageMovers/migration/endpoints/nas") || !strings.HasSuffix(props["agentResourceId"].(string), "/storageMovers/migration/agents/agent1") {
		t.Fatalf("unexpected job definition properties: %#v", props)
	}
	resp = do(http.MethodPost, job+"/startJob"+azureStorageMoverTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["jobRunResourceId"].(string); !strings.Contains(got, "/jobDefinitions/data/jobRuns/") {
		t.Fatalf("unexpected job run ID: %s", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, job+"/stopJob"+azureStorageMoverTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, project+"/jobDefinitions/noagent"+azureStorageMoverTestAPIVersion, `{"properties":{"copyMode":"Mirror","sourceName":"nas","targetName":"blob"}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, project+"/jobDefinitions/noagent/startJob"+azureStorageMoverTestAPIVersion, ""), http.StatusBadRequest, "BadRequest")
	azureResourcesExpectStatus(t, do(http.MethodPost, project+"/jobDefinitions/noagent/stopJob"+azureStorageMoverTestAPIVersion, ""), http.StatusOK)

	azureResourcesExpectStatus(t, do(http.MethodDelete, mover+azureStorageMoverTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, job+azureStorageMoverTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureStorageMoverValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-move")
	mover := sub + "/resourceGroups/rg-move/providers/Microsoft.StorageMover/storageMovers/migration"
	azureResourcesExpectStatus(t, do(http.MethodPut, mover+azureStorageMoverTestAPIVersion, `{"location":"eastus","properties":{}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, mover+"/projects/p"+azureStorageMoverTestAPIVersion, `{"properties":{}}`), http.StatusCreated)
	cases := []struct{ name, path, body string }{
		{"agent arc", "/agents/a", `{"properties":{"arcResourceId":"agent","arcVmUuid":"3bb2c024-eba9-4d18-9e7a-1d772fcc5fe9"}}`},
		{"agent uuid", "/agents/a", `{"properties":{"arcResourceId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.HybridCompute/machines/a","arcVmUuid":"vm"}}`},
		{"endpoint type", "/endpoints/e", `{"properties":{"endpointType":"S3"}}`},
		{"endpoint export", "/endpoints/e", `{"properties":{"endpointType":"NfsMount","host":"10.0.0.1"}}`},
		{"endpoint nfs version", "/endpoints/e", `{"properties":{"endpointType":"NfsMount","host":"10.0.0.1","export":"/data","nfsVersion":"NFSv2"}}`},
		{"endpoint account", "/endpoints/e", `{"properties":{"endpointType":"AzureStorageSmbFileShare","storageAccountResourceId":"target","fileShareName":"data"}}`},
		{"endpoint secret", "/endpoints/e", `{"properties":{"endpointType":"SmbMount","host":"fs01","shareName":"data","credentials":{"usernameUri":"user","passwordUri":"password"}}}`},
		{"job copy mode", "/projects/p/jobDefinitions/j", `{"properties":{"copyMode":"Sync","sourceName":"a","targetName":"b"}}`},
		{"job target", "/projects/p/jobDefinitions/j", `{"properties":{"copyMode":"Mirror","sourceName":"a"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, mover+tc.path+azureStorageMoverTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
