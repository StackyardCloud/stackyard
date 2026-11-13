package server

import (
	"net/http"
	"regexp"
	"testing"
)

const azureDPSTestAPIVersion = "?api-version=2022-02-05"

func TestAzureDeviceProvisioningServicesLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dps")
	service := sub + "/resourceGroups/rg-dps/providers/Microsoft.Devices/provisioningServices/plant-dps"

	resp := do(http.MethodPut, service+azureDPSTestAPIVersion, `{"location":"eastus","sku":{"name":"S1","capacity":1},"properties":{"iotHubs":[{"connectionString":"HostName=plant-hub.azure-devices.net;SharedAccessKeyName=iothubowner;SharedAccessKey=c2VjcmV0","location":"eastus"}]}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	props, _ := providerContractJSONMap(t, resp)["properties"].(map[string]any)
	if props["serviceOperationsHostName"] != "plant-dps.azure-devices-provisioning.net" || props["allocationPolicy"] != "Hashed" || props["state"] != "Active" {
		t.Fatalf("unexpected service properties: %#v", props)
	}
	if scope, _ := props["idScope"].(string); !regexp.MustCompile(`^0ne[0-9A-F]{8}$`).MatchString(scope) {
		t.Fatalf("unexpected idScope: %q", scope)
	}
	hub, _ := props["iotHubs"].([]any)[0].(map[string]any)
	if hub["connectionString"] != "HostName=plant-hub.azure-devices.net;SharedAccessKeyName=iothubowner;SharedAccessKey=****" || hub["name"] != "plant-hub.azure-devices.net" {
		t.Fatalf("linked hub keys must be masked: %#v", hub)
	}
	// The ID scope is stable across updates.
	resp = do(http.MethodPatch, service+azureDPSTestAPIVersion, `{"tags":{"env":"test"}}`)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any); got["idScope"] != props["idScope"] {
		t.Fatalf("idScope changed: %#v", got)
	}

	resp = do(http.MethodPost, service+"/listkeys"+azureDPSTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if keys, _ := providerContractJSONMap(t, resp)["value"].([]any); len(keys) != 1 || keys[0].(map[string]any)["keyName"] != "provisioningserviceowner" {
		t.Fatalf("unexpected keys: %#v", keys)
	}

	check := func(name string) map[string]any {
		resp := do(http.MethodPost, sub+"/providers/Microsoft.Devices/checkProvisioningServiceNameAvailability"+azureDPSTestAPIVersion, `{"name":"`+name+`"}`)
		azureResourcesExpectStatus(t, resp, http.StatusOK)
		return providerContractJSONMap(t, resp)
	}
	if got := check("plant-dps"); got["reason"] != "AlreadyExists" {
		t.Fatalf("expected AlreadyExists, got %#v", got)
	}
	// Hub and provisioning service names are checked separately.
	resp = do(http.MethodPost, sub+"/providers/Microsoft.Devices/checkNameAvailability?api-version=2023-06-30", `{"name":"plant-dps"}`)
	if got := providerContractJSONMap(t, resp); got["nameAvailable"] != true {
		t.Fatalf("expected hub name to stay available, got %#v", got)
	}
	azureResourcesExpectStatus(t, do(http.MethodDelete, service+azureDPSTestAPIVersion, ""), http.StatusOK)
	if got := check("plant-dps"); got["nameAvailable"] != true {
		t.Fatalf("expected available after delete, got %#v", got)
	}
}

func TestAzureDeviceProvisioningServicesValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-dps")
	services := sub + "/resourceGroups/rg-dps/providers/Microsoft.Devices/provisioningServices/"
	cases := []struct{ name, path, body, code string }{
		{"bad name", services + "plant_dps", `{"location":"eastus","sku":{"name":"S1"}}`, "InvalidResourceName"},
		{"bad sku", services + "plant-dps", `{"location":"eastus","sku":{"name":"F1"}}`, "InvalidRequestContent"},
		{"bad allocation", services + "plant-dps", `{"location":"eastus","sku":{"name":"S1"},"properties":{"allocationPolicy":"RoundRobin"}}`, "InvalidRequestContent"},
		{"hub without host", services + "plant-dps", `{"location":"eastus","sku":{"name":"S1"},"properties":{"iotHubs":[{"connectionString":"SharedAccessKey=x","location":"eastus"}]}}`, "InvalidRequestContent"},
		{"hub without location", services + "plant-dps", `{"location":"eastus","sku":{"name":"S1"},"properties":{"iotHubs":[{"connectionString":"HostName=h.azure-devices.net;SharedAccessKey=x"}]}}`, "InvalidRequestContent"},
	}
	for _, tc := range cases {
		resp := do(http.MethodPut, tc.path+azureDPSTestAPIVersion, tc.body)
		t.Run(tc.name, func(t *testing.T) { azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, tc.code) })
	}
}
