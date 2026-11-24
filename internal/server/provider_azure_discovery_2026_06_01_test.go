package server

import (
	"net/http"
	"strings"
	"testing"
)

const discTestAPIVersion = "?api-version=2026-06-01"

func TestAzureDiscoveryLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-disc")
	rg := sub + "/resourceGroups/rg-disc/providers/Microsoft.Discovery/"
	subnet := "/subscriptions/" + azureResourcesTestSubscription + "/resourceGroups/rg-disc/providers/Microsoft.Network/virtualNetworks/hub/subnets/compute"
	storageAccount := "/subscriptions/" + azureResourcesTestSubscription + "/resourceGroups/rg-disc/providers/Microsoft.Storage/storageAccounts/data"
	idOf := func(path string) string { return strings.TrimPrefix(path, "/azure") }

	supercomputer := rg + "supercomputers/sc1"
	azureResourcesExpectStatus(t, do(http.MethodPut, supercomputer+discTestAPIVersion, `{"location":"eastus","properties":{"subnetId":"`+subnet+`"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, supercomputer+"/nodePools/gpu"+discTestAPIVersion,
		`{"location":"eastus","properties":{"subnetId":"`+subnet+`","vmSize":"Standard_NC24ads_A100_v4","maxNodeCount":4,"minNodeCount":0,"scaleSetPriority":"Spot"}}`), http.StatusCreated)

	container := rg + "storageContainers/data"
	azureResourcesExpectStatus(t, do(http.MethodPut, container+discTestAPIVersion,
		`{"location":"eastus","properties":{"storageStore":{"kind":"AzureStorageBlob","storageAccountId":"`+storageAccount+`"}}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, container+"/storageAssets/genomes"+discTestAPIVersion, `{"location":"eastus","properties":{"path":"genomes/"}}`), http.StatusCreated)

	workspace := rg + "workspaces/Research"
	resp := do(http.MethodPut, workspace+discTestAPIVersion, `{"location":"eastus","properties":{"supercomputerIds":["`+idOf(supercomputer)+`"]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if uri := providerContractJSONMap(t, resp)["properties"].(map[string]any)["workspaceApiUri"]; uri != "https://research.eastus.discovery.azure.com" {
		t.Fatalf("unexpected workspace URI %v", uri)
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, workspace+"/projects/p1"+discTestAPIVersion, `{"location":"eastus","properties":{"storageContainerIds":["`+idOf(container)+`"]}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, workspace+"/chatModelDeployments/gpt"+discTestAPIVersion, `{"location":"eastus","properties":{"modelFormat":"OpenAI","modelName":"gpt-4o"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPatch, workspace+discTestAPIVersion, `{"tags":{"lab":"bio"}}`), http.StatusOK)

	azureResourcesExpectStatus(t, do(http.MethodPut, rg+"tools/fold"+discTestAPIVersion, `{"location":"eastus","properties":{"version":"1.0","definitionContent":{"name":"fold"}}}`), http.StatusCreated)
	resp = do(http.MethodPut, rg+"bookshelves/shelf"+discTestAPIVersion, `{"location":"eastus","properties":{"publicNetworkAccess":"Disabled"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if uri := providerContractJSONMap(t, resp)["properties"].(map[string]any)["bookshelfUri"]; uri == nil {
		t.Fatal("expected a bookshelf URI")
	}
	for _, parent := range []string{workspace, rg + "bookshelves/shelf"} {
		azureResourcesExpectStatus(t, do(http.MethodPut, parent+"/privateEndpointConnections/pe"+discTestAPIVersion,
			`{"properties":{"privateLinkServiceConnectionState":{"status":"Pending"}}}`), http.StatusCreated)
		if links := adnsTestValue(t, do(http.MethodGet, parent+"/privateLinkResources"+discTestAPIVersion, "")); len(links) != 1 {
			t.Fatalf("unexpected private link resources: %#v", links)
		}
	}
	if ops := adnsTestValue(t, do(http.MethodGet, "/azure/providers/Microsoft.Discovery/operations"+discTestAPIVersion, "")); len(ops) == 0 {
		t.Fatal("expected provider operations")
	}
	if list := adnsTestValue(t, do(http.MethodGet, sub+"/providers/Microsoft.Discovery/workspaces"+discTestAPIVersion, "")); len(list) != 1 {
		t.Fatalf("expected one workspace in the subscription, got %d", len(list))
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, workspace+discTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, workspace+"/projects/p1"+discTestAPIVersion, ""), http.StatusNotFound)
}

func TestAzureDiscoveryValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-disc")
	rg := sub + "/resourceGroups/rg-disc/providers/Microsoft.Discovery/"
	subnet := "/subscriptions/" + azureResourcesTestSubscription + "/resourceGroups/rg-disc/providers/Microsoft.Network/virtualNetworks/hub/subnets/compute"

	cases := []struct{ name, path, body string }{
		{"supercomputer without subnet", "supercomputers/sc", `{"location":"eastus","properties":{}}`},
		{"storage container bad kind", "storageContainers/c", `{"location":"eastus","properties":{"storageStore":{"kind":"Tape"}}}`},
		{"tool without definition", "tools/t", `{"location":"eastus","properties":{"version":"1"}}`},
		{"workspace bad supercomputer ref", "workspaces/w", `{"location":"eastus","properties":{"supercomputerIds":["/subscriptions/x/whatever"]}}`},
		{"bookshelf bad access", "bookshelves/b", `{"location":"eastus","properties":{"publicNetworkAccess":"Sometimes"}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, rg+tc.path+discTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}

	supercomputer := rg + "supercomputers/sc"
	azureResourcesExpectStatus(t, do(http.MethodPut, supercomputer+discTestAPIVersion, `{"location":"eastus","properties":{"subnetId":"`+subnet+`"}}`), http.StatusCreated)
	pools := []struct{ name, body string }{
		{"no max nodes", `{"location":"eastus","properties":{"subnetId":"` + subnet + `","vmSize":"Standard_D4s_v5"}}`},
		{"min above max", `{"location":"eastus","properties":{"subnetId":"` + subnet + `","vmSize":"Standard_D4s_v5","maxNodeCount":2,"minNodeCount":5}}`},
		{"bad priority", `{"location":"eastus","properties":{"subnetId":"` + subnet + `","vmSize":"Standard_D4s_v5","maxNodeCount":2,"scaleSetPriority":"Low"}}`},
	}
	for _, tc := range pools {
		resp := do(http.MethodPut, supercomputer+"/nodePools/np"+discTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, rg+"workspaces/absent/projects/p"+discTestAPIVersion, `{"location":"eastus","properties":{}}`), http.StatusNotFound, "ParentResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, rg+"workspaces/absent/privateEndpointConnections/pe"+discTestAPIVersion,
		`{"properties":{"privateLinkServiceConnectionState":{"status":"Pending"}}}`), http.StatusNotFound, "ParentResourceNotFound")
}
