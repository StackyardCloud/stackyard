package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPAutoMLRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPAutoMLPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPAutoMLListDatasets(w, r, path) {
			return true
		}
		if handleGCPAutoMLGetDataset(w, path) {
			return true
		}
		if handleGCPAutoMLListModels(w, r, path) {
			return true
		}
		if handleGCPAutoMLGetModel(w, path) {
			return true
		}
		if handleGCPAutoMLListModelEvaluations(w, r, path) {
			return true
		}
		if handleGCPAutoMLGetModelEvaluation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPAutoMLPredict(w, r, path) {
			return true
		}
		if handleGCPAutoMLBatchPredict(w, r, path) {
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

func isGCPAutoMLPath(path string) bool {
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/datasets") ||
		strings.Contains(path, "/models") ||
		strings.Contains(path, "/modelEvaluations") ||
		strings.Contains(path, "/annotationSpecs") ||
		strings.Contains(path, ":importData") ||
		strings.Contains(path, ":exportData") ||
		strings.Contains(path, ":deploy") ||
		strings.Contains(path, ":undeploy") ||
		(strings.Contains(path, "/models/") && strings.Contains(path, ":export")) ||
		strings.Contains(path, ":predict") ||
		strings.Contains(path, ":batchPredict")
}

func handleGCPAutoMLListDatasets(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPAutoMLCollectionParentPath(path, "datasets")
	if !ok {
		return false
	}

	pageSize, start, valid := parseGCPAutoMLPagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpAutoMLDataset(project, location, "team-dataset"),
	}
	return respondGCPAutoMLList(w, "dataset", items, pageSize, start, path)
}

func handleGCPAutoMLGetDataset(w http.ResponseWriter, path string) bool {
	project, location, datasetID, ok := parseGCPAutoMLResourcePath(path, "datasets")
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpAutoMLDataset(project, location, datasetID))
	return true
}

func handleGCPAutoMLListModels(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPAutoMLCollectionParentPath(path, "models")
	if !ok {
		return false
	}

	pageSize, start, valid := parseGCPAutoMLPagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpAutoMLModel(project, location, "team-model"),
	}
	return respondGCPAutoMLList(w, "model", items, pageSize, start, path)
}

func handleGCPAutoMLGetModel(w http.ResponseWriter, path string) bool {
	project, location, modelID, ok := parseGCPAutoMLResourcePath(path, "models")
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpAutoMLModel(project, location, modelID))
	return true
}

func handleGCPAutoMLListModelEvaluations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, modelID, ok := parseGCPAutoMLModelEvaluationsCollectionPath(path)
	if !ok {
		return false
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("filter")); raw == "" && r.URL.Query().Has("filter") {
		respondGCPAutoMLInvalidArgument(w, path, "filter must not be empty when provided")
		return true
	}

	pageSize, start, valid := parseGCPAutoMLPagination(w, r, path)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpAutoMLModelEvaluation(project, location, modelID, "eval-1"),
	}
	return respondGCPAutoMLList(w, "modelEvaluation", items, pageSize, start, path)
}

func handleGCPAutoMLGetModelEvaluation(w http.ResponseWriter, path string) bool {
	project, location, modelID, evaluationID, ok := parseGCPAutoMLModelEvaluationPath(path)
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, gcpAutoMLModelEvaluation(project, location, modelID, evaluationID))
	return true
}

func handleGCPAutoMLPredict(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, modelID, ok := parseGCPAutoMLModelActionPath(path, "predict")
	if !ok {
		return false
	}

	body, valid := decodeGCPAutoMLJSONBody(w, r, path)
	if !valid {
		return true
	}
	payload, _ := body["payload"].(map[string]any)
	if len(payload) == 0 {
		respondGCPAutoMLInvalidArgument(w, path, "payload is required")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"payload": []any{
			map[string]any{
				"displayName": "positive",
				"classification": map[string]any{
					"score": 0.91,
				},
			},
		},
		"metadata": map[string]any{
			"model": fmt.Sprintf("projects/%s/locations/%s/models/%s", project, location, modelID),
		},
	})
	return true
}

