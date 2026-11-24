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
	azureDataProtection20260301APIVersion = "2026-03-01"
	azureDPPNamespace                     = "Microsoft.DataProtection"
	azureDPPMaxJobs                       = 100  // kept per vault, newest first
	azureDPPMaxDeleted                    = 1000 // soft-deleted vaults kept
)

// Azure Backup vaults (Microsoft.DataProtection 2026-03-01) on the shared
// ARM handler: backup vaults, policies and instances with ad hoc backups
// and protection changes, backup jobs, job exports, operation status,
// soft-deleted vaults, feature and name checks and the operations list.
// Backups copy nothing and finish at once.
var (
	azureDPPRedundancy     = []string{"LocallyRedundant", "GeoRedundant", "ZoneRedundant"}
	azureDPPDatastoreTypes = []string{"VaultStore", "ArchiveStore", "OperationalStore"}
	// azureDPPRouteRE matches the routes served by the router below: job
	// export, export results and operation status and results.
	azureDPPRouteRE = regexp.MustCompile(`(?i)^/subscriptions/([^/]+)(?:/resourceGroups/([^/]+))?/providers/Microsoft\.DataProtection/(?:locations/([^/]+)/(operationStatus|operationResults)/([^/]+)|backupVaults/([^/]+)/(exportBackupJobs|backupJobs/operations/([^/]+)|operationStatus/([^/]+)))$`)
)

type azureDPPState struct {
	mu      sync.Mutex
	jobs    map[string][]map[string]any // by vault key
	deleted []map[string]any
	ops     azureDMSOperations
}

func (s *Server) azureDPP() *azureDPPState {
	return azureDMSStateFor(s, "dataprotection", func() *azureDPPState { return &azureDPPState{jobs: map[string][]map[string]any{}} })
}

func init() {
	ns := azureDPPNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "backupVaults",
		Validate: azureDPPVaultValidate,
		ReadOnlyProperties: func(*azureARMResource) map[string]any {
			return map[string]any{
				"resourceMoveState": "Unknown", "isVaultProtectedByResourceGuard": false, "secureScore": "None",
				"bcdrSecurityLevel": "Poor", "replicatedRegions": []any{},
			}
		},
		ValidateDelete: func(s *Server, existing *azureARMResource) string {
			if len(s.azureDPPChildrenLocked(existing.ID, ns+"/backupVaults/backupInstances")) > 0 {
				return "The vault has backup instances; delete them first."
			}
			return ""
		},
		Deleted: func(s *Server, removed *azureARMResource) { s.azureDPP().recordDeleted(removed) },
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "backupVaults/backupPolicies", Proxy: true, NoPatch: true,
		Validate: azureDPPPolicyValidate,
		ValidateDelete: func(s *Server, existing *azureARMResource) string {
			for _, bi := range s.azureDPPChildrenLocked(existing.ID[:strings.LastIndex(strings.ToLower(existing.ID), "/backuppolicies/")], ns+"/backupVaults/backupInstances") {
				if strings.EqualFold(azureARMString(azureARMObject(bi.Properties["policyInfo"])["policyId"]), existing.ID) {
					return "The policy protects backup instance '" + bi.Name + "'."
				}
			}
			return ""
		},
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "backupVaults/backupInstances", Proxy: true, NoPatch: true,
		Validate:        azureDPPInstanceValidate,
		Reconcile:       azureDPPInstanceReconcile,
		StateProperties: map[string]any{"currentProtectionState": "ProtectionConfigured"},
		ReadOnlyProperties: func(rec *azureARMResource) map[string]any {
			return map[string]any{"protectionStatus": map[string]any{"status": "ProtectionConfigured"}}
		},
		Transitions: map[string]map[string]any{
			"stopprotection":   {"currentProtectionState": "ProtectionStopped"},
			"suspendbackups":   {"currentProtectionState": "BackupsSuspended"},
			"resumeprotection": {"currentProtectionState": "ProtectionConfigured"},
			"resumebackups":    {"currentProtectionState": "ProtectionConfigured"},
		},
		ServerActions: map[string]func(*Server, *azureARMResource, map[string]any) (int, any){"backup": azureDPPAdhocBackup},
	})
	registerAzureARMView(ns, "backupVaults/backupJobs", func(s *Server, scope string) []map[string]any {
		st := s.azureDPP()
		st.mu.Lock()
		defer st.mu.Unlock()
		out := make([]map[string]any, 0, len(st.jobs[azureARMKey(scope)]))
		for _, job := range st.jobs[azureARMKey(scope)] {
			out = append(out, azureARMMerge(job, nil))
		}
		return out
	})
	registerAzureARMView(ns, "locations/deletedVaults", func(s *Server, scope string) []map[string]any {
		st := s.azureDPP()
		st.mu.Lock()
		defer st.mu.Unlock()
		var out []map[string]any
		for _, vault := range st.deleted {
			if strings.HasPrefix(azureARMKey(azureARMString(vault["id"])), azureARMKey(scope+"/")) {
				out = append(out, azureARMMerge(vault, nil))
			}
		}
		return out
	})
	registerAzureARMLocationAction(ns, "checkFeatureSupport", func(_ *Server, _, _ string, payload map[string]any) (int, any) {
		if azureARMString(payload["objectType"]) != "FeatureValidationRequest" {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "objectType must be FeatureValidationRequest.")
		}
		featureType := azureARMString(payload["featureType"])
		if featureType != "DataSourceType" && featureType != "Invalid" {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "featureType must be DataSourceType.")
		}
		return http.StatusOK, map[string]any{
			"objectType": "FeatureValidationResponse", "featureType": featureType,
			"features": []any{map[string]any{"featureName": azureARMString(payload["featureName"]), "supportStatus": "GenerallyAvailable", "exposureControlledFeatures": []any{}}},
		}
	})
	registerAzureARMGroupAction(ns, "checkNameAvailability", func(s *Server, sub, rg string, payload map[string]any) (int, any) {
		if !strings.EqualFold(azureARMString(payload["type"]), ns+"/backupVaults") {
			return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "type must be Microsoft.DataProtection/BackupVaults.")
		}
		return s.azureDMSCheckName("/subscriptions/"+sub+"/resourceGroups/"+rg, ns+"/backupVaults", payload)
	})
	azureDMSOperationsView(ns, "Microsoft Data Protection",
		[2]string{"backupVaults", "backup vaults"}, [2]string{"backupVaults/backupPolicies", "backup policies"},
		[2]string{"backupVaults/backupInstances", "backup instances"})
}

