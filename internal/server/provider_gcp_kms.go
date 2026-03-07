package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPKMSRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_kms(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPKMSPath(path) {
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

func isGCPKMSPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.kms.v1.KeyManagementService/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.kms.v1.EkmService/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.kms.v1.Autokey/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.kms.v1.AutokeyAdmin/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.kms.v1.HsmManagement/") {
		return true
	}

	if !(strings.HasPrefix(path, "/gcp/v1/projects/") ||
		strings.HasPrefix(path, "/gcp/v1/organizations/") ||
		strings.HasPrefix(path, "/gcp/v1/folders/")) {
		return false
	}

	return strings.Contains(path, "/keyRings") ||
		strings.Contains(path, "/cryptoKeys") ||
		strings.Contains(path, "/cryptoKeyVersions") ||
		strings.Contains(path, "/importJobs") ||
		strings.Contains(path, "/retiredResources") ||
		strings.Contains(path, "/publicKey") ||
		strings.Contains(path, "/ekmConnections") ||
		strings.Contains(path, "/ekmConfig") ||
		strings.Contains(path, "/keyHandles") ||
		strings.Contains(path, "/autokeyConfig") ||
		strings.Contains(path, "/singleTenantHsmInstances") ||
		strings.Contains(path, "/singleTenantHsmInstanceProposals") ||
		strings.Contains(path, ":updatePrimaryVersion") ||
		strings.Contains(path, ":destroy") ||
		strings.Contains(path, ":restore") ||
		strings.Contains(path, ":encrypt") ||
		strings.Contains(path, ":decrypt") ||
		strings.Contains(path, ":rawEncrypt") ||
		strings.Contains(path, ":rawDecrypt") ||
		strings.Contains(path, ":asymmetricSign") ||
		strings.Contains(path, ":asymmetricDecrypt") ||
		strings.Contains(path, ":macSign") ||
		strings.Contains(path, ":macVerify") ||
		strings.Contains(path, ":decapsulate") ||
		strings.Contains(path, ":generateRandomBytes") ||
		strings.Contains(path, ":verifyConnectivity") ||
		strings.Contains(path, ":setIamPolicy") ||
		strings.Contains(path, ":getIamPolicy") ||
		strings.Contains(path, ":testIamPermissions")
}

func handleGCPContractProbe_kms(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "kms") {
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
			"name":     "projects/stackyard/locations/us-central1/kms/sample",
			"service":  "kms",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
