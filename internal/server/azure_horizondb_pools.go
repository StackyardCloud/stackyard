package server

import (
	"cmp"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
)

// HorizonDB pools and what hangs off them: nodes (computed from the
// pool's size), firewall rules, maintenance events and the log files its
// nodes write.

var (
	azureHZProcessors     = []string{"AMD", "Intel"}
	azureHZHAModes        = []string{"Disabled", "AnyZone", "ZoneRedundant"}
	azureHZApplyMethods   = []string{"Immediate", "PendingReboot"}
	azureHZFirewallName   = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9_.-]{0,78}[a-zA-Z0-9_])?$`)
	azureHZIPv4           = regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])[.]){3}(?:25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9]?[0-9])$`)
	azureHZStatusFilter   = regexp.MustCompile(`^properties/status eq '([A-Za-z]+)'$`)
	azureHZMaxGeneration  = 3.0 // the newest compute generation the capability catalog lists
	azureHZMaintenanceDue = 7 * 24 * time.Hour
)

func azureHZRegisterPools() {
	ns := azureHZNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/pools", Proxy: true,
		ValidateName: azureHZName("pool"),
		Validate: func(payload map[string]any) string {
			return cmp.Or(azureDataBoxObjects(payload, "", "properties"), azureHZPoolShape(azureARMObject(payload["properties"]), true))
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			props := azureARMObject(payload["properties"])
			return payload, nil, cmp.Or(
				azureOracleOnly(payload, "", []string{"properties"}),
				azureDataBoxObjects(payload, "", "properties"),
				azureOracleOnly(props, "properties.", []string{"computeModel", "nodeCount", "highAvailability", "parameterGroup", "scheduledMaintenance"}),
				azureHZPoolShape(props, false),
			)
		},
		ValidatePut: func(s *Server, existing, rec *azureARMResource) (string, string) {
			if existing == nil && len(s.azureHZPoolsLocked(azureHZClusterOf(rec.ID))) > 0 {
				return "", "This API version supports one pool per cluster."
			}
			return "", ""
		},
		Reconcile:       azureHZPoolReconcile,
		StateProperties: map[string]any{"state": "Ready"},
		DefaultChildren: azureHZPoolChildren,
		Deleted:         azureHZPoolDeleted,
		DeleteStatus:    http.StatusNoContent,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"start":   azureHZPoolPower("Stopped", "Ready", "Ready"),
			"stop":    azureHZPoolPower("Ready", "Stopped", "Stopped"),
			"restart": azureHZPoolPower("Ready", "", "Ready"),
		},
	})
	registerAzureARMView(ns, "clusters/pools/nodes", (*Server).azureHZNodes)
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/pools/firewallRules", Proxy: true,
		ValidateName: func(name string) string {
			if !azureHZFirewallName.MatchString(name) {
				return "The firewall rule name '" + name + "' must be 1 to 80 letters, digits, '_', '.' and '-', starting with a letter or digit and ending with a letter, digit or '_'."
			}
			return ""
		},
		Validate: func(payload map[string]any) string {
			props := azureARMObject(payload["properties"])
			for _, key := range []string{"startIpAddress", "endIpAddress"} {
				if !azureHZIPv4.MatchString(azureARMString(props[key])) {
					return "properties." + key + " is required and must be a dotted-decimal IPv4 address."
				}
			}
			return cmp.Or(azureValidateFirewallRule(payload), azureLabLength(props, "properties.", "description", 0, 1024))
		},
		NoPatch: true, DeleteStatus: http.StatusNoContent,
	})
	// Maintenance events are stored for their actions; the view serves
	// GET with $filter and answers writes with 405.
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/pools/maintenanceEvents", Proxy: true,
		NoPut: true, NoPatch: true, NoDelete: true,
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){
			"applynow":   azureHZApplyNow,
			"reschedule": azureHZReschedule,
		},
	})
	registerAzureARMQueryView(ns, "clusters/pools/maintenanceEvents", (*Server).azureHZMaintenanceEvents)
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/logFiles", Proxy: true,
		NoPut: true, NoPatch: true, NoDelete: true,
		Actions: map[string]azureARMAction{"listdownloadurls": azureHZLogDownload},
	})
}

