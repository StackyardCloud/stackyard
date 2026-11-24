package server

import (
	"net/http"
	"testing"
)

const (
	azureFabTestAPI  = "?api-version=2023-11-01"
	azureFabTestBody = `{"location":"eastus","sku":{"name":"F2","tier":"Fabric"},"properties":{"administration":{"members":["admin@contoso.com"]}}}`
)

func TestAzureFabricCapacityLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-fabric")
	capacity := sub + "/resourceGroups/rg-fabric/providers/Microsoft.Fabric/capacities/analytics"

	resp := do(http.MethodPut, capacity+azureFabTestAPI, azureFabTestBody)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	body := providerContractJSONMap(t, resp)
	if body["properties"].(map[string]any)["state"] != "Active" || body["sku"].(map[string]any)["name"] != "F2" {
		t.Fatalf("unexpected capacity: %#v", body)
	}

	state := func() any {
		resp := do(http.MethodGet, capacity+azureFabTestAPI, "")
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)["properties"].(map[string]any)["state"]
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, capacity+"/suspend"+azureFabTestAPI, ""), http.StatusOK)
	if got := state(); got != "Paused" {
		t.Fatalf("state after suspend = %v, want Paused", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, capacity+"/resume"+azureFabTestAPI, ""), http.StatusOK)
	if got := state(); got != "Active" {
		t.Fatalf("state after resume = %v, want Active", got)
	}

	resp = do(http.MethodPost, sub+"/providers/Microsoft.Fabric/locations/eastus/checkNameAvailability"+azureFabTestAPI, `{"name":"analytics","type":"Microsoft.Fabric/capacities"}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp); got["nameAvailable"] != false || got["reason"] != "AlreadyExists" {
		t.Fatalf("name check for a used name: %#v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, sub+"/providers/Microsoft.Fabric/locations/eastus/checkNameAvailability"+azureFabTestAPI, `{"name":"analytics"}`), http.StatusBadRequest)

	resp = do(http.MethodGet, sub+"/providers/Microsoft.Fabric/skus"+azureFabTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := len(providerContractJSONMap(t, resp)["value"].([]any)); got != len(azureFabSKUs) {
		t.Fatalf("listed %d SKUs, want %d", got, len(azureFabSKUs))
	}
	resp = do(http.MethodGet, capacity+"/skus"+azureFabTestAPI, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if items := providerContractJSONMap(t, resp)["value"].([]any); items[0].(map[string]any)["sku"].(map[string]any)["tier"] != "Fabric" {
		t.Fatalf("unexpected capacity SKUs: %#v", items)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, capacity+azureFabTestAPI, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, capacity+"/skus"+azureFabTestAPI, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureFabricCapacityValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-fabric")
	base := sub + "/resourceGroups/rg-fabric/providers/Microsoft.Fabric/capacities/"
	azureResourcesExpectErrorCode(t, do(http.MethodPut, base+"Bad-Name"+azureFabTestAPI, azureFabTestBody), http.StatusBadRequest, "InvalidResourceName")
	for name, body := range map[string]string{
		"unknown sku": `{"location":"eastus","sku":{"name":"P1","tier":"Fabric"},"properties":{"administration":{"members":["a@b.com"]}}}`,
		"wrong tier":  `{"location":"eastus","sku":{"name":"F2","tier":"Premium"},"properties":{"administration":{"members":["a@b.com"]}}}`,
		"no admins":   `{"location":"eastus","sku":{"name":"F2","tier":"Fabric"},"properties":{"administration":{"members":[]}}}`,
	} {
		resp := do(http.MethodPut, base+"capacity1"+azureFabTestAPI, body)
		t.Run(name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
}
