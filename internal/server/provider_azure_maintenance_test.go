package server

import (
	"net/http"
	"strings"
	"testing"
)

const (
	azureMaintenanceTestAPIVersion = "?api-version=2023-04-01"
	azureMaintenanceTestPatchBody  = `{"location":"eastus","properties":{"maintenanceScope":"InGuestPatch","visibility":"Custom","extensionProperties":{"InGuestPatchMode":"User"},"maintenanceWindow":{"startDateTime":"2026-01-01 00:00","duration":"03:55","timeZone":"UTC","recurEvery":"Week Saturday"},"installPatches":{"rebootSetting":"IfRequired","linuxParameters":{"classificationsToInclude":["Critical","Security"]}}}}`
)

func TestAzureMaintenanceLifecycle(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-patch")
	config := sub + "/resourceGroups/rg-patch/providers/Microsoft.Maintenance/maintenanceConfigurations/weekly"
	azureResourcesExpectStatus(t, do(http.MethodPut, config+azureMaintenanceTestAPIVersion, azureMaintenanceTestPatchBody), http.StatusCreated)
	resp := do(http.MethodPatch, config+azureMaintenanceTestAPIVersion, `{"tags":{"team":"ops"}}`)
	if got := providerContractJSONMap(t, resp)["tags"].(map[string]any)["team"]; got != "ops" {
		t.Fatalf("expected the tag patched, got %v", got)
	}
	configID := strings.TrimPrefix(config, "/azure")

	vm := sub + "/resourcegroups/rg-patch/providers/Microsoft.Compute/virtualMachines/web1"
	assignment := vm + "/providers/Microsoft.Maintenance/configurationAssignments/weekly"
	resp = do(http.MethodPut, assignment+azureMaintenanceTestAPIVersion, `{"location":"eastus","properties":{"maintenanceConfigurationId":"`+configID+`"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	body := providerContractJSONMap(t, resp)
	if body["location"] != "eastus" || body["properties"].(map[string]any)["resourceId"] != strings.TrimPrefix(vm, "/azure") {
		t.Fatalf("unexpected assignment: %#v", body)
	}
	resp = do(http.MethodGet, vm+"/providers/Microsoft.Maintenance/configurationAssignments"+azureMaintenanceTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 1 {
		t.Fatalf("expected one assignment, got %d", len(got))
	}
	dynamic := sub + "/providers/Microsoft.Maintenance/configurationAssignments/linux"
	azureResourcesExpectStatus(t, do(http.MethodPut, dynamic+azureMaintenanceTestAPIVersion, `{"properties":{"maintenanceConfigurationId":"`+configID+`","filter":{"resourceTypes":["Microsoft.Compute/virtualMachines"],"osTypes":["Linux"],"tagSettings":{"tags":{"env":["prod"]},"filterOperator":"Any"}}}}`), http.StatusCreated)

	resp = do(http.MethodPut, vm+"/providers/Microsoft.Maintenance/applyUpdates/default"+azureMaintenanceTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	if got := providerContractJSONMap(t, resp)["properties"].(map[string]any)["status"]; got != "Completed" {
		t.Fatalf("expected a completed update, got %v", got)
	}
	resp = do(http.MethodGet, vm+"/providers/Microsoft.Maintenance/updates"+azureMaintenanceTestAPIVersion, "")
	if got := providerContractJSONMap(t, resp)["value"].([]any); len(got) != 0 {
		t.Fatalf("expected no pending updates, got %d", len(got))
	}
	resp = do(http.MethodGet, sub+"/providers/Microsoft.Maintenance/publicMaintenanceConfigurations/SQL_WestEurope_DB_2"+azureMaintenanceTestAPIVersion, "")
	azureResourcesExpectStatus(t, resp, http.StatusOK)
	if got := providerContractJSONMap(t, resp)["location"]; got != "westeurope" {
		t.Fatalf("unexpected public configuration location: %v", got)
	}

	azureResourcesExpectStatus(t, do(http.MethodDelete, assignment+azureMaintenanceTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, config+azureMaintenanceTestAPIVersion, ""), http.StatusOK)
}

func TestAzureMaintenanceValidation(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-patch")
	config := sub + "/resourceGroups/rg-patch/providers/Microsoft.Maintenance/maintenanceConfigurations/weekly" + azureMaintenanceTestAPIVersion
	cases := []struct{ name, old, new string }{
		{"scope", `"maintenanceScope":"InGuestPatch"`, `"maintenanceScope":"Guest"`},
		{"visibility", `"visibility":"Custom"`, `"visibility":"Private"`},
		{"start", `"startDateTime":"2026-01-01 00:00"`, `"startDateTime":"2026-01-01T00:00:00Z"`},
		{"duration", `"duration":"03:55"`, `"duration":"4h"`},
		{"recurrence", `,"recurEvery":"Week Saturday"`, ``},
		{"patch mode", `"extensionProperties":{"InGuestPatchMode":"User"},`, ``},
		{"patches", `,"installPatches":{"rebootSetting":"IfRequired","linuxParameters":{"classificationsToInclude":["Critical","Security"]}}`, ``},
		{"reboot", `"rebootSetting":"IfRequired"`, `"rebootSetting":"Sometimes"`},
	}
	for _, tc := range cases {
		body := strings.Replace(azureMaintenanceTestPatchBody, tc.old, tc.new, 1)
		if body == azureMaintenanceTestPatchBody {
			t.Fatalf("case %s did not change the body", tc.name)
		}
		resp := do(http.MethodPut, config, body)
		t.Run(tc.name, func(t *testing.T) {
			azureResourcesExpectErrorCode(t, resp, http.StatusBadRequest, "InvalidRequestContent")
		})
	}
	assignment := sub + "/providers/Microsoft.Maintenance/configurationAssignments/a" + azureMaintenanceTestAPIVersion
	azureResourcesExpectErrorCode(t, do(http.MethodPut, assignment, `{"properties":{"maintenanceConfigurationId":"weekly"}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, assignment, `{"properties":{"maintenanceConfigurationId":"/subscriptions/x/resourceGroups/rg/providers/Microsoft.Maintenance/maintenanceConfigurations/c","filter":{"osTypes":["BSD"]}}}`), http.StatusBadRequest, "InvalidRequestContent")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/providers/Microsoft.Maintenance/applyUpdates/default"+azureMaintenanceTestAPIVersion, ""), http.StatusBadRequest, "InvalidScope")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourcegroups/rg-patch/providers/Microsoft.Compute/virtualMachines/web1/providers/Microsoft.Maintenance/applyUpdates/now"+azureMaintenanceTestAPIVersion, ""), http.StatusBadRequest, "InvalidResourceName")
	// Only InGuestPatch needs a window, and a recurrence in it.
	azureResourcesExpectStatus(t, do(http.MethodPut, config, `{"location":"eastus","properties":{"maintenanceScope":"Host"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPut, config, `{"location":"eastus","properties":{"maintenanceScope":"OSImage","maintenanceWindow":{"startDateTime":"2026-01-01 00:00","duration":"05:00"}}}`), http.StatusOK)
}
