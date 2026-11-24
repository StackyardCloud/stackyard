package server

import (
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	azureDataReplication20260501APIVersion = "2026-05-01"
	azureDREPNamespace                     = "Microsoft.DataReplication"
	azureDREPMaxHistory                    = 200 // jobs and events kept per vault, newest first
)

// Azure Site Recovery's Data Replication provider (Microsoft.DataReplication
// 2026-05-01) on the shared ARM handler: vaults, fabrics and their agents,
// policies, replication extensions, alert settings, private endpoint
// connection proxies (each creating its connection), protected items with
// planned failover and recovery points, jobs, events, operation results,
// name checks, deployment preflight and the operations list. Nothing
// replicates; protection and failover finish at once and are recorded as
// jobs and events.
var (
	azureDREPVaultTypes = []string{"DisasterRecovery", "Migrate"}
	azureDREPEmailRE    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	// azureDREPRouteRE matches the routes served by the router below.
	azureDREPRouteRE = regexp.MustCompile(`(?i)^/subscriptions/([^/]+)/resourceGroups/([^/]+)/providers/Microsoft\.DataReplication/(?:deployments/([^/]+)/preflight|locations/([^/]+)/operationResults/([^/]+))$`)
)

type azureDREPState struct {
	mu     sync.Mutex
	jobs   map[string][]map[string]any // by vault key
	events map[string][]map[string]any // by vault key
	ops    azureDMSOperations
}

func (s *Server) azureDREP() *azureDREPState {
	return azureDMSStateFor(s, "datareplication", func() *azureDREPState {
		return &azureDREPState{jobs: map[string][]map[string]any{}, events: map[string][]map[string]any{}}
	})
}

