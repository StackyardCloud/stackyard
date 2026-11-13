package server

import (
	"net/http"
	"testing"
)

const azureResourceHealthTestAPIVersion = "?api-version=2022-10-01"

func TestAzureResourceHealthAvailabilityStatuses(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-health")
	vault := sub + "/resourceGroups/rg-health/providers/Microsoft.RecoveryServices/vaults/v1"
	azureResourcesExpectStatus(t, do(http.MethodPut, vault+"?api-version=2024-04-01", `{"location":"westus","sku":{"name":"Standard"}}`), http.StatusCreated)

	// The vault is listed, but not its proxy children (backup policies and so on).
	for _, scope := range []string{sub, sub + "/resourceGroups/rg-health"} {
		resp := do(http.MethodGet, scope+"/providers/Microsoft.ResourceHealth/availabilityStatuses"+azureResourceHealthTestAPIVersion, "")
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		items, _ := providerContractJSONMap(t, resp)["value"].([]any)
		if len(items) != 1 {
			t.Fatalf("%s: expected one status, got %#v", scope, items)
		}
		status := items[0].(map[string]any)
		if status["location"] != "westus" || status["properties"].(map[string]any)["availabilityState"] != "Available" {
			t.Fatalf("unexpected status: %#v", status)
		}
	}

	resp := do(http.MethodGet, vault+"/providers/Microsoft.ResourceHealth/availabilityStatuses/current"+azureResourceHealthTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp); got["id"] != vault[len("/azure"):]+"/providers/Microsoft.ResourceHealth/availabilityStatuses/current" {
		t.Fatalf("unexpected status id: %#v", got["id"])
	}
	// "current" exists only at resource scope, not for whichever resource a list holds.
	azureResourcesExpectErrorCode(t, do(http.MethodGet, sub+"/providers/Microsoft.ResourceHealth/availabilityStatuses/current"+azureResourceHealthTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodGet, sub+"/resourceGroups/rg-missing/providers/Microsoft.ResourceHealth/availabilityStatuses"+azureResourceHealthTestAPIVersion, ""), http.StatusNotFound, "ResourceGroupNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodGet, vault+"/providers/Microsoft.ResourceHealth/availabilityStatuses/yesterday"+azureResourceHealthTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, vault+"/providers/Microsoft.ResourceHealth/availabilityStatuses/current"+azureResourceHealthTestAPIVersion, `{}`), http.StatusMethodNotAllowed, "MethodNotAllowed")

	resp = do(http.MethodGet, sub+"/providers/Microsoft.ResourceHealth/events"+azureResourceHealthTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if items, ok := providerContractJSONMap(t, resp)["value"].([]any); !ok || len(items) != 0 {
		t.Fatalf("expected an empty event list, got %#v", items)
	}
}
