package server

import (
	"cmp"
	"encoding/base64"
	"net/http"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Azure Operator Nexus (Microsoft.NetworkCloud 2026-07-01), infrastructure
// half: cluster managers, clusters with their key sets and metrics
// configuration, the racks, bare metal machines and storage appliances a
// cluster's rack definitions describe (provider_azure_networkcloud_infra.go
// and _machines.go), access bridges, and the rack SKU and Kubernetes
// version catalogs. Workloads (networks, volumes, virtual machines,
// Kubernetes clusters) are in provider_azure_networkcloud_workloads*.go.
// No hardware exists: long-running operations finish at once.

const (
	azureNCNamespace      = "Microsoft.NetworkCloud"
	azureNCClusterType    = azureNCNamespace + "/clusters"
	azureNCManagerType    = azureNCNamespace + "/clusterManagers"
	azureNCRackType       = azureNCNamespace + "/racks"
	azureNCMachineType    = azureNCNamespace + "/bareMetalMachines"
	azureNCApplianceType  = azureNCNamespace + "/storageAppliances"
	azureNCCustomLocation = "Microsoft.ExtendedLocation/customLocations"
)

var (
	azureNCNameRE    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-_]{0,28}[a-zA-Z0-9]$`)
	azureNCMachineRE = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9]{0,62}[a-zA-Z0-9]$`)
	azureNCMACRE     = regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}$`)
	azureNCOSGroupRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
	// azureNCVersions are the cluster versions cluster managers offer,
	// oldest first, with their support end dates.
	azureNCVersions = [][2]string{{"4.1.0", "2026-12-31"}, {"4.2.0", "2027-06-30"}, {"4.3.0", "2027-12-31"}}
	// azureNCKubernetesVersions are the versions kubernetesVersions/default
	// lists, for Kubernetes clusters to choose from.
	azureNCKubernetesVersions = []string{"1.32.7", "1.33.3", "1.34.1"}
	// azureNCManagedCredentials are the credentials rotateCredential accepts.
	azureNCManagedCredentials = []string{"Bare Metal Machine Identity - console", "Bare Metal Machine Identity - root", "BMC Credential", "local-path-provisioner volume passphrase", "Storage Appliance User"}
	// azureNCOptionalMetrics are the metrics a metrics configuration can
	// enable on top of the core set.
	azureNCOptionalMetrics = []string{"kubernetes_apiserver_request_duration", "node_cpu_frequency_hertz", "node_disk_io_time_seconds", "node_network_receive_errors", "nexus_storage_appliance_latency"}
	azureNCIdentityTypes   = []string{"None", "SystemAssigned", "UserAssigned", "SystemAssigned,UserAssigned"}
	azureNCTrueFalse       = []string{"True", "False"}
	azureNCSafeguard       = []string{"All", "None"}
)

func init() {
	ns := azureNCNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusterManagers",
		ValidateName: azureNCName("cluster manager", azureNCNameRE),
		Validate:     azureNCManagerValidate,
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			if msg := azureNCIdentity(payload); msg != "" {
				return nil, nil, msg
			}
			return azureLabPatchOnly(payload, func(map[string]any) string { return "" })
		},
		Reconcile: func(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
			rec.Identity = azureNCIdentityIDs(rec.Identity, rec.ID)
			azureNCKeepManagedGroup(existing, rec)
			if msg := azureNCCreateOnly(existing, rec, "analyticsWorkspaceId", "availabilityZones", "fabricControllerId", "managedResourceGroupConfiguration", "vmSize"); msg != "" {
				return nil, msg
			}
			return map[string]any{"managedResourceGroupConfiguration": azureNCManagedGroup(rec)}, ""
		},
		ReadOnlyProperties: azureNCManagerReadOnly,
		ValidateDelete: func(s *Server, existing *azureARMResource) string {
			for _, cluster := range s.azureNCClustersLocked() {
				// A cluster created before its manager still sits on the
				// manager's extended location.
				if strings.EqualFold(azureARMString(cluster.Properties["clusterManagerId"]), existing.ID) ||
					strings.EqualFold(azureARMString(cluster.ExtendedLocation["name"]), azureARMString(azureARMObject(existing.Properties["managerExtendedLocation"])["name"])) {
					return "The cluster manager still manages cluster " + cluster.ID + "."
				}
			}
			return ""
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"updaterelayprivateendpointconnection": azureNCRelayConnection,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "kubernetesVersions",
		ValidateName: azureNCDefaultName("Kubernetes version"),
		Validate:     func(payload map[string]any) string { return azureNCExtendedLocation(payload) },
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			return azureLabPatchOnly(payload, func(map[string]any) string { return "" })
		},
		Reconcile: func(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
			return nil, azureNCCreateOnly(existing, rec)
		},
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			var values []any
			for _, version := range azureNCKubernetesVersions {
				values = append(values, map[string]any{"version": version, "description": "Kubernetes " + version})
			}
			return map[string]any{"values": values}
		},
	})
	azureNCRegisterKeySets()
	registerAzureARMView(ns, "rackSkus", func(_ *Server, scope string) []map[string]any {
		var out []map[string]any
		for _, sku := range azureNCRackSkus {
			out = append(out, map[string]any{
				"id": scope + "/providers/" + ns + "/rackSkus/" + sku.name, "name": sku.name, "type": ns + "/rackSkus",
				"properties": sku.fixture(),
			})
		}
		return out
	})
	registerAzureARMView(ns, "operations", func(*Server, string) []map[string]any {
		var ops []map[string]any
		for _, op := range [][2]string{
			{"clusterManagers", "cluster managers"}, {"clusters", "clusters"}, {"racks", "racks"},
			{"bareMetalMachines", "bare metal machines"}, {"storageAppliances", "storage appliances"},
			{"accessBridges", "access bridges"}, {"kubernetesVersions", "Kubernetes versions"},
		} {
			for _, verb := range []string{"read", "write", "delete"} {
				ops = append(ops, map[string]any{
					"name": ns + "/" + op[0] + "/" + verb, "isDataAction": false, "origin": "user,system",
					"display": map[string]any{"provider": "Microsoft Network Cloud", "resource": op[1], "operation": verb + " " + op[1], "description": strings.ToUpper(verb[:1]) + verb[1:] + " " + op[1] + "."},
				})
			}
		}
		return ops
	})
	azureNCRegisterClusters()
	azureNCRegisterMachines()
}

// azureNCName returns a ValidateName hook for names matching re.
func azureNCName(kind string, re *regexp.Regexp) func(string) string {
	return func(name string) string {
		if !re.MatchString(name) {
			return "The " + kind + " name '" + name + "' doesn't match " + re.String() + "."
		}
		return ""
	}
}

// azureNCDefaultName returns a ValidateName hook for singletons named default.
func azureNCDefaultName(kind string) func(string) string {
	return func(name string) string {
		if name != "default" {
			return "The " + kind + " resource must be named 'default'."
		}
		return ""
	}
}

// azureNCIDRE matches resource IDs of one type, such as
// "Microsoft.ManagedNetworkFabric/networkRacks". A type naming several
// levels, such as "a/b/c", matches IDs ending in a/{n}/b/{n}/c/{n}.
func azureNCIDRE(fullType string) *regexp.Regexp {
	parts := strings.Split(fullType, "/")
	path := regexp.QuoteMeta(parts[0])
	for _, part := range parts[1:] {
		path += "/" + regexp.QuoteMeta(part) + "/[^/]+"
	}
	return regexp.MustCompile(`(?i)^/subscriptions/[^/]+(/resourceGroups/[^/]+)?/providers/` + path + `$`)
}

var (
	azureNCWorkspaceID  = azureNCIDRE("Microsoft.OperationalInsights/workspaces")
	azureNCIdentityID   = azureNCIDRE("Microsoft.ManagedIdentity/userAssignedIdentities")
	azureNCKeyVaultID   = azureNCIDRE("Microsoft.KeyVault/vaults")
	azureNCAnyID        = regexp.MustCompile(`(?i)^/subscriptions/[^/]+(/resourceGroups/[^/]+)?/providers/[^/]+/[^/]+/[^/]+(/[^/]+/[^/]+)*$`)
	azureNCPrivateEPID  = azureNCIDRE("Microsoft.Network/privateEndpoints")
	azureNCRackSkuID    = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/providers/Microsoft\.NetworkCloud/rackSkus/([^/]+)$`)
	azureNCControllerID = azureNCIDRE("Microsoft.ManagedNetworkFabric/networkFabricControllers")
)

