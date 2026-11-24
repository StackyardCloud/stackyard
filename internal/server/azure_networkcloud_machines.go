package server

import (
	"cmp"
	"fmt"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
)

// Network Cloud racks, bare metal machines and storage appliances, and
// access bridges. A cluster's rack definitions create its racks, machines
// and appliances; a client may also PUT one directly, which creates a
// standalone record outside any cluster, reported ready. A PUT on a
// cluster's member keeps its server-owned state and its cluster, and only
// standalone records can be deleted: the cluster's go with it.

var azureNCPortRE = regexp.MustCompile(`^(\d{1,5})(?:-(\d{1,5}))?$`)

var (
	azureNCRackID = azureNCIDRE(azureNCRackType)
	// Server-owned member properties, kept on PUT and PATCH, with the
	// values a standalone record starts with.
	azureNCRackState    = map[string]any{"clusterId": nil, "detailedStatus": "Available", "detailedStatusMessage": "The rack is available."}
	azureNCMachineState = map[string]any{
		"clusterId": nil, "detailedStatus": "Provisioned", "detailedStatusMessage": "The bare metal machine is provisioned.",
		"powerState": "On", "readyState": "True", "cordonStatus": "Uncordoned", "actionStates": nil, "machineRoles": nil,
		"bmcIpv4Address": nil, "serviceTag": nil, "kubernetesNodeName": nil, "kubernetesVersion": nil, "osImage": nil, "oamIpv4Address": nil,
	}
	azureNCApplianceState = map[string]any{
		"clusterId": nil, "detailedStatus": "Available", "detailedStatusMessage": "The storage appliance is up and running.",
		"remoteVendorManagementFeature": "Supported", "remoteVendorManagementStatus": "Disabled", "model": nil, "manufacturer": nil,
		"version": nil, "capacity": nil, "capacityUsed": nil, "managementIpv4Address": nil,
	}
)

