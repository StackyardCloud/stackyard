package server

import (
	"net/http"
	"strings"
	"testing"
)

func azureNCWVM(h *azureNCWTest, attachments ...any) map[string]any {
	return azureNCWBody(map[string]any{
		"adminUsername":                  "azureuser",
		"cloudServicesNetworkAttachment": map[string]any{"attachedNetworkId": h.arm + "cloudServicesNetworks/csn", "ipAllocationMethod": "Dynamic"},
		"cpuCores":                       2, "memorySizeGB": 8, "vmImage": "registry.example.com/vm:1",
		"networkAttachments":           attachments,
		"storageProfile":               map[string]any{"osDisk": map[string]any{"diskSizeGB": 120}, "volumeAttachments": []any{h.arm + "volumes/vol"}},
		"vmImageRepositoryCredentials": map[string]any{"registryUrl": "registry.example.com", "username": "u", "password": "secret"},
		"userDataContent":              "c2VjcmV0",
	})
}

func TestAzureNetworkCloudVirtualMachine(t *testing.T) {
	h := newAzureNCWTest(t)
	h.networks()
	l3 := map[string]any{"attachedNetworkId": h.arm + "l3Networks/l3", "ipAllocationMethod": "Static", "ipv4Address": "198.51.100.10", "ipv6Address": "2001:db8::10", "defaultGateway": "True"}
	l2 := map[string]any{"attachedNetworkId": h.arm + "l2Networks/l2", "ipAllocationMethod": "Disabled"}

	h.fails(http.MethodPut, "virtualMachines/vmbad", azureNCWVM(h, map[string]any{"attachedNetworkId": h.arm + "l2Networks/l2", "ipAllocationMethod": "Dynamic"}), http.StatusBadRequest, "must be Disabled")
	h.fails(http.MethodPut, "virtualMachines/vmbad", azureNCWVM(h, azureARMMerge(l3, map[string]any{"ipv4Address": "203.0.113.5"})), http.StatusBadRequest, "within 198.51.100.0/24")
	h.fails(http.MethodPut, "virtualMachines/vmbad", azureNCWVM(h, l3, azureARMMerge(l2, map[string]any{"defaultGateway": "True"})), http.StatusBadRequest, "defaultGateway")
	h.fails(http.MethodPut, "virtualMachines/vmbad", azureNCWVM(h, azureARMMerge(l2, map[string]any{"attachedNetworkId": h.arm + "l2Networks/missing"})), http.StatusBadRequest, "isn't an existing")
	h.fails(http.MethodPut, "virtualMachines/vm-bad", azureNCWVM(h), http.StatusBadRequest, "name")
	noCSN := azureNCWVM(h)
	azureARMObject(noCSN["properties"])["cloudServicesNetworkAttachment"] = map[string]any{"attachedNetworkId": h.arm + "cloudServicesNetworks/nope", "ipAllocationMethod": "Dynamic"}
	h.fails(http.MethodPut, "virtualMachines/vmbad", noCSN, http.StatusBadRequest, "existing cloud services network")
	h.fails(http.MethodPut, "virtualMachines/vmbad", azureARMMerge(azureNCWVM(h), map[string]any{"properties": map[string]any{"cpuCores": 1}}), http.StatusBadRequest, "cpuCores")

	vm := h.props(http.MethodPut, "virtualMachines/vm1", azureNCWVM(h, l3, l2), http.StatusCreated)
	if vm["powerState"] != "On" || vm["bootMethod"] != "UEFI" || len(azureARMSlice(vm["volumes"])) != 1 || vm["userDataContent"] != nil ||
		azureARMObject(vm["vmImageRepositoryCredentials"])["password"] != nil || azureARMObject(vm["cloudServicesNetworkAttachment"])["macAddress"] == nil {
		t.Fatalf("vm = %v", vm)
	}
	vmID := h.arm + "virtualMachines/vm1"
	if p := h.props(http.MethodGet, "l3Networks/l3", nil, http.StatusOK); azureARMSlice(p["virtualMachinesAssociatedIds"])[0] != vmID {
		t.Fatalf("l3 associations = %v", p)
	}
	if p := h.props(http.MethodGet, "volumes/vol", nil, http.StatusOK); len(azureARMSlice(p["attachedTo"])) != 1 {
		t.Fatalf("volume attachedTo = %v", p)
	}
	// Replacing a network in use keeps its usage lists.
	h.call(http.MethodPut, "l3Networks/l3", azureNCWBody(map[string]any{"l3IsolationDomainId": azureNCWTestDomain, "vlan": 100, "ipv4ConnectedPrefix": "198.51.100.0/24", "ipv6ConnectedPrefix": "2001:db8::/64"}), http.StatusOK)
	h.fails(http.MethodDelete, "l3Networks/l3", nil, http.StatusConflict, "in use by "+vmID)
	missingVolume := azureNCWVM(h)
	azureARMObject(azureARMObject(missingVolume["properties"])["storageProfile"])["volumeAttachments"] = []any{h.arm + "volumes/missing"}
	h.fails(http.MethodPut, "virtualMachines/vm2", missingVolume, http.StatusBadRequest, "doesn't exist")
	h.fails(http.MethodDelete, "volumes/vol", nil, http.StatusConflict, "in use")

	// The same PUT again is fine; changing a create-only property isn't.
	h.call(http.MethodPut, "virtualMachines/vm1", azureNCWVM(h, l3, l2), http.StatusOK)
	h.fails(http.MethodPut, "virtualMachines/vm1", azureARMMerge(azureNCWVM(h, l3, l2), map[string]any{"properties": map[string]any{"cpuCores": 4}}), http.StatusBadRequest, "cpuCores can't be changed")
	h.call(http.MethodPatch, "virtualMachines/vm1", map[string]any{"properties": map[string]any{"vmImageRepositoryCredentials": map[string]any{"password": "new"}}}, http.StatusOK)
	h.fails(http.MethodPatch, "virtualMachines/vm1", map[string]any{"properties": map[string]any{"cpuCores": 4}}, http.StatusBadRequest, "can't be updated")

	h.fails(http.MethodPost, "virtualMachines/vm1/powerOff", map[string]any{"skipShutdown": "Maybe"}, http.StatusBadRequest, "skipShutdown")
	h.call(http.MethodPost, "virtualMachines/vm1/powerOff", map[string]any{"skipShutdown": "True"}, http.StatusAccepted)
	if p := h.props(http.MethodGet, "virtualMachines/vm1", nil, http.StatusOK); p["powerState"] != "Off" || p["detailedStatus"] != "Stopped" {
		t.Fatalf("powered off vm = %v", p)
	}
	h.fails(http.MethodPost, "virtualMachines/vm1/restart", nil, http.StatusConflict, "powered off")
	if out := h.call(http.MethodPost, "virtualMachines/vm1/restart", nil, http.StatusConflict); azureARMObject(out["error"])["code"] != "Conflict" {
		t.Fatalf("restart error = %v", out)
	}
	h.call(http.MethodPost, "virtualMachines/vm1/start", nil, http.StatusAccepted)
	if p := h.props(http.MethodGet, "virtualMachines/vm1", nil, http.StatusOK); p["powerState"] != "On" || p["detailedStatus"] != "Running" {
		t.Fatalf("started vm = %v", p)
	}
	h.call(http.MethodPost, "virtualMachines/vm1/restart", nil, http.StatusAccepted)
	h.call(http.MethodPost, "virtualMachines/vm1/reimage", nil, http.StatusAccepted)
	h.fails(http.MethodPost, "virtualMachines/vm1/assignRelay", map[string]any{"machineId": h.arm + "virtualMachines/vm1"}, http.StatusBadRequest, "HybridCompute")
	h.call(http.MethodPost, "virtualMachines/vm1/assignRelay", map[string]any{"machineId": "/subscriptions/s/resourceGroups/r/providers/Microsoft.HybridCompute/machines/m", "relayType": "Public"}, http.StatusAccepted)

	console := azureNCWBody(map[string]any{"enabled": "True", "sshPublicKey": map[string]any{"keyData": "ssh-rsa AAAA"}, "expiration": "2030-01-01T00:00:00Z"})
	h.fails(http.MethodPut, "virtualMachines/vm1/consoles/other", console, http.StatusBadRequest, "name")
	h.fails(http.MethodPut, "virtualMachines/vm1/consoles/default", azureNCWBody(map[string]any{"enabled": "Yes", "sshPublicKey": map[string]any{"keyData": "k"}}), http.StatusBadRequest, "enabled")
	if c := h.props(http.MethodPut, "virtualMachines/vm1/consoles/default", console, http.StatusCreated); c["detailedStatus"] != "Ready" || c["virtualMachineAccessId"] == "" {
		t.Fatalf("console = %v", c)
	}
	if c := h.props(http.MethodPatch, "virtualMachines/vm1/consoles/default", map[string]any{"properties": map[string]any{"enabled": "False"}}, http.StatusOK); c["enabled"] != "False" {
		t.Fatalf("patched console = %v", c)
	}

	h.call(http.MethodDelete, "virtualMachines/vm1", nil, http.StatusOK)
	h.call(http.MethodGet, "virtualMachines/vm1/consoles/default", nil, http.StatusNotFound)
	if p := h.props(http.MethodGet, "l3Networks/l3", nil, http.StatusOK); len(azureARMSlice(p["associatedResourceIds"])) != 0 {
		t.Fatalf("l3 still associated: %v", p)
	}
	if p := h.props(http.MethodGet, "volumes/vol", nil, http.StatusOK); len(azureARMSlice(p["attachedTo"])) != 0 {
		t.Fatalf("volume still attached: %v", p)
	}
	h.call(http.MethodDelete, "l3Networks/l3", nil, http.StatusOK)
	h.call(http.MethodDelete, "volumes/vol", nil, http.StatusOK)
}