// azureNCIDOf checks that m[key], when present, is a resource ID re
// matches; required also rejects it missing.
func azureNCIDOf(m map[string]any, path, key, what string, re *regexp.Regexp, required bool) string {
	raw, present := m[key]
	switch {
	case !present && required:
		return path + key + " is required."
	case present && !re.MatchString(azureARMString(raw)):
		return path + key + " must be the resource ID of " + what + "."
	}
	return ""
}

// azureNCExtendedLocation checks the extendedLocation every Network Cloud
// resource but cluster managers needs: a custom location or edge zone.
func azureNCExtendedLocation(payload map[string]any) string {
	loc, ok := payload["extendedLocation"].(map[string]any)
	if !ok {
		return "extendedLocation is required."
	}
	return azureWorkloadsEnums(loc, "extendedLocation.", []string{"name", "type"}, map[string][]string{"type": {"EdgeZone", "CustomLocation"}})
}

// azureNCCustomLocationOf returns the custom location ARM creates for a
// Network Cloud resource, in its resource group, named after it.
func azureNCCustomLocationOf(rec *azureARMResource, suffix string) map[string]any {
	return map[string]any{
		"type": "CustomLocation",
		"name": "/subscriptions/" + rec.SubscriptionID + "/resourceGroups/" + rec.ResourceGroup + "/providers/" + azureNCCustomLocation + "/" + rec.Name + suffix,
	}
}

