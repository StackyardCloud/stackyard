package server

import (
	"net/http"
	"strings"
	"testing"
)

// deployed creates and deploys cluster c1, returning its URL.
func (h *azureNCTest) deployed(body map[string]any) string {
	h.t.Helper()
	cluster := h.base + "clusters/c1"
	h.call(http.MethodPut, cluster, body, http.StatusCreated)
	h.call(http.MethodPost, cluster+"/deploy", nil, http.StatusOK)
	return cluster
}

func TestAzureNetworkCloudMachineActions(t *testing.T) {
	t.Parallel()
	h := newAzureNCTest(t)
	body := h.cluster(azureNCTestLocation("cm-cl"))
	cluster := h.base + "clusters/c1"
	h.call(http.MethodPut, cluster, body, http.StatusCreated)
	machine := h.base + "bareMetalMachines/bmm1"
	h.fails(http.MethodPost, machine+"/powerOff", nil, http.StatusConflict, "once its cluster is deployed")
	h.call(http.MethodPost, cluster+"/deploy", nil, http.StatusOK)

	h.rejects(http.MethodPost, machine+"/powerOff", map[string]any{"skipShutdown": "Yes"}, "skipShutdown")
	h.call(http.MethodPost, machine+"/powerOff", map[string]any{"skipShutdown": "True"}, http.StatusOK)
	if got := h.props(http.MethodGet, machine, nil, http.StatusOK); got["powerState"] != "Off" || got["readyState"] != "False" {
		t.Fatalf("powered off = %#v", got)
	}
	h.fails(http.MethodPost, machine+"/powerOff", nil, http.StatusConflict, "needs On")
	h.fails(http.MethodPost, machine+"/restart", nil, http.StatusConflict, "needs On")
	h.call(http.MethodPost, machine+"/start", nil, http.StatusOK)
	h.fails(http.MethodPost, machine+"/start", nil, http.StatusConflict, "needs Off")
	h.call(http.MethodPost, machine+"/restart", nil, http.StatusOK)
	h.rejects(http.MethodPost, machine+"/cordon", map[string]any{"evacuate": true}, "evacuate")
	h.call(http.MethodPost, machine+"/cordon", map[string]any{"evacuate": "True"}, http.StatusOK)
	if h.props(http.MethodGet, machine, nil, http.StatusOK)["cordonStatus"] != "Cordoned" {
		t.Fatal("cordon sets cordonStatus")
	}
	h.call(http.MethodPost, machine+"/uncordon", nil, http.StatusOK)
	h.rejects(http.MethodPost, machine+"/reimage", map[string]any{"safeguardMode": "Some"}, "safeguardMode")
	h.call(http.MethodPost, machine+"/reimage", map[string]any{"safeguardMode": "None"}, http.StatusOK)

	h.rejects(http.MethodPost, machine+"/replace", map[string]any{"bootMacAddress": "zz"}, "bootMacAddress")
	h.rejects(http.MethodPost, machine+"/replace", map[string]any{"storagePolicy": "Keep"}, "storagePolicy")
	h.rejects(http.MethodPost, machine+"/replace", map[string]any{"bmcCredentials": map[string]any{"username": "u"}}, "bmcCredentials.password is required")
	h.call(http.MethodPost, machine+"/replace", map[string]any{"serialNumber": "NEW1", "bootMacAddress": "00:11:22:33:44:55",
		"bmcCredentials": map[string]any{"username": "u2", "password": "p"}}, http.StatusOK)
	got := h.props(http.MethodGet, machine, nil, http.StatusOK)
	if got["serialNumber"] != "NEW1" || got["bootMacAddress"] != "00:11:22:33:44:55" || azureARMObject(got["bmcCredentials"])["username"] != "u2" || azureARMObject(got["bmcCredentials"])["password"] != nil {
		t.Fatalf("replaced = %#v", got)
	}
	if states := azureARMSlice(got["actionStates"]); len(states) != 7 {
		t.Fatalf("one state per action type run: %#v", states)
	}

	h.rejects(http.MethodPost, machine+"/runCommand", map[string]any{"script": "not base64!", "limitTimeSeconds": 60}, "base64")
	h.rejects(http.MethodPost, machine+"/runCommand", map[string]any{"script": "cHdkCg=="}, "limitTimeSeconds is required")
	h.rejects(http.MethodPost, machine+"/runCommand", map[string]any{"script": "cHdkCg==", "limitTimeSeconds": 59}, "limitTimeSeconds")
	run := h.call(http.MethodPost, machine+"/runCommand", map[string]any{"script": "cHdkCg==", "limitTimeSeconds": 60, "arguments": []any{"-a"}}, http.StatusOK)
	if out := azureARMObject(run["properties"]); out["exitCode"] != "0" || !strings.HasPrefix(azureARMString(out["resultUrl"]), "https://") {
		t.Fatalf("run command = %#v", run)
	}
	h.rejects(http.MethodPost, machine+"/runReadCommands", map[string]any{"commands": []any{}, "limitTimeSeconds": 60}, "at least one command")
	h.rejects(http.MethodPost, machine+"/runReadCommands", map[string]any{"commands": []any{map[string]any{"arguments": []any{"x"}}}, "limitTimeSeconds": 60}, "commands[0].command is required")
	h.call(http.MethodPost, machine+"/runReadCommands", map[string]any{"commands": []any{map[string]any{"command": "hostname"}}, "limitTimeSeconds": 60}, http.StatusOK)
	h.call(http.MethodPost, machine+"/runDataExtracts", map[string]any{"commands": []any{map[string]any{"command": "hardware-support-data-collection"}}, "limitTimeSeconds": 600}, http.StatusOK)
	h.rejects(http.MethodPost, machine+"/runDataExtractsRestricted", map[string]any{"commands": []any{map[string]any{"command": "x"}}, "limitTimeSeconds": 14401}, "limitTimeSeconds")

	h.rejects(http.MethodPatch, machine, map[string]any{"properties": map[string]any{"rackSlot": 4}}, "can't be updated")
	if h.props(http.MethodPatch, machine, map[string]any{"properties": map[string]any{"machineDetails": "rack 1 top"}}, http.StatusOK)["machineDetails"] != "rack 1 top" {
		t.Fatal("PATCH sets machineDetails")
	}
	rack := h.base + "racks/c1-rack1"
	h.rejects(http.MethodPatch, rack, map[string]any{"properties": map[string]any{"rackSerialNumber": ""}}, "rackSerialNumber")
	if h.props(http.MethodPatch, rack, map[string]any{"properties": map[string]any{"rackLocation": "Aisle 2"}}, http.StatusOK)["rackLocation"] != "Aisle 2" {
		t.Fatal("PATCH sets rackLocation")
	}
}

