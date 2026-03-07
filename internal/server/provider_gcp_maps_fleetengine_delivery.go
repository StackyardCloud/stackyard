package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPMapsFleetEngineDeliveryRouter(w http.ResponseWriter, r *http.Request) bool {
	path := strings.TrimSpace(r.URL.Path)
	if path == "" {
		path = rawRequestPath(r)
	}
	if !isGCPMapsFleetEngineDeliveryPath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/maps.fleetengine.delivery.v1.DeliveryService/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	if !strings.HasPrefix(path, "/gcp/v1/providers/") {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPMapsFleetEngineDeliveryListDeliveryVehicles(w, r, path) {
			return true
		}
		if handleGCPMapsFleetEngineDeliveryGetDeliveryVehicle(w, path) {
			return true
		}
		if handleGCPMapsFleetEngineDeliveryListTasks(w, r, path) {
			return true
		}
		if handleGCPMapsFleetEngineDeliveryGetTask(w, path) {
			return true
		}
		if handleGCPMapsFleetEngineDeliveryGetTaskTrackingInfo(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPMapsFleetEngineDeliveryCreateDeliveryVehicle(w, r, path) {
			return true
		}
		if handleGCPMapsFleetEngineDeliveryCreateTask(w, r, path) {
			return true
		}
		if handleGCPMapsFleetEngineDeliveryBatchCreateTasks(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPMapsFleetEngineDeliveryUpdateDeliveryVehicle(w, r, path) {
			return true
		}
		if handleGCPMapsFleetEngineDeliveryUpdateTask(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPMapsFleetEngineDeliveryDeleteDeliveryVehicle(w, path) {
			return true
		}
		if handleGCPMapsFleetEngineDeliveryDeleteTask(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPMapsFleetEngineDeliveryPath(path string) bool {
	lowerPath := strings.ToLower(path)

	if strings.HasPrefix(path, "/gcp/maps.fleetengine.delivery.v1.DeliveryService/") {
		return true
	}

	if !strings.HasPrefix(path, "/gcp/v1/providers/") {
		return false
	}

	return strings.Contains(path, "/deliveryVehicles/") ||
		strings.Contains(path, "/deliveryVehicles") ||
		strings.Contains(lowerPath, "/tasks") ||
		strings.HasSuffix(lowerPath, "/tasks:batchcreate") ||
		strings.HasSuffix(lowerPath, "/tasks%3abatchcreate") ||
		strings.Contains(path, "/taskTrackingInfo/")
}

func handleGCPMapsFleetEngineDeliveryCreateDeliveryVehicle(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, _, list, ok := parseGCPMapsFleetEngineDeliveryCollectionPath(path, "deliveryVehicles")
	if !ok || !list {
		return false
	}

	body, valid := decodeGCPMapsFleetEngineDeliveryJSONBody(w, r, path)
	if !valid {
		return true
	}
	if len(body) == 0 {
		respondGCPMapsFleetEngineDeliveryInvalidArgument(w, path, "delivery vehicle payload is required")
		return true
	}

	deliveryVehicleID := strings.TrimSpace(r.URL.Query().Get("deliveryVehicleId"))
	if deliveryVehicleID == "" {
		respondGCPMapsFleetEngineDeliveryInvalidArgument(w, path, "deliveryVehicleId is required")
		return true
	}

	respondJSON(w, http.StatusOK, gcpMapsFleetEngineDeliveryVehicle(provider, deliveryVehicleID))
	return true
}

func handleGCPMapsFleetEngineDeliveryGetDeliveryVehicle(w http.ResponseWriter, path string) bool {
	provider, deliveryVehicleID, list, ok := parseGCPMapsFleetEngineDeliveryCollectionPath(path, "deliveryVehicles")
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpMapsFleetEngineDeliveryVehicle(provider, deliveryVehicleID))
	return true
}

func handleGCPMapsFleetEngineDeliveryUpdateDeliveryVehicle(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, deliveryVehicleID, list, ok := parseGCPMapsFleetEngineDeliveryCollectionPath(path, "deliveryVehicles")
	if !ok || list {
		return false
	}

	body, valid := decodeGCPMapsFleetEngineDeliveryJSONBody(w, r, path)
	if !valid {
		return true
	}
	name, _ := body["name"].(string)
	if strings.TrimSpace(name) == "" {
		respondGCPMapsFleetEngineDeliveryInvalidArgument(w, path, "name is required for update")
		return true
	}

	respondJSON(w, http.StatusOK, gcpMapsFleetEngineDeliveryVehicle(provider, deliveryVehicleID))
	return true
}

func handleGCPMapsFleetEngineDeliveryDeleteDeliveryVehicle(w http.ResponseWriter, path string) bool {
	_, _, list, ok := parseGCPMapsFleetEngineDeliveryCollectionPath(path, "deliveryVehicles")
	if !ok || list {
		return false
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"deleted": true,
	})
	return true
}

func handleGCPMapsFleetEngineDeliveryListDeliveryVehicles(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, _, list, ok := parseGCPMapsFleetEngineDeliveryCollectionPath(path, "deliveryVehicles")
	if !ok || !list {
		return false
	}

	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPMapsFleetEngineDeliveryInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}

	start := 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = parseOptionalNonNegativeInt(pageToken)
		if err != nil {
			respondGCPMapsFleetEngineDeliveryInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return true
		}
	}

	deliveryVehicles := []map[string]any{
		gcpMapsFleetEngineDeliveryVehicle(provider, "delivery-vehicle-1"),
	}
	if start > len(deliveryVehicles) {
		respondGCPMapsFleetEngineDeliveryInvalidArgument(w, path, "pageToken is out of range")
		return true
	}

	end := len(deliveryVehicles)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}

	nextPageToken := ""
	if end < len(deliveryVehicles) {
		nextPageToken = strconv.Itoa(end)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"deliveryVehicles": deliveryVehicles[start:end],
		"nextPageToken":    nextPageToken,
		"totalSize":        len(deliveryVehicles),
	})
	return true
}

