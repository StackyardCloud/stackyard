package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var gcpSecurityCenterManagementReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPSecurityCenterManagementRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_securitycentermanagement(w, r) {
		return true
	}

	path := normalizeGCPSecurityCenterManagementPath(rawRequestPath(r))
	if isGCPSecurityCenterManagementLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPSecurityCenterManagementListLocations(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterManagementGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPSecurityCenterManagementPath(path, hasGCPSecurityCenterManagementHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPSecurityCenterManagementListEffectiveSHAModules(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterManagementGetEffectiveSHAModule(w, path) {
			return true
		}
		if handleGCPSecurityCenterManagementListSHAModules(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterManagementListDescendantSHAModules(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterManagementGetSHAModule(w, path) {
			return true
		}
		if handleGCPSecurityCenterManagementListEffectiveETDModules(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterManagementGetEffectiveETDModule(w, path) {
			return true
		}
		if handleGCPSecurityCenterManagementListETDModules(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterManagementListDescendantETDModules(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterManagementGetETDModule(w, path) {
			return true
		}
		if handleGCPSecurityCenterManagementGetService(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterManagementListServices(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPSecurityCenterManagementCreateSHAModule(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterManagementSimulateSHAModule(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterManagementCreateETDModule(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterManagementValidateETDModule(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPSecurityCenterManagementUpdateSHAModule(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterManagementUpdateETDModule(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterManagementUpdateService(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPSecurityCenterManagementDeleteSHAModule(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterManagementDeleteETDModule(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPSecurityCenterManagementPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPSecurityCenterManagementHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "securitycentermanagement",
		"securitycentermanagement-apiv1",
		"securitycentermanagement_apiv1",
		"security-center-management",
		"security_center_management",
		"scc-management",
		"gcp-security-center-management":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-securitycentermanagement-apiv1") || strings.Contains(ua, "cloud.google.com/go/securitycentermanagement")
}

func isGCPSecurityCenterManagementLocationRequest(r *http.Request, path string) bool {
	if !hasGCPSecurityCenterManagementHint(r) {
		return false
	}
	_, _, _, _, ok := parseGCPSecurityCenterManagementLocationPath(path)
	return ok
}

func isGCPSecurityCenterManagementPath(path string, includeAmbiguous bool) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.securitycentermanagement.v1.SecurityCenterManagement/") {
		return true
	}
	_, _, _, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || len(tail) == 0 {
		return false
	}
	first := strings.TrimSpace(tail[0])
	if first == "securityCenterServices" ||
		strings.HasPrefix(first, "securityHealthAnalyticsCustomModules") ||
		strings.HasPrefix(first, "effectiveSecurityHealthAnalyticsCustomModules") ||
		strings.HasPrefix(first, "eventThreatDetectionCustomModules") ||
		strings.HasPrefix(first, "effectiveEventThreatDetectionCustomModules") {
		return true
	}
	return includeAmbiguous
}

func handleGCPSecurityCenterManagementListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, _, list, ok := parseGCPSecurityCenterManagementLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterManagementPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterManagementLocation(scope, scopeID, "us-central1"),
		gcpSecurityCenterManagementLocation(scope, scopeID, "global"),
	}
	return respondGCPSecurityCenterManagementList(w, "locations", items, pageSize, start, path)
}

func handleGCPSecurityCenterManagementGetLocation(w http.ResponseWriter, path string) bool {
	scope, scopeID, location, list, ok := parseGCPSecurityCenterManagementLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterManagementLocation(scope, scopeID, location))
	return true
}

func handleGCPSecurityCenterManagementListEffectiveSHAModules(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || !isGCPSecurityCenterManagementEffectiveSHAModuleCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterManagementPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterManagementEffectiveSHAModule(scope, scopeID, location, "effective-sha-module-1"),
		gcpSecurityCenterManagementEffectiveSHAModule(scope, scopeID, location, "effective-sha-module-2"),
	}
	return respondGCPSecurityCenterManagementList(w, "effectiveSecurityHealthAnalyticsCustomModules", items, pageSize, start, path)
}

func handleGCPSecurityCenterManagementGetEffectiveSHAModule(w http.ResponseWriter, path string) bool {
	scope, scopeID, location, moduleID, ok := parseGCPSecurityCenterManagementEffectiveSHAModulePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterManagementEffectiveSHAModule(scope, scopeID, location, moduleID))
	return true
}

func handleGCPSecurityCenterManagementListSHAModules(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || !isGCPSecurityCenterManagementSHAModuleCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterManagementPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterManagementSHAModule(scope, scopeID, location, "sha-module-1"),
		gcpSecurityCenterManagementSHAModule(scope, scopeID, location, "sha-module-2"),
	}
	return respondGCPSecurityCenterManagementList(w, "securityHealthAnalyticsCustomModules", items, pageSize, start, path)
}

func handleGCPSecurityCenterManagementListDescendantSHAModules(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || !isGCPSecurityCenterManagementListDescendantSHATail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterManagementPagination(w, r, path)
	if !valid {
		return true
	}
	item := gcpSecurityCenterManagementSHAModule(scope, scopeID, location, "descendant-sha-module-1")
	item["ancestorModule"] = gcpSecurityCenterManagementSHAModuleName(scope, scopeID, location, "sha-module-1")
	items := []map[string]any{item}
	return respondGCPSecurityCenterManagementList(w, "securityHealthAnalyticsCustomModules", items, pageSize, start, path)
}

func handleGCPSecurityCenterManagementGetSHAModule(w http.ResponseWriter, path string) bool {
	scope, scopeID, location, moduleID, ok := parseGCPSecurityCenterManagementSHAModulePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterManagementSHAModule(scope, scopeID, location, moduleID))
	return true
}

func handleGCPSecurityCenterManagementCreateSHAModule(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || !isGCPSecurityCenterManagementSHAModuleCollectionTail(tail) {
		return false
	}
	if _, valid := parseGCPSecurityCenterManagementOptionalBoolQuery(w, path, r.URL.Query().Get("validateOnly"), "validateOnly"); !valid {
		return true
	}
	body, valid := decodeGCPSecurityCenterManagementJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	module := gcpSecurityCenterManagementBodyMap(body, "securityHealthAnalyticsCustomModule")
	if len(module) == 0 {
		module = body
	}
	if len(module) == 0 {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "securityHealthAnalyticsCustomModule is required")
		return true
	}
	if !gcpSecurityCenterManagementHasMap(module, "customConfig") {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "securityHealthAnalyticsCustomModule.customConfig is required")
		return true
	}

	moduleID := "sha-module-created"
	if rawName := strings.TrimSpace(gcpSecurityCenterManagementString(module, "name")); rawName != "" {
		nameScope, nameScopeID, nameLocation, nameID, nameOK := parseGCPSecurityCenterManagementSHAModuleName(rawName)
		if !nameOK {
			respondGCPSecurityCenterManagementInvalidArgument(w, path, "securityHealthAnalyticsCustomModule.name is invalid")
			return true
		}
		if nameScope != scope || nameScopeID != scopeID || nameLocation != location {
			respondGCPSecurityCenterManagementInvalidArgument(w, path, "securityHealthAnalyticsCustomModule.name must match parent")
			return true
		}
		moduleID = nameID
	}

	resp := gcpSecurityCenterManagementSHAModule(scope, scopeID, location, moduleID)
	applyGCPSecurityCenterManagementSHAOverrides(resp, module)
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPSecurityCenterManagementUpdateSHAModule(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, moduleID, ok := parseGCPSecurityCenterManagementSHAModulePath(path)
	if !ok {
		return false
	}
	updateMask, valid := parseGCPSecurityCenterManagementUpdateMask(w, path, r.URL.Query().Get("updateMask"), []string{"customConfig", "enablementState", "displayName"})
	if !valid {
		return true
	}
	if len(updateMask) == 0 {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "updateMask is required")
		return true
	}
	if _, valid := parseGCPSecurityCenterManagementOptionalBoolQuery(w, path, r.URL.Query().Get("validateOnly"), "validateOnly"); !valid {
		return true
	}
	body, valid := decodeGCPSecurityCenterManagementJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	module := gcpSecurityCenterManagementBodyMap(body, "securityHealthAnalyticsCustomModule")
	if len(module) == 0 {
		module = body
	}
	if len(module) == 0 {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "securityHealthAnalyticsCustomModule is required")
		return true
	}
	expectedName := gcpSecurityCenterManagementSHAModuleName(scope, scopeID, location, moduleID)
	if got := strings.TrimSpace(gcpSecurityCenterManagementString(module, "name")); got == "" || got != expectedName {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "securityHealthAnalyticsCustomModule.name must match requested resource")
		return true
	}
	if gcpSecurityCenterManagementMaskContains(updateMask, "customConfig") && !gcpSecurityCenterManagementHasMap(module, "customConfig") {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "securityHealthAnalyticsCustomModule.customConfig is required by updateMask")
		return true
	}

	resp := gcpSecurityCenterManagementSHAModule(scope, scopeID, location, moduleID)
	applyGCPSecurityCenterManagementSHAOverrides(resp, module)
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPSecurityCenterManagementDeleteSHAModule(w http.ResponseWriter, r *http.Request, path string) bool {
	if _, _, _, _, ok := parseGCPSecurityCenterManagementSHAModulePath(path); !ok {
		return false
	}
	if _, valid := parseGCPSecurityCenterManagementOptionalBoolQuery(w, path, r.URL.Query().Get("validateOnly"), "validateOnly"); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSecurityCenterManagementSimulateSHAModule(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || !isGCPSecurityCenterManagementSimulateSHATail(tail) {
		return false
	}
	body, valid := decodeGCPSecurityCenterManagementJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	if parent := strings.TrimSpace(gcpSecurityCenterManagementString(body, "parent")); parent != "" && parent != gcpSecurityCenterManagementParent(scope, scopeID, location) {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "parent in body must match request path")
		return true
	}
	if !gcpSecurityCenterManagementHasMap(body, "customConfig") {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "customConfig is required")
		return true
	}
	if !gcpSecurityCenterManagementHasMap(body, "resource") {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "resource is required")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"result": map[string]any{
			"finding": map[string]any{
				"name":         fmt.Sprintf("%s/sources/source-1/findings/simulated-finding-1", gcpSecurityCenterManagementScopeRoot(scope, scopeID)),
				"parent":       fmt.Sprintf("%s/sources/source-1", gcpSecurityCenterManagementScopeRoot(scope, scopeID)),
				"resourceName": fmt.Sprintf("//cloudresourcemanager.googleapis.com/%s/%s", scope, scopeID),
				"category":     "stackyard_custom_sha_violation",
				"state":        "ACTIVE",
				"severity":     "HIGH",
				"findingClass": "MISCONFIGURATION",
				"sourceProperties": map[string]any{
					"simulated": true,
				},
				"eventTime": gcpSecurityCenterManagementReferenceTime.Format(time.RFC3339),
			},
		},
	})
	return true
}

func handleGCPSecurityCenterManagementListEffectiveETDModules(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || !isGCPSecurityCenterManagementEffectiveETDModuleCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterManagementPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterManagementEffectiveETDModule(scope, scopeID, location, "effective-etd-module-1"),
		gcpSecurityCenterManagementEffectiveETDModule(scope, scopeID, location, "effective-etd-module-2"),
	}
	return respondGCPSecurityCenterManagementList(w, "effectiveEventThreatDetectionCustomModules", items, pageSize, start, path)
}

func handleGCPSecurityCenterManagementGetEffectiveETDModule(w http.ResponseWriter, path string) bool {
	scope, scopeID, location, moduleID, ok := parseGCPSecurityCenterManagementEffectiveETDModulePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterManagementEffectiveETDModule(scope, scopeID, location, moduleID))
	return true
}

