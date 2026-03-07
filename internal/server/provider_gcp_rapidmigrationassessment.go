package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPRapidMigrationAssessmentRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if isGCPRapidMigrationAssessmentLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPRapidMigrationAssessmentListLocations(w, r, path) {
			return true
		}
		if handleGCPRapidMigrationAssessmentGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPRapidMigrationAssessmentPath(path, hasGCPRapidMigrationAssessmentHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPRapidMigrationAssessmentListCollectors(w, r, path) {
			return true
		}
		if handleGCPRapidMigrationAssessmentGetCollector(w, path) {
			return true
		}
		if handleGCPRapidMigrationAssessmentGetAnnotation(w, path) {
			return true
		}
		if handleGCPRapidMigrationAssessmentListOperations(w, r, path) {
			return true
		}
		if handleGCPRapidMigrationAssessmentGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPRapidMigrationAssessmentCreateCollector(w, r, path) {
			return true
		}
		if handleGCPRapidMigrationAssessmentPauseCollector(w, r, path) {
			return true
		}
		if handleGCPRapidMigrationAssessmentResumeCollector(w, r, path) {
			return true
		}
		if handleGCPRapidMigrationAssessmentRegisterCollector(w, r, path) {
			return true
		}
		if handleGCPRapidMigrationAssessmentCreateAnnotation(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPRapidMigrationAssessmentUpdateCollector(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPRapidMigrationAssessmentDeleteCollector(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPRapidMigrationAssessmentLocationRequest(r *http.Request, path string) bool {
	return isGCPProjectLocationDiscoveryPath(path) && hasGCPRapidMigrationAssessmentHint(r)
}

func hasGCPRapidMigrationAssessmentHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	if service == "rapidmigrationassessment" || service == "rapid-migration-assessment" || service == "rma" {
		return true
	}
	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(userAgent, "stackyard-rapidmigrationassessment-apiv1") || strings.Contains(userAgent, "rapidmigrationassessment")
}

func isGCPRapidMigrationAssessmentPath(path string, includeOperations bool) bool {
	_, _, tail, ok := parseGCPRapidMigrationAssessmentLocationTail(path)
	if !ok || len(tail) == 0 {
		return false
	}
	if isGCPRapidMigrationAssessmentCollectorsCollectionTail(tail) ||
		isGCPRapidMigrationAssessmentCollectorTail(tail) ||
		isGCPRapidMigrationAssessmentCollectorActionTail(tail, "pause") ||
		isGCPRapidMigrationAssessmentCollectorActionTail(tail, "resume") ||
		isGCPRapidMigrationAssessmentCollectorActionTail(tail, "register") ||
		isGCPRapidMigrationAssessmentAnnotationsCollectionTail(tail) ||
		isGCPRapidMigrationAssessmentAnnotationTail(tail) {
		return true
	}
	return includeOperations && (isGCPRapidMigrationAssessmentOperationsCollectionTail(tail) || isGCPRapidMigrationAssessmentOperationTail(tail))
}

func handleGCPRapidMigrationAssessmentListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPRapidMigrationAssessmentPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRapidMigrationAssessmentLocation(project, "us-central1"),
		gcpRapidMigrationAssessmentLocation(project, "global"),
	}
	return respondGCPRapidMigrationAssessmentList(w, "locations", items, pageSize, start, path)
}

func handleGCPRapidMigrationAssessmentGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRapidMigrationAssessmentLocation(project, location))
	return true
}

func handleGCPRapidMigrationAssessmentListCollectors(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRapidMigrationAssessmentLocationTail(path)
	if !ok || !isGCPRapidMigrationAssessmentCollectorsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPRapidMigrationAssessmentPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpRapidMigrationAssessmentCollector(project, location, "collector-1", "Collector_STATE_ACTIVE")}
	return respondGCPRapidMigrationAssessmentList(w, "collectors", items, pageSize, start, path)
}

func handleGCPRapidMigrationAssessmentGetCollector(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPRapidMigrationAssessmentLocationTail(path)
	if !ok || !isGCPRapidMigrationAssessmentCollectorTail(tail) {
		return false
	}
	collectorID := strings.TrimSpace(tail[1])
	state := "Collector_STATE_ACTIVE"
	if strings.Contains(collectorID, "paused") {
		state = "Collector_STATE_PAUSED"
	}
	respondJSON(w, http.StatusOK, gcpRapidMigrationAssessmentCollector(project, location, collectorID, state))
	return true
}

