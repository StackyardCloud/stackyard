package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const gcpWebSecurityScannerGRPCPathPrefix = "/gcp/google.cloud.websecurityscanner.v1.WebSecurityScanner/"

var (
	gcpWebSecurityScannerReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpWebSecurityScannerFilterRe      = regexp.MustCompile(`^\s*finding_type\s*=\s*"?([A-Za-z0-9_]+)"?\s*$`)
)

func (s *Server) handleGCPWebSecurityScannerRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_websecurityscanner(w, r) {
		return true
	}

	path := normalizeGCPWebSecurityScannerPath(rawRequestPath(r))
	if strings.HasPrefix(path, gcpWebSecurityScannerGRPCPathPrefix) {
		return handleGCPWebSecurityScannerGRPCBridge(w, r, path)
	}

	if !isGCPWebSecurityScannerRESTPath(path, hasGCPWebSecurityScannerHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPWebSecurityScannerListScanConfigs(w, r, path) {
			return true
		}
		if handleGCPWebSecurityScannerGetScanConfig(w, path) {
			return true
		}
		if handleGCPWebSecurityScannerGetScanRun(w, path) {
			return true
		}
		if handleGCPWebSecurityScannerListScanRuns(w, r, path) {
			return true
		}
		if handleGCPWebSecurityScannerListCrawledURLs(w, r, path) {
			return true
		}
		if handleGCPWebSecurityScannerGetFinding(w, path) {
			return true
		}
		if handleGCPWebSecurityScannerListFindings(w, r, path) {
			return true
		}
		if handleGCPWebSecurityScannerListFindingTypeStats(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPWebSecurityScannerCreateScanConfig(w, r, path) {
			return true
		}
		if handleGCPWebSecurityScannerStartScanRun(w, r, path) {
			return true
		}
		if handleGCPWebSecurityScannerStopScanRun(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPWebSecurityScannerUpdateScanConfig(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPWebSecurityScannerDeleteScanConfig(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPWebSecurityScannerPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPWebSecurityScannerHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "websecurityscanner",
		"websecurityscanner-apiv1",
		"websecurityscanner_apiv1",
		"web-security-scanner",
		"web_security_scanner",
		"gcp-websecurityscanner",
		"gcp-web-security-scanner":
		return true
	}

	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-websecurityscanner-apiv1") || strings.Contains(ua, "cloud.google.com/go/websecurityscanner")
}

func isGCPWebSecurityScannerRESTPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, gcpWebSecurityScannerGRPCPathPrefix) {
		return true
	}
	if _, tail, ok := parseGCPWebSecurityScannerProjectTail(path); ok {
		if isGCPWebSecurityScannerScanConfigsCollectionTail(tail) ||
			isGCPWebSecurityScannerScanConfigResourceTail(tail) ||
			isGCPWebSecurityScannerScanConfigActionTail(tail, "start") ||
			isGCPWebSecurityScannerScanRunsCollectionTail(tail) ||
			isGCPWebSecurityScannerScanRunResourceTail(tail) ||
			isGCPWebSecurityScannerScanRunActionTail(tail, "stop") ||
			isGCPWebSecurityScannerCrawledURLsCollectionTail(tail) ||
			isGCPWebSecurityScannerFindingsCollectionTail(tail) ||
			isGCPWebSecurityScannerFindingTypeStatsTail(tail) ||
			isGCPWebSecurityScannerFindingResourceTail(tail) {
			return true
		}
	}
	if includeHint && strings.HasPrefix(path, "/gcp/v1/projects/") {
		return true
	}
	return false
}

func handleGCPWebSecurityScannerListScanConfigs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPWebSecurityScannerProjectTail(path)
	if !ok || !isGCPWebSecurityScannerScanConfigsCollectionTail(tail) {
		return false
	}

	pageSize, start, valid := parseGCPWebSecurityScannerPagination(w, r, path, 1000)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpWebSecurityScannerScanConfig(project, "scan-config-1"),
		gcpWebSecurityScannerScanConfig(project, "scan-config-2"),
	}
	return respondGCPWebSecurityScannerList(w, "scanConfigs", items, pageSize, start, path)
}

