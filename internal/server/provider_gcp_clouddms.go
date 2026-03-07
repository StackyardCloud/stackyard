package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPCloudDMSRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	hasHint := hasGCPCloudDMSHint(r)
	if strings.HasPrefix(path, "/gcp/google.cloud.clouddms.v1.DataMigrationService/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}
	if !isGCPCloudDMSPathWithHint(path, hasHint) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPCloudDMSListMigrationJobs(w, r, path) {
			return true
		}
		if handleGCPCloudDMSGetMigrationJob(w, path) {
			return true
		}
		if handleGCPCloudDMSListConnectionProfiles(w, r, path) {
			return true
		}
		if handleGCPCloudDMSGetConnectionProfile(w, path) {
			return true
		}
		if handleGCPCloudDMSListPrivateConnections(w, r, path) {
			return true
		}
		if handleGCPCloudDMSGetPrivateConnection(w, path) {
			return true
		}
		if handleGCPCloudDMSListConversionWorkspaces(w, r, path) {
			return true
		}
		if handleGCPCloudDMSGetConversionWorkspace(w, path) {
			return true
		}
		if handleGCPCloudDMSListMappingRules(w, r, path) {
			return true
		}
		if handleGCPCloudDMSGetMappingRule(w, path) {
			return true
		}
		if handleGCPCloudDMSFetchStaticIps(w, r, path) {
			return true
		}
		if handleGCPCloudDMSListOperations(w, r, path) {
			return true
		}
		if handleGCPCloudDMSGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPCloudDMSCreateMigrationJob(w, r, path) {
			return true
		}
		if handleGCPCloudDMSStartMigrationJob(w, path) {
			return true
		}
		if handleGCPCloudDMSCreateConnectionProfile(w, r, path) {
			return true
		}
		if handleGCPCloudDMSCreatePrivateConnection(w, r, path) {
			return true
		}
		if handleGCPCloudDMSCreateConversionWorkspace(w, r, path) {
			return true
		}
		if handleGCPCloudDMSCreateMappingRule(w, r, path) {
			return true
		}
		if handleGCPCloudDMSDescribeDatabaseEntities(w, r, path) {
			return true
		}
		if handleGCPCloudDMSSearchBackgroundJobs(w, r, path) {
			return true
		}
		if handleGCPCloudDMSDescribeConversionWorkspaceRevisions(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch, http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPCloudDMSPath(path string) bool {
	return isGCPCloudDMSPathWithHint(path, false)
}

func isGCPCloudDMSPathWithHint(path string, includeHint bool) bool {
	normalized := normalizeGCPCloudDMSActionSegment(path)
	if !strings.HasPrefix(normalized, "/gcp/v1/projects/") || !strings.Contains(normalized, "/locations/") {
		return false
	}
	if !includeHint {
		return false
	}
	markers := []string{
		"/migrationJobs",
		"/connectionProfiles",
		"/privateConnections",
		"/conversionWorkspaces",
		"/mappingRules",
		"/operations",
		":start",
		":describeDatabaseEntities",
		":searchBackgroundJobs",
		":describeConversionWorkspaceRevisions",
		":fetchStaticIps",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func isGCPCloudDMSSharedPath(path string) bool {
	normalized := normalizeGCPCloudDMSActionSegment(path)
	return strings.Contains(normalized, "/connectionProfiles") ||
		strings.Contains(normalized, "/privateConnections") ||
		strings.Contains(normalized, ":fetchStaticIps")
}

func hasGCPCloudDMSHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	if service == "clouddms" || service == "dms" || service == "datamigration" {
		return true
	}
	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(userAgent, "stackyard-clouddms-apiv1")
}

func handleGCPCloudDMSListMigrationJobs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "migrationJobs" {
		return false
	}
	pageSize, start, valid := parseGCPCloudDMSPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudDMSMigrationJob(project, location, "team-job")}
	return respondGCPCloudDMSList(w, "migrationJobs", items, pageSize, start, path)
}

func handleGCPCloudDMSGetMigrationJob(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "migrationJobs" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudDMSMigrationJob(project, location, tail[1]))
	return true
}

