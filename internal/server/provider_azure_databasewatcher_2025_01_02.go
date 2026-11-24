package server

import (
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Azure Database Watcher (Microsoft.DatabaseWatcher 2025-01-02): watchers
// with start and stop, their SQL targets, alert rule resources and shared
// private link resources, served by the shared ARM handler, plus health
// validations and the operations list. No monitoring data is collected:
// health validations succeed with no issues as soon as they start.

const (
	azureDatabaseWatcher20250102APIVersion = "2025-01-02"
	azureDBWNamespace                      = "Microsoft.DatabaseWatcher"
	azureDBWWatcherType                    = azureDBWNamespace + "/watchers"
	azureDBWValidationType                 = azureDBWWatcherType + "/healthValidations"
	azureDBWMaxValidations                 = 100 // per watcher; the oldest go first
)

var azureDBWChildName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,62}[a-zA-Z0-9_]?$`)

func init() {
	ns := azureDBWNamespace
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "watchers",
		ValidateName:    azureDBWName,
		Validate:        azureDBWWatcherValidate,
		StateProperties: map[string]any{"status": "Stopped"},
		Transitions: map[string]map[string]any{
			"start": {"status": "Running"},
			"stop":  {"status": "Stopped"},
		},
		DeleteStatus: http.StatusNoContent,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "watchers/targets", Proxy: true,
		ValidateName: azureDBWName, Validate: azureDBWTargetValidate,
		NoPatch: true,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "watchers/alertRuleResources", Proxy: true,
		ValidateName: azureDBWName,
		Validate: azureDataPlatAll(
			azureDataPlatRequired("alertRuleResourceId", "createdWithProperties", "creationTime", "alertRuleTemplateId", "alertRuleTemplateVersion"),
			azureDataPlatEnum("createdWithProperties", "CreatedWithActionGroup", "None"),
		),
		NoPatch: true,
	})
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "watchers/sharedPrivateLinkResources", Proxy: true,
		ValidateName:    azureDBWName,
		Validate:        azureDataPlatRequired("privateLinkResourceId", "groupId", "requestMessage"),
		StateProperties: map[string]any{"status": "Pending"},
		NoPatch:         true, DeleteStatus: http.StatusNoContent,
	})
	// Health validations exist once started; see startValidation below.
	registerAzureARMResourceType(azureARMResourceType{
		Namespace: ns, Type: "watchers/healthValidations", Proxy: true,
		NoPut: true, NoPatch: true, NoDelete: true,
	})
	registerAzureARMView(ns, "operations", azureDataPlatOperations(ns, "Microsoft Database Watcher",
		"watchers/read", "watchers/write", "watchers/delete", "watchers/start/action", "watchers/stop/action",
		"watchers/targets/read", "watchers/targets/write", "watchers/targets/delete",
		"watchers/alertRuleResources/read", "watchers/alertRuleResources/write", "watchers/alertRuleResources/delete",
		"watchers/healthValidations/read", "watchers/healthValidations/startValidation/action",
		"watchers/sharedPrivateLinkResources/read", "watchers/sharedPrivateLinkResources/write", "watchers/sharedPrivateLinkResources/delete"))
}

func azureDBWName(name string) string {
	if len(name) > 64 || !azureDBWChildName.MatchString(name) {
		return "The name '" + name + "' must be 1 to 64 letters, digits, periods, underscores and hyphens, starting with a letter or digit."
	}
	return ""
}

func azureDBWWatcherValidate(payload map[string]any) string {
	if identity := azureARMObject(payload["identity"]); len(identity) > 0 {
		switch azureARMString(identity["type"]) {
		case "None", "SystemAssigned", "UserAssigned":
		default:
			return "identity.type must be None, SystemAssigned or UserAssigned."
		}
	}
	if azureARMObject(payload["properties"])["datastore"] == nil {
		return ""
	}
	return azureDataPlatAll(
		azureDataPlatRequired("datastore.kustoClusterUri", "datastore.kustoDataIngestionUri", "datastore.kustoDatabaseName", "datastore.kustoManagementUrl", "datastore.kustoOfferingType"),
		azureDataPlatEnum("datastore.kustoOfferingType", "adx", "free", "fabric"),
	)(payload)
}

// azureDBWTargetValidate checks a target, whose properties depend on its
// targetType discriminator.
func azureDBWTargetValidate(payload map[string]any) string {
	if msg := azureDataPlatAll(
		azureDataPlatRequired("targetType", "targetAuthenticationType", "connectionServerName"),
		azureDataPlatEnum("targetType", "SqlDb", "SqlEp", "SqlMi"),
		azureDataPlatEnum("targetAuthenticationType", "Aad", "Sql"),
	)(payload); msg != "" {
		return msg
	}
	props := azureARMObject(payload["properties"])
	required := map[string][]string{
		"SqlDb": {"sqlDbResourceId"},
		"SqlEp": {"sqlEpResourceId", "anchorDatabaseResourceId"},
		"SqlMi": {"sqlMiResourceId"},
	}[azureARMString(props["targetType"])]
	if azureARMString(props["targetAuthenticationType"]) == "Sql" {
		required = append(required, "targetVault.akvResourceId", "targetVault.akvTargetUser", "targetVault.akvTargetPassword")
	}
	return azureDataPlatRequired(required...)(payload)
}

// handleAzureDatabaseWatcher20250102Router serves POST .../watchers/{w}/
// healthValidations/{name}/startValidation, which creates the validation
// when it doesn't exist yet, so the shared handler's actions (which need a
// stored resource) can't serve it.
func (s *Server) handleAzureDatabaseWatcher20250102Router(w http.ResponseWriter, r *http.Request) bool {
	seg := splitPathSegments(strings.TrimPrefix(rawRequestPath(r), "/azure/"))
	if len(seg) != 11 || !strings.EqualFold(seg[0], "subscriptions") || !strings.EqualFold(seg[2], "resourceGroups") ||
		!strings.EqualFold(seg[5], azureDBWNamespace) || !strings.EqualFold(seg[6], "watchers") ||
		!strings.EqualFold(seg[8], "healthValidations") || !strings.EqualFold(seg[10], "startValidation") {
		return false
	}
	if r.Method != http.MethodPost {
		azureARMError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed")
		return true
	}
	if hasAzureInvalidAPIVersion(r) {
		azureARMError(w, http.StatusBadRequest, "InvalidApiVersionParameter", "invalid api-version query parameter")
		return true
	}
	name := seg[9]
	if msg := azureDBWName(name); msg != "" {
		azureARMError(w, http.StatusBadRequest, "InvalidResourceName", msg)
		return true
	}
	watcherID := "/subscriptions/" + seg[1] + "/resourceGroups/" + seg[3] + "/providers/" + azureDBWNamespace + "/watchers/" + seg[7]
	now := time.Now().UTC().Format(time.RFC3339)
	s.providerStorageMu.Lock()
	watcher := s.azureDataPlatStoredLocked(watcherID, azureDBWWatcherType)
	if watcher == nil {
		s.providerStorageMu.Unlock()
		azureARMNotFound(w, "ParentResourceNotFound", watcherID+"/healthValidations/"+name, watcherID)
		return true
	}
	rec := s.azureDataPlatPutChildLocked(watcher, azureDBWValidationType, "healthValidations", name, map[string]any{
		"status": "Succeeded", "startTime": now, "endTime": now, "issues": []any{}, "provisioningState": "Succeeded",
	}, azureDBWMaxValidations)
	body := rec.fixture()
	s.providerStorageMu.Unlock()
	respondJSON(w, http.StatusOK, body)
	return true
}