func handleGCPWebSecurityScannerGetScanConfig(w http.ResponseWriter, path string) bool {
	project, tail, ok := parseGCPWebSecurityScannerProjectTail(path)
	if !ok || !isGCPWebSecurityScannerScanConfigResourceTail(tail) {
		return false
	}
	scanConfigID := strings.TrimSpace(tail[1])
	if isGCPWebSecurityScannerMissingID(scanConfigID) {
		respondGCPWebSecurityScannerNotFound(w, path, "scan_config not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpWebSecurityScannerScanConfig(project, scanConfigID))
	return true
}

func handleGCPWebSecurityScannerCreateScanConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPWebSecurityScannerProjectTail(path)
	if !ok || !isGCPWebSecurityScannerScanConfigsCollectionTail(tail) {
		return false
	}

	body, valid := decodeGCPWebSecurityScannerJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}

	if parent := strings.TrimSpace(gcpWebSecurityScannerString(body, "parent")); parent != "" {
		parentProject, parsed := parseGCPWebSecurityScannerProjectParent(parent)
		if !parsed {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "parent is invalid")
			return true
		}
		if parentProject != project {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "parent must match requested project")
			return true
		}
	}

	scanConfig := gcpWebSecurityScannerScanConfigFromCreateBody(body)
	if len(scanConfig) == 0 {
		respondGCPWebSecurityScannerInvalidArgument(w, path, "scanConfig is required")
		return true
	}
	if !validateGCPWebSecurityScannerScanConfig(w, path, scanConfig, false) {
		return true
	}

	scanConfigID := gcpWebSecurityScannerScanConfigIDFromMap(scanConfig, "scan-config-1")
	if isGCPWebSecurityScannerMissingID(scanConfigID) {
		respondGCPWebSecurityScannerAlreadyExists(w, path, "scan_config already exists")
		return true
	}

	response := gcpWebSecurityScannerScanConfig(project, scanConfigID)
	applyGCPWebSecurityScannerScanConfigOverrides(response, scanConfig)
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPWebSecurityScannerUpdateScanConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPWebSecurityScannerProjectTail(path)
	if !ok || !isGCPWebSecurityScannerScanConfigResourceTail(tail) {
		return false
	}

	scanConfigID := strings.TrimSpace(tail[1])
	if isGCPWebSecurityScannerMissingID(scanConfigID) {
		respondGCPWebSecurityScannerNotFound(w, path, "scan_config not found")
		return true
	}

	body, valid := decodeGCPWebSecurityScannerJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	scanConfig := gcpWebSecurityScannerScanConfigFromUpdateBody(body)
	if len(scanConfig) == 0 {
		respondGCPWebSecurityScannerInvalidArgument(w, path, "scanConfig is required")
		return true
	}
	if !validateGCPWebSecurityScannerScanConfig(w, path, scanConfig, true) {
		return true
	}

	expectedName := gcpWebSecurityScannerScanConfigName(project, scanConfigID)
	if strings.TrimSpace(gcpWebSecurityScannerString(scanConfig, "name")) != expectedName {
		respondGCPWebSecurityScannerInvalidArgument(w, path, "scanConfig.name must match requested resource")
		return true
	}

	if !validateGCPWebSecurityScannerUpdateMask(w, path, r, body) {
		return true
	}

	response := gcpWebSecurityScannerScanConfig(project, scanConfigID)
	applyGCPWebSecurityScannerScanConfigOverrides(response, scanConfig)
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPWebSecurityScannerDeleteScanConfig(w http.ResponseWriter, path string) bool {
	_, tail, ok := parseGCPWebSecurityScannerProjectTail(path)
	if !ok || !isGCPWebSecurityScannerScanConfigResourceTail(tail) {
		return false
	}
	scanConfigID := strings.TrimSpace(tail[1])
	if isGCPWebSecurityScannerMissingID(scanConfigID) {
		respondGCPWebSecurityScannerNotFound(w, path, "scan_config not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPWebSecurityScannerStartScanRun(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPWebSecurityScannerProjectTail(path)
	if !ok || !isGCPWebSecurityScannerScanConfigActionTail(tail, "start") {
		return false
	}
	scanConfigID, _, _ := parseGCPWebSecurityScannerScanConfigActionTail(tail)
	if isGCPWebSecurityScannerMissingID(scanConfigID) {
		respondGCPWebSecurityScannerNotFound(w, path, "scan_config not found")
		return true
	}

	if body, valid := decodeGCPWebSecurityScannerJSONBodyOptional(w, r, path); !valid {
		return true
	} else if rawName := strings.TrimSpace(gcpWebSecurityScannerString(body, "name")); rawName != "" {
		reqProject, reqScanConfigID, parsed := parseGCPWebSecurityScannerScanConfigName(rawName)
		if !parsed || reqProject != project || reqScanConfigID != scanConfigID {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "name must match requested scan config")
			return true
		}
	}

	respondJSON(w, http.StatusOK, gcpWebSecurityScannerScanRun(project, scanConfigID, "scan-run-1", "SCANNING", "RESULT_STATE_UNSPECIFIED"))
	return true
}

func handleGCPWebSecurityScannerGetScanRun(w http.ResponseWriter, path string) bool {
	project, tail, ok := parseGCPWebSecurityScannerProjectTail(path)
	if !ok || !isGCPWebSecurityScannerScanRunResourceTail(tail) {
		return false
	}
	scanConfigID := strings.TrimSpace(tail[1])
	scanRunID := strings.TrimSpace(tail[3])
	if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) {
		respondGCPWebSecurityScannerNotFound(w, path, "scan_run not found")
		return true
	}
	executionState, resultState := gcpWebSecurityScannerScanRunStates(scanRunID, false)
	respondJSON(w, http.StatusOK, gcpWebSecurityScannerScanRun(project, scanConfigID, scanRunID, executionState, resultState))
	return true
}

func handleGCPWebSecurityScannerListScanRuns(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPWebSecurityScannerProjectTail(path)
	if !ok || !isGCPWebSecurityScannerScanRunsCollectionTail(tail) {
		return false
	}

	scanConfigID := strings.TrimSpace(tail[1])
	if isGCPWebSecurityScannerMissingID(scanConfigID) {
		respondGCPWebSecurityScannerNotFound(w, path, "scan_config not found")
		return true
	}

	pageSize, start, valid := parseGCPWebSecurityScannerPagination(w, r, path, 1000)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpWebSecurityScannerScanRun(project, scanConfigID, "scan-run-1", "SCANNING", "RESULT_STATE_UNSPECIFIED"),
		gcpWebSecurityScannerScanRun(project, scanConfigID, "scan-run-finished-2", "FINISHED", "SUCCESS"),
	}
	return respondGCPWebSecurityScannerList(w, "scanRuns", items, pageSize, start, path)
}

func handleGCPWebSecurityScannerStopScanRun(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPWebSecurityScannerProjectTail(path)
	if !ok || !isGCPWebSecurityScannerScanRunActionTail(tail, "stop") {
		return false
	}

	scanConfigID, scanRunID, _, _ := parseGCPWebSecurityScannerScanRunActionTail(tail)
	if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) {
		respondGCPWebSecurityScannerNotFound(w, path, "scan_run not found")
		return true
	}

	if body, valid := decodeGCPWebSecurityScannerJSONBodyOptional(w, r, path); !valid {
		return true
	} else if rawName := strings.TrimSpace(gcpWebSecurityScannerString(body, "name")); rawName != "" {
		reqProject, reqScanConfigID, reqScanRunID, parsed := parseGCPWebSecurityScannerScanRunName(rawName)
		if !parsed || reqProject != project || reqScanConfigID != scanConfigID || reqScanRunID != scanRunID {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "name must match requested scan run")
			return true
		}
	}

	respondJSON(w, http.StatusOK, gcpWebSecurityScannerScanRun(project, scanConfigID, scanRunID, "FINISHED", "SUCCESS"))
	return true
}

func handleGCPWebSecurityScannerListCrawledURLs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPWebSecurityScannerProjectTail(path)
	if !ok || !isGCPWebSecurityScannerCrawledURLsCollectionTail(tail) {
		return false
	}
	scanConfigID := strings.TrimSpace(tail[1])
	scanRunID := strings.TrimSpace(tail[3])
	if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) {
		respondGCPWebSecurityScannerNotFound(w, path, "scan_run not found")
		return true
	}

	pageSize, start, valid := parseGCPWebSecurityScannerPagination(w, r, path, 1000)
	if !valid {
		return true
	}

	scanRunName := gcpWebSecurityScannerScanRunName(project, scanConfigID, scanRunID)
	items := []map[string]any{
		gcpWebSecurityScannerCrawledURL(scanRunName, 1),
		gcpWebSecurityScannerCrawledURL(scanRunName, 2),
	}
	return respondGCPWebSecurityScannerList(w, "crawledUrls", items, pageSize, start, path)
}