func TestAzureNetworkCloudCommandOutput(t *testing.T) {
	t.Parallel()
	h := newAzureNCTest(t)
	body := h.cluster(azureNCTestLocation("cm-cl"))
	azureARMObject(body["properties"])["commandOutputSettings"] = map[string]any{
		"containerUrl": "https://acct.blob.core.windows.net/all",
		"overrides":    []any{map[string]any{"commandOutputType": "StorageRunReadCommands", "containerUrl": "https://acct.blob.core.windows.net/storage"}},
	}
	h.deployed(body)
	machine := h.call(http.MethodPost, h.base+"bareMetalMachines/bmm1/runCommand", map[string]any{"script": "cHdkCg==", "limitTimeSeconds": 60}, http.StatusOK)
	if url := azureARMString(azureARMObject(machine["properties"])["resultUrl"]); !strings.HasPrefix(url, "https://acct.blob.core.windows.net/all/") {
		t.Fatalf("machine output went to %s", url)
	}
	appliance := h.base + "storageAppliances/sa1"
	storage := h.call(http.MethodPost, appliance+"/runReadCommands", map[string]any{"commands": []any{map[string]any{"command": "purehw", "arguments": []any{"list"}}}, "limitTimeSeconds": 60}, http.StatusOK)
	if url := azureARMString(azureARMObject(storage["properties"])["resultUrl"]); !strings.HasPrefix(url, "https://acct.blob.core.windows.net/storage/") {
		t.Fatalf("storage output went to %s", url)
	}

	h.rejects(http.MethodPost, appliance+"/enableRemoteVendorManagement", map[string]any{"supportEndpoints": []any{"vendor.example.com"}}, "supportEndpoints")
	h.call(http.MethodPost, appliance+"/enableRemoteVendorManagement", map[string]any{"supportEndpoints": []any{"10.0.0.0/24", "10.1.0.5"}}, http.StatusOK)
	if h.props(http.MethodGet, appliance, nil, http.StatusOK)["remoteVendorManagementStatus"] != "Enabled" {
		t.Fatal("remote vendor management is enabled")
	}
	h.call(http.MethodPost, appliance+"/disableRemoteVendorManagement", nil, http.StatusOK)
	got := h.props(http.MethodGet, appliance, nil, http.StatusOK)
	if got["remoteVendorManagementStatus"] != "Disabled" || got["actionStates"] != nil {
		t.Fatalf("disabled = %#v", got)
	}
	h.rejects(http.MethodPatch, appliance, map[string]any{"properties": map[string]any{"rackSlot": 1}}, "can't be updated")
	h.props(http.MethodPatch, appliance, map[string]any{"properties": map[string]any{"serialNumber": "SA2"}}, http.StatusOK)
}

