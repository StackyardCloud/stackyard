package server

import (
	"cmp"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Network Cloud clusters: validation of their rack definitions, the racks,
// bare metal machines and storage appliances those definitions create, and
// the cluster actions (deploy, version updates, scans, credential rotation
// and inspection).

// azureNCRackSku is a rack SKU of the regional catalog: which slots hold
// control plane machines, compute machines and storage appliances.
type azureNCRackSku struct {
	name, rackType, description string
	controllers, computes       []int
	storage                     []int
}

var azureNCRackSkus = []azureNCRackSku{
	{"NearEdge_Single_R750", "Single", "Single rack with two control plane machines, eight compute machines and a storage appliance.", []int{1, 2}, []int{3, 4, 5, 6, 7, 8, 9, 10}, []int{11}},
	{"NearEdge_Aggregator_R750", "Aggregator", "Aggregator rack of a multi-rack cluster.", []int{1, 2, 3}, []int{4, 5, 6, 7, 8, 9, 10, 11}, []int{12}},
	{"NearEdge_Compute_R750", "Compute", "Compute rack of a multi-rack cluster.", nil, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, nil},
}

// Machine and storage appliance sizes, the same in every SKU.
const (
	azureNCCores, azureNCMemoryGB, azureNCHostDiskGB, azureNCApplianceGB = 28, 512, 893, 9100
	azureNCMachineModel, azureNCApplianceModel                           = "PowerEdge R750", "x70r3-9"
)

func (sku azureNCRackSku) fixture() map[string]any {
	machines := func(slots []int, model string, cores, memory int) []any {
		out := []any{}
		for _, slot := range slots {
			out = append(out, map[string]any{"rackSlot": slot, "properties": map[string]any{
				"bootstrapProtocol": "PXE", "cpuCores": cores, "cpuSockets": 2, "memoryCapacityGB": memory, "model": model, "vendor": "Dell",
				"generation": "8", "hardwareVersion": "3", "totalThreads": cores * 4,
				"disks": []any{map[string]any{"capacityGB": azureNCHostDiskGB, "connection": "SAS", "type": "SSD"}},
			}})
		}
		return out
	}
	appliances := []any{}
	for _, slot := range sku.storage {
		appliances = append(appliances, map[string]any{"rackSlot": slot, "properties": map[string]any{"capacityGB": azureNCApplianceGB, "model": azureNCApplianceModel}})
	}
	return map[string]any{
		"provisioningState": "Succeeded", "rackType": sku.rackType, "description": sku.description, "deploymentType": "Nexus",
		"maxClusterSlots":     map[string]int{"Aggregator": 8}[sku.rackType],
		"controllerMachines":  machines(sku.controllers, "PowerEdge R650", 8, 128),
		"computeMachines":     machines(sku.computes, azureNCMachineModel, azureNCCores, azureNCMemoryGB),
		"storageAppliances":   appliances,
		"supportedRackSkuIds": []any{},
	}
}

// azureNCRackSkuOf returns the catalog SKU a rack SKU ID names.
func azureNCRackSkuOf(id string) (azureNCRackSku, bool) {
	match := azureNCRackSkuID.FindStringSubmatch(id)
	if match == nil {
		return azureNCRackSku{}, false
	}
	i := slices.IndexFunc(azureNCRackSkus, func(sku azureNCRackSku) bool { return strings.EqualFold(sku.name, match[1]) })
	if i < 0 {
		return azureNCRackSku{}, false
	}
	return azureNCRackSkus[i], true
}

var (
	azureNCFabricID     = azureNCIDRE("Microsoft.ManagedNetworkFabric/networkFabrics")
	azureNCNetworkRack  = azureNCIDRE("Microsoft.ManagedNetworkFabric/networkRacks")
	azureNCZoneRE       = regexp.MustCompile(`^[a-zA-Z0-9]{1,10}$`)
	azureNCOutputTypes  = []string{"BareMetalMachineRunCommand", "BareMetalMachineRunDataExtracts", "BareMetalMachineRunReadCommands", "ClusterSupportAdministrativeActions", "StorageRunReadCommands", "BareMetalMachineRunDataExtractsRestricted"}
	azureNCClusterPatch = []string{"aggregatorOrSingleRackDefinition", "analyticsOutputSettings", "clusterLocation", "clusterServicePrincipal", "commandOutputSettings", "computeDeploymentThreshold", "computeRackDefinitions", "runtimeProtectionConfiguration", "secretArchive", "secretArchiveSettings", "updateStrategy", "vulnerabilityScanningSettings"}
)

var azureNCClusterWriteOnly = []string{
	"clusterServicePrincipal.password",
	"aggregatorOrSingleRackDefinition.bareMetalMachineConfigurationData[].bmcCredentials.password",
	"aggregatorOrSingleRackDefinition.storageApplianceConfigurationData[].adminCredentials.password",
	"computeRackDefinitions[].bareMetalMachineConfigurationData[].bmcCredentials.password",
	"computeRackDefinitions[].storageApplianceConfigurationData[].adminCredentials.password",
}

// azureNCSameRacks reports whether two clusters' rack definitions match,
// an absent compute rack list matching an empty one.
func azureNCSameRacks(a, b map[string]any) bool {
	return reflect.DeepEqual(a["aggregatorOrSingleRackDefinition"], b["aggregatorOrSingleRackDefinition"]) &&
		(len(azureARMSlice(a["computeRackDefinitions"])) == 0 && len(azureARMSlice(b["computeRackDefinitions"])) == 0 ||
			reflect.DeepEqual(a["computeRackDefinitions"], b["computeRackDefinitions"]))
}

func azureNCRegisterClusters() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureNCNamespace, Type: "clusters",
		ValidateName: azureNCName("cluster", azureNCNameRE),
		Validate: func(payload map[string]any) string {
			return cmp.Or(azureNCExtendedLocation(payload), azureNCIdentity(payload), azureDataBoxObjects(payload, "", "properties"),
				azureLabEnum(payload, "", "kind", []string{"Nexus", "AzureLocal"}), azureNCClusterShape(azureARMObject(payload["properties"]), true))
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			if msg := azureNCIdentity(payload); msg != "" {
				return nil, nil, msg
			}
			return azureLabPatchOnly(payload, func(props map[string]any) string { return azureNCClusterShape(props, false) }, azureNCClusterPatch...)
		},
		WriteOnlyProperties: azureNCClusterWriteOnly,
		// ponytail: the rack definitions' hardware is made by PUT (Linked),
		// so a PATCH may only resend the definitions unchanged.
		ValidatePatch: func(_ *Server, existing *azureARMResource, payload map[string]any) string {
			merged := azureARMMerge(existing.Properties, azureARMObject(payload["properties"]))
			for _, path := range azureNCClusterWriteOnly {
				merged = azureARMWithout(merged, path)
			}
			if !azureNCSameRacks(existing.Properties, merged) {
				return "The rack definitions can only be changed with a PUT of the cluster."
			}
			return ""
		},
		StateProperties:    map[string]any{"detailedStatus": "PendingDeployment", "detailedStatusMessage": "The cluster is ready for deployment.", "actionStates": nil, "lastSuccessfulVersionUpdateTime": nil},
		Reconcile:          azureNCClusterReconcile,
		ReadOnlyProperties: azureNCClusterReadOnly,
		Linked:             azureNCClusterLinked,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"deploy":                azureNCDeploy,
			"updateversion":         azureNCUpdateVersion,
			"continueupdateversion": azureNCContinueUpdate,
			"scanruntime":           azureNCScanRuntime,
			"rotatecredential":      azureNCRotateCredential,
			"inspect":               azureNCInspect,
		},
	})
}

