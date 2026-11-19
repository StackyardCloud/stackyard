package server

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"time"
)

// Microsoft.DBforPostgreSQL/serverGroupsv2: Azure Cosmos DB for PostgreSQL
// (Citus) clusters with their firewall rules, roles, private endpoint
// connections and coordinator and worker node parameters. Servers, private
// link resources and the cluster-wide parameter view are computed from the
// cluster, so scaling it changes them at once. No PostgreSQL runs.
const (
	azurePGClusterNamespace = "Microsoft.DBforPostgreSQL"
	azurePGClusterType      = azurePGClusterNamespace + "/serverGroupsv2"
	azurePGClusterDNSSuffix = ".postgres.database.azure.com"
	// azurePGClusterAdmin is the administrative role every cluster has.
	azurePGClusterAdmin = "citus"
	// azurePGMaxReplicas is how many read replicas a cluster can have.
	azurePGMaxReplicas = 5
	// azurePGRestoreWindow is how far back point-in-time restore reaches.
	azurePGRestoreWindow = 35 * 24 * time.Hour
)

// azurePGParameter is a server parameter both node roles have. Values come
// from the Learn server parameters reference and the REST examples.
type azurePGParameter struct {
	Name, Description, DataType, AllowedValues string
	// CoordinatorDefault and WorkerDefault differ for memory settings.
	CoordinatorDefault, WorkerDefault string
}

// ponytail: a subset of the documented parameters, sorted by name; add
// more as clients need them.
var azurePGParameters = []azurePGParameter{
	{"array_nulls", "Enable input of NULL elements in arrays.", "Boolean", "on,off", "on", "on"},
	{"backslash_quote", `Sets whether "\'" is allowed in string literals.`, "Enumeration", "safe_encoding,on,off", "safe_encoding", "safe_encoding"},
	{"citus.enable_binary_protocol", "Use PostgreSQL's binary serialization format, when applicable, to transfer data with workers.", "Boolean", "on,off", "on", "on"},
	{"citus.enable_repartition_joins", "Allow repartition joins by switching to the task-tracker executor.", "Boolean", "on,off", "off", "off"},
	{"citus.enable_version_checks", "Check that the Citus code and extension versions match.", "Boolean", "on,off", "on", "on"},
	{"citus.log_distributed_deadlock_detection", "Log distributed deadlock detection processing in the server log.", "Boolean", "on,off", "off", "off"},
	{"citus.node_connection_timeout", "Maximum time in milliseconds to wait for connection establishment.", "Integer", "10-3600000", "5000", "5000"},
	{"citus.stat_statements_track", "Record statistics for citus_stat_statements.", "Enumeration", "none,all", "all", "all"},
	{"citus.use_citus_managed_tables", "Add newly created local tables to Citus metadata.", "Boolean", "on,off", "off", "off"},
	{"citus.use_secondary_nodes", "Policy for choosing nodes for SELECT queries.", "Enumeration", "never,always", "never", "never"},
	{"work_mem", "Sets the amount of memory to be used by internal sort operations and hash tables before writing to temporary disk files.", "Integer", "4096-2097151", "158720", "115712"},
}

func azurePGFindParameter(name string) (azurePGParameter, bool) {
	i := slices.IndexFunc(azurePGParameters, func(p azurePGParameter) bool { return p.Name == name })
	if i < 0 {
		return azurePGParameter{}, false
	}
	return azurePGParameters[i], true
}

// azurePGConfigRoles maps each per-role parameter type to the role of the
// servers it configures.
var azurePGConfigRoles = map[string]string{"coordinatorConfigurations": "Coordinator", "nodeConfigurations": "Worker"}

