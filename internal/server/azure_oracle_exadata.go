package server

import (
	"math"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	azureOracleInfraType   = "cloudExadataInfrastructures"
	azureOracleClusterType = "cloudVmClusters"
	// Learn's FAQ: a quarter rack has 2 database and 3 storage servers, and
	// an infrastructure grows to 32 and 64.
	azureOracleMinCompute, azureOracleMaxCompute = 2, 32
	azureOracleMinStorage, azureOracleMaxStorage = 3, 64
)

var (
	azureOracleWeekdays = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	azureOracleMonths   = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
)

func azureOracleRegisterExadata() {
	ns := azureOracleNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: azureOracleInfraType,
		Validate:  azureOracleInfraValidate,
		PatchBody: azureOracleInfraPatch,
		Reconcile: azureOracleInfraReconcile,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"addstoragecapacity": azureOracleAddStorage,
		},
	})
	registerAzureARMView(ns, azureOracleInfraType+"/dbServers", (*Server).azureOracleDBServers)
	azureOracleRegisterClusters()
}

func azureOracleInfraValidate(payload map[string]any) string {
	props, msg := azureHNProperties(payload)
	if msg != "" {
		return msg
	}
	if msg := azureOracleZones(payload, true); msg != "" {
		return msg
	}
	if azureARMString(props["displayName"]) == "" {
		return "properties.displayName is required."
	}
	if _, ok := azureOracleShapeNamed(azureARMString(props["shape"])); !ok {
		return "properties.shape must be one of Exadata.X9M, Exadata.X11M."
	}
	for _, key := range []string{"databaseServerType", "storageServerType"} {
		if msg := azureLabLength(props, "properties.", key, 1, 255); msg != "" {
			return msg
		}
	}
	return azureOracleInfraFields(props)
}

// azureOracleInfraFields checks the fields PUT and PATCH share.
func azureOracleInfraFields(props map[string]any) string {
	checks := []string{
		azureLabLength(props, "properties.", "displayName", 1, 255),
		azureOracleNumber(props, "properties.", "computeCount", azureOracleMinCompute, azureOracleMaxCompute, true),
		azureOracleNumber(props, "properties.", "storageCount", azureOracleMinStorage, azureOracleMaxStorage, true),
		azureOracleContacts(props, 10),
		azureOracleMaintenance(props),
	}
	for _, msg := range checks {
		if msg != "" {
			return msg
		}
	}
	return ""
}

// azureOracleZones checks the top-level availability zones: Exadata
// infrastructure needs exactly one of 1, 2 or 3.
func azureOracleZones(payload map[string]any, required bool) string {
	raw, present := payload["zones"]
	if !present && !required {
		return ""
	}
	zones, ok := raw.([]any)
	if !ok || len(zones) != 1 || !slices.Contains([]string{"1", "2", "3"}, azureARMString(zones[0])) {
		return "zones must name one availability zone: 1, 2 or 3."
	}
	return ""
}

// azureOracleMaintenance checks maintenanceWindow against Learn's
// MaintenanceWindow model.
func azureOracleMaintenance(props map[string]any) string {
	raw, present := props["maintenanceWindow"]
	if !present {
		return ""
	}
	window, ok := raw.(map[string]any)
	if !ok {
		return "properties.maintenanceWindow must be an object."
	}
	path := "properties.maintenanceWindow."
	checks := []string{
		azureLabEnum(window, path, "preference", []string{"NoPreference", "CustomPreference"}),
		azureLabEnum(window, path, "patchingMode", []string{"Rolling", "NonRolling"}),
		azureOracleNumber(window, path, "customActionTimeoutInMins", 15, 120, true),
		azureOracleNumber(window, path, "leadTimeInWeeks", 1, 4, true),
		azureOracleBools(window, path, "isCustomActionTimeoutEnabled", "isMonthlyPatchingEnabled"),
	}
	for _, msg := range checks {
		if msg != "" {
			return msg
		}
	}
	for key, allowed := range map[string][]float64{"hoursOfDay": {0, 4, 8, 12, 16, 20}, "weeksOfMonth": {1, 2, 3, 4}} {
		for _, v := range azureARMSlice(window[key]) {
			if n, ok := v.(float64); !ok || !slices.Contains(allowed, n) {
				return path + key + " values must be among " + strings.ReplaceAll(azureOracleInts(allowed), " ", ", ") + "."
			}
		}
	}
	for key, names := range map[string][]string{"daysOfWeek": azureOracleWeekdays, "months": azureOracleMonths} {
		for _, v := range azureARMSlice(window[key]) {
			if !slices.Contains(names, azureARMString(azureARMObject(v)["name"])) {
				return path + key + " entries need a name among " + strings.Join(names, ", ") + "."
			}
		}
	}
	if window["preference"] == "CustomPreference" && len(azureARMSlice(window["hoursOfDay"])) == 0 {
		return path + "hoursOfDay is required with a CustomPreference."
	}
	return ""
}

func azureOracleInts(values []float64) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = strconv.Itoa(int(v))
	}
	return strings.Join(parts, " ")
}