func handleGCPCloudDMSCreateMigrationJob(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "migrationJobs" {
		return false
	}
	body, valid := decodeGCPCloudDMSJSONBody(w, r, path)
	if !valid {
		return true
	}
	resource := gcpCloudDMSBodyMap(body, "migrationJob")
	if len(resource) == 0 {
		respondGCPCloudDMSInvalidArgument(w, path, "migrationJob is required")
		return true
	}
	id := strings.TrimSpace(r.URL.Query().Get("migrationJobId"))
	if id == "" {
		id = "team-job"
	}
	respondJSON(w, http.StatusOK, gcpCloudDMSOperation(project, location, "createMigrationJob."+id))
	return true
}

func handleGCPCloudDMSStartMigrationJob(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "migrationJobs" {
		return false
	}
	jobID, action, found := strings.Cut(normalizeGCPCloudDMSActionSegment(tail[1]), ":")
	if !found || action != "start" || strings.TrimSpace(jobID) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudDMSOperation(project, location, "startMigrationJob."+strings.TrimSpace(jobID)))
	return true
}

func handleGCPCloudDMSListConnectionProfiles(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "connectionProfiles" {
		return false
	}
	pageSize, start, valid := parseGCPCloudDMSPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudDMSConnectionProfile(project, location, "team-profile")}
	return respondGCPCloudDMSList(w, "connectionProfiles", items, pageSize, start, path)
}

func handleGCPCloudDMSGetConnectionProfile(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "connectionProfiles" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudDMSConnectionProfile(project, location, tail[1]))
	return true
}

func handleGCPCloudDMSCreateConnectionProfile(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "connectionProfiles" {
		return false
	}
	body, valid := decodeGCPCloudDMSJSONBody(w, r, path)
	if !valid {
		return true
	}
	resource := gcpCloudDMSBodyMap(body, "connectionProfile")
	if len(resource) == 0 {
		respondGCPCloudDMSInvalidArgument(w, path, "connectionProfile is required")
		return true
	}
	id := strings.TrimSpace(r.URL.Query().Get("connectionProfileId"))
	if id == "" {
		id = "team-profile"
	}
	respondJSON(w, http.StatusOK, gcpCloudDMSOperation(project, location, "createConnectionProfile."+id))
	return true
}

func handleGCPCloudDMSListPrivateConnections(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "privateConnections" {
		return false
	}
	pageSize, start, valid := parseGCPCloudDMSPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudDMSPrivateConnection(project, location, "team-private-connection")}
	return respondGCPCloudDMSList(w, "privateConnections", items, pageSize, start, path)
}

func handleGCPCloudDMSGetPrivateConnection(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "privateConnections" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudDMSPrivateConnection(project, location, tail[1]))
	return true
}

func handleGCPCloudDMSCreatePrivateConnection(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "privateConnections" {
		return false
	}
	body, valid := decodeGCPCloudDMSJSONBody(w, r, path)
	if !valid {
		return true
	}
	resource := gcpCloudDMSBodyMap(body, "privateConnection")
	if len(resource) == 0 {
		respondGCPCloudDMSInvalidArgument(w, path, "privateConnection is required")
		return true
	}
	id := strings.TrimSpace(r.URL.Query().Get("privateConnectionId"))
	if id == "" {
		id = "team-private-connection"
	}
	respondJSON(w, http.StatusOK, gcpCloudDMSOperation(project, location, "createPrivateConnection."+id))
	return true
}

func handleGCPCloudDMSListConversionWorkspaces(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "conversionWorkspaces" {
		return false
	}
	pageSize, start, valid := parseGCPCloudDMSPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudDMSConversionWorkspace(project, location, "team-workspace")}
	return respondGCPCloudDMSList(w, "conversionWorkspaces", items, pageSize, start, path)
}

func handleGCPCloudDMSGetConversionWorkspace(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "conversionWorkspaces" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudDMSConversionWorkspace(project, location, tail[1]))
	return true
}

