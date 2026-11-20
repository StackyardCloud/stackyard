package server

import (
	"cmp"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Azure HorizonDB (Microsoft.HorizonDb 2026-10-01-preview): clusters with
// their pools, nodes, endpoints, firewall rules, maintenance events,
// authentication methods, Microsoft Entra administrators, private
// endpoint connections, log capture settings and log files, plus
// parameter groups and the regional capability and default parameter
// group catalogs, and change data capture (provider_azure_horizondb_cdc.go).
// No PostgreSQL runs: lifecycle states settle at once.

const (
	azureHZNamespace   = "Microsoft.HorizonDb"
	azureHZClusterType = azureHZNamespace + "/clusters"
	azureHZPoolType    = azureHZClusterType + "/pools"
	azureHZDNSSuffix   = ".horizondb.azure.com"
)

var (
	azureHZVersions   = []string{"17", "18"}
	azureHZStates     = []string{"Enabled", "Disabled"}
	azureHZNameRE     = regexp.MustCompile(`^[a-z0-9](?:[-a-z0-9]*[a-z0-9])?$`)
	azureHZClusterID  = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.HorizonDb/clusters/[^/]+$`)
	azureHZIdentityID = regexp.MustCompile(`(?i)^/subscriptions/[^/]+/resourceGroups/[^/]+/providers/Microsoft\.ManagedIdentity/userAssignedIdentities/[^/]+$`)
	azureHZKeyURI     = regexp.MustCompile(`^https://[A-Za-z0-9-]+[.]vault([.]azure[.]net|[.]usgovcloudapi[.]net|[.]azure[.]cn)/keys/[A-Za-z0-9-]+/?$`)
)

func init() {
	ns := azureHZNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters",
		ValidateName:       azureHZName("cluster"),
		Validate:           azureHZClusterValidate,
		PatchBody:          azureHZClusterPatch,
		Reconcile:          azureHZClusterReconcile,
		ReadOnlyProperties: azureHZClusterReadOnly,
		ChildProperties: func(s *Server, rec *azureARMResource) map[string]any {
			s.azureHZCDCRefreshLocked(rec.ID)
			return azureHZReadWritePool(s, rec)
		},
		DefaultChildren: azureHZClusterChildren,
		DeleteStatus:    http.StatusNoContent,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"failover": azureHZFailover,
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/privateEndpointConnections", Proxy: true,
		Validate: azureFileShareConnectionValidate,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"groupIds": []any{azureHZNameAt(azureHZClusterOf(rec.ID))}}
		},
		ValidatePut: func(s *Server, _, rec *azureARMResource) (string, string) {
			status := azureARMString(azureARMObject(rec.Properties["privateLinkServiceConnectionState"])["status"])
			if status == "Approved" && len(s.azureHZPoolsLocked(azureHZClusterOf(rec.ID))) == 0 {
				return "PrivateEndpointRequiresPool", "A private endpoint connection can be approved only when the cluster has a pool."
			}
			return "", ""
		},
		NoPatch: true, DeleteStatus: http.StatusNoContent,
	})
	registerAzureARMView(ns, "clusters/endpoints", (*Server).azureHZEndpoints)
	registerAzureARMView(ns, "clusters/privateLinkResources", func(_ *Server, scope string) []map[string]any {
		name := azureHZNameAt(scope)
		return []map[string]any{{
			"id": scope + "/privateLinkResources/" + name, "name": name, "type": ns + "/clusters/privateLinkResources",
			"properties": map[string]any{"groupId": name, "requiredMembers": []any{name}, "requiredZoneNames": []any{"privatelink" + azureHZDNSSuffix}},
		}}
	})
	registerAzureARMView(ns, "operations", func(*Server, string) []map[string]any {
		var ops []map[string]any
		for _, op := range [][3]string{
			{"clusters/read", "HorizonDB cluster", "Read HorizonDB clusters"},
			{"clusters/write", "HorizonDB cluster", "Create or replace HorizonDB clusters"},
			{"clusters/delete", "HorizonDB cluster", "Delete HorizonDB clusters"},
			{"clusters/failover/action", "HorizonDB cluster", "Fail over HorizonDB clusters"},
			{"clusters/pools/read", "HorizonDB pool", "Read HorizonDB pools"},
			{"clusters/pools/write", "HorizonDB pool", "Create or replace HorizonDB pools"},
			{"clusters/pools/delete", "HorizonDB pool", "Delete HorizonDB pools"},
			{"parameterGroups/read", "HorizonDB parameter group", "Read HorizonDB parameter groups"},
			{"parameterGroups/write", "HorizonDB parameter group", "Create or replace HorizonDB parameter groups"},
			{"clusters/pools/maintenanceEvents/reschedule/action", "HorizonDB maintenance event", "Reschedule maintenance"},
		} {
			ops = append(ops, map[string]any{
				"name": ns + "/" + op[0], "isDataAction": false,
				"display": map[string]any{"provider": "Microsoft HorizonDB", "resource": op[1], "operation": op[2], "description": op[2] + "."},
			})
		}
		return ops
	})
	azureHZRegisterPools()
	azureHZRegisterSettings()
	azureHZRegisterCDC()
}