// azureOracleInfraPatch takes the update model: tags, zones and the
// counts, contacts, name and maintenance window.
func azureOracleInfraPatch(payload map[string]any) (map[string]any, map[string]any, string) {
	if msg := azureOracleOnly(payload, "", []string{"tags", "zones", "properties"}); msg != "" {
		return nil, nil, msg
	}
	if msg := azureDataBoxObjects(payload, "", "tags", "properties"); msg != "" {
		return nil, nil, msg
	}
	if msg := azureOracleZones(payload, false); msg != "" {
		return nil, nil, msg
	}
	props := azureARMObject(payload["properties"])
	if msg := azureOracleOnly(props, "properties.", []string{"computeCount", "customerContacts", "displayName", "maintenanceWindow", "storageCount"}); msg != "" {
		return nil, nil, msg
	}
	return payload, nil, azureOracleInfraFields(props)
}

// azureOracleInfraReconcile sizes the infrastructure from its shape and
// counts. The shape and server types never change, a scale-down can't
// drop a database server a VM cluster uses, and activated storage
// servers can't be removed. New storage servers wait for
// addStorageCapacity.
func azureOracleInfraReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	props := rec.Properties
	shape, _ := azureOracleShapeNamed(azureARMString(props["shape"]))
	compute, storage := azureOracleCount(props["computeCount"], azureOracleMinCompute), azureOracleCount(props["storageCount"], azureOracleMinStorage)
	activated := storage
	out := map[string]any{}
	if existing == nil {
		if msg := s.azureOracleEntitledLocked(rec.SubscriptionID); msg != "" {
			return nil, msg
		}
		out["ocid"], out["timeCreated"] = azureOracleOCID("cloudexadatainfrastructure", rec.ID), rec.CreatedAt.Format(time.RFC3339)
	} else {
		old := existing.Properties
		out["ocid"], out["timeCreated"] = old["ocid"], old["timeCreated"]
		for _, key := range []string{"shape", "databaseServerType", "storageServerType"} {
			if v, present := props[key]; present && !strings.EqualFold(azureARMString(v), azureARMString(old[key])) {
				return nil, "properties." + key + " can't be changed after creation."
			}
			if old[key] != nil {
				out[key] = old[key]
			}
		}
		shape, _ = azureOracleShapeNamed(azureARMString(old["shape"]))
		activated = azureOracleCount(old["activatedStorageCount"], azureOracleMinStorage)
		if storage < activated {
			return nil, "properties.storageCount can't drop below the " + strconv.Itoa(activated) + " storage servers already activated."
		}
		servers := azureOracleServerOCIDs(existing.ID, azureOracleCount(old["computeCount"], azureOracleMinCompute))
		for _, removed := range servers[min(compute, len(servers)):] {
			if cluster := s.azureOracleClusterOnLocked(existing.ID, removed); cluster != "" {
				return nil, "properties.computeCount can't drop database server " + removed + ", which VM cluster " + cluster + " uses."
			}
		}
	}
	return azureARMMerge(out, azureOracleInfraSizing(shape, compute, storage, activated)), ""
}

// azureOracleCount reads a server count, or its default.
func azureOracleCount(raw any, fallback int) int {
	switch n := raw.(type) {
	case float64: // from a request
		return int(n)
	case int: // set by the emulator
		return n
	}
	return fallback
}

func azureOracleInfraSizing(shape azureOracleShape, compute, storage, activated int) map[string]any {
	dataTB := math.Round(float64(activated)*shape.tbPerStorageServer*10) / 10
	return map[string]any{
		"shape": shape.name, "computeModel": shape.model, "computeCount": compute, "storageCount": storage,
		"activatedStorageCount": activated, "additionalStorageCount": storage - activated,
		"maxCpuCount": compute * shape.coresPerNode, "maxMemoryInGbs": compute * shape.memoryPerNode,
		"maxDbNodeStorageSizeInGbs": compute * shape.nodeGB, "maxDataStorageInTbs": dataTB,
		"totalStorageSizeInGbs": int(dataTB * 1024), "lifecycleState": "Available",
		"dbServerVersion": "24.1.0.0.0.240118", "storageServerVersion": "24.1.0.0.0.240118",
	}
}

// azureOracleAddStorage makes storage servers added by raising
// storageCount available to VM clusters.
func azureOracleAddStorage(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	stored := s.azureARMResources[azureARMKey(rec.ID)]
	if stored == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The Resource '"+rec.ID+"' was not found.")
	}
	p := stored.Properties
	storage := azureOracleCount(p["storageCount"], azureOracleMinStorage)
	if azureOracleCount(p["additionalStorageCount"], 0) == 0 {
		return azureOracleConflict("There is no storage capacity to add: raise properties.storageCount first.")
	}
	shape, _ := azureOracleShapeNamed(azureARMString(p["shape"]))
	updated := s.azureOracleUpdateLocked(rec.ID, azureOracleInfraSizing(shape, azureOracleCount(p["computeCount"], azureOracleMinCompute), storage, storage))
	return http.StatusOK, updated.fixture()
}

