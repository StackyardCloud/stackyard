package server

import (
	"cmp"
	"net/http"
	"reflect"
	"strings"
	"time"
)

// HorizonDB change data capture (Microsoft Fabric Mirroring): the
// cluster's singleton capability, reusable Fabric destinations with the
// user-assigned identities authorized for them, and per-database
// configurations. Nothing replicates: states settle at once and a
// configuration's destination session stays NotConnected.

const (
	azureHZCapabilityType  = azureHZClusterType + "/changeDataCapture"
	azureHZDestinationType = azureHZClusterType + "/changeDataCaptureDestinations"
	azureHZAssignmentType  = azureHZDestinationType + "/identityAssignments"
	azureHZCDCConfigType   = azureHZClusterType + "/changeDataCaptureConfigurations"
	azureHZCDCExtension    = "azure_cdc"
)

// azureHZCDCServerOwned are the properties only the service sets on each
// change data capture type; values clients send for them are dropped.
var azureHZCDCServerOwned = map[string][]string{
	azureHZCapabilityType:  {"effectiveState", "replicationArchitecture", "extension", "prerequisites", "destinationCount", "configurationCount", "status"},
	azureHZDestinationType: {"effectiveState", "activeIdentityAssignmentResourceId", "configurationCount", "status"},
	azureHZAssignmentType:  {"effectiveState", "activationState", "principalId", "tenantId", "validation", "status"},
	azureHZCDCConfigType:   {"logicalStreamId", "effectiveState", "extension", "replication", "topology", "status"},
}

func azureHZRegisterCDC() {
	ns := azureHZNamespace
	desiredOnly := func(payload map[string]any, extra ...string) (map[string]any, map[string]any, string) {
		props := azureARMObject(payload["properties"])
		return payload, nil, cmp.Or(
			azureOracleOnly(payload, "", []string{"properties"}),
			azureDataBoxObjects(payload, "", "properties"),
			azureOracleOnly(props, "properties.", append([]string{"desiredState"}, extra...)),
			azureLabEnum(props, "properties.", "desiredState", azureHZStates),
			azureOracleBools(props, "properties.", "autoUpgradeExtension"),
		)
	}
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/changeDataCapture", Proxy: true, ETag: true,
		ValidateName: func(name string) string {
			if name != "default" {
				return "The change data capture capability name must be default."
			}
			return ""
		},
		Validate: func(payload map[string]any) string {
			return cmp.Or(azureHZDesired(payload), azureOracleBools(azureARMObject(payload["properties"]), "properties.", "autoUpgradeExtension"))
		},
		PatchBody: func(payload map[string]any) (map[string]any, map[string]any, string) {
			return desiredOnly(payload, "autoUpgradeExtension")
		},
		Reconcile: azureHZCapabilityReconcile,
		ValidateDelete: func(s *Server, existing *azureARMResource) string {
			cluster := azureHZClusterOf(existing.ID)
			if len(s.azureHZChildrenLocked(cluster, azureHZDestinationType))+len(s.azureHZChildrenLocked(cluster, azureHZCDCConfigType)) > 0 {
				return "Delete the cluster's change data capture destinations and configurations before the capability."
			}
			return ""
		},
		DeleteStatus: http.StatusNoContent,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/changeDataCaptureDestinations", Proxy: true, ETag: true,
		ValidateName: azureHZName("change data capture destination"),
		Validate:     azureHZDestinationValidate,
		PatchBody:    func(payload map[string]any) (map[string]any, map[string]any, string) { return desiredOnly(payload) },
		ValidatePut:  azureHZNeedsCapability,
		Reconcile:    azureHZDestinationReconcile,
		ValidateDelete: func(s *Server, existing *azureARMResource) string {
			// Its identity assignments go with it.
			if len(s.azureHZConfigsForLocked(existing.ID)) > 0 {
				return "Delete the configurations that use this destination first."
			}
			return ""
		},
		ServerActions:  map[string]func(*Server, *azureARMResource, map[string]any) (int, any){"activateidentity": azureHZActivateIdentity},
		IfMatchActions: map[string]bool{"activateidentity": true},
		DeleteStatus:   http.StatusNoContent,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/changeDataCaptureDestinations/identityAssignments", Proxy: true, ETag: true,
		ValidateName: azureHZName("identity assignment"),
		Validate: func(payload map[string]any) string {
			props := azureARMObject(payload["properties"])
			if msg := cmp.Or(azureDataBoxObjects(payload, "", "properties"), azureWorkloadsEnums(props, "properties.", []string{"identityResourceId"}, nil)); msg != "" {
				return msg
			}
			if !azureHZIdentityID.MatchString(azureARMString(props["identityResourceId"])) {
				return "properties.identityResourceId must be a user-assigned managed identity ID."
			}
			return ""
		},
		Reconcile: azureHZAssignmentReconcile,
		ValidateDelete: func(_ *Server, existing *azureARMResource) string {
			if existing.Properties["activationState"] == "Active" {
				return "The identity assignment is active; activate another assignment for the destination before deleting it."
			}
			return ""
		},
		NoPatch: true, DeleteStatus: http.StatusNoContent,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "clusters/changeDataCaptureConfigurations", Proxy: true, ETag: true,
		ValidateName: azureHZName("change data capture configuration"),
		Validate:     azureHZConfigValidate,
		PatchBody:    func(payload map[string]any) (map[string]any, map[string]any, string) { return desiredOnly(payload) },
		ValidatePut:  azureHZNeedsCapability,
		Reconcile:    azureHZConfigReconcile,
		DeleteStatus: http.StatusNoContent,
	})
}

