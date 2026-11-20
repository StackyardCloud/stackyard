package server

import (
	"cmp"
	"net"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Network Cloud (Operator Nexus) workload resources: the networks, volumes,
// virtual machines and Kubernetes clusters tenants run on an on-premises
// Nexus cluster, addressed through its custom location (extendedLocation).
// Nothing is scheduled anywhere: resources settle at once and report the
// detailed status a healthy instance has.

const azureNCWNamespace = "Microsoft.NetworkCloud"

var (
	azureNCWName30     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-_]{0,28}[a-zA-Z0-9]$`)
	azureNCWName64     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-_]{0,62}[a-zA-Z0-9]$`)
	azureNCWVMName     = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9]{0,62}[a-zA-Z0-9]$`)
	azureNCWInterface  = regexp.MustCompile(`^[a-zA-Z0-9@._-]*$`)
	azureNCWARMID      = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/[^/]+/[^/]+/[^/]+(/[^/]+/[^/]+)*$`)
	azureNCWCustomLoc  = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.ExtendedLocation/customLocations/[^/]+$`)
	azureNCWBools      = []string{"True", "False"}
	azureNCWPlugins    = []string{"DPDK", "SRIOV", "OSDevice"}
	azureNCWNetworks   = []string{"l2Networks", "l3Networks", "trunkedNetworks"}
	azureNCWStorageIDs = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.NetworkCloud/storageAppliances/[^/]+$`)
)

func init() {
	network := func(kind string, validate func(map[string]any) string, defaults map[string]any, createOnly []string) {
		registerAzureARMResourceType(azureARMResourceType{
			Namespace: azureNCWNamespace, Type: kind, ETag: true,
			ValidateName: azureNCWNameHook(kind, azureNCWName30),
			Validate: func(payload map[string]any) string {
				return cmp.Or(azureNCWExtendedLocation(payload, true), azureNCWPresent(payload, "", "properties"), validate(azureARMObject(payload["properties"])))
			},
			ReadOnlyProperties: func(*azureARMResource) map[string]any {
				return map[string]any{"detailedStatus": "Available", "detailedStatusMessage": "Network is available."}
			},
			PatchBody:      azureNCWTagsOnly,
			Reconcile:      azureNCWReconcile(defaults, createOnly, nil),
			ValidateDelete: azureNCWInUse,
		})
	}
	network("l2Networks", azureNCWL2, map[string]any{"hybridAksPluginType": "SRIOV"},
		[]string{"hybridAksPluginType", "interfaceName", "l2IsolationDomainId"})
	network("l3Networks", azureNCWL3, map[string]any{"hybridAksPluginType": "SRIOV", "hybridAksIpamEnabled": "True", "ipAllocationType": "DualStack"},
		[]string{"hybridAksIpamEnabled", "hybridAksPluginType", "interfaceName", "ipv4ConnectedPrefix", "ipv6ConnectedPrefix", "l3IsolationDomainId", "vlan"})
	network("trunkedNetworks", azureNCWTrunked, map[string]any{"hybridAksPluginType": "SRIOV"},
		[]string{"hybridAksPluginType", "interfaceName", "isolationDomainIds", "vlans"})

	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureNCWNamespace, Type: "cloudServicesNetworks", ETag: true,
		ValidateName: azureNCWNameHook("cloudServicesNetworks", azureNCWName30),
		Validate: func(payload map[string]any) string {
			return cmp.Or(azureNCWExtendedLocation(payload, true), azureDataBoxObjects(payload, "", "properties"), azureNCWCSN(azureARMObject(payload["properties"])))
		},
		ReadOnlyProperties: azureNCWCSNStatus,
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			props := azureARMObject(payload["properties"])
			return payload, nil, cmp.Or(azureOracleOnly(payload, "", []string{"tags", "properties"}),
				azureOracleOnly(props, "properties.", []string{"additionalEgressEndpoints", "enableDefaultEgressEndpoints", "storageOptions"}),
				azureNCWCSN(props))
		},
		Reconcile:      azureNCWReconcile(map[string]any{"enableDefaultEgressEndpoints": "True"}, nil, nil),
		ValidateDelete: azureNCWInUse,
	})

	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureNCWNamespace, Type: "volumes", ETag: true,
		ValidateName: azureNCWNameHook("volumes", azureNCWName64),
		Validate: func(payload map[string]any) string {
			props := azureARMObject(payload["properties"])
			return cmp.Or(azureNCWExtendedLocation(payload, true), azureNCWPresent(payload, "", "properties"),
				azureNCWPresent(props, "properties.", "sizeMiB"), azureOracleNumber(props, "properties.", "sizeMiB", 1, 1<<53, true),
				azureNCWMatch(props, "properties.", "storageApplianceId", azureNCWStorageIDs, "a Network Cloud storage appliance ID"))
		},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			out := map[string]any{
				"allocatedSizeMiB": rec.Properties["sizeMiB"], "detailedStatus": "Active", "detailedStatusMessage": "Volume is active.",
				"serialNumber": azureWorkloadsHash(rec.ID, 16),
			}
			if id := azureARMString(rec.Properties["storageApplianceId"]); id != "" {
				out["assignedStorageApplianceId"] = id
			}
			return out
		},
		PatchBody:      azureNCWTagsOnly,
		Reconcile:      azureNCWReconcile(nil, []string{"sizeMiB", "storageApplianceId"}, nil),
		ValidateDelete: azureNCWInUse,
	})

	azureNCWRegisterVMs()
	azureNCWRegisterKubernetes()
}

