package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

var gcpChronicleSupportedLocations = map[string]struct{}{
	"us": {},
	"eu": {},
}

func (s *Server) handleGCPChronicleRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if strings.HasPrefix(path, "/gcp/google.cloud.chronicle.v1.") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}
	if !isGCPChroniclePath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPChronicleGetInstance(w, path) {
			return true
		}
		if handleGCPChronicleListDataAccessLabels(w, r, path) {
			return true
		}
		if handleGCPChronicleGetDataAccessLabel(w, path) {
			return true
		}
		if handleGCPChronicleListDataAccessScopes(w, r, path) {
			return true
		}
		if handleGCPChronicleGetDataAccessScope(w, path) {
			return true
		}
		if handleGCPChronicleListReferenceLists(w, r, path) {
			return true
		}
		if handleGCPChronicleGetReferenceList(w, path) {
			return true
		}
		if handleGCPChronicleListRules(w, r, path) {
			return true
		}
		if handleGCPChronicleGetRule(w, path) {
			return true
		}
		if handleGCPChronicleListRuleRevisions(w, r, path) {
			return true
		}
		if handleGCPChronicleListRuleDeployments(w, r, path) {
			return true
		}
		if handleGCPChronicleGetRuleDeployment(w, path) {
			return true
		}
		if handleGCPChronicleListRetrohunts(w, r, path) {
			return true
		}
		if handleGCPChronicleGetRetrohunt(w, path) {
			return true
		}
		if handleGCPChronicleListWatchlists(w, r, path) {
			return true
		}
		if handleGCPChronicleGetWatchlist(w, path) {
			return true
		}
		if handleGCPChronicleListOperations(w, r, path) {
			return true
		}
		if handleGCPChronicleGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost, http.MethodPatch, http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPChroniclePath(path string) bool {
	if !strings.HasPrefix(path, "/gcp/v1/projects/") || !strings.Contains(path, "/locations/") {
		return false
	}
	_, _, _, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok {
		return false
	}
	if len(tail) == 0 {
		return true
	}
	afterInstance := strings.Join(tail, "/")
	if afterInstance == "" {
		return false
	}
	if !strings.ContainsAny(afterInstance, "/:") {
		return true
	}
	markers := []string{
		"/dataAccessLabels",
		"/dataAccessScopes",
		"/referenceLists",
		"/rules",
		"/retrohunts",
		"/deployments",
		"/watchlists",
		"/operations",
		":listRevisions",
		":cancel",
	}
	for _, marker := range markers {
		if strings.Contains(path, marker) {
			return true
		}
	}
	return false
}

func handleGCPChronicleGetInstance(w http.ResponseWriter, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 0 {
		return false
	}
	respondJSON(w, http.StatusOK, gcpChronicleInstance(project, location, instance))
	return true
}

func handleGCPChronicleListDataAccessLabels(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 1 || tail[0] != "dataAccessLabels" {
		return false
	}
	pageSize, start, valid := parseGCPChroniclePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpChronicleDataAccessLabel(project, location, instance, "team-label")}
	return respondGCPChronicleList(w, "dataAccessLabels", items, pageSize, start, path)
}

func handleGCPChronicleGetDataAccessLabel(w http.ResponseWriter, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 2 || tail[0] != "dataAccessLabels" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpChronicleDataAccessLabel(project, location, instance, tail[1]))
	return true
}

func handleGCPChronicleListDataAccessScopes(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 1 || tail[0] != "dataAccessScopes" {
		return false
	}
	pageSize, start, valid := parseGCPChroniclePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpChronicleDataAccessScope(project, location, instance, "team-scope")}
	return respondGCPChronicleList(w, "dataAccessScopes", items, pageSize, start, path)
}

func handleGCPChronicleGetDataAccessScope(w http.ResponseWriter, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 2 || tail[0] != "dataAccessScopes" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpChronicleDataAccessScope(project, location, instance, tail[1]))
	return true
}

func handleGCPChronicleListReferenceLists(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 1 || tail[0] != "referenceLists" {
		return false
	}
	pageSize, start, valid := parseGCPChroniclePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpChronicleReferenceList(project, location, instance, "team-reference-list")}
	return respondGCPChronicleList(w, "referenceLists", items, pageSize, start, path)
}

func handleGCPChronicleGetReferenceList(w http.ResponseWriter, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 2 || tail[0] != "referenceLists" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpChronicleReferenceList(project, location, instance, tail[1]))
	return true
}

func handleGCPChronicleListRules(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 1 || tail[0] != "rules" {
		return false
	}
	pageSize, start, valid := parseGCPChroniclePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpChronicleRule(project, location, instance, "team-rule")}
	return respondGCPChronicleList(w, "rules", items, pageSize, start, path)
}

func handleGCPChronicleGetRule(w http.ResponseWriter, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 2 || tail[0] != "rules" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpChronicleRule(project, location, instance, tail[1]))
	return true
}

func handleGCPChronicleListRuleRevisions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 2 || tail[0] != "rules" {
		return false
	}
	ruleID, action, found := strings.Cut(normalizeGCPChronicleActionSegment(tail[1]), ":")
	if !found || action != "listRevisions" || strings.TrimSpace(ruleID) == "" {
		return false
	}
	pageSize, start, valid := parseGCPChroniclePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		{
			"name":     fmt.Sprintf("projects/%s/locations/%s/instances/%s/rules/%s/revisions/r1", project, location, instance, strings.TrimSpace(ruleID)),
			"revision": "r1",
			"state":    "LIVE",
		},
	}
	return respondGCPChronicleList(w, "ruleRevisions", items, pageSize, start, path)
}

