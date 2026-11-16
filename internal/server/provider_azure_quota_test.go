package server

import (
	"net/http"
	"strings"
	"testing"
)

const azureQuotaTestAPIVersion = "?api-version=2025-09-01"

func TestAzureQuotaLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-quota")
	azureResourcesExpectStatus(t, do(http.MethodPut, sub+"/resourceGroups/rg-quota/providers/Microsoft.Network/virtualNetworks/hub?api-version=2024-05-01", `{"location":"East US","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}}}`), http.StatusCreated)
	network := sub + "/providers/Microsoft.Network/locations/eastus/providers/Microsoft.Quota"

	resp := do(http.MethodGet, network+"/usages/VirtualNetworks"+azureQuotaTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["usages"].(map[string]any)["value"]; got != float64(1) {
		t.Fatalf("expected one virtual network in use, got %v", got)
	}
	resp = do(http.MethodGet, network+"/quotas/virtualnetworks"+azureQuotaTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["limit"].(map[string]any)["value"]; got != float64(1000) {
		t.Fatalf("expected the default limit, got %v", got)
	}

	// A PUT that raises a limit answers 200, as the quota always existed.
	resp = do(http.MethodPut, network+"/quotas/VirtualNetworks"+azureQuotaTestAPIVersion, `{"properties":{"limit":{"limitObjectType":"LimitValue","value":1500}}}`)
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["name"].(map[string]any)["localizedValue"]; got != "Virtual Networks" {
		t.Fatalf("unexpected localized name: %v", got)
	}
	resp = do(http.MethodGet, network+"/quotas"+azureQuotaTestAPIVersion, "")
	items := providerContractJSONMap(t, resp)["value"].([]any)
	if len(items) != len(azureQuotaDefaults["microsoft.network"]) {
		t.Fatalf("expected one limit per default, got %d", len(items))
	}
	if got := items[0].(map[string]any)["properties"].(map[string]any)["limit"].(map[string]any)["value"]; got != float64(1500) {
		t.Fatalf("expected the raised limit listed, got %v", got)
	}
	resp = do(http.MethodGet, network+"/quotaRequests"+azureQuotaTestAPIVersion, "")
	requests := providerContractJSONMap(t, resp)["value"].([]any)
	if len(requests) != 1 {
		t.Fatalf("expected one quota request, got %d", len(requests))
	}
	request := requests[0].(map[string]any)
	if got := request["properties"].(map[string]any)["value"].([]any)[0].(map[string]any)["name"].(map[string]any)["value"]; got != "VirtualNetworks" {
		t.Fatalf("unexpected request name: %v", got)
	}
	resp = do(http.MethodGet, network+"/quotaRequests/"+request["name"].(string)+azureQuotaTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["provisioningState"]; got != "Succeeded" {
		t.Fatalf("unexpected request state: %v", got)
	}

	compute := sub + "/providers/Microsoft.Compute/locations/westus2/providers/Microsoft.Quota"
	azureResourcesExpectStatus(t, do(http.MethodPut, compute+"/quotas/standardHBv4Family"+azureQuotaTestAPIVersion, `{"properties":{"limit":{"limitObjectType":"LimitValue","value":176}}}`), http.StatusOK)
	resp = do(http.MethodGet, compute+"/quotas"+azureQuotaTestAPIVersion, "")
	if items := providerContractJSONMap(t, resp)["value"].([]any); len(items) != len(azureQuotaDefaults["microsoft.compute"])+1 {
		t.Fatalf("expected the new family after the defaults, got %d", len(items))
	}
	resp = do(http.MethodPatch, compute+"/quotas/standardHBv4Family"+azureQuotaTestAPIVersion, `{"properties":{"limit":{"limitObjectType":"LimitValue","value":352}}}`)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["limit"].(map[string]any)["value"]; got != float64(352) {
		t.Fatalf("expected the patched limit, got %v", got)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodGet, compute+"/quotas/standardXYZFamily"+azureQuotaTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
}

func TestAzureQuotaValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-quota")
	scope := sub + "/providers/Microsoft.Compute/locations/eastus/providers/Microsoft.Quota/quotas/cores" + azureQuotaTestAPIVersion
	cases := []struct{ name, body string }{
		{"object type", `{"properties":{"limit":{"limitObjectType":"Limit","value":10}}}`},
		{"negative", `{"properties":{"limit":{"limitObjectType":"LimitValue","value":-1}}}`},
		{"fraction", `{"properties":{"limit":{"limitObjectType":"LimitValue","value":1.5}}}`},
		{"limit type", `{"properties":{"limit":{"limitObjectType":"LimitValue","value":10,"limitType":"Pooled"}}}`},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, scope, tc.body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	group := strings.Replace(scope, "/providers/Microsoft.Compute/locations/eastus", "/resourceGroups/rg-quota", 1)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, group, `{"properties":{"limit":{"limitObjectType":"LimitValue","value":10}}}`), http.StatusBadRequest, "InvalidScope")
}
