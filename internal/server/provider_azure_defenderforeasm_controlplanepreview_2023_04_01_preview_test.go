package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureEASMCTestAPI = "?api-version=2023-04-01-preview"

func TestAzureDefenderForEasmControlPlanePreview20230401Lifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-easm")
	workspace := sub + "/resourceGroups/rg-easm/providers/Microsoft.Easm/workspaces/ws1"

	resp := do(http.MethodPut, workspace+azureEASMCTestAPI, `{"location":"eastus","properties":{}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if endpoint := providerContractJSONMap(t, resp)["properties"].(map[string]any)["dataPlaneEndpoint"]; !strings.HasPrefix(endpoint.(string), "https://eastus.easm.") {
		t.Fatalf("unexpected dataPlaneEndpoint %v", endpoint)
	}
	label := workspace + "/labels/l1"
	azureResourcesExpectStatus(t, do(http.MethodPut, label+azureEASMCTestAPI, `{"properties":{"color":"blue","displayName":"Label"}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, label+azureEASMCTestAPI, `{"properties":{"color":"not a color!"}}`), http.StatusBadRequest, "InvalidRequestContent")
	if n := len(providerContractJSONMap(t, do(http.MethodGet, workspace+"/labels"+azureEASMCTestAPI, ""))["value"].([]any)); n != 1 {
		t.Fatalf("expected 1 label, got %d", n)
	}

	// Workspace tasks are the data plane's tasks.
	azureResourcesExpectStatus(t, do(http.MethodGet, workspace+"/tasks/missing"+azureEASMCTestAPI, ""), http.StatusNotFound)
	task := providerContractJSONMap(t, do(http.MethodPost, "/azure/defenderforeasm-dataplanepreview/2024-10-01-preview/assets:export?api-version=2024-10-01-preview", `{"fileName":"a.csv","columns":["name"]}`))
	azureResourcesExpectStatus(t, do(http.MethodGet, workspace+"/tasks/"+task["id"].(string)+azureEASMCTestAPI, ""), http.StatusOK)

	azureResourcesExpectErrorCode(t, do(http.MethodPut, workspace+"/widgets/w"+azureEASMCTestAPI, `{}`), http.StatusNotFound, "InvalidResourceType")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-easm/providers/Microsoft.Easm/workspaces/-bad-"+azureEASMCTestAPI, `{"location":"eastus"}`), http.StatusBadRequest, "InvalidResourceName")
	azureResourcesExpectStatus(t, do(http.MethodDelete, label+azureEASMCTestAPI, ""), http.StatusAccepted)
	azureResourcesExpectStatus(t, do(http.MethodDelete, workspace+azureEASMCTestAPI, ""), http.StatusAccepted)
	azureResourcesExpectStatus(t, do(http.MethodGet, "/azure/providers/Microsoft.Easm/operations"+azureEASMCTestAPI, ""), http.StatusOK)
}
