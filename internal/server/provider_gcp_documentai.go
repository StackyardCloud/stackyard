package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

var gcpDocumentAISupportedLocations = map[string]struct{}{
	"us": {},
	"eu": {},
}

func (s *Server) handleGCPDocumentAIRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPDocumentAIPath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.cloud.documentai.v1.DocumentProcessorService/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPDocumentAIGetLocation(w, path) {
			return true
		}
		if handleGCPDocumentAIListLocations(w, r, path) {
			return true
		}
		if handleGCPDocumentAIListProcessorTypes(w, r, path) {
			return true
		}
		if handleGCPDocumentAIGetProcessorType(w, path) {
			return true
		}
		if handleGCPDocumentAIListProcessors(w, r, path) {
			return true
		}
		if handleGCPDocumentAIGetProcessor(w, path) {
			return true
		}
		if handleGCPDocumentAIListProcessorVersions(w, r, path) {
			return true
		}
		if handleGCPDocumentAIGetProcessorVersion(w, path) {
			return true
		}
		if handleGCPDocumentAIListEvaluations(w, r, path) {
			return true
		}
		if handleGCPDocumentAIGetEvaluation(w, path) {
			return true
		}
		if handleGCPDocumentAIListOperations(w, r, path) {
			return true
		}
		if handleGCPDocumentAIGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPDocumentAIFetchProcessorTypes(w, path) {
			return true
		}
		if handleGCPDocumentAICreateProcessor(w, r, path) {
			return true
		}
		if handleGCPDocumentAIProcessDocument(w, r, path) {
			return true
		}
		if handleGCPDocumentAIBatchProcessDocuments(w, r, path) {
			return true
		}
		if handleGCPDocumentAITrainProcessorVersion(w, path) {
			return true
		}
		if handleGCPDocumentAIDeployProcessorVersion(w, path) {
			return true
		}
		if handleGCPDocumentAIUndeployProcessorVersion(w, path) {
			return true
		}
		if handleGCPDocumentAIEnableProcessor(w, path) {
			return true
		}
		if handleGCPDocumentAIDisableProcessor(w, path) {
			return true
		}
		if handleGCPDocumentAISetDefaultProcessorVersion(w, r, path) {
			return true
		}
		if handleGCPDocumentAIReviewDocument(w, r, path) {
			return true
		}
		if handleGCPDocumentAIEvaluateProcessorVersion(w, path) {
			return true
		}
		if handleGCPDocumentAICancelOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPDocumentAIDeleteProcessor(w, path) {
			return true
		}
		if handleGCPDocumentAIDeleteProcessorVersion(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPDocumentAIPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.documentai.v1.DocumentProcessorService/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}

	if _, _, tail, ok := parseGCPDocumentAILocationTail(path); ok {
		if len(tail) == 0 {
			return true
		}
		switch tail[0] {
		case "locations", "processorTypes", "processors", "operations":
			return true
		default:
			return false
		}
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 ||
		parts[0] != "gcp" ||
		parts[1] != "v1" ||
		parts[2] != "projects" ||
		parts[4] != "locations" ||
		strings.TrimSpace(parts[3]) == "" {
		return false
	}
	location, action, hasAction := strings.Cut(normalizeGCPDocumentAIActionSegment(parts[5]), ":")
	if !hasAction || strings.TrimSpace(location) == "" {
		return false
	}
	if !isGCPDocumentAISupportedLocation(location) {
		return false
	}
	return action == "fetchProcessorTypes"
}

func isGCPDocumentAISupportedLocation(location string) bool {
	_, ok := gcpDocumentAISupportedLocations[strings.ToLower(strings.TrimSpace(location))]
	return ok
}

func handleGCPDocumentAIGetLocation(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 0 {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDocumentAILocation(project, location))
	return true
}

func handleGCPDocumentAIListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "locations" {
		return false
	}
	pageSize, start, valid := parseGCPDocumentAIPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDocumentAILocation(project, location)}
	return respondGCPDocumentAIList(w, "locations", items, pageSize, start, path)
}