// azureHZDesired checks the properties object and its required desiredState.
func azureHZDesired(payload map[string]any) string {
	if _, ok := payload["properties"].(map[string]any); !ok {
		return "properties is required and must be an object."
	}
	return azureWorkloadsEnums(azureARMObject(payload["properties"]), "properties.", []string{"desiredState"}, map[string][]string{"desiredState": azureHZStates})
}

// azureHZCDCStrip drops the server-owned properties a client sent. rec is
// not yet stored, so it may be changed in place.
func azureHZCDCStrip(rec *azureARMResource) {
	for _, key := range azureHZCDCServerOwned[rec.Type] {
		delete(rec.Properties, key)
	}
}

// azureHZChildrenLocked returns the records of the full type under
// parentID, sorted by ID; the type fixes their depth. Caller holds providerStorageMu.
func (s *Server) azureHZChildrenLocked(parentID, fullType string) []*azureARMResource {
	return s.azureARMInlineChildrenLocked(parentID, azureARMResourceTypes[strings.ToLower(fullType)])
}

// azureHZConfigsForLocked returns the configurations that use the
// destination. Caller holds providerStorageMu.
func (s *Server) azureHZConfigsForLocked(destinationID string) []*azureARMResource {
	var out []*azureARMResource
	for _, config := range s.azureHZChildrenLocked(azureHZClusterOf(destinationID), azureHZCDCConfigType) {
		if strings.EqualFold(azureARMString(config.Properties["destinationResourceId"]), destinationID) {
			out = append(out, config)
		}
	}
	return out
}

// azureHZNeedsCapability rejects destinations and configurations on a
// cluster without the change data capture capability, which the service
// deletes only after them.
func azureHZNeedsCapability(s *Server, _, rec *azureARMResource) (string, string) {
	if s.azureARMResources[azureARMKey(azureHZClusterOf(rec.ID)+"/changeDataCapture/default")] == nil {
		return "ChangeDataCaptureNotConfigured", "Create the cluster's change data capture capability (changeDataCapture/default) first."
	}
	return "", ""
}

// azureHZCDCStatus is the customer-safe status for an effective state.
func azureHZCDCStatus(state string) map[string]any {
	return map[string]any{"code": state, "message": "The resource is " + strings.ToLower(state) + "."}
}

func azureHZCapabilityReconcile(s *Server, _, rec *azureARMResource) (map[string]any, string) {
	azureHZCDCStrip(rec)
	extension := "NotInstalled"
	if rec.Properties["desiredState"] == "Enabled" {
		extension = "Ready"
	}
	state := azureARMString(rec.Properties["desiredState"])
	return azureARMMerge(map[string]any{
		"effectiveState": state, "replicationArchitecture": "ComputeBased",
		"extension": map[string]any{"name": azureHZCDCExtension, "version": "1.0", "state": extension},
		"status":    azureHZCDCStatus(state),
	}, s.azureHZCDCCountsLocked(azureHZClusterOf(rec.ID))), ""
}

// azureHZCDCCountsLocked returns the capability properties derived from
// the rest of the cluster. Caller holds providerStorageMu.
func (s *Server) azureHZCDCCountsLocked(clusterID string) map[string]any {
	endpoint := "Unavailable"
	if len(s.azureHZPoolsLocked(clusterID)) > 0 {
		endpoint = "Ready"
	}
	return map[string]any{
		"destinationCount":   float64(len(s.azureHZChildrenLocked(clusterID, azureHZDestinationType))),
		"configurationCount": float64(len(s.azureHZChildrenLocked(clusterID, azureHZCDCConfigType))),
		"prerequisites":      map[string]any{"engineConfigurationState": "Ready", "readWriteEndpointState": endpoint},
	}
}

