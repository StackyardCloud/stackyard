package server

import (
	"net/http"
	"strings"
)

const azureAppConfigurationDataPlanePrefix = "/azure/appconfiguration/"

func (s *Server) handleAzureAppConfigurationDataPlaneRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureAppConfigurationDataPlanePrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureAppConfigurationDataPlanePrefix)
	normalized := strings.Trim(strings.TrimSpace(relative), "/")
	if normalized == "" {
		return false
	}

	segments := splitPathSegments(normalized)
	if len(segments) == 0 {
		return false
	}

	if !azureAppConfigurationDataPlaneAllowedMethod(r.Method) {
		return false
	}

	if handleAzureAppConfigurationDataPlaneKV(w, path, r.Method, segments) {
		return true
	}
	if handleAzureAppConfigurationDataPlaneCollections(w, path, segments) {
		return true
	}
	if handleAzureAppConfigurationDataPlaneSnapshots(w, path, r.Method, segments) {
		return true
	}
	if handleAzureAppConfigurationDataPlaneLocks(w, path, r.Method, segments) {
		return true
	}
	if handleAzureAppConfigurationDataPlaneOperations(w, path, r.Method, segments) {
		return true
	}

	respondAzureImplemented(w, path)
	return true
}

func azureAppConfigurationDataPlaneAllowedMethod(method string) bool {
	switch method {
	case http.MethodHead, http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func handleAzureAppConfigurationDataPlaneKV(w http.ResponseWriter, path, method string, segments []string) bool {
	if len(segments) == 1 && strings.EqualFold(segments[0], "kv") && (method == http.MethodHead || method == http.MethodGet) {
		respondAzureImplemented(w, path)
		return true
	}
	if len(segments) == 2 && strings.EqualFold(segments[0], "kv") && strings.TrimSpace(segments[1]) != "" {
		switch method {
		case http.MethodHead, http.MethodGet, http.MethodPut, http.MethodDelete:
			respondAzureImplemented(w, path)
			return true
		}
	}
	return false
}

func handleAzureAppConfigurationDataPlaneCollections(w http.ResponseWriter, path string, segments []string) bool {
	if len(segments) != 1 {
		return false
	}
	switch {
	case strings.EqualFold(segments[0], "keys"):
		respondAzureImplemented(w, path)
		return true
	case strings.EqualFold(segments[0], "labels"):
		respondAzureImplemented(w, path)
		return true
	case strings.EqualFold(segments[0], "revisions"):
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureAppConfigurationDataPlaneSnapshots(w http.ResponseWriter, path, method string, segments []string) bool {
	if len(segments) == 1 && strings.EqualFold(segments[0], "snapshots") && (method == http.MethodHead || method == http.MethodGet) {
		respondAzureImplemented(w, path)
		return true
	}
	if len(segments) == 2 && strings.EqualFold(segments[0], "snapshots") && strings.TrimSpace(segments[1]) != "" {
		switch method {
		case http.MethodHead, http.MethodGet, http.MethodPut, http.MethodPatch:
			respondAzureImplemented(w, path)
			return true
		}
	}
	return false
}

func handleAzureAppConfigurationDataPlaneLocks(w http.ResponseWriter, path, method string, segments []string) bool {
	if len(segments) != 2 || !strings.EqualFold(segments[0], "locks") || strings.TrimSpace(segments[1]) == "" {
		return false
	}
	switch method {
	case http.MethodPut, http.MethodDelete:
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureAppConfigurationDataPlaneOperations(w http.ResponseWriter, path, method string, segments []string) bool {
	if len(segments) != 1 || !strings.EqualFold(segments[0], "operations") || method != http.MethodGet {
		return false
	}
	respondAzureImplemented(w, path)
	return true
}