func azureNCRegisterMachines() {
	ns := azureNCNamespace
	patchOnly := func(check func(map[string]any) string, keys ...string) func(map[string]any) (map[string]any, map[string]any, string) {
		return func(payload map[string]any) (map[string]any, map[string]any, string) {
			return azureLabPatchOnly(payload, check, keys...)
		}
	}
	createOnly := func(keys ...string) func(*Server, *azureARMResource, *azureARMResource) (map[string]any, string) {
		return func(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
			return nil, azureNCCreateOnly(existing, rec, keys...)
		}
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "racks",
		ValidateName: azureNCName("rack", azureNCNameRE),
		Validate: func(payload map[string]any) string {
			props := azureARMObject(payload["properties"])
			return cmp.Or(azureNCExtendedLocation(payload), azureDataBoxObjects(payload, "", "properties"),
				azureWorkloadsRequired(props, "properties.", "availabilityZone", "rackLocation", "rackSerialNumber"),
				azureNCIDOf(props, "properties.", "rackSkuId", "a rack SKU", azureNCRackSkuID, true), azureNCRackDefinition(props, "properties.", false))
		},
		PatchBody: patchOnly(func(props map[string]any) string {
			return cmp.Or(azureLabLength(props, "properties.", "rackLocation", 0, 256), azureLabLength(props, "properties.", "rackSerialNumber", 1, 64))
		}, "rackLocation", "rackSerialNumber"),
		StateProperties: azureNCRackState,
		Reconcile:       createOnly(),
		ValidateDelete:  azureNCMemberDelete,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "bareMetalMachines",
		ValidateName: azureNCName("bare metal machine", azureNCMachineRE),
		Validate: func(payload map[string]any) string {
			props := azureARMObject(payload["properties"])
			return cmp.Or(azureNCExtendedLocation(payload), azureDataBoxObjects(payload, "", "properties"),
				azureWorkloadsRequired(props, "properties.", "bmcConnectionString", "machineDetails", "machineName", "machineSkuId"),
				azureNCMachineConfig(props, "properties.", true), azureNCIDOf(props, "properties.", "rackId", "a rack", azureNCRackID, true))
		},
		WriteOnlyProperties: []string{"bmcCredentials.password"},
		PatchBody: patchOnly(func(props map[string]any) string {
			return azureLabLength(props, "properties.", "machineDetails", 0, 256)
		}, "machineDetails"),
		StateProperties: azureNCMachineState,
		Reconcile:       createOnly(),
		ValidateDelete:  azureNCMemberDelete,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"cordon":                    azureNCMachineSet("cordon", map[string][]string{"evacuate": azureNCTrueFalse}, "", map[string]any{"cordonStatus": "Cordoned"}),
			"uncordon":                  azureNCMachineSet("uncordon", nil, "", map[string]any{"cordonStatus": "Uncordoned"}),
			"poweroff":                  azureNCMachineSet("powerOff", map[string][]string{"skipShutdown": azureNCTrueFalse}, "On", map[string]any{"powerState": "Off", "readyState": "False"}),
			"start":                     azureNCMachineSet("start", nil, "Off", map[string]any{"powerState": "On", "readyState": "True"}),
			"restart":                   azureNCMachineSet("restart", nil, "On", map[string]any{"readyState": "True"}),
			"reimage":                   azureNCMachineSet("reimage", map[string][]string{"safeguardMode": azureNCSafeguard}, "", map[string]any{"readyState": "True"}),
			"replace":                   azureNCMachineReplace,
			"runcommand":                azureNCMachineRun("runCommand", "BareMetalMachineRunCommand", azureNCScript),
			"runreadcommands":           azureNCMachineRun("runReadCommands", "BareMetalMachineRunReadCommands", azureNCCommands),
			"rundataextracts":           azureNCMachineRun("runDataExtracts", "BareMetalMachineRunDataExtracts", azureNCCommands),
			"rundataextractsrestricted": azureNCMachineRun("runDataExtractsRestricted", "BareMetalMachineRunDataExtractsRestricted", azureNCCommands),
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "storageAppliances",
		ValidateName: azureNCName("storage appliance", azureNCNameRE),
		Validate: func(payload map[string]any) string {
			props := azureARMObject(payload["properties"])
			if msg := cmp.Or(azureNCExtendedLocation(payload), azureDataBoxObjects(payload, "", "properties"),
				azureWorkloadsRequired(props, "properties.", "serialNumber", "storageApplianceSkuId"),
				azureNCRequiredKeys(props, "properties.", true, "rackSlot", "administratorCredentials"),
				azureOracleNumber(props, "properties.", "rackSlot", 1, 256, true),
				azureNCIDOf(props, "properties.", "rackId", "a rack", azureNCRackID, true)); msg != "" {
				return msg
			}
			return azureWorkloadsRequired(azureARMObject(props["administratorCredentials"]), "properties.administratorCredentials.", "username", "password")
		},
		WriteOnlyProperties: []string{"administratorCredentials.password"},
		PatchBody: patchOnly(func(props map[string]any) string {
			return azureLabLength(props, "properties.", "serialNumber", 1, 64)
		}, "serialNumber"),
		StateProperties: azureNCApplianceState,
		Reconcile:       createOnly("administratorCredentials", "rackId", "rackSlot", "storageApplianceSkuId"),
		ValidateDelete:  azureNCMemberDelete,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"enableremotevendormanagement":  azureNCVendorManagement("Enabled"),
			"disableremotevendormanagement": azureNCVendorManagement("Disabled"),
			"runreadcommands": func(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
				if msg := azureNCCommands(payload); msg != "" {
					return azureOracleBad(msg)
				}
				return s.azureNCMemberAction(rec, "Available", func(_, cluster *azureARMResource) (map[string]any, map[string]any, string) {
					return nil, azureNCOutput(cluster, "StorageRunReadCommands", rec.ID), ""
				}, "runReadCommands")
			},
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "accessBridges",
		Validate: func(payload map[string]any) string {
			props := azureARMObject(payload["properties"])
			return cmp.Or(azureNCExtendedLocation(payload), azureDataBoxObjects(payload, "", "properties"),
				azureNCIDOf(props, "properties.", "networkId", "a network", azureNCAnyID, true),
				azureNCPrefix(props, "ipv4ConnectedPrefix", true), azureNCPrefix(props, "ipv6ConnectedPrefix", false),
				azureNCSecurityRules(props))
		},
		PatchBody: patchOnly(azureNCSecurityRules, "securityRules"),
		Reconcile: func(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
			return nil, azureNCCreateOnly(existing, rec, "networkId", "ipv4ConnectedPrefix", "ipv6ConnectedPrefix")
		},
		ReadOnlyProperties: azureNCBridgeReadOnly,
	})
}