func handleGCPSecurityCenterManagementListETDModules(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || !isGCPSecurityCenterManagementETDModuleCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterManagementPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterManagementETDModule(scope, scopeID, location, "etd-module-1"),
		gcpSecurityCenterManagementETDModule(scope, scopeID, location, "etd-module-2"),
	}
	return respondGCPSecurityCenterManagementList(w, "eventThreatDetectionCustomModules", items, pageSize, start, path)
}

func handleGCPSecurityCenterManagementListDescendantETDModules(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || !isGCPSecurityCenterManagementListDescendantETDTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterManagementPagination(w, r, path)
	if !valid {
		return true
	}
	item := gcpSecurityCenterManagementETDModule(scope, scopeID, location, "descendant-etd-module-1")
	item["ancestorModule"] = gcpSecurityCenterManagementETDModuleName(scope, scopeID, location, "etd-module-1")
	items := []map[string]any{item}
	return respondGCPSecurityCenterManagementList(w, "eventThreatDetectionCustomModules", items, pageSize, start, path)
}

func handleGCPSecurityCenterManagementGetETDModule(w http.ResponseWriter, path string) bool {
	scope, scopeID, location, moduleID, ok := parseGCPSecurityCenterManagementETDModulePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterManagementETDModule(scope, scopeID, location, moduleID))
	return true
}