// azureDPPChildrenLocked returns records of fullType under parentID.
// Caller holds providerStorageMu.
func (s *Server) azureDPPChildrenLocked(parentID, fullType string) []*azureARMResource {
	prefix := azureARMKey(parentID + "/")
	return s.azureARMResourcesMatchingLocked(func(rec *azureARMResource) bool {
		return strings.EqualFold(rec.Type, fullType) && strings.HasPrefix(azureARMKey(rec.ID), prefix)
	})
}

func azureDPPVaultValidate(payload map[string]any) string {
	settings := azureARMSlice(azureARMObject(payload["properties"])["storageSettings"])
	if len(settings) == 0 || len(settings) > 3 {
		return "properties.storageSettings needs one to three entries."
	}
	for _, raw := range settings {
		setting := azureARMObject(raw)
		if !slices.Contains(azureDPPRedundancy, azureARMString(setting["type"])) {
			return "storageSettings type must be one of " + strings.Join(azureDPPRedundancy, ", ") + "."
		}
		if !slices.Contains(azureDPPDatastoreTypes, azureARMString(setting["datastoreType"])) {
			return "storageSettings datastoreType must be one of " + strings.Join(azureDPPDatastoreTypes, ", ") + "."
		}
	}
	if identity := azureARMObject(payload["identity"]); len(identity) > 0 && !slices.Contains([]string{"None", "SystemAssigned", "UserAssigned", "SystemAssigned,UserAssigned"}, azureARMString(identity["type"])) {
		return "identity.type must be None, SystemAssigned, UserAssigned or SystemAssigned,UserAssigned."
	}
	return ""
}

func azureDPPPolicyValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	if azureARMString(props["objectType"]) != "BackupPolicy" {
		return "properties.objectType must be BackupPolicy."
	}
	types := azureARMSlice(props["datasourceTypes"])
	if len(types) == 0 {
		return "properties.datasourceTypes needs at least one datasource type."
	}
	for _, t := range types {
		if strings.TrimSpace(azureARMString(t)) == "" {
			return "properties.datasourceTypes entries must be resource types."
		}
	}
	if raw, present := props["policyRules"]; present {
		if _, ok := raw.([]any); !ok {
			return "properties.policyRules must be an array."
		}
	}
	return ""
}

func azureDPPInstanceValidate(payload map[string]any) string {
	props := azureARMObject(payload["properties"])
	source := azureARMObject(props["dataSourceInfo"])
	switch {
	case azureARMString(props["objectType"]) != "BackupInstance":
		return "properties.objectType must be BackupInstance."
	case azureARMString(source["datasourceType"]) == "" || azureARMString(source["resourceID"]) == "":
		return "properties.dataSourceInfo.datasourceType and resourceID are required."
	case azureARMString(azureARMObject(props["policyInfo"])["policyId"]) == "":
		return "properties.policyInfo.policyId is required."
	}
	return ""
}

