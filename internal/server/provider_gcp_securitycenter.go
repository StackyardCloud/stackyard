package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	gcpSecurityCenterReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpSecurityCenterIDPattern     = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
)

func (s *Server) handleGCPSecurityCenterRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_securitycenter(w, r) {
		return true
	}

	path := normalizeGCPSecurityCenterPath(rawRequestPath(r))
	if !isGCPSecurityCenterPath(path, hasGCPSecurityCenterHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPSecurityCenterListSources(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterGetSource(w, path) {
			return true
		}
		if handleGCPSecurityCenterListFindings(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterGroupFindings(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterListMuteConfigs(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterGetMuteConfig(w, path) {
			return true
		}
		if handleGCPSecurityCenterListNotificationConfigs(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterGetNotificationConfig(w, path) {
			return true
		}
		if handleGCPSecurityCenterGetOrganizationSettings(w, path) {
			return true
		}
		if handleGCPSecurityCenterListBigQueryExports(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterGetBigQueryExport(w, path) {
			return true
		}
		if handleGCPSecurityCenterGetOperation(w, path) {
			return true
		}
		if handleGCPSecurityCenterListOperations(w, r, path) {
			return true
		}

		// Stage 9 extended surfaces.
		if handleGCPSecurityCenterGetSimulation(w, path) {
			return true
		}
		if handleGCPSecurityCenterGetValuedResource(w, path) {
			return true
		}
		if handleGCPSecurityCenterListValuedResources(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterListAttackPaths(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterListCustomModules(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterListDescendantCustomModules(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterListEffectiveCustomModules(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterGetCustomModule(w, path) {
			return true
		}
		if handleGCPSecurityCenterGetEffectiveCustomModule(w, path) {
			return true
		}
		if handleGCPSecurityCenterListResourceValueConfigs(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterGetResourceValueConfig(w, path) {
			return true
		}

		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPSecurityCenterCreateSource(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterGroupFindings(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterCreateFinding(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterSetFindingState(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterSetMute(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterCreateMuteConfig(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterCreateNotificationConfig(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterCreateBigQueryExport(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterCreateResourceValueConfig(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterBulkMuteFindings(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterGetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterSetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterTestIAMPermissions(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterCancelOperation(w, r, path) {
			return true
		}

		// Stage 9 extended surfaces.
		if handleGCPSecurityCenterCreateCustomModule(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterSimulateCustomModule(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterValidateCustomModule(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterBatchCreateResourceValueConfigs(w, r, path) {
			return true
		}

		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPSecurityCenterUpdateSource(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterUpdateFinding(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterUpdateExternalSystem(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterUpdateSecurityMarks(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterUpdateMuteConfig(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterUpdateNotificationConfig(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterUpdateOrganizationSettings(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterUpdateBigQueryExport(w, r, path) {
			return true
		}

		// Stage 9 extended surfaces.
		if handleGCPSecurityCenterUpdateCustomModule(w, r, path) {
			return true
		}
		if handleGCPSecurityCenterUpdateResourceValueConfig(w, r, path) {
			return true
		}

		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPSecurityCenterDeleteMuteConfig(w, path) {
			return true
		}
		if handleGCPSecurityCenterDeleteNotificationConfig(w, path) {
			return true
		}
		if handleGCPSecurityCenterDeleteBigQueryExport(w, path) {
			return true
		}
		if handleGCPSecurityCenterDeleteOperation(w, path) {
			return true
		}

		// Stage 9 extended surfaces.
		if handleGCPSecurityCenterDeleteCustomModule(w, path) {
			return true
		}
		if handleGCPSecurityCenterDeleteResourceValueConfig(w, path) {
			return true
		}

		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPSecurityCenterPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPSecurityCenterHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "securitycenter",
		"securitycenter-apiv1",
		"securitycenter_apiv1",
		"securitycenter-apiv2",
		"securitycenter_apiv2",
		"securitycenter-v2",
		"security-center",
		"security-center-v2",
		"security_center",
		"security_center_v2",
		"scc",
		"scc-v2",
		"gcp-security-command-center-v2":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-securitycenter-apiv1") ||
		strings.Contains(ua, "stackyard-securitycenter-apiv2") ||
		strings.Contains(ua, "cloud.google.com/go/securitycenter")
}

func isGCPSecurityCenterPath(path string, includeAmbiguous bool) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.securitycenter.v1.SecurityCenter/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.securitycenter.v2.SecurityCenter/") {
		return true
	}
	_, _, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) == 0 {
		return false
	}

	// Unique SCC endpoints are recognized without hint.
	if strings.Contains(path, "/muteConfigs") ||
		strings.Contains(path, "/notificationConfigs") ||
		strings.Contains(path, "/organizationSettings") ||
		strings.Contains(path, "/bigQueryExports") ||
		strings.Contains(path, "/simulations") ||
		strings.Contains(path, "/resourceValueConfigs") ||
		strings.Contains(path, "/valuedResources") ||
		strings.Contains(path, "/attackPaths") ||
		strings.Contains(path, "/customModules") ||
		strings.Contains(path, "/effectiveCustomModules") ||
		strings.Contains(path, ":bulkMute") ||
		strings.Contains(path, ":validateCustomModule") ||
		strings.Contains(path, ":getIamPolicy") ||
		strings.Contains(path, ":setIamPolicy") ||
		strings.Contains(path, ":testIamPermissions") {
		return true
	}

	if !includeAmbiguous {
		return false
	}

	return strings.Contains(path, "/sources") || strings.Contains(path, "/findings") || strings.Contains(path, "/operations")
}

func handleGCPSecurityCenterListSources(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterSourcesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterSourceFixture(scope, scopeID, "source-1"),
		gcpSecurityCenterSourceFixture(scope, scopeID, "source-2"),
	}
	return respondGCPSecurityCenterList(w, "sources", items, pageSize, start, path)
}

func handleGCPSecurityCenterGetSource(w http.ResponseWriter, path string) bool {
	scope, scopeID, sourceID, ok := parseGCPSecurityCenterSourcePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterSourceFixture(scope, scopeID, sourceID))
	return true
}

func handleGCPSecurityCenterCreateSource(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterSourcesCollectionTail(tail) {
		return false
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	source := gcpSecurityCenterBodyMap(body, "source")
	if len(source) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "source is required")
		return true
	}
	if strings.TrimSpace(gcpSecurityCenterString(source, "displayName")) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "source.displayName is required")
		return true
	}

	sourceID := "source-1"
	if gotName := strings.TrimSpace(gcpSecurityCenterString(source, "name")); gotName != "" {
		_, _, parsedSourceID, parsed := parseGCPSecurityCenterSourceName(gotName)
		if !parsed {
			respondGCPSecurityCenterInvalidArgument(w, path, "source.name is invalid")
			return true
		}
		sourceID = parsedSourceID
	}
	resp := gcpSecurityCenterSourceFixture(scope, scopeID, sourceID)
	resp["displayName"] = strings.TrimSpace(gcpSecurityCenterString(source, "displayName"))
	if d := strings.TrimSpace(gcpSecurityCenterString(source, "description")); d != "" {
		resp["description"] = d
	}
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPSecurityCenterUpdateSource(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, sourceID, ok := parseGCPSecurityCenterSourcePath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	source := gcpSecurityCenterBodyMap(body, "source")
	if len(source) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "source is required")
		return true
	}
	expectedName := gcpSecurityCenterSourceName(scope, scopeID, sourceID)
	if got := strings.TrimSpace(gcpSecurityCenterString(source, "name")); got == "" || got != expectedName {
		respondGCPSecurityCenterInvalidArgument(w, path, "source.name must match requested resource")
		return true
	}
	resp := gcpSecurityCenterSourceFixture(scope, scopeID, sourceID)
	if d := strings.TrimSpace(gcpSecurityCenterString(source, "displayName")); d != "" {
		resp["displayName"] = d
	}
	if d := strings.TrimSpace(gcpSecurityCenterString(source, "description")); d != "" {
		resp["description"] = d
	}
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPSecurityCenterListFindings(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, sourceID, ok := parseGCPSecurityCenterFindingsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		{
			"finding":     gcpSecurityCenterFindingFixture(scope, scopeID, sourceID, "finding-1"),
			"resource":    gcpSecurityCenterResourceFixture(scope, scopeID),
			"stateChange": "CHANGED",
		},
		{
			"finding":     gcpSecurityCenterFindingFixture(scope, scopeID, sourceID, "finding-2"),
			"resource":    gcpSecurityCenterResourceFixture(scope, scopeID),
			"stateChange": "UNCHANGED",
		},
	}
	return respondGCPSecurityCenterListEnvelope(w, "listFindingsResults", items, pageSize, start, path)
}

func handleGCPSecurityCenterGroupFindings(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, sourceID, ok := parseGCPSecurityCenterGroupFindingsPath(path)
	if !ok {
		return false
	}
	groupBy := strings.TrimSpace(r.URL.Query().Get("groupBy"))
	if groupBy == "" && r.Method == http.MethodPost {
		body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
		if !valid {
			return true
		}
		groupBy = strings.TrimSpace(gcpSecurityCenterString(body, "groupBy"))
	}
	if groupBy == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "groupBy is required")
		return true
	}
	pageSize, start, valid := parseGCPSecurityCenterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		{
			"properties": map[string]any{
				"category": "OPEN_FIREWALL",
				"source":   sourceID,
			},
			"count": "1",
		},
		{
			"properties": map[string]any{
				"category": "PUBLIC_BUCKET",
				"source":   sourceID,
			},
			"count": "1",
		},
	}
	if sourceID == "-" {
		items[1]["properties"].(map[string]any)["scope"] = fmt.Sprintf("%s/%s", scope, scopeID)
	}
	return respondGCPSecurityCenterListEnvelope(w, "groupByResults", items, pageSize, start, path)
}

func handleGCPSecurityCenterCreateFinding(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, sourceID, ok := parseGCPSecurityCenterFindingsCollectionPath(path)
	if !ok {
		return false
	}
	findingID := strings.TrimSpace(r.URL.Query().Get("findingId"))
	if findingID == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "findingId is required")
		return true
	}
	if !gcpSecurityCenterIDPattern.MatchString(findingID) {
		respondGCPSecurityCenterInvalidArgument(w, path, "findingId is invalid")
		return true
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	finding := gcpSecurityCenterBodyMap(body, "finding")
	if len(finding) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "finding is required")
		return true
	}
	expectedName := gcpSecurityCenterFindingName(scope, scopeID, sourceID, findingID)
	if got := strings.TrimSpace(gcpSecurityCenterString(finding, "name")); got != "" && got != expectedName {
		respondGCPSecurityCenterInvalidArgument(w, path, "finding.name must match parent and findingId")
		return true
	}
	resp := gcpSecurityCenterFindingFixture(scope, scopeID, sourceID, findingID)
	if c := strings.TrimSpace(gcpSecurityCenterString(finding, "category")); c != "" {
		resp["category"] = c
	}
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPSecurityCenterUpdateFinding(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, sourceID, findingID, ok := parseGCPSecurityCenterFindingPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	finding := gcpSecurityCenterBodyMap(body, "finding")
	if len(finding) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "finding is required")
		return true
	}
	expectedName := gcpSecurityCenterFindingName(scope, scopeID, sourceID, findingID)
	if got := strings.TrimSpace(gcpSecurityCenterString(finding, "name")); got == "" || got != expectedName {
		respondGCPSecurityCenterInvalidArgument(w, path, "finding.name must match requested resource")
		return true
	}
	resp := gcpSecurityCenterFindingFixture(scope, scopeID, sourceID, findingID)
	if sev := strings.TrimSpace(gcpSecurityCenterString(finding, "severity")); sev != "" {
		resp["severity"] = sev
	}
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPSecurityCenterSetFindingState(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, sourceID, findingID, ok := parseGCPSecurityCenterFindingActionPath(path, "setState")
	if !ok {
		return false
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpSecurityCenterFindingName(scope, scopeID, sourceID, findingID)
	if got := strings.TrimSpace(gcpSecurityCenterString(body, "name")); got == "" || got != expectedName {
		respondGCPSecurityCenterInvalidArgument(w, path, "name must match requested finding")
		return true
	}
	state := gcpSecurityCenterEnumString(body, "state", map[int]string{
		0: "STATE_UNSPECIFIED",
		1: "ACTIVE",
		2: "INACTIVE",
	})
	if state == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "state is required")
		return true
	}
	resp := gcpSecurityCenterFindingFixture(scope, scopeID, sourceID, findingID)
	resp["state"] = strings.ToUpper(state)
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPSecurityCenterSetMute(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, sourceID, findingID, ok := parseGCPSecurityCenterFindingActionPath(path, "setMute")
	if !ok {
		return false
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpSecurityCenterFindingName(scope, scopeID, sourceID, findingID)
	if got := strings.TrimSpace(gcpSecurityCenterString(body, "name")); got == "" || got != expectedName {
		respondGCPSecurityCenterInvalidArgument(w, path, "name must match requested finding")
		return true
	}
	mute := gcpSecurityCenterEnumString(body, "mute", map[int]string{
		0: "MUTE_UNSPECIFIED",
		1: "MUTED",
		2: "UNMUTED",
	})
	if mute == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "mute is required")
		return true
	}
	if strings.Contains(findingID, "already-muted") && strings.EqualFold(mute, "MUTED") {
		respondGCPSecurityCenterFailedPrecondition(w, path, "finding is already muted")
		return true
	}
	resp := gcpSecurityCenterFindingFixture(scope, scopeID, sourceID, findingID)
	resp["mute"] = strings.ToUpper(mute)
	resp["muteUpdateTime"] = gcpSecurityCenterReferenceTime.Add(5 * time.Minute).Format(time.RFC3339Nano)
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPSecurityCenterUpdateExternalSystem(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, sourceID, findingID, externalSystemID, ok := parseGCPSecurityCenterExternalSystemPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	external := gcpSecurityCenterBodyMap(body, "externalSystem")
	if len(external) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "externalSystem is required")
		return true
	}
	expectedName := gcpSecurityCenterExternalSystemName(scope, scopeID, sourceID, findingID, externalSystemID)
	if got := strings.TrimSpace(gcpSecurityCenterString(external, "name")); got == "" || got != expectedName {
		respondGCPSecurityCenterInvalidArgument(w, path, "externalSystem.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterExternalSystemFixture(scope, scopeID, sourceID, findingID, externalSystemID))
	return true
}

func handleGCPSecurityCenterUpdateSecurityMarks(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, sourceID, findingID, ok := parseGCPSecurityCenterFindingSecurityMarksPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	marks := gcpSecurityCenterBodyMap(body, "securityMarks")
	if len(marks) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "securityMarks is required")
		return true
	}
	expectedName := gcpSecurityCenterFindingSecurityMarksName(scope, scopeID, sourceID, findingID)
	if got := strings.TrimSpace(gcpSecurityCenterString(marks, "name")); got == "" || got != expectedName {
		respondGCPSecurityCenterInvalidArgument(w, path, "securityMarks.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterSecurityMarksFixture(scope, scopeID, sourceID, findingID))
	return true
}

func handleGCPSecurityCenterListMuteConfigs(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterMuteConfigsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterMuteConfigFixture(scope, scopeID, "mute-config-1"),
		gcpSecurityCenterMuteConfigFixture(scope, scopeID, "mute-config-2"),
	}
	return respondGCPSecurityCenterList(w, "muteConfigs", items, pageSize, start, path)
}

func handleGCPSecurityCenterGetMuteConfig(w http.ResponseWriter, path string) bool {
	scope, scopeID, id, ok := parseGCPSecurityCenterMuteConfigPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterMuteConfigFixture(scope, scopeID, id))
	return true
}

func handleGCPSecurityCenterCreateMuteConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterMuteConfigsCollectionTail(tail) {
		return false
	}
	id := strings.TrimSpace(r.URL.Query().Get("muteConfigId"))
	if id == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "muteConfigId is required")
		return true
	}
	if !gcpSecurityCenterIDPattern.MatchString(id) {
		respondGCPSecurityCenterInvalidArgument(w, path, "muteConfigId is invalid")
		return true
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	cfg := gcpSecurityCenterBodyMap(body, "muteConfig")
	if len(cfg) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "muteConfig is required")
		return true
	}
	if strings.TrimSpace(gcpSecurityCenterString(cfg, "filter")) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "muteConfig.filter is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterMuteConfigFixture(scope, scopeID, id))
	return true
}

func handleGCPSecurityCenterUpdateMuteConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, id, ok := parseGCPSecurityCenterMuteConfigPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	cfg := gcpSecurityCenterBodyMap(body, "muteConfig")
	if len(cfg) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "muteConfig is required")
		return true
	}
	expectedName := gcpSecurityCenterMuteConfigName(scope, scopeID, id)
	if got := strings.TrimSpace(gcpSecurityCenterString(cfg, "name")); got == "" || got != expectedName {
		respondGCPSecurityCenterInvalidArgument(w, path, "muteConfig.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterMuteConfigFixture(scope, scopeID, id))
	return true
}

func handleGCPSecurityCenterDeleteMuteConfig(w http.ResponseWriter, path string) bool {
	if _, _, _, ok := parseGCPSecurityCenterMuteConfigPath(path); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSecurityCenterListNotificationConfigs(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterNotificationConfigsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterNotificationConfigFixture(scope, scopeID, "notify-1"),
		gcpSecurityCenterNotificationConfigFixture(scope, scopeID, "notify-2"),
	}
	return respondGCPSecurityCenterList(w, "notificationConfigs", items, pageSize, start, path)
}

func handleGCPSecurityCenterGetNotificationConfig(w http.ResponseWriter, path string) bool {
	scope, scopeID, id, ok := parseGCPSecurityCenterNotificationConfigPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterNotificationConfigFixture(scope, scopeID, id))
	return true
}

func handleGCPSecurityCenterCreateNotificationConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterNotificationConfigsCollectionTail(tail) {
		return false
	}
	id := strings.TrimSpace(r.URL.Query().Get("configId"))
	if id == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "configId is required")
		return true
	}
	if !gcpSecurityCenterIDPattern.MatchString(id) {
		respondGCPSecurityCenterInvalidArgument(w, path, "configId is invalid")
		return true
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	cfg := gcpSecurityCenterBodyMap(body, "notificationConfig")
	if len(cfg) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "notificationConfig is required")
		return true
	}
	if strings.TrimSpace(gcpSecurityCenterString(cfg, "pubsubTopic")) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "notificationConfig.pubsubTopic is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterNotificationConfigFixture(scope, scopeID, id))
	return true
}

func handleGCPSecurityCenterUpdateNotificationConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, id, ok := parseGCPSecurityCenterNotificationConfigPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	cfg := gcpSecurityCenterBodyMap(body, "notificationConfig")
	if len(cfg) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "notificationConfig is required")
		return true
	}
	expectedName := gcpSecurityCenterNotificationConfigName(scope, scopeID, id)
	if got := strings.TrimSpace(gcpSecurityCenterString(cfg, "name")); got == "" || got != expectedName {
		respondGCPSecurityCenterInvalidArgument(w, path, "notificationConfig.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterNotificationConfigFixture(scope, scopeID, id))
	return true
}

func handleGCPSecurityCenterDeleteNotificationConfig(w http.ResponseWriter, path string) bool {
	if _, _, _, ok := parseGCPSecurityCenterNotificationConfigPath(path); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSecurityCenterGetOrganizationSettings(w http.ResponseWriter, path string) bool {
	orgID, ok := parseGCPSecurityCenterOrganizationSettingsPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterOrganizationSettingsFixture(orgID))
	return true
}

func handleGCPSecurityCenterUpdateOrganizationSettings(w http.ResponseWriter, r *http.Request, path string) bool {
	orgID, ok := parseGCPSecurityCenterOrganizationSettingsPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	settings := gcpSecurityCenterBodyMap(body, "organizationSettings")
	if len(settings) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "organizationSettings is required")
		return true
	}
	expectedName := gcpSecurityCenterOrganizationSettingsName(orgID)
	if got := strings.TrimSpace(gcpSecurityCenterString(settings, "name")); got == "" || got != expectedName {
		respondGCPSecurityCenterInvalidArgument(w, path, "organizationSettings.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterOrganizationSettingsFixture(orgID))
	return true
}

func handleGCPSecurityCenterListBigQueryExports(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterBigQueryExportsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterBigQueryExportFixture(scope, scopeID, "export-1"),
		gcpSecurityCenterBigQueryExportFixture(scope, scopeID, "export-2"),
	}
	return respondGCPSecurityCenterList(w, "bigQueryExports", items, pageSize, start, path)
}

func handleGCPSecurityCenterGetBigQueryExport(w http.ResponseWriter, path string) bool {
	scope, scopeID, id, ok := parseGCPSecurityCenterBigQueryExportPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterBigQueryExportFixture(scope, scopeID, id))
	return true
}

func handleGCPSecurityCenterCreateBigQueryExport(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterBigQueryExportsCollectionTail(tail) {
		return false
	}
	id := strings.TrimSpace(r.URL.Query().Get("bigQueryExportId"))
	if id == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "bigQueryExportId is required")
		return true
	}
	if !gcpSecurityCenterIDPattern.MatchString(id) {
		respondGCPSecurityCenterInvalidArgument(w, path, "bigQueryExportId is invalid")
		return true
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	export := gcpSecurityCenterBodyMap(body, "bigQueryExport")
	if len(export) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "bigQueryExport is required")
		return true
	}
	if strings.TrimSpace(gcpSecurityCenterString(export, "dataset")) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "bigQueryExport.dataset is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterBigQueryExportFixture(scope, scopeID, id))
	return true
}

