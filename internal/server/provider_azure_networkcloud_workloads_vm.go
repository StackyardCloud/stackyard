package server

import (
	"cmp"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var (
	azureNCWAdminUser  = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,31}$`)
	azureNCWVolumeID   = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.NetworkCloud/volumes/[^/]+$`)
	azureNCWMachineID  = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.HybridCompute/machines/[^/]+$`)
	azureNCWConsoleKey = regexp.MustCompile(`^default$`)
	// azureNCWVMCreateOnly are the virtual machine properties a later PUT
	// can't change; only the image credentials can.
	azureNCWVMCreateOnly = []string{"adminUsername", "bootMethod", "cloudServicesNetworkAttachment", "consoleExtendedLocation", "cpuCores", "isolateEmulatorThread",
		"memorySizeGB", "networkAttachments", "networkData", "placementHints", "sshPublicKeys", "storageProfile", "userData", "virtioInterface", "vmDeviceModel", "vmImage"}
)

func azureNCWRegisterVMs() {
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureNCWNamespace, Type: "virtualMachines", ETag: true,
		ValidateName:        azureNCWNameHook("virtualMachines", azureNCWVMName),
		Validate:            azureNCWVMValidate,
		WriteOnlyProperties: []string{"vmImageRepositoryCredentials.password", "userDataContent", "networkDataContent"},
		StateProperties:     map[string]any{"powerState": "On", "detailedStatus": "Running", "detailedStatusMessage": "Virtual machine is running."},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"volumes": azureNCWList(azureARMSlice(azureARMObject(rec.Properties["storageProfile"])["volumeAttachments"]))}
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			props := azureARMObject(payload["properties"])
			creds := azureARMObject(props["vmImageRepositoryCredentials"])
			return payload, nil, cmp.Or(azureOracleOnly(payload, "", []string{"tags", "identity", "properties"}), azureDataBoxObjects(payload, "", "identity", "properties"),
				azureOracleOnly(props, "properties.", []string{"vmImageRepositoryCredentials"}), azureDataBoxObjects(props, "properties.", "vmImageRepositoryCredentials"),
				azureOracleOnly(creds, "properties.vmImageRepositoryCredentials.", []string{"password", "registryUrl", "username"}),
				azureLabLength(creds, "properties.vmImageRepositoryCredentials.", "password", 1, 1<<20),
				azureLabLength(creds, "properties.vmImageRepositoryCredentials.", "username", 1, 1<<20))
		},
		Reconcile: azureNCWReconcile(map[string]any{"bootMethod": "UEFI", "isolateEmulatorThread": "True", "virtioInterface": "Modern", "vmDeviceModel": "T2"}, nil, azureNCWVMReconcile),
		Deleted:   func(s *Server, removed *azureARMResource) { s.azureNCWRelinkLocked(nil, removed.ID) },
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"start": azureNCWVMPower("start"), "poweroff": azureNCWVMPower("powerOff"),
			"restart": azureNCWVMPower("restart"), "reimage": azureNCWVMPower("reimage"),
			"assignrelay": azureNCWAssignRelay,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureNCWNamespace, Type: "virtualMachines/consoles", ETag: true,
		ValidateName: azureNCWNameHook("console", azureNCWConsoleKey),
		Validate: func(payload map[string]any) string {
			props := azureARMObject(payload["properties"])
			return cmp.Or(azureNCWExtendedLocation(payload, true), azureNCWPresent(payload, "", "properties"),
				azureNCWPresent(props, "properties.", "enabled", "sshPublicKey"), azureNCWConsole(props))
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"detailedStatus": "Ready", "detailedStatusMessage": "Console is ready.", "virtualMachineAccessId": azureHZGUID(rec.ID)}
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			props := azureARMObject(payload["properties"])
			return payload, nil, cmp.Or(azureOracleOnly(payload, "", []string{"tags", "properties"}),
				azureOracleOnly(props, "properties.", []string{"enabled", "expiration", "sshPublicKey"}), azureNCWConsole(props))
		},
		Reconcile: azureNCWReconcile(nil, nil, nil),
	})
}

// azureNCWConsole checks a console's enabled flag, expiration and key.
func azureNCWConsole(props map[string]any) string {
	if msg := cmp.Or(azureLabEnum(props, "properties.", "enabled", azureNCWBools), azureDataBoxObjects(props, "properties.", "sshPublicKey")); msg != "" {
		return msg
	}
	if raw, present := props["expiration"]; present {
		if _, err := time.Parse(time.RFC3339, azureARMString(raw)); err != nil {
			return "properties.expiration must be an RFC 3339 date-time."
		}
	}
	if key, present := props["sshPublicKey"]; present {
		return azureLabLength(azureARMObject(key), "properties.sshPublicKey.", "keyData", 1, 1<<20)
	}
	return ""
}

func azureNCWVMValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if msg := cmp.Or(azureNCWExtendedLocation(payload, true), azureNCWPresent(payload, "", "properties"), azureDataBoxObjects(payload, "", "identity"),
		azureNCWPresent(props, "properties.", "adminUsername", "cloudServicesNetworkAttachment", "cpuCores", "memorySizeGB", "storageProfile", "vmImage"),
		azureNCWMatch(props, "properties.", "adminUsername", azureNCWAdminUser, "1 to 32 lowercase letters, digits and underscores, not starting with a digit"),
		azureWorkloadsEnums(props, "properties.", []string{"vmImage"}, map[string][]string{
			"bootMethod": {"BIOS", "UEFI"}, "isolateEmulatorThread": azureNCWBools, "virtioInterface": {"Modern", "Transitional"}, "vmDeviceModel": {"T1", "T2", "T3"},
		}),
		azureOracleNumber(props, "properties.", "cpuCores", 2, 1<<53, true), azureOracleNumber(props, "properties.", "memorySizeGB", 1, 1<<53, true),
		azureDataBoxObjects(props, "properties.", "cloudServicesNetworkAttachment", "storageProfile", "vmImageRepositoryCredentials"),
		azureNCWSSHKeys(props, "properties."), azureNCWConsoleLocation(props)); msg != "" {
		return msg
	}
	attachments, msg := azureNCWObjects(props, "properties.", "networkAttachments", 0)
	if msg != "" {
		return msg
	}
	all := append([]map[string]any{azureARMObject(props["cloudServicesNetworkAttachment"])}, attachments...)
	gateways, names := 0, map[string]bool{}
	for i, a := range all {
		path := "properties.networkAttachments[]."
		if i == 0 {
			path = "properties.cloudServicesNetworkAttachment."
		}
		if msg := azureNCWAttachment(a, path); msg != "" {
			return msg
		}
		if a["defaultGateway"] == "True" {
			gateways++
		}
		if name := azureARMString(a["networkAttachmentName"]); name != "" && i > 0 {
			if names[name] {
				return "properties.networkAttachments networkAttachmentName '" + name + "' is used twice."
			}
			names[name] = true
		}
	}
	if gateways > 1 {
		return "Only one network attachment may set defaultGateway to True."
	}
	msg = cmp.Or(azureNCWPlacement(props), azureNCWStorage(azureARMObject(props["storageProfile"])))
	if creds, present := props["vmImageRepositoryCredentials"]; present && msg == "" {
		msg = azureWorkloadsRequired(azureARMObject(creds), "properties.vmImageRepositoryCredentials.", "password", "registryUrl", "username")
	}
	return msg
}

// azureNCWConsoleLocation checks the optional consoleExtendedLocation like
// a top-level extendedLocation.
func azureNCWConsoleLocation(props map[string]any) string {
	loc, present := props["consoleExtendedLocation"]
	if !present {
		return ""
	}
	return strings.Replace(azureNCWExtendedLocation(map[string]any{"extendedLocation": loc}, true), "extendedLocation", "properties.consoleExtendedLocation", 1)
}

func azureNCWAttachment(a map[string]any, path string) string {
	if msg := cmp.Or(azureWorkloadsEnums(a, path, []string{"attachedNetworkId", "ipAllocationMethod"}, map[string][]string{
		"ipAllocationMethod": {"Dynamic", "Static", "Disabled"}, "defaultGateway": azureNCWBools,
	}), azureNCWMatch(a, path, "attachedNetworkId", azureNCWARMID, "a resource ID"), azureLabLength(a, path, "networkAttachmentName", 0, 15)); msg != "" {
		return msg
	}
	for _, key := range []string{"ipv4Address", "ipv6Address"} {
		if raw, present := a[key]; present && net.ParseIP(azureARMString(raw)) == nil {
			return path + key + " must be an IP address."
		}
	}
	return ""
}

func azureNCWSSHKeys(m map[string]any, path string) string {
	keys, msg := azureNCWObjects(m, path, "sshPublicKeys", 0)
	for _, key := range keys {
		msg = cmp.Or(msg, azureWorkloadsRequired(key, path+"sshPublicKeys[].", "keyData"))
	}
	return msg
}

func azureNCWPlacement(props map[string]any) string {
	hints, msg := azureNCWObjects(props, "properties.", "placementHints", 0)
	for _, h := range hints {
		path := "properties.placementHints[]."
		msg = cmp.Or(msg, azureWorkloadsEnums(h, path, []string{"hintType", "resourceId", "schedulingExecution"}, map[string][]string{
			"hintType": {"Affinity", "AntiAffinity"}, "schedulingExecution": {"Hard", "Soft"},
			// Learn's own example sends an empty scope.
			"scope": {"Machine", "Rack", ""},
		}), azureNCWMatch(h, path, "resourceId", azureNCWARMID, "a resource ID"))
	}
	return msg
}

func azureNCWStorage(profile map[string]any) string {
	disk := azureARMObject(profile["osDisk"])
	if msg := cmp.Or(azureNCWPresent(profile, "properties.storageProfile.", "osDisk"), azureDataBoxObjects(profile, "properties.storageProfile.", "osDisk"),
		azureNCWPresent(disk, "properties.storageProfile.osDisk.", "diskSizeGB"),
		azureOracleNumber(disk, "properties.storageProfile.osDisk.", "diskSizeGB", 1, 1<<53, true),
		azureLabEnum(disk, "properties.storageProfile.osDisk.", "createOption", []string{"Ephemeral", "Persistent"}),
		azureLabEnum(disk, "properties.storageProfile.osDisk.", "deleteOption", []string{"Delete"}),
		azureOracleStrings(profile, "properties.storageProfile.", "volumeAttachments", 0, 1<<20)); msg != "" {
		return msg
	}
	for _, id := range azureARMSlice(profile["volumeAttachments"]) {
		if !azureNCWVolumeID.MatchString(azureARMString(id)) {
			return "properties.storageProfile.volumeAttachments must hold Network Cloud volume IDs."
		}
	}
	return ""
}

// azureNCWVMReconcile checks the networks and volumes a virtual machine
// attaches exist, keeps its create-only properties, gives each attachment
// a MAC address and records the machine's network and volume use.
func azureNCWVMReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	out := map[string]any{}
	if existing != nil {
		if msg := azureNCWKeep(azureNCWWithoutMACs(existing.Properties), azureNCWWithoutMACs(rec.Properties), "properties.", azureNCWVMCreateOnly, out); msg != "" {
			return nil, msg
		}
	}
	props := azureARMMerge(rec.Properties, out)
	csn := azureARMObject(props["cloudServicesNetworkAttachment"])
	if !s.azureNCWExistsLocked(azureARMString(csn["attachedNetworkId"]), "cloudServicesNetworks") {
		return nil, "properties.cloudServicesNetworkAttachment.attachedNetworkId must name an existing cloud services network."
	}
	var attachments []any
	for i, item := range azureARMSlice(props["networkAttachments"]) {
		a := azureARMObject(item)
		if msg := s.azureNCWAttachableLocked(a); msg != "" {
			return nil, msg
		}
		attachments = append(attachments, azureARMMerge(a, map[string]any{"macAddress": azureNCWMAC(rec.ID, i+1)}))
	}
	for _, id := range azureARMSlice(azureARMObject(props["storageProfile"])["volumeAttachments"]) {
		if !s.azureNCWExistsLocked(azureARMString(id), "volumes") {
			return nil, "properties.storageProfile.volumeAttachments: volume " + azureARMString(id) + " doesn't exist."
		}
	}
	out["cloudServicesNetworkAttachment"] = azureARMMerge(csn, map[string]any{"macAddress": azureNCWMAC(rec.ID, 0)})
	if attachments != nil {
		out["networkAttachments"] = attachments
	}
	linked := *rec
	linked.Properties = azureARMMerge(props, out)
	s.azureNCWRelinkLocked(&linked, "")
	return out, ""
}

// azureNCWAttachableLocked checks a network attachment against the network
// it names: Dynamic and Static allocation need an L3 network, and a Static
// address must sit in that network's prefix. Caller holds
// providerStorageMu.
func (s *Server) azureNCWAttachableLocked(a map[string]any) string {
	id, method := azureARMString(a["attachedNetworkId"]), azureARMString(a["ipAllocationMethod"])
	network := s.azureARMResources[azureARMKey(id)]
	kind := ""
	if network != nil {
		kind = strings.TrimPrefix(network.Type, azureNCWNamespace+"/")
	}
	switch {
	case !slices.Contains(azureNCWNetworks, kind):
		return "properties.networkAttachments: " + id + " isn't an existing L2, L3 or trunked network."
	case kind != "l3Networks" && method != "Disabled":
		return "properties.networkAttachments: ipAllocationMethod must be Disabled for " + kind + "; Dynamic and Static are only valid for an L3 network."
	case method != "Static":
		return ""
	}
	for _, family := range [][2]string{{"ipv4Address", "ipv4ConnectedPrefix"}, {"ipv6Address", "ipv6ConnectedPrefix"}} {
		prefix := azureARMString(network.Properties[family[1]])
		if prefix == "" {
			continue
		}
		_, subnet, err := net.ParseCIDR(prefix)
		ip := net.ParseIP(azureARMString(a[family[0]]))
		if err == nil && (ip == nil || !subnet.Contains(ip)) {
			return "properties.networkAttachments: a Static " + family[0] + " within " + prefix + " is required."
		}
	}
	return ""
}

// azureNCWWithoutMACs returns props without the server-assigned MAC
// addresses, as the client sent them.
func azureNCWWithoutMACs(props map[string]any) map[string]any {
	out := azureARMMerge(props, nil)
	if csn, ok := props["cloudServicesNetworkAttachment"].(map[string]any); ok {
		out["cloudServicesNetworkAttachment"] = azureARMWithout(csn, "macAddress")
	}
	if items, ok := props["networkAttachments"].([]any); ok {
		copied := make([]any, len(items))
		for i, item := range items {
			copied[i] = azureARMWithout(azureARMObject(item), "macAddress")
		}
		out["networkAttachments"] = copied
	}
	return out
}

// azureNCWMAC is a stable, locally administered MAC address.
func azureNCWMAC(id string, n int) string {
	h := azureWorkloadsHash(id+"#"+strconv.Itoa(n), 10)
	return fmt.Sprintf("02:%s:%s:%s:%s:%s", h[0:2], h[2:4], h[4:6], h[6:8], h[8:10])
}

// azureNCWVMPower runs a power action. Restarting needs a running machine;
// the others settle the machine in the state they lead to.
func azureNCWVMPower(action string) func(*Server, *azureARMResource, map[string]any) (int, any) {
	return func(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
		if action == "powerOff" {
			if msg := cmp.Or(azureOracleOnly(payload, "", []string{"skipShutdown"}), azureLabEnum(payload, "", "skipShutdown", azureNCWBools)); msg != "" {
				return azureOracleBad(msg)
			}
		}
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		current := s.azureARMResources[azureARMKey(rec.ID)]
		if current == nil {
			return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
		}
		if action == "restart" && current.Properties["powerState"] == "Off" {
			return azureOracleConflict("Virtual machine " + rec.Name + " is powered off; start it instead of restarting it.")
		}
		state := map[string]any{"powerState": "On", "detailedStatus": "Running", "detailedStatusMessage": "Virtual machine is running."}
		if action == "powerOff" {
			state = map[string]any{"powerState": "Off", "detailedStatus": "Stopped", "detailedStatusMessage": "Virtual machine is powered off."}
		}
		s.azureOracleUpdateLocked(rec.ID, state)
		return http.StatusAccepted, nil
	}
}

// azureNCWAssignRelay checks the Arc machine a relay is assigned to.
func azureNCWAssignRelay(_ *Server, _ *azureARMResource, payload map[string]any) (int, any) {
	if msg := cmp.Or(azureOracleOnly(payload, "", []string{"machineId", "relayType"}), azureNCWPresent(payload, "", "machineId"),
		azureNCWMatch(payload, "", "machineId", azureNCWMachineID, "a Microsoft.HybridCompute/machines resource ID"),
		azureLabEnum(payload, "", "relayType", []string{"Platform", "Public"})); msg != "" {
		return azureOracleBad(msg)
	}
	return http.StatusAccepted, nil
}