// azureDPPInstanceReconcile requires the policy to be in the same vault and
// to cover the datasource type.
func azureDPPInstanceReconcile(s *Server, _, rec *azureARMResource) (map[string]any, string) {
	policyID := azureARMString(azureARMObject(rec.Properties["policyInfo"])["policyId"])
	vaultID := rec.ID[:strings.LastIndex(strings.ToLower(rec.ID), "/backupinstances/")]
	policy := s.azureARMResources[azureARMKey(policyID)]
	if policy == nil || !strings.EqualFold(policy.Type, azureDPPNamespace+"/backupVaults/backupPolicies") || !strings.HasPrefix(azureARMKey(policyID), azureARMKey(vaultID+"/")) {
		return nil, "properties.policyInfo.policyId must name a backup policy in this vault."
	}
	datasourceType := azureARMString(azureARMObject(rec.Properties["dataSourceInfo"])["datasourceType"])
	if !slices.ContainsFunc(azureARMSlice(policy.Properties["datasourceTypes"]), func(t any) bool { return strings.EqualFold(azureARMString(t), datasourceType) }) {
		return nil, "The policy does not cover datasource type '" + datasourceType + "'."
	}
	return nil, ""
}

// azureDPPAdhocBackup records a finished on-demand backup job.
func azureDPPAdhocBackup(s *Server, rec *azureARMResource, payload map[string]any) (int, any) {
	if azureARMString(azureARMObject(payload["backupRuleOptions"])["ruleName"]) == "" {
		return http.StatusBadRequest, azureARMErrorBody("InvalidRequestContent", "backupRuleOptions.ruleName is required.")
	}
	if rec.Properties["currentProtectionState"] != "ProtectionConfigured" {
		return http.StatusConflict, azureARMErrorBody("UserErrorBackupInstanceNotProtected", "Backups need the instance's protection to be configured.")
	}
	vaultID := rec.ID[:strings.LastIndex(strings.ToLower(rec.ID), "/backupinstances/")]
	source := azureARMObject(rec.Properties["dataSourceInfo"])
	policyID := azureARMString(azureARMObject(rec.Properties["policyInfo"])["policyId"])
	jobID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	job := map[string]any{
		"id": vaultID + "/backupJobs/" + jobID, "name": jobID, "type": azureDPPNamespace + "/backupVaults/backupJobs",
		"properties": map[string]any{
			"activityID": uuid.NewString(), "backupInstanceFriendlyName": azureARMString(rec.Properties["friendlyName"]), "backupInstanceId": rec.ID,
			"dataSourceId": source["resourceID"], "dataSourceLocation": source["resourceLocation"], "dataSourceName": source["resourceName"],
			"dataSourceType": source["datasourceType"], "isUserTriggered": true, "operation": "OnDemandBackup", "operationCategory": "Backup",
			"policyId": policyID, "policyName": policyID[strings.LastIndex(policyID, "/")+1:], "progressEnabled": false,
			"sourceResourceGroup": rec.ResourceGroup, "sourceSubscriptionID": rec.SubscriptionID, "subscriptionId": rec.SubscriptionID,
			"startTime": now, "endTime": now, "duration": "PT0S", "status": "Completed", "supportedActions": []any{},
			"vaultName": vaultID[strings.LastIndex(vaultID, "/")+1:],
		},
	}
	st := s.azureDPP()
	st.mu.Lock()
	defer st.mu.Unlock()
	key := azureARMKey(vaultID)
	jobs := append([]map[string]any{job}, st.jobs[key]...)
	st.jobs[key] = jobs[:min(len(jobs), azureDPPMaxJobs)]
	return http.StatusOK, map[string]any{"objectType": "OperationJobExtendedInfo", "jobId": jobID}
}

// recordDeleted keeps a soft-deleted copy of a vault and drops its jobs.
// It runs under providerStorageMu (lock order: store, then st.mu).
func (st *azureDPPState) recordDeleted(vault *azureARMResource) {
	now := time.Now().UTC()
	name := uuid.NewString()
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.jobs, azureARMKey(vault.ID))
	if len(st.deleted) >= azureDPPMaxDeleted {
		st.deleted = st.deleted[1:]
	}
	st.deleted = append(st.deleted, map[string]any{
		"id":   "/subscriptions/" + vault.SubscriptionID + "/providers/" + azureDPPNamespace + "/locations/" + vault.Location + "/deletedVaults/" + name,
		"name": name, "type": azureDPPNamespace + "/locations/deletedVaults",
		"properties": map[string]any{
			"originalBackupVaultId": vault.ID, "originalBackupVaultName": vault.Name,
			"originalBackupVaultResourcePath": vault.ID, "provisioningState": "Succeeded",
			"resourceDeletionInfo": map[string]any{"deletionTime": now.Format(time.RFC3339), "scheduledPurgeTime": now.Add(14 * 24 * time.Hour).Format(time.RFC3339)},
			"storageSettings":      vault.Properties["storageSettings"],
		},
	})
}