// azureNCMachineProps returns a machine of rack number rack (from 1).
func azureNCMachineProps(machine map[string]any, name, rackID string, rack int, controller bool) map[string]any {
	slot := azureHZCount(machine["rackSlot"])
	bmc := fmt.Sprintf("10.10.%d.%d", rack%250, 10+slot%240)
	role := "platform.afo-nc.microsoft.com/compute=true"
	if controller {
		role = "platform.afo-nc.microsoft.com/management-plane=true"
	}
	return map[string]any{
		"bmcConnectionString": "redfish+https://" + bmc + "/redfish/v1/Systems/System.Embedded.1", "bmcIpv4Address": bmc,
		"bmcCredentials": map[string]any{"username": azureARMObject(machine["bmcCredentials"])["username"]},
		"bmcMacAddress":  machine["bmcMacAddress"], "bootMacAddress": machine["bootMacAddress"],
		"machineDetails": azureARMString(machine["machineDetails"]), "machineName": name, "machineSkuId": azureNCMachineModel,
		"machineRoles": []any{role}, "rackId": rackID, "rackSlot": slot, "serialNumber": machine["serialNumber"], "serviceTag": machine["serialNumber"],
		"detailedStatus": "Preparing", "detailedStatusMessage": "The bare metal machine is waiting for the cluster to deploy.",
		"powerState": "Off", "readyState": "False", "cordonStatus": "Uncordoned",
	}
}

// azureNCApplianceProps returns an appliance of rack number rack (from 1).
func azureNCApplianceProps(appliance map[string]any, rackID string, rack int) map[string]any {
	return map[string]any{
		"administratorCredentials": map[string]any{"username": azureARMObject(appliance["adminCredentials"])["username"]},
		"rackId":                   rackID, "rackSlot": azureHZCount(appliance["rackSlot"]), "serialNumber": appliance["serialNumber"],
		"storageApplianceSkuId": azureNCApplianceModel, "model": azureNCApplianceModel, "manufacturer": "Contoso Storage", "version": "6.5.0",
		"capacity": azureNCApplianceGB, "capacityUsed": 0, "managementIpv4Address": fmt.Sprintf("10.10.%d.%d", rack%250, 200+azureHZCount(appliance["rackSlot"])%50),
		"detailedStatus": "Provisioning", "detailedStatusMessage": "The storage appliance is waiting for the cluster to deploy.",
		"remoteVendorManagementFeature": "Supported", "remoteVendorManagementStatus": "Disabled",
	}
}

// azureNCMemberAction runs an action on a rack member (a machine or an
// appliance) whose detailedStatus is want. change returns the properties
// to store, the operation status properties, or a 409 message.
func (s *Server) azureNCMemberAction(rec *azureARMResource, want string, change func(stored, cluster *azureARMResource) (map[string]any, map[string]any, string), action string) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	stored := s.azureARMResources[azureARMKey(rec.ID)]
	if stored == nil {
		return azureNCMissing()
	}
	if status := azureARMString(stored.Properties["detailedStatus"]); status != want {
		return azureOracleConflict("The resource's detailedStatus is " + status + "; " + action + " needs " + want + ", which it reaches once its cluster is deployed.")
	}
	props, result, msg := change(stored, s.azureNCClusterOfLocked(stored))
	if msg != "" {
		return azureOracleConflict(msg)
	}
	if strings.EqualFold(stored.Type, azureNCMachineType) {
		props = azureARMMerge(props, map[string]any{"actionStates": azureNCActionState(stored.Properties["actionStates"], azureNCMachineType, action)})
	}
	if len(props) > 0 {
		s.azureOracleUpdateLocked(stored.ID, props)
	}
	return azureNCDone(stored, action, result)
}