func handleGCPSecurityCenterManagementCreateETDModule(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || !isGCPSecurityCenterManagementETDModuleCollectionTail(tail) {
		return false
	}
	if _, valid := parseGCPSecurityCenterManagementOptionalBoolQuery(w, path, r.URL.Query().Get("validateOnly"), "validateOnly"); !valid {
		return true
	}
	body, valid := decodeGCPSecurityCenterManagementJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	module := gcpSecurityCenterManagementBodyMap(body, "eventThreatDetectionCustomModule")
	if len(module) == 0 {
		module = body
	}
	if len(module) == 0 {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "eventThreatDetectionCustomModule is required")
		return true
	}
	if strings.TrimSpace(gcpSecurityCenterManagementString(module, "type")) == "" {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "eventThreatDetectionCustomModule.type is required")
		return true
	}
	if !gcpSecurityCenterManagementHasMap(module, "config") {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "eventThreatDetectionCustomModule.config is required")
		return true
	}

	moduleID := "etd-module-created"
	if rawName := strings.TrimSpace(gcpSecurityCenterManagementString(module, "name")); rawName != "" {
		nameScope, nameScopeID, nameLocation, nameID, nameOK := parseGCPSecurityCenterManagementETDModuleName(rawName)
		if !nameOK {
			respondGCPSecurityCenterManagementInvalidArgument(w, path, "eventThreatDetectionCustomModule.name is invalid")
			return true
		}
		if nameScope != scope || nameScopeID != scopeID || nameLocation != location {
			respondGCPSecurityCenterManagementInvalidArgument(w, path, "eventThreatDetectionCustomModule.name must match parent")
			return true
		}
		moduleID = nameID
	}

	resp := gcpSecurityCenterManagementETDModule(scope, scopeID, location, moduleID)
	applyGCPSecurityCenterManagementETDOverrides(resp, module)
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPSecurityCenterManagementUpdateETDModule(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, moduleID, ok := parseGCPSecurityCenterManagementETDModulePath(path)
	if !ok {
		return false
	}
	updateMask, valid := parseGCPSecurityCenterManagementUpdateMask(w, path, r.URL.Query().Get("updateMask"), []string{"config", "enablementState", "displayName", "description"})
	if !valid {
		return true
	}
	if len(updateMask) == 0 {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "updateMask is required")
		return true
	}
	if _, valid := parseGCPSecurityCenterManagementOptionalBoolQuery(w, path, r.URL.Query().Get("validateOnly"), "validateOnly"); !valid {
		return true
	}
	body, valid := decodeGCPSecurityCenterManagementJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	module := gcpSecurityCenterManagementBodyMap(body, "eventThreatDetectionCustomModule")
	if len(module) == 0 {
		module = body
	}
	if len(module) == 0 {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "eventThreatDetectionCustomModule is required")
		return true
	}
	expectedName := gcpSecurityCenterManagementETDModuleName(scope, scopeID, location, moduleID)
	if got := strings.TrimSpace(gcpSecurityCenterManagementString(module, "name")); got == "" || got != expectedName {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "eventThreatDetectionCustomModule.name must match requested resource")
		return true
	}
	if gcpSecurityCenterManagementMaskContains(updateMask, "config") && !gcpSecurityCenterManagementHasMap(module, "config") {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "eventThreatDetectionCustomModule.config is required by updateMask")
		return true
	}

	resp := gcpSecurityCenterManagementETDModule(scope, scopeID, location, moduleID)
	applyGCPSecurityCenterManagementETDOverrides(resp, module)
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPSecurityCenterManagementDeleteETDModule(w http.ResponseWriter, r *http.Request, path string) bool {
	if _, _, _, _, ok := parseGCPSecurityCenterManagementETDModulePath(path); !ok {
		return false
	}
	if _, valid := parseGCPSecurityCenterManagementOptionalBoolQuery(w, path, r.URL.Query().Get("validateOnly"), "validateOnly"); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSecurityCenterManagementValidateETDModule(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || !isGCPSecurityCenterManagementValidateETDTail(tail) {
		return false
	}
	body, valid := decodeGCPSecurityCenterManagementJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	if parent := strings.TrimSpace(gcpSecurityCenterManagementString(body, "parent")); parent != "" && parent != gcpSecurityCenterManagementParent(scope, scopeID, location) {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "parent in body must match request path")
		return true
	}
	rawText := strings.TrimSpace(gcpSecurityCenterManagementString(body, "rawText"))
	if rawText == "" {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "rawText is required")
		return true
	}
	if strings.TrimSpace(gcpSecurityCenterManagementString(body, "type")) == "" {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "type is required")
		return true
	}

	errors := []any{}
	if strings.Contains(strings.ToLower(rawText), "invalid") || strings.Contains(strings.ToLower(rawText), "error") {
		errors = append(errors, map[string]any{
			"description": "Raw module text failed validation",
			"fieldPath":   "/rawText",
			"start": map[string]any{
				"lineNumber":   1,
				"columnNumber": 1,
			},
			"end": map[string]any{
				"lineNumber":   1,
				"columnNumber": 8,
			},
		})
	}
	respondJSON(w, http.StatusOK, map[string]any{"errors": errors})
	return true
}