// azureNCCreateOnly rejects a re-PUT that changes the extendedLocation, the
// kind, or any of the create-only properties keys; one that omits such a
// key keeps the stored value.
func azureNCCreateOnly(existing, rec *azureARMResource, keys ...string) string {
	if existing == nil {
		return ""
	}
	if !strings.EqualFold(azureARMString(existing.ExtendedLocation["name"]), azureARMString(rec.ExtendedLocation["name"])) ||
		azureARMString(existing.ExtendedLocation["type"]) != azureARMString(rec.ExtendedLocation["type"]) {
		return "extendedLocation can't be changed."
	}
	if existing.Kind != rec.Kind {
		return "kind can't be changed."
	}
	for _, key := range keys {
		if _, sent := rec.Properties[key]; !sent {
			if stored, present := existing.Properties[key]; present {
				rec.Properties = azureARMMerge(rec.Properties, map[string]any{key: stored})
			}
			continue
		}
		if !reflect.DeepEqual(existing.Properties[key], rec.Properties[key]) {
			return "properties." + key + " can't be changed after creation."
		}
	}
	return ""
}

// azureNCIdentity checks an optional managed service identity.
func azureNCIdentity(payload map[string]any) string {
	raw, present := payload["identity"]
	if !present {
		return ""
	}
	identity, ok := raw.(map[string]any)
	if !ok {
		return "identity must be an object."
	}
	if msg := azureWorkloadsEnums(identity, "identity.", []string{"type"}, map[string][]string{"type": azureNCIdentityTypes}); msg != "" {
		return msg
	}
	entries, _ := identity["userAssignedIdentities"].(map[string]any)
	hasUser := strings.Contains(azureARMString(identity["type"]), "UserAssigned")
	if hasUser != (len(entries) > 0) {
		return "identity.userAssignedIdentities must list identities exactly when identity.type includes UserAssigned."
	}
	for id := range entries {
		if !azureNCIdentityID.MatchString(id) {
			return "identity.userAssignedIdentities key '" + id + "' must be a user-assigned managed identity ID."
		}
	}
	return ""
}

// azureNCIdentityIDs fills the principal, tenant and client IDs ARM
// assigns. ponytail: PATCH replaces userAssignedIdentities like PUT.
func azureNCIdentityIDs(identity map[string]any, owner string) map[string]any {
	if len(identity) == 0 || identity["type"] == "None" {
		return identity
	}
	out := map[string]any{"type": identity["type"]}
	if strings.HasPrefix(azureARMString(identity["type"]), "SystemAssigned") {
		out["principalId"] = azureHZGUID(owner + "#principal")
		out["tenantId"] = azureHZGUID(owner + "#tenant")
	}
	if entries := azureARMObject(identity["userAssignedIdentities"]); len(entries) > 0 {
		ids := map[string]any{}
		for id := range entries {
			ids[id] = map[string]any{"principalId": azureHZGUID(id + "#principal"), "clientId": azureHZGUID(id + "#client")}
		}
		out["userAssignedIdentities"] = ids
	}
	return out
}

// azureNCKeepManagedGroup fills the subfields of the managed resource group
// configuration a re-PUT sends without from the stored one, so resending
// part of it isn't a change.
func azureNCKeepManagedGroup(existing, rec *azureARMResource) {
	sent, ok := rec.Properties["managedResourceGroupConfiguration"].(map[string]any)
	if existing == nil || !ok {
		return
	}
	stored := azureARMObject(existing.Properties["managedResourceGroupConfiguration"])
	rec.Properties = azureARMMerge(rec.Properties, map[string]any{"managedResourceGroupConfiguration": azureARMMerge(stored, sent)})
}