// azureNCWNameHook checks a resource name against the API's name pattern.
func azureNCWNameHook(kind string, re *regexp.Regexp) func(string) string {
	return func(name string) string {
		if !re.MatchString(name) {
			return "The " + kind + " name '" + name + "' must match " + re.String() + "."
		}
		return ""
	}
}

// azureNCWExtendedLocation checks the top-level extendedLocation: a type
// (EdgeZone or CustomLocation) and a name, which for a custom location is
// its resource ID. It doesn't look the custom location up.
func azureNCWExtendedLocation(payload map[string]any, required bool) string {
	raw, present := payload["extendedLocation"]
	if !present {
		if required {
			return "extendedLocation is required."
		}
		return ""
	}
	loc, ok := raw.(map[string]any)
	if !ok {
		return "extendedLocation must be an object."
	}
	if msg := azureWorkloadsEnums(loc, "extendedLocation.", []string{"name", "type"}, map[string][]string{"type": {"EdgeZone", "CustomLocation"}}); msg != "" {
		return msg
	}
	if loc["type"] == "CustomLocation" && !azureNCWCustomLoc.MatchString(azureARMString(loc["name"])) {
		return "extendedLocation.name must be a custom location ID: /subscriptions/{id}/resourceGroups/{group}/providers/Microsoft.ExtendedLocation/customLocations/{name}."
	}
	return ""
}

// azureNCWPresent requires each key to be present and non-null.
func azureNCWPresent(m map[string]any, path string, keys ...string) string {
	for _, key := range keys {
		if m[key] == nil {
			return path + key + " is required."
		}
	}
	return ""
}

// azureNCWMatch checks that m[key], when present, is a string matching re.
func azureNCWMatch(m map[string]any, path, key string, re *regexp.Regexp, what string) string {
	if raw, present := m[key]; present && !re.MatchString(azureARMString(raw)) {
		return path + key + " must be " + what + "."
	}
	return ""
}

// azureNCWObjects returns the objects listed at m[key], requiring at least
// least of them, or a message when the value isn't a list of objects.
func azureNCWObjects(m map[string]any, path, key string, least int) ([]map[string]any, string) {
	raw := m[key]
	if raw == nil && least == 0 {
		return nil, ""
	}
	items, ok := raw.([]any)
	if !ok || len(items) < least {
		return nil, path + key + " must be a list of at least " + strconv.Itoa(least) + " objects."
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, path + key + " must hold objects."
		}
		out = append(out, obj)
	}
	return out, ""
}

// azureNCWInterfaceName checks an optional interfaceName: at most 12
// letters, digits and @._- characters.
func azureNCWInterfaceName(props map[string]any) string {
	return cmp.Or(azureLabLength(props, "properties.", "interfaceName", 0, 12),
		azureNCWMatch(props, "properties.", "interfaceName", azureNCWInterface, "letters, digits and @._- only"))
}

func azureNCWL2(props map[string]any) string {
	return cmp.Or(azureNCWPresent(props, "properties.", "l2IsolationDomainId"),
		azureNCWMatch(props, "properties.", "l2IsolationDomainId", azureNCWARMID, "a resource ID"),
		azureLabEnum(props, "properties.", "hybridAksPluginType", azureNCWPlugins), azureNCWInterfaceName(props))
}

