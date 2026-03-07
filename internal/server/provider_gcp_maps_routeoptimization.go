package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (s *Server) handleGCPMapsRouteOptimizationRouter(w http.ResponseWriter, r *http.Request) bool {
	path := strings.TrimSpace(r.URL.Path)
	if path == "" {
		path = rawRequestPath(r)
	}
	if !isGCPMapsRouteOptimizationPath(path) {
		return false
	}
	if !shouldHandleGCPMapsRouteOptimizationRequest(r) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.maps.routeoptimization.v1.RouteOptimization/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	switch r.Method {
	case http.MethodPost:
		if handleGCPMapsRouteOptimizationOptimizeTours(w, r, path) {
			return true
		}
		if handleGCPMapsRouteOptimizationBatchOptimizeTours(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodGet:
		if handleGCPMapsRouteOptimizationGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPMapsRouteOptimizationPath(path string) bool {
	normalizedPath := normalizeGCPMapsRouteOptimizationPath(path)

	if strings.HasPrefix(normalizedPath, "/gcp/google.maps.routeoptimization.v1.RouteOptimization/") {
		return true
	}

	if strings.HasPrefix(normalizedPath, "/gcp/v1/operations/") {
		return true
	}

	if !strings.HasPrefix(normalizedPath, "/gcp/v1/projects/") {
		return false
	}

	return strings.Contains(normalizedPath, ":optimizeTours") ||
		strings.Contains(normalizedPath, ":batchOptimizeTours")
}

func handleGCPMapsRouteOptimizationOptimizeTours(w http.ResponseWriter, r *http.Request, path string) bool {
	_, ok := parseGCPMapsRouteOptimizationParentActionPath(path, "optimizeTours")
	if !ok {
		return false
	}

	body, valid := decodeGCPMapsRouteOptimizationJSONBody(w, r, path)
	if !valid {
		return true
	}

	modelRaw, exists := body["model"]
	if !exists {
		respondGCPMapsRouteOptimizationInvalidArgument(w, path, "model is required")
		return true
	}
	if _, ok := modelRaw.(map[string]any); !ok {
		respondGCPMapsRouteOptimizationInvalidArgument(w, path, "model is required")
		return true
	}

	label, _ := body["label"].(string)
	if strings.TrimSpace(label) == "" {
		label = "stackyard-routeoptimization"
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"routes": []any{
			map[string]any{},
		},
		"requestLabel":     label,
		"skippedShipments": []any{},
		"metrics": map[string]any{
			"aggregatedRouteMetrics": map[string]any{
				"travelDistanceMeters": 7421,
			},
		},
		"validationErrors": []any{},
	})
	return true
}

func handleGCPMapsRouteOptimizationBatchOptimizeTours(w http.ResponseWriter, r *http.Request, path string) bool {
	parent, ok := parseGCPMapsRouteOptimizationParentActionPath(path, "batchOptimizeTours")
	if !ok {
		return false
	}

	body, valid := decodeGCPMapsRouteOptimizationJSONBody(w, r, path)
	if !valid {
		return true
	}

	modelConfigs, _ := body["modelConfigs"].([]any)
	if len(modelConfigs) == 0 {
		respondGCPMapsRouteOptimizationInvalidArgument(w, path, "modelConfigs must be a non-empty array")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"name": fmt.Sprintf("operations/routeopt-%s-batch-1", gcpMapsRouteOptimizationResourceID(parent)),
		"done": false,
	})
	return true
}

func handleGCPMapsRouteOptimizationGetOperation(w http.ResponseWriter, path string) bool {
	operationID, ok := parseGCPMapsRouteOptimizationOperationPath(path)
	if !ok {
		return false
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"name": fmt.Sprintf("operations/%s", operationID),
		"done": false,
	})
	return true
}

func decodeGCPMapsRouteOptimizationJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	var body map[string]any
	if r.Body == nil {
		return map[string]any{}, true
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPMapsRouteOptimizationInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func parseGCPMapsRouteOptimizationParentActionPath(path, expectedAction string) (parent string, ok bool) {
	normalizedPath := normalizeGCPMapsRouteOptimizationPath(path)
	if !strings.HasPrefix(normalizedPath, "/gcp/v1/") {
		return "", false
	}
	actionSuffix := ":" + expectedAction
	if !strings.HasSuffix(normalizedPath, actionSuffix) {
		return "", false
	}

	resource := strings.TrimPrefix(normalizedPath, "/gcp/v1/")
	resource = strings.TrimSuffix(resource, actionSuffix)
	if !strings.HasPrefix(resource, "projects/") {
		return "", false
	}

	parts := strings.Split(resource, "/")
	// projects/{project}
	if len(parts) == 2 && parts[0] == "projects" {
		if strings.TrimSpace(parts[1]) == "" {
			return "", false
		}
		return resource, true
	}
	// projects/{project}/locations/{location}
	if len(parts) >= 4 && parts[0] == "projects" && parts[2] == "locations" {
		if strings.TrimSpace(parts[1]) == "" || strings.TrimSpace(parts[3]) == "" {
			return "", false
		}
		return strings.Join(parts[:4], "/"), true
	}
	return "", false
}

func parseGCPMapsRouteOptimizationOperationPath(path string) (operationID string, ok bool) {
	normalizedPath := normalizeGCPMapsRouteOptimizationPath(path)
	if !strings.HasPrefix(normalizedPath, "/gcp/v1/operations/") {
		return "", false
	}

	operationID = strings.TrimSpace(strings.TrimPrefix(normalizedPath, "/gcp/v1/operations/"))
	operationID = strings.Trim(operationID, "/")
	if operationID == "" {
		return "", false
	}
	return operationID, true
}

func gcpMapsRouteOptimizationResourceID(parent string) string {
	replacer := strings.NewReplacer(
		"projects/", "",
		"locations/", "",
		"/", "-",
	)
	return replacer.Replace(parent)
}

func normalizeGCPMapsRouteOptimizationPath(path string) string {
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func respondGCPMapsRouteOptimizationInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func shouldHandleGCPMapsRouteOptimizationRequest(r *http.Request) bool {
	path := normalizeGCPMapsRouteOptimizationPath(rawRequestPath(r))

	if strings.HasPrefix(path, "/gcp/google.maps.routeoptimization.v1.RouteOptimization/") {
		return true
	}
	if strings.Contains(path, "/operations/routeopt-") {
		return true
	}
	if isGCPMapsRouteOptimizationClient(r.Header.Get("x-goog-api-client")) {
		return true
	}

	if r.Method != http.MethodPost {
		return false
	}
	if !strings.Contains(path, ":optimizeTours") && !strings.Contains(path, ":batchOptimizeTours") {
		return false
	}
	if r.Body == nil {
		return false
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	lowerBody := strings.ToLower(string(bodyBytes))
	return strings.Contains(lowerBody, "routeopt") || strings.Contains(lowerBody, "routeoptimization")
}
