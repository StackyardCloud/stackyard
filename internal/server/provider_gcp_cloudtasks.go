package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPCloudTasksRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPCloudTasksPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPCloudTasksListQueues(w, r, path) {
			return true
		}
		if handleGCPCloudTasksGetQueue(w, path) {
			return true
		}
		if handleGCPCloudTasksListTasks(w, r, path) {
			return true
		}
		if handleGCPCloudTasksGetTask(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPCloudTasksCreateQueue(w, r, path) {
			return true
		}
		if handleGCPCloudTasksPauseQueue(w, path) {
			return true
		}
		if handleGCPCloudTasksResumeQueue(w, path) {
			return true
		}
		if handleGCPCloudTasksPurgeQueue(w, path) {
			return true
		}
		if handleGCPCloudTasksCreateTask(w, r, path) {
			return true
		}
		if handleGCPCloudTasksRunTask(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPCloudTasksUpdateQueue(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPCloudTasksDeleteTask(w, path) {
			return true
		}
		if handleGCPCloudTasksDeleteQueue(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPCloudTasksPath(path string) bool {
	if !strings.HasPrefix(path, "/gcp/v2/projects/") || !strings.Contains(path, "/locations/") {
		return false
	}
	if _, _, ok := parseGCPCloudTasksQueuesCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPCloudTasksQueuePath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPCloudTasksTasksCollectionPath(path); ok {
		return true
	}
	if _, _, _, _, _, ok := parseGCPCloudTasksTaskPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPCloudTasksQueueActionPath(path); ok {
		return true
	}
	_, _, _, _, _, _, ok := parseGCPCloudTasksTaskActionPath(path)
	return ok
}

func handleGCPCloudTasksListQueues(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPCloudTasksQueuesCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPCloudTasksPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpCloudTasksQueue(project, location, "team-queue"),
	}
	return respondGCPCloudTasksList(w, "queues", items, pageSize, start, path)
}

func handleGCPCloudTasksGetQueue(w http.ResponseWriter, path string) bool {
	project, location, queueID, ok := parseGCPCloudTasksQueuePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudTasksQueue(project, location, queueID))
	return true
}

func handleGCPCloudTasksCreateQueue(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPCloudTasksQueuesCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPCloudTasksJSONBody(w, r, path)
	if !valid {
		return true
	}
	queue, _ := body["queue"].(map[string]any)
	if len(queue) == 0 {
		queue = body
	}
	name, _ := queue["name"].(string)
	if strings.TrimSpace(name) == "" {
		respondGCPCloudTasksInvalidArgument(w, path, "queue.name is required")
		return true
	}
	queueID := "team-queue"
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) > 0 && strings.TrimSpace(parts[len(parts)-1]) != "" {
		queueID = strings.TrimSpace(parts[len(parts)-1])
	}
	respondJSON(w, http.StatusOK, gcpCloudTasksQueue(project, location, queueID))
	return true
}

func handleGCPCloudTasksUpdateQueue(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, queueID, ok := parseGCPCloudTasksQueuePath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPCloudTasksJSONBody(w, r, path)
	if !valid {
		return true
	}
	queue, _ := body["queue"].(map[string]any)
	if len(queue) == 0 {
		queue = body
	}
	name, _ := queue["name"].(string)
	if strings.TrimSpace(name) == "" {
		respondGCPCloudTasksInvalidArgument(w, path, "queue.name is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpCloudTasksQueue(project, location, queueID))
	return true
}

func handleGCPCloudTasksPauseQueue(w http.ResponseWriter, path string) bool {
	project, location, queueID, action, ok := parseGCPCloudTasksQueueActionPath(path)
	if !ok || action != "pause" {
		return false
	}
	queue := gcpCloudTasksQueue(project, location, queueID)
	queue["state"] = "PAUSED"
	respondJSON(w, http.StatusOK, queue)
	return true
}

func handleGCPCloudTasksResumeQueue(w http.ResponseWriter, path string) bool {
	project, location, queueID, action, ok := parseGCPCloudTasksQueueActionPath(path)
	if !ok || action != "resume" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpCloudTasksQueue(project, location, queueID))
	return true
}

func handleGCPCloudTasksPurgeQueue(w http.ResponseWriter, path string) bool {
	project, location, queueID, action, ok := parseGCPCloudTasksQueueActionPath(path)
	if !ok || action != "purge" {
		return false
	}
	queue := gcpCloudTasksQueue(project, location, queueID)
	queue["purgeTime"] = "2026-01-01T00:00:00Z"
	respondJSON(w, http.StatusOK, queue)
	return true
}

func handleGCPCloudTasksDeleteQueue(w http.ResponseWriter, path string) bool {
	_, _, _, ok := parseGCPCloudTasksQueuePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPCloudTasksListTasks(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, queueID, ok := parseGCPCloudTasksTasksCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPCloudTasksPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpCloudTasksTask(project, location, queueID, "task-1"),
	}
	return respondGCPCloudTasksList(w, "tasks", items, pageSize, start, path)
}

func handleGCPCloudTasksGetTask(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, queueID, taskID, _, ok := parseGCPCloudTasksTaskPath(path)
	if !ok {
		return false
	}
	if !isValidGCPCloudTasksView(r.URL.Query().Get("responseView")) {
		respondGCPCloudTasksInvalidArgument(w, path, "responseView must be one of VIEW_UNSPECIFIED, BASIC, FULL")
		return true
	}
	respondJSON(w, http.StatusOK, gcpCloudTasksTask(project, location, queueID, taskID))
	return true
}

func handleGCPCloudTasksCreateTask(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, queueID, ok := parseGCPCloudTasksTasksCollectionPath(path)
	if !ok {
		return false
	}
	if !isValidGCPCloudTasksView(r.URL.Query().Get("responseView")) {
		respondGCPCloudTasksInvalidArgument(w, path, "responseView must be one of VIEW_UNSPECIFIED, BASIC, FULL")
		return true
	}
	body, valid := decodeGCPCloudTasksJSONBody(w, r, path)
	if !valid {
		return true
	}
	task, _ := body["task"].(map[string]any)
	if len(task) == 0 {
		task = body
	}
	name, _ := task["name"].(string)
	if strings.TrimSpace(name) == "" {
		respondGCPCloudTasksInvalidArgument(w, path, "task.name is required")
		return true
	}
	taskID := "task-1"
	parts := strings.Split(strings.TrimSpace(name), "/")
	if len(parts) > 0 && strings.TrimSpace(parts[len(parts)-1]) != "" {
		taskID = strings.TrimSpace(parts[len(parts)-1])
	}
	respondJSON(w, http.StatusOK, gcpCloudTasksTask(project, location, queueID, taskID))
	return true
}

func handleGCPCloudTasksRunTask(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, queueID, taskID, action, _, ok := parseGCPCloudTasksTaskActionPath(path)
	if !ok || action != "run" {
		return false
	}
	if !isValidGCPCloudTasksView(r.URL.Query().Get("responseView")) {
		respondGCPCloudTasksInvalidArgument(w, path, "responseView must be one of VIEW_UNSPECIFIED, BASIC, FULL")
		return true
	}
	respondJSON(w, http.StatusOK, gcpCloudTasksTask(project, location, queueID, taskID))
	return true
}

func handleGCPCloudTasksDeleteTask(w http.ResponseWriter, path string) bool {
	_, _, _, _, _, ok := parseGCPCloudTasksTaskPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPCloudTasksQueuesCollectionPath(path string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "queues" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPCloudTasksQueuePath(path string) (project, location, queueID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "queues" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	queueID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || queueID == "" {
		return "", "", "", false
	}
	return project, location, queueID, true
}

func parseGCPCloudTasksQueueActionPath(path string) (project, location, queueID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "queues" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	queueAndAction := normalizeGCPCloudTasksActionSegment(parts[7])
	queueID, action, found := strings.Cut(queueAndAction, ":")
	if !found {
		return "", "", "", "", false
	}
	queueID = strings.TrimSpace(queueID)
	action = strings.TrimSpace(action)
	if project == "" || location == "" || queueID == "" || action == "" {
		return "", "", "", "", false
	}
	return project, location, queueID, action, true
}

func parseGCPCloudTasksTasksCollectionPath(path string) (project, location, queueID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "queues" || parts[8] != "tasks" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	queueID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || queueID == "" {
		return "", "", "", false
	}
	return project, location, queueID, true
}

func parseGCPCloudTasksTaskPath(path string) (project, location, queueID, taskID, fullName string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "queues" || parts[8] != "tasks" {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	queueID = strings.TrimSpace(parts[7])
	taskID = strings.TrimSpace(parts[9])
	if project == "" || location == "" || queueID == "" || taskID == "" {
		return "", "", "", "", "", false
	}
	fullName = fmt.Sprintf("projects/%s/locations/%s/queues/%s/tasks/%s", project, location, queueID, taskID)
	return project, location, queueID, taskID, fullName, true
}

func parseGCPCloudTasksTaskActionPath(path string) (project, location, queueID, taskID, action, fullName string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "queues" || parts[8] != "tasks" {
		return "", "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	queueID = strings.TrimSpace(parts[7])
	taskAndAction := normalizeGCPCloudTasksActionSegment(parts[9])
	taskID, action, found := strings.Cut(taskAndAction, ":")
	if !found {
		return "", "", "", "", "", "", false
	}
	taskID = strings.TrimSpace(taskID)
	action = strings.TrimSpace(action)
	if project == "" || location == "" || queueID == "" || taskID == "" || action == "" {
		return "", "", "", "", "", "", false
	}
	fullName = fmt.Sprintf("projects/%s/locations/%s/queues/%s/tasks/%s", project, location, queueID, taskID)
	return project, location, queueID, taskID, action, fullName, true
}

func parseGCPCloudTasksPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPCloudTasksInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPCloudTasksInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPCloudTasksList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPCloudTasksInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPCloudTasksJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	var body map[string]any
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPCloudTasksInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpCloudTasksQueue(project, location, queueID string) map[string]any {
	return map[string]any{
		"name":  fmt.Sprintf("projects/%s/locations/%s/queues/%s", project, location, queueID),
		"state": "RUNNING",
		"rateLimits": map[string]any{
			"maxDispatchesPerSecond":  50,
			"maxConcurrentDispatches": 100,
		},
	}
}

func gcpCloudTasksTask(project, location, queueID, taskID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/queues/%s/tasks/%s", project, location, queueID, taskID),
		"httpRequest": map[string]any{
			"url":        "https://example.com/stackyard/tasks",
			"httpMethod": "POST",
			"headers": map[string]any{
				"content-type": "application/json",
			},
			"body": "eyJldmVudCI6ImNyZWF0ZWQifQ==",
		},
		"view": "BASIC",
	}
}

func isValidGCPCloudTasksView(raw string) bool {
	view := strings.TrimSpace(raw)
	if view == "" {
		return true
	}
	switch strings.ToUpper(view) {
	case "VIEW_UNSPECIFIED", "BASIC", "FULL", "0", "1", "2":
		return true
	default:
		return false
	}
}

func normalizeGCPCloudTasksActionSegment(segment string) string {
	normalized := strings.ReplaceAll(segment, "%3A", ":")
	normalized = strings.ReplaceAll(normalized, "%3a", ":")
	return normalized
}

func respondGCPCloudTasksInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