// azureHZPoolShape checks pool properties for types and allowed values.
// create requires what a PUT must send; PATCH sends any subset.
func azureHZPoolShape(props map[string]any, create bool) string {
	compute, ha := azureARMObject(props["computeModel"]), azureARMObject(props["highAvailability"])
	group, maintenance := azureARMObject(props["parameterGroup"]), azureARMObject(props["scheduledMaintenance"])
	if create {
		for _, key := range []string{"computeModel", "nodeCount", "highAvailability"} {
			if props[key] == nil {
				return "properties." + key + " is required."
			}
		}
		for _, key := range []string{"type", "computeProcessor", "computeGeneration", "vCores"} {
			if compute[key] == nil {
				return "properties.computeModel." + key + " is required."
			}
		}
		if ha["mode"] == nil {
			return "properties.highAvailability.mode is required."
		}
	}
	required := []string{}
	if props["parameterGroup"] != nil {
		required = []string{"parameterGroupResourceId", "applyMethod"}
	}
	maintenanceRequired := []string{}
	if props["scheduledMaintenance"] != nil {
		maintenanceRequired = []string{"maintenanceManagementType"}
	}
	return cmp.Or(
		azureDataBoxObjects(props, "properties.", "computeModel", "highAvailability", "parameterGroup", "scheduledMaintenance"),
		azureWorkloadsEnums(compute, "properties.computeModel.", nil, map[string][]string{"type": {"Provisioned"}, "computeProcessor": azureHZProcessors}),
		azureOracleNumber(compute, "properties.computeModel.", "computeGeneration", 1, azureHZMaxGeneration, true),
		azureOracleNumber(compute, "properties.computeModel.", "vCores", 2, 192, true),
		azureHZVCoresOffered(compute),
		azureOracleNumber(props, "properties.", "nodeCount", 1, 16, true),
		azureLabEnum(ha, "properties.highAvailability.", "mode", azureHZHAModes),
		azureWorkloadsEnums(group, "properties.parameterGroup.", required, map[string][]string{"applyMethod": azureHZApplyMethods}),
		azureWorkloadsEnums(maintenance, "properties.scheduledMaintenance.", maintenanceRequired, map[string][]string{"maintenanceManagementType": {"System"}}),
	)
}

// azureHZDefaultGroupID is the default parameter group a cluster's pools
// start with.
func azureHZDefaultGroupID(cluster *azureARMResource) string {
	return "/subscriptions/" + cluster.SubscriptionID + "/providers/" + azureHZNamespace + "/locations/" + azureHZLocation(cluster.Location) +
		"/parameterGroups/" + azureHZDefaultGroupName(azureARMString(cluster.Properties["postgreSqlVersion"]))
}

func azureHZDefaultGroupName(version string) string {
	return "default-postgresql-" + version + "-provisioned"
}

func azureHZLocation(location string) string {
	return strings.ToLower(strings.ReplaceAll(location, " ", ""))
}