func handleGCPRapidMigrationAssessmentCreateCollector(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRapidMigrationAssessmentLocationTail(path)
	if !ok || !isGCPRapidMigrationAssessmentCollectorsCollectionTail(tail) {
		return false
	}
	collectorID := strings.TrimSpace(r.URL.Query().Get("collectorId"))
	if collectorID == "" {
		respondGCPRapidMigrationAssessmentInvalidArgument(w, path, "collectorId is required")
		return true
	}
	body, valid := decodeGCPRapidMigrationAssessmentJSONBody(w, r, path)
	if !valid {
		return true
	}
	collector := gcpRapidMigrationAssessmentBodyMap(body, "collector")
	if len(collector) == 0 {
		respondGCPRapidMigrationAssessmentInvalidArgument(w, path, "collector is required")
		return true
	}
	displayName := strings.TrimSpace(gcpRapidMigrationAssessmentString(collector, "displayName"))
	if displayName == "" {
		respondGCPRapidMigrationAssessmentInvalidArgument(w, path, "collector.displayName is required")
		return true
	}
	if providedName := strings.TrimSpace(gcpRapidMigrationAssessmentString(collector, "name")); providedName != "" {
		expectedName := fmt.Sprintf("projects/%s/locations/%s/collectors/%s", project, location, collectorID)
		if providedName != expectedName {
			respondGCPRapidMigrationAssessmentInvalidArgument(w, path, "collector.name must match collectorId and parent")
			return true
		}
	}
	respondJSON(w, http.StatusOK, gcpRapidMigrationAssessmentOperation(project, location, "createCollector."+collectorID))
	return true
}

func handleGCPRapidMigrationAssessmentUpdateCollector(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRapidMigrationAssessmentLocationTail(path)
	if !ok || !isGCPRapidMigrationAssessmentCollectorTail(tail) {
		return false
	}
	body, valid := decodeGCPRapidMigrationAssessmentJSONBody(w, r, path)
	if !valid {
		return true
	}
	collector := gcpRapidMigrationAssessmentBodyMap(body, "collector")
	if len(collector) == 0 {
		respondGCPRapidMigrationAssessmentInvalidArgument(w, path, "collector is required")
		return true
	}
	collectorName := strings.TrimSpace(gcpRapidMigrationAssessmentString(collector, "name"))
	expectedName := fmt.Sprintf("projects/%s/locations/%s/collectors/%s", project, location, tail[1])
	if collectorName == "" || collectorName != expectedName {
		respondGCPRapidMigrationAssessmentInvalidArgument(w, path, "collector.name must match the requested resource")
		return true
	}
	mask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if mask == "" {
		mask = strings.TrimSpace(gcpRapidMigrationAssessmentString(body, "updateMask"))
	}
	if mask == "" {
		respondGCPRapidMigrationAssessmentInvalidArgument(w, path, "updateMask is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRapidMigrationAssessmentOperation(project, location, "updateCollector."+tail[1]))
	return true
}

func handleGCPRapidMigrationAssessmentDeleteCollector(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPRapidMigrationAssessmentLocationTail(path)
	if !ok || !isGCPRapidMigrationAssessmentCollectorTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRapidMigrationAssessmentOperation(project, location, "deleteCollector."+tail[1]))
	return true
}

func handleGCPRapidMigrationAssessmentPauseCollector(w http.ResponseWriter, r *http.Request, path string) bool {
	return handleGCPRapidMigrationAssessmentCollectorAction(w, r, path, "pause")
}

func handleGCPRapidMigrationAssessmentResumeCollector(w http.ResponseWriter, r *http.Request, path string) bool {
	return handleGCPRapidMigrationAssessmentCollectorAction(w, r, path, "resume")
}

func handleGCPRapidMigrationAssessmentRegisterCollector(w http.ResponseWriter, r *http.Request, path string) bool {
	return handleGCPRapidMigrationAssessmentCollectorAction(w, r, path, "register")
}

func handleGCPRapidMigrationAssessmentCollectorAction(w http.ResponseWriter, r *http.Request, path, action string) bool {
	project, location, tail, ok := parseGCPRapidMigrationAssessmentLocationTail(path)
	if !ok || !isGCPRapidMigrationAssessmentCollectorActionTail(tail, action) {
		return false
	}
	if _, valid := decodeGCPRapidMigrationAssessmentJSONBody(w, r, path); !valid {
		return true
	}
	collectorID, parsedAction, _ := parseGCPRapidMigrationAssessmentCollectorAction(tail)
	if parsedAction == "pause" && strings.Contains(collectorID, "paused") {
		respondGCPRapidMigrationAssessmentFailedPrecondition(w, path, "collector is already paused")
		return true
	}
	if parsedAction == "resume" && strings.Contains(collectorID, "active") {
		respondGCPRapidMigrationAssessmentFailedPrecondition(w, path, "collector is already active")
		return true
	}
	if parsedAction == "register" && strings.Contains(collectorID, "registered") {
		respondGCPRapidMigrationAssessmentFailedPrecondition(w, path, "collector is already registered")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRapidMigrationAssessmentOperation(project, location, parsedAction+"Collector."+collectorID))
	return true
}

func handleGCPRapidMigrationAssessmentCreateAnnotation(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRapidMigrationAssessmentLocationTail(path)
	if !ok || !isGCPRapidMigrationAssessmentAnnotationsCollectionTail(tail) {
		return false
	}
	body, valid := decodeGCPRapidMigrationAssessmentJSONBody(w, r, path)
	if !valid {
		return true
	}
	annotation := gcpRapidMigrationAssessmentBodyMap(body, "annotation")
	if rawType, ok := annotation["type"]; ok && !gcpRapidMigrationAssessmentValidAnnotationType(rawType) {
		respondGCPRapidMigrationAssessmentInvalidArgument(w, path, "annotation.type is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRapidMigrationAssessmentOperation(project, location, "createAnnotation.annotation-1"))
	return true
}

func handleGCPRapidMigrationAssessmentGetAnnotation(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPRapidMigrationAssessmentLocationTail(path)
	if !ok || !isGCPRapidMigrationAssessmentAnnotationTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRapidMigrationAssessmentAnnotation(project, location, tail[1], "Annotation_TYPE_LEGACY_EXPORT_CONSENT"))
	return true
}

func handleGCPRapidMigrationAssessmentListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRapidMigrationAssessmentLocationTail(path)
	if !ok || !isGCPRapidMigrationAssessmentOperationsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPRapidMigrationAssessmentPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpRapidMigrationAssessmentOperation(project, location, "createCollector.collector-1")}
	return respondGCPRapidMigrationAssessmentList(w, "operations", items, pageSize, start, path)
}