func handleGCPWebSecurityScannerGetFinding(w http.ResponseWriter, path string) bool {
	project, tail, ok := parseGCPWebSecurityScannerProjectTail(path)
	if !ok || !isGCPWebSecurityScannerFindingResourceTail(tail) {
		return false
	}
	scanConfigID := strings.TrimSpace(tail[1])
	scanRunID := strings.TrimSpace(tail[3])
	findingID := strings.TrimSpace(tail[5])
	if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) || isGCPWebSecurityScannerMissingID(findingID) {
		respondGCPWebSecurityScannerNotFound(w, path, "finding not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpWebSecurityScannerFinding(project, scanConfigID, scanRunID, findingID, "MIXED_CONTENT"))
	return true
}

func handleGCPWebSecurityScannerListFindings(w http.ResponseWriter, r *http.Request, path string) bool {
	project, tail, ok := parseGCPWebSecurityScannerProjectTail(path)
	if !ok || !isGCPWebSecurityScannerFindingsCollectionTail(tail) {
		return false
	}
	scanConfigID := strings.TrimSpace(tail[1])
	scanRunID := strings.TrimSpace(tail[3])
	if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) {
		respondGCPWebSecurityScannerNotFound(w, path, "scan_run not found")
		return true
	}

	filter, valid := validateGCPWebSecurityScannerFindingsFilter(w, path, strings.TrimSpace(r.URL.Query().Get("filter")))
	if !valid {
		return true
	}

	pageSize, start, valid := parseGCPWebSecurityScannerPagination(w, r, path, 1000)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpWebSecurityScannerFinding(project, scanConfigID, scanRunID, "finding-1", "MIXED_CONTENT"),
		gcpWebSecurityScannerFinding(project, scanConfigID, scanRunID, "finding-2", "OUTDATED_LIBRARY"),
	}
	if filter != "" {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if strings.EqualFold(strings.TrimSpace(gcpWebSecurityScannerString(item, "findingType")), filter) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	return respondGCPWebSecurityScannerList(w, "findings", items, pageSize, start, path)
}

func handleGCPWebSecurityScannerListFindingTypeStats(w http.ResponseWriter, path string) bool {
	_, tail, ok := parseGCPWebSecurityScannerProjectTail(path)
	if !ok || !isGCPWebSecurityScannerFindingTypeStatsTail(tail) {
		return false
	}
	scanConfigID := strings.TrimSpace(tail[1])
	scanRunID := strings.TrimSpace(tail[3])
	if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) {
		respondGCPWebSecurityScannerNotFound(w, path, "scan_run not found")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"findingTypeStats": []map[string]any{
			gcpWebSecurityScannerFindingTypeStat("MIXED_CONTENT", 1),
			gcpWebSecurityScannerFindingTypeStat("OUTDATED_LIBRARY", 1),
		},
	})
	return true
}

