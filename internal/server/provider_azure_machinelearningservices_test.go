package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureMLTestAPIVersion = "?api-version=2024-10-01"

func azureMLWorkspaceTestBody(sub string) string {
	group := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-ml/providers"
	return `{"location":"eastus","identity":{"type":"SystemAssigned"},"properties":{"storageAccount":"` + group + `/Microsoft.Storage/storageAccounts/mlstore","keyVault":"` + group + `/Microsoft.KeyVault/vaults/mlvault","applicationInsights":"` + group + `/Microsoft.Insights/components/mlinsights"}}`
}

func TestAzureMLLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-ml")
	workspace := sub + "/resourceGroups/rg-ml/providers/Microsoft.MachineLearningServices/workspaces/research"

	resp := do(http.MethodPut, workspace+azureMLTestAPIVersion, azureMLWorkspaceTestBody(sub))
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["discoveryUrl"] != "https://eastus.api.azureml.ms/discovery" || props["publicNetworkAccess"] != "Enabled" || props["workspaceId"] == nil ||
		!strings.HasSuffix(props["mlFlowTrackingUri"].(string), "/workspaces/research") {
		t.Fatalf("unexpected workspace properties: %#v", props)
	}
	resp = do(http.MethodPost, workspace+"/listKeys"+azureMLTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if keys := providerContractJSONMap(t, resp); keys["userStorageKey"] == "" || !strings.HasSuffix(keys["userStorageArmId"].(string), "/storageAccounts/mlstore") {
		t.Fatalf("unexpected workspace keys: %#v", keys)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-ml/providers/Microsoft.MachineLearningServices/workspaces/research-project"+azureMLTestAPIVersion, `{"location":"eastus","kind":"Project","properties":{"hubResourceId":"`+strings.TrimPrefix(workspace, "/azure")+`"}}`), http.StatusCreated)

	cluster := workspace + "/computes/cpu-cluster"
	resp = do(http.MethodPut, cluster+azureMLTestAPIVersion, `{"location":"eastus","properties":{"computeType":"AmlCompute","properties":{"scaleSettings":{"minNodeCount":0,"maxNodeCount":4},"userAccountCredentials":{"adminUserName":"azureuser","adminUserPassword":"Secret-123"}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props = providerContractJSONMap(t, resp)["properties"].(map[string]any)
	inner := props["properties"].(map[string]any)
	if props["isAttachedCompute"] != false || props["computeLocation"] != "eastus" || inner["vmSize"] != "STANDARD_DS3_V2" || inner["currentNodeCount"] != float64(0) ||
		inner["userAccountCredentials"].(map[string]any)["adminUserPassword"] != nil {
		t.Fatalf("unexpected compute properties: %#v", props)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, cluster+"/stop"+azureMLTestAPIVersion, ""), http.StatusBadRequest, "BadRequest")
	instance := workspace + "/computes/notebook-1"
	resp = do(http.MethodPut, instance+azureMLTestAPIVersion, `{"location":"eastus","properties":{"computeType":"ComputeInstance","properties":{"vmSize":"STANDARD_DS11_V2"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if inner := providerContractJSONMap(t, resp)["properties"].(map[string]any)["properties"].(map[string]any); inner["state"] != "Running" || inner["vmSize"] != "STANDARD_DS11_V2" {
		t.Fatalf("unexpected compute instance properties: %#v", inner)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, instance+"/stop"+azureMLTestAPIVersion, ""), http.StatusAccepted)

	resp = do(http.MethodPut, workspace+"/datastores/training"+azureMLTestAPIVersion, `{"properties":{"datastoreType":"AzureBlob","accountName":"mlstore","containerName":"training","credentials":{"credentialsType":"AccountKey","secrets":{"secretsType":"AccountKey","key":"c2VjcmV0"}}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props = providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["endpoint"] != "core.windows.net" || props["isDefault"] != false || props["credentials"].(map[string]any)["secrets"] != nil {
		t.Fatalf("unexpected datastore properties: %#v", props)
	}

	endpoint := workspace + "/onlineEndpoints/Scorer"
	resp = do(http.MethodPut, endpoint+azureMLTestAPIVersion, `{"location":"eastus","identity":{"type":"SystemAssigned"},"properties":{"authMode":"Key"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["scoringUri"]; got != "https://scorer.eastus.inference.ml.azure.com/score" {
		t.Fatalf("unexpected scoring URI: %v", got)
	}
	resp = do(http.MethodPost, endpoint+"/listKeys"+azureMLTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if keys := providerContractJSONMap(t, resp); keys["primaryKey"] == keys["secondaryKey"] {
		t.Fatalf("expected distinct keys: %#v", keys)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, endpoint+"/regenerateKeys"+azureMLTestAPIVersion, `{"keyType":"Primary"}`), http.StatusAccepted)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, endpoint+"/token"+azureMLTestAPIVersion, ""), http.StatusBadRequest, "BadRequest")
	deployment := endpoint + "/deployments/blue"
	azureResourcesExpectStatus(t, do(http.MethodPut, deployment+azureMLTestAPIVersion, `{"location":"eastus","sku":{"name":"Default","capacity":1},"properties":{"endpointComputeType":"Managed","model":"azureml:sklearn-model:1","instanceType":"Standard_DS3_v2"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPost, deployment+"/getLogs"+azureMLTestAPIVersion, `{"tail":100}`), http.StatusOK)

	azureResourcesExpectStatus(t, do(http.MethodDelete, workspace+azureMLTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, deployment+azureMLTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureMLValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-ml")
	workspace := sub + "/resourceGroups/rg-ml/providers/Microsoft.MachineLearningServices/workspaces/research"
	azureResourcesExpectStatus(t, do(http.MethodPut, workspace+azureMLTestAPIVersion, azureMLWorkspaceTestBody(sub)), http.StatusCreated)
	endpoint := workspace + "/onlineEndpoints/scorer"
	azureResourcesExpectStatus(t, do(http.MethodPut, endpoint+azureMLTestAPIVersion, `{"location":"eastus","properties":{"authMode":"AMLToken"}}`), http.StatusCreated)
	resp := do(http.MethodPost, endpoint+"/token"+azureMLTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if token := providerContractJSONMap(t, resp); token["tokenType"] != "Bearer" || token["accessToken"] == "" {
		t.Fatalf("unexpected token: %#v", token)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, endpoint+"/listKeys"+azureMLTestAPIVersion, ""), http.StatusBadRequest, "BadRequest")

	cases := []struct{ name, path, body string }{
		{"workspace kind", workspace, `{"location":"eastus","kind":"Lab","properties":{}}`},
		{"workspace storage", workspace, `{"location":"eastus","properties":{"keyVault":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/v"}}`},
		{"project hub", workspace + "-p", `{"location":"eastus","kind":"Project","properties":{}}`},
		{"compute type", workspace + "/computes/gpu", `{"location":"eastus","properties":{"computeType":"Quantum"}}`},
		{"compute nodes", workspace + "/computes/gpu", `{"location":"eastus","properties":{"computeType":"AmlCompute","properties":{"scaleSettings":{"minNodeCount":5,"maxNodeCount":2}}}}`},
		{"attached resource", workspace + "/computes/aks", `{"location":"eastus","properties":{"computeType":"AKS"}}`},
		{"datastore type", workspace + "/datastores/d1", `{"properties":{"datastoreType":"S3","credentials":{"credentialsType":"None"}}}`},
		{"datastore container", workspace + "/datastores/d1", `{"properties":{"datastoreType":"AzureBlob","accountName":"mlstore","credentials":{"credentialsType":"None"}}}`},
		{"datastore credentials", workspace + "/datastores/d1", `{"properties":{"datastoreType":"AzureDataLakeGen1","storeName":"lake","credentials":{"credentialsType":"Password"}}}`},
		{"endpoint auth", workspace + "/onlineEndpoints/ep-1", `{"location":"eastus","properties":{"authMode":"Basic"}}`},
		{"workspace registry", workspace, `{"location":"eastus","properties":{"storageAccount":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/s","keyVault":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/v","containerRegistry":"registry"}}`},
		{"workspace network", workspace, `{"location":"eastus","properties":{"storageAccount":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Storage/storageAccounts/s","keyVault":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.KeyVault/vaults/v","publicNetworkAccess":"Open"}}`},
		{"compute max nodes", workspace + "/computes/gpu", `{"location":"eastus","properties":{"computeType":"AmlCompute","properties":{"scaleSettings":{}}}}`},
		{"datastore file share", workspace + "/datastores/d1", `{"properties":{"datastoreType":"AzureFile","accountName":"mlstore","credentials":{"credentialsType":"None"}}}`},
		{"deployment compute", endpoint + "/deployments/blue", `{"location":"eastus","properties":{"endpointComputeType":"Serverless"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureMLTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, workspace+"/onlineEndpoints/scorer/regenerateKeys"+azureMLTestAPIVersion, `{"keyType":"Primary"}`), http.StatusBadRequest, "BadRequest")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, workspace+"/computes/1cpu"+azureMLTestAPIVersion, `{"location":"eastus","properties":{"computeType":"ComputeInstance"}}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, workspace+"/onlineEndpoints/e_1"+azureMLTestAPIVersion, `{"location":"eastus","properties":{"authMode":"Key"}}`), http.StatusBadRequest, "InvalidResourceName")
}

func TestAzureMLSecretsAndLocations(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-ml")
	workspace := sub + "/resourceGroups/rg-ml/providers/Microsoft.MachineLearningServices/workspaces/research"
	resp := do(http.MethodPut, workspace+azureMLTestAPIVersion, strings.Replace(azureMLWorkspaceTestBody(sub), `"eastus"`, `"East US"`, 1))
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["discoveryUrl"]; got != "https://eastus.api.azureml.ms/discovery" {
		t.Fatalf("unexpected discovery URL: %v", got)
	}

	// Attach-time secrets are never returned, by PUT, GET or PATCH.
	for name, body := range map[string]string{
		"databricks": `{"location":"eastus","properties":{"computeType":"Databricks","resourceId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Databricks/workspaces/dbx","properties":{"databricksAccessToken":"dapi-secret","workspaceUrl":"adb-1.azuredatabricks.net"}}}`,
		"dsvm":       `{"location":"eastus","properties":{"computeType":"VirtualMachine","resourceId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/dsvm","properties":{"administratorAccount":{"username":"azureuser","password":"Secret-123","privateKeyData":"-----BEGIN"}}}}`,
	} {
		compute := workspace + "/computes/" + name
		azureResourcesExpectStatus(t, do(http.MethodPut, compute+azureMLTestAPIVersion, body), http.StatusCreated)
		for _, resp := range []*http.Response{
			do(http.MethodGet, compute+azureMLTestAPIVersion, ""),
			do(http.MethodPatch, compute+azureMLTestAPIVersion, `{"properties":{"description":"patched"}}`),
		} {
			azureResourcesExpectStatus(t, resp, http.StatusOK)
			raw := string(providerContractBody(t, resp))
			if strings.Contains(raw, "dapi-secret") || strings.Contains(raw, "Secret-123") || strings.Contains(raw, "BEGIN") {
				t.Fatalf("%s compute leaked a secret: %s", name, raw)
			}
		}
	}
}