// azureNCMemberDelete lets only standalone racks, machines and appliances
// be deleted; a cluster's go with the cluster.
func azureNCMemberDelete(s *Server, existing *azureARMResource) string {
	if cluster := s.azureNCClusterOfLocked(existing); cluster != nil {
		return existing.Name + " belongs to cluster " + cluster.ID + "; remove it from the cluster's rack definitions instead."
	}
	return ""
}

// azureNCManagedGroup returns the managed resource group configuration,
// defaulting its name and location.
func azureNCManagedGroup(rec *azureARMResource) map[string]any {
	group := azureARMObject(rec.Properties["managedResourceGroupConfiguration"])
	return map[string]any{
		"name":     cmp.Or(azureARMString(group["name"]), rec.Name+"-managed-"+azureWorkloadsHash(rec.ID, 8)),
		"location": cmp.Or(azureARMString(group["location"]), rec.Location),
	}
}

func azureNCManagerValidate(payload map[string]any) string {
	if msg := cmp.Or(azureDataBoxObjects(payload, "", "properties"), azureNCIdentity(payload),
		azureLabEnum(payload, "", "kind", []string{"Nexus", "AzureLocal"})); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	group := azureARMObject(props["managedResourceGroupConfiguration"])
	return cmp.Or(
		azureNCIDOf(props, "properties.", "fabricControllerId", "a network fabric controller", azureNCControllerID, true),
		azureNCIDOf(props, "properties.", "analyticsWorkspaceId", "a Log Analytics workspace", azureNCWorkspaceID, false),
		azureOracleStrings(props, "properties.", "availabilityZones", 0, 10),
		azureDataBoxObjects(props, "properties.", "managedResourceGroupConfiguration"),
		azureLabLength(group, "properties.managedResourceGroupConfiguration.", "name", 1, 75),
		azureLabLength(props, "properties.", "vmSize", 1, 64),
	)
}

func azureNCManagerReadOnly(rec *azureARMResource) map[string]any {
	group := azureARMString(azureARMObject(rec.Properties["managedResourceGroupConfiguration"])["name"])
	var versions []any
	for _, v := range azureNCVersions {
		versions = append(versions, map[string]any{"targetClusterVersion": v[0], "supportExpiryDate": v[1]})
	}
	return map[string]any{
		"detailedStatus": "Available", "detailedStatusMessage": "The cluster manager is up and running.",
		"clusterVersions":         versions,
		"managerExtendedLocation": azureNCCustomLocationOf(rec, "-cl"),
		"relayConfiguration": map[string]any{
			"relayNamespaceId": "/subscriptions/" + rec.SubscriptionID + "/resourceGroups/" + group + "/providers/Microsoft.Relay/namespaces/" + rec.Name + "-relay",
		},
	}
}

// azureNCRelayConnection approves or rejects a private endpoint connection
// to the cluster manager's relay namespace. ponytail: the relay namespace
// isn't stored, so the decision is only validated.
func azureNCRelayConnection(_ *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := cmp.Or(
		azureWorkloadsEnums(payload, "", []string{"connectionState"}, map[string][]string{"connectionState": {"Approved", "Rejected"}}),
		azureNCIDOf(payload, "", "privateEndpointResourceId", "a private endpoint", azureNCPrivateEPID, true),
	); msg != "" {
		return azureOracleBad(msg)
	}
	return azureNCDone(rec, "updateRelayPrivateEndpointConnection", nil)
}

// azureNCDone answers a finished long-running action with its operation
// status, as the Location URL ARM returns would report it.
func azureNCDone(rec *azureARMResource, action string, props map[string]any) (int, any) {
	id := azureHZGUID(rec.ID + "#" + action + "#" + time.Now().UTC().Format(time.RFC3339Nano))
	now := azureHZTime(time.Now())
	out := map[string]any{
		"id":     "/subscriptions/" + rec.SubscriptionID + "/providers/" + azureNCNamespace + "/locations/" + rec.Location + "/operationStatuses/" + id,
		"name":   id,
		"status": "Succeeded", "percentComplete": 100, "startTime": now, "endTime": now, "resourceId": rec.ID,
	}
	if props != nil {
		out["properties"] = props
	}
	return http.StatusOK, out
}