func init() {
	ns := azurePGClusterNamespace
	configTypes := map[string]azureARMResourceType{}
	for collection, role := range azurePGConfigRoles {
		configTypes[collection] = azureARMResourceType{
			Namespace: ns, Type: "serverGroupsv2/" + collection, Proxy: true,
			ValidateName: func(name string) string {
				if _, ok := azurePGFindParameter(name); !ok {
					return "'" + name + "' isn't a server parameter of the cluster."
				}
				return ""
			},
			Validate: func(payload map[string]any) string {
				if _, ok := azureARMObject(payload["properties"])["value"].(string); !ok {
					return "properties.value is required and must be a string."
				}
				return ""
			},
			ReadOnlyProperties: func(rec *azureARMResource) map[string]any { return azurePGConfigProperties(rec, role) },
			Reconcile: func(_ *Server, _, rec *azureARMResource) (map[string]any, string) {
				param, _ := azurePGFindParameter(rec.Name)
				return nil, azurePGCheckValue(param, azureARMString(rec.Properties["value"]))
			},
			// Parameters exist with the cluster: they're only read and set.
			NoDelete: true, NoPatch: true,
		}
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "serverGroupsv2",
		ValidateName:        azurePGClusterValidateName,
		Validate:            azurePGClusterValidate,
		PatchBody:           azurePGClusterPatch,
		WriteOnlyProperties: []string{"administratorLoginPassword"},
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"provisioningState": "Succeeded", "administratorLogin": azurePGClusterAdmin}
		},
		Reconcile:       azurePGClusterReconcile,
		ChildProperties: azurePGClusterConnections,
		StateProperties: map[string]any{"state": "Ready"},
		Transitions:     map[string]map[string]any{"start": {"state": "Ready"}, "stop": {"state": "Stopped"}},
		Actions: map[string]azureARMAction{
			// Learn answers restart with 202 only.
			"restart": func(*azureARMResource, map[string]any) (int, any) { return http.StatusAccepted, nil },
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"promote": azurePGClusterPromote,
		},
		DefaultChildren: func(parent *azureARMResource) []*azureARMResource {
			var children []*azureARMResource
			for _, collection := range []string{"coordinatorConfigurations", "nodeConfigurations"} {
				for _, param := range azurePGParameters {
					children = append(children, azureARMChild(parent, configTypes[collection], param.Name, map[string]any{}))
				}
			}
			return children
		},
	})
	for _, typ := range configTypes {
		registerAzureARMResourceType(typ)
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "serverGroupsv2/firewallRules", Proxy: true,
		Validate: azureValidateFirewallRule, NoPatch: true,
		DeleteStatus: http.StatusNoContent,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "serverGroupsv2/roles", Proxy: true,
		Validate:            azurePGRoleValidate,
		ValidateInParent:    azurePGRoleInCluster,
		WriteOnlyProperties: []string{"password"},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			if azureARMString(rec.Properties["roleType"]) == "" {
				return map[string]any{"roleType": "user"}
			}
			return nil
		},
		NoPatch: true, DeleteStatus: http.StatusNoContent,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "serverGroupsv2/privateEndpointConnections", Proxy: true,
		Validate: azureFileShareConnectionValidate,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{"provisioningState": "Succeeded", "groupIds": []any{"coordinator"}}
		},
		NoPatch: true, DeleteStatus: http.StatusNoContent,
	})
	registerAzureARMView(ns, "serverGroupsv2/servers", azurePGServers)
	registerAzureARMView(ns, "serverGroupsv2/privateLinkResources", azurePGPrivateLinkResources)
	registerAzureARMView(ns, "serverGroupsv2/configurations", azurePGClusterConfigurations)
	registerAzureARMView(ns, "serverGroupsv2/servers/configurations", azurePGServerConfigurations)
}

// azurePGConfigProperties describes a per-role parameter and where its
// value comes from.
func azurePGConfigProperties(rec *azureARMResource, role string) map[string]any {
	param, _ := azurePGFindParameter(rec.Name)
	value, set := rec.Properties["value"].(string)
	if !set {
		value = param.defaultFor(role)
	}
	return map[string]any{
		"provisioningState": "Succeeded", "value": value,
		"description": param.Description, "dataType": param.DataType, "allowedValues": param.AllowedValues,
		"defaultValue": param.defaultFor(role), "source": azurePGSource(param, role, value),
	}
}

func (p azurePGParameter) defaultFor(role string) string {
	if role == "Coordinator" {
		return p.CoordinatorDefault
	}
	return p.WorkerDefault
}

func azurePGSource(param azurePGParameter, role, value string) string {
	if value == param.defaultFor(role) {
		return "system-default"
	}
	return "user-override"
}

// azurePGCheckValue checks a parameter value against its allowed values: a
// list, or an inclusive integer range written "min-max".
func azurePGCheckValue(param azurePGParameter, value string) string {
	if param.DataType == "Integer" {
		var lo, hi, n int64
		_, _ = fmt.Sscanf(param.AllowedValues, "%d-%d", &lo, &hi)
		if _, err := fmt.Sscan(value, &n); err != nil || fmt.Sprint(n) != value || n < lo || n > hi {
			return "Value '" + value + "' of " + param.Name + " must be an integer from " + fmt.Sprint(lo) + " to " + fmt.Sprint(hi) + "."
		}
		return ""
	}
	if !slices.Contains(strings.Split(param.AllowedValues, ","), value) {
		return "Value '" + value + "' of " + param.Name + " must be one of " + param.AllowedValues + "."
	}
	return ""
}

