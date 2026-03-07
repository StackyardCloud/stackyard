package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPManagedKafkaSchemaRegistryRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_managedkafka_schemaregistry(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPManagedKafkaSchemaRegistryPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPManagedKafkaSchemaRegistryPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.managedkafka.schemaregistry.v1.ManagedSchemaRegistry/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if isGCPManagedKafkaSchemaRegistryLocationDiscoveryPath(path) {
		return true
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return strings.Contains(path, "/schemaRegistries") ||
		strings.Contains(path, "/contexts") ||
		strings.Contains(path, "/schemas") ||
		strings.Contains(path, "/subjects") ||
		strings.Contains(path, "/versions") ||
		strings.Contains(path, "/referencedby") ||
		strings.Contains(path, "/compatibility") ||
		strings.Contains(path, "/config") ||
		strings.Contains(path, "/mode") ||
		strings.Contains(path, "/operations") ||
		strings.Contains(path, ":cancel")
}

func isGCPManagedKafkaSchemaRegistryLocationDiscoveryPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return false
	}
	if parts[4] != "locations" {
		return false
	}
	return len(parts) == 5 || len(parts) == 6
}

func handleGCPContractProbe_managedkafka_schemaregistry(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "managedkafka_schemaregistry") {
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
			"name":     "projects/stackyard/locations/us-central1/managedkafka_schemaregistry/sample",
			"service":  "managedkafka_schemaregistry",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
