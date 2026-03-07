package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPAiplatformRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPAiplatformPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPAiplatformListResources(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodPut:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPAiplatformPath(path string) bool {
	return strings.HasPrefix(path, "/gcp/aiplatform/v1/") ||
		strings.HasPrefix(path, "/google.cloud.aiplatform.v1.")
}

func handleGCPAiplatformListResources(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, collection, ok := parseGCPAiplatformCollectionPath(path)
	if !ok {
		return false
	}

	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageSize must be a non-negative integer",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	start := 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = parseOptionalNonNegativeInt(pageToken)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "pageToken must be a non-negative integer offset",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
	}

	items, field := gcpAiplatformCollectionItems(project, location, collection)
	if start > len(items) {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageToken is out of range",
			"provider": providerGCP,
			"path":     path,
		})
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
		field:           items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func parseGCPAiplatformCollectionPath(path string) (project, location, collection string, ok bool) {
	trimmed := strings.Trim(path, "/")
	parts := strings.Split(trimmed, "/")
	// /gcp/aiplatform/v1/projects/{project}/locations/{location}/{collection}
	if len(parts) != 8 {
		return "", "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "aiplatform" || parts[2] != "v1" || parts[3] != "projects" || parts[5] != "locations" {
		return "", "", "", false
	}

	project = strings.TrimSpace(parts[4])
	location = strings.TrimSpace(parts[6])
	collection = strings.TrimSpace(parts[7])
	if project == "" || location == "" {
		return "", "", "", false
	}
	switch collection {
	case "datasets", "models", "endpoints", "customJobs":
		return project, location, collection, true
	default:
		return "", "", "", false
	}
}

func gcpAiplatformCollectionItems(project, location, collection string) ([]map[string]any, string) {
	parent := fmt.Sprintf("projects/%s/locations/%s", project, location)
	switch collection {
	case "datasets":
		return []map[string]any{
			{
				"name":        parent + "/datasets/ds-1",
				"displayName": "stackyard-dataset",
			},
		}, "datasets"
	case "models":
		return []map[string]any{
			{
				"name":        parent + "/models/model-1",
				"displayName": "stackyard-model",
			},
		}, "models"
	case "endpoints":
		return []map[string]any{
			{
				"name":        parent + "/endpoints/endpoint-1",
				"displayName": "stackyard-endpoint",
			},
		}, "endpoints"
	case "customJobs":
		return []map[string]any{
			{
				"name":         parent + "/customJobs/job-1",
				"displayName":  "stackyard-job",
				"stateMessage": "SUCCEEDED",
			},
		}, "customJobs"
	default:
		return []map[string]any{}, "items"
	}
}