func TestAzureNetworkCloudAccessBridge(t *testing.T) {
	t.Parallel()
	h := newAzureNCTest(t)
	bridge := h.base + "accessBridges/ab1"
	rule := map[string]any{"direction": "Outbound", "port": "24562-24570", "ipv4Addresses": []any{"10.10.20.10-10.10.20.20", "10.0.0.0/8"}, "ipv6Addresses": []any{"2001:db8::1"}}
	body := map[string]any{"location": "eastus", "extendedLocation": azureNCTestLocation("c1-cluster-cl"), "properties": map[string]any{
		"networkId":           "/subscriptions/s/providers/Microsoft.ManagedNetworkFabric/l3IsolationDomains/l3/internalNetworks/n1",
		"ipv4ConnectedPrefix": "198.51.100.0/24", "ipv6ConnectedPrefix": "2001:db8::/64", "securityRules": []any{rule},
	}}
	got := h.props(http.MethodPut, bridge, body, http.StatusCreated)
	endpoint := azureARMObject(azureARMSlice(got["endpoints"])[0])
	if got["detailedStatus"] != "Running" || endpoint["ipv4Address"] != "198.51.100.10" || endpoint["ipv6Address"] != "2001:db8::a" {
		t.Fatalf("bridge = %#v", got)
	}
	bad := func(key string, value any) map[string]any {
		return map[string]any{"location": "eastus", "extendedLocation": body["extendedLocation"], "properties": azureARMMerge(azureARMObject(body["properties"]), map[string]any{key: value})}
	}
	h.rejects(http.MethodPut, h.base+"accessBridges/ab2", bad("networkId", "net"), "networkId must be")
	h.rejects(http.MethodPut, h.base+"accessBridges/ab2", bad("ipv4ConnectedPrefix", "2001:db8::/64"), "IPv4 CIDR")
	h.rejects(http.MethodPut, h.base+"accessBridges/ab2", bad("ipv6ConnectedPrefix", "10.0.0.0/8"), "IPv6 CIDR")
	for want, broken := range map[string]map[string]any{
		"direction must be one of": {"direction": "Both", "port": "22"},
		"port is required":         {"direction": "Inbound"},
		"port must be a port":      {"direction": "Inbound", "port": "ssh"},
		"start isn't after":        {"direction": "Inbound", "port": "30-20"},
		"ipv4Addresses holds":      {"direction": "Inbound", "port": "22", "ipv4Addresses": []any{"10.0.0.9-10.0.0.1"}},
		"ipv6Addresses holds":      {"direction": "Inbound", "port": "22", "ipv6Addresses": []any{"10.0.0.1"}},
	} {
		h.rejects(http.MethodPut, h.base+"accessBridges/ab2", bad("securityRules", []any{broken}), want)
	}
	h.rejects(http.MethodPut, bridge, bad("ipv4ConnectedPrefix", "203.0.113.0/24"), "ipv4ConnectedPrefix can't be changed")
	h.rejects(http.MethodPatch, bridge, map[string]any{"properties": map[string]any{"networkId": "x"}}, "can't be updated")
	patched := h.props(http.MethodPatch, bridge, map[string]any{"properties": map[string]any{"securityRules": []any{map[string]any{"direction": "Inbound", "port": "443"}}}}, http.StatusOK)
	if rules := azureARMSlice(patched["securityRules"]); len(rules) != 1 || azureARMObject(rules[0])["port"] != "443" {
		t.Fatalf("patched rules = %#v", patched["securityRules"])
	}
	h.call(http.MethodDelete, bridge, nil, http.StatusOK)
}

