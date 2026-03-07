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

var gcpSchedulerReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPSchedulerRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_scheduler(w, r) {
		return true
	}

	path := normalizeGCPSchedulerPath(rawRequestPath(r))
	if isGCPSchedulerLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPSchedulerListLocations(w, r, path) {
			return true
		}
		if handleGCPSchedulerGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPSchedulerPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPSchedulerListJobs(w, r, path) {
			return true
		}
		if handleGCPSchedulerGetJob(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPSchedulerCreateJob(w, r, path) {
			return true
		}
		if handleGCPSchedulerPauseJob(w, path) {
			return true
		}
		if handleGCPSchedulerResumeJob(w, path) {
			return true
		}
		if handleGCPSchedulerRunJob(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPSchedulerUpdateJob(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPSchedulerDeleteJob(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPSchedulerPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPSchedulerHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "scheduler", "scheduler-apiv1", "cloud-scheduler", "cloud_scheduler", "cloudscheduler":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-scheduler-apiv1") || strings.Contains(ua, "cloud.google.com/go/scheduler")
}

func isGCPSchedulerLocationRequest(r *http.Request, path string) bool {
	if !hasGCPSchedulerHint(r) {
		return false
	}
	_, _, _, ok := parseGCPSchedulerProjectLocationPath(path)
	return ok
}

func isGCPSchedulerPath(path string) bool {
	if _, _, ok := parseGCPSchedulerJobsCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPSchedulerJobPath(path); ok {
		return true
	}
	_, _, _, _, ok := parseGCPSchedulerJobActionPath(path)
	return ok
}

func handleGCPSchedulerListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPSchedulerProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPSchedulerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSchedulerLocation(project, "us-central1"),
		gcpSchedulerLocation(project, "global"),
	}
	return respondGCPSchedulerList(w, "locations", items, pageSize, start, path)
}

func handleGCPSchedulerGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPSchedulerProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSchedulerLocation(project, location))
	return true
}

func handleGCPSchedulerListJobs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPSchedulerJobsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSchedulerPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSchedulerJob(project, location, "job-1", "ENABLED"),
		gcpSchedulerJob(project, location, "job-paused", "PAUSED"),
	}
	return respondGCPSchedulerList(w, "jobs", items, pageSize, start, path)
}

func handleGCPSchedulerGetJob(w http.ResponseWriter, path string) bool {
	project, location, jobID, ok := parseGCPSchedulerJobPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpSchedulerJob(project, location, jobID, gcpSchedulerStateForJobID(jobID)))
	return true
}