// azureHZCDCRefreshLocked brings the counts the capability and each
// destination report up to date after a cluster child changed. Only
// records whose values differ are rewritten, so etags stay put otherwise.
// Caller holds providerStorageMu.
func (s *Server) azureHZCDCRefreshLocked(clusterID string) {
	update := func(rec *azureARMResource, props map[string]any) {
		for key, value := range props {
			if !reflect.DeepEqual(rec.Properties[key], value) {
				s.azureOracleUpdateLocked(rec.ID, props)
				return
			}
		}
	}
	if capability := s.azureARMResources[azureARMKey(clusterID+"/changeDataCapture/default")]; capability != nil {
		update(capability, s.azureHZCDCCountsLocked(clusterID))
	}
	for _, destination := range s.azureHZChildrenLocked(clusterID, azureHZDestinationType) {
		update(destination, map[string]any{"configurationCount": float64(len(s.azureHZConfigsForLocked(destination.ID)))})
	}
}

func azureHZDestinationValidate(payload map[string]any) string {
	if msg := azureHZDesired(payload); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	destination, ok := props["destination"].(map[string]any)
	if !ok {
		return "properties.destination is required and must be an object."
	}
	return cmp.Or(
		azureWorkloadsEnums(destination, "properties.destination.", []string{"destinationType", "protocol", "workspaceId"},
			map[string][]string{"destinationType": {"Fabric"}, "protocol": {"Native"}}),
		azureLabLength(destination, "properties.destination.", "workspaceId", 1, 256),
	)
}

// azureHZDestinationReconcile keeps the destination settings fixed and
// derives the state: an enabled destination waits for an active identity.
func azureHZDestinationReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	azureHZCDCStrip(rec)
	out := map[string]any{"configurationCount": float64(len(s.azureHZConfigsForLocked(rec.ID)))}
	if existing != nil {
		if !reflect.DeepEqual(existing.Properties["destination"], rec.Properties["destination"]) {
			return nil, "properties.destination can't be changed; delete and recreate the destination."
		}
		if active := existing.Properties["activeIdentityAssignmentResourceId"]; active != nil {
			out["activeIdentityAssignmentResourceId"] = active
		}
	}
	state := "Disabled"
	switch {
	case rec.Properties["desiredState"] == "Enabled" && out["activeIdentityAssignmentResourceId"] != nil:
		state = "Enabled"
	case rec.Properties["desiredState"] == "Enabled":
		state = "WaitingForIdentity"
	}
	out["effectiveState"], out["status"] = state, azureHZCDCStatus(state)
	return out, ""
}

// azureHZAssignmentReconcile requires the identity to be attached to the
// cluster and keeps it fixed; validation succeeds at once.
func azureHZAssignmentReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	azureHZCDCStrip(rec)
	id := azureARMString(rec.Properties["identityResourceId"])
	if existing != nil {
		if !strings.EqualFold(id, azureARMString(existing.Properties["identityResourceId"])) {
			return nil, "properties.identityResourceId can't be changed; delete and recreate the identity assignment."
		}
		return azureARMMerge(existing.Properties, nil), ""
	}
	cluster := s.azureHZClusterLocked(azureHZClusterOf(rec.ID))
	if cluster == nil {
		return nil, "The cluster was not found."
	}
	entry, attached := azureARMObject(cluster.Identity["userAssignedIdentities"])[id]
	if !attached {
		return nil, "properties.identityResourceId must exactly match a key in the cluster's identity.userAssignedIdentities."
	}
	return map[string]any{
		"effectiveState": "Assigned", "activationState": "Inactive",
		"principalId": azureARMObject(entry)["principalId"], "tenantId": azureDefaultTenantID,
		"validation": map[string]any{"state": "Succeeded", "validatedAtUtc": azureHZTime(time.Now())},
		"status":     azureHZCDCStatus("Assigned"),
	}, ""
}

