package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPDataformRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_dataform(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !isGCPDataformPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPDataformPath(path string) bool {
	return strings.Contains(path, "/repositories") ||
		strings.Contains(path, "/workspaces") ||
		strings.Contains(path, "/releaseConfigs") ||
		strings.Contains(path, "/compilationResults") ||
		strings.Contains(path, "/workflowConfigs") ||
		strings.Contains(path, "/workflowInvocations") ||
		strings.Contains(path, "/config") ||
		strings.Contains(path, ":commit") ||
		strings.Contains(path, ":readFile") ||
		strings.Contains(path, ":queryDirectoryContents") ||
		strings.Contains(path, ":fetchHistory") ||
		strings.Contains(path, ":computeAccessTokenStatus") ||
		strings.Contains(path, ":fetchRemoteBranches") ||
		strings.Contains(path, ":installNpmPackages") ||
		strings.Contains(path, ":pull") ||
		strings.Contains(path, ":push") ||
		strings.Contains(path, ":fetchFileGitStatuses") ||
		strings.Contains(path, ":fetchGitAheadBehind") ||
		strings.Contains(path, ":reset") ||
		strings.Contains(path, ":fetchFileDiff") ||
		strings.Contains(path, ":searchFiles") ||
		strings.Contains(path, ":makeDirectory") ||
		strings.Contains(path, ":removeDirectory") ||
		strings.Contains(path, ":moveDirectory") ||
		strings.Contains(path, ":removeFile") ||
		strings.Contains(path, ":moveFile") ||
		strings.Contains(path, ":writeFile") ||
		strings.Contains(path, ":query") ||
		strings.Contains(path, ":cancel") ||
		strings.Contains(path, ":getIamPolicy") ||
		strings.Contains(path, ":setIamPolicy") ||
		strings.Contains(path, ":testIamPermissions")
}

func handleGCPContractProbe_dataform(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "dataform") {
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
			"name":     "projects/stackyard/locations/us-central1/dataform/sample",
			"service":  "dataform",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