func azureNCWL3(props map[string]any) string {
	if msg := cmp.Or(azureNCWPresent(props, "properties.", "l3IsolationDomainId", "vlan"),
		azureNCWMatch(props, "properties.", "l3IsolationDomainId", azureNCWARMID, "a resource ID"),
		azureOracleNumber(props, "properties.", "vlan", 0, 1<<53, true),
		azureLabEnum(props, "properties.", "hybridAksPluginType", azureNCWPlugins),
		azureLabEnum(props, "properties.", "hybridAksIpamEnabled", azureNCWBools),
		azureLabEnum(props, "properties.", "ipAllocationType", []string{"IPV4", "IPV6", "DualStack"}),
		azureNCWInterfaceName(props)); msg != "" {
		return msg
	}
	kind := cmp.Or(azureARMString(props["ipAllocationType"]), "DualStack")
	for _, family := range []struct{ key, needs string }{{"ipv4ConnectedPrefix", "IPV4"}, {"ipv6ConnectedPrefix", "IPV6"}} {
		prefix := azureARMString(props[family.key])
		if prefix == "" {
			if kind == family.needs || kind == "DualStack" {
				return "properties." + family.key + " is required when ipAllocationType is " + kind + "."
			}
			continue
		}
		ip, _, err := net.ParseCIDR(prefix)
		if err != nil || (ip.To4() != nil) != (family.needs == "IPV4") {
			return "properties." + family.key + " must be an " + family.needs + " CIDR prefix."
		}
	}
	return ""
}

func azureNCWTrunked(props map[string]any) string {
	if msg := cmp.Or(azureOracleStrings(props, "properties.", "isolationDomainIds", 1, 1<<20), azureNCWPresent(props, "properties.", "isolationDomainIds", "vlans"),
		azureLabEnum(props, "properties.", "hybridAksPluginType", azureNCWPlugins), azureNCWInterfaceName(props)); msg != "" {
		return msg
	}
	for _, id := range azureARMSlice(props["isolationDomainIds"]) {
		if !azureNCWARMID.MatchString(azureARMString(id)) {
			return "properties.isolationDomainIds must hold resource IDs."
		}
	}
	vlans, ok := props["vlans"].([]any)
	if !ok || len(vlans) == 0 {
		return "properties.vlans must be a list of at least 1 VLAN."
	}
	for _, vlan := range vlans {
		if n, ok := azureFileShareInt(vlan); !ok || n < 0 {
			return "properties.vlans must hold whole numbers."
		}
	}
	return ""
}

// azureNCWCSN checks a cloud services network's egress endpoints and
// storage options, on PUT and PATCH alike.
func azureNCWCSN(props map[string]any) string {
	if msg := cmp.Or(azureLabEnum(props, "properties.", "enableDefaultEgressEndpoints", azureNCWBools), azureDataBoxObjects(props, "properties.", "storageOptions")); msg != "" {
		return msg
	}
	egress, msg := azureNCWObjects(props, "properties.", "additionalEgressEndpoints", 0)
	if msg != "" {
		return msg
	}
	for _, e := range egress {
		path := "properties.additionalEgressEndpoints[]."
		endpoints, msg := azureNCWObjects(e, path, "endpoints", 1)
		if msg = cmp.Or(azureWorkloadsRequired(e, path, "category"), msg); msg != "" {
			return msg
		}
		for _, ep := range endpoints {
			if msg := cmp.Or(azureWorkloadsRequired(ep, path+"endpoints[].", "domainName"), azureOracleNumber(ep, path+"endpoints[].", "port", 1, 65535, true)); msg != "" {
				return msg
			}
		}
	}
	storage := azureARMObject(props["storageOptions"])
	return cmp.Or(azureLabEnum(storage, "properties.storageOptions.", "mode", []string{"None", "Standard"}),
		azureOracleNumber(storage, "properties.storageOptions.", "sizeMiB", 1, 1<<53, true),
		azureNCWMatch(storage, "properties.storageOptions.", "storageApplianceId", azureNCWStorageIDs, "a Network Cloud storage appliance ID"))
}