// azureHZPoolReconcile checks the effective node count and
// high-availability mode, the pool name, and the parameter group, which
// starts as the cluster's default. rec is not yet stored.
func azureHZPoolReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	cluster := s.azureHZClusterLocked(azureHZClusterOf(rec.ID))
	if cluster == nil {
		return nil, "The pool's cluster was not found."
	}
	if strings.EqualFold(rec.Name, cluster.Name) {
		return nil, "The pool name must differ from its cluster's name, which the cluster's endpoint uses."
	}
	props := rec.Properties
	nodes, mode := azureHZCount(props["nodeCount"]), azureARMString(azureARMObject(props["highAvailability"])["mode"])
	if (nodes == 1) != (mode == "Disabled") {
		return nil, "properties.highAvailability.mode must be Disabled when nodeCount is 1, and AnyZone or ZoneRedundant when nodeCount is 2 through 16."
	}
	out := map[string]any{}
	if props["scheduledMaintenance"] == nil {
		out["scheduledMaintenance"] = map[string]any{"maintenanceManagementType": "System"}
	}
	group := azureARMObject(props["parameterGroup"])
	defaultID := azureHZDefaultGroupID(cluster)
	switch id := azureARMString(group["parameterGroupResourceId"]); {
	case len(group) == 0 && existing != nil:
		group = azureARMObject(existing.Properties["parameterGroup"])
	case len(group) == 0:
		group = map[string]any{"parameterGroupResourceId": defaultID, "applyMethod": "Immediate"}
	case strings.EqualFold(id, defaultID):
	case existing == nil:
		return nil, "A new pool's properties.parameterGroup must be the cluster's default parameter group, " + defaultID + "."
	default:
		pg := s.azureARMResources[azureARMKey(id)]
		if pg == nil || !strings.EqualFold(pg.Type, azureHZNamespace+"/parameterGroups") {
			return nil, "Parameter group " + id + " was not found."
		}
		if pg.Properties["postgreSqlVersion"] != cluster.Properties["postgreSqlVersion"] {
			return nil, "The parameter group's PostgreSQL version must match the cluster's."
		}
	}
	// ponytail: parameter changes apply at once, so pools are always in
	// sync; track PendingReboot if a client waits on restarts.
	out["parameterGroup"] = azureARMMerge(group, map[string]any{"syncStatus": "InSync"})
	return out, ""
}

// azureHZPoolPower returns a pool power action: it runs when the pool is
// in from (or already in to, as a no-op), moves it to state when set, and
// answers 200 with the pool.
func azureHZPoolPower(from, state, to string) func(*Server, *azureARMResource, map[string]any) (int, any) {
	return func(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
		s.providerStorageMu.Lock()
		defer s.providerStorageMu.Unlock()
		current := s.azureARMResources[azureARMKey(rec.ID)]
		if current == nil {
			return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The pool was not found.")
		}
		now := current.Properties["state"]
		if now != from && now != to {
			return azureOracleConflict("The pool is " + azureARMString(now) + "; this action needs it " + from + ".")
		}
		if state != "" {
			current = s.azureOracleUpdateLocked(rec.ID, map[string]any{"state": state})
		}
		return http.StatusOK, current.fixture()
	}
}

// azureHZNodes lists a running pool's nodes. node-01 hosts the read-write
// role; zone-redundant pools spread across zones 1 to 3.
func (s *Server) azureHZNodes(scope string) []map[string]any {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	pool := s.azureARMResources[azureARMKey(scope)]
	if pool == nil || pool.Properties["state"] == "Stopped" {
		return nil
	}
	zoneRedundant := azureARMObject(pool.Properties["highAvailability"])["mode"] == "ZoneRedundant"
	var out []map[string]any
	for i := range azureHZCount(pool.Properties["nodeCount"]) {
		name, role, zone := fmt.Sprintf("node-%02d", i+1), "ReadOnly", "1"
		if i == 0 {
			role = "ReadWrite"
		}
		if zoneRedundant {
			zone = fmt.Sprint(i%3 + 1)
		}
		out = append(out, map[string]any{
			"id": pool.ID + "/nodes/" + name, "name": name, "type": azureHZPoolType + "/nodes",
			"properties": map[string]any{"role": role, "availabilityZone": zone},
		})
	}
	return out
}