// azureHZName returns a ValidateName hook for the lowercase names clusters,
// pools and endpoints share: 1 to 63 lowercase letters, digits and
// hyphens, starting and ending with a letter or digit.
func azureHZName(kind string) func(string) string {
	return func(name string) string {
		if len(name) > 63 || !azureHZNameRE.MatchString(name) {
			return "The " + kind + " name '" + name + "' must be 1 to 63 lowercase letters, digits and hyphens, starting and ending with a letter or digit."
		}
		return ""
	}
}

// azureHZClusterOf returns the cluster ID a child resource ID is under.
func azureHZClusterOf(id string) string {
	parts := strings.Split(id, "/")
	if len(parts) < 9 {
		return id
	}
	return strings.Join(parts[:9], "/")
}

// azureHZNameAt returns the last segment of a resource ID.
func azureHZNameAt(id string) string { return id[strings.LastIndex(id, "/")+1:] }

// azureHZClusterLocked returns the stored cluster with the ID, or nil.
// Caller holds providerStorageMu.
func (s *Server) azureHZClusterLocked(id string) *azureARMResource {
	rec := s.azureARMResources[azureARMKey(id)]
	if rec == nil || !strings.EqualFold(rec.Type, azureHZClusterType) {
		return nil
	}
	return rec
}

// azureHZPoolsLocked returns the cluster's pools, sorted by ID. Caller
// holds providerStorageMu.
func (s *Server) azureHZPoolsLocked(clusterID string) []*azureARMResource {
	typ, _ := lookupAzureARMResourceType(azureHZNamespace, "clusters/pools")
	return s.azureARMInlineChildrenLocked(clusterID, typ)
}

// azureHZGUID is a deterministic GUID derived from text.
func azureHZGUID(text string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(strings.ToLower(text))).String()
}

func azureHZTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// azureHZClusterShape checks the cluster properties a PUT or PATCH sends
// for types and allowed values; rules that need the stored cluster are in
// azureHZClusterReconcile.
func azureHZClusterShape(props map[string]any) string {
	network, backup, encryption := azureARMObject(props["network"]), azureARMObject(props["backup"]), azureARMObject(props["dataEncryption"])
	_, pointErr := azureLabTime(props, "properties.", "pointInTimeUtc")
	msg := cmp.Or(
		azureDataBoxObjects(props, "properties.", "network", "backup", "dataEncryption"),
		azureWorkloadsEnums(props, "properties.", nil, map[string][]string{
			"postgreSqlVersion": azureHZVersions, "creationMode": {"Create", "PointInTimeRestore"}, "aiModelManagement": azureHZStates,
		}),
		azureLabEnum(network, "properties.network.", "publicAccess", azureHZStates),
		azureLabEnum(backup, "properties.backup.", "geoRedundancy", azureHZStates),
		azureOracleNumber(backup, "properties.backup.", "retentionDays", 1, 35, true),
		azureLabLength(props, "properties.", "creationSourceClusterLocation", 1, 90),
		pointErr,
		azureLabEnum(encryption, "properties.dataEncryption.", "type", []string{"SystemManaged", "AzureKeyVault"}),
	)
	if msg != "" {
		return msg
	}
	if raw, present := props["creationSourceClusterResourceId"]; present && !azureHZClusterID.MatchString(azureARMString(raw)) {
		return "properties.creationSourceClusterResourceId must be a HorizonDB cluster ID."
	}
	if raw, present := encryption["primaryKeyUri"]; present && !azureHZKeyURI.MatchString(azureARMString(raw)) {
		return "properties.dataEncryption.primaryKeyUri must be a versionless Azure Key Vault key URI, such as https://{vault}.vault.azure.net/keys/{key}."
	}
	if raw, present := encryption["primaryUserAssignedIdentityId"]; present && !azureHZIdentityID.MatchString(azureARMString(raw)) {
		return "properties.dataEncryption.primaryUserAssignedIdentityId must be a user-assigned managed identity ID."
	}
	return ""
}

// azureHZIdentity checks a cluster's optional identity: None, or
// UserAssigned with user-assigned identity IDs as keys.
func azureHZIdentity(payload map[string]any) string {
	raw, present := payload["identity"]
	if !present {
		return ""
	}
	identity, ok := raw.(map[string]any)
	if !ok {
		return "identity must be an object."
	}
	if msg := azureWorkloadsEnums(identity, "identity.", []string{"type"}, map[string][]string{"type": {"None", "UserAssigned"}}); msg != "" {
		return msg
	}
	ids, present := identity["userAssignedIdentities"]
	if !present {
		return ""
	}
	entries, ok := ids.(map[string]any)
	if !ok || (identity["type"] == "None" && len(entries) > 0) {
		return "identity.userAssignedIdentities must be an object, and empty when identity.type is None."
	}
	for id := range entries {
		if !azureHZIdentityID.MatchString(id) {
			return "identity.userAssignedIdentities key '" + id + "' must be a user-assigned managed identity ID."
		}
	}
	return ""
}

func azureHZClusterValidate(payload map[string]any) string {
	if msg := azureDataBoxObjects(payload, "", "properties"); msg != "" {
		return msg
	}
	return cmp.Or(azureHZClusterShape(azureARMObject(payload["properties"])), azureHZIdentity(payload))
}

// azureHZClusterPatch limits a PATCH to the cluster's update model.
func azureHZClusterPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	props := azureARMObject(payload["properties"])
	msg := cmp.Or(
		azureOracleOnly(payload, "", []string{"tags", "identity", "properties"}),
		azureDataBoxObjects(payload, "", "properties"),
		azureOracleOnly(props, "properties.", []string{"network", "backup", "dataEncryption", "aiModelManagement"}),
		azureOracleOnly(azureARMObject(props["backup"]), "properties.backup.", []string{"retentionDays"}),
		azureHZClusterShape(props),
		azureHZIdentity(payload),
	)
	return payload, nil, msg
}

// azureHZCreateOnly are the cluster properties fixed at creation: a PUT
// may omit or repeat them but not change them.
var azureHZCreateOnly = []string{"postgreSqlVersion", "creationMode", "pointInTimeUtc", "creationSourceClusterResourceId", "creationSourceClusterLocation"}

