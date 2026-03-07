package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPDataflowRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPDataflowPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPDataflowListJobMessages(w, r, path) {
			return true
		}
		if handleGCPDataflowGetJobMetrics(w, path) {
			return true
		}
		if handleGCPDataflowGetJobExecutionDetails(w, path) {
			return true
		}
		if handleGCPDataflowGetStageExecutionDetails(w, path) {
			return true
		}
		if handleGCPDataflowListJobs(w, r, path) {
			return true
		}
		if handleGCPDataflowAggregatedListJobs(w, r, path) {
			return true
		}
		if handleGCPDataflowGetJob(w, path) {
			return true
		}
		if handleGCPDataflowGetSnapshot(w, path) {
			return true
		}
		if handleGCPDataflowListSnapshots(w, path) {
			return true
		}
		if handleGCPDataflowGetTemplate(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPDataflowCreateJob(w, r, path) {
			return true
		}
		if handleGCPDataflowSnapshotJob(w, path) {
			return true
		}
		if handleGCPDataflowCreateJobFromTemplate(w, r, path) {
			return true
		}
		if handleGCPDataflowLaunchTemplate(w, r, path) {
			return true
		}
		if handleGCPDataflowLaunchFlexTemplate(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPut:
		if handleGCPDataflowUpdateJob(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPDataflowDeleteSnapshot(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPDataflowPath(path string) bool {
	if !strings.HasPrefix(path, "/gcp/v1b3/projects/") {
		return false
	}
	if _, _, ok := parseGCPDataflowJobsCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPDataflowJobPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPDataflowJobActionPath(path); ok {
		return true
	}
	if _, ok := parseGCPDataflowAggregatedJobsPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPDataflowJobMessagesPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPDataflowJobMetricsPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPDataflowJobExecutionDetailsPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPDataflowStageExecutionDetailsPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPDataflowJobSnapshotsPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPDataflowSnapshotsCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPDataflowSnapshotPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPDataflowTemplatesPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPDataflowTemplateActionPath(path); ok {
		return true
	}
	_, _, _, ok := parseGCPDataflowFlexTemplateActionPath(path)
	return ok
}

func handleGCPDataflowCreateJob(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPDataflowJobsCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPDataflowJSONBody(w, r, path)
	if !valid {
		return true
	}
	job, _ := body["job"].(map[string]any)
	if len(job) == 0 {
		job = body
	}
	name, _ := job["name"].(string)
	if strings.TrimSpace(name) == "" {
		respondGCPDataflowInvalidArgument(w, path, "job.name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDataflowJob(project, location, "team-job", name))
	return true
}

func handleGCPDataflowUpdateJob(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, jobID, ok := parseGCPDataflowJobPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPDataflowJSONBody(w, r, path)
	if !valid {
		return true
	}
	job, _ := body["job"].(map[string]any)
	if len(job) == 0 {
		job = body
	}
	if len(job) == 0 {
		respondGCPDataflowInvalidArgument(w, path, "job payload is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDataflowJob(project, location, jobID, jobID))
	return true
}

func handleGCPDataflowGetJob(w http.ResponseWriter, path string) bool {
	project, location, jobID, ok := parseGCPDataflowJobPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDataflowJob(project, location, jobID, jobID))
	return true
}

func handleGCPDataflowListJobs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPDataflowJobsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPDataflowPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDataflowJob(project, location, "team-job", "team-job")}
	return respondGCPDataflowList(w, "jobs", items, pageSize, start, path)
}

func handleGCPDataflowAggregatedListJobs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, ok := parseGCPDataflowAggregatedJobsPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPDataflowPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDataflowJob(project, "us-central1", "team-job", "team-job")}
	return respondGCPDataflowList(w, "jobs", items, pageSize, start, path)
}

func handleGCPDataflowSnapshotJob(w http.ResponseWriter, path string) bool {
	project, location, jobID, action, ok := parseGCPDataflowJobActionPath(path)
	if !ok || action != "snapshot" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDataflowSnapshot(project, location, "snap-1", jobID))
	return true
}

func handleGCPDataflowListJobMessages(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, jobID, ok := parseGCPDataflowJobMessagesPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPDataflowPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		{
			"id":                "msg-1",
			"jobId":             jobID,
			"time":              "2026-01-01T00:00:00Z",
			"messageText":       "Dataflow job started",
			"messageImportance": "JOB_MESSAGE_BASIC",
			"projectId":         project,
			"location":          location,
		},
	}
	return respondGCPDataflowList(w, "jobMessages", items, pageSize, start, path)
}

func handleGCPDataflowGetJobMetrics(w http.ResponseWriter, path string) bool {
	_, _, _, ok := parseGCPDataflowJobMetricsPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{"metrics": []any{}})
	return true
}

