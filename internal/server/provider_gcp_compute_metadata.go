package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPComputeMetadataRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_compute_metadata(w, r) {
		return true
	}

	path := rawRequestPath(r)
	flavor := strings.TrimSpace(r.Header.Get("Metadata-Flavor"))

	if path == "/" {
		if !strings.EqualFold(flavor, "Google") {
			return false
		}
		if !s.providerEnabled(providerGCP) {
			respondProviderDisabled(w, providerGCP, s.enabledProviders)
			return true
		}
		w.Header().Set("Metadata-Flavor", "Google")
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			return true
		}
		return false
	}

	if !isGCPComputeMetadataPath(path) {
		return false
	}

	if !s.providerEnabled(providerGCP) {
		respondProviderDisabled(w, providerGCP, s.enabledProviders)
		return true
	}

	if !strings.EqualFold(flavor, "Google") {
		respondJSON(w, http.StatusForbidden, map[string]any{
			"error":    "MetadataFlavorRequired",
			"message":  "missing required Metadata-Flavor: Google header",
			"provider": providerGCP,
		})
		return true
	}

	w.Header().Set("Metadata-Flavor", "Google")
	switch r.Method {
	case http.MethodGet:
		if body, ok := gcpComputeMetadataResponseBody(path); ok {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(body))
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPComputeMetadataPath(path string) bool {
	return path == "/computeMetadata/v1" ||
		strings.HasPrefix(path, "/computeMetadata/v1/")
}

func gcpComputeMetadataResponseBody(path string) (string, bool) {
	switch path {
	case "/computeMetadata/v1/project/project-id":
		return "stackyard", true
	case "/computeMetadata/v1/project/numeric-project-id":
		return "123456789012", true
	case "/computeMetadata/v1/instance/id":
		return "9876543210987654321", true
	case "/computeMetadata/v1/instance/name":
		return "stackyard-instance", true
	case "/computeMetadata/v1/instance/zone":
		return "projects/123456789012/zones/us-central1-a", true
	case "/computeMetadata/v1/instance/hostname":
		return "stackyard-instance.c.stackyard.internal", true
	case "/computeMetadata/v1/instance/network-interfaces/0/ip":
		return "10.10.0.12", true
	case "/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip":
		return "34.82.10.12", true
	case "/computeMetadata/v1/instance/service-accounts/default/email":
		return "stackyard-sa@stackyard.iam.gserviceaccount.com", true
	case "/computeMetadata/v1/instance/service-accounts/default/scopes":
		return "https://www.googleapis.com/auth/cloud-platform\nhttps://www.googleapis.com/auth/devstorage.read_write", true
	case "/computeMetadata/v1/project/attributes/":
		return "env\nowner", true
	case "/computeMetadata/v1/project/attributes/env":
		return "dev", true
	case "/computeMetadata/v1/instance/attributes/":
		return "role\nteam", true
	case "/computeMetadata/v1/instance/attributes/role":
		return "worker", true
	case "/computeMetadata/v1/instance/tags":
		return `["stackyard","gcp","compute"]`, true
	default:
		return "", false
	}
}

func handleGCPContractProbe_compute_metadata(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "compute_metadata") {
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
			"name":     "projects/stackyard/locations/us-central1/compute_metadata/sample",
			"service":  "compute_metadata",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
