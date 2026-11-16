package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureMoverTestAPIVersion = "?api-version=2023-08-01"

func azureMoverTestState(t *testing.T, do func(method, path, body string) *http.Response, resource string) string {
	t.Helper()
	resp := do(http.MethodGet, resource+azureMoverTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	return providerContractJSONMap(t, resp)["properties"].(map[string]any)["moveStatus"].(map[string]any)["moveState"].(string)
}

func TestAzureResourceMoverLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-move")
	ids := strings.TrimPrefix(sub, "/azure") + "/resourceGroups/rg-app"
	collection := sub + "/resourceGroups/rg-move/providers/Microsoft.Migrate/moveCollections/eastus-to-westus"
	azureResourcesExpectStatus(t, do(http.MethodPut, collection+azureMoverTestAPIVersion, `{"location":"eastus2","identity":{"type":"SystemAssigned"},"properties":{"sourceRegion":"eastus","targetRegion":"westus2"}}`), http.StatusCreated)

	group := collection + "/moveResources/rg-app"
	vm := collection + "/moveResources/web1"
	resp := do(http.MethodPut, group+azureMoverTestAPIVersion, `{"properties":{"sourceId":"`+ids+`","resourceSettings":{"resourceType":"resourceGroups","targetResourceName":"rg-app-westus2"}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPut, vm+azureMoverTestAPIVersion, `{"properties":{"sourceId":"`+ids+`/providers/Microsoft.Compute/virtualMachines/web1","resourceSettings":{"resourceType":"Microsoft.Compute/virtualMachines","targetResourceName":"web1"},"moveStatus":{"moveState":"Committed"}}}`), http.StatusOK)
	if got := azureMoverTestState(t, do, vm); got != "PreparePending" {
		t.Fatalf("expected a new move resource pending preparation, got %s", got)
	}
	vmID := strings.TrimPrefix(vm, "/azure")

	// Steps out of order fail and change nothing.
	azureResourcesExpectErrorCode(t, do(http.MethodPost, collection+"/commit"+azureMoverTestAPIVersion, `{"moveResources":["`+vmID+`"]}`), http.StatusBadRequest, "BadRequest")
	azureResourcesExpectStatus(t, do(http.MethodPost, collection+"/prepare"+azureMoverTestAPIVersion, `{"moveResources":["`+vmID+`"],"validateOnly":true}`), http.StatusOK)
	if got := azureMoverTestState(t, do, vm); got != "PreparePending" {
		t.Fatalf("validateOnly changed the state to %s", got)
	}
	steps := []struct{ action, body, want string }{
		{"prepare", `{"moveResources":["` + ids + `"],"moveResourceInputType":"MoveResourceSourceId"}`, ""},
		{"prepare", `{"moveResources":["` + vmID + `"]}`, "MovePending"},
		{"initiateMove", `{"moveResources":["` + vmID + `"]}`, "CommitPending"},
		{"discard", `{"moveResources":["` + vmID + `"]}`, "MovePending"},
		{"initiateMove", `{"moveResources":["` + vmID + `"]}`, "CommitPending"},
		{"commit", `{"moveResources":["` + vmID + `"]}`, "DeleteSourcePending"},
	}
	for _, step := range steps {
		resp = do(http.MethodPost, collection+"/"+step.action+azureMoverTestAPIVersion, step.body)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		if got := providerContractJSONMap(t, resp)["status"]; got != "Succeeded" {
			t.Fatalf("%s: unexpected operation status %v", step.action, got)
		}
		if step.want != "" {
			if got := azureMoverTestState(t, do, vm); got != step.want {
				t.Fatalf("after %s: expected %s, got %s", step.action, step.want, got)
			}
		}
	}
	if got := azureMoverTestState(t, do, group); got != "MovePending" {
		t.Fatalf("expected the group prepared by source ID, got %s", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, collection+"/resolveDependencies"+azureMoverTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodGet, collection+"/unresolvedDependencies"+azureMoverTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, collection+"/bulkRemove"+azureMoverTestAPIVersion, `{}`), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, collection+"/bulkRemove"+azureMoverTestAPIVersion, `{"moveResources":["`+vmID+`"]}`), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, vm+azureMoverTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodPatch, group+azureMoverTestAPIVersion, `{"properties":{}}`), http.StatusMethodNotAllowed, "MethodNotAllowed")
	azureResourcesExpectStatus(t, do(http.MethodDelete, collection+azureMoverTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, group+azureMoverTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureResourceMoverValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-move")
	collections := sub + "/resourceGroups/rg-move/providers/Microsoft.Migrate/moveCollections"
	for name, body := range map[string]string{
		"regions":     `{"location":"eastus2","properties":{"sourceRegion":"eastus"}}`,
		"same region": `{"location":"eastus2","properties":{"sourceRegion":"East US","targetRegion":"eastus"}}`,
		"zone region": `{"location":"eastus2","properties":{"moveType":"RegionToZone"}}`,
		"move type":   `{"location":"eastus2","properties":{"moveType":"ZoneToZone"}}`,
	} {
		resp := do(http.MethodPut, collections+"/c1"+azureMoverTestAPIVersion, body)
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	azureResourcesExpectStatus(t, do(http.MethodPut, collections+"/c1"+azureMoverTestAPIVersion, `{"location":"eastus2","properties":{"moveType":"RegionToZone","moveRegion":"eastus"}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, collections+"/c1/moveResources/m"+azureMoverTestAPIVersion, `{"properties":{"sourceId":"web1"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, collections+"/c1/moveResources/m"+azureMoverTestAPIVersion, `{"properties":{"sourceId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Web/sites/a","resourceSettings":{"resourceType":"Microsoft.Web/sites"}}}`), http.StatusBadRequest, "InvalidRequestContent")
	for name, body := range map[string]string{
		"empty":      `{"moveResources":[]}`,
		"input type": `{"moveResources":["x"],"moveResourceInputType":"Name"}`,
		"unknown":    `{"moveResources":["/subscriptions/x/resourceGroups/rg/providers/Microsoft.Migrate/moveCollections/c1/moveResources/none"]}`,
	} {
		resp := do(http.MethodPost, collections+"/c1/prepare"+azureMoverTestAPIVersion, body)
		t.Run("prepare "+name, func(t *testing.T) {
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", resp.StatusCode)
			}
		})
	}
}