func handleGCPSecurityCenterManagementGetService(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, serviceID, ok := parseGCPSecurityCenterManagementServicePath(path)
	if !ok {
		return false
	}
	if !isGCPSecurityCenterManagementServiceID(serviceID) {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "service id is invalid")
		return true
	}
	if _, valid := parseGCPSecurityCenterManagementOptionalBoolQuery(w, path, r.URL.Query().Get("showEligibleModulesOnly"), "showEligibleModulesOnly"); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterManagementService(scope, scopeID, location, serviceID))
	return true
}

func handleGCPSecurityCenterManagementListServices(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || !isGCPSecurityCenterManagementServiceCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterManagementPagination(w, r, path)
	if !valid {
		return true
	}
	showEligibleOnly, valid := parseGCPSecurityCenterManagementOptionalBoolQuery(w, path, r.URL.Query().Get("showEligibleModulesOnly"), "showEligibleModulesOnly")
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterManagementService(scope, scopeID, location, "security-health-analytics"),
		gcpSecurityCenterManagementService(scope, scopeID, location, "event-threat-detection"),
		gcpSecurityCenterManagementService(scope, scopeID, location, "vm-threat-detection"),
	}
	if showEligibleOnly {
		items = items[:2]
	}
	return respondGCPSecurityCenterManagementList(w, "securityCenterServices", items, pageSize, start, path)
}