func handleGCPSecurityCenterUpdateBigQueryExport(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, id, ok := parseGCPSecurityCenterBigQueryExportPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	export := gcpSecurityCenterBodyMap(body, "bigQueryExport")
	if len(export) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "bigQueryExport is required")
		return true
	}
	expectedName := gcpSecurityCenterBigQueryExportName(scope, scopeID, id)
	if got := strings.TrimSpace(gcpSecurityCenterString(export, "name")); got == "" || got != expectedName {
		respondGCPSecurityCenterInvalidArgument(w, path, "bigQueryExport.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterBigQueryExportFixture(scope, scopeID, id))
	return true
}

func handleGCPSecurityCenterDeleteBigQueryExport(w http.ResponseWriter, path string) bool {
	if _, _, _, ok := parseGCPSecurityCenterBigQueryExportPath(path); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSecurityCenterCreateResourceValueConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterResourceValueConfigsCollectionTail(tail) {
		return false
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	cfg := gcpSecurityCenterBodyMap(body, "resourceValueConfig")
	if len(cfg) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "resourceValueConfig is required")
		return true
	}
	configID := "config-1"
	if name := strings.TrimSpace(gcpSecurityCenterString(cfg, "name")); name != "" {
		nameScope, nameScopeID, nameID, nameOK := parseGCPSecurityCenterResourceValueConfigName(name)
		if !nameOK {
			respondGCPSecurityCenterInvalidArgument(w, path, "resourceValueConfig.name is invalid")
			return true
		}
		if nameScope != scope || nameScopeID != scopeID {
			respondGCPSecurityCenterInvalidArgument(w, path, "resourceValueConfig.name must match parent")
			return true
		}
		configID = nameID
	}
	if strings.TrimSpace(gcpSecurityCenterEnumString(cfg, "resourceValue", map[int]string{
		0: "RESOURCE_VALUE_UNSPECIFIED",
		1: "LOW",
		2: "MEDIUM",
		3: "HIGH",
	})) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "resourceValueConfig.resourceValue is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterResourceValueConfigFixture(scope, scopeID, configID))
	return true
}