func handleGCPChronicleListRuleDeployments(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok {
		return false
	}
	if len(tail) == 3 && tail[0] == "rules" && tail[2] == "deployments" {
		pageSize, start, valid := parseGCPChroniclePagination(w, r, path)
		if !valid {
			return true
		}
		items := []map[string]any{gcpChronicleRuleDeployment(project, location, instance, "team-rule")}
		return respondGCPChronicleList(w, "ruleDeployments", items, pageSize, start, path)
	}
	return false
}

func handleGCPChronicleGetRuleDeployment(w http.ResponseWriter, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 3 || tail[0] != "rules" || strings.TrimSpace(tail[1]) == "" || tail[2] != "deployment" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpChronicleRuleDeployment(project, location, instance, tail[1]))
	return true
}

func handleGCPChronicleListRetrohunts(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 3 || tail[0] != "rules" || strings.TrimSpace(tail[1]) == "" || tail[2] != "retrohunts" {
		return false
	}
	pageSize, start, valid := parseGCPChroniclePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpChronicleRetrohunt(project, location, instance, tail[1], "team-retrohunt")}
	return respondGCPChronicleList(w, "retrohunts", items, pageSize, start, path)
}

func handleGCPChronicleGetRetrohunt(w http.ResponseWriter, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 4 || tail[0] != "rules" || strings.TrimSpace(tail[1]) == "" || tail[2] != "retrohunts" || strings.TrimSpace(tail[3]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpChronicleRetrohunt(project, location, instance, tail[1], tail[3]))
	return true
}

func handleGCPChronicleListWatchlists(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 1 || tail[0] != "watchlists" {
		return false
	}
	pageSize, start, valid := parseGCPChroniclePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpChronicleWatchlist(project, location, instance, "high-risk-entities")}
	return respondGCPChronicleList(w, "watchlists", items, pageSize, start, path)
}

func handleGCPChronicleGetWatchlist(w http.ResponseWriter, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 2 || tail[0] != "watchlists" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpChronicleWatchlist(project, location, instance, tail[1]))
	return true
}

func handleGCPChronicleListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 1 || tail[0] != "operations" {
		return false
	}
	pageSize, start, valid := parseGCPChroniclePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpChronicleOperation(project, location, instance, "team-operation")}
	return respondGCPChronicleList(w, "operations", items, pageSize, start, path)
}

func handleGCPChronicleGetOperation(w http.ResponseWriter, path string) bool {
	project, location, instance, tail, ok := parseGCPChronicleInstanceTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpChronicleOperation(project, location, instance, tail[1]))
	return true
}

func parseGCPChronicleInstanceTail(path string) (project, location, instance string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "instances" {
		return "", "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	instance = strings.TrimSpace(parts[7])
	if project == "" || location == "" || instance == "" {
		return "", "", "", nil, false
	}
	if _, ok := gcpChronicleSupportedLocations[strings.ToLower(location)]; !ok {
		return "", "", "", nil, false
	}
	return project, location, instance, parts[8:], true
}

func parseGCPChroniclePagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPChronicleInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPChronicleInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPChronicleList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPChronicleInvalidArgument(w, path, "pageToken is out of range")
		return false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func normalizeGCPChronicleActionSegment(segment string) string {
	normalized := strings.TrimSpace(segment)
	normalized = strings.ReplaceAll(normalized, "%3A", ":")
	normalized = strings.ReplaceAll(normalized, "%3a", ":")
	return normalized
}

func gcpChronicleInstance(project, location, instance string) map[string]any {
	return map[string]any{
		"name":         fmt.Sprintf("projects/%s/locations/%s/instances/%s", project, location, instance),
		"displayName":  "Stackyard Chronicle",
		"state":        "ACTIVE",
		"customerTier": "STANDARD",
	}
}

func gcpChronicleDataAccessLabel(project, location, instance, labelID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/instances/%s/dataAccessLabels/%s", project, location, instance, labelID),
		"displayName": "Team Label",
	}
}

func gcpChronicleDataAccessScope(project, location, instance, scopeID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/instances/%s/dataAccessScopes/%s", project, location, instance, scopeID),
		"displayName": "Team Scope",
	}
}

func gcpChronicleReferenceList(project, location, instance, listID string) map[string]any {
	return map[string]any{
		"name":               fmt.Sprintf("projects/%s/locations/%s/instances/%s/referenceLists/%s", project, location, instance, listID),
		"displayName":        "Reference List",
		"revisionCreateTime": "2026-01-01T00:00:00Z",
	}
}

func gcpChronicleRule(project, location, instance, ruleID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/instances/%s/rules/%s", project, location, instance, ruleID),
		"displayName": "Team Rule",
		"severity":    "MEDIUM",
	}
}

func gcpChronicleRuleDeployment(project, location, instance, ruleID string) map[string]any {
	return map[string]any{
		"name":  fmt.Sprintf("projects/%s/locations/%s/instances/%s/rules/%s/deployment", project, location, instance, ruleID),
		"state": "LIVE",
	}
}

func gcpChronicleRetrohunt(project, location, instance, ruleID, retrohuntID string) map[string]any {
	return map[string]any{
		"name":  fmt.Sprintf("projects/%s/locations/%s/instances/%s/rules/%s/retrohunts/%s", project, location, instance, ruleID, retrohuntID),
		"state": "SUCCEEDED",
	}
}

func gcpChronicleWatchlist(project, location, instance, watchlistID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/instances/%s/watchlists/%s", project, location, instance, watchlistID),
		"displayName": "High Risk Entities",
	}
}

func gcpChronicleOperation(project, location, instance, operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/instances/%s/operations/%s", project, location, instance, operationID),
		"done": true,
	}
}

func respondGCPChronicleInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