func handleGCPSecurityCenterManagementUpdateService(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, location, serviceID, ok := parseGCPSecurityCenterManagementServicePath(path)
	if !ok {
		return false
	}
	if !isGCPSecurityCenterManagementServiceID(serviceID) {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "service id is invalid")
		return true
	}
	updateMask, valid := parseGCPSecurityCenterManagementUpdateMask(w, path, r.URL.Query().Get("updateMask"), []string{"intendedEnablementState", "modules", "serviceConfig"})
	if !valid {
		return true
	}
	if len(updateMask) == 0 {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "updateMask is required")
		return true
	}
	if _, valid := parseGCPSecurityCenterManagementOptionalBoolQuery(w, path, r.URL.Query().Get("validateOnly"), "validateOnly"); !valid {
		return true
	}
	body, valid := decodeGCPSecurityCenterManagementJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	service := gcpSecurityCenterManagementBodyMap(body, "securityCenterService")
	if len(service) == 0 {
		service = body
	}
	if len(service) == 0 {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "securityCenterService is required")
		return true
	}
	expectedName := gcpSecurityCenterManagementServiceName(scope, scopeID, location, serviceID)
	if got := strings.TrimSpace(gcpSecurityCenterManagementString(service, "name")); got == "" || got != expectedName {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "securityCenterService.name must match requested resource")
		return true
	}
	if gcpSecurityCenterManagementMaskContains(updateMask, "modules") {
		if modules, ok := service["modules"].(map[string]any); !ok || len(modules) == 0 {
			respondGCPSecurityCenterManagementInvalidArgument(w, path, "securityCenterService.modules is required by updateMask")
			return true
		}
	}
	if strings.EqualFold(gcpSecurityCenterManagementString(service, "intendedEnablementState"), "INGEST_ONLY") {
		respondGCPSecurityCenterManagementFailedPrecondition(w, path, "securityCenterService.intendedEnablementState INGEST_ONLY is read-only")
		return true
	}

	resp := gcpSecurityCenterManagementService(scope, scopeID, location, serviceID)
	applyGCPSecurityCenterManagementServiceOverrides(resp, service)
	respondJSON(w, http.StatusOK, resp)
	return true
}

func parseGCPSecurityCenterManagementLocationPath(path string) (scope, scopeID, location string, list, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 && len(parts) != 6 {
		return "", "", "", false, false
	}
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || !isGCPSecurityCenterManagementScope(parts[2]) || parts[4] != "locations" {
		return "", "", "", false, false
	}
	scope = strings.TrimSpace(parts[2])
	scopeID = strings.TrimSpace(parts[3])
	if scopeID == "" {
		return "", "", "", false, false
	}
	if len(parts) == 5 {
		return scope, scopeID, "", true, true
	}
	location = strings.TrimSpace(parts[5])
	if location == "" {
		return "", "", "", false, false
	}
	return scope, scopeID, location, false, true
}

func parseGCPSecurityCenterManagementScopeTail(path string) (scope, scopeID, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 {
		return "", "", "", nil, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || !isGCPSecurityCenterManagementScope(parts[2]) || parts[4] != "locations" {
		return "", "", "", nil, false
	}
	scope = strings.TrimSpace(parts[2])
	scopeID = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if scopeID == "" || location == "" {
		return "", "", "", nil, false
	}
	return scope, scopeID, location, parts[6:], true
}

func isGCPSecurityCenterManagementScope(scope string) bool {
	switch scope {
	case "organizations", "folders", "projects":
		return true
	default:
		return false
	}
}

func isGCPSecurityCenterManagementServiceCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "securityCenterServices"
}

func isGCPSecurityCenterManagementSHAModuleCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "securityHealthAnalyticsCustomModules"
}

func isGCPSecurityCenterManagementListDescendantSHATail(tail []string) bool {
	if len(tail) != 1 {
		return false
	}
	base, action, ok := strings.Cut(tail[0], ":")
	return ok && base == "securityHealthAnalyticsCustomModules" && action == "listDescendant"
}

func isGCPSecurityCenterManagementSimulateSHATail(tail []string) bool {
	if len(tail) != 1 {
		return false
	}
	base, action, ok := strings.Cut(tail[0], ":")
	return ok && base == "securityHealthAnalyticsCustomModules" && action == "simulate"
}

func isGCPSecurityCenterManagementEffectiveSHAModuleCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "effectiveSecurityHealthAnalyticsCustomModules"
}

func isGCPSecurityCenterManagementETDModuleCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "eventThreatDetectionCustomModules"
}

func isGCPSecurityCenterManagementListDescendantETDTail(tail []string) bool {
	if len(tail) != 1 {
		return false
	}
	base, action, ok := strings.Cut(tail[0], ":")
	return ok && base == "eventThreatDetectionCustomModules" && action == "listDescendant"
}

func isGCPSecurityCenterManagementValidateETDTail(tail []string) bool {
	if len(tail) != 1 {
		return false
	}
	base, action, ok := strings.Cut(tail[0], ":")
	return ok && base == "eventThreatDetectionCustomModules" && action == "validate"
}

func isGCPSecurityCenterManagementEffectiveETDModuleCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "effectiveEventThreatDetectionCustomModules"
}

func parseGCPSecurityCenterManagementServicePath(path string) (scope, scopeID, location, serviceID string, ok bool) {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "securityCenterServices" {
		return "", "", "", "", false
	}
	serviceID = strings.TrimSpace(tail[1])
	if serviceID == "" || strings.Contains(serviceID, ":") {
		return "", "", "", "", false
	}
	return scope, scopeID, location, serviceID, true
}