func handleGCPDocumentAIListProcessorTypes(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "processorTypes" {
		return false
	}
	pageSize, start, valid := parseGCPDocumentAIPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDocumentAIProcessorType(project, location, "FORM_PARSER_PROCESSOR")}
	return respondGCPDocumentAIList(w, "processorTypes", items, pageSize, start, path)
}

func handleGCPDocumentAIGetProcessorType(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "processorTypes" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDocumentAIProcessorType(project, location, tail[1]))
	return true
}

func handleGCPDocumentAIFetchProcessorTypes(w http.ResponseWriter, path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return false
	}
	project := strings.TrimSpace(parts[3])
	locationAction := normalizeGCPDocumentAIActionSegment(parts[5])
	location, action, hasAction := strings.Cut(locationAction, ":")
	if project == "" || !hasAction || strings.TrimSpace(location) == "" || !isGCPDocumentAISupportedLocation(location) || action != "fetchProcessorTypes" {
		return false
	}
	location = strings.ToLower(strings.TrimSpace(location))
	respondJSON(w, http.StatusOK, map[string]any{
		"processorTypes": []any{gcpDocumentAIProcessorType(project, location, "FORM_PARSER_PROCESSOR")},
	})
	return true
}

func handleGCPDocumentAIListProcessors(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "processors" {
		return false
	}
	pageSize, start, valid := parseGCPDocumentAIPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDocumentAIProcessor(project, location, "proc-1")}
	return respondGCPDocumentAIList(w, "processors", items, pageSize, start, path)
}

func handleGCPDocumentAICreateProcessor(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "processors" {
		return false
	}
	body, valid := decodeGCPDocumentAIJSONBody(w, r, path)
	if !valid {
		return true
	}
	processor := gcpDocumentAIBodyMap(body, "processor")
	if len(processor) == 0 {
		respondGCPDocumentAIInvalidArgument(w, path, "processor is required")
		return true
	}
	processorID := strings.TrimSpace(r.URL.Query().Get("processorId"))
	if processorID == "" {
		processorID = "proc-1"
	}
	respondJSON(w, http.StatusOK, gcpDocumentAIProcessor(project, location, processorID))
	return true
}

func handleGCPDocumentAIGetProcessor(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "processors" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDocumentAIProcessor(project, location, tail[1]))
	return true
}

func handleGCPDocumentAIDeleteProcessor(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "processors" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDocumentAIProcessDocument(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "processors" {
		return false
	}
	processorID, action, hasAction := strings.Cut(normalizeGCPDocumentAIActionSegment(tail[1]), ":")
	if !hasAction || strings.TrimSpace(processorID) == "" || action != "process" {
		return false
	}
	if _, valid := decodeGCPDocumentAIJSONBody(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"document": map[string]any{
			"mimeType": "application/pdf",
			"text":     "Stackyard processed document",
			"entities": []any{},
		},
		"humanReviewStatus": map[string]any{
			"state": "SKIPPED",
		},
		"name": fmt.Sprintf("projects/%s/locations/%s/processors/%s", project, location, processorID),
	})
	return true
}

func handleGCPDocumentAIBatchProcessDocuments(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "processors" {
		return false
	}
	processorID, action, hasAction := strings.Cut(normalizeGCPDocumentAIActionSegment(tail[1]), ":")
	if !hasAction || strings.TrimSpace(processorID) == "" || action != "batchProcess" {
		return false
	}
	if _, valid := decodeGCPDocumentAIJSONBody(w, r, path); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, gcpDocumentAIOperation(project, location, "batch-process-"+processorID))
	return true
}

func handleGCPDocumentAIListProcessorVersions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "processors" || strings.TrimSpace(tail[1]) == "" || tail[2] != "processorVersions" {
		return false
	}
	pageSize, start, valid := parseGCPDocumentAIPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDocumentAIProcessorVersion(project, location, tail[1], "ver-1")}
	return respondGCPDocumentAIList(w, "processorVersions", items, pageSize, start, path)
}

func handleGCPDocumentAIGetProcessorVersion(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "processors" || strings.TrimSpace(tail[1]) == "" || tail[2] != "processorVersions" || strings.TrimSpace(tail[3]) == "" || strings.Contains(tail[3], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDocumentAIProcessorVersion(project, location, tail[1], tail[3]))
	return true
}

