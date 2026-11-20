package server

import (
	"cmp"
	"net"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var (
	azureNCWFeatureName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]{0,61}[a-zA-Z0-9]$`)
	azureNCWLBName      = regexp.MustCompile(`^[a-z0-9]([a-z0-9.-]{0,61}[a-z0-9]){0,1}$`)
	azureNCWSurge       = regexp.MustCompile(`^[0-9]+%?$`)
	azureNCWBGPPassword = regexp.MustCompile(`^[a-zA-Z0-9]{0,80}$`)
	azureNCWK8sPlugins  = []string{"DPDK", "SRIOV", "OSDevice", "MACVLAN", "IPVLAN"}
	// azureNCWPoolCreateOnly are the agent pool properties fixed at
	// creation; count, upgrade settings and SSH keys can change.
	azureNCWPoolCreateOnly = []string{"agentOptions", "attachedNetworkConfiguration", "availabilityZones", "labels", "mode", "taints", "vmSkuName"}
)

const azureNCWPoolType = "kubernetesClusters/agentPools"

func azureNCWRegisterKubernetes() {
	relinkOnDelete := func(s *Server, removed *azureARMResource) { s.azureNCWRelinkLocked(nil, removed.ID) }
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureNCWNamespace, Type: "kubernetesClusters", ETag: true,
		ValidateName:        azureNCWNameHook("kubernetesClusters", azureNCWName30),
		Validate:            azureNCWK8sValidate,
		WriteOnlyProperties: []string{"networkConfiguration.bgpServiceLoadBalancerConfiguration.bgpPeers[].password"},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{
				"availableUpgrades": []any{}, "controlPlaneKubernetesVersion": rec.Properties["kubernetesVersion"],
				"detailedStatus": "Available", "detailedStatusMessage": "Kubernetes cluster is available.",
			}
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			props := azureARMObject(payload["properties"])
			cp := azureARMObject(props["controlPlaneNodeConfiguration"])
			return payload, nil, cmp.Or(azureOracleOnly(payload, "", []string{"tags", "properties"}),
				azureOracleOnly(props, "properties.", []string{"administratorConfiguration", "controlPlaneNodeConfiguration", "kubernetesVersion"}),
				azureOracleOnly(cp, "properties.controlPlaneNodeConfiguration.", []string{"administratorConfiguration", "count"}),
				azureOracleNumber(cp, "properties.controlPlaneNodeConfiguration.", "count", 1, 1<<53, true),
				azureLabEnum(props, "properties.", "kubernetesVersion", azureNCKubernetesVersions),
				azureNCWKeysOnly(props, "properties."), azureNCWKeysOnly(cp, "properties.controlPlaneNodeConfiguration."))
		},
		Reconcile: azureNCWReconcile(nil, []string{"aadConfiguration", "managedResourceGroupConfiguration", "networkConfiguration"}, azureNCWK8sReconcile),
		ChildProperties: func(s *Server, rec *azureARMResource) map[string]any {
			out := s.azureNCWK8sDerivedLocked(rec)
			// ponytail: ChildProperties also runs before DefaultChildren
			// creates the pools, which still need the list; a cluster keeps
			// at least one pool afterwards, so the list empties for good.
			if len(s.azureNCWPoolsLocked(rec.ID)) != 0 {
				out["initialAgentPoolConfigurations"] = []any{}
			}
			return out
		},
		DefaultChildren: azureNCWInitialPools,
		Deleted:         relinkOnDelete,
		ServerActions:   map[string]func(*Server, *azureARMResource, map[string]any) (int, any){"restartnode": azureNCWRestartNode},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureNCWNamespace, Type: azureNCWPoolType, ETag: true,
		ValidateName: azureNCWNameHook("agentPools", azureNCWName30),
		Validate: func(payload map[string]any) string {
			return cmp.Or(azureNCWExtendedLocation(payload, false), azureNCWPresent(payload, "", "properties"), azureNCWPool(azureARMObject(payload["properties"]), "properties."))
		},
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"detailedStatus": "Available", "detailedStatusMessage": "Agent pool is available."}
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			props := azureARMObject(payload["properties"])
			return payload, nil, cmp.Or(azureOracleOnly(payload, "", []string{"tags", "properties"}),
				azureOracleOnly(props, "properties.", []string{"administratorConfiguration", "count", "upgradeSettings"}),
				azureOracleNumber(props, "properties.", "count", 0, 1<<53, true), azureNCWKeysOnly(props, "properties."), azureNCWUpgrade(props, "properties.", false))
		},
		Reconcile: azureNCWReconcile(nil, azureNCWPoolCreateOnly, azureNCWPoolReconcile),
		ValidatePut: func(s *Server, existing, rec *azureARMResource) (string, string) {
			if existing == nil {
				return "", ""
			}
			return "", s.azureNCWSystemLeftLocked(existing, rec.Properties["count"])
		},
		ValidatePatch: func(s *Server, existing *azureARMResource, payload map[string]any) string {
			if count, present := azureARMObject(payload["properties"])["count"]; present {
				return s.azureNCWSystemLeftLocked(existing, count)
			}
			return ""
		},
		ValidateDelete: func(s *Server, existing *azureARMResource) string { return s.azureNCWSystemLeftLocked(existing, 0.0) },
		Deleted:        relinkOnDelete,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: azureNCWNamespace, Type: "kubernetesClusters/features", ETag: true,
		ValidateName: azureNCWNameHook("features", azureNCWFeatureName),
		Validate: func(payload map[string]any) string {
			return cmp.Or(azureDataBoxObjects(payload, "", "properties"), azureNCWOptions(azureARMObject(payload["properties"])))
		},
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"availabilityLifecycle": "GenerallyAvailable", "detailedStatus": "Installed", "detailedStatusMessage": "Feature is installed.", "required": "False", "version": "1.0.0"}
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			props := azureARMObject(payload["properties"])
			return payload, nil, cmp.Or(azureOracleOnly(payload, "", []string{"tags", "properties"}), azureOracleOnly(props, "properties.", []string{"options"}), azureNCWOptions(props))
		},
	})
}

func azureNCWOptions(props map[string]any) string {
	options, msg := azureNCWObjects(props, "properties.", "options", 0)
	for _, o := range options {
		msg = cmp.Or(msg, azureWorkloadsRequired(o, "properties.options[].", "key", "value"))
	}
	return msg
}

// azureNCWKeysOnly checks an update's administratorConfiguration, which
// can only replace SSH keys.
func azureNCWKeysOnly(m map[string]any, path string) string {
	admin := azureARMObject(m["administratorConfiguration"])
	return cmp.Or(azureDataBoxObjects(m, path, "administratorConfiguration"),
		azureOracleOnly(admin, path+"administratorConfiguration.", []string{"sshPublicKeys"}), azureNCWSSHKeys(admin, path+"administratorConfiguration."))
}

func azureNCWAdmin(m map[string]any, path string) string {
	admin := azureARMObject(m["administratorConfiguration"])
	return cmp.Or(azureDataBoxObjects(m, path, "administratorConfiguration"), azureLabLength(admin, path+"administratorConfiguration.", "adminUsername", 1, 32),
		azureNCWSSHKeys(admin, path+"administratorConfiguration."))
}

func azureNCWK8sValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	cp := azureARMObject(props["controlPlaneNodeConfiguration"])
	if msg := cmp.Or(azureNCWExtendedLocation(payload, true), azureNCWPresent(payload, "", "properties"),
		azureNCWPresent(props, "properties.", "controlPlaneNodeConfiguration", "initialAgentPoolConfigurations", "kubernetesVersion", "networkConfiguration"),
		// Versions come from kubernetesVersions/default.
		azureLabEnum(props, "properties.", "kubernetesVersion", azureNCKubernetesVersions),
		azureDataBoxObjects(props, "properties.", "aadConfiguration", "controlPlaneNodeConfiguration", "managedResourceGroupConfiguration", "networkConfiguration"),
		azureNCWAdmin(props, "properties."),
		azureNCWPresent(cp, "properties.controlPlaneNodeConfiguration.", "count", "vmSkuName"),
		azureOracleNumber(cp, "properties.controlPlaneNodeConfiguration.", "count", 1, 1<<53, true),
		azureWorkloadsRequired(cp, "properties.controlPlaneNodeConfiguration.", "vmSkuName"),
		azureOracleStrings(cp, "properties.controlPlaneNodeConfiguration.", "availabilityZones", 0, 1<<20),
		azureNCWAdmin(cp, "properties.controlPlaneNodeConfiguration."),
		azureLabLength(azureARMObject(props["managedResourceGroupConfiguration"]), "properties.managedResourceGroupConfiguration.", "name", 1, 75),
		azureNCWNetworkConfig(azureARMObject(props["networkConfiguration"]))); msg != "" {
		return msg
	}
	if aad, present := props["aadConfiguration"]; present {
		if msg := cmp.Or(azureNCWPresent(azureARMObject(aad), "properties.aadConfiguration.", "adminGroupObjectIds"),
			azureOracleStrings(azureARMObject(aad), "properties.aadConfiguration.", "adminGroupObjectIds", 1, 1<<20)); msg != "" {
			return msg
		}
	}
	// The list is empty after creation, so a GET body may send it back
	// empty; azureNCWK8sReconcile requires a System pool on create.
	pools, msg := azureNCWObjects(props, "properties.", "initialAgentPoolConfigurations", 0)
	if msg != "" {
		return msg
	}
	names := map[string]bool{}
	for _, pool := range pools {
		path := "properties.initialAgentPoolConfigurations[]."
		name := azureARMString(pool["name"])
		if msg := cmp.Or(azureNCWPresent(pool, path, "name"), azureNCWMatch(pool, path, "name", azureNCWName30, "a valid agent pool name"), azureNCWPool(pool, path)); msg != "" {
			return msg
		}
		if names[strings.ToLower(name)] {
			return path + "name '" + name + "' is used twice."
		}
		names[strings.ToLower(name)] = true
	}
	return ""
}

// azureNCWHasSystemPool reports whether pools hold a System pool with at
// least one node, as every Kubernetes cluster must.
func azureNCWHasSystemPool(pools []any) bool {
	return slices.ContainsFunc(pools, func(v any) bool {
		pool := azureARMObject(v)
		n, _ := azureFileShareInt(pool["count"])
		return pool["mode"] == "System" && n >= 1
	})
}

// azureNCWPool checks an agent pool's configuration, inline or standalone.
func azureNCWPool(m map[string]any, path string) string {
	options := azureARMObject(m["agentOptions"])
	if msg := cmp.Or(azureNCWPresent(m, path, "count", "mode", "vmSkuName"), azureOracleNumber(m, path, "count", 0, 1<<53, true),
		azureWorkloadsEnums(m, path, []string{"vmSkuName"}, map[string][]string{"mode": {"System", "User", "NotApplicable"}}),
		azureOracleStrings(m, path, "availabilityZones", 0, 1<<20), azureNCWAdmin(m, path),
		azureDataBoxObjects(m, path, "agentOptions", "attachedNetworkConfiguration", "upgradeSettings"),
		azureNCWAttached(azureARMObject(m["attachedNetworkConfiguration"]), path+"attachedNetworkConfiguration."),
		azureNCWLabels(m, path, "labels"), azureNCWLabels(m, path, "taints"), azureNCWUpgrade(m, path, true)); msg != "" {
		return msg
	}
	if _, present := m["agentOptions"]; !present {
		return ""
	}
	return cmp.Or(azureNCWPresent(options, path+"agentOptions.", "hugepagesCount"), azureOracleNumber(options, path+"agentOptions.", "hugepagesCount", 0, 1<<53, true),
		azureLabEnum(options, path+"agentOptions.", "hugepagesSize", []string{"2M", "1G"}))
}

func azureNCWLabels(m map[string]any, path, key string) string {
	labels, msg := azureNCWObjects(m, path, key, 0)
	for _, l := range labels {
		msg = cmp.Or(msg, azureNCWPresent(l, path+key+"[].", "key", "value"))
	}
	return msg
}

// azureNCWUpgrade checks upgrade settings: a drain timeout of 60 to 86400
// seconds and surge and unavailability counts or percentages. When whole
// is set, m holds the complete settings and at least one of the two must
// be above 0 (maxSurge defaults to 1, maxUnavailable to 0); a PATCH
// fragment gets that check after merging, in azureNCWPoolReconcile.
func azureNCWUpgrade(m map[string]any, path string, whole bool) string {
	u := azureARMObject(m["upgradeSettings"])
	path += "upgradeSettings."
	if msg := azureOracleNumber(u, path, "drainTimeout", 60, 86400, true); msg != "" {
		return msg
	}
	for _, key := range []string{"maxSurge", "maxUnavailable"} {
		if raw, present := u[key]; present {
			if text, ok := raw.(string); !ok || !azureNCWSurge.MatchString(text) {
				return path + key + " must be a whole number or a percentage such as '50%'."
			}
		}
	}
	zero := func(text string) bool { return strings.TrimRight(strings.TrimLeft(text, "0"), "%") == "" }
	if whole && zero(cmp.Or(azureARMString(u["maxSurge"]), "1")) && zero(cmp.Or(azureARMString(u["maxUnavailable"]), "0")) {
		return "One of " + path + "maxSurge and maxUnavailable must be greater than 0."
	}
	return ""
}

// azureNCWAttached checks attached L2, L3 and trunked network references.
func azureNCWAttached(m map[string]any, path string) string {
	for _, kind := range azureNCWNetworks {
		items, msg := azureNCWObjects(m, path, kind, 0)
		for _, item := range items {
			msg = cmp.Or(msg, azureNCWPresent(item, path+kind+"[].", "networkId"), azureNCWMatch(item, path+kind+"[].", "networkId", azureNCWARMID, "a resource ID"),
				azureLabEnum(item, path+kind+"[].", "pluginType", azureNCWK8sPlugins), azureLabEnum(item, path+kind+"[].", "ipamEnabled", azureNCWBools))
		}
		if msg != "" {
			return msg
		}
	}
	return ""
}

// azureNCWAttachedLocked checks attached networks exist with the type
// their list names. Caller holds providerStorageMu.
func (s *Server) azureNCWAttachedLocked(m map[string]any, path string) string {
	for _, kind := range azureNCWNetworks {
		for _, item := range azureARMSlice(m[kind]) {
			if id := azureARMString(azureARMObject(item)["networkId"]); !s.azureNCWExistsLocked(id, kind) {
				return path + kind + ": " + id + " isn't an existing " + kind[:len(kind)-1] + "."
			}
		}
	}
	return ""
}

func azureNCWNetworkConfig(n map[string]any) string {
	const path = "properties.networkConfiguration."
	if msg := cmp.Or(azureNCWPresent(n, path, "cloudServicesNetworkId", "cniNetworkId"),
		azureNCWMatch(n, path, "cloudServicesNetworkId", azureNCWARMID, "a resource ID"), azureNCWMatch(n, path, "cniNetworkId", azureNCWARMID, "a resource ID"),
		azureDataBoxObjects(n, path, "attachedNetworkConfiguration", "bgpServiceLoadBalancerConfiguration", "l2ServiceLoadBalancerConfiguration"),
		azureNCWAttached(azureARMObject(n["attachedNetworkConfiguration"]), path+"attachedNetworkConfiguration."),
		azureNCWCIDRs(n, path, "podCidrs"), azureNCWCIDRs(n, path, "serviceCidrs")); msg != "" {
		return msg
	}
	if raw, present := n["dnsServiceIp"]; present {
		ip := net.ParseIP(azureARMString(raw))
		inside := ip != nil && len(azureARMSlice(n["serviceCidrs"])) == 0
		for _, cidr := range azureARMSlice(n["serviceCidrs"]) {
			if _, subnet, _ := net.ParseCIDR(azureARMString(cidr)); ip != nil && subnet.Contains(ip) {
				inside = true
			}
		}
		if !inside {
			return path + "dnsServiceIp must be an IP address within the service CIDRs."
		}
	}
	if _, msg := azureNCWPools(azureARMObject(n["l2ServiceLoadBalancerConfiguration"]), path+"l2ServiceLoadBalancerConfiguration.", false); msg != "" {
		return msg
	}
	return azureNCWBGP(azureARMObject(n["bgpServiceLoadBalancerConfiguration"]), path+"bgpServiceLoadBalancerConfiguration.")
}

func azureNCWCIDRs(m map[string]any, path, key string) string {
	if msg := azureOracleStrings(m, path, key, 0, 1<<20); msg != "" {
		return msg
	}
	for _, cidr := range azureARMSlice(m[key]) {
		if _, _, err := net.ParseCIDR(azureARMString(cidr)); err != nil {
			return path + key + " must hold CIDR prefixes."
		}
	}
	return ""
}

// azureNCWPools checks IP address pools and returns their names. BGP pools
// take CIDR prefixes only, and not single-address ones.
func azureNCWPools(m map[string]any, path string, bgp bool) (map[string]bool, string) {
	pools, msg := azureNCWObjects(m, path, "ipAddressPools", 0)
	names := map[string]bool{}
	for _, p := range pools {
		at := path + "ipAddressPools[]."
		msg = cmp.Or(msg, azureNCWPresent(p, at, "addresses", "name"), azureNCWMatch(p, at, "name", azureNCWLBName, "lowercase letters, digits, '.' and '-'"),
			azureOracleStrings(p, at, "addresses", 1, 1<<20), azureLabEnum(p, at, "autoAssign", azureNCWBools), azureLabEnum(p, at, "onlyUseHostIps", azureNCWBools))
		for _, raw := range azureARMSlice(p["addresses"]) {
			msg = cmp.Or(msg, azureNCWAddressRange(azureARMString(raw), at, bgp))
		}
		names[azureARMString(p["name"])] = true
	}
	return names, msg
}

func azureNCWAddressRange(text, path string, bgp bool) string {
	if _, subnet, err := net.ParseCIDR(text); err == nil {
		if ones, bits := subnet.Mask.Size(); bgp && ones == bits {
			return path + "addresses: a BGP pool can't use the single-address prefix " + text + "."
		}
		return ""
	}
	start, end, ok := strings.Cut(text, "-")
	if !bgp && ok && net.ParseIP(strings.TrimSpace(start)) != nil && net.ParseIP(strings.TrimSpace(end)) != nil {
		return ""
	}
	if bgp {
		return path + "addresses must be CIDR prefixes for a BGP service load balancer."
	}
	return path + "addresses must be CIDR prefixes or start-end ranges."
}

func azureNCWBGP(m map[string]any, path string) string {
	pools, msg := azureNCWPools(m, path, true)
	peers, peerMsg := azureNCWObjects(m, path, "bgpPeers", 0)
	if msg = cmp.Or(msg, peerMsg, azureLabEnum(m, path, "fabricPeeringEnabled", azureNCWBools)); msg != "" {
		return msg
	}
	peerNames := map[string]bool{}
	for _, p := range peers {
		at := path + "bgpPeers[]."
		if msg := cmp.Or(azureNCWPresent(p, at, "name", "peerAddress", "peerAsn"), azureNCWMatch(p, at, "name", azureNCWLBName, "lowercase letters, digits, '.' and '-'"),
			azureOracleNumber(p, at, "peerAsn", 0, 4294967295, true), azureOracleNumber(p, at, "myAsn", 0, 4294967295, true),
			azureOracleNumber(p, at, "peerPort", 1, 65535, true), azureLabEnum(p, at, "bfdEnabled", azureNCWBools), azureLabEnum(p, at, "bgpMultiHop", azureNCWBools),
			azureNCWMatch(p, at, "password", azureNCWBGPPassword, "at most 80 letters and digits")); msg != "" {
			return msg
		}
		if net.ParseIP(azureARMString(p["peerAddress"])) == nil {
			return at + "peerAddress must be an IP address."
		}
		peerNames[azureARMString(p["name"])] = true
	}
	ads, msg := azureNCWObjects(m, path, "bgpAdvertisements", 0)
	for _, ad := range ads {
		at := path + "bgpAdvertisements[]."
		msg = cmp.Or(msg, azureNCWPresent(ad, at, "ipAddressPools"), azureOracleStrings(ad, at, "ipAddressPools", 1, 1<<20),
			azureOracleStrings(ad, at, "peers", 0, 1<<20), azureLabEnum(ad, at, "advertiseToFabric", azureNCWBools))
		for _, name := range azureARMSlice(ad["ipAddressPools"]) {
			if !pools[azureARMString(name)] && msg == "" {
				msg = at + "ipAddressPools names pool '" + azureARMString(name) + "', which ipAddressPools doesn't define."
			}
		}
		for _, name := range azureARMSlice(ad["peers"]) {
			if !peerNames[azureARMString(name)] && msg == "" {
				msg = at + "peers names peer '" + azureARMString(name) + "', which bgpPeers doesn't define."
			}
		}
	}
	return msg
}

// azureNCWK8sReconcile checks the networks a new cluster attaches exist,
// keeps the create-only parts of the control plane and administrator
// configuration, carries a version change to the agent pools and records
// the cluster's network use. A later PUT's initialAgentPoolConfigurations
// are ignored: pools are managed through agentPools after creation.
func azureNCWK8sReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	out := map[string]any{}
	if existing == nil {
		if !azureNCWHasSystemPool(azureARMSlice(rec.Properties["initialAgentPoolConfigurations"])) {
			return nil, "Every Kubernetes cluster must contain at least one System agent pool with at least one node."
		}
		network := azureARMObject(rec.Properties["networkConfiguration"])
		const path = "properties.networkConfiguration."
		switch {
		case !s.azureNCWExistsLocked(azureARMString(network["cloudServicesNetworkId"]), "cloudServicesNetworks"):
			return nil, path + "cloudServicesNetworkId must name an existing cloud services network."
		case !s.azureNCWExistsLocked(azureARMString(network["cniNetworkId"]), "l3Networks"):
			return nil, path + "cniNetworkId must name an existing L3 network."
		}
		if msg := s.azureNCWAttachedLocked(azureARMObject(network["attachedNetworkConfiguration"]), path+"attachedNetworkConfiguration."); msg != "" {
			return nil, msg
		}
		for _, pool := range azureARMSlice(rec.Properties["initialAgentPoolConfigurations"]) {
			if msg := s.azureNCWAttachedLocked(azureARMObject(azureARMObject(pool)["attachedNetworkConfiguration"]), "properties.initialAgentPoolConfigurations[].attachedNetworkConfiguration."); msg != "" {
				return nil, msg
			}
		}
	} else {
		out["initialAgentPoolConfigurations"] = []any{}
		for _, part := range []struct {
			key  string
			keep []string
		}{{"controlPlaneNodeConfiguration", []string{"availabilityZones", "vmSkuName"}}, {"administratorConfiguration", []string{"adminUsername"}}} {
			kept := map[string]any{}
			if msg := azureNCWKeep(azureARMObject(existing.Properties[part.key]), azureARMObject(rec.Properties[part.key]), "properties."+part.key+".", part.keep, kept); msg != "" {
				return nil, msg
			}
			if len(kept) != 0 {
				out[part.key] = kept
			}
		}
		if version := rec.Properties["kubernetesVersion"]; version != existing.Properties["kubernetesVersion"] {
			for _, pool := range s.azureNCWPoolsLocked(rec.ID) {
				s.azureOracleUpdateLocked(pool.ID, map[string]any{"kubernetesVersion": version})
			}
		}
	}
	pending := *rec
	pending.Properties = azureARMMerge(rec.Properties, out)
	out = azureARMMerge(out, s.azureNCWK8sDerivedLocked(&pending))
	s.azureNCWRelinkLocked(&pending, "")
	return out, ""
}

func (s *Server) azureNCWPoolsLocked(clusterID string) []*azureARMResource {
	typ, _ := lookupAzureARMResourceType(azureNCWNamespace, azureNCWPoolType)
	return s.azureARMInlineChildrenLocked(clusterID, typ)
}

// azureNCWK8sDerivedLocked lists a cluster's nodes (its control plane and
// each agent pool's), the networks it and its pools attach, and its
// features. Caller holds providerStorageMu.
func (s *Server) azureNCWK8sDerivedLocked(rec *azureARMResource) map[string]any {
	version := rec.Properties["kubernetesVersion"]
	cp := azureARMObject(rec.Properties["controlPlaneNodeConfiguration"])
	node := func(name, role string, pool map[string]any, poolID string) map[string]any {
		out := map[string]any{
			"name": name, "role": role, "kubernetesVersion": version, "vmSkuName": pool["vmSkuName"], "mode": cmp.Or(azureARMString(pool["mode"]), "NotApplicable"),
			"detailedStatus": "Running", "detailedStatusMessage": "Node is running.", "powerState": "On",
			"labels": azureNCWList(azureARMSlice(pool["labels"])), "taints": azureNCWList(azureARMSlice(pool["taints"])),
		}
		if poolID != "" {
			out["agentPoolId"] = poolID
		}
		return out
	}
	var nodes []any
	count, _ := azureFileShareInt(cp["count"])
	for i := 1; i <= count; i++ {
		nodes = append(nodes, node(rec.Name+"-control-plane-"+strconv.Itoa(i), "ControlPlane", cp, ""))
	}
	networks := azureNCWRefs(map[string]any{"networkConfiguration": rec.Properties["networkConfiguration"]})
	for _, pool := range s.azureNCWPoolsLocked(rec.ID) {
		n, _ := azureFileShareInt(pool.Properties["count"])
		for i := 1; i <= n; i++ {
			nodes = append(nodes, node(rec.Name+"-"+strings.ToLower(pool.Name)+"-"+strconv.Itoa(i), "Worker", pool.Properties, pool.ID))
		}
		networks = append(networks, azureNCWRefs(pool.Properties)...)
	}
	attached := []any{}
	for _, id := range networks {
		if !slices.ContainsFunc(attached, func(v any) bool { return strings.EqualFold(azureARMString(v), id) }) {
			attached = append(attached, id)
		}
	}
	features := []any{}
	typ, _ := lookupAzureARMResourceType(azureNCWNamespace, "kubernetesClusters/features")
	for _, f := range s.azureARMInlineChildrenLocked(rec.ID, typ) {
		features = append(features, map[string]any{"name": f.Name, "version": "1.0.0", "detailedStatus": "Running", "detailedStatusMessage": "Feature is running."})
	}
	return map[string]any{"nodes": azureNCWList(nodes), "attachedNetworkIds": attached, "featureStatuses": features}
}

// azureNCWInitialPools creates the agent pools a new cluster lists in
// initialAgentPoolConfigurations.
func azureNCWInitialPools(parent *azureARMResource) []*azureARMResource {
	now := time.Now().UTC()
	var pools []*azureARMResource
	for _, item := range azureARMSlice(parent.Properties["initialAgentPoolConfigurations"]) {
		cfg := azureARMObject(item)
		name := azureARMString(cfg["name"])
		props := azureARMWithout(cfg, "name")
		props = azureARMMerge(props, map[string]any{
			"provisioningState": "Succeeded", "detailedStatus": "Available", "detailedStatusMessage": "Agent pool is available.",
			"kubernetesVersion": parent.Properties["kubernetesVersion"],
		})
		pools = append(pools, &azureARMResource{
			ID: parent.ID + "/agentPools/" + name, SubscriptionID: parent.SubscriptionID, ResourceGroup: parent.ResourceGroup, Name: name,
			Type: azureNCWNamespace + "/" + azureNCWPoolType, Location: parent.Location, Tags: map[string]any{}, ExtendedLocation: parent.ExtendedLocation,
			Properties: props, CreatedAt: now, UpdatedAt: now, APIVersion: parent.APIVersion,
		})
	}
	return pools
}

// azureNCWPoolReconcile checks a pool's attached networks exist and its
// merged upgrade settings, takes the cluster's Kubernetes version (and its
// extendedLocation, when the pool names none) and records the pool's
// network use.
func azureNCWPoolReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	if msg := azureNCWUpgrade(rec.Properties, "properties.", true); msg != "" {
		return nil, msg
	}
	if existing == nil {
		if msg := s.azureNCWAttachedLocked(azureARMObject(rec.Properties["attachedNetworkConfiguration"]), "properties.attachedNetworkConfiguration."); msg != "" {
			return nil, msg
		}
	}
	out := map[string]any{}
	if cluster := s.azureARMResources[azureARMKey(azureHZClusterOf(rec.ID))]; cluster != nil {
		out["kubernetesVersion"] = cluster.Properties["kubernetesVersion"]
		if len(rec.ExtendedLocation) == 0 {
			rec.ExtendedLocation = cluster.ExtendedLocation
		}
	}
	s.azureNCWRelinkLocked(rec, "")
	return out, ""
}

// azureNCWSystemLeftLocked refuses to leave a cluster without a System
// agent pool that has nodes, when pool would drop to count nodes. Caller
// holds providerStorageMu.
func (s *Server) azureNCWSystemLeftLocked(pool *azureARMResource, count any) string {
	if n, _ := azureFileShareInt(count); pool.Properties["mode"] != "System" || n >= 1 {
		return ""
	}
	for _, other := range s.azureNCWPoolsLocked(azureHZClusterOf(pool.ID)) {
		if n, _ := azureFileShareInt(other.Properties["count"]); !strings.EqualFold(other.ID, pool.ID) && other.Properties["mode"] == "System" && n >= 1 {
			return ""
		}
	}
	return "Every Kubernetes cluster must contain at least one System agent pool with at least one node; " + pool.Name + " is the last one."
}

// azureNCWRestartNode restarts one of the cluster's nodes.
func azureNCWRestartNode(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := cmp.Or(azureOracleOnly(payload, "", []string{"nodeName"}), azureWorkloadsRequired(payload, "", "nodeName")); msg != "" {
		return azureOracleBad(msg)
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	current := s.azureARMResources[azureARMKey(rec.ID)]
	if current == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
	}
	name := azureARMString(payload["nodeName"])
	if !slices.ContainsFunc(azureARMSlice(current.Properties["nodes"]), func(v any) bool { return azureARMString(azureARMObject(v)["name"]) == name }) {
		return azureOracleBad("nodeName '" + name + "' isn't a node of Kubernetes cluster " + rec.Name + ".")
	}
	return http.StatusAccepted, nil
}