// azureHZPoolChildren creates a pool's first planned maintenance event
// and the server log file its read-write node writes, which belongs to
// the cluster and outlives the pool.
func azureHZPoolChildren(pool *azureARMResource) []*azureARMResource {
	eventType, _ := lookupAzureARMResourceType(azureHZNamespace, "clusters/pools/maintenanceEvents")
	hash := strings.ToUpper(azureWorkloadsHash(pool.ID, 7))
	id := hash[:4] + "-" + hash[4:]
	start := pool.CreatedAt.Add(azureHZMaintenanceDue).Truncate(time.Hour)
	event := azureARMChild(pool, eventType, id, map[string]any{
		"maintenanceEventId": id, "maintenanceType": "PlannedMaintenance", "status": "Planned",
		"description":  "Scheduled platform maintenance for the HorizonDB pool.",
		"startTimeUtc": azureHZTime(start), "endTimeUtc": azureHZTime(start.Add(time.Hour)), "originalStartTimeUtc": azureHZTime(start),
		"estimatedDowntime": "PT60S", "deferrable": true, "deferralDeadlineUtc": azureHZTime(start.Add(2 * azureHZMaintenanceDue)),
		"lastUpdatedTimeUtc": azureHZTime(pool.CreatedAt),
	})
	logType, _ := lookupAzureARMResourceType(azureHZNamespace, "clusters/logFiles")
	cluster := &azureARMResource{ID: azureHZClusterOf(pool.ID), SubscriptionID: pool.SubscriptionID, CreatedAt: pool.CreatedAt, UpdatedAt: pool.UpdatedAt, APIVersion: pool.APIVersion}
	created := azureHZTime(pool.CreatedAt)
	log := azureARMChild(cluster, logType, azureHZGUID(pool.ID+"#log"), map[string]any{
		"logType": "PostgreSqlServer", "createdAtUtc": created, "lastModifiedAtUtc": created,
		"retentionExpiresAtUtc": azureHZTime(pool.CreatedAt.Add(azureHZMaintenanceDue)), "isMutable": true,
		"sizeInKilobytes": 64.0, "pool": pool.Name, "node": "node-01", "role": "ReadWrite",
	})
	log.ResourceGroup = pool.ResourceGroup
	return []*azureARMResource{event, log}
}

// azureHZPoolDeleted disconnects the cluster's private endpoints when its
// last pool goes: they stay, rejected, for inspection and cleanup.
func azureHZPoolDeleted(s *Server, removed *azureARMResource) {
	clusterID := azureHZClusterOf(removed.ID)
	if len(s.azureHZPoolsLocked(clusterID)) > 0 {
		return
	}
	typ, _ := lookupAzureARMResourceType(azureHZNamespace, "clusters/privateEndpointConnections")
	for _, conn := range s.azureARMInlineChildrenLocked(clusterID, typ) {
		s.azureOracleUpdateLocked(conn.ID, map[string]any{"privateLinkServiceConnectionState": map[string]any{
			"status": "Rejected", "description": "Disconnected because the last pool in the cluster was deleted.",
			"actionsRequired": "Create a pool before requesting approval again.",
		}})
	}
}

// azureHZMaintenanceEvents lists a pool's maintenance events, filtered by
// $filter clauses "properties/status eq '{status}'" joined with "or". A
// filter it can't read matches nothing.
func (s *Server) azureHZMaintenanceEvents(scope string, query url.Values) []map[string]any {
	var statuses []string
	if filter := strings.TrimSpace(query.Get("$filter")); filter != "" {
		for _, clause := range strings.Split(filter, " or ") {
			m := azureHZStatusFilter.FindStringSubmatch(strings.TrimSpace(clause))
			if m == nil {
				return nil
			}
			statuses = append(statuses, m[1])
		}
	}
	typ, _ := lookupAzureARMResourceType(azureHZNamespace, "clusters/pools/maintenanceEvents")
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	var out []map[string]any
	for _, event := range s.azureARMInlineChildrenLocked(scope, typ) {
		if statuses == nil || slices.Contains(statuses, azureARMString(event.Properties["status"])) {
			out = append(out, event.fixture())
		}
	}
	return out
}