func handleGCPWebSecurityScannerGRPCBridge(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.Method != http.MethodPost {
		return false
	}
	method := strings.TrimPrefix(path, gcpWebSecurityScannerGRPCPathPrefix)
	if method == "" || strings.Contains(method, "/") {
		return false
	}

	body, valid := decodeGCPWebSecurityScannerJSONBodyOptional(w, r, path)
	if !valid {
		return true
	}

	switch method {
	case "CreateScanConfig":
		parent := strings.TrimSpace(gcpWebSecurityScannerString(body, "parent"))
		project, ok := parseGCPWebSecurityScannerProjectParent(parent)
		if !ok {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "parent is required")
			return true
		}
		scanConfig := gcpWebSecurityScannerScanConfigFromBridgeBody(body, "scanConfig", "scan_config")
		if len(scanConfig) == 0 {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "scanConfig is required")
			return true
		}
		if !validateGCPWebSecurityScannerScanConfig(w, path, scanConfig, false) {
			return true
		}
		scanConfigID := gcpWebSecurityScannerScanConfigIDFromMap(scanConfig, "scan-config-1")
		if isGCPWebSecurityScannerMissingID(scanConfigID) {
			respondGCPWebSecurityScannerAlreadyExists(w, path, "scan_config already exists")
			return true
		}
		resp := gcpWebSecurityScannerScanConfig(project, scanConfigID)
		applyGCPWebSecurityScannerScanConfigOverrides(resp, scanConfig)
		respondJSON(w, http.StatusOK, resp)
		return true
	case "DeleteScanConfig":
		project, scanConfigID, ok := parseGCPWebSecurityScannerScanConfigName(strings.TrimSpace(gcpWebSecurityScannerString(body, "name")))
		if !ok {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "name is required")
			return true
		}
		if project == "" || isGCPWebSecurityScannerMissingID(scanConfigID) {
			respondGCPWebSecurityScannerNotFound(w, path, "scan_config not found")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	case "GetScanConfig":
		project, scanConfigID, ok := parseGCPWebSecurityScannerScanConfigName(strings.TrimSpace(gcpWebSecurityScannerString(body, "name")))
		if !ok {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "name is required")
			return true
		}
		if project == "" || isGCPWebSecurityScannerMissingID(scanConfigID) {
			respondGCPWebSecurityScannerNotFound(w, path, "scan_config not found")
			return true
		}
		respondJSON(w, http.StatusOK, gcpWebSecurityScannerScanConfig(project, scanConfigID))
		return true
	case "ListScanConfigs":
		project, ok := parseGCPWebSecurityScannerProjectParent(strings.TrimSpace(gcpWebSecurityScannerString(body, "parent")))
		if !ok {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "parent is required")
			return true
		}
		pageSize, start, valid := parseGCPWebSecurityScannerBridgePagination(w, path, body, 1000)
		if !valid {
			return true
		}
		items := []map[string]any{
			gcpWebSecurityScannerScanConfig(project, "scan-config-1"),
			gcpWebSecurityScannerScanConfig(project, "scan-config-2"),
		}
		return respondGCPWebSecurityScannerList(w, "scanConfigs", items, pageSize, start, path)
	case "UpdateScanConfig":
		scanConfig := gcpWebSecurityScannerScanConfigFromBridgeBody(body, "scanConfig", "scan_config")
		if len(scanConfig) == 0 {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "scanConfig is required")
			return true
		}
		if !validateGCPWebSecurityScannerScanConfig(w, path, scanConfig, true) {
			return true
		}
		if !validateGCPWebSecurityScannerUpdateMaskFromBody(w, path, body) {
			return true
		}
		project, scanConfigID, ok := parseGCPWebSecurityScannerScanConfigName(strings.TrimSpace(gcpWebSecurityScannerString(scanConfig, "name")))
		if !ok {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "scanConfig.name is required")
			return true
		}
		if isGCPWebSecurityScannerMissingID(scanConfigID) {
			respondGCPWebSecurityScannerNotFound(w, path, "scan_config not found")
			return true
		}
		resp := gcpWebSecurityScannerScanConfig(project, scanConfigID)
		applyGCPWebSecurityScannerScanConfigOverrides(resp, scanConfig)
		respondJSON(w, http.StatusOK, resp)
		return true
	case "StartScanRun":
		project, scanConfigID, ok := parseGCPWebSecurityScannerScanConfigName(strings.TrimSpace(gcpWebSecurityScannerString(body, "name")))
		if !ok {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "name is required")
			return true
		}
		if isGCPWebSecurityScannerMissingID(scanConfigID) {
			respondGCPWebSecurityScannerNotFound(w, path, "scan_config not found")
			return true
		}
		respondJSON(w, http.StatusOK, gcpWebSecurityScannerScanRun(project, scanConfigID, "scan-run-1", "SCANNING", "RESULT_STATE_UNSPECIFIED"))
		return true
	case "GetScanRun":
		project, scanConfigID, scanRunID, ok := parseGCPWebSecurityScannerScanRunName(strings.TrimSpace(gcpWebSecurityScannerString(body, "name")))
		if !ok {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "name is required")
			return true
		}
		if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) {
			respondGCPWebSecurityScannerNotFound(w, path, "scan_run not found")
			return true
		}
		executionState, resultState := gcpWebSecurityScannerScanRunStates(scanRunID, false)
		respondJSON(w, http.StatusOK, gcpWebSecurityScannerScanRun(project, scanConfigID, scanRunID, executionState, resultState))
		return true
	case "ListScanRuns":
		project, scanConfigID, ok := parseGCPWebSecurityScannerScanConfigName(strings.TrimSpace(gcpWebSecurityScannerString(body, "parent")))
		if !ok {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "parent is required")
			return true
		}
		if isGCPWebSecurityScannerMissingID(scanConfigID) {
			respondGCPWebSecurityScannerNotFound(w, path, "scan_config not found")
			return true
		}
		pageSize, start, valid := parseGCPWebSecurityScannerBridgePagination(w, path, body, 1000)
		if !valid {
			return true
		}
		items := []map[string]any{
			gcpWebSecurityScannerScanRun(project, scanConfigID, "scan-run-1", "SCANNING", "RESULT_STATE_UNSPECIFIED"),
			gcpWebSecurityScannerScanRun(project, scanConfigID, "scan-run-finished-2", "FINISHED", "SUCCESS"),
		}
		return respondGCPWebSecurityScannerList(w, "scanRuns", items, pageSize, start, path)
	case "StopScanRun":
		project, scanConfigID, scanRunID, ok := parseGCPWebSecurityScannerScanRunName(strings.TrimSpace(gcpWebSecurityScannerString(body, "name")))
		if !ok {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "name is required")
			return true
		}
		if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) {
			respondGCPWebSecurityScannerNotFound(w, path, "scan_run not found")
			return true
		}
		respondJSON(w, http.StatusOK, gcpWebSecurityScannerScanRun(project, scanConfigID, scanRunID, "FINISHED", "SUCCESS"))
		return true
	case "ListCrawledUrls":
		project, scanConfigID, scanRunID, ok := parseGCPWebSecurityScannerScanRunName(strings.TrimSpace(gcpWebSecurityScannerString(body, "parent")))
		if !ok {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "parent is required")
			return true
		}
		if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) {
			respondGCPWebSecurityScannerNotFound(w, path, "scan_run not found")
			return true
		}
		pageSize, start, valid := parseGCPWebSecurityScannerBridgePagination(w, path, body, 1000)
		if !valid {
			return true
		}
		scanRunName := gcpWebSecurityScannerScanRunName(project, scanConfigID, scanRunID)
		items := []map[string]any{
			gcpWebSecurityScannerCrawledURL(scanRunName, 1),
			gcpWebSecurityScannerCrawledURL(scanRunName, 2),
		}
		return respondGCPWebSecurityScannerList(w, "crawledUrls", items, pageSize, start, path)
	case "GetFinding":
		project, scanConfigID, scanRunID, findingID, ok := parseGCPWebSecurityScannerFindingName(strings.TrimSpace(gcpWebSecurityScannerString(body, "name")))
		if !ok {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "name is required")
			return true
		}
		if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) || isGCPWebSecurityScannerMissingID(findingID) {
			respondGCPWebSecurityScannerNotFound(w, path, "finding not found")
			return true
		}
		respondJSON(w, http.StatusOK, gcpWebSecurityScannerFinding(project, scanConfigID, scanRunID, findingID, "MIXED_CONTENT"))
		return true
	case "ListFindings":
		project, scanConfigID, scanRunID, ok := parseGCPWebSecurityScannerScanRunName(strings.TrimSpace(gcpWebSecurityScannerString(body, "parent")))
		if !ok {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "parent is required")
			return true
		}
		if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) {
			respondGCPWebSecurityScannerNotFound(w, path, "scan_run not found")
			return true
		}
		filter, valid := validateGCPWebSecurityScannerFindingsFilter(w, path, strings.TrimSpace(gcpWebSecurityScannerString(body, "filter")))
		if !valid {
			return true
		}
		pageSize, start, valid := parseGCPWebSecurityScannerBridgePagination(w, path, body, 1000)
		if !valid {
			return true
		}
		items := []map[string]any{
			gcpWebSecurityScannerFinding(project, scanConfigID, scanRunID, "finding-1", "MIXED_CONTENT"),
			gcpWebSecurityScannerFinding(project, scanConfigID, scanRunID, "finding-2", "OUTDATED_LIBRARY"),
		}
		if filter != "" {
			filtered := make([]map[string]any, 0, len(items))
			for _, item := range items {
				if strings.EqualFold(strings.TrimSpace(gcpWebSecurityScannerString(item, "findingType")), filter) {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}
		return respondGCPWebSecurityScannerList(w, "findings", items, pageSize, start, path)
	case "ListFindingTypeStats":
		_, scanConfigID, scanRunID, ok := parseGCPWebSecurityScannerScanRunName(strings.TrimSpace(gcpWebSecurityScannerString(body, "parent")))
		if !ok {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "parent is required")
			return true
		}
		if isGCPWebSecurityScannerMissingID(scanConfigID) || isGCPWebSecurityScannerMissingID(scanRunID) {
			respondGCPWebSecurityScannerNotFound(w, path, "scan_run not found")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"findingTypeStats": []map[string]any{
				gcpWebSecurityScannerFindingTypeStat("MIXED_CONTENT", 1),
				gcpWebSecurityScannerFindingTypeStat("OUTDATED_LIBRARY", 1),
			},
		})
		return true
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func decodeGCPWebSecurityScannerJSONBodyOptional(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPWebSecurityScannerInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPWebSecurityScannerInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func decodeGCPWebSecurityScannerJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, ok := decodeGCPWebSecurityScannerJSONBodyOptional(w, r, path)
	if !ok {
		return nil, false
	}
	if len(body) == 0 {
		respondGCPWebSecurityScannerInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func parseGCPWebSecurityScannerPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0

	pageSizeRaw := strings.TrimSpace(r.URL.Query().Get("pageSize"))
	if pageSizeRaw == "" {
		pageSizeRaw = strings.TrimSpace(r.URL.Query().Get("page_size"))
	}
	if pageSizeRaw != "" {
		n, err := strconv.Atoi(pageSizeRaw)
		if err != nil || n < 0 || n > maxPageSize {
			respondGCPWebSecurityScannerInvalidArgument(w, path, fmt.Sprintf("pageSize must be a non-negative integer <= %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = n
	}

	pageTokenRaw := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageTokenRaw == "" {
		pageTokenRaw = strings.TrimSpace(r.URL.Query().Get("page_token"))
	}
	if pageTokenRaw != "" {
		n, err := strconv.Atoi(pageTokenRaw)
		if err != nil || n < 0 {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = n
	}

	return pageSize, start, true
}

func parseGCPWebSecurityScannerBridgePagination(w http.ResponseWriter, path string, body map[string]any, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0

	if raw, exists := gcpWebSecurityScannerLookup(body, "pageSize", "page_size"); exists {
		n, valid := gcpWebSecurityScannerNumberToInt(raw)
		if !valid || n < 0 || n > maxPageSize {
			respondGCPWebSecurityScannerInvalidArgument(w, path, fmt.Sprintf("pageSize must be a non-negative integer <= %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = n
	}

	if raw, exists := gcpWebSecurityScannerLookup(body, "pageToken", "page_token"); exists {
		token := strings.TrimSpace(gcpWebSecurityScannerToString(raw))
		if token != "" {
			n, err := strconv.Atoi(token)
			if err != nil || n < 0 {
				respondGCPWebSecurityScannerInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
				return 0, 0, false
			}
			start = n
		}
	}

	return pageSize, start, true
}

func respondGCPWebSecurityScannerList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPWebSecurityScannerOutOfRange(w, path, "pageToken is out of range")
		return true
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
		field:           items[start:end],
		"nextPageToken": next,
	})
	return true
}

func gcpWebSecurityScannerScanConfigFromCreateBody(body map[string]any) map[string]any {
	if scanConfig := gcpWebSecurityScannerScanConfigFromBridgeBody(body, "scanConfig", "scan_config"); len(scanConfig) > 0 {
		return scanConfig
	}
	return body
}

func gcpWebSecurityScannerScanConfigFromUpdateBody(body map[string]any) map[string]any {
	if scanConfig := gcpWebSecurityScannerScanConfigFromBridgeBody(body, "scanConfig", "scan_config"); len(scanConfig) > 0 {
		return scanConfig
	}
	return body
}

func gcpWebSecurityScannerScanConfigFromBridgeBody(body map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if nested, ok := body[key].(map[string]any); ok && len(nested) > 0 {
			return nested
		}
	}
	return nil
}

func validateGCPWebSecurityScannerScanConfig(w http.ResponseWriter, path string, scanConfig map[string]any, requireName bool) bool {
	name := strings.TrimSpace(gcpWebSecurityScannerString(scanConfig, "name"))
	if requireName && name == "" {
		respondGCPWebSecurityScannerInvalidArgument(w, path, "scanConfig.name is required")
		return false
	}
	if name != "" {
		if _, _, ok := parseGCPWebSecurityScannerScanConfigName(name); !ok {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "scanConfig.name is invalid")
			return false
		}
	}

	if strings.TrimSpace(gcpWebSecurityScannerString(scanConfig, "displayName", "display_name")) == "" {
		respondGCPWebSecurityScannerInvalidArgument(w, path, "scanConfig.displayName is required")
		return false
	}

	startingURLs := gcpWebSecurityScannerStringSlice(scanConfig, "startingUrls", "starting_urls")
	if len(startingURLs) == 0 {
		respondGCPWebSecurityScannerInvalidArgument(w, path, "scanConfig.startingUrls must include at least one URL")
		return false
	}
	for idx, startingURL := range startingURLs {
		if !isGCPWebSecurityScannerURI(startingURL) {
			respondGCPWebSecurityScannerInvalidArgument(w, path, fmt.Sprintf("scanConfig.startingUrls[%d] must be an absolute http(s) URI", idx))
			return false
		}
	}

	if maxQPS, present, ok := gcpWebSecurityScannerOptionalInt(scanConfig, "maxQps", "max_qps"); !ok {
		respondGCPWebSecurityScannerInvalidArgument(w, path, "scanConfig.maxQps must be an integer")
		return false
	} else if present && maxQPS != 0 && (maxQPS < 5 || maxQPS > 20) {
		respondGCPWebSecurityScannerInvalidArgument(w, path, "scanConfig.maxQps must be 0 or between 5 and 20")
		return false
	}

	if rawSchedule, exists := gcpWebSecurityScannerLookup(scanConfig, "schedule"); exists {
		if _, ok := rawSchedule.(map[string]any); !ok {
			respondGCPWebSecurityScannerInvalidArgument(w, path, "scanConfig.schedule must be an object")
			return false
		}
	}

	return true
}

func validateGCPWebSecurityScannerUpdateMask(w http.ResponseWriter, path string, r *http.Request, body map[string]any) bool {
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		updateMask = strings.TrimSpace(r.URL.Query().Get("update_mask"))
	}
	if updateMask != "" {
		return true
	}
	return validateGCPWebSecurityScannerUpdateMaskFromBody(w, path, body)
}

func validateGCPWebSecurityScannerUpdateMaskFromBody(w http.ResponseWriter, path string, body map[string]any) bool {
	if raw, exists := gcpWebSecurityScannerLookup(body, "updateMask", "update_mask"); exists {
		switch v := raw.(type) {
		case string:
			if strings.TrimSpace(v) == "" {
				respondGCPWebSecurityScannerInvalidArgument(w, path, "updateMask is required")
				return false
			}
			return true
		case map[string]any:
			rawPaths, ok := gcpWebSecurityScannerLookup(v, "paths")
			if !ok {
				respondGCPWebSecurityScannerInvalidArgument(w, path, "updateMask.paths is required")
				return false
			}
			paths := gcpWebSecurityScannerStringSlice(map[string]any{"paths": rawPaths}, "paths")
			if len(paths) == 0 {
				respondGCPWebSecurityScannerInvalidArgument(w, path, "updateMask.paths must include at least one field")
				return false
			}
			return true
		default:
			respondGCPWebSecurityScannerInvalidArgument(w, path, "updateMask must be a string or object")
			return false
		}
	}
	respondGCPWebSecurityScannerInvalidArgument(w, path, "updateMask is required")
	return false
}

func validateGCPWebSecurityScannerFindingsFilter(w http.ResponseWriter, path, rawFilter string) (string, bool) {
	rawFilter = strings.TrimSpace(rawFilter)
	if rawFilter == "" {
		return "", true
	}
	matches := gcpWebSecurityScannerFilterRe.FindStringSubmatch(rawFilter)
	if len(matches) != 2 {
		respondGCPWebSecurityScannerInvalidArgument(w, path, "filter must use finding_type=<VALUE> format")
		return "", false
	}
	return strings.ToUpper(strings.TrimSpace(matches[1])), true
}

func isGCPWebSecurityScannerURI(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return strings.TrimSpace(parsed.Host) != ""
}

func gcpWebSecurityScannerString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if raw, ok := m[key]; ok {
			return strings.TrimSpace(gcpWebSecurityScannerToString(raw))
		}
	}
	return ""
}

func gcpWebSecurityScannerToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case float64:
		if v == float64(int(v)) {
			return strconv.Itoa(int(v))
		}
		return fmt.Sprintf("%f", v)
	case int:
		return strconv.Itoa(v)
	case int32:
		return strconv.Itoa(int(v))
	case int64:
		return strconv.FormatInt(v, 10)
	default:
		return ""
	}
}

func gcpWebSecurityScannerLookup(m map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if raw, ok := m[key]; ok {
			return raw, true
		}
	}
	return nil, false
}

func gcpWebSecurityScannerNumberToInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		if v != float64(int(v)) {
			return 0, false
		}
		return int(v), true
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func gcpWebSecurityScannerOptionalInt(m map[string]any, keys ...string) (value int, present bool, ok bool) {
	raw, exists := gcpWebSecurityScannerLookup(m, keys...)
	if !exists {
		return 0, false, true
	}
	n, valid := gcpWebSecurityScannerNumberToInt(raw)
	if !valid {
		return 0, true, false
	}
	return n, true, true
}

func gcpWebSecurityScannerStringSlice(m map[string]any, keys ...string) []string {
	raw, exists := gcpWebSecurityScannerLookup(m, keys...)
	if !exists {
		return nil
	}
	switch v := raw.(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s := strings.TrimSpace(gcpWebSecurityScannerToString(item))
			if s == "" {
				continue
			}
			out = append(out, s)
		}
		return out
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s := strings.TrimSpace(item)
			if s == "" {
				continue
			}
			out = append(out, s)
		}
		return out
	default:
		return nil
	}
}