// azureNCClusterShape checks cluster properties; full requires what a PUT
// must send, while a PATCH may send any part.
func azureNCClusterShape(props map[string]any, full bool) string {
	if full {
		if msg := cmp.Or(
			azureWorkloadsRequired(props, "properties.", "clusterType", "clusterVersion"),
			azureNCIDOf(props, "properties.", "networkFabricId", "a network fabric", azureNCFabricID, true),
		); msg != "" {
			return msg
		}
		if _, ok := props["aggregatorOrSingleRackDefinition"].(map[string]any); !ok {
			return "properties.aggregatorOrSingleRackDefinition is required."
		}
	}
	if msg := cmp.Or(
		azureDataBoxObjects(props, "properties.", "aggregatorOrSingleRackDefinition", "analyticsOutputSettings", "clusterServicePrincipal", "commandOutputSettings",
			"computeDeploymentThreshold", "managedResourceGroupConfiguration", "runtimeProtectionConfiguration", "secretArchive", "secretArchiveSettings", "updateStrategy", "vulnerabilityScanningSettings"),
		azureLabEnum(props, "properties.", "clusterType", []string{"SingleRack", "MultiRack"}),
		azureNCIDOf(props, "properties.", "analyticsWorkspaceId", "a Log Analytics workspace", azureNCWorkspaceID, false),
		azureLabLength(props, "properties.", "clusterLocation", 0, 256),
		azureLabLength(azureARMObject(props["managedResourceGroupConfiguration"]), "properties.managedResourceGroupConfiguration.", "name", 1, 75),
	); msg != "" {
		return msg
	}
	if version, present := props["clusterVersion"]; present && !slices.ContainsFunc(azureNCVersions, func(v [2]string) bool { return v[0] == version }) {
		return "properties.clusterVersion " + azureARMString(version) + " isn't a supported cluster version."
	}
	if raw, present := props["aggregatorOrSingleRackDefinition"]; present {
		if msg := azureNCRackDefinition(azureARMObject(raw), "properties.aggregatorOrSingleRackDefinition.", full); msg != "" {
			return msg
		}
	}
	racks, ok := props["computeRackDefinitions"].([]any)
	if _, present := props["computeRackDefinitions"]; present && !ok {
		return "properties.computeRackDefinitions must be a list."
	}
	for i, raw := range racks {
		if msg := azureNCRackDefinition(azureARMObject(raw), "properties.computeRackDefinitions["+strconv.Itoa(i)+"].", full); msg != "" {
			return msg
		}
	}
	return azureNCClusterSettings(props, full)
}