func handleGCPSecurityCenterBulkMuteFindings(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, sourceID, ok := parseGCPSecurityCenterBulkMutePath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(gcpSecurityCenterString(body, "filter")) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "filter is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterOperationFixture(scope, scopeID, "bulkMuteFindings."+sourceID, "bulkMute"))
	return true
}

func handleGCPSecurityCenterGetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	if !isGCPSecurityCenterIAMActionPath(path, "getIamPolicy") || r.Method != http.MethodPost {
		return false
	}
	if _, valid := decodeGCPSecurityCenterJSONBody(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterPolicyFixture())
	return true
}

func handleGCPSecurityCenterSetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	if !isGCPSecurityCenterIAMActionPath(path, "setIamPolicy") || r.Method != http.MethodPost {
		return false
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	policy, _ := body["policy"].(map[string]any)
	if len(policy) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "policy is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterPolicyFixture())
	return true
}

func handleGCPSecurityCenterTestIAMPermissions(w http.ResponseWriter, r *http.Request, path string) bool {
	if !isGCPSecurityCenterIAMActionPath(path, "testIamPermissions") || r.Method != http.MethodPost {
		return false
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	permissions := []string{"securitycenter.sources.get"}
	if raw, ok := body["permissions"].([]any); ok && len(raw) > 0 {
		permissions = permissions[:0]
		for _, entry := range raw {
			if permission, ok := entry.(string); ok && strings.TrimSpace(permission) != "" {
				permissions = append(permissions, strings.TrimSpace(permission))
			}
		}
		if len(permissions) == 0 {
			permissions = []string{"securitycenter.sources.get"}
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"permissions": permissions})
	return true
}

func handleGCPSecurityCenterListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterOperationsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterOperationFixture(scope, scopeID, "op-1", "create"),
		gcpSecurityCenterOperationFixture(scope, scopeID, "op-2", "update"),
	}
	return respondGCPSecurityCenterList(w, "operations", items, pageSize, start, path)
}