// Usage lists stay in a stable order, so a write to one user leaves the
// network's etag alone, and users a resource group delete removed don't
// keep the network in use.
func TestAzureNetworkCloudUsageLists(t *testing.T) {
	h := newAzureNCWTest(t)
	h.networks()
	l2 := map[string]any{"attachedNetworkId": h.arm + "l2Networks/l2", "ipAllocationMethod": "Disabled"}
	h.call(http.MethodPut, "virtualMachines/vm1", azureNCWVM(h, l2), http.StatusCreated)
	h.call(http.MethodPut, "virtualMachines/vm2", azureNCWVM(h, l2), http.StatusCreated)
	before := h.call(http.MethodGet, "l2Networks/l2", nil, http.StatusOK)
	h.call(http.MethodPatch, "virtualMachines/vm1", map[string]any{"tags": map[string]any{"a": "b"}}, http.StatusOK)
	after := h.call(http.MethodGet, "l2Networks/l2", nil, http.StatusOK)
	if before["etag"] != after["etag"] || len(azureARMSlice(azureARMObject(after["properties"])["virtualMachinesAssociatedIds"])) != 2 {
		t.Fatalf("l2 changed by an unrelated write: %v -> %v", before["etag"], after)
	}

	// A VM in another resource group, deleted with that group.
	sub := strings.TrimSuffix(h.base, "/resourceGroups/rg-nc/providers/Microsoft.NetworkCloud/")
	other := &azureNCWTest{t: t, base: sub + "/resourceGroups/rg-other/providers/Microsoft.NetworkCloud/", do: h.do}
	other.arm = h.arm
	mustBody(t, h.do(http.MethodPut, sub+"/resourcegroups/rg-other?api-version=2021-04-01", `{"location":"eastus"}`))
	l3 := map[string]any{"attachedNetworkId": h.arm + "l3Networks/l3", "ipAllocationMethod": "Disabled"}
	other.call(http.MethodPut, "virtualMachines/vm3", azureNCWVM(h, l3), http.StatusCreated)
	h.fails(http.MethodDelete, "l3Networks/l3", nil, http.StatusConflict, "in use")
	if resp := h.do(http.MethodDelete, sub+"/resourcegroups/rg-other?api-version=2021-04-01", ""); resp.StatusCode >= 300 {
		t.Fatalf("delete resource group = %d: %s", resp.StatusCode, mustBody(t, resp))
	}
	h.call(http.MethodDelete, "l3Networks/l3", nil, http.StatusOK)
}