// azureNCWCSNStatus reports the egress endpoints in force and the shared
// storage volume's status.
func azureNCWCSNStatus(rec *azureARMResource) map[string]any {
	storage := azureARMObject(rec.Properties["storageOptions"])
	status := map[string]any{"mode": "None", "sizeMiB": 0, "status": "None"}
	if mode := cmp.Or(azureARMString(storage["mode"]), "Standard"); len(storage) != 0 && mode == "Standard" {
		status = map[string]any{"mode": mode, "sizeMiB": storage["sizeMiB"], "status": "Available"}
	}
	return map[string]any{
		// ponytail: only the additional endpoints are listed; the platform's
		// default endpoint list isn't documented.
		"enabledEgressEndpoints": azureNCWList(azureARMSlice(rec.Properties["additionalEgressEndpoints"])),
		"detailedStatus":         "Available", "detailedStatusMessage": "Cloud services network is available.",
		"interfaceName": "csn-" + azureWorkloadsHash(rec.ID, 6),
		"storageStatus": status,
	}
}

// azureNCWList returns items, or an empty list for nil, since responses
// show empty lists.
func azureNCWList(items []any) []any {
	if items == nil {
		return []any{}
	}
	return items
}

// azureNCWTagsOnly is the PatchBody of types whose update model has tags only.
func azureNCWTagsOnly(payload map[string]any) (map[string]any, map[string]any, string) {
	return payload, nil, azureOracleOnly(payload, "", []string{"tags"})
}

// azureNCWReconcile fills defaults on create, keeps createOnly properties
// (and extendedLocation) from changing on a later PUT, and then runs more
// when set. A later PUT that omits a create-only property keeps it.
func azureNCWReconcile(defaults map[string]any, createOnly []string, more func(s *Server, existing, rec *azureARMResource) (map[string]any, string)) func(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	return func(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
		out := map[string]any{}
		// The usage lists belong to azureNCWRelinkLocked: a PUT keeps them.
		for _, key := range azureNCWLinkKeys(rec.Type) {
			out[key] = []any{}
			if existing != nil && existing.Properties[key] != nil {
				out[key] = existing.Properties[key]
			}
		}
		if existing == nil {
			for key, value := range defaults {
				if _, present := rec.Properties[key]; !present {
					out[key] = value
				}
			}
		} else {
			if len(rec.ExtendedLocation) != 0 && !reflect.DeepEqual(rec.ExtendedLocation, existing.ExtendedLocation) {
				return nil, "extendedLocation can't be changed after creation."
			}
			rec.ExtendedLocation = existing.ExtendedLocation
			if msg := azureNCWKeep(existing.Properties, rec.Properties, "properties.", createOnly, out); msg != "" {
				return nil, msg
			}
		}
		if more == nil {
			return out, ""
		}
		extra, msg := more(s, existing, rec)
		return azureARMMerge(out, extra), msg
	}
}

// azureNCWKeep rejects a changed create-only key and copies omitted ones
// from the stored properties into out.
func azureNCWKeep(stored, sent map[string]any, path string, keys []string, out map[string]any) string {
	for _, key := range keys {
		value, present := sent[key]
		if !present {
			if old, had := stored[key]; had {
				out[key] = old
			}
			continue
		}
		if !reflect.DeepEqual(value, stored[key]) {
			return path + key + " can't be changed after creation."
		}
	}
	return ""
}

// azureNCWInUse refuses to delete a network or volume a virtual machine or
// Kubernetes cluster still uses.
func azureNCWInUse(s *Server, existing *azureARMResource) string {
	users := azureARMSlice(existing.Properties["associatedResourceIds"])
	if strings.HasSuffix(strings.ToLower(existing.Type), "/volumes") {
		users = azureARMSlice(existing.Properties["attachedTo"])
	}
	// Users no longer stored don't count: a resource group delete drops
	// them without updating the lists.
	for _, user := range users {
		if id := azureARMString(user); s.azureARMResources[azureARMKey(id)] != nil {
			return existing.Name + " is in use by " + id + "; delete or detach that resource first."
		}
	}
	return ""
}

// azureNCWExistsLocked reports whether id is a stored Network Cloud
// resource of the given type. Caller holds providerStorageMu.
func (s *Server) azureNCWExistsLocked(id, kind string) bool {
	rec := s.azureARMResources[azureARMKey(id)]
	return rec != nil && strings.EqualFold(rec.Type, azureNCWNamespace+"/"+kind)
}