func handleGCPDataflowGetJobExecutionDetails(w http.ResponseWriter, path string) bool {
	_, _, _, ok := parseGCPDataflowJobExecutionDetailsPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDataflowGetStageExecutionDetails(w http.ResponseWriter, path string) bool {
	_, _, _, _, ok := parseGCPDataflowStageExecutionDetailsPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDataflowGetSnapshot(w http.ResponseWriter, path string) bool {
	project, location, snapshotID, ok := parseGCPDataflowSnapshotPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDataflowSnapshot(project, location, snapshotID, "team-job"))
	return true
}

func handleGCPDataflowDeleteSnapshot(w http.ResponseWriter, path string) bool {
	_, _, _, ok := parseGCPDataflowSnapshotPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDataflowListSnapshots(w http.ResponseWriter, path string) bool {
	project, location, jobID, ok := parseGCPDataflowJobSnapshotsPath(path)
	if ok {
		respondJSON(w, http.StatusOK, map[string]any{
			"snapshots": []any{gcpDataflowSnapshot(project, location, "snap-1", jobID)},
		})
		return true
	}
	project, location, ok = parseGCPDataflowSnapshotsCollectionPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"snapshots": []any{gcpDataflowSnapshot(project, location, "snap-1", "team-job")},
	})
	return true
}