func parseGCPSecurityCenterManagementSHAModulePath(path string) (scope, scopeID, location, moduleID string, ok bool) {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "securityHealthAnalyticsCustomModules" {
		return "", "", "", "", false
	}
	moduleID = strings.TrimSpace(tail[1])
	if moduleID == "" || strings.Contains(moduleID, ":") {
		return "", "", "", "", false
	}
	return scope, scopeID, location, moduleID, true
}

func parseGCPSecurityCenterManagementEffectiveSHAModulePath(path string) (scope, scopeID, location, moduleID string, ok bool) {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "effectiveSecurityHealthAnalyticsCustomModules" {
		return "", "", "", "", false
	}
	moduleID = strings.TrimSpace(tail[1])
	if moduleID == "" || strings.Contains(moduleID, ":") {
		return "", "", "", "", false
	}
	return scope, scopeID, location, moduleID, true
}

func parseGCPSecurityCenterManagementETDModulePath(path string) (scope, scopeID, location, moduleID string, ok bool) {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "eventThreatDetectionCustomModules" {
		return "", "", "", "", false
	}
	moduleID = strings.TrimSpace(tail[1])
	if moduleID == "" || strings.Contains(moduleID, ":") {
		return "", "", "", "", false
	}
	return scope, scopeID, location, moduleID, true
}

func parseGCPSecurityCenterManagementEffectiveETDModulePath(path string) (scope, scopeID, location, moduleID string, ok bool) {
	scope, scopeID, location, tail, ok := parseGCPSecurityCenterManagementScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "effectiveEventThreatDetectionCustomModules" {
		return "", "", "", "", false
	}
	moduleID = strings.TrimSpace(tail[1])
	if moduleID == "" || strings.Contains(moduleID, ":") {
		return "", "", "", "", false
	}
	return scope, scopeID, location, moduleID, true
}

func parseGCPSecurityCenterManagementParentName(parent string) (scope, scopeID, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || !isGCPSecurityCenterManagementScope(parts[0]) || parts[2] != "locations" {
		return "", "", "", false
	}
	scope = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if scopeID == "" || location == "" {
		return "", "", "", false
	}
	return scope, scopeID, location, true
}

func parseGCPSecurityCenterManagementServiceName(name string) (scope, scopeID, location, serviceID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || !isGCPSecurityCenterManagementScope(parts[0]) || parts[2] != "locations" || parts[4] != "securityCenterServices" {
		return "", "", "", "", false
	}
	scope = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	serviceID = strings.TrimSpace(parts[5])
	if scopeID == "" || location == "" || serviceID == "" {
		return "", "", "", "", false
	}
	return scope, scopeID, location, serviceID, true
}

func parseGCPSecurityCenterManagementSHAModuleName(name string) (scope, scopeID, location, moduleID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || !isGCPSecurityCenterManagementScope(parts[0]) || parts[2] != "locations" || parts[4] != "securityHealthAnalyticsCustomModules" {
		return "", "", "", "", false
	}
	scope = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	moduleID = strings.TrimSpace(parts[5])
	if scopeID == "" || location == "" || moduleID == "" {
		return "", "", "", "", false
	}
	return scope, scopeID, location, moduleID, true
}

func parseGCPSecurityCenterManagementETDModuleName(name string) (scope, scopeID, location, moduleID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || !isGCPSecurityCenterManagementScope(parts[0]) || parts[2] != "locations" || parts[4] != "eventThreatDetectionCustomModules" {
		return "", "", "", "", false
	}
	scope = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	moduleID = strings.TrimSpace(parts[5])
	if scopeID == "" || location == "" || moduleID == "" {
		return "", "", "", "", false
	}
	return scope, scopeID, location, moduleID, true
}

func parseGCPSecurityCenterManagementEffectiveSHAModuleName(name string) (scope, scopeID, location, moduleID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || !isGCPSecurityCenterManagementScope(parts[0]) || parts[2] != "locations" || parts[4] != "effectiveSecurityHealthAnalyticsCustomModules" {
		return "", "", "", "", false
	}
	scope = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	moduleID = strings.TrimSpace(parts[5])
	if scopeID == "" || location == "" || moduleID == "" {
		return "", "", "", "", false
	}
	return scope, scopeID, location, moduleID, true
}

func parseGCPSecurityCenterManagementEffectiveETDModuleName(name string) (scope, scopeID, location, moduleID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || !isGCPSecurityCenterManagementScope(parts[0]) || parts[2] != "locations" || parts[4] != "effectiveEventThreatDetectionCustomModules" {
		return "", "", "", "", false
	}
	scope = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	moduleID = strings.TrimSpace(parts[5])
	if scopeID == "" || location == "" || moduleID == "" {
		return "", "", "", "", false
	}
	return scope, scopeID, location, moduleID, true
}

func parseGCPSecurityCenterManagementPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil || pageSize > 1000 {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "pageSize must be a non-negative integer no greater than 1000")
		return 0, 0, false
	}
	start = 0
	if token := strings.TrimSpace(r.URL.Query().Get("pageToken")); token != "" {
		start, err = parseOptionalNonNegativeInt(token)
		if err != nil {
			respondGCPSecurityCenterManagementInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func parseGCPSecurityCenterManagementOptionalBoolQuery(w http.ResponseWriter, path, raw, field string) (bool, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false, true
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, field+" must be a boolean")
		return false, false
	}
	return parsed, true
}

func parseGCPSecurityCenterManagementUpdateMask(w http.ResponseWriter, path, raw string, allowed []string) ([]string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, true
	}
	allowedSet := map[string]struct{}{}
	for _, item := range allowed {
		allowedSet[item] = struct{}{}
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		field := strings.TrimSpace(part)
		if field == "" {
			continue
		}
		if _, ok := allowedSet[field]; !ok {
			respondGCPSecurityCenterManagementInvalidArgument(w, path, "updateMask contains unsupported path: "+field)
			return nil, false
		}
		out = append(out, field)
	}
	if len(out) == 0 {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "updateMask must include at least one path")
		return nil, false
	}
	return out, true
}

func respondGCPSecurityCenterManagementList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "pageToken is out of range")
		return false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		key:             items[start:end],
		"nextPageToken": next,
	})
	return true
}

func decodeGCPSecurityCenterManagementJSONBody(w http.ResponseWriter, r *http.Request, path string, required bool) (map[string]any, bool) {
	if r.Body == nil {
		if required {
			respondGCPSecurityCenterManagementInvalidArgument(w, path, "request body is required")
			return nil, false
		}
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		if required {
			respondGCPSecurityCenterManagementInvalidArgument(w, path, "request body is required")
			return nil, false
		}
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpSecurityCenterManagementBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return map[string]any{}
}

func gcpSecurityCenterManagementString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func gcpSecurityCenterManagementHasMap(body map[string]any, key string) bool {
	nested, ok := body[key].(map[string]any)
	return ok && len(nested) > 0
}

func gcpSecurityCenterManagementMaskContains(mask []string, field string) bool {
	for _, value := range mask {
		if value == field {
			return true
		}
	}
	return false
}

func isGCPSecurityCenterManagementServiceID(serviceID string) bool {
	switch serviceID {
	case "container-threat-detection", "event-threat-detection", "security-health-analytics", "vm-threat-detection", "web-security-scanner":
		return true
	default:
		return false
	}
}

func gcpSecurityCenterManagementScopeRoot(scope, scopeID string) string {
	return fmt.Sprintf("%s/%s", scope, scopeID)
}

func gcpSecurityCenterManagementParent(scope, scopeID, location string) string {
	return fmt.Sprintf("%s/%s/locations/%s", scope, scopeID, location)
}

func gcpSecurityCenterManagementLocation(scope, scopeID, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("%s/%s/locations/%s", scope, scopeID, location),
		"locationId":  location,
		"displayName": "Security Center Management " + location,
		"labels": map[string]string{
			"service": "securitycentermanagement",
		},
	}
}

func gcpSecurityCenterManagementServiceName(scope, scopeID, location, serviceID string) string {
	return fmt.Sprintf("%s/%s/locations/%s/securityCenterServices/%s", scope, scopeID, location, serviceID)
}

func gcpSecurityCenterManagementService(scope, scopeID, location, serviceID string) map[string]any {
	return map[string]any{
		"name":                     gcpSecurityCenterManagementServiceName(scope, scopeID, location, serviceID),
		"intendedEnablementState":  "ENABLED",
		"effectiveEnablementState": "ENABLED",
		"modules": map[string]any{
			"default-module": map[string]any{
				"intendedEnablementState":  "ENABLED",
				"effectiveEnablementState": "ENABLED",
			},
		},
		"updateTime": gcpSecurityCenterManagementReferenceTime.Format(time.RFC3339),
		"serviceConfig": map[string]any{
			"mode": "STANDARD",
		},
	}
}

func applyGCPSecurityCenterManagementServiceOverrides(target map[string]any, service map[string]any) {
	if intended := gcpSecurityCenterManagementString(service, "intendedEnablementState"); intended != "" {
		target["intendedEnablementState"] = intended
	}
	if modules, ok := service["modules"].(map[string]any); ok && len(modules) > 0 {
		target["modules"] = modules
	}
	if config, ok := service["serviceConfig"].(map[string]any); ok && len(config) > 0 {
		target["serviceConfig"] = config
	}
}

func gcpSecurityCenterManagementSHAModuleName(scope, scopeID, location, moduleID string) string {
	return fmt.Sprintf("%s/%s/locations/%s/securityHealthAnalyticsCustomModules/%s", scope, scopeID, location, moduleID)
}

func gcpSecurityCenterManagementEffectiveSHAModuleName(scope, scopeID, location, moduleID string) string {
	return fmt.Sprintf("%s/%s/locations/%s/effectiveSecurityHealthAnalyticsCustomModules/%s", scope, scopeID, location, moduleID)
}