// azureNCMachineSet returns a machine action that checks enums in its
// body, needs powerState to be power (any, when empty), and stores props.
func azureNCMachineSet(action string, enums map[string][]string, power string, props map[string]any) func(*Server, *azureARMResource, map[string]any) (int, any) {
	return func(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
		if msg := azureWorkloadsEnums(payload, "", nil, enums); msg != "" {
			return azureOracleBad(msg)
		}
		return s.azureNCMemberAction(rec, "Provisioned", func(stored, _ *azureARMResource) (map[string]any, map[string]any, string) {
			if state := azureARMString(stored.Properties["powerState"]); power != "" && state != power {
				return nil, nil, "The bare metal machine's powerState is " + state + "; " + action + " needs " + power + "."
			}
			return props, nil, ""
		}, action)
	}
}

// azureNCMachineRun returns a command action that reports where its output
// went, as the cluster's command output settings direct.
func azureNCMachineRun(action, outputType string, check func(map[string]any) string) func(*Server, *azureARMResource, map[string]any) (int, any) {
	return func(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
		if msg := check(payload); msg != "" {
			return azureOracleBad(msg)
		}
		return s.azureNCMemberAction(rec, "Provisioned", func(_, cluster *azureARMResource) (map[string]any, map[string]any, string) {
			return nil, azureNCOutput(cluster, outputType, rec.ID+"#"+action), ""
		}, action)
	}
}

// azureNCOutput returns the operation status properties of a command run:
// the container its output type's override names, else the cluster's.
func azureNCOutput(cluster *azureARMResource, outputType, seed string) map[string]any {
	settings := map[string]any{}
	if cluster != nil {
		settings = azureARMObject(cluster.Properties["commandOutputSettings"])
	}
	container := azureARMString(settings["containerUrl"])
	for _, raw := range azureARMSlice(settings["overrides"]) {
		if override := azureARMObject(raw); override["commandOutputType"] == outputType && azureARMString(override["containerUrl"]) != "" {
			container = azureARMString(override["containerUrl"])
		}
	}
	if container == "" {
		container = "https://nexus" + azureWorkloadsHash(seed, 10) + ".blob.core.windows.net/command-output"
	}
	blob := strings.TrimSuffix(container, "/") + "/" + azureHZGUID(seed) + ".tar.gz"
	return map[string]any{"exitCode": "0", "outputHead": "====Action Command Output====\n", "resultRef": blob, "resultUrl": blob}
}

// azureNCMachineReplace records the new hardware of a replaced machine.
func azureNCMachineReplace(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := cmp.Or(
		azureWorkloadsEnums(payload, "", nil, map[string][]string{"safeguardMode": azureNCSafeguard, "storagePolicy": {"Preserve", "DiscardAll"}}),
		azureNCMachineConfig(payload, "", false),
		azureDataBoxObjects(payload, "", "bmcCredentials"),
	); msg != "" {
		return azureOracleBad(msg)
	}
	if creds, present := payload["bmcCredentials"]; present {
		if msg := azureWorkloadsRequired(azureARMObject(creds), "bmcCredentials.", "username", "password"); msg != "" {
			return azureOracleBad(msg)
		}
	}
	props := map[string]any{}
	for _, key := range []string{"bmcMacAddress", "bootMacAddress", "machineName", "serialNumber"} {
		if value, present := payload[key]; present {
			props[key] = value
		}
	}
	if creds := azureARMObject(payload["bmcCredentials"]); creds != nil {
		props["bmcCredentials"] = map[string]any{"username": creds["username"]}
	}
	return s.azureNCMemberAction(rec, "Provisioned", func(*azureARMResource, *azureARMResource) (map[string]any, map[string]any, string) {
		return props, nil, ""
	}, "replace")
}

// azureNCVendorManagement returns the action that turns remote vendor
// management of an available storage appliance on or off.
func azureNCVendorManagement(state string) func(*Server, *azureARMResource, map[string]any) (int, any) {
	action := map[string]string{"Enabled": "enableRemoteVendorManagement", "Disabled": "disableRemoteVendorManagement"}[state]
	return func(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
		if msg := azureOracleStrings(payload, "", "supportEndpoints", 0, 100); msg != "" {
			return azureOracleBad(msg)
		}
		for _, endpoint := range azureOracleStringsList(payload["supportEndpoints"]) {
			if !azureNCIP(endpoint) && !azureNCIsPrefix(endpoint) {
				return azureOracleBad("supportEndpoints must hold IP addresses or CIDR ranges; '" + endpoint + "' isn't one.")
			}
		}
		return s.azureNCMemberAction(rec, "Available", func(stored, _ *azureARMResource) (map[string]any, map[string]any, string) {
			return map[string]any{"remoteVendorManagementStatus": state}, nil, ""
		}, action)
	}
}