func handleGCPDataflowCreateJobFromTemplate(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPDataflowTemplatesPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPDataflowJSONBody(w, r, path)
	if !valid {
		return true
	}
	jobName, _ := body["jobName"].(string)
	if strings.TrimSpace(jobName) == "" {
		respondGCPDataflowInvalidArgument(w, path, "jobName is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDataflowJob(project, location, "template-job", jobName))
	return true
}

func handleGCPDataflowLaunchTemplate(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, action, ok := parseGCPDataflowTemplateActionPath(path)
	if !ok || action != "launch" {
		return false
	}
	body, valid := decodeGCPDataflowJSONBody(w, r, path)
	if !valid {
		return true
	}
	launchParameters, _ := body["launchParameters"].(map[string]any)
	jobName, _ := launchParameters["jobName"].(string)
	if strings.TrimSpace(jobName) == "" {
		jobName, _ = body["jobName"].(string)
	}
	if strings.TrimSpace(jobName) == "" {
		respondGCPDataflowInvalidArgument(w, path, "jobName is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"job": gcpDataflowJob(project, location, "launch-template-job", jobName),
	})
	return true
}

func handleGCPDataflowGetTemplate(w http.ResponseWriter, path string) bool {
	_, _, action, ok := parseGCPDataflowTemplateActionPath(path)
	if !ok || action != "get" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDataflowLaunchFlexTemplate(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, action, ok := parseGCPDataflowFlexTemplateActionPath(path)
	if !ok || action != "launch" {
		return false
	}
	body, valid := decodeGCPDataflowJSONBody(w, r, path)
	if !valid {
		return true
	}
	launchParameter, _ := body["launchParameter"].(map[string]any)
	jobName, _ := launchParameter["jobName"].(string)
	if strings.TrimSpace(jobName) == "" {
		respondGCPDataflowInvalidArgument(w, path, "launchParameter.jobName is required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"job": gcpDataflowJob(project, location, "flex-template-job", jobName),
	})
	return true
}

func parseGCPDataflowJobsCollectionPath(path string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1b3" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "jobs" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPDataflowJobPath(path string) (project, location, jobID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1b3" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "jobs" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	jobID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || jobID == "" || strings.Contains(jobID, ":") {
		return "", "", "", false
	}
	return project, location, jobID, true
}

func parseGCPDataflowJobActionPath(path string) (project, location, jobID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1b3" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "jobs" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	jobAction := normalizeGCPDataflowActionSegment(parts[7])
	jobID, action, found := strings.Cut(jobAction, ":")
	if !found {
		return "", "", "", "", false
	}
	jobID = strings.TrimSpace(jobID)
	action = strings.TrimSpace(action)
	if project == "" || location == "" || jobID == "" || action == "" {
		return "", "", "", "", false
	}
	return project, location, jobID, action, true
}

func parseGCPDataflowAggregatedJobsPath(path string) (project string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v1b3" || parts[2] != "projects" {
		return "", false
	}
	project = strings.TrimSpace(parts[3])
	action := normalizeGCPDataflowActionSegment(parts[4])
	if project == "" || action != "jobs:aggregated" {
		return "", false
	}
	return project, true
}

func parseGCPDataflowJobMessagesPath(path string) (project, location, jobID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "v1b3" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "jobs" || parts[8] != "messages" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	jobID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || jobID == "" {
		return "", "", "", false
	}
	return project, location, jobID, true
}

func parseGCPDataflowJobMetricsPath(path string) (project, location, jobID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "v1b3" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "jobs" || parts[8] != "metrics" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	jobID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || jobID == "" {
		return "", "", "", false
	}
	return project, location, jobID, true
}

func parseGCPDataflowJobExecutionDetailsPath(path string) (project, location, jobID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "v1b3" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "jobs" || parts[8] != "executionDetails" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	jobID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || jobID == "" {
		return "", "", "", false
	}
	return project, location, jobID, true
}

func parseGCPDataflowStageExecutionDetailsPath(path string) (project, location, jobID, stageID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 11 || parts[0] != "gcp" || parts[1] != "v1b3" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "jobs" || parts[8] != "stages" || parts[10] != "executionDetails" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	jobID = strings.TrimSpace(parts[7])
	stageID = strings.TrimSpace(parts[9])
	if project == "" || location == "" || jobID == "" || stageID == "" {
		return "", "", "", "", false
	}
	return project, location, jobID, stageID, true
}

func parseGCPDataflowJobSnapshotsPath(path string) (project, location, jobID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "v1b3" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "jobs" || parts[8] != "snapshots" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	jobID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || jobID == "" {
		return "", "", "", false
	}
	return project, location, jobID, true
}

func parseGCPDataflowSnapshotsCollectionPath(path string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1b3" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "snapshots" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPDataflowSnapshotPath(path string) (project, location, snapshotID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1b3" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "snapshots" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	snapshotID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || snapshotID == "" {
		return "", "", "", false
	}
	return project, location, snapshotID, true
}

func parseGCPDataflowTemplatesPath(path string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1b3" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "templates" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPDataflowTemplateActionPath(path string) (project, location, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1b3" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	templateAction := normalizeGCPDataflowActionSegment(parts[6])
	name, action, found := strings.Cut(templateAction, ":")
	if !found || strings.TrimSpace(name) != "templates" {
		return "", "", "", false
	}
	action = strings.TrimSpace(action)
	if project == "" || location == "" || action == "" {
		return "", "", "", false
	}
	return project, location, action, true
}

func parseGCPDataflowFlexTemplateActionPath(path string) (project, location, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1b3" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	flexAction := normalizeGCPDataflowActionSegment(parts[6])
	name, action, found := strings.Cut(flexAction, ":")
	if !found || strings.TrimSpace(name) != "flexTemplates" {
		return "", "", "", false
	}
	action = strings.TrimSpace(action)
	if project == "" || location == "" || action == "" {
		return "", "", "", false
	}
	return project, location, action, true
}

func parseGCPDataflowPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	size, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPDataflowInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	token := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if token == "" {
		return size, 0, true
	}
	start, err = parseOptionalNonNegativeInt(token)
	if err != nil {
		respondGCPDataflowInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return size, start, true
}

func respondGCPDataflowList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPDataflowInvalidArgument(w, path, "pageToken is out of range")
		return true
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

func decodeGCPDataflowJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPDataflowInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func normalizeGCPDataflowActionSegment(segment string) string {
	trimmed := strings.TrimSpace(segment)
	trimmed = strings.ReplaceAll(trimmed, "%3A", ":")
	trimmed = strings.ReplaceAll(trimmed, "%3a", ":")
	return trimmed
}

func gcpDataflowJob(project, location, jobID, name string) map[string]any {
	return map[string]any{
		"id":           jobID,
		"name":         name,
		"projectId":    project,
		"location":     location,
		"currentState": "JOB_STATE_RUNNING",
		"type":         "JOB_TYPE_BATCH",
	}
}

func gcpDataflowSnapshot(project, location, snapshotID, jobID string) map[string]any {
	return map[string]any{
		"id":          snapshotID,
		"projectId":   project,
		"location":    location,
		"sourceJobId": jobID,
		"description": "Stackyard snapshot",
	}
}

func respondGCPDataflowInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
