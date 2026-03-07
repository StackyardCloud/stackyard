package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPManagedKafkaRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_managedkafka(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPManagedKafkaPath(path) {
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

func isGCPManagedKafkaPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.managedkafka.v1.ManagedKafka/") {
		return true
	}
	if strings.HasPrefix(path, "/gcp/google.cloud.managedkafka.v1.ManagedKafkaConnect/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if isGCPManagedKafkaLocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/clusters") ||
		strings.Contains(path, "/topics") ||
		strings.Contains(path, "/consumerGroups") ||
		strings.Contains(path, "/acls") ||
		strings.Contains(path, ":addAclEntry") ||
		strings.Contains(path, ":removeAclEntry") ||
		strings.Contains(path, "/connectClusters") ||
		strings.Contains(path, "/connectors") ||
		strings.Contains(path, ":pause") ||
		strings.Contains(path, ":resume") ||
		strings.Contains(path, ":restart") ||
		strings.Contains(path, ":stop") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":cancel")
}

func isGCPManagedKafkaLocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func handleGCPContractProbe_managedkafka(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "managedkafka") {
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
			"name":     "projects/stackyard/locations/us-central1/managedkafka/sample",
			"service":  "managedkafka",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