// azureNCClusterSettings checks the cluster's optional settings objects.
func azureNCClusterSettings(props map[string]any, full bool) string {
	principal := azureARMObject(props["clusterServicePrincipal"])
	threshold := azureARMObject(props["computeDeploymentThreshold"])
	strategy := azureARMObject(props["updateStrategy"])
	archive := azureARMObject(props["secretArchive"])
	protection := azureARMObject(props["runtimeProtectionConfiguration"])
	var need func(m map[string]any, path string, keys ...string) string = func(map[string]any, string, ...string) string { return "" }
	if full {
		need = azureWorkloadsRequired
	}
	thresholdTypes := map[string][]string{"type": {"CountSuccess", "PercentSuccess"}, "grouping": {"PerCluster", "PerRack"}}
	msg := cmp.Or(
		azureNCWhenSet(props, "clusterServicePrincipal", func() string {
			return need(principal, "properties.clusterServicePrincipal.", "applicationId", "password", "principalId", "tenantId")
		}),
		azureNCWhenSet(props, "computeDeploymentThreshold", func() string {
			return cmp.Or(azureNCRequiredKeys(threshold, "properties.computeDeploymentThreshold.", full, "type", "grouping", "value"),
				azureWorkloadsEnums(threshold, "properties.computeDeploymentThreshold.", nil, thresholdTypes),
				azureOracleNumber(threshold, "properties.computeDeploymentThreshold.", "value", 0, 1e9, true))
		}),
		azureNCWhenSet(props, "updateStrategy", func() string {
			return cmp.Or(azureNCRequiredKeys(strategy, "properties.updateStrategy.", full, "strategyType", "thresholdType", "thresholdValue"),
				azureWorkloadsEnums(strategy, "properties.updateStrategy.", nil, map[string][]string{"strategyType": {"Rack", "PauseAfterRack"}, "thresholdType": thresholdTypes["type"]}),
				azureOracleNumber(strategy, "properties.updateStrategy.", "thresholdValue", 0, 1e9, true),
				azureOracleNumber(strategy, "properties.updateStrategy.", "maxUnavailable", 1, 1e9, true),
				azureOracleNumber(strategy, "properties.updateStrategy.", "waitTimeMinutes", 0, 60, true))
		}),
		azureNCWhenSet(props, "secretArchive", func() string {
			return cmp.Or(azureNCIDOf(archive, "properties.secretArchive.", "keyVaultId", "a key vault", azureNCKeyVaultID, full),
				azureLabEnum(archive, "properties.secretArchive.", "useKeyVault", azureNCTrueFalse))
		}),
		azureWorkloadsEnums(protection, "properties.runtimeProtectionConfiguration.", nil, map[string][]string{
			"definitionUpdateMode": {"Automatic", "None"}, "enforcementLevel": {"Audit", "Disabled", "OnDemand", "Passive", "RealTime"},
		}),
		azureLabEnum(azureARMObject(props["vulnerabilityScanningSettings"]), "properties.vulnerabilityScanningSettings.", "containerScan", []string{"Disabled", "Enabled"}),
		azureNCIDOf(azureARMObject(props["analyticsOutputSettings"]), "properties.analyticsOutputSettings.", "analyticsWorkspaceId", "a Log Analytics workspace", azureNCWorkspaceID, false),
	)
	if msg != "" {
		return msg
	}
	output := azureARMObject(props["commandOutputSettings"])
	for i, raw := range azureARMSlice(output["overrides"]) {
		if msg := azureLabEnum(azureARMObject(raw), "properties.commandOutputSettings.overrides["+strconv.Itoa(i)+"].", "commandOutputType", azureNCOutputTypes); msg != "" {
			return msg
		}
	}
	return ""
}

// azureNCWhenSet runs check when props[key] is present.
func azureNCWhenSet(props map[string]any, key string, check func() string) string {
	if _, present := props[key]; !present {
		return ""
	}
	return check()
}

// azureNCRequiredKeys requires keys of any type when full.
func azureNCRequiredKeys(m map[string]any, path string, full bool, keys ...string) string {
	for _, key := range keys {
		if _, present := m[key]; full && !present {
			return path + key + " is required."
		}
	}
	return ""
}