func gcpWebSecurityScannerScanConfigIDFromMap(scanConfig map[string]any, defaultID string) string {
	name := strings.TrimSpace(gcpWebSecurityScannerString(scanConfig, "name"))
	if name != "" {
		if _, scanConfigID, ok := parseGCPWebSecurityScannerScanConfigName(name); ok {
			return scanConfigID
		}
	}
	return defaultID
}

func gcpWebSecurityScannerScanConfigName(project, scanConfigID string) string {
	return fmt.Sprintf("projects/%s/scanConfigs/%s", project, scanConfigID)
}

func gcpWebSecurityScannerScanRunName(project, scanConfigID, scanRunID string) string {
	return fmt.Sprintf("projects/%s/scanConfigs/%s/scanRuns/%s", project, scanConfigID, scanRunID)
}

func gcpWebSecurityScannerFindingName(project, scanConfigID, scanRunID, findingID string) string {
	return fmt.Sprintf("projects/%s/scanConfigs/%s/scanRuns/%s/findings/%s", project, scanConfigID, scanRunID, findingID)
}

func gcpWebSecurityScannerScanConfig(project, scanConfigID string) map[string]any {
	return map[string]any{
		"name":         gcpWebSecurityScannerScanConfigName(project, scanConfigID),
		"displayName":  "Stackyard Scan Config " + scanConfigID,
		"maxQps":       15,
		"startingUrls": []string{fmt.Sprintf("https://%s.stackyard.test", scanConfigID)},
		"userAgent":    "CHROME_LINUX",
		"riskLevel":    "NORMAL",
		"schedule": map[string]any{
			"scheduleTime":      gcpWebSecurityScannerReferenceTime.Format(time.RFC3339),
			"intervalDuration":  "86400s",
			"scheduleTimezone":  "UTC",
			"disabled":          false,
			"nextRunTime":       gcpWebSecurityScannerReferenceTime.Add(24 * time.Hour).Format(time.RFC3339),
			"scheduleInterval":  "DAILY",
			"rescheduleOnError": true,
		},
		"managedScan":            false,
		"staticIpScan":           false,
		"ignoreHttpStatusErrors": false,
	}
}