// handleAzureDataProtection20260301Router serves job exports and operation
// status and results, which the shared ARM handler has no route for.
func (s *Server) handleAzureDataProtection20260301Router(w http.ResponseWriter, r *http.Request) bool {
	m := azureDPPRouteRE.FindStringSubmatch(strings.TrimPrefix(rawRequestPath(r), "/azure"))
	if m == nil {
		return azureDMSUnknownRoute(w, r, azureDPPNamespace)
	}
	sub, rg, location, kind, opID, vault, vaultRoute := m[1], m[2], m[3], strings.ToLower(m[4]), m[5], m[6], strings.ToLower(m[7])
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	wantMethod := http.MethodGet
	if vaultRoute == "exportbackupjobs" {
		wantMethod = http.MethodPost
	}
	if r.Method != wantMethod {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}
	st := s.azureDPP()
	if vault == "" {
		op, ok := st.ops.get(opID)
		if !ok || !strings.EqualFold(op.Location, location) || !strings.HasPrefix(azureARMKey(op.Scope), azureARMKey(strings.TrimSuffix("/subscriptions/"+sub+"/resourceGroups/"+rg, "/resourceGroups/")+"/")) {
			azureARMNotFound(w, "OperationNotFound", strings.TrimPrefix(rawRequestPath(r), "/azure"), "")
			return true
		}
		if kind == "operationresults" {
			respondJSON(w, http.StatusOK, azureARMMerge(op.Result, map[string]any{"objectType": "OperationJobExtendedInfo"}))
			return true
		}
		respondJSON(w, http.StatusOK, op.status(strings.TrimPrefix(rawRequestPath(r), "/azure")))
		return true
	}
	vaultID := "/subscriptions/" + sub + "/resourceGroups/" + rg + "/providers/" + azureDPPNamespace + "/backupVaults/" + vault
	s.providerStorageMu.Lock()
	stored := s.azureARMResources[azureARMKey(vaultID)]
	s.providerStorageMu.Unlock()
	if stored == nil {
		azureARMNotFound(w, "ResourceNotFound", vaultID, "")
		return true
	}
	switch {
	case vaultRoute == "exportbackupjobs":
		key := strings.ToLower(stored.Name)
		op := st.ops.add(stored.ID, "exportBackupJobs", stored.Location, map[string]any{
			"blobUrl": "https://stackyarddpp.blob.core.windows.net/exports/" + key + ".json", "blobSasKey": "sv=2022-11-02&sig=stackyard",
			"excelFileBlobUrl": "https://stackyarddpp.blob.core.windows.net/exports/" + key + ".xlsx", "excelFileBlobSasKey": "sv=2022-11-02&sig=stackyard",
		})
		w.Header().Set("Location", "/subscriptions/"+stored.SubscriptionID+"/resourceGroups/"+stored.ResourceGroup+"/providers/"+azureDPPNamespace+"/backupVaults/"+stored.Name+"/backupJobs/operations/"+op.ID+"?api-version="+azureDataProtection20260301APIVersion)
		w.Header().Set("Azure-AsyncOperation", "/subscriptions/"+stored.SubscriptionID+"/providers/"+azureDPPNamespace+"/locations/"+stored.Location+"/operationStatus/"+op.ID+"?api-version="+azureDataProtection20260301APIVersion)
		w.WriteHeader(http.StatusAccepted)
		return true
	case strings.HasPrefix(vaultRoute, "backupjobs/operations/"):
		opID = m[8]
	default:
		opID = m[9]
	}
	op, ok := st.ops.get(opID)
	if !ok || !strings.EqualFold(op.Scope, stored.ID) {
		azureARMNotFound(w, "OperationNotFound", vaultID+"/"+vaultRoute, "")
		return true
	}
	if strings.HasPrefix(vaultRoute, "backupjobs/operations/") {
		respondJSON(w, http.StatusOK, op.Result)
		return true
	}
	respondJSON(w, http.StatusOK, op.status(strings.TrimPrefix(rawRequestPath(r), "/azure")))
	return true
}