// azureHZClusterReconcile applies the creation-mode rules, defaults and
// the rules that compare a write with the stored cluster. rec is not yet
// stored, so it may be changed in place.
func azureHZClusterReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	props := rec.Properties
	backup := azureARMObject(props["backup"])
	out := map[string]any{}
	if existing != nil {
		for _, key := range azureHZCreateOnly {
			stored, sent := existing.Properties[key], props[key]
			switch {
			case sent == nil && stored != nil:
				out[key] = stored
			case sent != nil && stored == nil && key == "pointInTimeUtc" && existing.Properties["creationSourceClusterLocation"] != nil:
				// Cross-location restores never return the timestamp.
				delete(props, key)
			case sent != nil && sent != stored && !azureHZSameTime(key, sent, stored):
				return nil, "properties." + key + " can't be changed after the cluster is created."
			}
		}
		storedGeo := azureARMObject(existing.Properties["backup"])["geoRedundancy"]
		if geo, sent := backup["geoRedundancy"]; sent && geo != storedGeo {
			return nil, "properties.backup.geoRedundancy can't be changed after the cluster is created."
		}
		out["backup"] = map[string]any{"geoRedundancy": storedGeo}
	} else {
		created, msg := s.azureHZCreation(rec)
		if msg != "" {
			return nil, msg
		}
		out = created
	}
	if azureARMObject(props["network"])["publicAccess"] == nil {
		out["network"] = map[string]any{"publicAccess": "Disabled"}
	}
	if backup["retentionDays"] == nil {
		out["backup"] = azureARMMerge(azureARMObject(out["backup"]), map[string]any{"retentionDays": 7.0})
	}
	if len(rec.Identity) == 0 && existing != nil {
		rec.Identity = existing.Identity
	}
	rec.Identity = azureHZIdentityIDs(rec.Identity)
	if existing != nil {
		if msg := s.azureHZAssignedIdentityLocked(rec.ID, rec.Identity); msg != "" {
			return nil, msg
		}
	}
	encryption, msg := azureHZEncryption(existing, rec, azureARMMerge(backup, azureARMObject(out["backup"])))
	if msg != "" {
		return nil, msg
	}
	out["dataEncryption"] = encryption
	return out, ""
}

// azureHZCreation checks a new cluster's creation mode and returns the
// properties it implies: a point-in-time restore takes its PostgreSQL
// version and geo-redundancy from the source cluster. Caller holds
// providerStorageMu.
func (s *Server) azureHZCreation(rec *azureARMResource) (map[string]any, string) {
	props := rec.Properties
	mode := cmp.Or(azureARMString(props["creationMode"]), "Create")
	out := map[string]any{"creationMode": mode}
	if mode == "Create" {
		for _, key := range azureHZCreateOnly[2:] {
			if props[key] != nil {
				return nil, "properties." + key + " is only allowed when properties.creationMode is PointInTimeRestore."
			}
		}
		switch {
		case props["postgreSqlVersion"] == nil:
			return nil, "properties.postgreSqlVersion is required."
		case azureARMObject(props["backup"])["geoRedundancy"] == nil:
			return nil, "properties.backup.geoRedundancy is required."
		}
		return out, ""
	}
	if props["postgreSqlVersion"] != nil {
		return nil, "properties.postgreSqlVersion must be omitted for a point-in-time restore: it comes from the source backup."
	}
	sourceID := azureARMString(props["creationSourceClusterResourceId"])
	at, _ := time.Parse(time.RFC3339, azureARMString(props["pointInTimeUtc"]))
	if sourceID == "" || at.IsZero() {
		return nil, "properties.pointInTimeUtc and properties.creationSourceClusterResourceId are required for a point-in-time restore."
	}
	source := s.azureHZClusterLocked(sourceID)
	if source == nil {
		return nil, "Source cluster " + sourceID + " was not found."
	}
	sourceLocation := azureARMString(props["creationSourceClusterLocation"])
	sameLocation := strings.EqualFold(source.Location, rec.Location)
	switch {
	case sourceLocation != "" && !strings.EqualFold(sourceLocation, source.Location):
		return nil, "properties.creationSourceClusterLocation must be the source cluster's location, " + source.Location + "."
	case !sameLocation && sourceLocation == "":
		return nil, "properties.creationSourceClusterLocation is required when the source cluster is in another location."
	case sameLocation && (at.Before(source.CreatedAt.Truncate(time.Second)) || at.After(time.Now())):
		return nil, "properties.pointInTimeUtc must be between the source cluster's earliest restore point, " + azureHZTime(source.CreatedAt) + ", and now."
	case !sameLocation:
		// The latest backup is restored; the timestamp isn't returned.
		delete(props, "pointInTimeUtc")
	}
	out["postgreSqlVersion"] = source.Properties["postgreSqlVersion"]
	out["backup"] = map[string]any{"geoRedundancy": azureARMObject(source.Properties["backup"])["geoRedundancy"]}
	return out, ""
}