func applyGCPWebSecurityScannerScanConfigOverrides(out, in map[string]any) {
	for _, key := range []string{
		"displayName",
		"display_name",
		"startingUrls",
		"starting_urls",
		"maxQps",
		"max_qps",
		"userAgent",
		"user_agent",
		"blacklistPatterns",
		"blacklist_patterns",
		"schedule",
		"riskLevel",
		"risk_level",
		"staticIpScan",
		"static_ip_scan",
		"ignoreHttpStatusErrors",
		"ignore_http_status_errors",
	} {
		if raw, ok := in[key]; ok {
			normalized := key
			switch key {
			case "display_name":
				normalized = "displayName"
			case "starting_urls":
				normalized = "startingUrls"
			case "max_qps":
				normalized = "maxQps"
			case "user_agent":
				normalized = "userAgent"
			case "blacklist_patterns":
				normalized = "blacklistPatterns"
			case "risk_level":
				normalized = "riskLevel"
			case "static_ip_scan":
				normalized = "staticIpScan"
			case "ignore_http_status_errors":
				normalized = "ignoreHttpStatusErrors"
			}
			out[normalized] = raw
		}
	}
}

func gcpWebSecurityScannerScanRun(project, scanConfigID, scanRunID, executionState, resultState string) map[string]any {
	return map[string]any{
		"name":               gcpWebSecurityScannerScanRunName(project, scanConfigID, scanRunID),
		"executionState":     executionState,
		"resultState":        resultState,
		"startTime":          gcpWebSecurityScannerReferenceTime.Add(5 * time.Minute).Format(time.RFC3339),
		"endTime":            gcpWebSecurityScannerReferenceTime.Add(8 * time.Minute).Format(time.RFC3339),
		"urlsCrawledCount":   12,
		"urlsTestedCount":    36,
		"hasVulnerabilities": true,
		"progressPercent":    gcpWebSecurityScannerProgressForExecutionState(executionState),
	}
}

func gcpWebSecurityScannerScanRunStates(scanRunID string, stopped bool) (executionState, resultState string) {
	if stopped {
		return "FINISHED", "SUCCESS"
	}
	normalized := strings.ToLower(strings.TrimSpace(scanRunID))
	if strings.Contains(normalized, "finished") || strings.Contains(normalized, "done") {
		return "FINISHED", "SUCCESS"
	}
	return "SCANNING", "RESULT_STATE_UNSPECIFIED"
}

func gcpWebSecurityScannerProgressForExecutionState(executionState string) int {
	switch strings.ToUpper(strings.TrimSpace(executionState)) {
	case "FINISHED":
		return 100
	case "QUEUED":
		return 0
	default:
		return 45
	}
}

func gcpWebSecurityScannerCrawledURL(scanRunName string, index int) map[string]any {
	return map[string]any{
		"httpMethod": "GET",
		"url":        fmt.Sprintf("https://scan-%s.stackyard.test/page-%d", strings.ReplaceAll(scanRunName, "/", "-"), index),
		"body":       "",
	}
}

func gcpWebSecurityScannerFinding(project, scanConfigID, scanRunID, findingID, findingType string) map[string]any {
	return map[string]any{
		"name":            gcpWebSecurityScannerFindingName(project, scanConfigID, scanRunID, findingID),
		"findingType":     strings.ToUpper(strings.TrimSpace(findingType)),
		"severity":        "MEDIUM",
		"httpMethod":      "GET",
		"fuzzedUrl":       fmt.Sprintf("https://%s.stackyard.test/vuln/%s", scanConfigID, findingID),
		"description":     "Stackyard staged finding fixture",
		"reproductionUrl": fmt.Sprintf("https://%s.stackyard.test/repro/%s", scanConfigID, findingID),
		"frameUrl":        fmt.Sprintf("https://%s.stackyard.test/frame", scanConfigID),
		"finalUrl":        fmt.Sprintf("https://%s.stackyard.test/final", scanConfigID),
		"trackingId":      fmt.Sprintf("tracking-%s", findingID),
	}
}

func gcpWebSecurityScannerFindingTypeStat(findingType string, count int) map[string]any {
	return map[string]any{
		"findingType":  strings.ToUpper(strings.TrimSpace(findingType)),
		"findingCount": count,
	}
}

func parseGCPWebSecurityScannerProjectTail(path string) (project string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" {
		return "", nil, false
	}
	return project, parts[4:], true
}

func parseGCPWebSecurityScannerProjectResourceName(name string) (project string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) < 2 || parts[0] != "projects" {
		return "", nil, false
	}
	project = strings.TrimSpace(parts[1])
	if project == "" {
		return "", nil, false
	}
	return project, parts[2:], true
}

func parseGCPWebSecurityScannerProjectParent(parent string) (project string, ok bool) {
	project, tail, parsed := parseGCPWebSecurityScannerProjectResourceName(strings.TrimSpace(parent))
	if !parsed || len(tail) != 0 {
		return "", false
	}
	return project, true
}

func parseGCPWebSecurityScannerScanConfigName(name string) (project, scanConfigID string, ok bool) {
	project, tail, parsed := parseGCPWebSecurityScannerProjectResourceName(strings.TrimSpace(name))
	if !parsed || len(tail) != 2 || tail[0] != "scanConfigs" {
		return "", "", false
	}
	scanConfigID = strings.TrimSpace(tail[1])
	if scanConfigID == "" {
		return "", "", false
	}
	return project, scanConfigID, true
}

func parseGCPWebSecurityScannerScanRunParent(parent string) (project, scanConfigID string, ok bool) {
	project, tail, parsed := parseGCPWebSecurityScannerProjectResourceName(strings.TrimSpace(parent))
	if !parsed || len(tail) != 3 || tail[0] != "scanConfigs" || tail[2] != "scanRuns" {
		return "", "", false
	}
	scanConfigID = strings.TrimSpace(tail[1])
	if scanConfigID == "" {
		return "", "", false
	}
	return project, scanConfigID, true
}

func parseGCPWebSecurityScannerScanRunName(name string) (project, scanConfigID, scanRunID string, ok bool) {
	project, tail, parsed := parseGCPWebSecurityScannerProjectResourceName(strings.TrimSpace(name))
	if !parsed || len(tail) != 4 || tail[0] != "scanConfigs" || tail[2] != "scanRuns" {
		return "", "", "", false
	}
	scanConfigID = strings.TrimSpace(tail[1])
	scanRunID = strings.TrimSpace(tail[3])
	if scanConfigID == "" || scanRunID == "" {
		return "", "", "", false
	}
	return project, scanConfigID, scanRunID, true
}

