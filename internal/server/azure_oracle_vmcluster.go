package server

import (
	"net"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

var (
	// Learn: a cluster name starts with a letter, may hold hyphens but not
	// underscores, and is at most 11 characters.
	azureOracleClusterName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]{0,10}$`)
	azureOracleSSHKey      = regexp.MustCompile(`^(ssh-(rsa|ed25519|dss)|ecdsa-sha2-nistp(256|384|521)) [A-Za-z0-9+/=]+( .*)?$`)
	azureOracleOCIDPattern = regexp.MustCompile(`^ocid1\.[a-z0-9]+\.`)
	// azureOracleClusterFixed never changes after creation; dbServers
	// change only through addVms and removeVms.
	azureOracleClusterFixed = []string{"cloudExadataInfrastructureId", "subnetId", "vnetId", "hostname", "clusterName", "domain",
		"backupSubnetCidr", "timeZone", "giVersion", "dataStoragePercentage", "isLocalBackupEnabled", "isSparseDiskgroupEnabled",
		"scanListenerPortTcp", "scanListenerPortTcpSsl", "zoneId", "systemVersion", "dbServers"}
	// azureOracleClusterUpdatable is Learn's VM cluster update model.
	azureOracleClusterUpdatable = []string{"computeNodes", "cpuCoreCount", "dataCollectionOptions", "dataStorageSizeInTbs", "dbNodeStorageSizeInGbs",
		"displayName", "fileSystemConfigurationDetails", "isAcceleratedNetworkEnabled", "licenseModel", "memorySizeInGbs", "ocpuCount",
		"sshPublicKeys", "storageSizeInGbs"}
)

const (
	azureOracleNodeType = azureOracleClusterType + "/dbNodes"
	azureOracleVNAType  = azureOracleClusterType + "/virtualNetworkAddresses"
)

func azureOracleRegisterClusters() {
	ns := azureOracleNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: azureOracleClusterType,
		Validate:        azureOracleClusterValidate,
		PatchBody:       azureOracleClusterPatch,
		Reconcile:       azureOracleClusterReconcile,
		DefaultChildren: azureOracleClusterNodes,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"addvms":                 azureOracleVMs(true),
			"removevms":              azureOracleVMs(false),
			"listprivateipaddresses": azureOraclePrivateIPs,
		},
	})
	// Nodes come and go with the cluster's database servers: the view
	// serves GET and refuses writes, and the type carries the node action.
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: azureOracleNodeType, Proxy: true, NoPatch: true, NoDelete: true,
		Validate: func(map[string]any) string { return "Database nodes are created with their VM cluster." },
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"action": azureOracleNodeAction,
		},
	})
	registerAzureARMView(ns, azureOracleNodeType, func(s *Server, scope string) []map[string]any {
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		prefix := azureARMKey(scope) + "/dbnodes/"
		out := []map[string]any{}
		for _, rec := range s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool { return strings.HasPrefix(azureARMKey(r.ID), prefix) }) {
			out = append(out, s.azureOracleNodeFixtureLocked(rec))
		}
		return out
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: azureOracleVNAType, Proxy: true, NoPatch: true,
		ValidateName: func(name string) string {
			if len(name) > 63 {
				return "A virtual network address name is a hostname of at most 63 characters."
			}
			return ""
		},
		Validate:  azureOracleVNAValidate,
		Reconcile: azureOracleVNAReconcile,
	})
}

func azureOracleClusterValidate(payload map[string]any) string {
	props, msg := azureHNProperties(payload)
	if msg != "" {
		return msg
	}
	for _, key := range []string{"cloudExadataInfrastructureId", "displayName", "giVersion", "hostname"} {
		if azureARMString(props[key]) == "" {
			return "properties." + key + " is required."
		}
	}
	if _, present := props["cpuCoreCount"]; !present {
		return "properties.cpuCoreCount is required."
	}
	if _, present := props["sshPublicKeys"]; !present {
		return "properties.sshPublicKeys is required."
	}
	if msg := azureOracleNetwork(props, true); msg != "" {
		return msg
	}
	if name, present := props["clusterName"]; present && !azureOracleClusterName.MatchString(azureARMString(name)) {
		return "properties.clusterName must start with a letter, hold only letters, digits and hyphens, and be at most 11 characters."
	}
	if raw, present := props["dataStoragePercentage"]; present && !slices.Contains([]any{35.0, 40.0, 60.0, 80.0}, raw) {
		return "properties.dataStoragePercentage must be 35, 40, 60 or 80."
	}
	if cidr, present := props["backupSubnetCidr"]; present {
		if _, _, err := net.ParseCIDR(azureARMString(cidr)); err != nil || len(azureARMString(cidr)) > 32 {
			return "properties.backupSubnetCidr must be a CIDR block such as 192.168.252.0/22."
		}
	}
	checks := []string{
		azureOracleNumber(props, "properties.", "recoStoragePercentage", 0, 100, true),
		azureOracleNumber(props, "properties.", "sparseStoragePercentage", 0, 100, true),
		azureOracleNumber(props, "properties.", "scanListenerPortTcp", 1, 65535, true),
		azureOracleNumber(props, "properties.", "scanListenerPortTcpSsl", 1, 65535, true),
		azureLabLength(props, "properties.", "timeZone", 1, 255),
		azureLabLength(props, "properties.", "systemVersion", 1, 255),
		azureOracleBools(props, "properties.", "isLocalBackupEnabled", "isSparseDiskgroupEnabled"),
		azureOracleStrings(props, "properties.", "dbServers", 1, azureOracleMaxCompute),
		azureOracleNSG(props),
	}
	for _, msg := range checks {
		if msg != "" {
			return msg
		}
	}
	return azureOracleClusterFields(props)
}

// azureOracleClusterFields checks the fields PUT and PATCH share.
func azureOracleClusterFields(props map[string]any) string {
	checks := []string{
		azureLabLength(props, "properties.", "displayName", 1, 255),
		azureOracleNumber(props, "properties.", "cpuCoreCount", 0, 32*760, true),
		azureOracleNumber(props, "properties.", "memorySizeInGbs", 0, 32*1390, true),
		azureOracleNumber(props, "properties.", "dbNodeStorageSizeInGbs", 0, 32*2243, true),
		azureOracleNumber(props, "properties.", "storageSizeInGbs", 0, 1<<30, true),
		azureOracleNumber(props, "properties.", "dataStorageSizeInTbs", 0, 64*80, false),
		azureLabEnum(props, "properties.", "licenseModel", []string{"LicenseIncluded", "BringYourOwnLicense"}),
		azureOracleBools(props, "properties.", "isAcceleratedNetworkEnabled"),
		azureOracleStrings(props, "properties.", "sshPublicKeys", 1, 100),
		azureOracleStrings(props, "properties.", "computeNodes", 1, azureOracleMaxCompute),
	}
	for _, msg := range checks {
		if msg != "" {
			return msg
		}
	}
	for _, key := range azureARMSlice(props["sshPublicKeys"]) {
		if !azureOracleSSHKey.MatchString(azureARMString(key)) {
			return "properties.sshPublicKeys must hold OpenSSH public keys, such as ssh-rsa AAAA...."
		}
	}
	if ocpu, present := props["ocpuCount"]; present {
		n, ok := ocpu.(float64)
		if !ok || n < 0 || n*10 != float64(int64(n*10)) {
			return "properties.ocpuCount allows only one decimal place."
		}
	}
	options := azureARMObject(props["dataCollectionOptions"])
	return azureOracleBools(options, "properties.dataCollectionOptions.", "isDiagnosticsEventsEnabled", "isHealthMonitoringEnabled", "isIncidentLogsEnabled")
}

// azureOracleNSG checks nsgCidrs: a source CIDR and an optional
// destination port range.
func azureOracleNSG(props map[string]any) string {
	for _, item := range azureARMSlice(props["nsgCidrs"]) {
		rule := azureARMObject(item)
		if _, _, err := net.ParseCIDR(azureARMString(rule["source"])); err != nil {
			return "properties.nsgCidrs entries need a source CIDR block."
		}
		if ports, present := rule["destinationPortRange"]; present {
			r := azureARMObject(ports)
			if msg := azureOracleNumber(r, "properties.nsgCidrs.destinationPortRange.", "min", 1, 65535, true) + azureOracleNumber(r, "properties.nsgCidrs.destinationPortRange.", "max", 1, 65535, true); msg != "" {
				return msg
			}
			if lo, hi := r["min"].(float64), r["max"].(float64); lo > hi {
				return "properties.nsgCidrs.destinationPortRange.min can't exceed max."
			}
		}
	}
	return ""
}

func azureOracleClusterPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	if msg := azureOracleOnly(payload, "", []string{"tags", "properties"}); msg != "" {
		return nil, nil, msg
	}
	if msg := azureDataBoxObjects(payload, "", "tags", "properties"); msg != "" {
		return nil, nil, msg
	}
	props := azureARMObject(payload["properties"])
	if msg := azureOracleOnly(props, "properties.", azureOracleClusterUpdatable); msg != "" {
		return nil, nil, msg
	}
	return payload, nil, azureOracleClusterFields(props)
}

// azureOracleClusterReconcile places the cluster on its infrastructure:
// the infrastructure exists in the same region, runs the Grid
// Infrastructure version, owns the chosen database servers (all of them by
// default), and has room on each for the cluster's share of cores, memory
// and local storage.
func azureOracleClusterReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	props := rec.Properties
	out := map[string]any{}
	if existing != nil {
		for _, key := range azureOracleClusterFixed {
			if v, present := props[key]; present && !azureOracleSame(v, existing.Properties[key]) {
				return nil, "properties." + key + " can't be changed after creation."
			}
			if old, present := existing.Properties[key]; present {
				out[key] = old
			}
		}
		props = azureARMMerge(props, out)
	} else if msg := s.azureOracleEntitledLocked(rec.SubscriptionID); msg != "" {
		return nil, msg
	}
	infraID := azureARMString(props["cloudExadataInfrastructureId"])
	infra := s.azureARMResources[azureARMKey(infraID)]
	if infra == nil || !strings.EqualFold(infra.Type, azureOracleNamespace+"/"+azureOracleInfraType) {
		return nil, "The Exadata infrastructure '" + infraID + "' was not found."
	}
	if !strings.EqualFold(infra.Location, rec.Location) {
		return nil, "A VM cluster must be in its Exadata infrastructure's region, " + infra.Location + "."
	}
	shape, _ := azureOracleShapeNamed(azureARMString(infra.Properties["shape"]))
	if !slices.Contains(azureOracleGIVersions[azureARMString(props["giVersion"])], shape.name) {
		return nil, "properties.giVersion must be a Grid Infrastructure version " + shape.name + " supports; list them at locations/{location}/giVersions."
	}
	all := azureOracleServerOCIDs(infra.ID, azureOracleCount(infra.Properties["computeCount"], azureOracleMinCompute))
	servers := azureOracleServers(props)
	if len(servers) == 0 {
		servers = all
		props = azureARMMerge(props, map[string]any{"dbServers": azureOracleStringsOrEmpty(all)})
	}
	if msg := azureOracleOwned(servers, all, "properties.dbServers"); msg != "" {
		return nil, msg
	}
	if msg := azureOracleOwned(azureOracleStringsList(props["computeNodes"]), all, "properties.computeNodes"); msg != "" {
		return nil, msg
	}
	// Memory and local storage default to the per-VM minimums, so every
	// cluster takes its share of each server.
	sized := map[string]any{}
	for key, perNode := range map[string]float64{"memorySizeInGbs": azureOracleMinMemoryPerVM, "dbNodeStorageSizeInGbs": azureOracleMinStoragePerVM} {
		if _, present := props[key]; !present {
			sized[key] = perNode * float64(len(servers))
		}
	}
	props = azureARMMerge(props, sized)
	if msg := s.azureOracleFitsLocked(infra, rec.ID, props, shape); msg != "" {
		return nil, msg
	}
	out = azureARMMerge(out, sized)
	defaults := map[string]any{
		"dataStoragePercentage": 80.0, "licenseModel": "LicenseIncluded", "backupSubnetCidr": "192.168.252.0/22", "timeZone": "UTC",
		"scanListenerPortTcp": 1521.0, "scanListenerPortTcpSsl": 2484.0, "isLocalBackupEnabled": false, "isSparseDiskgroupEnabled": false,
		"isAcceleratedNetworkEnabled": false, "systemVersion": azureOracleSystemVersions[len(azureOracleSystemVersions)-1],
	}
	for key, value := range defaults {
		if _, present := props[key]; !present {
			out[key] = value
		}
	}
	domain := azureARMString(props["domain"])
	if domain == "" {
		domain = strings.ToLower(rec.Name) + ".oraclevcn.com"
		out["domain"] = domain
	}
	return azureARMMerge(out, map[string]any{
		"dbServers": props["dbServers"], "nodeCount": len(servers), "computeModel": shape.model, "shape": shape.name,
		"ocid": azureOracleOCID("cloudvmcluster", rec.ID), "subnetOcid": azureOracleOCID("subnet", azureARMString(props["subnetId"])),
		"lifecycleState": "Available", "diskRedundancy": "High", "listenerPort": azureOracleCount(props["scanListenerPortTcp"], 1521),
		"scanDnsName":   strings.ToLower(azureARMString(props["hostname"])) + "-scan." + domain,
		"compartmentId": azureOracleOCID("compartment", rec.SubscriptionID), "timeCreated": rec.CreatedAt.Format(time.RFC3339),
		"ociUrl": "https://cloud.oracle.com/dbaas/cloudvmclusters/" + azureOracleOCID("cloudvmcluster", rec.ID),
	}), ""
}

func azureOracleStringsList(raw any) []string {
	var out []string
	for _, v := range azureARMSlice(raw) {
		out = append(out, azureARMString(v))
	}
	return out
}

// azureOracleOwned checks that servers are distinct database servers of
// the infrastructure.
func azureOracleOwned(servers, all []string, what string) string {
	seen := map[string]bool{}
	for _, server := range servers {
		if !slices.Contains(all, server) {
			return what + " names " + server + ", which isn't a database server of the Exadata infrastructure."
		}
		if seen[server] {
			return what + " names " + server + " twice."
		}
		seen[server] = true
	}
	return ""
}

// azureOracleFitsLocked checks each chosen server has room for the
// cluster's per-node share beside the other clusters on it. Caller holds
// providerStorageMu.
func (s *Server) azureOracleFitsLocked(infra *azureARMResource, clusterID string, props map[string]any, shape azureOracleShape) string {
	need := map[string][2]int{
		"cpuCoreCount":           {azureOracleNodeShare(props, "cpuCoreCount"), shape.coresPerNode},
		"memorySizeInGbs":        {azureOracleNodeShare(props, "memorySizeInGbs"), shape.memoryPerNode},
		"dbNodeStorageSizeInGbs": {azureOracleNodeShare(props, "dbNodeStorageSizeInGbs"), shape.nodeGB},
	}
	for _, server := range azureOracleServers(props) {
		cores, memory, storage, _ := s.azureOracleUsageLocked(infra.ID, server, clusterID)
		usage := map[string]int{"cpuCoreCount": cores, "memorySizeInGbs": memory, "dbNodeStorageSizeInGbs": storage}
		for _, key := range []string{"cpuCoreCount", "memorySizeInGbs", "dbNodeStorageSizeInGbs"} {
			if want, used := need[key], usage[key]; used+want[0] > want[1] {
				return "properties." + key + " needs " + strconv.Itoa(want[0]) + " per node, but database server " + server + " has " + strconv.Itoa(want[1]-used) + " left."
			}
		}
	}
	return ""
}

// azureOracleClusterNodes are a new cluster's database nodes, one per
// database server.
func azureOracleClusterNodes(cluster *azureARMResource) []*azureARMResource {
	out := []*azureARMResource{}
	for _, server := range azureOracleServers(cluster.Properties) {
		out = append(out, azureOracleNode(cluster, server))
	}
	return out
}

func azureOracleNode(cluster *azureARMResource, server string) *azureARMResource {
	typ := azureARMResourceTypes[strings.ToLower(azureOracleNamespace+"/"+azureOracleNodeType)]
	ocid := azureOracleOCID("dbnode", strings.ToLower(cluster.ID)+"#"+server)
	p := cluster.Properties
	index := len(azureOracleServers(p))
	if i := slices.Index(azureOracleServers(p), server); i >= 0 {
		index = i
	}
	return azureARMChild(cluster, typ, ocid, map[string]any{
		"ocid": ocid, "dbServerId": server, "dbSystemId": p["ocid"], "lifecycleState": "Available",
		"hostname":     strings.ToLower(azureARMString(p["hostname"])) + "-" + strconv.Itoa(index+1),
		"cpuCoreCount": azureOracleNodeShare(p, "cpuCoreCount"), "memorySizeInGbs": azureOracleNodeShare(p, "memorySizeInGbs"),
		"dbNodeStorageSizeInGbs": azureOracleNodeShare(p, "dbNodeStorageSizeInGbs"), "faultDomain": "FAULT-DOMAIN-" + strconv.Itoa(index%3+1),
		"vnicId": azureOracleOCID("vnic", ocid), "hostIpId": azureOracleOCID("privateip", ocid), "timeCreated": cluster.CreatedAt.Format(time.RFC3339),
	})
}

// azureOracleVMs adds or removes database nodes on database servers; a
// cluster keeps at least one node.
func azureOracleVMs(add bool) func(*Server, *azureARMResource, map[string]any) (int, any) {
	return func(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
		if azureOracleStrings(payload, "", "dbServers", 1, azureOracleMaxCompute) != "" || payload["dbServers"] == nil {
			return azureOracleBad("dbServers must be a non-empty list of database server OCIDs.")
		}
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		cluster := s.azureARMResources[azureARMKey(rec.ID)]
		if cluster == nil {
			return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
		}
		infra := s.azureARMResources[azureARMKey(azureARMString(cluster.Properties["cloudExadataInfrastructureId"]))]
		if infra == nil {
			return azureOracleConflict("The VM cluster's Exadata infrastructure no longer exists.")
		}
		current, change := azureOracleServers(cluster.Properties), azureOracleStringsList(payload["dbServers"])
		next := slices.Clone(current)
		for _, server := range change {
			switch {
			case add && slices.Contains(next, server):
				return azureOracleBad("Database server " + server + " already hosts a node of this cluster.")
			case add:
				next = append(next, server)
			case !slices.Contains(next, server):
				return azureOracleBad("Database server " + server + " hosts no node of this cluster.")
			default:
				next = slices.DeleteFunc(next, func(v string) bool { return v == server })
			}
		}
		if len(next) == 0 {
			return azureOracleBad("A VM cluster keeps at least one database node.")
		}
		all := azureOracleServerOCIDs(infra.ID, azureOracleCount(infra.Properties["computeCount"], azureOracleMinCompute))
		if msg := azureOracleOwned(next, all, "dbServers"); msg != "" {
			return azureOracleBad(msg)
		}
		if !add {
			for _, server := range change {
				node := azureOracleNode(cluster, server)
				if vna := s.azureOracleVNAOnLocked(cluster.ID, node.Name); vna != "" {
					return azureOracleConflict("Virtual network address " + vna + " is on the node of database server " + server + "; delete it first.")
				}
			}
		}
		props := azureARMMerge(cluster.Properties, map[string]any{"dbServers": azureOracleStringsOrEmpty(next), "nodeCount": len(next)})
		shape, _ := azureOracleShapeNamed(azureARMString(infra.Properties["shape"]))
		if msg := s.azureOracleFitsLocked(infra, cluster.ID, props, shape); msg != "" {
			return azureOracleBad(msg)
		}
		updated := s.azureOracleUpdateLocked(cluster.ID, props)
		taken := map[string]bool{}
		for _, server := range current {
			id := azureARMKey(azureOracleNode(cluster, server).ID)
			if old := s.azureARMResources[id]; old != nil && slices.Contains(next, server) {
				taken[azureARMString(old.Properties["hostname"])] = true
			} else {
				delete(s.azureARMResources, id)
			}
		}
		for _, server := range next {
			// Every node's share changes with the node count; a surviving
			// node keeps its state, hostname and fault domain.
			node := azureOracleNode(updated, server)
			if old := s.azureARMResources[azureARMKey(node.ID)]; old != nil {
				node.CreatedAt = old.CreatedAt
				node.Properties = azureARMMerge(node.Properties, map[string]any{
					"lifecycleState": old.Properties["lifecycleState"], "hostname": old.Properties["hostname"], "faultDomain": old.Properties["faultDomain"],
				})
			} else {
				// A new node takes the first free host number.
				base := strings.ToLower(azureARMString(props["hostname"])) + "-"
				for n := 1; ; n++ {
					if host := base + strconv.Itoa(n); !taken[host] {
						taken[host] = true
						node.Properties = azureARMMerge(node.Properties, map[string]any{"hostname": host, "faultDomain": "FAULT-DOMAIN-" + strconv.Itoa((n-1)%3+1)})
						break
					}
				}
			}
			s.azureARMResources[azureARMKey(node.ID)] = node
		}
		return http.StatusOK, updated.fixture()
	}
}

// azureOracleNodeAction starts, stops or resets a database node.
func azureOracleNodeAction(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	action := azureARMString(payload["action"])
	from := map[string]string{"Start": "Stopped", "Stop": "Available", "SoftReset": "Available", "Reset": "Available"}[action]
	if from == "" {
		return azureOracleBad("action must be one of Start, Stop, SoftReset, Reset.")
	}
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	node := s.azureARMResources[azureARMKey(rec.ID)]
	if node == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
	}
	if state := azureARMString(node.Properties["lifecycleState"]); state != from {
		return azureOracleConflict(action + " needs a node that is " + from + "; this one is " + state + ".")
	}
	state := map[string]string{"Start": "Available", "Stop": "Stopped"}[action]
	if state == "" {
		state = "Available"
	}
	return http.StatusOK, s.azureOracleNodeFixtureLocked(s.azureOracleUpdateLocked(rec.ID, map[string]any{"lifecycleState": state}))
}

// azureOracleNodeFixtureLocked is a node as GET returns it: its share of
// the cluster's cores, memory and storage follows the cluster's current
// size. Caller holds providerStorageMu.
func (s *Server) azureOracleNodeFixtureLocked(node *azureARMResource) map[string]any {
	out := node.fixture()
	if cluster := s.azureARMResources[azureARMKey(azureWorkloadsParentID(node.ID))]; cluster != nil {
		p := cluster.Properties
		out["properties"] = azureARMMerge(node.Properties, map[string]any{
			"cpuCoreCount": azureOracleNodeShare(p, "cpuCoreCount"), "memorySizeInGbs": azureOracleNodeShare(p, "memorySizeInGbs"),
			"dbNodeStorageSizeInGbs": azureOracleNodeShare(p, "dbNodeStorageSizeInGbs"),
		})
	}
	return out
}

// azureOraclePrivateIPs lists the private IP addresses of the cluster's
// nodes on a VNIC in the cluster's subnet.
func azureOraclePrivateIPs(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	subnet, vnic := azureARMString(payload["subnetId"]), azureARMString(payload["vnicId"])
	if !azureOracleOCIDPattern.MatchString(subnet) || !azureOracleOCIDPattern.MatchString(vnic) {
		return azureOracleBad("subnetId and vnicId must be OCIDs.")
	}
	out := []any{}
	if !strings.EqualFold(subnet, azureARMString(rec.Properties["subnetOcid"])) {
		return http.StatusOK, out
	}
	prefix := azureARMKey(rec.ID) + "/dbnodes/"
	nodes := s.azureARMResourcesWhere(func(r *azureARMResource) bool { return strings.HasPrefix(azureARMKey(r.ID), prefix) })
	for _, node := range nodes {
		p := node.Properties
		host := azureARMString(p["hostname"])
		n, _ := strconv.Atoi(host[strings.LastIndex(host, "-")+1:])
		if azureARMString(p["vnicId"]) != vnic {
			continue
		}
		out = append(out, map[string]any{
			"displayName": azureARMString(p["hostname"]) + "-ip", "hostnameLabel": p["hostname"], "ocid": p["hostIpId"],
			"ipAddress": "10.0.0." + strconv.Itoa(10+n), "subnetId": subnet,
		})
	}
	return http.StatusOK, out
}

func azureOracleVNAValidate(payload map[string]any) string {
	props, msg := azureHNProperties(payload)
	if msg != "" {
		return msg
	}
	if ip, present := props["ipAddress"]; present && net.ParseIP(azureARMString(ip)).To4() == nil {
		return "properties.ipAddress must be an IPv4 address."
	}
	return azureLabLength(props, "properties.", "vmOcid", 1, 255)
}

// azureOracleVNAReconcile assigns a virtual IP on the cluster: its address
// is unique in the cluster and its VM, when named, is one of the
// cluster's nodes.
func azureOracleVNAReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	clusterID := azureWorkloadsParentID(rec.ID)
	cluster := s.azureARMResources[azureARMKey(clusterID)]
	if cluster == nil {
		return nil, "The VM cluster '" + clusterID + "' was not found."
	}
	props := rec.Properties
	fixed := map[string]any{
		"ocid": azureOracleOCID("vnicattachment", rec.ID), "lifecycleState": "Available",
		"domain": strings.ToLower(rec.Name) + "." + azureARMString(cluster.Properties["domain"]), "timeAssigned": rec.CreatedAt.Format(time.RFC3339),
	}
	if existing != nil {
		for _, key := range []string{"ipAddress", "vmOcid"} {
			if v, present := props[key]; present && !azureOracleSame(v, existing.Properties[key]) {
				return nil, "properties." + key + " can't be changed after creation."
			}
			if old, present := existing.Properties[key]; present {
				fixed[key] = old
			}
		}
		return fixed, ""
	}
	if vm := azureARMString(props["vmOcid"]); vm != "" && s.azureARMResources[azureARMKey(clusterID+"/dbNodes/"+vm)] == nil {
		return nil, "properties.vmOcid must be the OCID of one of the VM cluster's database nodes."
	}
	siblings := s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool {
		return strings.HasPrefix(azureARMKey(r.ID), azureARMKey(clusterID)+"/virtualnetworkaddresses/") && !strings.EqualFold(r.ID, rec.ID)
	})
	used := map[string]bool{}
	for _, sibling := range siblings {
		used[azureARMString(sibling.Properties["ipAddress"])] = true
	}
	ip := azureARMString(props["ipAddress"])
	if used[ip] {
		return nil, "properties.ipAddress " + ip + " is already a virtual IP of this cluster."
	}
	for i := 100; ip == "" && i <= 254; i++ {
		if candidate := "10.0.1." + strconv.Itoa(i); !used[candidate] {
			ip = candidate
		}
	}
	if ip == "" {
		return nil, "The cluster has no free virtual IP left; name properties.ipAddress."
	}
	fixed["ipAddress"] = ip
	return fixed, ""
}

// Per-VM minimums Oracle sets for Exadata VM clusters; a cluster that
// doesn't size memory or local storage takes these.
const (
	azureOracleMinMemoryPerVM  = 30
	azureOracleMinStoragePerVM = 60
)

// azureOracleVNAOnLocked names a virtual network address of the cluster
// placed on the node, if any. Caller holds providerStorageMu.
func (s *Server) azureOracleVNAOnLocked(clusterID, nodeOCID string) string {
	prefix := azureARMKey(clusterID) + "/virtualnetworkaddresses/"
	for _, r := range s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool { return strings.HasPrefix(azureARMKey(r.ID), prefix) }) {
		if strings.EqualFold(azureARMString(r.Properties["vmOcid"]), nodeOCID) {
			return r.Name
		}
	}
	return ""
}