func gcpSecurityCenterManagementSHAModule(scope, scopeID, location, moduleID string) map[string]any {
	return map[string]any{
		"name":            gcpSecurityCenterManagementSHAModuleName(scope, scopeID, location, moduleID),
		"displayName":     "stackyard_sha_" + strings.ReplaceAll(moduleID, "-", "_"),
		"enablementState": "ENABLED",
		"updateTime":      gcpSecurityCenterManagementReferenceTime.Format(time.RFC3339),
		"lastEditor":      "stackyard@example.com",
		"customConfig":    gcpSecurityCenterManagementSHACustomConfig(),
	}
}

func gcpSecurityCenterManagementEffectiveSHAModule(scope, scopeID, location, moduleID string) map[string]any {
	return map[string]any{
		"name":            gcpSecurityCenterManagementEffectiveSHAModuleName(scope, scopeID, location, moduleID),
		"displayName":     "effective_sha_" + strings.ReplaceAll(moduleID, "-", "_"),
		"enablementState": "ENABLED",
		"customConfig":    gcpSecurityCenterManagementSHACustomConfig(),
	}
}

func gcpSecurityCenterManagementSHACustomConfig() map[string]any {
	return map[string]any{
		"predicate": map[string]any{
			"expression": "resource.name.startsWith(\"//compute.googleapis.com\")",
		},
		"resourceSelector": map[string]any{
			"resourceTypes": []any{"compute.googleapis.com/Instance"},
		},
		"severity":       "HIGH",
		"description":    "Detect risky compute configuration",
		"recommendation": "Review and remediate the resource configuration",
		"customOutput": map[string]any{
			"properties": []any{
				map[string]any{
					"name": "assetType",
					"valueExpression": map[string]any{
						"expression": "resource.assetType",
					},
				},
			},
		},
	}
}

func applyGCPSecurityCenterManagementSHAOverrides(target map[string]any, module map[string]any) {
	if displayName := gcpSecurityCenterManagementString(module, "displayName"); displayName != "" {
		target["displayName"] = displayName
	}
	if enablementState := gcpSecurityCenterManagementString(module, "enablementState"); enablementState != "" {
		target["enablementState"] = enablementState
	}
	if customConfig, ok := module["customConfig"].(map[string]any); ok && len(customConfig) > 0 {
		target["customConfig"] = customConfig
	}
}

func gcpSecurityCenterManagementETDModuleName(scope, scopeID, location, moduleID string) string {
	return fmt.Sprintf("%s/%s/locations/%s/eventThreatDetectionCustomModules/%s", scope, scopeID, location, moduleID)
}

func gcpSecurityCenterManagementEffectiveETDModuleName(scope, scopeID, location, moduleID string) string {
	return fmt.Sprintf("%s/%s/locations/%s/effectiveEventThreatDetectionCustomModules/%s", scope, scopeID, location, moduleID)
}

func gcpSecurityCenterManagementETDModule(scope, scopeID, location, moduleID string) map[string]any {
	return map[string]any{
		"name":            gcpSecurityCenterManagementETDModuleName(scope, scopeID, location, moduleID),
		"config":          gcpSecurityCenterManagementETDConfig(),
		"enablementState": "ENABLED",
		"type":            "CONFIGURABLE_BAD_IP",
		"displayName":     "Stackyard ETD " + moduleID,
		"description":     "Detect suspicious IP events",
		"updateTime":      gcpSecurityCenterManagementReferenceTime.Format(time.RFC3339),
		"lastEditor":      "stackyard@example.com",
	}
}

func gcpSecurityCenterManagementEffectiveETDModule(scope, scopeID, location, moduleID string) map[string]any {
	return map[string]any{
		"name":            gcpSecurityCenterManagementEffectiveETDModuleName(scope, scopeID, location, moduleID),
		"config":          gcpSecurityCenterManagementETDConfig(),
		"enablementState": "ENABLED",
		"type":            "CONFIGURABLE_BAD_IP",
		"displayName":     "Effective ETD " + moduleID,
		"description":     "Effective suspicious IP detector",
	}
}

func gcpSecurityCenterManagementETDConfig() map[string]any {
	return map[string]any{
		"allowedIp": "10.0.0.1",
		"mode":      "BLOCK",
	}
}

func applyGCPSecurityCenterManagementETDOverrides(target map[string]any, module map[string]any) {
	if config, ok := module["config"].(map[string]any); ok && len(config) > 0 {
		target["config"] = config
	}
	if enablementState := gcpSecurityCenterManagementString(module, "enablementState"); enablementState != "" {
		target["enablementState"] = enablementState
	}
	if typeName := gcpSecurityCenterManagementString(module, "type"); typeName != "" {
		target["type"] = typeName
	}
	if displayName := gcpSecurityCenterManagementString(module, "displayName"); displayName != "" {
		target["displayName"] = displayName
	}
	if description := gcpSecurityCenterManagementString(module, "description"); description != "" {
		target["description"] = description
	}
}

func respondGCPSecurityCenterManagementInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPSecurityCenterManagementFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_securitycentermanagement(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "securitycentermanagement") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPSecurityCenterManagementInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/securityCenterServices/security-health-analytics",
			"service":  "securitycentermanagement",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