func parseGCPWebSecurityScannerFindingParent(parent string) (project, scanConfigID, scanRunID string, ok bool) {
	project, tail, parsed := parseGCPWebSecurityScannerProjectResourceName(strings.TrimSpace(parent))
	if !parsed || len(tail) != 5 || tail[0] != "scanConfigs" || tail[2] != "scanRuns" {
		return "", "", "", false
	}
	scanConfigID = strings.TrimSpace(tail[1])
	scanRunID = strings.TrimSpace(tail[3])
	resource := strings.TrimSpace(tail[4])
	if (resource != "findings" && resource != "crawledUrls" && resource != "findingTypeStats") || scanConfigID == "" || scanRunID == "" {
		return "", "", "", false
	}
	return project, scanConfigID, scanRunID, true
}

func parseGCPWebSecurityScannerFindingName(name string) (project, scanConfigID, scanRunID, findingID string, ok bool) {
	project, tail, parsed := parseGCPWebSecurityScannerProjectResourceName(strings.TrimSpace(name))
	if !parsed || len(tail) != 6 || tail[0] != "scanConfigs" || tail[2] != "scanRuns" || tail[4] != "findings" {
		return "", "", "", "", false
	}
	scanConfigID = strings.TrimSpace(tail[1])
	scanRunID = strings.TrimSpace(tail[3])
	findingID = strings.TrimSpace(tail[5])
	if scanConfigID == "" || scanRunID == "" || findingID == "" {
		return "", "", "", "", false
	}
	return project, scanConfigID, scanRunID, findingID, true
}

func isGCPWebSecurityScannerScanConfigsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "scanConfigs"
}

func isGCPWebSecurityScannerScanConfigResourceTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "scanConfigs" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPWebSecurityScannerScanConfigActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "scanConfigs" {
		return false
	}
	_, parsedAction, found := parseGCPWebSecurityScannerResourceActionSegment(tail[1])
	return found && parsedAction == action
}

func parseGCPWebSecurityScannerScanConfigActionTail(tail []string) (scanConfigID, action string, ok bool) {
	if len(tail) != 2 || tail[0] != "scanConfigs" {
		return "", "", false
	}
	scanConfigID, action, ok = parseGCPWebSecurityScannerResourceActionSegment(tail[1])
	return scanConfigID, action, ok
}

func isGCPWebSecurityScannerScanRunsCollectionTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "scanConfigs" && strings.TrimSpace(tail[1]) != "" && tail[2] == "scanRuns"
}

func isGCPWebSecurityScannerScanRunResourceTail(tail []string) bool {
	return len(tail) == 4 &&
		tail[0] == "scanConfigs" &&
		strings.TrimSpace(tail[1]) != "" &&
		tail[2] == "scanRuns" &&
		strings.TrimSpace(tail[3]) != "" &&
		!strings.Contains(tail[3], ":")
}

func isGCPWebSecurityScannerScanRunActionTail(tail []string, action string) bool {
	if len(tail) != 4 || tail[0] != "scanConfigs" || strings.TrimSpace(tail[1]) == "" || tail[2] != "scanRuns" {
		return false
	}
	_, parsedAction, found := parseGCPWebSecurityScannerResourceActionSegment(tail[3])
	return found && parsedAction == action
}

func parseGCPWebSecurityScannerScanRunActionTail(tail []string) (scanConfigID, scanRunID, action string, ok bool) {
	if len(tail) != 4 || tail[0] != "scanConfigs" || tail[2] != "scanRuns" {
		return "", "", "", false
	}
	scanConfigID = strings.TrimSpace(tail[1])
	scanRunID, action, found := parseGCPWebSecurityScannerResourceActionSegment(tail[3])
	if scanConfigID == "" || scanRunID == "" || !found {
		return "", "", "", false
	}
	return scanConfigID, scanRunID, action, true
}

func isGCPWebSecurityScannerCrawledURLsCollectionTail(tail []string) bool {
	return len(tail) == 5 &&
		tail[0] == "scanConfigs" &&
		strings.TrimSpace(tail[1]) != "" &&
		tail[2] == "scanRuns" &&
		strings.TrimSpace(tail[3]) != "" &&
		tail[4] == "crawledUrls"
}

func isGCPWebSecurityScannerFindingsCollectionTail(tail []string) bool {
	return len(tail) == 5 &&
		tail[0] == "scanConfigs" &&
		strings.TrimSpace(tail[1]) != "" &&
		tail[2] == "scanRuns" &&
		strings.TrimSpace(tail[3]) != "" &&
		tail[4] == "findings"
}

func isGCPWebSecurityScannerFindingTypeStatsTail(tail []string) bool {
	return len(tail) == 5 &&
		tail[0] == "scanConfigs" &&
		strings.TrimSpace(tail[1]) != "" &&
		tail[2] == "scanRuns" &&
		strings.TrimSpace(tail[3]) != "" &&
		tail[4] == "findingTypeStats"
}

func isGCPWebSecurityScannerFindingResourceTail(tail []string) bool {
	return len(tail) == 6 &&
		tail[0] == "scanConfigs" &&
		strings.TrimSpace(tail[1]) != "" &&
		tail[2] == "scanRuns" &&
		strings.TrimSpace(tail[3]) != "" &&
		tail[4] == "findings" &&
		strings.TrimSpace(tail[5]) != "" &&
		!strings.Contains(tail[5], ":")
}

func parseGCPWebSecurityScannerResourceActionSegment(segment string) (resource, action string, hasAction bool) {
	parts := strings.SplitN(strings.TrimSpace(segment), ":", 2)
	resource = strings.TrimSpace(parts[0])
	if resource == "" || len(parts) != 2 {
		return resource, "", false
	}
	action = strings.TrimSpace(parts[1])
	if action == "" {
		return resource, "", false
	}
	return resource, action, true
}

func isGCPWebSecurityScannerMissingID(id string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	return strings.Contains(id, "missing") || strings.Contains(id, "notfound")
}

func respondGCPWebSecurityScannerInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPWebSecurityScannerError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPWebSecurityScannerOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPWebSecurityScannerError(w, http.StatusBadRequest, "OutOfRange", path, message)
}

func respondGCPWebSecurityScannerNotFound(w http.ResponseWriter, path, message string) {
	respondGCPWebSecurityScannerError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPWebSecurityScannerAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPWebSecurityScannerError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPWebSecurityScannerFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPWebSecurityScannerError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPWebSecurityScannerError(w http.ResponseWriter, status int, errType, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    errType,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_websecurityscanner(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "websecurityscanner") {
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
			"name":     "projects/stackyard/scanConfigs/scan-config-1",
			"service":  "websecurityscanner",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