func handleGCPSecurityCenterGetOperation(w http.ResponseWriter, path string) bool {
	scope, scopeID, opID, ok := parseGCPSecurityCenterOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterOperationFixture(scope, scopeID, opID, "get"))
	return true
}

func handleGCPSecurityCenterCancelOperation(w http.ResponseWriter, r *http.Request, path string) bool {
	if _, _, _, ok := parseGCPSecurityCenterOperationActionPath(path, "cancel"); !ok {
		return false
	}
	if _, valid := decodeGCPSecurityCenterJSONBody(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSecurityCenterDeleteOperation(w http.ResponseWriter, path string) bool {
	if _, _, _, ok := parseGCPSecurityCenterOperationPath(path); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSecurityCenterGetSimulation(w http.ResponseWriter, path string) bool {
	scope, scopeID, simID, ok := parseGCPSecurityCenterSimulationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":       fmt.Sprintf("%s/%s/simulations/%s", scope, scopeID, simID),
		"createTime": gcpSecurityCenterReferenceTime.Format(time.RFC3339Nano),
		"state":      "SUCCEEDED",
	})
	return true
}

func handleGCPSecurityCenterGetValuedResource(w http.ResponseWriter, path string) bool {
	scope, scopeID, simulationID, resourceID, ok := parseGCPSecurityCenterValuedResourcePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterValuedResourceFixture(scope, scopeID, simulationID, resourceID))
	return true
}

func handleGCPSecurityCenterListValuedResources(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, simulationID, ok := parseGCPSecurityCenterValuedResourcesCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterValuedResourceFixture(scope, scopeID, simulationID, "resource-1"),
		gcpSecurityCenterValuedResourceFixture(scope, scopeID, simulationID, "resource-2"),
	}
	return respondGCPSecurityCenterList(w, "valuedResources", items, pageSize, start, path)
}

func handleGCPSecurityCenterListAttackPaths(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, simulationID, valuedResourceID, ok := parseGCPSecurityCenterAttackPathsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterPagination(w, r, path)
	if !valid {
		return true
	}
	attackPathParent := fmt.Sprintf("%s/%s", scope, scopeID)
	if simulationID != "" {
		attackPathParent += "/simulations/" + simulationID
	}
	if valuedResourceID != "" {
		attackPathParent += "/valuedResources/" + valuedResourceID
	}
	items := []map[string]any{
		{
			"name":        attackPathParent + "/attackPaths/path-1",
			"displayName": "Stackyard Attack Path 1",
		},
		{
			"name":        attackPathParent + "/attackPaths/path-2",
			"displayName": "Stackyard Attack Path 2",
		},
	}
	return respondGCPSecurityCenterList(w, "attackPaths", items, pageSize, start, path)
}