func handleGCPDocumentAIDeleteProcessorVersion(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "processors" || strings.TrimSpace(tail[1]) == "" || tail[2] != "processorVersions" || strings.TrimSpace(tail[3]) == "" || strings.Contains(tail[3], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPDocumentAITrainProcessorVersion(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "processors" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	segment, action, hasAction := strings.Cut(normalizeGCPDocumentAIActionSegment(tail[2]), ":")
	if !hasAction || segment != "processorVersions" || action != "train" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDocumentAIOperation(project, location, "train-version-"+tail[1]))
	return true
}

func handleGCPDocumentAIDeployProcessorVersion(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "processors" || strings.TrimSpace(tail[1]) == "" || tail[2] != "processorVersions" {
		return false
	}
	versionID, action, hasAction := strings.Cut(normalizeGCPDocumentAIActionSegment(tail[3]), ":")
	if !hasAction || strings.TrimSpace(versionID) == "" || action != "deploy" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDocumentAIOperation(project, location, "deploy-version-"+versionID))
	return true
}

func handleGCPDocumentAIUndeployProcessorVersion(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "processors" || strings.TrimSpace(tail[1]) == "" || tail[2] != "processorVersions" {
		return false
	}
	versionID, action, hasAction := strings.Cut(normalizeGCPDocumentAIActionSegment(tail[3]), ":")
	if !hasAction || strings.TrimSpace(versionID) == "" || action != "undeploy" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDocumentAIOperation(project, location, "undeploy-version-"+versionID))
	return true
}

func handleGCPDocumentAIEnableProcessor(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "processors" {
		return false
	}
	processorID, action, hasAction := strings.Cut(normalizeGCPDocumentAIActionSegment(tail[1]), ":")
	if !hasAction || strings.TrimSpace(processorID) == "" || action != "enable" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDocumentAIOperation(project, location, "enable-processor-"+processorID))
	return true
}

func handleGCPDocumentAIDisableProcessor(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "processors" {
		return false
	}
	processorID, action, hasAction := strings.Cut(normalizeGCPDocumentAIActionSegment(tail[1]), ":")
	if !hasAction || strings.TrimSpace(processorID) == "" || action != "disable" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDocumentAIOperation(project, location, "disable-processor-"+processorID))
	return true
}

func handleGCPDocumentAISetDefaultProcessorVersion(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "processors" {
		return false
	}
	processorID, action, hasAction := strings.Cut(normalizeGCPDocumentAIActionSegment(tail[1]), ":")
	if !hasAction || strings.TrimSpace(processorID) == "" || action != "setDefaultProcessorVersion" {
		return false
	}
	body, valid := decodeGCPDocumentAIJSONBody(w, r, path)
	if !valid {
		return true
	}
	if strings.TrimSpace(stringFromMap(body, "defaultProcessorVersion")) == "" {
		respondGCPDocumentAIInvalidArgument(w, path, "defaultProcessorVersion is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDocumentAIOperation(project, location, "set-default-version-"+processorID))
	return true
}

func handleGCPDocumentAIReviewDocument(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 3 || tail[0] != "processors" || strings.TrimSpace(tail[1]) == "" {
		return false
	}
	reviewConfig, action, hasAction := strings.Cut(normalizeGCPDocumentAIActionSegment(tail[2]), ":")
	if !hasAction || reviewConfig != "humanReviewConfig" || action != "reviewDocument" {
		return false
	}
	body, valid := decodeGCPDocumentAIJSONBody(w, r, path)
	if !valid {
		return true
	}
	if _, ok := body["humanReviewConfig"]; !ok {
		respondGCPDocumentAIInvalidArgument(w, path, "humanReviewConfig is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpDocumentAIOperation(project, location, "review-document-"+tail[1]))
	return true
}

func handleGCPDocumentAIEvaluateProcessorVersion(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 4 || tail[0] != "processors" || strings.TrimSpace(tail[1]) == "" || tail[2] != "processorVersions" {
		return false
	}
	versionID, action, hasAction := strings.Cut(normalizeGCPDocumentAIActionSegment(tail[3]), ":")
	if !hasAction || strings.TrimSpace(versionID) == "" || action != "evaluateProcessorVersion" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDocumentAIOperation(project, location, "evaluate-version-"+versionID))
	return true
}

func handleGCPDocumentAIListEvaluations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 5 || tail[0] != "processors" || strings.TrimSpace(tail[1]) == "" || tail[2] != "processorVersions" || strings.TrimSpace(tail[3]) == "" || tail[4] != "evaluations" {
		return false
	}
	pageSize, start, valid := parseGCPDocumentAIPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDocumentAIEvaluation(project, location, tail[1], tail[3], "eval-1")}
	return respondGCPDocumentAIList(w, "evaluations", items, pageSize, start, path)
}

func handleGCPDocumentAIGetEvaluation(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 6 || tail[0] != "processors" || strings.TrimSpace(tail[1]) == "" || tail[2] != "processorVersions" || strings.TrimSpace(tail[3]) == "" || tail[4] != "evaluations" || strings.TrimSpace(tail[5]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDocumentAIEvaluation(project, location, tail[1], tail[3], tail[5]))
	return true
}

func handleGCPDocumentAIListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 1 || tail[0] != "operations" {
		return false
	}
	pageSize, start, valid := parseGCPDocumentAIPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpDocumentAIOperation(project, location, "op-1")}
	return respondGCPDocumentAIList(w, "operations", items, pageSize, start, path)
}