// azureNCRackDefinition checks one rack definition and its machines.
func azureNCRackDefinition(rack map[string]any, path string, full bool) string {
	if full {
		if msg := cmp.Or(azureWorkloadsRequired(rack, path, "rackSerialNumber"),
			azureNCIDOf(rack, path, "networkRackId", "a network rack", azureNCNetworkRack, true),
			azureNCIDOf(rack, path, "rackSkuId", "a rack SKU", azureNCRackSkuID, true)); msg != "" {
			return msg
		}
	}
	if zone, present := rack["availabilityZone"]; present && !azureNCZoneRE.MatchString(azureARMString(zone)) {
		return path + "availabilityZone must be 1 to 10 letters and digits."
	}
	if msg := cmp.Or(azureLabLength(rack, path, "rackLocation", 0, 256), azureLabLength(rack, path, "rackSerialNumber", 1, 64),
		azureNCIDOf(rack, path, "networkRackId", "a network rack", azureNCNetworkRack, false)); msg != "" {
		return msg
	}
	if id, present := rack["rackSkuId"]; present {
		if _, ok := azureNCRackSkuOf(azureARMString(id)); !ok {
			return path + "rackSkuId must name a rack SKU of the catalog (" + azureNCRackSkuNames() + ")."
		}
	}
	for i, raw := range azureARMSlice(rack["bareMetalMachineConfigurationData"]) {
		machine := azureARMObject(raw)
		at := path + "bareMetalMachineConfigurationData[" + strconv.Itoa(i) + "]."
		if msg := azureNCMachineConfig(machine, at, full); msg != "" {
			return msg
		}
	}
	for i, raw := range azureARMSlice(rack["storageApplianceConfigurationData"]) {
		appliance := azureARMObject(raw)
		at := path + "storageApplianceConfigurationData[" + strconv.Itoa(i) + "]."
		if full {
			if msg := cmp.Or(azureNCRequiredKeys(appliance, at, true, "rackSlot", "serialNumber", "adminCredentials"),
				azureWorkloadsRequired(azureARMObject(appliance["adminCredentials"]), at+"adminCredentials.", "username", "password")); msg != "" {
				return msg
			}
		}
		if name, present := appliance["storageApplianceName"]; present && !azureNCNameRE.MatchString(azureARMString(name)) {
			return at + "storageApplianceName must match " + azureNCNameRE.String() + "."
		}
		if msg := azureOracleNumber(appliance, at, "rackSlot", 1, 256, true); msg != "" {
			return msg
		}
	}
	return ""
}

func azureNCMachineConfig(machine map[string]any, at string, full bool) string {
	if full {
		if msg := cmp.Or(azureNCRequiredKeys(machine, at, true, "bmcCredentials", "bmcMacAddress", "bootMacAddress", "rackSlot", "serialNumber"),
			azureWorkloadsRequired(azureARMObject(machine["bmcCredentials"]), at+"bmcCredentials.", "username", "password")); msg != "" {
			return msg
		}
	}
	for _, key := range []string{"bmcMacAddress", "bootMacAddress"} {
		if mac, present := machine[key]; present && !azureNCMACRE.MatchString(azureARMString(mac)) {
			return at + key + " must be a MAC address such as AA:BB:CC:DD:EE:FF."
		}
	}
	if name, present := machine["machineName"]; present && !azureNCMachineRE.MatchString(azureARMString(name)) {
		return at + "machineName must be 2 to 64 letters and digits."
	}
	return cmp.Or(azureOracleNumber(machine, at, "rackSlot", 1, 256, true), azureLabLength(machine, at, "serialNumber", 1, 64), azureLabLength(machine, at, "machineDetails", 0, 256))
}

func azureNCRackSkuNames() string {
	var names []string
	for _, sku := range azureNCRackSkus {
		names = append(names, sku.name)
	}
	return strings.Join(names, ", ")
}

// azureNCRackDefinitions returns the cluster's rack definitions, the
// aggregator or single rack first.
func azureNCRackDefinitions(props map[string]any) []map[string]any {
	racks := []map[string]any{azureARMObject(props["aggregatorOrSingleRackDefinition"])}
	for _, raw := range azureARMSlice(props["computeRackDefinitions"]) {
		racks = append(racks, azureARMObject(raw))
	}
	return racks
}

// azureNCClusterTopology checks that the rack definitions fit the cluster
// type and their SKUs: one Single rack and no compute racks for a single
// rack cluster, an Aggregator rack and Compute racks for a multi-rack one,
// machines and appliances only in slots the SKU has, each slot used once.
func azureNCClusterTopology(props map[string]any) string {
	racks := azureNCRackDefinitions(props)
	single := props["clusterType"] == "SingleRack"
	if single && len(racks) > 1 {
		return "properties.computeRackDefinitions must be empty for a SingleRack cluster."
	}
	for i, rack := range racks {
		sku, _ := azureNCRackSkuOf(azureARMString(rack["rackSkuId"]))
		path := "properties.aggregatorOrSingleRackDefinition."
		want := map[bool]string{true: "Single", false: "Aggregator"}[single]
		if i > 0 {
			path, want = "properties.computeRackDefinitions["+strconv.Itoa(i-1)+"].", "Compute"
		}
		if sku.rackType != want {
			return path + "rackSkuId must be a " + want + " rack SKU; " + sku.name + " is a " + sku.rackType + " rack SKU."
		}
		if msg := cmp.Or(
			azureNCSlots(rack["bareMetalMachineConfigurationData"], append(slices.Clone(sku.controllers), sku.computes...), path+"bareMetalMachineConfigurationData", sku.name),
			azureNCSlots(rack["storageApplianceConfigurationData"], sku.storage, path+"storageApplianceConfigurationData", sku.name),
		); msg != "" {
			return msg
		}
	}
	return ""
}

