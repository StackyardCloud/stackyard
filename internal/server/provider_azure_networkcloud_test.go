package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

const azureNCTestAPI = "?api-version=2026-07-01"

type azureNCTest struct {
	t    *testing.T
	sub  string // /azure/subscriptions/{id}
	base string // the resource group's Microsoft.NetworkCloud path
	arm  string // base without /azure, as resource IDs have it
	do   func(method, path, body string) *http.Response
}

func newAzureNCTest(t *testing.T) *azureNCTest {
	t.Helper()
	sub, do := newAzureDatabaseTestServer(t, "rg-nc")
	base := sub + "/resourceGroups/rg-nc/providers/Microsoft.NetworkCloud/"
	return &azureNCTest{t: t, sub: sub, base: base, arm: strings.TrimPrefix(base, "/azure"), do: do}
}

func (h *azureNCTest) call(method, path string, body any, want int) map[string]any {
	h.t.Helper()
	text := ""
	if body != nil {
		raw, _ := json.Marshal(body)
		text = string(raw)
	}
	resp := h.do(method, path+azureNCTestAPI, text)
	raw := mustBody(h.t, resp)
	if resp.StatusCode != want {
		h.t.Fatalf("%s %s = %d, want %d: %s", method, path, resp.StatusCode, want, raw)
	}
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func (h *azureNCTest) props(method, path string, body any, want int) map[string]any {
	h.t.Helper()
	return azureARMObject(h.call(method, path, body, want)["properties"])
}

func (h *azureNCTest) fails(method, path string, body any, status int, text string) {
	h.t.Helper()
	out := h.call(method, path, body, status)
	if msg := azureARMString(azureARMObject(out["error"])["message"]); !strings.Contains(msg, text) {
		h.t.Fatalf("%s %s error = %q, want it to mention %q", method, path, msg, text)
	}
}

func (h *azureNCTest) rejects(method, path string, body any, text string) {
	h.t.Helper()
	h.fails(method, path, body, http.StatusBadRequest, text)
}

func (h *azureNCTest) skuID(name string) string {
	return strings.TrimPrefix(h.sub, "/azure") + "/providers/Microsoft.NetworkCloud/rackSkus/" + name
}

func azureNCTestLocation(name string) map[string]any {
	return map[string]any{"type": "CustomLocation", "name": "/subscriptions/s/resourceGroups/rg-nc/providers/Microsoft.ExtendedLocation/customLocations/" + name}
}

func azureNCTestMachine(name string, slot int) map[string]any {
	return map[string]any{
		"machineName": name, "rackSlot": slot, "serialNumber": "SN" + name, "machineDetails": "details",
		"bmcMacAddress": "AA:BB:CC:DD:EE:0" + string(rune('0'+slot%10)), "bootMacAddress": "00:BB:CC:DD:EE:0" + string(rune('0'+slot%10)),
		"bmcCredentials": map[string]any{"username": "root", "password": "bmc-secret"},
	}
}

// cluster returns a single rack cluster body with two machines and an
// appliance on the extended location loc.
func (h *azureNCTest) cluster(loc map[string]any) map[string]any {
	return map[string]any{
		"location": "eastus", "extendedLocation": loc,
		"properties": map[string]any{
			"clusterType": "SingleRack", "clusterVersion": "4.1.0",
			"networkFabricId": "/subscriptions/s/resourceGroups/rg/providers/Microsoft.ManagedNetworkFabric/networkFabrics/fab",
			"aggregatorOrSingleRackDefinition": map[string]any{
				"networkRackId":    "/subscriptions/s/resourceGroups/rg/providers/Microsoft.ManagedNetworkFabric/networkRacks/r1",
				"rackSkuId":        h.skuID("NearEdge_Single_R750"),
				"rackSerialNumber": "RACK1", "rackLocation": "Floor 3",
				"bareMetalMachineConfigurationData": []any{azureNCTestMachine("bmm1", 1), azureNCTestMachine("bmm2", 3)},
				"storageApplianceConfigurationData": []any{map[string]any{
					"rackSlot": 11, "serialNumber": "SA1", "storageApplianceName": "sa1",
					"adminCredentials": map[string]any{"username": "admin", "password": "sa-secret"},
				}},
			},
			"clusterServicePrincipal": map[string]any{"applicationId": "app", "password": "sp-secret", "principalId": "p", "tenantId": "t"},
		},
	}
}

func TestAzureNetworkCloudClusterManager(t *testing.T) {
	t.Parallel()
	h := newAzureNCTest(t)
	manager := h.base + "clusterManagers/cm1"
	body := map[string]any{"location": "eastus", "identity": map[string]any{"type": "SystemAssigned"}, "properties": map[string]any{
		"fabricControllerId": "/subscriptions/s/resourceGroups/rg/providers/Microsoft.ManagedNetworkFabric/networkFabricControllers/nfc",
		"vmSize":             "Standard_D8s_v3",
	}}
	out := h.call(http.MethodPut, manager, body, http.StatusCreated)
	props := azureARMObject(out["properties"])
	if props["detailedStatus"] != "Available" || len(azureARMSlice(props["clusterVersions"])) != len(azureNCVersions) {
		t.Fatalf("cluster manager = %#v", props)
	}
	if azureARMObject(out["identity"])["principalId"] == nil || !strings.HasSuffix(azureARMString(azureARMObject(props["managerExtendedLocation"])["name"]), "/customLocations/cm1-cl") {
		t.Fatalf("identity and extended location = %#v", out)
	}
	if !strings.HasPrefix(azureARMString(azureARMObject(props["managedResourceGroupConfiguration"])["name"]), "cm1-managed-") {
		t.Fatalf("managed resource group defaults: %#v", props["managedResourceGroupConfiguration"])
	}

	h.rejects(http.MethodPut, h.base+"clusterManagers/cm2", map[string]any{"location": "eastus", "properties": map[string]any{}}, "fabricControllerId is required")
	h.rejects(http.MethodPut, h.base+"clusterManagers/cm2", map[string]any{"location": "eastus", "properties": map[string]any{"fabricControllerId": "/subscriptions/s/x"}}, "network fabric controller")
	changed := map[string]any{"location": "eastus", "identity": body["identity"], "properties": map[string]any{"fabricControllerId": azureARMObject(body["properties"])["fabricControllerId"], "vmSize": "Standard_D16s_v3"}}
	h.rejects(http.MethodPut, manager, changed, "vmSize can't be changed")
	h.rejects(http.MethodPut, manager, azureARMMerge(body, map[string]any{"kind": "AzureLocal"}), "kind can't be changed")
	// A re-PUT that omits a create-only property keeps it.
	omitted := map[string]any{"location": "eastus", "identity": body["identity"], "properties": map[string]any{"fabricControllerId": azureARMObject(body["properties"])["fabricControllerId"]}}
	if got := h.props(http.MethodPut, manager, omitted, http.StatusOK); got["vmSize"] != "Standard_D8s_v3" {
		t.Fatalf("an omitted vmSize was dropped: %#v", got)
	}
	h.rejects(http.MethodPatch, manager, map[string]any{"properties": map[string]any{"vmSize": "x"}}, "can't be updated")
	if tags := h.call(http.MethodPatch, manager, map[string]any{"tags": map[string]any{"env": "lab"}}, http.StatusOK)["tags"]; azureARMObject(tags)["env"] != "lab" {
		t.Fatalf("tags = %#v", tags)
	}
	relay := manager + "/updateRelayPrivateEndpointConnection"
	h.rejects(http.MethodPost, relay, map[string]any{"connectionState": "Pending", "privateEndpointResourceId": "x"}, "connectionState")
	h.rejects(http.MethodPost, relay, map[string]any{"connectionState": "Approved", "privateEndpointResourceId": "/subscriptions/s/x"}, "private endpoint")
	done := h.call(http.MethodPost, relay, map[string]any{"connectionState": "Approved",
		"privateEndpointResourceId": "/subscriptions/s/resourceGroups/rg/providers/Microsoft.Network/privateEndpoints/pe"}, http.StatusOK)
	if done["status"] != "Succeeded" {
		t.Fatalf("relay connection = %#v", done)
	}

	// A cluster on the manager's extended location is managed by it, and
	// keeps the manager from being deleted.
	cluster := h.props(http.MethodPut, h.base+"clusters/c1", h.cluster(azureARMObject(props["managerExtendedLocation"])), http.StatusCreated)
	if !strings.EqualFold(azureARMString(cluster["clusterManagerId"]), strings.TrimPrefix(manager, "/azure")) {
		t.Fatalf("clusterManagerId = %v", cluster["clusterManagerId"])
	}
	h.fails(http.MethodDelete, manager, nil, http.StatusConflict, "still manages cluster")
	h.call(http.MethodDelete, h.base+"clusters/c1", nil, http.StatusOK)
	h.call(http.MethodDelete, manager, nil, http.StatusOK)

	// A cluster created before its manager, on the manager's future
	// extended location, still keeps the manager from being deleted.
	early := azureNCTestLocation("cm3-cl")
	early["name"] = strings.Replace(azureARMString(early["name"]), "/subscriptions/s/", "/subscriptions/"+strings.TrimPrefix(h.sub, "/azure/subscriptions/")+"/", 1)
	if got := h.props(http.MethodPut, h.base+"clusters/early", h.cluster(early), http.StatusCreated); got["clusterManagerId"] != "" {
		t.Fatalf("no manager yet: %v", got["clusterManagerId"])
	}
	h.call(http.MethodPut, h.base+"clusterManagers/cm3", body, http.StatusCreated)
	h.fails(http.MethodDelete, h.base+"clusterManagers/cm3", nil, http.StatusConflict, "still manages cluster")
}

func TestAzureNetworkCloudClusterLifecycle(t *testing.T) {
	t.Parallel()
	h := newAzureNCTest(t)
	cluster := h.base + "clusters/c1"
	out := h.call(http.MethodPut, cluster, h.cluster(azureNCTestLocation("cm-cl")), http.StatusCreated)
	props := azureARMObject(out["properties"])
	if props["detailedStatus"] != "PendingDeployment" || props["supportExpiryDate"] != "2026-12-31" || len(azureARMSlice(props["availableUpgradeVersions"])) != 2 {
		t.Fatalf("new cluster = %#v", props)
	}
	if capacity := azureARMObject(props["clusterCapacity"]); capacity["totalCoreCount"] != float64(2*azureNCCores) || capacity["totalApplianceStorageGB"] != float64(azureNCApplianceGB) {
		t.Fatalf("capacity = %#v", capacity)
	}
	raw, _ := json.Marshal(out)
	if strings.Contains(string(raw), "secret") {
		t.Fatalf("a password was returned: %s", raw)
	}
	if out["extendedLocation"] == nil || props["clusterManagerId"] != "" {
		t.Fatalf("extended location and manager = %#v", out)
	}

	// The rack definitions created a rack, two machines and an appliance.
	rack := h.props(http.MethodGet, h.base+"racks/c1-rack1", nil, http.StatusOK)
	machine := h.call(http.MethodGet, h.base+"bareMetalMachines/bmm1", nil, http.StatusOK)
	appliance := h.props(http.MethodGet, h.base+"storageAppliances/sa1", nil, http.StatusOK)
	mp := azureARMObject(machine["properties"])
	if rack["rackSerialNumber"] != "RACK1" || mp["detailedStatus"] != "Preparing" || mp["powerState"] != "Off" || appliance["detailedStatus"] != "Provisioning" {
		t.Fatalf("members = %#v %#v %#v", rack, mp, appliance)
	}
	if !strings.HasSuffix(azureARMString(azureARMObject(machine["extendedLocation"])["name"]), "/c1-cluster-cl") || mp["clusterId"] != strings.TrimPrefix(cluster, "/azure") {
		t.Fatalf("machine location and cluster = %#v", machine)
	}
	if azureARMSlice(mp["machineRoles"])[0] != "platform.afo-nc.microsoft.com/management-plane=true" {
		t.Fatalf("slot 1 is a control plane slot: %#v", mp["machineRoles"])
	}
	h.fails(http.MethodDelete, h.base+"bareMetalMachines/bmm1", nil, http.StatusConflict, "belongs to cluster")
	h.fails(http.MethodDelete, h.base+"racks/c1-rack1", nil, http.StatusConflict, "belongs to cluster")

	// A re-PUT resending part of the managed resource group is no change;
	// a different name is.
	group := azureARMObject(props["managedResourceGroupConfiguration"])
	partial := h.cluster(azureNCTestLocation("cm-cl"))
	azureARMObject(partial["properties"])["managedResourceGroupConfiguration"] = map[string]any{"name": group["name"]}
	h.call(http.MethodPut, cluster, partial, http.StatusOK)
	azureARMObject(partial["properties"])["managedResourceGroupConfiguration"] = map[string]any{"name": "other-rg"}
	h.rejects(http.MethodPut, cluster, partial, "managedResourceGroupConfiguration can't be changed")

	// Rack definitions change only by PUT, and only before deployment.
	moved := h.cluster(azureNCTestLocation("cm-cl"))
	azureARMObject(azureARMObject(moved["properties"])["aggregatorOrSingleRackDefinition"])["rackLocation"] = "Floor 4"
	h.fails(http.MethodPatch, cluster, map[string]any{"properties": map[string]any{"aggregatorOrSingleRackDefinition": map[string]any{"rackLocation": "Floor 4"}}}, http.StatusConflict, "only be changed with a PUT")
	h.fails(http.MethodPatch, cluster, map[string]any{"properties": map[string]any{"computeRackDefinitions": []any{azureARMObject(moved["properties"])["aggregatorOrSingleRackDefinition"]}}}, http.StatusConflict, "only be changed with a PUT")
	h.call(http.MethodPatch, cluster, map[string]any{"properties": map[string]any{"aggregatorOrSingleRackDefinition": map[string]any{"rackLocation": "Floor 3"}}}, http.StatusOK)
	// Resending the definitions unchanged, passwords included, is no change.
	unchanged := azureARMObject(h.cluster(azureNCTestLocation("cm-cl"))["properties"])["aggregatorOrSingleRackDefinition"]
	h.call(http.MethodPatch, cluster, map[string]any{"properties": map[string]any{"aggregatorOrSingleRackDefinition": unchanged}}, http.StatusOK)
	h.call(http.MethodPut, cluster, moved, http.StatusOK)
	if h.props(http.MethodGet, cluster, nil, http.StatusOK)["aggregatorOrSingleRackDefinition"].(map[string]any)["rackLocation"] != "Floor 4" {
		t.Fatal("a PUT before deployment changes the rack definitions")
	}
	h.call(http.MethodPut, cluster, h.cluster(azureNCTestLocation("cm-cl")), http.StatusOK)
	if got := h.call(http.MethodPatch, cluster, map[string]any{"tags": map[string]any{"a": "b"}}, http.StatusOK); azureARMObject(got["extendedLocation"])["name"] != azureNCTestLocation("cm-cl")["name"] {
		t.Fatalf("PATCH dropped extendedLocation: %#v", got)
	}

	// Actions on a deployed cluster wait for deployment.
	h.fails(http.MethodPost, cluster+"/scanRuntime", map[string]any{}, http.StatusConflict, "PendingDeployment")
	h.fails(http.MethodPost, cluster+"/updateVersion", map[string]any{"targetClusterVersion": "4.2.0"}, http.StatusConflict, "needs Running")
	h.rejects(http.MethodPost, cluster+"/deploy", map[string]any{"skipValidationsForMachines": []any{"nope"}}, "isn't a bare metal machine")
	h.call(http.MethodPost, cluster+"/deploy", map[string]any{"skipValidationsForMachines": []any{"bmm2"}}, http.StatusOK)
	props = h.props(http.MethodGet, cluster, nil, http.StatusOK)
	if props["detailedStatus"] != "Running" || azureARMObject(azureARMSlice(props["actionStates"])[0])["actionType"] != "Microsoft.NetworkCloud/clusters/deploy/action" {
		t.Fatalf("deployed cluster = %#v", props)
	}
	if mp := h.props(http.MethodGet, h.base+"bareMetalMachines/bmm2", nil, http.StatusOK); mp["detailedStatus"] != "Provisioned" || mp["powerState"] != "On" {
		t.Fatalf("deployed machine = %#v", mp)
	}
	if h.props(http.MethodGet, h.base+"storageAppliances/sa1", nil, http.StatusOK)["detailedStatus"] != "Available" {
		t.Fatal("the appliance is available after deployment")
	}
	h.fails(http.MethodPost, cluster+"/deploy", map[string]any{}, http.StatusConflict, "is Running")

	// A re-PUT keeps the deployment and member state; an empty compute
	// rack list is the same rack definitions as none.
	same := h.cluster(azureNCTestLocation("cm-cl"))
	azureARMObject(same["properties"])["computeRackDefinitions"] = []any{}
	h.props(http.MethodPut, cluster, same, http.StatusOK)
	if h.props(http.MethodGet, cluster, nil, http.StatusOK)["detailedStatus"] != "Running" || h.props(http.MethodGet, h.base+"bareMetalMachines/bmm1", nil, http.StatusOK)["powerState"] != "On" {
		t.Fatal("a re-PUT reset the deployed cluster")
	}

	// Version updates go to an available upgrade version only.
	h.rejects(http.MethodPost, cluster+"/updateVersion", map[string]any{}, "targetClusterVersion is required")
	h.rejects(http.MethodPost, cluster+"/updateVersion", map[string]any{"targetClusterVersion": "4.1.0"}, "isn't one of the cluster's availableUpgradeVersions")
	h.rejects(http.MethodPost, cluster+"/updateVersion", map[string]any{"targetClusterVersion": "9.9.9"}, "isn't one of the cluster's availableUpgradeVersions")
	moved = h.cluster(azureNCTestLocation("cm-cl"))
	azureARMObject(azureARMObject(moved["properties"])["aggregatorOrSingleRackDefinition"])["rackLocation"] = "Floor 9"
	h.rejects(http.MethodPut, cluster, moved, "can't change once the cluster leaves PendingDeployment")
	h.call(http.MethodPost, cluster+"/updateVersion", map[string]any{"targetClusterVersion": "4.2.0"}, http.StatusOK)
	props = h.props(http.MethodGet, cluster, nil, http.StatusOK)
	if props["clusterVersion"] != "4.2.0" || len(azureARMSlice(props["availableUpgradeVersions"])) != 1 || props["supportExpiryDate"] != "2027-06-30" || props["lastSuccessfulVersionUpdateTime"] == nil {
		t.Fatalf("updated cluster = %#v", props)
	}
	h.fails(http.MethodPost, cluster+"/continueUpdateVersion", map[string]any{}, http.StatusConflict, "needs UpdatePaused")
	stale := h.cluster(azureNCTestLocation("cm-cl"))
	h.rejects(http.MethodPut, cluster, stale, "only be changed with the updateVersion action")

	h.rejects(http.MethodPost, cluster+"/rotateCredential", map[string]any{"credentials": []any{}}, "at least one credential")
	h.rejects(http.MethodPost, cluster+"/rotateCredential", map[string]any{"credentials": []any{"Root"}}, "isn't one of the cluster's managedCredentials")
	h.call(http.MethodPost, cluster+"/rotateCredential", map[string]any{"credentials": []any{"BMC Credential"}}, http.StatusOK)
	h.rejects(http.MethodPost, cluster+"/inspect", map[string]any{"filterDevices": map[string]any{"rackNames": []any{"other"}}}, "isn't part of the cluster")
	h.rejects(http.MethodPost, cluster+"/inspect", map[string]any{"additionalActions": []any{"Reboot"}}, "ResetHardware")
	h.call(http.MethodPost, cluster+"/inspect", map[string]any{"additionalActions": []any{"ResetHardware"}, "filterDevices": map[string]any{"rackNames": []any{"c1-rack1"}, "bareMetalMachineNames": []any{"bmm1"}}}, http.StatusOK)
	h.rejects(http.MethodPost, cluster+"/scanRuntime", map[string]any{"scanActivity": "Now"}, "scanActivity")
	h.call(http.MethodPost, cluster+"/scanRuntime", map[string]any{"scanActivity": "Scan"}, http.StatusOK)
	if states := azureARMSlice(h.props(http.MethodGet, cluster, nil, http.StatusOK)["actionStates"]); len(states) != 5 {
		t.Fatalf("one action state per action type: %#v", states)
	}

	// PATCH changes settings; members go with the cluster.
	patched := h.props(http.MethodPatch, cluster, map[string]any{"properties": map[string]any{"clusterLocation": "Building 4", "updateStrategy": map[string]any{
		"strategyType": "PauseAfterRack", "thresholdType": "PercentSuccess", "thresholdValue": 80}}}, http.StatusOK)
	if patched["clusterLocation"] != "Building 4" || patched["detailedStatus"] != "Running" {
		t.Fatalf("patched = %#v", patched)
	}
	h.rejects(http.MethodPatch, cluster, map[string]any{"properties": map[string]any{"clusterType": "MultiRack"}}, "can't be updated")
	h.rejects(http.MethodPatch, cluster, map[string]any{"properties": map[string]any{"updateStrategy": map[string]any{"waitTimeMinutes": 61}}}, "waitTimeMinutes")
	h.call(http.MethodDelete, cluster, nil, http.StatusOK)
	h.call(http.MethodGet, h.base+"bareMetalMachines/bmm1", nil, http.StatusNotFound)
	h.call(http.MethodGet, h.base+"racks/c1-rack1", nil, http.StatusNotFound)
}

func TestAzureNetworkCloudMultiRackUpdate(t *testing.T) {
	t.Parallel()
	h := newAzureNCTest(t)
	body := h.cluster(azureNCTestLocation("cm-cl"))
	props := azureARMObject(body["properties"])
	props["clusterType"] = "MultiRack"
	aggregator := azureARMObject(props["aggregatorOrSingleRackDefinition"])
	h.rejects(http.MethodPut, h.base+"clusters/m1", body, "must be a Aggregator rack SKU")
	aggregator["rackSkuId"] = h.skuID("NearEdge_Aggregator_R750")
	h.rejects(http.MethodPut, h.base+"clusters/m1", body, "storageApplianceConfigurationData[0].rackSlot 11 isn't a slot")
	azureARMObject(azureARMSlice(aggregator["storageApplianceConfigurationData"])[0])["rackSlot"] = 12
	props["computeRackDefinitions"] = []any{map[string]any{
		"networkRackId": aggregator["networkRackId"], "rackSkuId": h.skuID("NearEdge_Compute_R750"), "rackSerialNumber": "RACK2",
		"bareMetalMachineConfigurationData": []any{azureNCTestMachine("cmp1", 5), azureNCTestMachine("cmp2", 5)},
	}}
	h.rejects(http.MethodPut, h.base+"clusters/m1", body, "is used twice in the rack")
	azureARMObject(azureARMSlice(azureARMObject(azureARMSlice(props["computeRackDefinitions"])[0])["bareMetalMachineConfigurationData"])[1])["rackSlot"] = 6
	props["updateStrategy"] = map[string]any{"strategyType": "PauseAfterRack", "thresholdType": "CountSuccess", "thresholdValue": 1}
	h.call(http.MethodPut, h.base+"clusters/m1", body, http.StatusCreated)
	if h.props(http.MethodGet, h.base+"racks/m1-rack2", nil, http.StatusOK)["rackSkuId"] != h.skuID("NearEdge_Compute_R750") {
		t.Fatal("the compute rack was created")
	}
	cluster := h.base + "clusters/m1"
	h.call(http.MethodPost, cluster+"/deploy", nil, http.StatusOK)
	h.call(http.MethodPost, cluster+"/updateVersion", map[string]any{"targetClusterVersion": "4.3.0", "safeguardMode": "All"}, http.StatusOK)
	if got := h.props(http.MethodGet, cluster, nil, http.StatusOK); got["detailedStatus"] != "UpdatePaused" || got["lastSuccessfulVersionUpdateTime"] != nil {
		t.Fatalf("a PauseAfterRack update pauses: %#v", got)
	}
	h.rejects(http.MethodPost, cluster+"/continueUpdateVersion", map[string]any{"machineGroupTargetingMode": "ByZone"}, "machineGroupTargetingMode")
	h.call(http.MethodPost, cluster+"/continueUpdateVersion", map[string]any{"machineGroupTargetingMode": "AlphaByRack"}, http.StatusOK)
	if got := h.props(http.MethodGet, cluster, nil, http.StatusOK); got["detailedStatus"] != "Running" || len(azureARMSlice(got["availableUpgradeVersions"])) != 0 {
		t.Fatalf("continued update = %#v", got)
	}
	// Another cluster can't take a machine name already in use.
	other := h.cluster(azureNCTestLocation("cm-cl"))
	azureARMObject(azureARMObject(other["properties"])["aggregatorOrSingleRackDefinition"])["bareMetalMachineConfigurationData"] = []any{azureNCTestMachine("cmp1", 3)}
	h.rejects(http.MethodPut, h.base+"clusters/c2", other, "already exists in cluster")
}

func TestAzureNetworkCloudClusterValidation(t *testing.T) {
	t.Parallel()
	h := newAzureNCTest(t)
	url := h.base + "clusters/bad"
	mutate := func(change func(body, props, rack map[string]any)) map[string]any {
		body := h.cluster(azureNCTestLocation("cm-cl"))
		props := azureARMObject(body["properties"])
		change(body, props, azureARMObject(props["aggregatorOrSingleRackDefinition"]))
		return body
	}
	for want, change := range map[string]func(body, props, rack map[string]any){
		"extendedLocation is required": func(body, _, _ map[string]any) { delete(body, "extendedLocation") },
		"extendedLocation.type": func(body, _, _ map[string]any) {
			body["extendedLocation"] = map[string]any{"type": "Region", "name": "x"}
		},
		"clusterVersion is required":          func(_, props, _ map[string]any) { delete(props, "clusterVersion") },
		"isn't a supported cluster version":   func(_, props, _ map[string]any) { props["clusterVersion"] = "9.9.9" },
		"clusterType must be one of":          func(_, props, _ map[string]any) { props["clusterType"] = "Huge" },
		"network fabric":                      func(_, props, _ map[string]any) { props["networkFabricId"] = "/subscriptions/s/x" },
		"aggregatorOrSingleRackDefinition is": func(_, props, _ map[string]any) { delete(props, "aggregatorOrSingleRackDefinition") },
		"rackSkuId must name a rack SKU":      func(_, _, rack map[string]any) { rack["rackSkuId"] = h.skuID("Other") },
		"rackSerialNumber is required":        func(_, _, rack map[string]any) { delete(rack, "rackSerialNumber") },
		"availabilityZone":                    func(_, _, rack map[string]any) { rack["availabilityZone"] = "zone-1" },
		"must be empty for a SingleRack":      func(_, props, rack map[string]any) { props["computeRackDefinitions"] = []any{rack} },
		"bmcMacAddress must be a MAC": func(_, _, rack map[string]any) {
			azureARMObject(azureARMSlice(rack["bareMetalMachineConfigurationData"])[0])["bmcMacAddress"] = "nope"
		},
		"bmcCredentials.password is required": func(_, _, rack map[string]any) {
			azureARMObject(azureARMSlice(rack["bareMetalMachineConfigurationData"])[0])["bmcCredentials"] = map[string]any{"username": "u"}
		},
		"rackSlot must be a whole number": func(_, _, rack map[string]any) {
			azureARMObject(azureARMSlice(rack["bareMetalMachineConfigurationData"])[0])["rackSlot"] = 0
		},
		"machineName must be 2 to 64": func(_, _, rack map[string]any) {
			azureARMObject(azureARMSlice(rack["bareMetalMachineConfigurationData"])[0])["machineName"] = "bad-name"
		},
		"rackSlot 11 isn't a slot": func(_, _, rack map[string]any) {
			azureARMObject(azureARMSlice(rack["bareMetalMachineConfigurationData"])[0])["rackSlot"] = 11
		},
		"adminCredentials.password is required": func(_, _, rack map[string]any) {
			azureARMObject(azureARMSlice(rack["storageApplianceConfigurationData"])[0])["adminCredentials"] = map[string]any{"username": "u"}
		},
		"clusterServicePrincipal.tenantId": func(_, props, _ map[string]any) { delete(azureARMObject(props["clusterServicePrincipal"]), "tenantId") },
		"computeDeploymentThreshold.value": func(_, props, _ map[string]any) {
			props["computeDeploymentThreshold"] = map[string]any{"type": "CountSuccess", "grouping": "PerRack"}
		},
		"secretArchive.keyVaultId is required": func(_, props, _ map[string]any) { props["secretArchive"] = map[string]any{"useKeyVault": "True"} },
		"enforcementLevel": func(_, props, _ map[string]any) {
			props["runtimeProtectionConfiguration"] = map[string]any{"enforcementLevel": "Max"}
		},
		"commandOutputType": func(_, props, _ map[string]any) {
			props["commandOutputSettings"] = map[string]any{"overrides": []any{map[string]any{"commandOutputType": "Other"}}}
		},
		"which the cluster doesn't have": func(_, props, _ map[string]any) {
			props["commandOutputSettings"] = map[string]any{"associatedIdentity": map[string]any{"identityType": "SystemAssignedIdentity"}}
		},
		"one of the cluster's user-assigned identities": func(_, props, _ map[string]any) {
			props["secretArchiveSettings"] = map[string]any{"associatedIdentity": map[string]any{"identityType": "UserAssignedIdentity", "userAssignedIdentityResourceId": "/subscriptions/s/x"}}
		},
		"identity.userAssignedIdentities": func(body, _, _ map[string]any) { body["identity"] = map[string]any{"type": "UserAssigned"} },
	} {
		h.rejects(http.MethodPut, url, mutate(change), want)
	}
	h.fails(http.MethodPut, h.base+"clusters/bad.name", h.cluster(azureNCTestLocation("cm-cl")), http.StatusBadRequest, "doesn't match")

	// A selected identity the cluster has is accepted.
	uami := "/subscriptions/s/resourceGroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/u1"
	ok := mutate(func(body, props, _ map[string]any) {
		body["identity"] = map[string]any{"type": "SystemAssigned,UserAssigned", "userAssignedIdentities": map[string]any{uami: map[string]any{}}}
		props["commandOutputSettings"] = map[string]any{"containerUrl": "https://acct.blob.core.windows.net/out",
			"associatedIdentity": map[string]any{"identityType": "UserAssignedIdentity", "userAssignedIdentityResourceId": uami}}
		props["analyticsOutputSettings"] = map[string]any{"associatedIdentity": map[string]any{"identityType": "SystemAssignedIdentity"}}
	})
	out := h.call(http.MethodPut, h.base+"clusters/ok", ok, http.StatusCreated)
	if azureARMObject(azureARMObject(azureARMObject(out["identity"])["userAssignedIdentities"])[uami])["principalId"] == nil {
		t.Fatalf("identity = %#v", out["identity"])
	}
	h.rejects(http.MethodPut, h.base+"clusters/ok", mutate(func(body, _, _ map[string]any) { body["extendedLocation"] = azureNCTestLocation("other") }), "extendedLocation can't be changed")
}

func TestAzureNetworkCloudKeySetsAndMetrics(t *testing.T) {
	t.Parallel()
	h := newAzureNCTest(t)
	cluster := h.base + "clusters/c1"
	h.call(http.MethodPut, cluster, h.cluster(azureNCTestLocation("cm-cl")), http.StatusCreated)
	future := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	user := map[string]any{"azureUserName": "alice", "sshPublicKey": map[string]any{"keyData": "ssh-rsa AAAA"}}
	keySet := map[string]any{"location": "eastus", "extendedLocation": azureNCTestLocation("c1-cluster-cl"), "properties": map[string]any{
		"azureGroupId": "g1", "expiration": future, "privilegeLevel": "Standard", "jumpHostsAllowed": []any{"10.0.0.4"}, "userList": []any{user}, "osGroupName": "ops",
	}}
	got := h.props(http.MethodPut, cluster+"/bareMetalMachineKeySets/ks1", keySet, http.StatusCreated)
	if got["detailedStatus"] != "AllActive" || azureARMObject(azureARMSlice(got["userListStatus"])[0])["status"] != "Active" {
		t.Fatalf("key set = %#v", got)
	}
	bad := func(key string, value any) map[string]any {
		props := azureARMMerge(azureARMObject(keySet["properties"]), map[string]any{key: value})
		return map[string]any{"location": "eastus", "extendedLocation": keySet["extendedLocation"], "properties": props}
	}
	h.rejects(http.MethodPut, cluster+"/bareMetalMachineKeySets/ks2", bad("jumpHostsAllowed", []any{"host"}), "IP addresses")
	h.rejects(http.MethodPut, cluster+"/bareMetalMachineKeySets/ks2", bad("privilegeLevel", "Root"), "privilegeLevel")
	h.rejects(http.MethodPut, cluster+"/bareMetalMachineKeySets/ks2", bad("privilegeLevel", "Other"), "privilegeLevelName is required")
	h.rejects(http.MethodPut, cluster+"/bareMetalMachineKeySets/ks2", bad("osGroupName", "1ops"), "osGroupName")
	h.rejects(http.MethodPut, cluster+"/bareMetalMachineKeySets/ks2", bad("expiration", "tomorrow"), "RFC 3339")
	h.rejects(http.MethodPut, cluster+"/bareMetalMachineKeySets/ks2", bad("userList", []any{user, user}), "listed twice")
	h.rejects(http.MethodPut, cluster+"/bareMetalMachineKeySets/ks2", bad("userList", []any{map[string]any{"azureUserName": "bob"}}), "keyData is required")
	h.rejects(http.MethodPut, cluster+"/bareMetalMachineKeySets/ks1", bad("azureGroupId", "g2"), "azureGroupId can't be changed")
	h.rejects(http.MethodPatch, cluster+"/bareMetalMachineKeySets/ks1", map[string]any{"properties": map[string]any{"privilegeLevel": "Superuser"}}, "can't be updated")
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	if got := h.props(http.MethodPatch, cluster+"/bareMetalMachineKeySets/ks1", map[string]any{"properties": map[string]any{"expiration": past}}, http.StatusOK); got["detailedStatus"] != "AllInvalid" {
		t.Fatalf("an expired key set = %#v", got)
	}

	bmc := map[string]any{"location": "eastus", "extendedLocation": keySet["extendedLocation"], "properties": map[string]any{
		"azureGroupId": "g1", "expiration": future, "privilegeLevel": "Administrator", "userList": []any{user},
	}}
	h.call(http.MethodPut, cluster+"/bmcKeySets/bk1", bmc, http.StatusCreated)
	azureARMObject(bmc["properties"])["privilegeLevel"] = "Standard"
	h.rejects(http.MethodPut, cluster+"/bmcKeySets/bk2", bmc, "privilegeLevel")

	metrics := cluster + "/metricsConfigurations/default"
	h.fails(http.MethodPut, cluster+"/metricsConfigurations/other", map[string]any{"location": "eastus"}, http.StatusBadRequest, "named 'default'")
	h.rejects(http.MethodPut, metrics, map[string]any{"location": "eastus", "extendedLocation": keySet["extendedLocation"], "properties": map[string]any{}}, "collectionInterval is required")
	h.rejects(http.MethodPut, metrics, map[string]any{"location": "eastus", "extendedLocation": keySet["extendedLocation"], "properties": map[string]any{"collectionInterval": 1441}}, "collectionInterval")
	h.rejects(http.MethodPut, metrics, map[string]any{"location": "eastus", "extendedLocation": keySet["extendedLocation"], "properties": map[string]any{"collectionInterval": 5, "enabledMetrics": []any{"made_up"}}}, "isn't an optional metric")
	got = h.props(http.MethodPut, metrics, map[string]any{"location": "eastus", "extendedLocation": keySet["extendedLocation"], "properties": map[string]any{"collectionInterval": 5, "enabledMetrics": []any{azureNCOptionalMetrics[0]}}}, http.StatusCreated)
	if got["detailedStatus"] != "Applied" || len(azureARMSlice(got["disabledMetrics"])) != len(azureNCOptionalMetrics)-1 {
		t.Fatalf("metrics = %#v", got)
	}
	if got := h.props(http.MethodPatch, metrics, map[string]any{"properties": map[string]any{"enabledMetrics": []any{}}}, http.StatusOK); len(azureARMSlice(got["disabledMetrics"])) != len(azureNCOptionalMetrics) {
		t.Fatalf("patched metrics = %#v", got)
	}
}

func TestAzureNetworkCloudCatalogs(t *testing.T) {
	t.Parallel()
	h := newAzureNCTest(t)
	skus := h.call(http.MethodGet, h.sub+"/providers/Microsoft.NetworkCloud/rackSkus", nil, http.StatusOK)
	if len(azureARMSlice(skus["value"])) != len(azureNCRackSkus) {
		t.Fatalf("rack SKUs = %#v", skus)
	}
	single := h.props(http.MethodGet, h.sub+"/providers/Microsoft.NetworkCloud/rackSkus/NearEdge_Single_R750", nil, http.StatusOK)
	if single["rackType"] != "Single" || len(azureARMSlice(single["storageAppliances"])) != 1 {
		t.Fatalf("single rack SKU = %#v", single)
	}
	versions := h.base + "kubernetesVersions/default"
	h.fails(http.MethodPut, h.base+"kubernetesVersions/v1", map[string]any{"location": "eastus", "extendedLocation": azureNCTestLocation("c1")}, http.StatusBadRequest, "named 'default'")
	h.rejects(http.MethodPut, versions, map[string]any{"location": "eastus"}, "extendedLocation is required")
	got := h.props(http.MethodPut, versions, map[string]any{"location": "eastus", "extendedLocation": azureNCTestLocation("c1")}, http.StatusCreated)
	if len(azureARMSlice(got["values"])) != len(azureNCKubernetesVersions) {
		t.Fatalf("Kubernetes versions = %#v", got)
	}
	h.rejects(http.MethodPatch, versions, map[string]any{"properties": map[string]any{"values": []any{}}}, "can't be updated")
	h.call(http.MethodPatch, versions, map[string]any{"tags": map[string]any{"a": "b"}}, http.StatusOK)
	ops := h.call(http.MethodGet, "/azure/providers/Microsoft.NetworkCloud/operations", nil, http.StatusOK)
	if len(azureARMSlice(ops["value"])) == 0 {
		t.Fatalf("operations = %#v", ops)
	}
}