func azureNCIP(text string) bool {
	_, err := netip.ParseAddr(text)
	return err == nil
}

func azureNCIsPrefix(text string) bool {
	_, err := netip.ParsePrefix(text)
	return err == nil
}

// azureNCPrefix checks an optional connected prefix of the family.
func azureNCPrefix(props map[string]any, key string, v4 bool) string {
	raw, present := props[key]
	if !present {
		return ""
	}
	prefix, err := netip.ParsePrefix(azureARMString(raw))
	if err != nil || prefix.Addr().Is4() != v4 {
		return "properties." + key + " must be an " + map[bool]string{true: "IPv4", false: "IPv6"}[v4] + " CIDR prefix."
	}
	return ""
}

// azureNCSecurityRules checks access bridge security rules: a direction, a
// port or port range, and addresses, CIDR ranges or address ranges.
func azureNCSecurityRules(props map[string]any) string {
	for i, raw := range azureARMSlice(props["securityRules"]) {
		rule := azureARMObject(raw)
		path := "properties.securityRules[" + strconv.Itoa(i) + "]."
		if msg := cmp.Or(azureWorkloadsEnums(rule, path, []string{"direction", "port"}, map[string][]string{"direction": {"Inbound", "Outbound"}}),
			azureNCPort(azureARMString(rule["port"]), path)); msg != "" {
			return msg
		}
		for key, v4 := range map[string]bool{"ipv4Addresses": true, "ipv6Addresses": false} {
			for _, address := range azureOracleStringsList(rule[key]) {
				if !azureNCAddressRange(address, v4) {
					return path + key + " holds '" + address + "', which isn't an address, CIDR range or address range of that family."
				}
			}
		}
	}
	return ""
}

func azureNCPort(port, path string) string {
	match := azureNCPortRE.FindStringSubmatch(port)
	if match == nil {
		return path + "port must be a port or port range such as 24562 or 24562-24570."
	}
	low, _ := strconv.Atoi(match[1])
	high := low
	if match[2] != "" {
		high, _ = strconv.Atoi(match[2])
	}
	if low < 1 || high > 65535 || low > high {
		return path + "port must be a port from 1 to 65535, or a range whose start isn't after its end."
	}
	return ""
}

func azureNCAddressRange(text string, v4 bool) bool {
	if prefix, err := netip.ParsePrefix(text); err == nil {
		return prefix.Addr().Is4() == v4
	}
	low, high, isRange := strings.Cut(text, "-")
	a, errA := netip.ParseAddr(low)
	if !isRange {
		return errA == nil && a.Is4() == v4
	}
	b, errB := netip.ParseAddr(high)
	return errA == nil && errB == nil && a.Is4() == v4 && b.Is4() == v4 && a.Compare(b) <= 0
}

// azureNCBridgeReadOnly reports a running bridge with one endpoint, at the
// tenth address of each connected prefix.
func azureNCBridgeReadOnly(rec *azureARMResource) map[string]any {
	endpoint := map[string]any{"name": "vip", "fqdn": rec.Name + "." + azureWorkloadsHash(rec.ID, 8) + ".bridge.nexus.azure.com"}
	for key, field := range map[string]string{"ipv4ConnectedPrefix": "ipv4Address", "ipv6ConnectedPrefix": "ipv6Address"} {
		if prefix, err := netip.ParsePrefix(azureARMString(rec.Properties[key])); err == nil {
			addr := prefix.Masked().Addr()
			for range 10 {
				addr = addr.Next()
			}
			if prefix.Contains(addr) {
				endpoint[field] = addr.String()
			}
		}
	}
	return map[string]any{
		"detailedStatus": "Running", "detailedStatusMessage": "The access bridge is serving traffic.", "protocol": "TCP",
		"endpoints": []any{endpoint},
	}
}