func azureNCSlots(items any, slots []int, path, sku string) string {
	used := map[int]bool{}
	for i, raw := range azureARMSlice(items) {
		slot := azureHZCount(azureARMObject(raw)["rackSlot"])
		at := path + "[" + strconv.Itoa(i) + "].rackSlot"
		switch {
		case !slices.Contains(slots, slot):
			return fmt.Sprintf("%s %d isn't a slot rack SKU %s has for it (%v).", at, slot, sku, slots)
		case used[slot]:
			return fmt.Sprintf("%s %d is used twice in the rack.", at, slot)
		}
		used[slot] = true
	}
	return ""
}

func azureNCClusterReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	rec.Identity = azureNCIdentityIDs(rec.Identity, rec.ID)
	azureNCKeepManagedGroup(existing, rec)
	if existing != nil && existing.Properties["detailedStatus"] != "PendingDeployment" && !azureNCSameRacks(existing.Properties, rec.Properties) {
		return nil, "The rack definitions can't change once the cluster leaves PendingDeployment; its detailedStatus is " + azureARMString(existing.Properties["detailedStatus"]) + "."
	}
	if msg := cmp.Or(azureNCCreateOnly(existing, rec, "analyticsWorkspaceId", "networkFabricId", "managedResourceGroupConfiguration"),
		azureNCClusterTopology(rec.Properties), azureNCSelectors(rec)); msg != "" {
		return nil, msg
	}
	if existing != nil && existing.Properties["clusterVersion"] != rec.Properties["clusterVersion"] {
		return nil, "properties.clusterVersion can only be changed with the updateVersion action."
	}
	out := map[string]any{"managedResourceGroupConfiguration": azureNCManagedGroup(rec), "clusterManagerId": ""}
	for _, manager := range s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool { return strings.EqualFold(r.Type, azureNCManagerType) }) {
		if strings.EqualFold(azureARMString(azureARMObject(manager.Properties["managerExtendedLocation"])["name"]), azureARMString(rec.ExtendedLocation["name"])) {
			out["clusterManagerId"] = manager.ID
		}
	}
	return out, ""
}

// azureNCSelectors checks that the identities the output settings select
// are ones the cluster has.
func azureNCSelectors(rec *azureARMResource) string {
	identityType := azureARMString(rec.Identity["type"])
	selectors := map[string]any{}
	for _, key := range []string{"commandOutputSettings", "analyticsOutputSettings", "secretArchiveSettings"} {
		selectors["properties."+key+".associatedIdentity"] = azureARMObject(rec.Properties[key])["associatedIdentity"]
	}
	for i, raw := range azureARMSlice(azureARMObject(rec.Properties["commandOutputSettings"])["overrides"]) {
		selectors["properties.commandOutputSettings.overrides["+strconv.Itoa(i)+"].associatedIdentity"] = azureARMObject(raw)["associatedIdentity"]
	}
	for _, path := range slices.Sorted(maps.Keys(selectors)) {
		selector := azureARMObject(selectors[path])
		if len(selector) == 0 {
			continue
		}
		id := azureARMString(selector["userAssignedIdentityResourceId"])
		if msg := azureLabEnum(selector, path+".", "identityType", []string{"SystemAssignedIdentity", "UserAssignedIdentity"}); msg != "" {
			return msg
		}
		switch selector["identityType"] {
		case "SystemAssignedIdentity":
			if id != "" {
				return path + ".userAssignedIdentityResourceId can't be set with a system-assigned identity."
			}
			if !strings.HasPrefix(identityType, "SystemAssigned") {
				return path + " selects the system-assigned identity, which the cluster doesn't have."
			}
		case "UserAssignedIdentity":
			if _, ok := azureARMObject(rec.Identity["userAssignedIdentities"])[id]; !ok {
				return path + ".userAssignedIdentityResourceId must be one of the cluster's user-assigned identities."
			}
		}
	}
	return ""
}

// azureNCVersionProps returns the properties the cluster version decides.
func azureNCVersionProps(version string) map[string]any {
	upgrades := []any{}
	support := ""
	for i, v := range azureNCVersions {
		if v[0] == version {
			support = v[1]
			for _, next := range azureNCVersions[i+1:] {
				upgrades = append(upgrades, map[string]any{
					"targetClusterVersion": next[0], "supportExpiryDate": next[1], "controlImpact": "False", "workloadImpact": "False",
					"expectedDuration": "2:0:0", "impactDescription": "Updated rack by rack in place.",
				})
			}
		}
	}
	return map[string]any{"supportExpiryDate": support, "availableUpgradeVersions": upgrades}
}