// azurePGClusterLocked returns the stored cluster with the ID, or nil.
// Caller holds providerStorageMu.
func (s *Server) azurePGClusterLocked(id string) *azureARMResource {
	rec := s.azureARMResources[azureARMKey(id)]
	if rec == nil || !strings.EqualFold(rec.Type, azurePGClusterType) {
		return nil
	}
	return rec
}

func (s *Server) azurePGCluster(id string) *azureARMResource {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	return s.azurePGClusterLocked(id)
}

// azurePGIsReplica reports whether a cluster is a read replica: it has a
// source cluster and, unlike a restored cluster, no restore point.
func azurePGIsReplica(props map[string]any) bool {
	return azureARMString(props["sourceResourceId"]) != "" && azureARMString(props["pointInTimeUTC"]) == ""
}

func azurePGInt(props map[string]any, key string) int {
	n, _ := azureFileShareInt(props[key])
	return n
}

// azurePGServerNames lists the coordinator ("{cluster}-c") and the worker
// nodes ("{cluster}-w{n}").
func azurePGServerNames(cluster string, nodeCount int) []string {
	names := []string{cluster + "-c"}
	for i := range nodeCount {
		names = append(names, fmt.Sprintf("%s-w%d", cluster, i))
	}
	return names
}

func azurePGServers(s *Server, scope string) []map[string]any {
	cluster := s.azurePGCluster(scope)
	if cluster == nil {
		return nil
	}
	p := cluster.Properties
	var items []map[string]any
	for i, name := range azurePGServerNames(cluster.Name, azurePGInt(p, "nodeCount")) {
		prefix, role := "node", "Worker"
		if i == 0 {
			prefix, role = "coordinator", "Coordinator"
		}
		haState := "NotEnabled"
		if p["enableHa"] == true {
			haState = "Healthy"
		}
		props := map[string]any{
			"fullyQualifiedDomainName": name + azurePGClusterDNSSuffix, "role": role,
			"postgresqlVersion": p["postgresqlVersion"], "citusVersion": p["citusVersion"],
			"serverEdition": p[prefix+"ServerEdition"], "storageQuotaInMb": p[prefix+"StorageQuotaInMb"], "vCores": p[prefix+"VCores"],
			"enablePublicIpAccess": p[prefix+"EnablePublicIpAccess"] == true,
			"enableHa":             p["enableHa"] == true, "haState": haState, "state": p["state"],
			"administratorLogin": azurePGClusterAdmin, "isReadOnly": false,
		}
		if zone := azureARMString(p["preferredPrimaryZone"]); zone != "" {
			props["availabilityZone"] = zone
		}
		items = append(items, map[string]any{
			"id": scope + "/servers/" + name, "name": name, "type": azurePGClusterType + "/servers",
			"systemData": azureARMSystemData(cluster.CreatedAt, cluster.UpdatedAt), "properties": props,
		})
	}
	return items
}

// azurePGPrivateLinkResources has a private link group for the coordinator
// and for each worker node.
func azurePGPrivateLinkResources(s *Server, scope string) []map[string]any {
	cluster := s.azurePGCluster(scope)
	if cluster == nil {
		return nil
	}
	groups := []string{"coordinator"}
	for i := range azurePGInt(cluster.Properties, "nodeCount") {
		groups = append(groups, fmt.Sprintf("worker-%d", i))
	}
	items := make([]map[string]any, 0, len(groups))
	for _, group := range groups {
		items = append(items, map[string]any{
			"id": scope + "/privateLinkResources/" + group, "name": group, "type": azurePGClusterType + "/privateLinkResources",
			"properties": map[string]any{
				"groupId": group, "requiredMembers": []any{group},
				"requiredZoneNames": []any{"privatelink." + strings.ToLower(cluster.Name) + azurePGClusterDNSSuffix},
			},
		})
	}
	return items
}

// azurePGRoleValues returns each parameter's per-role stored properties.
func (s *Server) azurePGRoleValues(clusterID, collection string) map[string]map[string]any {
	typ, _ := lookupAzureARMResourceType(azurePGClusterNamespace, "serverGroupsv2/"+collection)
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	out := map[string]map[string]any{}
	for _, rec := range s.azureARMInlineChildrenLocked(clusterID, typ) {
		out[rec.Name] = rec.Properties
	}
	return out
}