func handleGCPRapidMigrationAssessmentGetOperation(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPRapidMigrationAssessmentLocationTail(path)
	if !ok || !isGCPRapidMigrationAssessmentOperationTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRapidMigrationAssessmentOperation(project, location, tail[1]))
	return true
}

func parseGCPRapidMigrationAssessmentLocationTail(path string) (project, location string, tail []string, ok bool) {
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

func isGCPRapidMigrationAssessmentCollectorsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "collectors"
}

func isGCPRapidMigrationAssessmentCollectorTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "collectors" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPRapidMigrationAssessmentCollectorActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "collectors" {
		return false
	}
	collectorID, parsedAction, found := strings.Cut(strings.TrimSpace(tail[1]), ":")
	return found && strings.TrimSpace(collectorID) != "" && parsedAction == action
}

func parseGCPRapidMigrationAssessmentCollectorAction(tail []string) (collectorID, action string, ok bool) {
	if len(tail) != 2 || tail[0] != "collectors" {
		return "", "", false
	}
	collectorID, action, ok = strings.Cut(strings.TrimSpace(tail[1]), ":")
	if !ok {
		return "", "", false
	}
	collectorID = strings.TrimSpace(collectorID)
	action = strings.TrimSpace(action)
	if collectorID == "" {
		return "", "", false
	}
	switch action {
	case "pause", "resume", "register":
		return collectorID, action, true
	default:
		return "", "", false
	}
}

func isGCPRapidMigrationAssessmentAnnotationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "annotations"
}

func isGCPRapidMigrationAssessmentAnnotationTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "annotations" && strings.TrimSpace(tail[1]) != ""
}

func isGCPRapidMigrationAssessmentOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPRapidMigrationAssessmentOperationTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != ""
}

func parseGCPRapidMigrationAssessmentPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPRapidMigrationAssessmentInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPRapidMigrationAssessmentInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPRapidMigrationAssessmentList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPRapidMigrationAssessmentInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPRapidMigrationAssessmentJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPRapidMigrationAssessmentInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpRapidMigrationAssessmentBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func gcpRapidMigrationAssessmentString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func gcpRapidMigrationAssessmentValidAnnotationType(raw any) bool {
	switch typed := raw.(type) {
	case string:
		value := strings.TrimSpace(typed)
		return value != "" && value != "Annotation_TYPE_UNSPECIFIED"
	case float64:
		return typed > 0
	case int:
		return typed > 0
	default:
		return false
	}
}

func gcpRapidMigrationAssessmentLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Rapid Migration Assessment " + location,
	}
}

func gcpRapidMigrationAssessmentCollector(project, location, collectorID, state string) map[string]any {
	return map[string]any{
		"name":               fmt.Sprintf("projects/%s/locations/%s/collectors/%s", project, location, collectorID),
		"displayName":        "Stackyard Collector " + collectorID,
		"description":        "Stackyard staged Rapid Migration Assessment collector",
		"serviceAccount":     fmt.Sprintf("collector-%s@%s.iam.gserviceaccount.com", collectorID, project),
		"expectedAssetCount": 42,
		"state":              state,
		"collectionDays":     7,
		"eulaUri":            "https://example.com/stackyard/eula",
		"labels": map[string]string{
			"env": "staged",
		},
		"bucket":        "stackyard-rma-bucket",
		"clientVersion": "1.0.0",
	}
}

func gcpRapidMigrationAssessmentAnnotation(project, location, annotationID, annotationType string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/annotations/%s", project, location, annotationID),
		"type": annotationType,
		"labels": map[string]string{
			"source": "stackyard",
		},
	}
}

func gcpRapidMigrationAssessmentOperation(project, location, operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"done": false,
	}
}

func respondGCPRapidMigrationAssessmentInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func respondGCPRapidMigrationAssessmentFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "FailedPrecondition",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