func handleGCPAutoMLBatchPredict(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, modelID, ok := parseGCPAutoMLModelActionPath(path, "batchPredict")
	if !ok {
		return false
	}

	body, valid := decodeGCPAutoMLJSONBody(w, r, path)
	if !valid {
		return true
	}
	inputConfig, _ := body["inputConfig"].(map[string]any)
	if len(inputConfig) == 0 {
		respondGCPAutoMLInvalidArgument(w, path, "inputConfig is required")
		return true
	}
	outputConfig, _ := body["outputConfig"].(map[string]any)
	if len(outputConfig) == 0 {
		respondGCPAutoMLInvalidArgument(w, path, "outputConfig is required")
		return true
	}

	respondJSON(w, http.StatusOK, gcpAutoMLOperation(
		fmt.Sprintf("operations/automl.batchPredict.%s.%s.%s", project, location, modelID),
	))
	return true
}

func decodeGCPAutoMLJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	var body map[string]any
	if r.Body == nil {
		return map[string]any{}, true
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPAutoMLInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func parseGCPAutoMLCollectionParentPath(path, collection string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != collection {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPAutoMLResourcePath(path, collection string) (project, location, resourceID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != collection {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	resourceID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || resourceID == "" {
		return "", "", "", false
	}
	return project, location, resourceID, true
}

func parseGCPAutoMLModelEvaluationsCollectionPath(path string) (project, location, modelID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "models" || parts[8] != "modelEvaluations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	modelID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || modelID == "" {
		return "", "", "", false
	}
	return project, location, modelID, true
}

func parseGCPAutoMLModelEvaluationPath(path string) (project, location, modelID, evaluationID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "models" || parts[8] != "modelEvaluations" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	modelID = strings.TrimSpace(parts[7])
	evaluationID = strings.TrimSpace(parts[9])
	if project == "" || location == "" || modelID == "" || evaluationID == "" {
		return "", "", "", "", false
	}
	return project, location, modelID, evaluationID, true
}

func parseGCPAutoMLModelActionPath(path, action string) (project, location, modelID string, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 8 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "models" {
		return "", "", "", false
	}

	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	modelAndAction := strings.TrimSpace(parts[7])
	modelID, actionName, found := strings.Cut(modelAndAction, ":")
	if !found || !strings.EqualFold(strings.TrimSpace(actionName), action) {
		return "", "", "", false
	}
	modelID = strings.TrimSpace(modelID)
	if project == "" || location == "" || modelID == "" {
		return "", "", "", false
	}
	return project, location, modelID, true
}

func parseGCPAutoMLPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPAutoMLInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}

	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPAutoMLInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPAutoMLList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPAutoMLInvalidArgument(w, path, "pageToken is out of range")
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

func gcpAutoMLDataset(project, location, datasetID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s/datasets/%s", project, location, datasetID),
		"displayName": "Team Dataset",
		"etag":        "stackyard-dataset-etag",
	}
}

func gcpAutoMLModel(project, location, modelID string) map[string]any {
	return map[string]any{
		"name":            fmt.Sprintf("projects/%s/locations/%s/models/%s", project, location, modelID),
		"displayName":     "Team Model",
		"deploymentState": "DEPLOYED",
	}
}

func gcpAutoMLModelEvaluation(project, location, modelID, evaluationID string) map[string]any {
	return map[string]any{
		"name":             fmt.Sprintf("projects/%s/locations/%s/models/%s/modelEvaluations/%s", project, location, modelID, evaluationID),
		"annotationSpecId": "positive",
	}
}

func gcpAutoMLOperation(name string) map[string]any {
	return map[string]any{
		"name": name,
		"done": true,
	}
}

func respondGCPAutoMLInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