func azureNCClusterReadOnly(rec *azureARMResource) map[string]any {
	out := azureNCVersionProps(azureARMString(rec.Properties["clusterVersion"]))
	machines, appliances := 0, 0
	for _, rack := range azureNCRackDefinitions(rec.Properties) {
		machines += len(azureARMSlice(rack["bareMetalMachineConfigurationData"]))
		appliances += len(azureARMSlice(rack["storageApplianceConfigurationData"]))
	}
	// ponytail: capacity counts every machine and appliance as free;
	// subtract workload usage if a client depends on it.
	out["clusterCapacity"] = map[string]any{
		"totalCoreCount": machines * azureNCCores, "availableCoreCount": machines * azureNCCores,
		"totalMemoryGB": machines * azureNCMemoryGB, "availableMemoryGB": machines * azureNCMemoryGB,
		"totalHostStorageGB": machines * azureNCHostDiskGB, "availableHostStorageGB": machines * azureNCHostDiskGB,
		"totalApplianceStorageGB": appliances * azureNCApplianceGB, "availableApplianceStorageGB": appliances * azureNCApplianceGB,
	}
	out["clusterExtendedLocation"] = azureNCCustomLocationOf(rec, "-cluster-cl")
	out["hybridAksExtendedLocation"] = azureNCCustomLocationOf(rec, "-hybridaks-cl")
	out["clusterConnectionStatus"], out["clusterManagerConnectionStatus"] = "Connected", "Connected"
	out["managedCredentials"] = azureOracleStringsOrEmpty(azureNCManagedCredentials)
	out["manualActionCount"] = 0
	out["workloadResourceIds"] = []any{}
	return out
}

// azureNCClusterLinked creates the racks, bare metal machines and storage
// appliances the rack definitions describe, in the cluster's resource
// group. Records a previous PUT created keep their state.
func azureNCClusterLinked(s *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
	location := azureNCCustomLocationOf(rec, "-cluster-cl")
	group := "/subscriptions/" + rec.SubscriptionID + "/resourceGroups/" + rec.ResourceGroup + "/providers/"
	var linked []*azureARMResource
	add := func(typ, name string, props map[string]any) string {
		id := group + typ + "/" + name
		if slices.ContainsFunc(linked, func(r *azureARMResource) bool { return strings.EqualFold(r.ID, id) }) {
			return "The rack definitions name " + id + " twice."
		}
		if stored := s.azureARMResources[azureARMKey(id)]; stored != nil {
			if owner := azureARMString(stored.Properties["clusterId"]); !strings.EqualFold(owner, rec.ID) {
				return id + " already exists" + map[bool]string{true: " outside any cluster.", false: " in cluster " + owner + "."}[owner == ""]
			}
			linked = append(linked, stored)
			return ""
		}
		props["clusterId"] = rec.ID
		linked = append(linked, &azureARMResource{
			ID: id, SubscriptionID: rec.SubscriptionID, ResourceGroup: rec.ResourceGroup, Name: name, Type: typ, Location: rec.Location,
			ExtendedLocation: location, Properties: azureARMMerge(props, map[string]any{"provisioningState": "Succeeded"}),
			CreatedAt: rec.UpdatedAt, UpdatedAt: rec.UpdatedAt, Tags: map[string]any{},
		})
		return ""
	}
	for i, rack := range azureNCRackDefinitions(rec.Properties) {
		rackName := fmt.Sprintf("%s-rack%d", rec.Name, i+1)
		rackID := group + azureNCRackType + "/" + rackName
		if msg := add(azureNCRackType, rackName, azureNCRackProps(rack)); msg != "" {
			return nil, nil, msg
		}
		sku, _ := azureNCRackSkuOf(azureARMString(rack["rackSkuId"]))
		for _, raw := range azureARMSlice(rack["bareMetalMachineConfigurationData"]) {
			machine := azureARMObject(raw)
			slot := azureHZCount(machine["rackSlot"])
			name := cmp.Or(azureARMString(machine["machineName"]), fmt.Sprintf("%sr%ds%d", azureNCAlnum(rec.Name), i+1, slot))
			if msg := add(azureNCMachineType, name, azureNCMachineProps(machine, name, rackID, i+1, slices.Contains(sku.controllers, slot))); msg != "" {
				return nil, nil, msg
			}
		}
		for _, raw := range azureARMSlice(rack["storageApplianceConfigurationData"]) {
			appliance := azureARMObject(raw)
			name := cmp.Or(azureARMString(appliance["storageApplianceName"]), fmt.Sprintf("%s-rack%d-sa%d", rec.Name, i+1, azureHZCount(appliance["rackSlot"])))
			if msg := add(azureNCApplianceType, name, azureNCApplianceProps(appliance, rackID, i+1)); msg != "" {
				return nil, nil, msg
			}
		}
	}
	return nil, linked, ""
}

func azureNCAlnum(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || r == '_' {
			return -1
		}
		return r
	}, name)
}