func handleGCPSecurityCenterListCustomModules(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterCustomModulesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterCustomModuleFixture(scope, scopeID, "custom-module-1"),
		gcpSecurityCenterCustomModuleFixture(scope, scopeID, "custom-module-2"),
	}
	return respondGCPSecurityCenterList(w, "customModules", items, pageSize, start, path)
}

func handleGCPSecurityCenterListDescendantCustomModules(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterListDescendantCustomModulesTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterCustomModuleFixture(scope, scopeID, "descendant-module-1"),
	}
	return respondGCPSecurityCenterList(w, "securityHealthAnalyticsCustomModules", items, pageSize, start, path)
}

func handleGCPSecurityCenterListEffectiveCustomModules(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterEffectiveCustomModulesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterEffectiveCustomModuleFixture(scope, scopeID, "effective-module-1"),
	}
	return respondGCPSecurityCenterList(w, "effectiveSecurityHealthAnalyticsCustomModules", items, pageSize, start, path)
}

func handleGCPSecurityCenterGetCustomModule(w http.ResponseWriter, path string) bool {
	scope, scopeID, id, ok := parseGCPSecurityCenterCustomModulePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterCustomModuleFixture(scope, scopeID, id))
	return true
}

func handleGCPSecurityCenterGetEffectiveCustomModule(w http.ResponseWriter, path string) bool {
	scope, scopeID, id, ok := parseGCPSecurityCenterEffectiveCustomModulePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterEffectiveCustomModuleFixture(scope, scopeID, id))
	return true
}

func handleGCPSecurityCenterCreateCustomModule(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterCustomModulesCollectionTail(tail) {
		return false
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	module := gcpSecurityCenterBodyMap(body, "securityHealthAnalyticsCustomModule")
	if len(module) == 0 {
		module = gcpSecurityCenterBodyMap(body, "eventThreatDetectionCustomModule")
	}
	if len(module) == 0 {
		module = body
	}
	name := strings.TrimSpace(gcpSecurityCenterString(module, "displayName"))
	if name == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "custom module displayName is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterCustomModuleFixture(scope, scopeID, "custom-module-1"))
	return true
}

func handleGCPSecurityCenterUpdateCustomModule(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, id, ok := parseGCPSecurityCenterCustomModulePath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	module := gcpSecurityCenterBodyMap(body, "securityHealthAnalyticsCustomModule")
	if len(module) == 0 {
		module = gcpSecurityCenterBodyMap(body, "eventThreatDetectionCustomModule")
	}
	if len(module) == 0 {
		module = body
	}
	expected := gcpSecurityCenterCustomModuleName(scope, scopeID, id)
	if got := strings.TrimSpace(gcpSecurityCenterString(module, "name")); got == "" || got != expected {
		respondGCPSecurityCenterInvalidArgument(w, path, "custom module name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterCustomModuleFixture(scope, scopeID, id))
	return true
}

func handleGCPSecurityCenterDeleteCustomModule(w http.ResponseWriter, path string) bool {
	if _, _, _, ok := parseGCPSecurityCenterCustomModulePath(path); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSecurityCenterSimulateCustomModule(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterSimulateCustomModuleTail(tail) {
		return false
	}
	if _, valid := decodeGCPSecurityCenterJSONBody(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"result": map[string]any{
			"name":   fmt.Sprintf("%s/%s/customModules/simulated", scope, scopeID),
			"state":  "ACTIVE",
			"source": "SIMULATION",
		},
	})
	return true
}

func handleGCPSecurityCenterValidateCustomModule(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, action, ok := parseGCPSecurityCenterScopeActionPath(path)
	if !ok || action != "validateCustomModule" {
		return false
	}
	if _, valid := decodeGCPSecurityCenterJSONBody(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"name":    fmt.Sprintf("%s/%s:validateCustomModule", scope, scopeID),
		"success": true,
	})
	return true
}

func handleGCPSecurityCenterBatchCreateResourceValueConfigs(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterBatchCreateResourceValueConfigsTail(tail) {
		return false
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	if _, ok := body["requests"].([]any); !ok {
		respondGCPSecurityCenterInvalidArgument(w, path, "requests is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"resourceValueConfigs": []map[string]any{
			gcpSecurityCenterResourceValueConfigFixture(scope, scopeID, "config-1"),
		},
	})
	return true
}

func handleGCPSecurityCenterListResourceValueConfigs(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || !isGCPSecurityCenterResourceValueConfigsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPSecurityCenterPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSecurityCenterResourceValueConfigFixture(scope, scopeID, "config-1"),
		gcpSecurityCenterResourceValueConfigFixture(scope, scopeID, "config-2"),
	}
	return respondGCPSecurityCenterList(w, "resourceValueConfigs", items, pageSize, start, path)
}

func handleGCPSecurityCenterGetResourceValueConfig(w http.ResponseWriter, path string) bool {
	scope, scopeID, id, ok := parseGCPSecurityCenterResourceValueConfigPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterResourceValueConfigFixture(scope, scopeID, id))
	return true
}

func handleGCPSecurityCenterUpdateResourceValueConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	scope, scopeID, id, ok := parseGCPSecurityCenterResourceValueConfigPath(path)
	if !ok {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPSecurityCenterInvalidArgument(w, path, "updateMask is required")
		return true
	}
	body, valid := decodeGCPSecurityCenterJSONBody(w, r, path)
	if !valid {
		return true
	}
	cfg := gcpSecurityCenterBodyMap(body, "resourceValueConfig")
	if len(cfg) == 0 {
		respondGCPSecurityCenterInvalidArgument(w, path, "resourceValueConfig is required")
		return true
	}
	expected := gcpSecurityCenterResourceValueConfigName(scope, scopeID, id)
	if got := strings.TrimSpace(gcpSecurityCenterString(cfg, "name")); got == "" || got != expected {
		respondGCPSecurityCenterInvalidArgument(w, path, "resourceValueConfig.name must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSecurityCenterResourceValueConfigFixture(scope, scopeID, id))
	return true
}

func handleGCPSecurityCenterDeleteResourceValueConfig(w http.ResponseWriter, path string) bool {
	if _, _, _, ok := parseGCPSecurityCenterResourceValueConfigPath(path); !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPSecurityCenterScopeTail(path string) (scope, scopeID string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || (parts[1] != "v1" && parts[1] != "v2") {
		return "", "", nil, false
	}
	scope = strings.TrimSpace(parts[2])
	if scope != "organizations" && scope != "folders" && scope != "projects" {
		return "", "", nil, false
	}
	scopeID = strings.TrimSpace(parts[3])
	if scopeID == "" {
		return "", "", nil, false
	}
	return scope, scopeID, parts[4:], true
}

func parseGCPSecurityCenterScopeActionPath(path string) (scope, scopeID, action string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 1 {
		return "", "", "", false
	}
	_, action, found := splitGCPSecurityCenterActionSegment(tail[0])
	if !found {
		return "", "", "", false
	}
	return scope, scopeID, action, true
}

func isGCPSecurityCenterSourcesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "sources"
}

func parseGCPSecurityCenterSourcePath(path string) (scope, scopeID, sourceID string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "sources" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), true
}