// azureHZEncryption checks the effective data encryption and returns it:
// SystemManaged unless chosen at creation, never switched, and for
// AzureKeyVault a key and an assigned identity, changing one at a time.
func azureHZEncryption(existing, rec *azureARMResource, backup map[string]any) (map[string]any, string) {
	sent := azureARMObject(rec.Properties["dataEncryption"])
	stored := map[string]any{"type": "SystemManaged"}
	if existing != nil {
		stored = azureARMObject(existing.Properties["dataEncryption"])
	}
	kind := azureARMString(sent["type"])
	switch {
	case existing == nil && len(sent) > 0 && kind == "":
		return nil, "properties.dataEncryption.type is required."
	case existing == nil && kind == "":
		kind = "SystemManaged"
	case existing != nil && kind == "":
		kind = azureARMString(stored["type"])
	case existing != nil && kind != stored["type"]:
		return nil, "properties.dataEncryption.type can't be changed after the cluster is created."
	}
	// Updates keep the stored key and identity they omit.
	base := map[string]any{}
	if existing != nil {
		base = stored
	}
	out := azureARMMerge(azureARMMerge(base, sent), map[string]any{"type": kind})
	key, identity := azureARMString(out["primaryKeyUri"]), azureARMString(out["primaryUserAssignedIdentityId"])
	if kind == "SystemManaged" {
		if key != "" || identity != "" {
			return nil, "properties.dataEncryption key and identity must be omitted for SystemManaged encryption."
		}
		return out, ""
	}
	switch {
	case key == "" || identity == "":
		return nil, "properties.dataEncryption.primaryKeyUri and primaryUserAssignedIdentityId are required for AzureKeyVault encryption."
	case backup["geoRedundancy"] == "Enabled":
		return nil, "AzureKeyVault encryption requires properties.backup.geoRedundancy Disabled."
	case existing != nil && key != stored["primaryKeyUri"] && identity != stored["primaryUserAssignedIdentityId"]:
		return nil, "Change properties.dataEncryption.primaryKeyUri and primaryUserAssignedIdentityId in separate updates."
	}
	for id := range azureARMObject(rec.Identity["userAssignedIdentities"]) {
		if strings.EqualFold(id, identity) {
			return out, ""
		}
	}
	return nil, "properties.dataEncryption.primaryUserAssignedIdentityId must also be assigned in identity.userAssignedIdentities."
}

// azureHZIdentityIDs fills the client and principal IDs of each assigned
// identity and drops removed (null) entries. Type None clears them.
// ponytail: PATCH replaces userAssignedIdentities like PUT rather than
// merging entries; merge when a client depends on it.
func azureHZIdentityIDs(identity map[string]any) map[string]any {
	if len(identity) == 0 {
		return nil
	}
	if identity["type"] == "None" {
		return map[string]any{"type": "None"}
	}
	ids := map[string]any{}
	for id, value := range azureARMObject(identity["userAssignedIdentities"]) {
		if value != nil {
			ids[id] = map[string]any{"principalId": azureHZGUID(id + "#principal"), "clientId": azureHZGUID(id + "#client")}
		}
	}
	return map[string]any{"type": "UserAssigned", "userAssignedIdentities": ids}
}

func azureHZClusterReadOnly(rec *azureARMResource) map[string]any {
	out := map[string]any{"state": "Ready", "backup": map[string]any{"earliestRestorePointUtc": azureHZTime(rec.CreatedAt)}}
	if encryption := azureARMObject(rec.Properties["dataEncryption"]); encryption["type"] == "AzureKeyVault" {
		out["dataEncryption"] = map[string]any{
			"primaryEncryptionKeyStatus":         "Valid",
			"primaryEncryptionKeyCurrentVersion": azureWorkloadsHash(azureARMString(encryption["primaryKeyUri"]), 32),
		}
	}
	return out
}