// azureNCWRelinkLocked recomputes which virtual machines and Kubernetes
// clusters use each network and volume, as if pending (when not nil) were
// stored and removed were gone. It rewrites only records whose lists
// change, so their etags stay put otherwise. Caller holds
// providerStorageMu.
func (s *Server) azureNCWRelinkLocked(pending *azureARMResource, removed string) {
	vms, clusters := map[string][]any{}, map[string][]any{}
	users := s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool {
		return azureNCWUserKind(r.Type) != "" && !strings.EqualFold(r.ID, removed) && (pending == nil || !strings.EqualFold(r.ID, pending.ID))
	})
	if pending != nil {
		users = append(users, pending)
	}
	for _, user := range users {
		owner, list := user.ID, vms
		switch azureNCWUserKind(user.Type) {
		case "kubernetesClusters/agentPools":
			owner, list = azureHZClusterOf(user.ID), clusters
		case "kubernetesClusters":
			list = clusters
		}
		for _, id := range azureNCWRefs(user.Properties) {
			key := azureARMKey(id)
			if !slices.ContainsFunc(list[key], func(v any) bool { return strings.EqualFold(azureARMString(v), owner) }) {
				list[key] = append(list[key], owner)
			}
		}
	}
	// Sorted lists leave a record (and its etag) alone when only the order
	// users were visited in differs.
	byID := func(a, b any) int {
		return strings.Compare(azureARMKey(azureARMString(a)), azureARMKey(azureARMString(b)))
	}
	targets := s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool { return azureNCWLinkKeys(r.Type) != nil })
	for _, target := range targets {
		key := azureARMKey(target.ID)
		vm, k8s := azureNCWList(vms[key]), azureNCWList(clusters[key])
		slices.SortFunc(vm, byID)
		slices.SortFunc(k8s, byID)
		props := map[string]any{"associatedResourceIds": append(slices.Clone(vm), k8s...), "virtualMachinesAssociatedIds": vm, "hybridAksClustersAssociatedIds": k8s}
		if len(azureNCWLinkKeys(target.Type)) == 1 {
			props = map[string]any{"attachedTo": vm}
		}
		if !reflect.DeepEqual(azureARMMerge(target.Properties, props), target.Properties) {
			s.azureOracleUpdateLocked(target.ID, props)
		}
	}
}

// azureNCWLinkKeys returns the usage lists azureNCWRelinkLocked keeps on a
// network or volume, or nil for other types.
func azureNCWLinkKeys(fullType string) []string {
	namespace, kind, _ := strings.Cut(fullType, "/")
	switch {
	case !strings.EqualFold(namespace, azureNCWNamespace):
		return nil
	case kind == "volumes":
		return []string{"attachedTo"}
	case kind == "cloudServicesNetworks" || slices.Contains(azureNCWNetworks, kind):
		return []string{"associatedResourceIds", "virtualMachinesAssociatedIds", "hybridAksClustersAssociatedIds"}
	}
	return nil
}

// azureNCWUserKind names the type of a record that attaches networks or
// volumes, or returns "".
func azureNCWUserKind(fullType string) string {
	namespace, kind, _ := strings.Cut(fullType, "/")
	if strings.EqualFold(namespace, azureNCWNamespace) && slices.Contains([]string{"virtualMachines", "kubernetesClusters", "kubernetesClusters/agentPools"}, kind) {
		return kind
	}
	return ""
}

// azureNCWRefs lists the network and volume IDs a virtual machine,
// Kubernetes cluster or agent pool attaches.
func azureNCWRefs(props map[string]any) []string {
	var ids []string
	add := func(v any) {
		if id := azureARMString(v); id != "" {
			ids = append(ids, id)
		}
	}
	attached := func(m map[string]any) {
		for _, kind := range azureNCWNetworks {
			for _, item := range azureARMSlice(m[kind]) {
				add(azureARMObject(item)["networkId"])
			}
		}
	}
	add(azureARMObject(props["cloudServicesNetworkAttachment"])["attachedNetworkId"])
	for _, item := range azureARMSlice(props["networkAttachments"]) {
		add(azureARMObject(item)["attachedNetworkId"])
	}
	for _, item := range azureARMSlice(azureARMObject(props["storageProfile"])["volumeAttachments"]) {
		add(item)
	}
	network := azureARMObject(props["networkConfiguration"])
	add(network["cloudServicesNetworkId"])
	add(network["cniNetworkId"])
	attached(azureARMObject(network["attachedNetworkConfiguration"]))
	attached(azureARMObject(props["attachedNetworkConfiguration"]))
	for _, pool := range azureARMSlice(props["initialAgentPoolConfigurations"]) {
		attached(azureARMObject(azureARMObject(pool)["attachedNetworkConfiguration"]))
	}
	return ids
}
