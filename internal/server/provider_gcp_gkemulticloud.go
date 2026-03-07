package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPGKEMultiCloudRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_gkemulticloud(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPGKEMultiCloudPath(path) {
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

func isGCPGKEMultiCloudPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.gkemulticloud.v1.AttachedClusters/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.gkemulticloud.v1.AwsClusters/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.gkemulticloud.v1.AzureClusters/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") || !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/attachedClusters") ||
		strings.Contains(path, "/attachedServerConfig") ||
		strings.Contains(path, "/awsClusters") ||
		strings.Contains(path, "/awsNodePools") ||
		strings.Contains(path, "/awsOpenIdConfig") ||
		strings.Contains(path, "/awsJsonWebKeys") ||
		strings.Contains(path, "/awsServerConfig") ||
		strings.Contains(path, "/azureClients") ||
		strings.Contains(path, "/azureClusters") ||
		strings.Contains(path, "/azureNodePools") ||
		strings.Contains(path, "/azureOpenIdConfig") ||
		strings.Contains(path, "/azureJsonWebKeys") ||
		strings.Contains(path, "/azureServerConfig") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":import") ||
		strings.Contains(path, ":rollback") ||
		strings.Contains(path, ":generateInstallManifest") ||
		strings.Contains(path, ":generateAttachedClusterInstallManifest") ||
		strings.Contains(path, ":generateAgentToken") ||
		strings.Contains(path, ":generateAttachedClusterAgentToken") ||
		strings.Contains(path, ":generateAwsClusterAgentToken") ||
		strings.Contains(path, ":generateAwsAccessToken") ||
		strings.Contains(path, ":generateAzureClusterAgentToken") ||
		strings.Contains(path, ":generateAzureAccessToken") ||
		strings.Contains(path, ":cancel")
}

func handleGCPContractProbe_gkemulticloud(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "gkemulticloud") {
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
			"name":     "projects/stackyard/locations/us-central1/gkemulticloud/sample",
			"service":  "gkemulticloud",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