// azureHZEventAction updates a planned or rescheduled event and answers
// with the maintenance event action response.
func azureHZEventAction(s *Server, rec *azureARMResource, props func(current map[string]any) (map[string]any, string), appliedNow bool) (int, any) {
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	current := s.azureARMResources[azureARMKey(rec.ID)]
	if current == nil {
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The maintenance event was not found.")
	}
	if status := current.Properties["status"]; status != "Planned" && status != "Rescheduled" {
		return azureOracleConflict("The maintenance event is " + azureARMString(status) + "; only planned or rescheduled events can change.")
	}
	changes, msg := props(current.Properties)
	if msg != "" {
		return azureOracleBad(msg)
	}
	changes["lastUpdatedTimeUtc"] = azureHZTime(time.Now())
	updated := s.azureOracleUpdateLocked(rec.ID, changes).Properties
	return http.StatusOK, map[string]any{
		"maintenanceEventId": updated["maintenanceEventId"], "poolResourceId": rec.ID[:strings.LastIndex(rec.ID, "/maintenanceEvents/")],
		"status": updated["status"], "plannedStartTimeUtc": updated["startTimeUtc"], "plannedEndTimeUtc": updated["endTimeUtc"],
		"appliedNow": appliedNow, "lastUpdatedTimeUtc": updated["lastUpdatedTimeUtc"],
	}
}

func azureHZApplyNow(s *Server, rec *azureARMResource, _ map[string]any) (int, any) {
	return azureHZEventAction(s, rec, func(current map[string]any) (map[string]any, string) {
		now := time.Now().UTC().Truncate(time.Second)
		start, _ := time.Parse(time.RFC3339, azureARMString(current["startTimeUtc"]))
		end, _ := time.Parse(time.RFC3339, azureARMString(current["endTimeUtc"]))
		return map[string]any{"status": "InProgress", "startTimeUtc": azureHZTime(now), "endTimeUtc": azureHZTime(now.Add(end.Sub(start)))}, ""
	}, true)
}

// azureHZReschedule postpones a deferrable event to a later time no later
// than its deferral deadline.
func azureHZReschedule(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	return azureHZEventAction(s, rec, func(current map[string]any) (map[string]any, string) {
		to, err := time.Parse(time.RFC3339, azureARMString(payload["postponeToDateTimeUtc"]))
		if err != nil {
			return nil, "postponeToDateTimeUtc is required and must be an RFC 3339 date and time."
		}
		start, _ := time.Parse(time.RFC3339, azureARMString(current["startTimeUtc"]))
		end, _ := time.Parse(time.RFC3339, azureARMString(current["endTimeUtc"]))
		deadline, _ := time.Parse(time.RFC3339, azureARMString(current["deferralDeadlineUtc"]))
		switch {
		case current["deferrable"] != true:
			return nil, "The maintenance event can't be rescheduled."
		case !to.After(time.Now()) || !to.After(start) || to.After(deadline):
			return nil, "postponeToDateTimeUtc must be in the future, after the current start " + azureHZTime(start) + ", and no later than the deferral deadline, " + azureHZTime(deadline) + "."
		}
		return map[string]any{
			"status": "Rescheduled", "rescheduledFromUtc": cmp.Or(current["originalStartTimeUtc"], current["startTimeUtc"]),
			"startTimeUtc": azureHZTime(to), "endTimeUtc": azureHZTime(to.Add(end.Sub(start))),
		}, ""
	}, false)
}

// azureHZLogDownload returns a short-lived signed URL for a log file; only
// this action exposes it.
func azureHZLogDownload(rec *azureARMResource, _ map[string]any) (int, any) {
	expires := azureHZTime(time.Now().Add(time.Hour))
	account := "hzlogs" + azureWorkloadsHash(azureHZClusterOf(rec.ID), 12)
	return http.StatusOK, map[string]any{
		"downloadUri":  "https://" + account + ".blob.core.windows.net/logs/" + rec.Name + ".log?se=" + url.QueryEscape(expires) + "&sp=r&sig=" + azureWorkloadsHash(rec.ID+expires, 43),
		"expiresAtUtc": expires,
	}
}

// azureHZVCoresOffered checks a valid vCores count against the sizes the
// capability catalog offers.
func azureHZVCoresOffered(compute map[string]any) string {
	if v, ok := compute["vCores"].(float64); ok && !slices.Contains(azureHZVCores, int(v)) {
		return "properties.computeModel.vCores must be one of the capability catalog's sizes: 2, 4, 8, 16, 32, 48, 64, 96, 128 or 192."
	}
	return ""
}