func handleGCPCloudDMSCreateConversionWorkspace(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "conversionWorkspaces" {
		return false
	}
	body, valid := decodeGCPCloudDMSJSONBody(w, r, path)
	if !valid {
		return true
	}
	resource := gcpCloudDMSBodyMap(body, "conversionWorkspace")
	if len(resource) == 0 {
		respondGCPCloudDMSInvalidArgument(w, path, "conversionWorkspace is required")
		return true
	}
	id := strings.TrimSpace(r.URL.Query().Get("conversionWorkspaceId"))
	if id == "" {
		id = "team-workspace"
	}
	respondJSON(w, http.StatusOK, gcpCloudDMSOperation(project, location, "createConversionWorkspace."+id))
	return true
}

func handleGCPCloudDMSListMappingRules(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, workspaceID, tail, ok := parseGCPCloudDMSWorkspaceTail(path)
	if !ok || len(tail) != 1 || tail[0] != "mappingRules" {
		return false
	}
	pageSize, start, valid := parseGCPCloudDMSPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudDMSMappingRule(project, location, workspaceID, "team-rule")}
	return respondGCPCloudDMSList(w, "mappingRules", items, pageSize, start, path)
}

func handleGCPCloudDMSGetMappingRule(w http.ResponseWriter, path string) bool {
	project, location, workspaceID, tail, ok := parseGCPCloudDMSWorkspaceTail(path)
	if !ok || len(tail) != 2 || tail[0] != "mappingRules" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudDMSMappingRule(project, location, workspaceID, tail[1]))
	return true
}

func handleGCPCloudDMSCreateMappingRule(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, _, tail, ok := parseGCPCloudDMSWorkspaceTail(path)
	if !ok || len(tail) != 1 || tail[0] != "mappingRules" {
		return false
	}
	body, valid := decodeGCPCloudDMSJSONBody(w, r, path)
	if !valid {
		return true
	}
	resource := gcpCloudDMSBodyMap(body, "mappingRule")
	if len(resource) == 0 {
		respondGCPCloudDMSInvalidArgument(w, path, "mappingRule is required")
		return true
	}
	id := strings.TrimSpace(r.URL.Query().Get("mappingRuleId"))
	if id == "" {
		id = "team-rule"
	}
	respondJSON(w, http.StatusOK, gcpCloudDMSOperation(project, location, "createMappingRule."+id))
	return true
}

func handleGCPCloudDMSDescribeDatabaseEntities(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, workspaceID, action, ok := parseGCPCloudDMSWorkspaceActionPath(path)
	if !ok || action != "describeDatabaseEntities" {
		return false
	}
	if _, _, valid := parseGCPCloudDMSPagination(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"databaseEntities": []any{
			map[string]any{
				"name": fmt.Sprintf("projects/%s/locations/%s/conversionWorkspaces/%s/databaseEntities/entity-1", project, location, workspaceID),
				"tree": "SOURCE_TREE",
			},
		},
		"nextPageToken": "",
	})
	return true
}

func handleGCPCloudDMSSearchBackgroundJobs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, workspaceID, action, ok := parseGCPCloudDMSWorkspaceActionPath(path)
	if !ok || action != "searchBackgroundJobs" {
		return false
	}
	if _, valid := decodeGCPCloudDMSJSONBody(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"jobs": []any{
			map[string]any{
				"name":  fmt.Sprintf("projects/%s/locations/%s/conversionWorkspaces/%s/backgroundJobs/job-1", project, location, workspaceID),
				"state": "SUCCEEDED",
			},
		},
	})
	return true
}

func handleGCPCloudDMSDescribeConversionWorkspaceRevisions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, workspaceID, action, ok := parseGCPCloudDMSWorkspaceActionPath(path)
	if !ok || action != "describeConversionWorkspaceRevisions" {
		return false
	}
	if _, valid := decodeGCPCloudDMSJSONBody(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"revisions": []any{
			map[string]any{
				"name":  fmt.Sprintf("projects/%s/locations/%s/conversionWorkspaces/%s/revisions/r1", project, location, workspaceID),
				"state": "READY",
			},
		},
	})
	return true
}