func handleGCPSchedulerCreateJob(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPSchedulerJobsCollectionPath(path)
	if !ok {
		return false
	}

	body, valid := decodeGCPSchedulerJSONBody(w, r, path)
	if !valid {
		return true
	}
	job := gcpSchedulerJobFromBody(body)
	if len(job) == 0 {
		respondGCPSchedulerInvalidArgument(w, path, "job is required")
		return true
	}
	if !gcpSchedulerValidateJobConfig(w, path, job, true) {
		return true
	}

	nameFromPayload := gcpSchedulerString(job, "name")
	jobIDFromPayload := ""
	if nameFromPayload != "" {
		payloadProject, payloadLocation, payloadJobID, nameOK := parseGCPSchedulerJobResourceName(nameFromPayload)
		if !nameOK {
			respondGCPSchedulerInvalidArgument(w, path, "job.name is invalid")
			return true
		}
		if payloadProject != project || payloadLocation != location {
			respondGCPSchedulerInvalidArgument(w, path, "job.name must match parent")
			return true
		}
		jobIDFromPayload = payloadJobID
	}

	jobID := strings.TrimSpace(r.URL.Query().Get("jobId"))
	if jobID != "" {
		if !isGCPSchedulerJobID(jobID) {
			respondGCPSchedulerInvalidArgument(w, path, "jobId is invalid")
			return true
		}
		if jobIDFromPayload != "" && jobIDFromPayload != jobID {
			respondGCPSchedulerInvalidArgument(w, path, "jobId must match job.name")
			return true
		}
	} else if jobIDFromPayload != "" {
		jobID = jobIDFromPayload
	} else {
		jobID = "job-1"
	}

	resp := gcpSchedulerJob(project, location, jobID, "ENABLED")
	applyGCPSchedulerJobOverrides(resp, job)
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPSchedulerUpdateJob(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, jobID, ok := parseGCPSchedulerJobPath(path)
	if !ok {
		return false
	}

	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		respondGCPSchedulerInvalidArgument(w, path, "updateMask is required")
		return true
	}

	body, valid := decodeGCPSchedulerJSONBody(w, r, path)
	if !valid {
		return true
	}
	job := gcpSchedulerJobFromBody(body)
	if len(job) == 0 {
		respondGCPSchedulerInvalidArgument(w, path, "job is required")
		return true
	}
	if !gcpSchedulerValidateJobConfig(w, path, job, true) {
		return true
	}

	expectedName := gcpSchedulerJobName(project, location, jobID)
	if name := gcpSchedulerString(job, "name"); name == "" || name != expectedName {
		respondGCPSchedulerInvalidArgument(w, path, "job.name must match requested resource")
		return true
	}

	resp := gcpSchedulerJob(project, location, jobID, gcpSchedulerStateForJobID(jobID))
	applyGCPSchedulerJobOverrides(resp, job)
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPSchedulerDeleteJob(w http.ResponseWriter, path string) bool {
	_, _, _, ok := parseGCPSchedulerJobPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSchedulerPauseJob(w http.ResponseWriter, path string) bool {
	project, location, jobID, action, ok := parseGCPSchedulerJobActionPath(path)
	if !ok || action != "pause" {
		return false
	}
	if gcpSchedulerStateForJobID(jobID) != "ENABLED" {
		respondGCPSchedulerFailedPrecondition(w, path, "job must be ENABLED to pause")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSchedulerJob(project, location, jobID, "PAUSED"))
	return true
}

func handleGCPSchedulerResumeJob(w http.ResponseWriter, path string) bool {
	project, location, jobID, action, ok := parseGCPSchedulerJobActionPath(path)
	if !ok || action != "resume" {
		return false
	}
	if gcpSchedulerStateForJobID(jobID) != "PAUSED" {
		respondGCPSchedulerFailedPrecondition(w, path, "job must be PAUSED to resume")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSchedulerJob(project, location, jobID, "ENABLED"))
	return true
}

func handleGCPSchedulerRunJob(w http.ResponseWriter, path string) bool {
	project, location, jobID, action, ok := parseGCPSchedulerJobActionPath(path)
	if !ok || action != "run" {
		return false
	}
	job := gcpSchedulerJob(project, location, jobID, gcpSchedulerStateForJobID(jobID))
	job["lastAttemptTime"] = gcpSchedulerReferenceTime.Add(45 * time.Second).Format(time.RFC3339)
	job["scheduleTime"] = gcpSchedulerReferenceTime.Add(15 * time.Minute).Format(time.RFC3339)
	respondJSON(w, http.StatusOK, job)
	return true
}

func decodeGCPSchedulerJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPSchedulerInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		respondGCPSchedulerInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPSchedulerInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func gcpSchedulerJobFromBody(body map[string]any) map[string]any {
	if nested, ok := body["job"].(map[string]any); ok && len(nested) > 0 {
		return nested
	}
	return body
}

func gcpSchedulerValidateJobConfig(w http.ResponseWriter, path string, job map[string]any, requireTarget bool) bool {
	if strings.TrimSpace(gcpSchedulerString(job, "schedule")) == "" {
		respondGCPSchedulerInvalidArgument(w, path, "job.schedule is required")
		return false
	}
	if strings.TrimSpace(gcpSchedulerString(job, "timeZone")) == "" {
		respondGCPSchedulerInvalidArgument(w, path, "job.timeZone is required")
		return false
	}
	targetField, target := gcpSchedulerFindTarget(job)
	if requireTarget && targetField == "" {
		respondGCPSchedulerInvalidArgument(w, path, "job target is required")
		return false
	}
	if targetField == "httpTarget" && strings.TrimSpace(gcpSchedulerString(target, "uri")) == "" {
		respondGCPSchedulerInvalidArgument(w, path, "job.httpTarget.uri is required")
		return false
	}
	if targetField == "pubsubTarget" && strings.TrimSpace(gcpSchedulerString(target, "topicName")) == "" {
		respondGCPSchedulerInvalidArgument(w, path, "job.pubsubTarget.topicName is required")
		return false
	}
	return true
}

func gcpSchedulerFindTarget(job map[string]any) (string, map[string]any) {
	for _, key := range []string{"httpTarget", "pubsubTarget", "appEngineHttpTarget"} {
		if target, ok := job[key].(map[string]any); ok && len(target) > 0 {
			return key, target
		}
	}
	return "", nil
}

func applyGCPSchedulerJobOverrides(out, in map[string]any) {
	for _, key := range []string{"description", "schedule", "timeZone", "retryConfig", "attemptDeadline"} {
		if val, ok := in[key]; ok {
			out[key] = val
		}
	}
	if targetKey, target := gcpSchedulerFindTarget(in); targetKey != "" {
		delete(out, "httpTarget")
		delete(out, "pubsubTarget")
		delete(out, "appEngineHttpTarget")
		out[targetKey] = target
	}
}

func parseGCPSchedulerPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0

	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 || n > 500 {
			respondGCPSchedulerInvalidArgument(w, path, "pageSize must be a non-negative integer <= 500")
			return 0, 0, false
		}
		pageSize = n
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			respondGCPSchedulerInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
		start = n
	}
	return pageSize, start, true
}

