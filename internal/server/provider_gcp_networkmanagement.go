package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPNetworkManagementRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_networkmanagement(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPNetworkManagementPath(path) {
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

func isGCPNetworkManagementPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.networkmanagement.v1.ReachabilityService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.networkmanagement.v1.VpcFlowLogsService/") {
		return true
	}

	isProjectScope := strings.HasPrefix(path, "/gcp/v1/projects/") && strings.Contains(path, "/locations/global/")
	isOrgScope := strings.HasPrefix(path, "/gcp/v1/organizations/") && strings.Contains(path, "/locations/global/")
	if !isProjectScope && !isOrgScope {
		return false
	}

	return strings.Contains(path, "/connectivityTests") ||
		strings.Contains(path, "/vpcFlowLogsConfigs") ||
		strings.Contains(path, ":rerun")
}

func handleGCPContractProbe_networkmanagement(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "networkmanagement") {
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
			"name":     "projects/stackyard/locations/us-central1/networkmanagement/sample",
			"service":  "networkmanagement",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