func init() {
	ns := azureDREPNamespace
	custom := func(instanceType string) func(map[string]any) string {
		return func(payload map[string]any) string {
			if got := azureARMString(azureARMObject(azureARMObject(payload["properties"])["customProperties"])["instanceType"]); got == "" {
				return "properties.customProperties.instanceType is required (for example " + instanceType + ")."
			}
			return ""
		}
	}
	health := func(*azureARMResource) map[string]any {
		return map[string]any{"health": "Normal", "healthErrors": []any{}}
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "replicationVaults", DeleteStatus: http.StatusAccepted,
		Validate: azureEnumProperty("vaultType", azureDREPVaultTypes...),
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"serviceResourceId": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String()}
		},
		Deleted: func(s *Server, removed *azureARMResource) {
			st := s.azureDREP()
			st.mu.Lock()
			defer st.mu.Unlock()
			delete(st.jobs, azureARMKey(removed.ID))
			delete(st.events, azureARMKey(removed.ID))
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "replicationFabrics", DeleteStatus: http.StatusAccepted,
		Validate: custom("FabricModelCustomProperties"),
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return azureARMMerge(health(rec), map[string]any{
				"serviceEndpoint":   "https://" + rec.Location + ".datareplication.stackyard.local/" + rec.Name,
				"serviceResourceId": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(),
			})
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "replicationFabrics/fabricAgents", Proxy: true, DeleteStatus: http.StatusAccepted, NoPatch: true,
		Validate: azureDREPAgentValidate,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return azureARMMerge(health(rec), map[string]any{
				"isResponsive": true, "lastHeartbeat": rec.UpdatedAt.Format(time.RFC3339), "versionNumber": "1.0.0",
				"correlationId": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(),
			})
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "replicationVaults/replicationPolicies", Proxy: true, DeleteStatus: http.StatusAccepted, NoPatch: true,
		Validate: custom("PolicyModelCustomProperties"),
		ValidateDelete: func(s *Server, existing *azureARMResource) string {
			return s.azureDREPInUseLocked(existing, "policyName")
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "replicationVaults/replicationExtensions", Proxy: true, DeleteStatus: http.StatusAccepted, NoPatch: true,
		Validate: custom("ReplicationExtensionModelCustomProperties"),
		ValidateDelete: func(s *Server, existing *azureARMResource) string {
			return s.azureDREPInUseLocked(existing, "replicationExtensionName")
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "replicationVaults/alertSettings", Proxy: true, NoPatch: true,
		Validate: azureDREPAlertValidate,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "replicationVaults/privateEndpointConnectionProxies", Proxy: true, DeleteStatus: http.StatusAccepted,
		Linked: azureDREPProxyLinked,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "replicationVaults/privateEndpointConnections", Proxy: true, DeleteStatus: http.StatusAccepted, NoPatch: true,
		ValidatePut: func(_ *Server, existing, _ *azureARMResource) (string, string) {
			if existing == nil {
				return "PrivateEndpointConnectionNotFound", "Private endpoint connections are created by their connection proxy."
			}
			return "", ""
		},
		Validate: azureDREPConnectionStateValidate,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "replicationVaults/protectedItems", Proxy: true, DeleteStatus: http.StatusAccepted,
		Validate:  custom("ProtectedItemModelCustomProperties"),
		Reconcile: azureDREPProtectedItemReconcile,
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{
				"protectionState": "Protected", "protectionStateDescription": "Protected", "replicationHealth": "Normal",
				"healthErrors": []any{}, "allowedJobs": []any{"PlannedFailover", "DisableProtection"}, "correlationId": uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(rec.ID))).String(),
			}
		},
		Deleted: func(s *Server, removed *azureARMResource) {
			s.azureDREP().record(removed, "DisableProtection", "Disable protection")
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){"plannedfailover": azureDREPPlannedFailover},
	})
	registerAzureARMView(ns, "replicationVaults/protectedItems/recoveryPoints", func(s *Server, scope string) []map[string]any {
		s.providerStorageMu.Lock()
		item := s.azureARMResources[azureARMKey(scope)]
		s.providerStorageMu.Unlock()
		if item == nil {
			return nil
		}
		name := uuid.NewSHA1(uuid.NameSpaceURL, []byte(azureARMKey(item.ID)+"#initial")).String()
		return []map[string]any{{
			"id": scope + "/recoveryPoints/" + name, "name": name, "type": ns + "/replicationVaults/protectedItems/recoveryPoints",
			"properties": map[string]any{
				"recoveryPointTime": item.CreatedAt.Format(time.RFC3339), "recoveryPointType": "CrashConsistent",
				"customProperties": map[string]any{"instanceType": "RecoveryPointModelCustomProperties"},
			},
		}}
	})
	history := func(pick func(*azureDREPState) map[string][]map[string]any) azureARMView {
		return func(s *Server, scope string) []map[string]any {
			st := s.azureDREP()
			st.mu.Lock()
			defer st.mu.Unlock()
			items := pick(st)[azureARMKey(scope)]
			out := make([]map[string]any, 0, len(items))
			for _, item := range items {
				out = append(out, azureARMMerge(item, nil))
			}
			return out
		}
	}
	registerAzureARMView(ns, "replicationVaults/jobs", history(func(st *azureDREPState) map[string][]map[string]any { return st.jobs }))
	registerAzureARMView(ns, "replicationVaults/events", history(func(st *azureDREPState) map[string][]map[string]any { return st.events }))
	registerAzureARMView(ns, "replicationVaults/privateLinkResources", func(_ *Server, scope string) []map[string]any {
		return []map[string]any{{
			"id": scope + "/privateLinkResources/vault", "name": "vault", "type": ns + "/replicationVaults/privateLinkResources",
			"properties": map[string]any{"groupId": "vault", "requiredMembers": []any{"vault"}, "requiredZoneNames": []any{"privatelink.siterecovery.windowsazure.com"}},
		}}
	})
	registerAzureARMLocationAction(ns, "checkNameAvailability", func(s *Server, sub, _ string, payload map[string]any) (int, any) {
		if !strings.EqualFold(azureARMString(payload["type"]), ns+"/replicationVaults") {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "type must be Microsoft.DataReplication/replicationVaults.")
		}
		return s.azureDMSCheckName("/subscriptions/"+sub, ns+"/replicationVaults", payload)
	})
	providerOps := azureDMSOperationsList(ns, "Microsoft Data Replication",
		[2]string{"replicationVaults", "vaults"}, [2]string{"replicationFabrics", "fabrics"}, [2]string{"replicationFabrics/fabricAgents", "fabric agents"},
		[2]string{"replicationVaults/replicationPolicies", "policies"}, [2]string{"replicationVaults/replicationExtensions", "replication extensions"},
		[2]string{"replicationVaults/protectedItems", "protected items"}, [2]string{"replicationVaults/alertSettings", "alert settings"})
	// GET /providers/{ns}/operations lists the provider's operations; under a
	// resource group, operations/{id} is the status of one of its jobs' operations.
	registerAzureARMView(ns, "operations", func(s *Server, scope string) []map[string]any {
		if scope == "" {
			return providerOps()
		}
		var out []map[string]any
		for _, op := range s.azureDREP().ops.within(scope + "/") {
			out = append(out, op.status(scope+"/providers/"+ns+"/operations/"+op.ID))
		}
		return out
	})
}

func azureDREPAgentValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if azureARMString(props["machineId"]) == "" || azureARMString(props["machineName"]) == "" {
		return "properties.machineId and machineName are required."
	}
	for _, key := range []string{"authenticationIdentity", "resourceAccessIdentity"} {
		identity := azureARMObject(props[key])
		for _, field := range []string{"tenantId", "applicationId", "objectId", "audience", "aadAuthority"} {
			if azureARMString(identity[field]) == "" {
				return "properties." + key + "." + field + " is required."
			}
		}
	}
	return ""
}

func azureDREPAlertValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if _, ok := props["sendToOwners"].(bool); !ok {
		return "properties.sendToOwners must be a boolean."
	}
	emails := azureARMSlice(props["customEmailAddresses"])
	if len(emails) > 20 {
		return "properties.customEmailAddresses allows at most 20 addresses."
	}
	for _, raw := range emails {
		if !azureDREPEmailRE.MatchString(azureARMString(raw)) {
			return "properties.customEmailAddresses entries must be email addresses."
		}
	}
	return ""
}

func azureDREPConnectionStateValidate(payload map[string]any) string {
	state := azureARMObject(azureARMObject(payload["properties"])["privateLinkServiceConnectionState"])
	if status := state["status"]; status != nil && !slices.Contains([]any{"Approved", "Pending", "Rejected", "Disconnected"}, status) {
		return "properties.privateLinkServiceConnectionState.status must be Approved, Pending, Rejected or Disconnected."
	}
	return ""
}

// azureDREPProxyLinked creates the private endpoint connection a proxy stands for.
func azureDREPProxyLinked(_ *Server, rec *azureARMResource) (map[string]any, []*azureARMResource, string) {
	vaultID := rec.ID[:strings.LastIndex(strings.ToLower(rec.ID), "/privateendpointconnectionproxies/")]
	connection := &azureARMResource{
		ID: vaultID + "/privateEndpointConnections/" + rec.Name, SubscriptionID: rec.SubscriptionID, ResourceGroup: rec.ResourceGroup,
		Name: rec.Name, Type: azureDREPNamespace + "/replicationVaults/privateEndpointConnections", CreatedAt: rec.CreatedAt, UpdatedAt: rec.UpdatedAt,
		Properties: map[string]any{
			"provisioningState":                 "Succeeded",
			"privateEndpoint":                   map[string]any{"id": "/subscriptions/" + rec.SubscriptionID + "/resourceGroups/" + rec.ResourceGroup + "/providers/Microsoft.Network/privateEndpoints/" + rec.Name},
			"privateLinkServiceConnectionState": map[string]any{"status": "Approved", "description": "Auto-approved", "actionsRequired": "None"},
			"groupIds":                          []any{azureARMString(rec.Properties["groupId"])},
		},
	}
	return map[string]any{"privateEndpointConnectionId": connection.ID}, []*azureARMResource{connection}, ""
}

// azureDREPInUseLocked rejects deleting a policy or extension a protected
// item names. Caller holds providerStorageMu.
func (s *Server) azureDREPInUseLocked(existing *azureARMResource, field string) string {
	vaultID := existing.ID[:strings.LastIndex(existing.ID, "/")]
	vaultID = vaultID[:strings.LastIndex(vaultID, "/")]
	prefix := azureARMKey(vaultID + "/protectedItems/")
	for _, item := range s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		return strings.HasPrefix(azureARMKey(rec.ID), prefix) && strings.EqualFold(azureARMString(rec.Properties[field]), existing.Name)
	}) {
		return "Protected item '" + item.Name + "' uses it."
	}
	return ""
}

// azureDREPProtectedItemReconcile requires the policy and replication
// extension to exist in the vault and records the protection job.
func azureDREPProtectedItemReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	vaultID := rec.ID[:strings.LastIndex(strings.ToLower(rec.ID), "/protecteditems/")]
	for _, ref := range [][2]string{{"policyName", "replicationPolicies"}, {"replicationExtensionName", "replicationExtensions"}} {
		name := azureARMString(rec.Properties[ref[0]])
		if name == "" || s.azureARMResources[azureARMKey(vaultID+"/"+ref[1]+"/"+name)] == nil {
			return nil, "properties." + ref[0] + " must name a " + ref[1][:len(ref[1])-1] + " in this vault."
		}
		if existing != nil && !strings.EqualFold(azureARMString(existing.Properties[ref[0]]), name) {
			return nil, "properties." + ref[0] + " can't change."
		}
	}
	if existing == nil {
		s.azureDREP().record(rec, "EnableProtection", "Enable protection")
	}
	return nil, ""
}