func respondGCPSchedulerList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPSchedulerInvalidArgument(w, path, "pageToken out of range")
		return true
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextToken := ""
	if end < len(items) {
		nextToken = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		field:           items[start:end],
		"nextPageToken": nextToken,
	})
	return true
}

func parseGCPSchedulerProjectLocationPath(path string) (project, location string, list bool, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 5 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "projects" && parts[4] == "locations" {
		project = strings.TrimSpace(parts[3])
		if project == "" {
			return "", "", false, false
		}
		return project, "", true, true
	}
	if len(parts) == 6 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "projects" && parts[4] == "locations" {
		project = strings.TrimSpace(parts[3])
		location = strings.TrimSpace(parts[5])
		if project == "" || location == "" {
			return "", "", false, false
		}
		return project, location, false, true
	}
	return "", "", false, false
}

func parseGCPSchedulerLocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", nil, false
	}
	return project, location, parts[6:], true
}

func parseGCPSchedulerJobsCollectionPath(path string) (project, location string, ok bool) {
	project, location, tail, ok := parseGCPSchedulerLocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "jobs" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPSchedulerJobPath(path string) (project, location, jobID string, ok bool) {
	project, location, tail, ok := parseGCPSchedulerLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "jobs" {
		return "", "", "", false
	}
	resource, _, hasAction := gcpSchedulerResourceActionSegment(tail[1])
	if hasAction || !isGCPSchedulerJobID(resource) {
		return "", "", "", false
	}
	return project, location, resource, true
}

func parseGCPSchedulerJobActionPath(path string) (project, location, jobID, action string, ok bool) {
	project, location, tail, ok := parseGCPSchedulerLocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "jobs" {
		return "", "", "", "", false
	}
	resource, action, hasAction := gcpSchedulerResourceActionSegment(tail[1])
	if !hasAction || !isGCPSchedulerJobID(resource) {
		return "", "", "", "", false
	}
	return project, location, resource, action, true
}

func gcpSchedulerResourceActionSegment(segment string) (resource, action string, hasAction bool) {
	parts := strings.SplitN(strings.TrimSpace(segment), ":", 2)
	resource = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		action = strings.TrimSpace(parts[1])
		hasAction = action != ""
	}
	return resource, action, hasAction
}

func parseGCPSchedulerJobResourceName(name string) (project, location, jobID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "jobs" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	jobID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || !isGCPSchedulerJobID(jobID) {
		return "", "", "", false
	}
	return project, location, jobID, true
}

func isGCPSchedulerJobID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func gcpSchedulerString(body map[string]any, key string) string {
	raw, ok := body[key]
	if !ok {
		return ""
	}
	str, _ := raw.(string)
	return strings.TrimSpace(str)
}

func gcpSchedulerParentName(project, location string) string {
	return fmt.Sprintf("projects/%s/locations/%s", project, location)
}

func gcpSchedulerJobName(project, location, jobID string) string {
	return fmt.Sprintf("%s/jobs/%s", gcpSchedulerParentName(project, location), jobID)
}

func gcpSchedulerStateForJobID(jobID string) string {
	if strings.Contains(strings.ToLower(strings.TrimSpace(jobID)), "paused") {
		return "PAUSED"
	}
	return "ENABLED"
}

func gcpSchedulerLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": strings.ToUpper(strings.ReplaceAll(location, "-", " ")),
		"labels": map[string]any{
			"cloud.googleapis.com/region": location,
		},
		"metadata": map[string]any{
			"provider": providerGCP,
			"service":  "scheduler",
		},
	}
}

func gcpSchedulerJob(project, location, jobID, state string) map[string]any {
	name := gcpSchedulerJobName(project, location, jobID)
	return map[string]any{
		"name":            name,
		"description":     "Stackyard Scheduler job " + jobID,
		"schedule":        "*/15 * * * *",
		"timeZone":        "UTC",
		"userUpdateTime":  gcpSchedulerReferenceTime.Format(time.RFC3339),
		"state":           state,
		"scheduleTime":    gcpSchedulerReferenceTime.Add(15 * time.Minute).Format(time.RFC3339),
		"lastAttemptTime": gcpSchedulerReferenceTime.Add(30 * time.Second).Format(time.RFC3339),
		"retryConfig": map[string]any{
			"retryCount":         3,
			"maxRetryDuration":   "60s",
			"minBackoffDuration": "5s",
			"maxBackoffDuration": "3600s",
			"maxDoublings":       5,
		},
		"attemptDeadline": "180s",
		"httpTarget": map[string]any{
			"uri":        "https://example.com/stackyard-scheduler",
			"httpMethod": "POST",
			"headers": map[string]any{
				"Content-Type": "application/json",
			},
		},
	}
}

func respondGCPSchedulerInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPSchedulerError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPSchedulerFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPSchedulerError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPSchedulerError(w http.ResponseWriter, status int, err, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    err,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_scheduler(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "scheduler") {
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
			"name":     "projects/stackyard/locations/us-central1/jobs/sample-job",
			"service":  "scheduler",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