// azureNCActionState returns states with the action's completed state
// replacing any earlier one of the same type.
func azureNCActionState(states any, typ, action string) []any {
	now := azureHZTime(time.Now())
	actionType := typ + "/" + action + "/action"
	out := []any{map[string]any{
		"actionType": actionType, "correlationId": azureHZGUID(actionType + now), "status": "Completed",
		"startTime": now, "endTime": now, "message": "Action completed successfully",
	}}
	for _, state := range azureARMSlice(states) {
		if azureARMObject(state)["actionType"] != actionType {
			out = append(out, state)
		}
	}
	return out
}

// azureNCCommands checks the body of the run*Commands and runDataExtracts
// actions: one or more commands, and a time limit.
func azureNCCommands(payload map[string]any) string {
	commands, ok := payload["commands"].([]any)
	if !ok || len(commands) == 0 {
		return "commands must list at least one command."
	}
	for i, raw := range commands {
		command := azureARMObject(raw)
		path := "commands[" + strconv.Itoa(i) + "]."
		if msg := cmp.Or(azureWorkloadsRequired(command, path, "command"), azureOracleStrings(command, path, "arguments", 0, 100)); msg != "" {
			return msg
		}
	}
	return azureNCTimeLimit(payload)
}

func azureNCTimeLimit(payload map[string]any) string {
	if _, present := payload["limitTimeSeconds"]; !present {
		return "limitTimeSeconds is required."
	}
	return azureOracleNumber(payload, "", "limitTimeSeconds", 60, 14400, true)
}

// azureNCScript checks a runCommand body: a base64 script and a time limit.
func azureNCScript(payload map[string]any) string {
	script := azureARMString(payload["script"])
	if _, err := base64.StdEncoding.DecodeString(script); script == "" || err != nil {
		return "script must be a base64-encoded script."
	}
	return cmp.Or(azureOracleStrings(payload, "", "arguments", 0, 100), azureNCTimeLimit(payload))
}

// azureNCClustersLocked returns the stored clusters. Caller holds
// providerStorageMu.
func (s *Server) azureNCClustersLocked() []*azureARMResource {
	return s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool { return strings.EqualFold(r.Type, azureNCClusterType) })
}

// azureNCRegisterKeySets registers the cluster's bare metal machine and
// BMC key sets and its metrics configuration.
func azureNCRegisterKeySets() {
	for _, keySet := range []struct {
		typ, kind string
		levels    []string
		bmm       bool
	}{
		{"clusters/bareMetalMachineKeySets", "bare metal machine key set", []string{"Standard", "Superuser", "Other"}, true},
		{"clusters/bmcKeySets", "BMC key set", []string{"ReadOnly", "Administrator"}, false},
	} {
		patchKeys := []string{"expiration", "userList"}
		createOnly := []string{"azureGroupId", "privilegeLevel"}
		if keySet.bmm {
			patchKeys = append(patchKeys, "jumpHostsAllowed")
			createOnly = append(createOnly, "osGroupName", "privilegeLevelName")
		}
		registerAzureARMResourceType(azureARMResourceType{
			Namespace: azureNCNamespace, Type: keySet.typ,
			ValidateName: azureNCName(keySet.kind, azureNCNameRE),
			Validate: func(payload map[string]any) string {
				return cmp.Or(azureNCExtendedLocation(payload), azureDataBoxObjects(payload, "", "properties"),
					azureNCKeySetShape(azureARMObject(payload["properties"]), keySet.levels, keySet.bmm, true))
			},
			PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
				return azureLabPatchOnly(payload, func(props map[string]any) string {
					return azureNCKeySetShape(props, keySet.levels, keySet.bmm, false)
				}, patchKeys...)
			},
			Reconcile: func(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
				return nil, azureNCCreateOnly(existing, rec, createOnly...)
			},
			ReadOnlyProperties: azureNCKeySetStatus,
		})
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureNCNamespace, Type: "clusters/metricsConfigurations",
		ValidateName: azureNCDefaultName("metrics configuration"),
		Validate: func(payload map[string]any) string {
			props := azureARMObject(payload["properties"])
			if _, present := props["collectionInterval"]; !present {
				return cmp.Or(azureNCExtendedLocation(payload), "properties.collectionInterval is required.")
			}
			return cmp.Or(azureNCExtendedLocation(payload), azureNCMetricsShape(props))
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			return azureLabPatchOnly(payload, azureNCMetricsShape, "collectionInterval", "enabledMetrics")
		},
		Reconcile: func(_ *Server, existing, rec *azureARMResource) (map[string]any, string) {
			return nil, azureNCCreateOnly(existing, rec)
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			enabled := azureOracleStringsList(rec.Properties["enabledMetrics"])
			disabled := []any{}
			for _, metric := range azureNCOptionalMetrics {
				if !slices.Contains(enabled, metric) {
					disabled = append(disabled, metric)
				}
			}
			return map[string]any{"detailedStatus": "Applied", "detailedStatusMessage": "The metrics configuration is applied.", "disabledMetrics": disabled}
		},
	})
}