// azureHZActivateIdentity makes one of the destination's identity
// assignments the active one, deactivating the previous one.
func azureHZActivateIdentity(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if msg := cmp.Or(
		azureOracleOnly(payload, "", []string{"identityAssignmentResourceId", "drainExistingSessions", azureARMIfMatch}),
		azureWorkloadsEnums(payload, "", []string{"identityAssignmentResourceId"}, nil),
		azureOracleBools(payload, "", "drainExistingSessions"),
	); msg != "" {
		return azureOracleBad(msg)
	}
	target := azureARMString(payload["identityAssignmentResourceId"])
	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()
	destination := s.azureARMResources[azureARMKey(rec.ID)]
	assignment := s.azureARMResources[azureARMKey(target)]
	switch {
	case destination == nil:
		return http.StatusNotFound, azureARMErrorBody("ResourceNotFound", "The destination was not found.")
	case azureARMIfMatchStale(destination, payload):
		return azureARMPreconditionFailed()
	case assignment == nil || !strings.EqualFold(assignment.Type, azureHZAssignmentType) || !strings.HasPrefix(azureARMKey(target), azureARMKey(rec.ID+"/")):
		return azureOracleBad("identityAssignmentResourceId must be an identity assignment under this destination.")
	}
	result := map[string]any{"status": "Succeeded", "targetIdentityAssignmentResourceId": assignment.ID}
	previous := azureARMString(destination.Properties["activeIdentityAssignmentResourceId"])
	if strings.EqualFold(previous, assignment.ID) {
		return http.StatusOK, result
	}
	if previous != "" {
		result["previousIdentityAssignmentResourceId"] = previous
		s.azureOracleUpdateLocked(previous, map[string]any{"activationState": "Inactive"})
	}
	s.azureOracleUpdateLocked(assignment.ID, map[string]any{"activationState": "Active"})
	state := "Disabled"
	if destination.Properties["desiredState"] == "Enabled" {
		state = "Enabled"
	}
	s.azureOracleUpdateLocked(destination.ID, map[string]any{
		"activeIdentityAssignmentResourceId": assignment.ID, "effectiveState": state, "status": azureHZCDCStatus(state),
	})
	return http.StatusOK, result
}

func azureHZConfigValidate(payload map[string]any) string {
	if msg := azureHZDesired(payload); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	return cmp.Or(
		azureWorkloadsEnums(props, "properties.", []string{"databaseName", "usageProfile", "destinationResourceId"},
			map[string][]string{"usageProfile": {"Mirroring"}}),
		azureLabLength(props, "properties.", "databaseName", 1, 63),
		azureLabLength(props, "properties.", "destinationResourceId", 1, 1024),
	)
}

// azureHZConfigCreateOnly are the configuration properties fixed at creation.
var azureHZConfigCreateOnly = []string{"databaseName", "usageProfile", "destinationResourceId"}

// azureHZConfigReconcile keeps the enrollment fixed and requires an
// enabled destination under the same cluster.
func azureHZConfigReconcile(s *Server, existing, rec *azureARMResource) (map[string]any, string) {
	azureHZCDCStrip(rec)
	cluster := azureHZClusterOf(rec.ID)
	out := map[string]any{
		"logicalStreamId": azureHZGUID(rec.ID + "#stream"),
		"extension":       map[string]any{"name": azureHZCDCExtension, "version": "1.0", "state": "Ready"},
		"topology":        map[string]any{"state": "Active", "generation": 1.0, "activeClusterResourceId": cluster},
	}
	if existing != nil {
		for _, key := range azureHZConfigCreateOnly {
			stored, sent := existing.Properties[key], rec.Properties[key]
			if !reflect.DeepEqual(stored, sent) && (key != "destinationResourceId" || !strings.EqualFold(azureARMString(stored), azureARMString(sent))) {
				return nil, "properties." + key + " can't be changed after the configuration is created."
			}
		}
	} else {
		destinationID := azureARMString(rec.Properties["destinationResourceId"])
		destination := s.azureARMResources[azureARMKey(destinationID)]
		switch {
		case destination == nil || !strings.EqualFold(destination.Type, azureHZDestinationType) || !strings.EqualFold(azureHZClusterOf(destinationID), cluster):
			return nil, "properties.destinationResourceId must be a change data capture destination under the same cluster."
		case destination.Properties["effectiveState"] != "Enabled":
			return nil, "The destination " + destination.Name + " isn't enabled: enable it and activate an identity assignment first."
		}
	}
	state := azureARMString(rec.Properties["desiredState"])
	replication := "NotApplicable"
	if state == "Enabled" {
		replication = "NotConnected"
	}
	out["effectiveState"], out["status"] = state, azureHZCDCStatus(state)
	out["replication"] = map[string]any{"state": replication}
	return out, ""
}

// azureHZAssignedIdentityLocked returns a message when the cluster
// identity a write leaves would drop one an identity assignment uses.
// Caller holds providerStorageMu.
func (s *Server) azureHZAssignedIdentityLocked(clusterID string, identity map[string]any) string {
	kept := azureARMObject(identity["userAssignedIdentities"])
	for _, destination := range s.azureHZChildrenLocked(clusterID, azureHZDestinationType) {
		for _, assignment := range s.azureHZChildrenLocked(destination.ID, azureHZAssignmentType) {
			id := azureARMString(assignment.Properties["identityResourceId"])
			if _, ok := kept[id]; !ok {
				return "identity.userAssignedIdentities must keep " + id + ": change data capture identity assignment " + assignment.ID + " uses it."
			}
		}
	}
	return ""
}
