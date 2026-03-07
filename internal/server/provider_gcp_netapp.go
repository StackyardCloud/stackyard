package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPNetAppRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_netapp(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPNetAppPath(path) {
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

func isGCPNetAppPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.netapp.v1.NetApp/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/activeDirectories") ||
		strings.Contains(path, "/storagePools") ||
		strings.Contains(path, "/volumes") ||
		strings.Contains(path, "/snapshots") ||
		strings.Contains(path, "/replications") ||
		strings.Contains(path, "/backups") ||
		strings.Contains(path, "/backupPolicies") ||
		strings.Contains(path, "/backupVaults") ||
		strings.Contains(path, "/quotaRules") ||
		strings.Contains(path, "/hostGroups") ||
		strings.Contains(path, "/kmsConfigs") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":revert") ||
		strings.Contains(path, ":stop") ||
		strings.Contains(path, ":resume") ||
		strings.Contains(path, ":sync") ||
		strings.Contains(path, ":verify") ||
		strings.Contains(path, ":establishPeering") ||
		strings.Contains(path, ":reverseDirection")
}

func handleGCPContractProbe_netapp(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "netapp") {
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
			"name":     "projects/stackyard/locations/us-central1/netapp/sample",
			"service":  "netapp",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