// azureOracleServerOCIDs are an infrastructure's database servers, one per
// compute server, in order.
func azureOracleServerOCIDs(infraID string, count int) []string {
	out := make([]string, count)
	for i := range out {
		out[i] = azureOracleOCID("dbserver", strings.ToLower(infraID)+"#"+strconv.Itoa(i+1))
	}
	return out
}

// azureOracleClustersLocked are the VM clusters on an infrastructure.
// Caller holds providerStorageMu.
func (s *Server) azureOracleClustersLocked(infraID string) []*azureARMResource {
	return s.azureARMResourcesMatchingLocked(func(r *azureARMResource) bool {
		return strings.EqualFold(r.Type, azureOracleNamespace+"/"+azureOracleClusterType) &&
			strings.EqualFold(azureARMString(r.Properties["cloudExadataInfrastructureId"]), infraID)
	})
}

// azureOracleClusterOnLocked names a VM cluster using the database
// server, or returns "". Caller holds providerStorageMu.
func (s *Server) azureOracleClusterOnLocked(infraID, server string) string {
	for _, cluster := range s.azureOracleClustersLocked(infraID) {
		if slices.Contains(azureOracleServers(cluster.Properties), server) {
			return cluster.Name
		}
	}
	return ""
}

func azureOracleServers(props map[string]any) []string {
	var out []string
	for _, v := range azureARMSlice(props["dbServers"]) {
		out = append(out, azureARMString(v))
	}
	return out
}

// azureOracleNodeShare is what one node of a VM cluster takes from its
// database server: the cluster's totals split evenly across its nodes.
func azureOracleNodeShare(props map[string]any, key string) int {
	nodes := max(len(azureARMSlice(props["dbServers"])), 1)
	total, _ := props[key].(float64)
	return int(math.Ceil(total / float64(nodes)))
}

// azureOracleUsageLocked is what the VM clusters other than skip take on
// one database server. Caller holds providerStorageMu.
func (s *Server) azureOracleUsageLocked(infraID, server, skip string) (cores, memory, storage int, clusters []string) {
	for _, cluster := range s.azureOracleClustersLocked(infraID) {
		if strings.EqualFold(cluster.ID, skip) || !slices.Contains(azureOracleServers(cluster.Properties), server) {
			continue
		}
		cores += azureOracleNodeShare(cluster.Properties, "cpuCoreCount")
		memory += azureOracleNodeShare(cluster.Properties, "memorySizeInGbs")
		storage += azureOracleNodeShare(cluster.Properties, "dbNodeStorageSizeInGbs")
		clusters = append(clusters, cluster.ID)
	}
	return cores, memory, storage, clusters
}

// azureOracleDBServers lists an infrastructure's database servers with
// what its VM clusters have allocated on each.
func (s *Server) azureOracleDBServers(scope string) []map[string]any {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	infra := s.azureARMResources[azureARMKey(scope)]
	if infra == nil {
		return nil
	}
	p := infra.Properties
	shape, _ := azureOracleShapeNamed(azureARMString(p["shape"]))
	out := []map[string]any{}
	for i, ocid := range azureOracleServerOCIDs(infra.ID, azureOracleCount(p["computeCount"], azureOracleMinCompute)) {
		cores, memory, storage, clusters := s.azureOracleUsageLocked(infra.ID, ocid, "")
		var nodes []any
		for _, cluster := range clusters {
			nodes = append(nodes, azureOracleOCID("dbnode", strings.ToLower(cluster)+"#"+ocid))
		}
		out = append(out, map[string]any{
			"id": infra.ID + "/dbServers/" + ocid, "name": ocid, "type": azureOracleNamespace + "/" + azureOracleInfraType + "/dbServers",
			"properties": map[string]any{
				"ocid": ocid, "displayName": "dbServer-" + strconv.Itoa(i+1), "exadataInfrastructureId": p["ocid"], "shape": shape.name,
				"computeModel": shape.model, "cpuCoreCount": cores, "maxCpuCount": shape.coresPerNode, "memorySizeInGbs": memory,
				"maxMemoryInGbs": shape.memoryPerNode, "dbNodeStorageSizeInGbs": storage, "maxDbNodeStorageInGbs": shape.nodeGB,
				"vmClusterIds": azureOracleStringsOrEmpty(clusters), "dbNodeIds": azureOracleAnyOrEmpty(nodes), "lifecycleState": "Available",
				"provisioningState": "Succeeded", "timeCreated": p["timeCreated"],
			},
		})
	}
	return out
}

func azureOracleStringsOrEmpty(values []string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

func azureOracleAnyOrEmpty(values []any) []any {
	if values == nil {
		return []any{}
	}
	return values
}

// azureOracleSame reports whether a re-PUT keeps an immutable value.
func azureOracleSame(a, b any) bool {
	if sa, ok := a.(string); ok {
		return strings.EqualFold(sa, azureARMString(b))
	}
	return reflect.DeepEqual(a, b)
}