// azureHZReadWritePool designates the cluster's pool, if it has one, to
// host the read-write node. This API version allows one pool per cluster.
func azureHZReadWritePool(s *Server, rec *azureARMResource) map[string]any {
	designated := map[string]any{}
	if pools := s.azureHZPoolsLocked(rec.ID); len(pools) > 0 {
		designated["designatedPoolResourceId"] = pools[0].ID
	}
	return map[string]any{"readWritePool": designated}
}

// azureHZFailover moves the read-write role within the designated pool,
// which needs a second running node to take it over.
func azureHZFailover(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	cluster := s.azureHZClusterLocked(rec.ID)
	pools := s.azureHZPoolsLocked(rec.ID)
	switch {
	case cluster == nil:
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The cluster was not found.")
	case len(pools) == 0:
		return azureOracleConflict("The cluster has no read-write pool to fail over.")
	case pools[0].Properties["state"] != "Ready":
		return azureOracleConflict("The read-write pool must be Ready to fail over.")
	case azureHZCount(pools[0].Properties["nodeCount"]) < 2:
		return azureOracleConflict("The read-write pool needs at least two nodes to fail over.")
	}
	// ponytail: roles are recomputed from the pool, so node-01 keeps the
	// read-write role; track the active node if a client needs it to move.
	return http.StatusOK, cluster.fixture()
}

// azureHZCount returns a JSON number as an int, or 0.
func azureHZCount(raw any) int {
	n, _ := raw.(float64)
	return int(n)
}

// azureHZEndpoints lists the cluster's read-write endpoint, named after
// the cluster, and one read-only endpoint per pool, named after the pool.
func (s *Server) azureHZEndpoints(scope string) []map[string]any {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	cluster := s.azureHZClusterLocked(scope)
	if cluster == nil {
		return nil
	}
	host := cluster.Name + "." + azureWorkloadsHash(cluster.ID, 12) + "." + strings.ToLower(strings.ReplaceAll(cluster.Location, " ", "")) + azureHZDNSSuffix
	endpoint := func(name, scopeName, owner, fqdn, role string) map[string]any {
		return map[string]any{
			"id": cluster.ID + "/endpoints/" + name, "name": name, "type": azureHZClusterType + "/endpoints",
			"properties": map[string]any{"scope": scopeName, "associatedResourceId": owner, "fullyQualifiedDomainName": fqdn, "role": role},
		}
	}
	out := []map[string]any{endpoint(cluster.Name, "Cluster", cluster.ID, host, "ReadWrite")}
	for _, pool := range s.azureHZPoolsLocked(cluster.ID) {
		out = append(out, endpoint(pool.Name, "Pool", pool.ID, pool.Name+"."+host, "ReadOnly"))
	}
	return out
}

// azureHZClusterChildren creates the configurations every cluster has:
// both authentication methods, disabled, and server log capture.
func azureHZClusterChildren(cluster *azureARMResource) []*azureARMResource {
	auth, _ := lookupAzureARMResourceType(azureHZNamespace, "clusters/authentications")
	logs, _ := lookupAzureARMResourceType(azureHZNamespace, "clusters/logCaptureConfigurations")
	return []*azureARMResource{
		azureARMChild(cluster, auth, "passwordMethod", map[string]any{"state": "Disabled"}),
		azureARMChild(cluster, auth, "microsoftEntra", map[string]any{"state": "Disabled"}),
		azureARMChild(cluster, logs, "PostgreSqlServer", map[string]any{"state": "Enabled", "retentionDays": 7.0}),
	}
}

// azureHZSameTime reports whether sent and stored pointInTimeUtc values
// name the same instant, however they're written.
func azureHZSameTime(key string, sent, stored any) bool {
	a, errA := time.Parse(time.RFC3339, azureARMString(sent))
	b, errB := time.Parse(time.RFC3339, azureARMString(stored))
	return key == "pointInTimeUtc" && errA == nil && errB == nil && a.Equal(b)
}