func azureNCRackProps(rack map[string]any) map[string]any {
	return map[string]any{
		"availabilityZone": cmp.Or(azureARMString(rack["availabilityZone"]), "1"), "rackLocation": azureARMString(rack["rackLocation"]),
		"rackSerialNumber": rack["rackSerialNumber"], "rackSkuId": rack["rackSkuId"],
		"detailedStatus": "Provisioning", "detailedStatusMessage": "The rack is waiting for the cluster to deploy.",
	}
}

// azureNCClusterOfLocked returns the cluster a rack, machine or appliance
// belongs to, or nil. Caller holds providerStorageMu.
func (s *Server) azureNCClusterOfLocked(rec *azureARMResource) *azureARMResource {
	return s.azureARMResources[azureARMKey(azureARMString(rec.Properties["clusterId"]))]
}

// azureNCMembersLocked returns the cluster's linked records of typ.
// Caller holds providerStorageMu.
func (s *Server) azureNCMembersLocked(cluster *azureARMResource, typ string) []*azureARMResource {
	var out []*azureARMResource
	for _, id := range cluster.LinkedIDs {
		if rec := s.azureARMResources[azureARMKey(id)]; rec != nil && strings.EqualFold(rec.Type, typ) {
			out = append(out, rec)
		}
	}
	return out
}

// azureNCClusterAction returns the stored cluster rec names when its
// detailedStatus is one of allowed (any, when none are given), or the
// error answer. Caller holds providerStorageMu.
func (s *Server) azureNCClusterAction(rec *azureARMResource, allowed ...string) (*azureARMResource, int, any) {
	cluster := s.azureARMResources[azureARMKey(rec.ID)]
	if cluster == nil {
		status, body := azureNCMissing()
		return nil, status, body
	}
	if status := azureARMString(cluster.Properties["detailedStatus"]); len(allowed) > 0 && !slices.Contains(allowed, status) {
		status, body := azureOracleConflict("The cluster's detailedStatus is " + status + "; this action needs " + strings.Join(allowed, " or ") + ".")
		return nil, status, body
	}
	return cluster, 0, nil
}

func azureNCMissing() (int, any) {
	return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The resource was not found.")
}