func handleGCPCloudDMSFetchStaticIps(w http.ResponseWriter, r *http.Request, path string) bool {
	_, location, action, ok := parseGCPCloudDMSLocationAction(path)
	if !ok || action != "fetchStaticIps" {
		return false
	}
	pageSize, start, valid := parseGCPCloudDMSPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		{"ipAddress": "203.0.113.10", "region": location},
	}
	return respondGCPCloudDMSList(w, "staticIps", items, pageSize, start, path)
}

func handleGCPCloudDMSListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "operations" {
		return false
	}
	pageSize, start, valid := parseGCPCloudDMSPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpCloudDMSOperation(project, location, "team-operation")}
	return respondGCPCloudDMSList(w, "operations", items, pageSize, start, path)
}

func handleGCPCloudDMSGetOperation(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudDMSOperation(project, location, tail[1]))
	return true
}

func parseGCPCloudDMSLocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" || strings.Contains(location, ":") {
		return "", "", nil, false
	}
	return project, location, parts[6:], true
}

func parseGCPCloudDMSWorkspaceTail(path string) (project, location, workspaceID string, tail []string, ok bool) {
	project, location, baseTail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(baseTail) < 2 || baseTail[0] != "conversionWorkspaces" {
		return "", "", "", nil, false
	}
	workspaceID = strings.TrimSpace(baseTail[1])
	if workspaceID == "" || strings.Contains(workspaceID, ":") {
		return "", "", "", nil, false
	}
	return project, location, workspaceID, baseTail[2:], true
}

func parseGCPCloudDMSWorkspaceActionPath(path string) (project, location, workspaceID, action string, ok bool) {
	project, location, tail, ok := parseGCPCloudDMSLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "conversionWorkspaces" {
		return "", "", "", "", false
	}
	workspaceAndAction := normalizeGCPCloudDMSActionSegment(tail[1])
	workspaceID, action, ok = strings.Cut(workspaceAndAction, ":")
	if !ok {
		return "", "", "", "", false
	}
	workspaceID = strings.TrimSpace(workspaceID)
	action = strings.TrimSpace(action)
	if workspaceID == "" || action == "" {
		return "", "", "", "", false
	}
	return project, location, workspaceID, action, true
}

func parseGCPCloudDMSLocationAction(path string) (project, location, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	locationAndAction := normalizeGCPCloudDMSActionSegment(parts[5])
	location, action, ok = strings.Cut(locationAndAction, ":")
	if !ok {
		return "", "", "", false
	}
	location = strings.TrimSpace(location)
	action = strings.TrimSpace(action)
	if project == "" || location == "" || action == "" {
		return "", "", "", false
	}
	return project, location, action, true
}

func parseGCPCloudDMSPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPCloudDMSInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPCloudDMSInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPCloudDMSList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPCloudDMSInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPCloudDMSJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPCloudDMSInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpCloudDMSBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func normalizeGCPCloudDMSActionSegment(segment string) string {
	normalized := strings.TrimSpace(segment)
	normalized = strings.ReplaceAll(normalized, "%3A", ":")
	normalized = strings.ReplaceAll(normalized, "%3a", ":")
	return normalized
}

func gcpCloudDMSMigrationJob(project, location, id string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/migrationJobs/%s", project, location, id),
		"displayName": "Stackyard Migration Job",
		"state":       "RUNNING",
	}
}

func gcpCloudDMSConnectionProfile(project, location, id string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/connectionProfiles/%s", project, location, id),
		"displayName": "Stackyard Connection Profile",
	}
}

func gcpCloudDMSPrivateConnection(project, location, id string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/privateConnections/%s", project, location, id),
		"displayName": "Stackyard Private Connection",
	}
}

func gcpCloudDMSConversionWorkspace(project, location, id string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/conversionWorkspaces/%s", project, location, id),
		"displayName": "Stackyard Conversion Workspace",
	}
}

func gcpCloudDMSMappingRule(project, location, workspaceID, id string) map[string]any {
	return map[string]any{
		"name":  fmt.Sprintf("projects/%s/locations/%s/conversionWorkspaces/%s/mappingRules/%s", project, location, workspaceID, id),
		"state": "ACTIVE",
	}
}

func gcpCloudDMSOperation(project, location, id string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, id),
		"done": true,
	}
}

func respondGCPCloudDMSInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
