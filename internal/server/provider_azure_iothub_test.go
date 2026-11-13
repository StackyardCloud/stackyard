package server

import (
	"net/http"
	"testing"
)

const azureIoTHubTestAPIVersion = "?api-version=2023-06-30"

func TestAzureIoTHubLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-iot")
	hub := sub + "/resourceGroups/rg-iot/providers/Microsoft.Devices/IotHubs/Plant-Hub"

	resp := do(http.MethodPut, hub+azureIoTHubTestAPIVersion, `{"location":"eastus","sku":{"name":"S1","capacity":1},"properties":{"eventHubEndpoints":{"events":{"partitionCount":2,"retentionTimeInDays":3}},"authorizationPolicies":[{"keyName":"custom","primaryKey":"secret","rights":"ServiceConnect"}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	events, _ := props["eventHubEndpoints"].(map[string]any)["events"].(map[string]any)
	if props["hostName"] != "plant-hub.azure-devices.net" || props["state"] != "Active" || props["authorizationPolicies"] != nil {
		t.Fatalf("unexpected hub properties: %#v", props)
	}
	if ids, _ := events["partitionIds"].([]any); len(ids) != 2 || events["retentionTimeInDays"] != float64(3) || events["path"] != "plant-hub" {
		t.Fatalf("unexpected built-in endpoint: %#v", events)
	}

	resp = do(http.MethodPost, hub+"/listkeys"+azureIoTHubTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	keys, _ := providerContractJSONMap(t, resp)["value"].([]any)
	if len(keys) != 5 || keys[0].(map[string]any)["keyName"] != "iothubowner" || len(keys[0].(map[string]any)["primaryKey"].(string)) != 44 {
		t.Fatalf("unexpected keys: %#v", keys)
	}

	groups := hub + "/eventHubEndpoints/events/ConsumerGroups"
	resp = do(http.MethodGet, groups+azureIoTHubTestAPIVersion, "")
	if items, _ := providerContractJSONMap(t, resp)["value"].([]any); len(items) != 1 || items[0].(map[string]any)["name"] != "$Default" {
		t.Fatalf("expected the $Default consumer group, got %#v", items)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPut, groups+"/analytics"+azureIoTHubTestAPIVersion, `{"properties":{}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectStatus(t, do(http.MethodPut, groups+"/analytics"+azureIoTHubTestAPIVersion, `{"properties":{"name":"analytics"}}`), http.StatusCreated)
	azureResourcesExpectErrorCode(t, do(http.MethodPut, hub+"/eventHubEndpoints/other/ConsumerGroups/x"+azureIoTHubTestAPIVersion, `{"properties":{"name":"x"}}`), http.StatusNotFound, "ParentResourceNotFound")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, hub+"/eventHubEndpoints/other"+azureIoTHubTestAPIVersion, `{}`), http.StatusBadRequest, "InvalidResourceName")

	// PATCH skips SKU validation; an invalid partition count must neither
	// panic nor wedge the store.
	for _, count := range []string{"-1", "1e12", "2.5"} {
		resp = do(http.MethodPatch, hub+azureIoTHubTestAPIVersion, `{"properties":{"eventHubEndpoints":{"events":{"partitionCount":`+count+`}}}}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		events := providerContractJSONMap(t, resp)["properties"].(map[string]any)["eventHubEndpoints"].(map[string]any)["events"].(map[string]any)
		if ids, _ := events["partitionIds"].([]any); len(ids) != 4 {
			t.Fatalf("partitionCount %s: expected the default 4 partitions, got %d", count, len(ids))
		}
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, hub+azureIoTHubTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, groups+"/analytics"+azureIoTHubTestAPIVersion, ""), http.StatusNotFound, "ParentResourceNotFound")
}

func TestAzureIoTHubValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-iot")
	hubs := sub + "/resourceGroups/rg-iot/providers/Microsoft.Devices/IotHubs/"
	cases := []struct{ name, path, body, code string }{
		{"trailing hyphen", hubs + "plant-", `{"location":"eastus","sku":{"name":"S1"}}`, "InvalidResourceName"},
		{"missing sku", hubs + "plant", `{"location":"eastus"}`, "InvalidRequestContent"},
		{"bad sku", hubs + "plant", `{"location":"eastus","sku":{"name":"S4"}}`, "InvalidRequestContent"},
		{"one partition", hubs + "plant", `{"location":"eastus","sku":{"name":"S1"},"properties":{"eventHubEndpoints":{"events":{"partitionCount":1}}}}`, "InvalidRequestContent"},
		{"long retention", hubs + "plant", `{"location":"eastus","sku":{"name":"S1"},"properties":{"eventHubEndpoints":{"events":{"retentionTimeInDays":8}}}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureIoTHubTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}

	check := func(name string) map[string]any {
		resp := do(http.MethodPost, sub+"/providers/Microsoft.Devices/checkNameAvailability"+azureIoTHubTestAPIVersion, `{"name":"`+name+`"}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}
	if got := check("plant"); got["nameAvailable"] != true {
		t.Fatalf("expected available, got %#v", got)
	}
	resp := do(http.MethodPut, hubs+"plant"+azureIoTHubTestAPIVersion, `{"location":"westus","sku":{"name":"F1"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if events := providerContractJSONMap(t, resp)["properties"].(map[string]any)["eventHubEndpoints"].(map[string]any)["events"].(map[string]any); events["partitionCount"] != float64(4) || events["retentionTimeInDays"] != float64(1) {
		t.Fatalf("expected default partitions and retention, got %#v", events)
	}
	if got := check("PLANT"); got["reason"] != "AlreadyExists" {
		t.Fatalf("expected AlreadyExists, got %#v", got)
	}
	if got := check("-plant"); got["reason"] != "Invalid" {
		t.Fatalf("expected Invalid, got %#v", got)
	}
}