// azureNCDeploy deploys a cluster waiting for deployment: its machines
// are provisioned and powered on, its racks and appliances available.
func azureNCDeploy(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := azureOracleStrings(payload, "", "skipValidationsForMachines", 0, 256); msg != "" {
		return azureOracleBad(msg)
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	cluster, status, body := s.azureNCClusterAction(rec, "PendingDeployment", "Failed")
	if cluster == nil {
		return status, body
	}
	machines := s.azureNCMembersLocked(cluster, azureNCMachineType)
	for _, name := range azureOracleStringsList(payload["skipValidationsForMachines"]) {
		if !slices.ContainsFunc(machines, func(m *azureARMResource) bool { return strings.EqualFold(m.Name, name) }) {
			return azureOracleBad("skipValidationsForMachines names " + name + ", which isn't a bare metal machine of the cluster.")
		}
	}
	for i, machine := range machines {
		s.azureOracleUpdateLocked(machine.ID, map[string]any{
			"detailedStatus": "Provisioned", "detailedStatusMessage": "The bare metal machine is provisioned.", "powerState": "On", "readyState": "True",
			"cordonStatus": "Uncordoned", "kubernetesNodeName": strings.ToLower(machine.Name), "kubernetesVersion": azureNCKubernetesVersions[len(azureNCKubernetesVersions)-1],
			"osImage": "nexus-" + azureARMString(cluster.Properties["clusterVersion"]), "oamIpv4Address": fmt.Sprintf("10.0.%d.%d", 1+i/240, 10+i%240),
		})
	}
	for _, rack := range s.azureNCMembersLocked(cluster, azureNCRackType) {
		s.azureOracleUpdateLocked(rack.ID, map[string]any{"detailedStatus": "Available", "detailedStatusMessage": "The rack is available."})
	}
	for _, appliance := range s.azureNCMembersLocked(cluster, azureNCApplianceType) {
		s.azureOracleUpdateLocked(appliance.ID, map[string]any{"detailedStatus": "Available", "detailedStatusMessage": "The storage appliance is up and running."})
	}
	s.azureOracleUpdateLocked(cluster.ID, map[string]any{
		"detailedStatus": "Running", "detailedStatusMessage": "The cluster is running.",
		"actionStates": azureNCActionState(cluster.Properties["actionStates"], azureNCClusterType, "deploy"),
	})
	return azureNCDone(cluster, "deploy", nil)
}

// azureNCUpdateVersion moves a running cluster to one of its available
// upgrade versions. With the PauseAfterRack strategy and more than one
// rack, the update pauses until continueUpdateVersion.
// ponytail: clusterVersion changes at once, not after the last rack.
func azureNCUpdateVersion(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := cmp.Or(azureWorkloadsRequired(payload, "", "targetClusterVersion"), azureLabEnum(payload, "", "safeguardMode", azureNCSafeguard)); msg != "" {
		return azureOracleBad(msg)
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	cluster, status, body := s.azureNCClusterAction(rec, "Running")
	if cluster == nil {
		return status, body
	}
	target := azureARMString(payload["targetClusterVersion"])
	if !slices.ContainsFunc(azureARMSlice(cluster.Properties["availableUpgradeVersions"]), func(v any) bool { return azureARMObject(v)["targetClusterVersion"] == target }) {
		return azureOracleBad("targetClusterVersion " + target + " isn't one of the cluster's availableUpgradeVersions.")
	}
	props := azureNCVersionProps(target)
	props["clusterVersion"] = target
	props["detailedStatus"], props["detailedStatusMessage"] = "Running", "The cluster is running."
	props["lastSuccessfulVersionUpdateTime"] = azureHZTime(time.Now())
	if azureARMObject(cluster.Properties["updateStrategy"])["strategyType"] == "PauseAfterRack" && len(azureNCRackDefinitions(cluster.Properties)) > 1 {
		props["detailedStatus"], props["detailedStatusMessage"] = "UpdatePaused", "The update paused after the first rack."
		delete(props, "lastSuccessfulVersionUpdateTime")
	}
	props["actionStates"] = azureNCActionState(cluster.Properties["actionStates"], azureNCClusterType, "updateVersion")
	s.azureOracleUpdateLocked(cluster.ID, props)
	return azureNCDone(cluster, "updateVersion", nil)
}

func azureNCContinueUpdate(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := cmp.Or(azureLabEnum(payload, "", "machineGroupTargetingMode", []string{"AlphaByRack"}), azureLabEnum(payload, "", "safeguardMode", azureNCSafeguard)); msg != "" {
		return azureOracleBad(msg)
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	cluster, status, body := s.azureNCClusterAction(rec, "UpdatePaused")
	if cluster == nil {
		return status, body
	}
	s.azureOracleUpdateLocked(cluster.ID, map[string]any{
		"detailedStatus": "Running", "detailedStatusMessage": "The cluster is running.", "lastSuccessfulVersionUpdateTime": azureHZTime(time.Now()),
		"actionStates": azureNCActionState(cluster.Properties["actionStates"], azureNCClusterType, "continueUpdateVersion"),
	})
	return azureNCDone(cluster, "continueUpdateVersion", nil)
}

// azureNCDeployed are the detailedStatus values of a deployed cluster.
var azureNCDeployed = []string{"Running", "Updating", "UpdatePaused", "Degraded"}

func azureNCScanRuntime(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := azureLabEnum(payload, "", "scanActivity", []string{"Scan", "Skip"}); msg != "" {
		return azureOracleBad(msg)
	}
	return s.azureNCRecordAction(rec, "scanRuntime")
}

func azureNCRotateCredential(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	credentials, ok := payload["credentials"].([]any)
	if !ok || len(credentials) == 0 {
		return azureOracleBad("credentials must list at least one credential to rotate.")
	}
	for _, raw := range credentials {
		if !slices.Contains(azureNCManagedCredentials, azureARMString(raw)) {
			return azureOracleBad("credentials names '" + azureARMString(raw) + "', which isn't one of the cluster's managedCredentials.")
		}
	}
	return s.azureNCRecordAction(rec, "rotateCredential")
}

func azureNCInspect(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	for _, action := range azureARMSlice(payload["additionalActions"]) {
		if action != "ResetHardware" {
			return azureOracleBad("additionalActions must hold only ResetHardware.")
		}
	}
	if msg := azureDataBoxObjects(payload, "", "filterDevices"); msg != "" {
		return azureOracleBad(msg)
	}
	filter := azureARMObject(payload["filterDevices"])
	s.providerStorageMu.Lock()
	cluster := s.azureARMResources[azureARMKey(rec.ID)]
	var missing string
	if cluster != nil {
		for key, typ := range map[string]string{"rackNames": azureNCRackType, "bareMetalMachineNames": azureNCMachineType} {
			members := s.azureNCMembersLocked(cluster, typ)
			for _, name := range azureOracleStringsList(filter[key]) {
				if missing == "" && !slices.ContainsFunc(members, func(m *azureARMResource) bool { return strings.EqualFold(m.Name, name) }) {
					missing = "filterDevices." + key + " names " + name + ", which isn't part of the cluster."
				}
			}
		}
	}
	s.providerStorageMu.Unlock()
	if missing != "" {
		return azureOracleBad(missing)
	}
	return s.azureNCRecordAction(rec, "inspect")
}

// azureNCRecordAction records a finished action on a deployed cluster.
func (s *Server) azureNCRecordAction(rec *azureARMResource, action string) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	cluster, status, body := s.azureNCClusterAction(rec, azureNCDeployed...)
	if cluster == nil {
		return status, body
	}
	s.azureOracleUpdateLocked(cluster.ID, map[string]any{"actionStates": azureNCActionState(cluster.Properties["actionStates"], azureNCClusterType, action)})
	return azureNCDone(cluster, action, nil)
}
