package server

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

const azureSerialTestAPIVersion = "?api-version=2024-07-01"

var azureSerialTestConnection = regexp.MustCompile(`^wss://serialconsole\.stackyard\.invalid/n/connector/[0-9a-f]{12}/sessions/[0-9a-f-]{36}/client$`)

func TestAzureSerialConsolePorts(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-serial")
	vm := sub + "/resourceGroups/rg-serial/providers/Microsoft.Compute/virtualMachines/vm1"
	port := vm + "/providers/Microsoft.SerialConsole/serialPorts/0"

	resp := do(http.MethodPut, port+azureSerialTestAPIVersion, `{"properties":{"state":"enabled"}}`)
	azureResourcesExpectStatus(t, resp, http.StatusCreated)
	created := providerContractJSONMap(t, resp)
	props := created["properties"].(map[string]any)
	if created["type"] != "Microsoft.SerialConsole/serialPorts" || props["state"] != "enabled" || props["connectionState"] != "inactive" {
		t.Fatalf("unexpected port: %#v", created)
	}
	got := providerContractJSONMap(t, do(http.MethodGet, port+azureSerialTestAPIVersion, ""))
	if got["name"] != "0" || !strings.HasSuffix(azureARMString(got["id"]), "/virtualMachines/vm1/providers/Microsoft.SerialConsole/serialPorts/0") {
		t.Fatalf("unexpected GET: %#v", got)
	}

	// A scale set instance's port; lists hold a resource's own ports, and
	// the subscription list holds all of them.
	instance := sub + "/resourceGroups/rg-serial/providers/Microsoft.Compute/virtualMachineScaleSets/ss1/virtualMachines/2"
	azureResourcesExpectStatus(t, do(http.MethodPut, instance+"/providers/Microsoft.SerialConsole/serialPorts/0"+azureSerialTestAPIVersion, `{"properties":{"state":"disabled","connectionState":"active"}}`), http.StatusCreated)
	if list := providerContractJSONMap(t, do(http.MethodGet, vm+"/providers/Microsoft.SerialConsole/serialPorts"+azureSerialTestAPIVersion, ""))["value"].([]any); len(list) != 1 {
		t.Fatalf("expected the VM's one port, got %#v", list)
	}
	if list := providerContractJSONMap(t, do(http.MethodGet, sub+"/providers/Microsoft.SerialConsole/serialPorts"+azureSerialTestAPIVersion, ""))["value"].([]any); len(list) != 2 {
		t.Fatalf("expected both ports in the subscription, got %#v", list)
	}

	// Connect needs an enabled port.
	conn := providerContractJSONMap(t, do(http.MethodPost, port+"/connect"+azureSerialTestAPIVersion, ""))
	if !azureSerialTestConnection.MatchString(azureARMString(conn["connectionString"])) {
		t.Fatalf("unexpected connect result: %#v", conn)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, instance+"/providers/Microsoft.SerialConsole/serialPorts/0/connect"+azureSerialTestAPIVersion, ""), http.StatusConflict, "SerialPortDisabled")

	// Validation, scope and verbs.
	azureSentinelExpectRejected(t, do(http.MethodPut, port+azureSerialTestAPIVersion, `{"properties":{"state":"on"}}`), "properties.state")
	azureSentinelExpectRejected(t, do(http.MethodPut, port+azureSerialTestAPIVersion, `{"properties":{"connectionState":"open"}}`), "properties.connectionState")
	azureSentinelExpectRejected(t, do(http.MethodPut, port+azureSerialTestAPIVersion, `{"properties":"x"}`), "properties must be an object")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/rg-serial/providers/Microsoft.Storage/storageAccounts/sa/providers/Microsoft.SerialConsole/serialPorts/0"+azureSerialTestAPIVersion, `{"properties":{}}`), http.StatusBadRequest, "InvalidScope")
	azureResourcesExpectErrorCode(t, do(http.MethodPut, sub+"/resourceGroups/missing/providers/Microsoft.Compute/virtualMachines/vm1/providers/Microsoft.SerialConsole/serialPorts/0"+azureSerialTestAPIVersion, `{"properties":{}}`), http.StatusNotFound, "ResourceGroupNotFound")
	azureResourcesExpectStatus(t, do(http.MethodPatch, port+azureSerialTestAPIVersion, `{"properties":{"state":"disabled"}}`), http.StatusMethodNotAllowed)

	azureResourcesExpectStatus(t, do(http.MethodDelete, port+azureSerialTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodDelete, port+azureSerialTestAPIVersion, ""), http.StatusNoContent)
	azureResourcesExpectErrorCode(t, do(http.MethodGet, port+azureSerialTestAPIVersion, ""), http.StatusNotFound, "ResourceNotFound")
}

