package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPNetworkServicesRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_networkservices(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPNetworkServicesPath(path) {
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

func isGCPNetworkServicesPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.networkservices.v1.NetworkServices/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.networkservices.v1.DepService/") {
		return true
	}

	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if isGCPNetworkServicesLocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/endpointPolicies") ||
		strings.Contains(path, "/wasmPlugins") ||
		strings.Contains(path, "/versions") ||
		strings.Contains(path, "/gateways") ||
		strings.Contains(path, "/grpcRoutes") ||
		strings.Contains(path, "/httpRoutes") ||
		strings.Contains(path, "/tcpRoutes") ||
		strings.Contains(path, "/tlsRoutes") ||
		strings.Contains(path, "/serviceBindings") ||
		strings.Contains(path, "/meshes") ||
		strings.Contains(path, "/serviceLbPolicies") ||
		strings.Contains(path, "/routeViews") ||
		strings.Contains(path, "/lbTrafficExtensions") ||
		strings.Contains(path, "/lbRouteExtensions") ||
		strings.Contains(path, "/lbEdgeExtensions") ||
		strings.Contains(path, "/authzExtensions") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":getIamPolicy") ||
		strings.Contains(path, ":setIamPolicy") ||
		strings.Contains(path, ":testIamPermissions") ||
		strings.Contains(path, ":cancel")
}

func isGCPNetworkServicesLocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func handleGCPContractProbe_networkservices(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "networkservices") {
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
			"name":     "projects/stackyard/locations/us-central1/networkservices/sample",
			"service":  "networkservices",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