func handleGCPMapsFleetEngineDeliveryCreateTask(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, _, list, ok := parseGCPMapsFleetEngineDeliveryCollectionPath(path, "tasks")
	if !ok || !list {
		return false
	}

	body, valid := decodeGCPMapsFleetEngineDeliveryJSONBody(w, r, path)
	if !valid {
		return true
	}
	if len(body) == 0 {
		respondGCPMapsFleetEngineDeliveryInvalidArgument(w, path, "task payload is required")
		return true
	}

	taskID := strings.TrimSpace(r.URL.Query().Get("taskId"))
	if taskID == "" {
		respondGCPMapsFleetEngineDeliveryInvalidArgument(w, path, "taskId is required")
		return true
	}

	trackingID, _ := body["trackingId"].(string)
	if strings.TrimSpace(trackingID) == "" {
		trackingID = "tracking-1"
	}

	respondJSON(w, http.StatusOK, gcpMapsFleetEngineDeliveryTask(provider, taskID, trackingID))
	return true
}

func handleGCPMapsFleetEngineDeliveryBatchCreateTasks(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, ok := parseGCPMapsFleetEngineDeliveryBatchCreateTasksPath(path)
	if !ok {
		return false
	}

	body, valid := decodeGCPMapsFleetEngineDeliveryJSONBody(w, r, path)
	if !valid {
		return true
	}

	requests, _ := body["requests"].([]any)
	if len(requests) == 0 {
		respondGCPMapsFleetEngineDeliveryInvalidArgument(w, path, "requests must be a non-empty array")
		return true
	}

	tasks := make([]map[string]any, 0, len(requests))
	for i, rawRequest := range requests {
		requestMap, _ := rawRequest.(map[string]any)
		taskID, _ := requestMap["taskId"].(string)
		taskID = strings.TrimSpace(taskID)
		if taskID == "" {
			taskID = fmt.Sprintf("task-%d", i+1)
		}

		trackingID := "tracking-1"
		if taskMap, ok := requestMap["task"].(map[string]any); ok {
			if value, ok := taskMap["trackingId"].(string); ok && strings.TrimSpace(value) != "" {
				trackingID = strings.TrimSpace(value)
			}
		}

		tasks = append(tasks, gcpMapsFleetEngineDeliveryTask(provider, taskID, trackingID))
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"tasks": tasks,
	})
	return true
}

func handleGCPMapsFleetEngineDeliveryGetTask(w http.ResponseWriter, path string) bool {
	provider, taskID, list, ok := parseGCPMapsFleetEngineDeliveryCollectionPath(path, "tasks")
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpMapsFleetEngineDeliveryTask(provider, taskID, "tracking-1"))
	return true
}

func handleGCPMapsFleetEngineDeliveryUpdateTask(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, taskID, list, ok := parseGCPMapsFleetEngineDeliveryCollectionPath(path, "tasks")
	if !ok || list {
		return false
	}

	body, valid := decodeGCPMapsFleetEngineDeliveryJSONBody(w, r, path)
	if !valid {
		return true
	}

	name, _ := body["name"].(string)
	if strings.TrimSpace(name) == "" {
		respondGCPMapsFleetEngineDeliveryInvalidArgument(w, path, "name is required for update")
		return true
	}

	trackingID := "tracking-1"
	if value, ok := body["trackingId"].(string); ok && strings.TrimSpace(value) != "" {
		trackingID = strings.TrimSpace(value)
	}

	respondJSON(w, http.StatusOK, gcpMapsFleetEngineDeliveryTask(provider, taskID, trackingID))
	return true
}

func handleGCPMapsFleetEngineDeliveryDeleteTask(w http.ResponseWriter, path string) bool {
	_, _, list, ok := parseGCPMapsFleetEngineDeliveryCollectionPath(path, "tasks")
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"deleted": true,
	})
	return true
}