func TestAzureSerialConsoleSubscriptionSwitch(t *testing.T) {
	t.Parallel()
	sub, do := newAzureDatabaseTestServer(t, "rg-serial")
	service := sub + "/providers/Microsoft.SerialConsole/consoleServices/default"
	disabled := func() any {
		return providerContractJSONMap(t, do(http.MethodGet, service+azureSerialTestAPIVersion, ""))["properties"].(map[string]any)["disabled"]
	}
	if disabled() != false {
		t.Fatal("Serial Console should start enabled")
	}
	port := sub + "/resourceGroups/rg-serial/providers/Microsoft.Compute/virtualMachines/vm1/providers/Microsoft.SerialConsole/serialPorts/0"
	azureResourcesExpectStatus(t, do(http.MethodPut, port+azureSerialTestAPIVersion, `{"properties":{"state":"enabled"}}`), http.StatusCreated)

	off := providerContractJSONMap(t, do(http.MethodPost, service+"/disableConsole"+azureSerialTestAPIVersion, ""))
	if off["properties"].(map[string]any)["disabled"] != true || disabled() != true {
		t.Fatalf("disableConsole should turn Serial Console off: %#v", off)
	}
	azureResourcesExpectErrorCode(t, do(http.MethodPost, port+"/connect"+azureSerialTestAPIVersion, ""), http.StatusConflict, "SerialConsoleDisabled")

	on := providerContractJSONMap(t, do(http.MethodPost, service+"/enableConsole"+azureSerialTestAPIVersion, ""))
	if on["properties"].(map[string]any)["disabled"] != false || disabled() != false {
		t.Fatalf("enableConsole should turn Serial Console on: %#v", on)
	}
	azureResourcesExpectStatus(t, do(http.MethodPost, port+"/connect"+azureSerialTestAPIVersion, ""), http.StatusOK)

	// The switch is per subscription and has no other verbs.
	otherSub := "/azure/subscriptions/11111111-1111-1111-1111-111111111111"
	azureResourcesExpectStatus(t, do(http.MethodPut, otherSub+"/resourcegroups/rg-serial?api-version=2021-04-01", `{"location":"eastus"}`), http.StatusCreated)
	otherPort := otherSub + "/resourceGroups/rg-serial/providers/Microsoft.Compute/virtualMachines/vm1/providers/Microsoft.SerialConsole/serialPorts/0"
	azureResourcesExpectStatus(t, do(http.MethodPut, otherPort+azureSerialTestAPIVersion, `{"properties":{"state":"enabled"}}`), http.StatusCreated)
	azureResourcesExpectStatus(t, do(http.MethodPost, service+"/disableConsole"+azureSerialTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectStatus(t, do(http.MethodPost, otherPort+"/connect"+azureSerialTestAPIVersion, ""), http.StatusOK)
	azureResourcesExpectErrorCode(t, do(http.MethodPost, sub+"/resourceGroups/rg-serial/providers/Microsoft.SerialConsole/consoleServices/default/disableConsole"+azureSerialTestAPIVersion, ""), http.StatusBadRequest, "InvalidScope")
	azureResourcesExpectStatus(t, do(http.MethodPut, service+azureSerialTestAPIVersion, `{}`), http.StatusMethodNotAllowed)
	azureResourcesExpectStatus(t, do(http.MethodGet, sub+"/providers/Microsoft.SerialConsole/consoleServices/other"+azureSerialTestAPIVersion, ""), http.StatusNotFound)
}