func parseGCPSecurityCenterFindingsCollectionPath(path string) (scope, scopeID, sourceID string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 3 || tail[0] != "sources" || strings.TrimSpace(tail[1]) == "" || tail[2] != "findings" {
		return "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), true
}

func parseGCPSecurityCenterGroupFindingsPath(path string) (scope, scopeID, sourceID string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 3 || tail[0] != "sources" || strings.TrimSpace(tail[1]) == "" || !strings.HasPrefix(tail[2], "findings:") {
		return "", "", "", false
	}
	_, action, found := splitGCPSecurityCenterActionSegment(tail[2])
	if !found || action != "group" {
		return "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), true
}

func parseGCPSecurityCenterBulkMutePath(path string) (scope, scopeID, sourceID string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 3 || tail[0] != "sources" || strings.TrimSpace(tail[1]) == "" {
		return "", "", "", false
	}
	_, action, found := splitGCPSecurityCenterActionSegment(tail[2])
	if !found || action != "bulkMute" {
		return "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), true
}

func parseGCPSecurityCenterFindingPath(path string) (scope, scopeID, sourceID, findingID string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 4 || tail[0] != "sources" || strings.TrimSpace(tail[1]) == "" || tail[2] != "findings" {
		return "", "", "", "", false
	}
	if strings.TrimSpace(tail[3]) == "" {
		return "", "", "", "", false
	}
	if strings.Contains(tail[3], ":") {
		return "", "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), true
}

func parseGCPSecurityCenterFindingActionPath(path, action string) (scope, scopeID, sourceID, findingID string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 4 || tail[0] != "sources" || strings.TrimSpace(tail[1]) == "" || tail[2] != "findings" {
		return "", "", "", "", false
	}
	id, parsedAction, found := splitGCPSecurityCenterActionSegment(tail[3])
	if !found || parsedAction != action {
		return "", "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), strings.TrimSpace(id), true
}

func parseGCPSecurityCenterExternalSystemPath(path string) (scope, scopeID, sourceID, findingID, externalSystemID string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 6 || tail[0] != "sources" || tail[2] != "findings" || tail[4] != "externalSystems" {
		return "", "", "", "", "", false
	}
	if strings.TrimSpace(tail[1]) == "" || strings.TrimSpace(tail[3]) == "" || strings.TrimSpace(tail[5]) == "" {
		return "", "", "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), strings.TrimSpace(tail[5]), true
}

func parseGCPSecurityCenterFindingSecurityMarksPath(path string) (scope, scopeID, sourceID, findingID string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 5 || tail[0] != "sources" || tail[2] != "findings" || tail[4] != "securityMarks" {
		return "", "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), true
}

func isGCPSecurityCenterMuteConfigsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "muteConfigs"
}

func parseGCPSecurityCenterMuteConfigPath(path string) (scope, scopeID, id string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "muteConfigs" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), true
}

func isGCPSecurityCenterNotificationConfigsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "notificationConfigs"
}

func parseGCPSecurityCenterNotificationConfigPath(path string) (scope, scopeID, id string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "notificationConfigs" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), true
}

func parseGCPSecurityCenterOrganizationSettingsPath(path string) (orgID string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || scope != "organizations" || len(tail) != 1 || tail[0] != "organizationSettings" {
		return "", false
	}
	return scopeID, true
}

func isGCPSecurityCenterBigQueryExportsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "bigQueryExports"
}

func parseGCPSecurityCenterBigQueryExportPath(path string) (scope, scopeID, id string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "bigQueryExports" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), true
}

func isGCPSecurityCenterIAMActionPath(path, action string) bool {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	var body string
	switch {
	case strings.HasPrefix(trimmed, "gcp/v1/"):
		body = strings.TrimPrefix(trimmed, "gcp/v1/")
	case strings.HasPrefix(trimmed, "gcp/v2/"):
		body = strings.TrimPrefix(trimmed, "gcp/v2/")
	default:
		return false
	}
	resource, gotAction, ok := splitGCPSecurityCenterActionSegment(body)
	return ok && gotAction == action && strings.TrimSpace(resource) != ""
}

func isGCPSecurityCenterOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func parseGCPSecurityCenterOperationPath(path string) (scope, scopeID, operationID string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), true
}

func parseGCPSecurityCenterOperationActionPath(path, action string) (scope, scopeID, operationID string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" {
		return "", "", "", false
	}
	id, gotAction, found := splitGCPSecurityCenterActionSegment(tail[1])
	if !found || gotAction != action || strings.TrimSpace(id) == "" {
		return "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(id), true
}

func parseGCPSecurityCenterSimulationPath(path string) (scope, scopeID, simulationID string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "simulations" || strings.TrimSpace(tail[1]) == "" {
		return "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), true
}

func parseGCPSecurityCenterValuedResourcesCollectionPath(path string) (scope, scopeID, simulationID string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok {
		return "", "", "", false
	}
	if len(tail) == 1 && tail[0] == "valuedResources" {
		return scope, scopeID, "", true
	}
	if len(tail) == 3 && tail[0] == "simulations" && strings.TrimSpace(tail[1]) != "" && tail[2] == "valuedResources" {
		return scope, scopeID, strings.TrimSpace(tail[1]), true
	}
	return "", "", "", false
}

func parseGCPSecurityCenterValuedResourcePath(path string) (scope, scopeID, simulationID, resourceID string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok {
		return "", "", "", "", false
	}
	if len(tail) == 2 && tail[0] == "valuedResources" && strings.TrimSpace(tail[1]) != "" {
		return scope, scopeID, "", strings.TrimSpace(tail[1]), true
	}
	if len(tail) == 4 &&
		tail[0] == "simulations" &&
		strings.TrimSpace(tail[1]) != "" &&
		tail[2] == "valuedResources" &&
		strings.TrimSpace(tail[3]) != "" {
		return scope, scopeID, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), true
	}
	return "", "", "", "", false
}

func parseGCPSecurityCenterAttackPathsCollectionPath(path string) (scope, scopeID, simulationID, valuedResourceID string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok {
		return "", "", "", "", false
	}
	if len(tail) == 1 && tail[0] == "attackPaths" {
		return scope, scopeID, "", "", true
	}
	if len(tail) == 3 && tail[0] == "simulations" && strings.TrimSpace(tail[1]) != "" && tail[2] == "attackPaths" {
		return scope, scopeID, strings.TrimSpace(tail[1]), "", true
	}
	if len(tail) == 5 &&
		tail[0] == "simulations" &&
		strings.TrimSpace(tail[1]) != "" &&
		tail[2] == "valuedResources" &&
		strings.TrimSpace(tail[3]) != "" &&
		tail[4] == "attackPaths" {
		return scope, scopeID, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), true
	}
	return "", "", "", "", false
}

func isGCPSecurityCenterCustomModulesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "customModules"
}