// record adds a finished job and its event for item to the vault's history,
// and the job's operation. It may run under providerStorageMu (lock order:
// store, then st.mu).
func (st *azureDREPState) record(item *azureARMResource, jobType, display string) map[string]any {
	vaultID := item.ID[:strings.LastIndex(strings.ToLower(item.ID), "/protecteditems/")]
	op := st.ops.add(vaultID, jobType, "", map[string]any{"objectId": item.ID})
	now := time.Now().UTC().Format(time.RFC3339)
	job := map[string]any{
		"id": vaultID + "/jobs/" + op.ID, "name": op.ID, "type": azureDREPNamespace + "/replicationVaults/jobs",
		"properties": map[string]any{
			"displayName": display, "state": "Succeeded", "startTime": now, "endTime": now, "objectId": item.ID, "objectName": item.Name,
			"objectInternalId": item.Properties["correlationId"], "objectType": "ProtectedItem", "replicationProviderId": item.Properties["replicationExtensionName"],
			"allowedActions": []any{}, "activityId": op.ID, "tasks": []any{}, "errors": []any{},
			"customProperties": map[string]any{"instanceType": jobType + "WorkflowModelCustomProperties"},
		},
	}
	eventName := uuid.NewString()
	event := map[string]any{
		"id": vaultID + "/events/" + eventName, "name": eventName, "type": azureDREPNamespace + "/replicationVaults/events",
		"properties": map[string]any{
			"resourceType": item.Type, "resourceName": item.Name, "eventType": jobType, "eventName": display, "timeOfOccurrence": now,
			"severity": "Information", "description": display + " completed for " + item.Name + ".", "correlationId": op.ID,
			"healthErrors": []any{}, "customProperties": map[string]any{"instanceType": "EventModelCustomProperties"},
		},
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	key := azureARMKey(vaultID)
	jobs := append([]map[string]any{job}, st.jobs[key]...)
	st.jobs[key] = jobs[:min(len(jobs), azureDREPMaxHistory)]
	events := append([]map[string]any{event}, st.events[key]...)
	st.events[key] = events[:min(len(events), azureDREPMaxHistory)]
	return azureARMMerge(job, nil)
}

func azureDREPPlannedFailover(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	custom := azureARMObject(azureARMObject(payload["properties"])["customProperties"])
	if azureARMString(custom["instanceType"]) == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "properties.customProperties.instanceType is required.")
	}
	if direction := custom["failoverDirection"]; direction != nil && direction != "PrimaryToRecovery" && direction != "RecoveryToPrimary" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "failoverDirection must be PrimaryToRecovery or RecoveryToPrimary.")
	}
	s.azureDREP().record(rec, "PlannedFailover", "Planned failover")
	return http.StatusOK, map[string]any{"properties": map[string]any{"customProperties": custom}}
}

// handleAzureDataReplication20260501Router serves deployment preflight and
// operation results, which the shared ARM handler has no route for.
func (s *Server) handleAzureDataReplication20260501Router(w http.ResponseWriter, r *http.Request) bool {
	path := strings.TrimPrefix(rawRequestPath(r), "/azure")
	m := azureDREPRouteRE.FindStringSubmatch(path)
	if m == nil {
		return azureDMSUnknownRoute(w, r, azureDREPNamespace)
	}
	if m[3] != "" {
		s.serveAzureARMPost(w, r, func(payload map[string]any) (int, any) { return azureDREPPreflight(payload) })
		return true
	}
	switch {
	case r.Method != http.MethodGet:
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	case hasAzureInvalidAPIVersion(r):
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	opID := m[5]
	op, ok := s.azureDREP().ops.get(opID)
	if !ok || !strings.HasPrefix(azureARMKey(op.Scope), azureARMKey("/subscriptions/"+m[1]+"/resourceGroups/"+m[2]+"/")) {
		azureARMNotFound(w, "OperationNotFound", path, "")
		return true
	}
	respondJSON(w, http.StatusOK, op.status(path))
	return true
}

// azureDREPPreflight checks a deployment's resources and echoes them back.
func azureDREPPreflight(payload map[string]any) (int, any) {
	resources := azureARMSlice(payload["resources"])
	if len(resources) > 100 {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "A deployment may hold at most 100 resources.")
	}
	for _, raw := range resources {
		res := azureARMObject(raw)
		typ := azureARMString(res["type"])
		if azureARMString(res["name"]) == "" || !strings.HasPrefix(strings.ToLower(typ), strings.ToLower(azureDREPNamespace+"/")) {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "Each resource needs a name and a Microsoft.DataReplication type.")
		}
		if _, ok := lookupAzureARMResourceType(azureDREPNamespace, typ[len(azureDREPNamespace)+1:]); !ok {
			return http.StatusBadRequest, azureARMErrorBody("InvalidResourceType", "Unknown resource type '"+typ+"'.")
		}
	}
	return http.StatusOK, map[string]any{"resources": resources}
}