func TestAzureNetworkCloudStandaloneHardware(t *testing.T) {
	t.Parallel()
	h := newAzureNCTest(t)
	loc := azureNCTestLocation("c1-cluster-cl")
	rack := h.base + "racks/r1"
	rackBody := map[string]any{"location": "eastus", "extendedLocation": loc, "properties": map[string]any{
		"availabilityZone": "1", "rackLocation": "Lab", "rackSerialNumber": "R1", "rackSkuId": h.skuID("NearEdge_Compute_R750"),
	}}
	h.rejects(http.MethodPut, rack, map[string]any{"location": "eastus", "extendedLocation": loc, "properties": map[string]any{"availabilityZone": "1", "rackLocation": "Lab", "rackSkuId": h.skuID("NearEdge_Compute_R750")}}, "rackSerialNumber is required")
	h.rejects(http.MethodPut, rack, azureARMMerge(rackBody, map[string]any{"properties": map[string]any{"rackSkuId": h.skuID("Nope")}}), "rack SKU of the catalog")
	got := h.props(http.MethodPut, rack, rackBody, http.StatusCreated)
	if got["detailedStatus"] != "Available" || got["clusterId"] != nil {
		t.Fatalf("standalone rack = %#v", got)
	}

	machine := h.base + "bareMetalMachines/solo1"
	machineProps := azureARMMerge(azureNCTestMachine("solo1", 4), map[string]any{
		"bmcConnectionString": "redfish+https://10.0.0.9/redfish/v1/Systems/1", "machineSkuId": "R750", "rackId": strings.TrimPrefix(rack, "/azure"),
	})
	h.rejects(http.MethodPut, machine, map[string]any{"location": "eastus", "extendedLocation": loc, "properties": azureARMMerge(machineProps, map[string]any{"rackId": "/subscriptions/s/x"})}, "rackId must be the resource ID of a rack")
	delete(machineProps, "bmcConnectionString")
	h.rejects(http.MethodPut, machine, map[string]any{"location": "eastus", "extendedLocation": loc, "properties": machineProps}, "bmcConnectionString is required")
	machineProps["bmcConnectionString"] = "redfish+https://10.0.0.9/redfish/v1/Systems/1"
	out := h.call(http.MethodPut, machine, map[string]any{"location": "eastus", "extendedLocation": loc, "properties": machineProps}, http.StatusCreated)
	if mp := azureARMObject(out["properties"]); mp["detailedStatus"] != "Provisioned" || azureARMObject(mp["bmcCredentials"])["password"] != nil {
		t.Fatalf("standalone machine = %#v", mp)
	}
	h.call(http.MethodPost, machine+"/powerOff", nil, http.StatusOK)
	h.call(http.MethodPut, machine, map[string]any{"location": "eastus", "extendedLocation": loc, "properties": azureARMMerge(machineProps, map[string]any{"powerState": "On"})}, http.StatusOK)
	if h.props(http.MethodGet, machine, nil, http.StatusOK)["powerState"] != "Off" {
		t.Fatal("a PUT can't set the server-owned powerState")
	}

	appliance := h.base + "storageAppliances/sa9"
	applianceProps := map[string]any{"rackId": strings.TrimPrefix(rack, "/azure"), "rackSlot": 2, "serialNumber": "SA9", "storageApplianceSkuId": "x70r3-9",
		"administratorCredentials": map[string]any{"username": "admin", "password": "pw"}}
	h.rejects(http.MethodPut, appliance, map[string]any{"location": "eastus", "extendedLocation": loc, "properties": azureARMMerge(applianceProps, map[string]any{"administratorCredentials": map[string]any{"username": "admin", "password": ""}})}, "password is required")
	h.call(http.MethodPut, appliance, map[string]any{"location": "eastus", "extendedLocation": loc, "properties": applianceProps}, http.StatusCreated)
	h.rejects(http.MethodPut, appliance, map[string]any{"location": "eastus", "extendedLocation": loc, "properties": azureARMMerge(applianceProps, map[string]any{"rackSlot": 3})}, "rackSlot can't be changed")
	h.call(http.MethodPost, appliance+"/enableRemoteVendorManagement", nil, http.StatusOK)

	// A cluster can't take a standalone name; standalone records delete.
	body := h.cluster(azureNCTestLocation("cm-cl"))
	azureARMObject(azureARMObject(body["properties"])["aggregatorOrSingleRackDefinition"])["bareMetalMachineConfigurationData"] = []any{azureNCTestMachine("solo1", 3)}
	h.rejects(http.MethodPut, h.base+"clusters/c1", body, "already exists outside any cluster")
	for _, url := range []string{machine, appliance, rack} {
		h.call(http.MethodDelete, url, nil, http.StatusOK)
	}

	// A PUT on a cluster's machine keeps its cluster and state.
	h.deployed(h.cluster(azureNCTestLocation("cm-cl")))
	member := h.call(http.MethodGet, h.base+"bareMetalMachines/bmm1", nil, http.StatusOK)
	props := azureARMMerge(azureARMObject(member["properties"]), map[string]any{"bmcCredentials": map[string]any{"username": "root", "password": "pw"}, "machineDetails": "new"})
	got = h.props(http.MethodPut, h.base+"bareMetalMachines/bmm1", map[string]any{"location": "eastus", "extendedLocation": member["extendedLocation"], "properties": props}, http.StatusOK)
	if got["clusterId"] == nil || got["powerState"] != "On" || got["machineDetails"] != "new" {
		t.Fatalf("PUT on a cluster's machine = %#v", got)
	}
}