func isGCPSecurityCenterListDescendantCustomModulesTail(tail []string) bool {
	if len(tail) != 1 {
		return false
	}
	_, action, ok := splitGCPSecurityCenterActionSegment(tail[0])
	return ok && strings.HasPrefix(tail[0], "customModules:") && action == "listDescendant"
}

func isGCPSecurityCenterEffectiveCustomModulesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "effectiveCustomModules"
}

func parseGCPSecurityCenterCustomModulePath(path string) (scope, scopeID, id string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "customModules" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), true
}

func parseGCPSecurityCenterEffectiveCustomModulePath(path string) (scope, scopeID, id string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "effectiveCustomModules" || strings.TrimSpace(tail[1]) == "" {
		return "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), true
}

func isGCPSecurityCenterSimulateCustomModuleTail(tail []string) bool {
	if len(tail) != 1 {
		return false
	}
	id, action, ok := splitGCPSecurityCenterActionSegment(tail[0])
	return ok && id == "customModules" && action == "simulate"
}

func isGCPSecurityCenterBatchCreateResourceValueConfigsTail(tail []string) bool {
	if len(tail) != 1 {
		return false
	}
	id, action, ok := splitGCPSecurityCenterActionSegment(tail[0])
	return ok && id == "resourceValueConfigs" && action == "batchCreate"
}

func isGCPSecurityCenterResourceValueConfigsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "resourceValueConfigs"
}

func parseGCPSecurityCenterResourceValueConfigPath(path string) (scope, scopeID, id string, ok bool) {
	scope, scopeID, tail, ok := parseGCPSecurityCenterScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "resourceValueConfigs" || strings.TrimSpace(tail[1]) == "" {
		return "", "", "", false
	}
	return scope, scopeID, strings.TrimSpace(tail[1]), true
}

func splitGCPSecurityCenterActionSegment(raw string) (id, action string, ok bool) {
	segment := strings.TrimSpace(raw)
	if segment == "" {
		return "", "", false
	}
	if decoded, err := url.PathUnescape(segment); err == nil {
		segment = decoded
	}
	id, action, ok = strings.Cut(segment, ":")
	if !ok {
		return "", "", false
	}
	id = strings.TrimSpace(id)
	action = strings.TrimSpace(action)
	if id == "" || action == "" {
		return "", "", false
	}
	return id, action, true
}

func parseGCPSecurityCenterPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPSecurityCenterInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize > 1000 {
		respondGCPSecurityCenterInvalidArgument(w, path, "pageSize must be less than or equal to 1000")
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPSecurityCenterInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func decodeGCPSecurityCenterJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPSecurityCenterInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpSecurityCenterBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func gcpSecurityCenterString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func gcpSecurityCenterEnumString(body map[string]any, key string, valueMap map[int]string) string {
	if body == nil {
		return ""
	}
	switch raw := body[key].(type) {
	case string:
		return strings.TrimSpace(raw)
	case float64:
		return strings.TrimSpace(valueMap[int(raw)])
	}
	return ""
}

func respondGCPSecurityCenterList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPSecurityCenterInvalidArgument(w, path, "pageToken is out of range")
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

func respondGCPSecurityCenterListEnvelope(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPSecurityCenterInvalidArgument(w, path, "pageToken is out of range")
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
		"readTime":      gcpSecurityCenterReferenceTime.Format(time.RFC3339Nano),
		"nextPageToken": next,
		"totalSize":     len(items),
	})
	return true
}

func gcpSecurityCenterSourceName(scope, scopeID, sourceID string) string {
	return fmt.Sprintf("%s/%s/sources/%s", scope, scopeID, sourceID)
}

func gcpSecurityCenterFindingName(scope, scopeID, sourceID, findingID string) string {
	return fmt.Sprintf("%s/%s/sources/%s/findings/%s", scope, scopeID, sourceID, findingID)
}

func gcpSecurityCenterExternalSystemName(scope, scopeID, sourceID, findingID, externalSystemID string) string {
	return fmt.Sprintf("%s/%s/sources/%s/findings/%s/externalSystems/%s", scope, scopeID, sourceID, findingID, externalSystemID)
}

func gcpSecurityCenterFindingSecurityMarksName(scope, scopeID, sourceID, findingID string) string {
	return fmt.Sprintf("%s/%s/sources/%s/findings/%s/securityMarks", scope, scopeID, sourceID, findingID)
}

func gcpSecurityCenterMuteConfigName(scope, scopeID, muteConfigID string) string {
	return fmt.Sprintf("%s/%s/muteConfigs/%s", scope, scopeID, muteConfigID)
}

func gcpSecurityCenterNotificationConfigName(scope, scopeID, notificationConfigID string) string {
	return fmt.Sprintf("%s/%s/notificationConfigs/%s", scope, scopeID, notificationConfigID)
}

func gcpSecurityCenterOrganizationSettingsName(orgID string) string {
	return fmt.Sprintf("organizations/%s/organizationSettings", orgID)
}

func gcpSecurityCenterBigQueryExportName(scope, scopeID, exportID string) string {
	return fmt.Sprintf("%s/%s/bigQueryExports/%s", scope, scopeID, exportID)
}

func gcpSecurityCenterCustomModuleName(scope, scopeID, moduleID string) string {
	return fmt.Sprintf("%s/%s/customModules/%s", scope, scopeID, moduleID)
}

func gcpSecurityCenterResourceValueConfigName(scope, scopeID, configID string) string {
	return fmt.Sprintf("%s/%s/resourceValueConfigs/%s", scope, scopeID, configID)
}

func parseGCPSecurityCenterSourceName(name string) (scope, scopeID, sourceID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[2] != "sources" {
		return "", "", "", false
	}
	scope = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	sourceID = strings.TrimSpace(parts[3])
	if (scope != "organizations" && scope != "folders" && scope != "projects") || scopeID == "" || sourceID == "" {
		return "", "", "", false
	}
	return scope, scopeID, sourceID, true
}

func parseGCPSecurityCenterResourceValueConfigName(name string) (scope, scopeID, configID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[2] != "resourceValueConfigs" {
		return "", "", "", false
	}
	scope = strings.TrimSpace(parts[0])
	scopeID = strings.TrimSpace(parts[1])
	configID = strings.TrimSpace(parts[3])
	if (scope != "organizations" && scope != "folders" && scope != "projects") || scopeID == "" || configID == "" {
		return "", "", "", false
	}
	return scope, scopeID, configID, true
}

func gcpSecurityCenterSourceFixture(scope, scopeID, sourceID string) map[string]any {
	name := gcpSecurityCenterSourceName(scope, scopeID, sourceID)
	return map[string]any{
		"name":          name,
		"displayName":   "Stackyard Source " + sourceID,
		"description":   "Stackyard emulated Security Command Center source",
		"canonicalName": name,
	}
}

func gcpSecurityCenterResourceFixture(scope, scopeID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("//cloudresourcemanager.googleapis.com/%s/%s", scope, scopeID),
		"displayName": "Stackyard Resource",
		"type":        "google.cloud.resourcemanager.Project",
	}
}