// azurePGClusterConfigurations lists each parameter with its coordinator
// and worker values.
func azurePGClusterConfigurations(s *Server, scope string) []map[string]any {
	if s.azurePGCluster(scope) == nil {
		return nil
	}
	coordinator, worker := s.azurePGRoleValues(scope, "coordinatorConfigurations"), s.azurePGRoleValues(scope, "nodeConfigurations")
	items := make([]map[string]any, 0, len(azurePGParameters))
	for _, param := range azurePGParameters {
		group := func(role string, values map[string]any) any {
			return map[string]any{"role": role, "value": values["value"], "defaultValue": values["defaultValue"], "source": values["source"]}
		}
		groups := []any{group("Coordinator", coordinator[param.Name]), group("Worker", worker[param.Name])}
		items = append(items, map[string]any{
			"id": scope + "/configurations/" + param.Name, "name": param.Name, "type": azurePGClusterType + "/configurations",
			"properties": map[string]any{
				"description": param.Description, "dataType": param.DataType, "allowedValues": param.AllowedValues,
				"provisioningState": "Succeeded", "serverRoleGroupConfigurations": groups,
			},
		})
	}
	return items
}

// azurePGServerConfigurations lists the parameters of one server: the
// coordinator's or the workers'.
func azurePGServerConfigurations(s *Server, scope string) []map[string]any {
	clusterID := scope[:strings.LastIndex(scope[:strings.LastIndex(scope, "/")], "/")]
	collection := "nodeConfigurations"
	if strings.HasSuffix(strings.ToLower(scope), "-c") {
		collection = "coordinatorConfigurations"
	}
	values := s.azurePGRoleValues(clusterID, collection)
	items := make([]map[string]any, 0, len(azurePGParameters))
	for _, param := range azurePGParameters {
		items = append(items, map[string]any{
			"id": scope + "/configurations/" + param.Name, "name": param.Name, "type": azurePGClusterType + "/servers/configurations",
			"properties": maps.Clone(values[param.Name]),
		})
	}
	return items
}

// azurePGClusterConnections lists the cluster's private endpoint
// connections in its properties.
func azurePGClusterConnections(s *Server, rec *azureARMResource) map[string]any {
	typ, _ := lookupAzureARMResourceType(azurePGClusterNamespace, "serverGroupsv2/privateEndpointConnections")
	connections := []any{}
	for _, child := range s.azureARMInlineChildrenLocked(rec.ID, typ) {
		connections = append(connections, map[string]any{
			"id": child.ID, "name": child.Name, "type": child.Type,
			"properties": map[string]any{
				"groupIds":                          child.Properties["groupIds"],
				"privateEndpoint":                   child.Properties["privateEndpoint"],
				"privateLinkServiceConnectionState": child.Properties["privateLinkServiceConnectionState"],
			},
		})
	}
	return map[string]any{"privateEndpointConnections": connections}
}

// azurePGClusterPromote makes a read replica an independent cluster; a
// promoted replica no longer has a source.
func azurePGClusterPromote(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	geo, present := payload["enableGeoBackup"]
	if _, ok := geo.(bool); present && !ok {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "enableGeoBackup must be a boolean.")
	}
	typ, _ := lookupAzureARMResourceType(azurePGClusterNamespace, "serverGroupsv2")
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	current := s.azurePGClusterLocked(rec.ID)
	if current == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
	}
	if !azurePGIsReplica(current.Properties) {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "Cluster "+current.Name+" isn't a read replica.")
	}
	updated := *current
	updated.UpdatedAt = time.Now().UTC()
	updated.Properties = azureARMMerge(current.Properties, nil)
	delete(updated.Properties, "sourceResourceId")
	delete(updated.Properties, "sourceLocation")
	if geo == true && azureARMString(azureARMObject(current.Properties["dataEncryption"])["type"]) == "AzureKeyVault" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "Customer-managed keys aren't supported with geo-redundant backup.")
	}
	if present {
		updated.Properties["enableGeoBackup"] = geo
	}
	updated.Properties = azureARMProperties(typ, &updated)
	s.azureARMResources[azureARMKey(rec.ID)] = &updated
	return http.StatusAccepted, nil
}

// azurePGClusterCheckName checks a cluster name; names are global, so it
// compares across all subscriptions.
func (s *Server) azurePGClusterCheckName(payload map[string]any) (int, any) {
	name := strings.TrimSpace(azureARMString(payload["name"]))
	out := map[string]any{"name": name, "type": azurePGClusterType, "nameAvailable": true}
	if msg := azurePGClusterValidateName(name); msg != "" {
		out["nameAvailable"], out["message"] = false, msg
		return http.StatusOK, out
	}
	if len(s.azureARMResourcesWhere(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, azurePGClusterType) && strings.EqualFold(rec.Name, name)
	})) > 0 {
		out["nameAvailable"], out["message"] = false, "Cluster name '"+name+"' is already in use."
	}
	return http.StatusOK, out
}
