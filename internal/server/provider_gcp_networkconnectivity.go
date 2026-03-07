package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPNetworkConnectivityRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_networkconnectivity(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPNetworkConnectivityPath(path) {
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

func isGCPNetworkConnectivityPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.networkconnectivity.v1.HubService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.networkconnectivity.v1.CrossNetworkAutomationService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.networkconnectivity.v1.InternalRangeService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.networkconnectivity.v1.PolicyBasedRoutingService/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.networkconnectivity.v1.DataTransferService/") {
		return true
	}

	if !strings.HasPrefix(path, "/gcp/v1/projects/") || !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/hubs") ||
		strings.Contains(path, "/spokes") ||
		strings.Contains(path, "/groups") ||
		strings.Contains(path, "/routeTables") ||
		strings.Contains(path, "/routes") ||
		strings.Contains(path, "/serviceConnectionMaps") ||
		strings.Contains(path, "/serviceConnectionPolicies") ||
		strings.Contains(path, "/serviceConnectionTokens") ||
		strings.Contains(path, "/serviceClasses") ||
		strings.Contains(path, "/internalRanges") ||
		strings.Contains(path, "/policyBasedRoutes") ||
		strings.Contains(path, "/multicloudDataTransferConfigs") ||
		strings.Contains(path, "/multicloudDataTransferSupportedServices") ||
		strings.Contains(path, "/destinations") ||
		strings.Contains(path, ":acceptSpoke") ||
		strings.Contains(path, ":rejectSpoke") ||
		strings.Contains(path, ":queryStatus")
}

func handleGCPContractProbe_networkconnectivity(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "networkconnectivity") {
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
			"name":     "projects/stackyard/locations/us-central1/networkconnectivity/sample",
			"service":  "networkconnectivity",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