func gcpSecurityCenterFindingFixture(scope, scopeID, sourceID, findingID string) map[string]any {
	name := gcpSecurityCenterFindingName(scope, scopeID, sourceID, findingID)
	return map[string]any{
		"name":         name,
		"parent":       gcpSecurityCenterSourceName(scope, scopeID, sourceID),
		"resourceName": fmt.Sprintf("//cloudresourcemanager.googleapis.com/%s/%s", scope, scopeID),
		"state":        "ACTIVE",
		"category":     "OPEN_FIREWALL",
		"severity":     "HIGH",
		"eventTime":    gcpSecurityCenterReferenceTime.Format(time.RFC3339Nano),
		"createTime":   gcpSecurityCenterReferenceTime.Add(-1 * time.Minute).Format(time.RFC3339Nano),
		"mute":         "UNMUTED",
		"securityMarks": gcpSecurityCenterSecurityMarksFixture(
			scope,
			scopeID,
			sourceID,
			findingID,
		),
		"externalSystems": map[string]any{
			"jira": gcpSecurityCenterExternalSystemFixture(scope, scopeID, sourceID, findingID, "jira"),
		},
	}
}

func gcpSecurityCenterExternalSystemFixture(scope, scopeID, sourceID, findingID, externalSystemID string) map[string]any {
	return map[string]any{
		"name":                     gcpSecurityCenterExternalSystemName(scope, scopeID, sourceID, findingID, externalSystemID),
		"externalUid":              "EXT-" + findingID,
		"status":                   "OPEN",
		"externalSystemUpdateTime": gcpSecurityCenterReferenceTime.Format(time.RFC3339Nano),
		"caseUri":                  "https://example.invalid/tickets/" + findingID,
	}
}

func gcpSecurityCenterSecurityMarksFixture(scope, scopeID, sourceID, findingID string) map[string]any {
	name := gcpSecurityCenterFindingSecurityMarksName(scope, scopeID, sourceID, findingID)
	return map[string]any{
		"name":          name,
		"canonicalName": name,
		"marks": map[string]string{
			"env":       "test",
			"stackyard": "true",
		},
	}
}

func gcpSecurityCenterMuteConfigFixture(scope, scopeID, muteConfigID string) map[string]any {
	return map[string]any{
		"name":             gcpSecurityCenterMuteConfigName(scope, scopeID, muteConfigID),
		"displayName":      "Stackyard Mute Config " + muteConfigID,
		"description":      "Mute configuration for staged SCC emulation",
		"filter":           `severity="HIGH"`,
		"createTime":       gcpSecurityCenterReferenceTime.Format(time.RFC3339Nano),
		"updateTime":       gcpSecurityCenterReferenceTime.Add(5 * time.Minute).Format(time.RFC3339Nano),
		"mostRecentEditor": "stackyard@example.invalid",
		"type":             "STATIC",
	}
}

func gcpSecurityCenterNotificationConfigFixture(scope, scopeID, notificationConfigID string) map[string]any {
	return map[string]any{
		"name":           gcpSecurityCenterNotificationConfigName(scope, scopeID, notificationConfigID),
		"description":    "Stackyard SCC notification config",
		"pubsubTopic":    fmt.Sprintf("projects/%s/topics/stackyard-scc", scopeID),
		"serviceAccount": "service-1234567890@gcp-sa-scc-notification.iam.gserviceaccount.com",
		"streamingConfig": map[string]any{
			"filter": `severity="HIGH"`,
		},
	}
}

func gcpSecurityCenterOrganizationSettingsFixture(orgID string) map[string]any {
	return map[string]any{
		"name":                 gcpSecurityCenterOrganizationSettingsName(orgID),
		"enableAssetDiscovery": true,
		"assetDiscoveryConfig": map[string]any{
			"inclusionMode": "INCLUDE_ONLY",
		},
	}
}

func gcpSecurityCenterBigQueryExportFixture(scope, scopeID, exportID string) map[string]any {
	return map[string]any{
		"name":             gcpSecurityCenterBigQueryExportName(scope, scopeID, exportID),
		"description":      "Stackyard SCC BigQuery export",
		"filter":           `severity="HIGH"`,
		"dataset":          fmt.Sprintf("projects/%s/datasets/stackyard_scc", scopeID),
		"createTime":       gcpSecurityCenterReferenceTime.Format(time.RFC3339Nano),
		"updateTime":       gcpSecurityCenterReferenceTime.Add(10 * time.Minute).Format(time.RFC3339Nano),
		"mostRecentEditor": "stackyard@example.invalid",
		"principal":        "serviceAccount:stackyard-scc@stackyard.iam.gserviceaccount.com",
	}
}

func gcpSecurityCenterOperationFixture(scope, scopeID, operationID, verb string) map[string]any {
	_ = verb
	return map[string]any{
		"name": fmt.Sprintf("%s/%s/operations/%s", scope, scopeID, operationID),
		"done": false,
	}
}

func gcpSecurityCenterPolicyFixture() map[string]any {
	return map[string]any{
		"version": 1,
		"etag":    "c3RhY2t5YXJk",
		"bindings": []map[string]any{
			{
				"role":    "roles/securitycenter.findingsEditor",
				"members": []string{"user:analyst@example.invalid"},
			},
		},
	}
}

func gcpSecurityCenterCustomModuleFixture(scope, scopeID, moduleID string) map[string]any {
	return map[string]any{
		"name":            gcpSecurityCenterCustomModuleName(scope, scopeID, moduleID),
		"displayName":     "Stackyard Custom Module " + moduleID,
		"enablementState": "ENABLED",
		"type":            "SECURITY_HEALTH_ANALYTICS",
	}
}

func gcpSecurityCenterEffectiveCustomModuleFixture(scope, scopeID, moduleID string) map[string]any {
	return map[string]any{
		"name":            fmt.Sprintf("%s/%s/effectiveCustomModules/%s", scope, scopeID, moduleID),
		"enablementState": "ENABLED",
		"inheritance":     "INHERITED",
	}
}

func gcpSecurityCenterResourceValueConfigFixture(scope, scopeID, configID string) map[string]any {
	return map[string]any{
		"name": gcpSecurityCenterResourceValueConfigName(scope, scopeID, configID),
		"tagValues": []string{
			"env:prod",
		},
		"resourceValue": "HIGH",
	}
}

func gcpSecurityCenterValuedResourceFixture(scope, scopeID, simulationID, resourceID string) map[string]any {
	name := fmt.Sprintf("%s/%s/valuedResources/%s", scope, scopeID, resourceID)
	if simulationID != "" {
		name = fmt.Sprintf("%s/%s/simulations/%s/valuedResources/%s", scope, scopeID, simulationID, resourceID)
	}
	return map[string]any{
		"name":        name,
		"displayName": "Stackyard Valued Resource " + resourceID,
		"resource":    fmt.Sprintf("//cloudresourcemanager.googleapis.com/%s/%s", scope, scopeID),
	}
}

func respondGCPSecurityCenterInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPSecurityCenterError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPSecurityCenterFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPSecurityCenterError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPSecurityCenterError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_securitycenter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "securitycenter") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageSize must be a non-negative integer",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "organizations/stackyard/sources/securitycenter/sample",
			"service":  "securitycenter",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