func azureNCMetricsShape(props map[string]any) string {
	if msg := cmp.Or(azureOracleNumber(props, "properties.", "collectionInterval", 1, 1440, true),
		azureOracleStrings(props, "properties.", "enabledMetrics", 0, len(azureNCOptionalMetrics))); msg != "" {
		return msg
	}
	for _, metric := range azureOracleStringsList(props["enabledMetrics"]) {
		if !slices.Contains(azureNCOptionalMetrics, metric) {
			return "properties.enabledMetrics names " + metric + ", which isn't an optional metric of the cluster; choose from " + strings.Join(azureNCOptionalMetrics, ", ") + "."
		}
	}
	return ""
}

// azureNCKeySetShape checks key set properties; full requires what a PUT
// must send.
func azureNCKeySetShape(props map[string]any, levels []string, bmm, full bool) string {
	required := []string{}
	if full {
		required = []string{"azureGroupId", "expiration", "privilegeLevel"}
		if _, ok := props["userList"].([]any); !ok {
			return "properties.userList is required."
		}
		if _, ok := props["jumpHostsAllowed"].([]any); bmm && !ok {
			return "properties.jumpHostsAllowed is required."
		}
	}
	_, expiry := azureLabTime(props, "properties.", "expiration")
	if msg := cmp.Or(azureWorkloadsEnums(props, "properties.", required, map[string][]string{"privilegeLevel": levels}), expiry); msg != "" {
		return msg
	}
	if bmm {
		if msg := azureOracleStrings(props, "properties.", "jumpHostsAllowed", 0, 100); msg != "" {
			return msg
		}
		for _, host := range azureOracleStringsList(props["jumpHostsAllowed"]) {
			if !azureNCIP(host) {
				return "properties.jumpHostsAllowed must hold IP addresses; '" + host + "' isn't one."
			}
		}
		if name, present := props["osGroupName"]; present && (len(azureARMString(name)) > 32 || !azureNCOSGroupRE.MatchString(azureARMString(name))) {
			return "properties.osGroupName must be at most 32 letters and digits, starting with a letter."
		}
		if props["privilegeLevel"] == "Other" && strings.TrimSpace(azureARMString(props["privilegeLevelName"])) == "" && full {
			return "properties.privilegeLevelName is required when properties.privilegeLevel is Other."
		}
	}
	seen := map[string]bool{}
	for i, raw := range azureARMSlice(props["userList"]) {
		user := azureARMObject(raw)
		path := "properties.userList[" + strconv.Itoa(i) + "]."
		name := azureARMString(user["azureUserName"])
		if msg := cmp.Or(azureWorkloadsRequired(user, path, "azureUserName"), azureLabLength(user, path, "description", 0, 256),
			azureWorkloadsRequired(azureARMObject(user["sshPublicKey"]), path+"sshPublicKey.", "keyData")); msg != "" {
			return msg
		}
		if seen[name] {
			return path + "azureUserName " + name + " is listed twice."
		}
		seen[name] = true
	}
	return ""
}

// azureNCKeySetStatus reports every user active until the key set expires.
func azureNCKeySetStatus(rec *azureARMResource) map[string]any {
	expiry, _ := time.Parse(time.RFC3339, azureARMString(rec.Properties["expiration"]))
	status, detailed := "Active", "AllActive"
	if !expiry.IsZero() && expiry.Before(time.Now()) {
		status, detailed = "Invalid", "AllInvalid"
	}
	users := []any{}
	for _, raw := range azureARMSlice(rec.Properties["userList"]) {
		users = append(users, map[string]any{"azureUserName": azureARMObject(raw)["azureUserName"], "status": status})
	}
	return map[string]any{"detailedStatus": detailed, "lastValidation": azureHZTime(rec.UpdatedAt), "userListStatus": users}
}