func handleGCPMapsFleetEngineDeliveryListTasks(w http.ResponseWriter, r *http.Request, path string) bool {
	provider, _, list, ok := parseGCPMapsFleetEngineDeliveryCollectionPath(path, "tasks")
	if !ok || !list {
		return false
	}

	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPMapsFleetEngineDeliveryInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}

	start := 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = parseOptionalNonNegativeInt(pageToken)
		if err != nil {
			respondGCPMapsFleetEngineDeliveryInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return true
		}
	}

	tasks := []map[string]any{
		gcpMapsFleetEngineDeliveryTask(provider, "task-1", "tracking-1"),
	}
	if start > len(tasks) {
		respondGCPMapsFleetEngineDeliveryInvalidArgument(w, path, "pageToken is out of range")
		return true
	}

	end := len(tasks)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}

	nextPageToken := ""
	if end < len(tasks) {
		nextPageToken = strconv.Itoa(end)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"tasks":         tasks[start:end],
		"nextPageToken": nextPageToken,
		"totalSize":     len(tasks),
	})
	return true
}

func handleGCPMapsFleetEngineDeliveryGetTaskTrackingInfo(w http.ResponseWriter, path string) bool {
	provider, trackingID, ok := parseGCPMapsFleetEngineDeliveryTaskTrackingPath(path)
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpMapsFleetEngineDeliveryTaskTrackingInfo(provider, trackingID))
	return true
}

func decodeGCPMapsFleetEngineDeliveryJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	var body map[string]any
	if r.Body == nil {
		return map[string]any{}, true
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPMapsFleetEngineDeliveryInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func parseGCPMapsFleetEngineDeliveryCollectionPath(path, collection string) (provider, resource string, list, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	// /gcp/v1/providers/{provider}/{collection}
	// /gcp/v1/providers/{provider}/{collection}/{resource}
	if len(parts) < 5 || len(parts) > 6 {
		return "", "", false, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "providers" || parts[4] != collection {
		return "", "", false, false
	}

	provider = strings.TrimSpace(parts[3])
	if provider == "" {
		return "", "", false, false
	}

	if len(parts) == 5 {
		return provider, "", true, true
	}

	resource = strings.TrimSpace(parts[5])
	if resource == "" {
		return "", "", false, false
	}
	return provider, resource, false, true
}

func parseGCPMapsFleetEngineDeliveryBatchCreateTasksPath(path string) (provider string, ok bool) {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	if !strings.HasPrefix(trimmed, "gcp/v1/providers/") {
		return "", false
	}

	rest := strings.TrimPrefix(trimmed, "gcp/v1/providers/")
	provider, action, found := strings.Cut(rest, "/")
	if !found {
		return "", false
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return "", false
	}
	action = strings.ToLower(action)
	action = strings.ReplaceAll(action, "%3a", ":")
	if !strings.Contains(action, "tasks") || !strings.Contains(action, "batchcreate") {
		return "", false
	}
	return provider, true
}

func parseGCPMapsFleetEngineDeliveryTaskTrackingPath(path string) (provider, trackingID string, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	// /gcp/v1/providers/{provider}/taskTrackingInfo/{trackingID}
	if len(parts) != 6 {
		return "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "providers" || parts[4] != "taskTrackingInfo" {
		return "", "", false
	}

	provider = strings.TrimSpace(parts[3])
	trackingID = strings.TrimSpace(parts[5])
	if provider == "" || trackingID == "" {
		return "", "", false
	}
	return provider, trackingID, true
}

func respondGCPMapsFleetEngineDeliveryInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func gcpMapsFleetEngineDeliveryVehicle(provider, deliveryVehicleID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("providers/%s/deliveryVehicles/%s", provider, deliveryVehicleID),
		"type": 1,
		"lastLocation": map[string]any{
			"location": map[string]any{
				"latitude":  37.7749,
				"longitude": -122.4194,
			},
		},
		"attributes": []any{
			map[string]any{
				"key":         "env",
				"stringValue": "stackyard",
			},
		},
	}
}

func gcpMapsFleetEngineDeliveryTask(provider, taskID, trackingID string) map[string]any {
	return map[string]any{
		"name":       fmt.Sprintf("providers/%s/tasks/%s", provider, taskID),
		"type":       1,
		"state":      1,
		"trackingId": trackingID,
		"plannedLocation": map[string]any{
			"point": map[string]any{
				"latitude":  37.7749,
				"longitude": -122.4194,
			},
		},
		"taskDuration": "300s",
	}
}

func gcpMapsFleetEngineDeliveryTaskTrackingInfo(provider, trackingID string) map[string]any {
	return map[string]any{
		"name":       fmt.Sprintf("providers/%s/taskTrackingInfo/%s", provider, trackingID),
		"trackingId": trackingID,
		"state":      1,
		"plannedLocation": map[string]any{
			"point": map[string]any{
				"latitude":  37.7749,
				"longitude": -122.4194,
			},
		},
		"vehicleLocation": map[string]any{
			"location": map[string]any{
				"latitude":  37.7755,
				"longitude": -122.4187,
			},
		},
		"remainingStopCount":             1,
		"remainingDrivingDistanceMeters": 1200,
		"estimatedArrivalTime":           "2026-01-01T00:05:00Z",
	}
}