func handleGCPDocumentAIGetOperation(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpDocumentAIOperation(project, location, tail[1]))
	return true
}

func handleGCPDocumentAICancelOperation(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPDocumentAILocationTail(path)
	if !ok || len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	opID, action, hasAction := strings.Cut(normalizeGCPDocumentAIActionSegment(tail[1]), ":")
	if !hasAction || strings.TrimSpace(opID) == "" || action != "cancel" {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPDocumentAILocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.ToLower(strings.TrimSpace(parts[5]))
	if project == "" || location == "" || strings.Contains(location, ":") || !isGCPDocumentAISupportedLocation(location) {
		return "", "", nil, false
	}
	if len(parts) == 6 {
		return project, location, nil, true
	}
	return project, location, parts[6:], true
}

func parseGCPDocumentAIPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPDocumentAIInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPDocumentAIInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPDocumentAIList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPDocumentAIInvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPDocumentAIJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPDocumentAIInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpDocumentAIBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func normalizeGCPDocumentAIActionSegment(raw string) string {
	segment := strings.TrimSpace(raw)
	segment = strings.ReplaceAll(segment, "%3A", ":")
	segment = strings.ReplaceAll(segment, "%3a", ":")
	return segment
}

func gcpDocumentAILocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": strings.ToUpper(location),
	}
}

func gcpDocumentAIProcessorType(project, location, processorType string) map[string]any {
	return map[string]any{
		"name":          fmt.Sprintf("projects/%s/locations/%s/processorTypes/%s", project, location, processorType),
		"type":          processorType,
		"category":      "GENERAL",
		"allowCreation": true,
	}
}

func gcpDocumentAIProcessor(project, location, processorID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/processors/%s", project, location, processorID),
		"type":        "FORM_PARSER_PROCESSOR",
		"displayName": "stackyard-processor",
		"state":       "ENABLED",
	}
}

func gcpDocumentAIProcessorVersion(project, location, processorID, versionID string) map[string]any {
	return map[string]any{
		"name":                    fmt.Sprintf("projects/%s/locations/%s/processors/%s/processorVersions/%s", project, location, processorID, versionID),
		"displayName":             versionID,
		"state":                   "DEPLOYED",
		"defaultProcessorVersion": fmt.Sprintf("projects/%s/locations/%s/processors/%s/processorVersions/%s", project, location, processorID, versionID),
	}
}

func gcpDocumentAIEvaluation(project, location, processorID, versionID, evaluationID string) map[string]any {
	return map[string]any{
		"name":       fmt.Sprintf("projects/%s/locations/%s/processors/%s/processorVersions/%s/evaluations/%s", project, location, processorID, versionID, evaluationID),
		"kmsKeyName": "",
		"documentCounters": map[string]any{
			"inputDocumentsCount": "1",
		},
	}
}

func gcpDocumentAIOperation(project, location, operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"done": true,
	}
}

func respondGCPDocumentAIInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
