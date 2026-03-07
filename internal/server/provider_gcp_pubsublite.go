package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPPubSubLiteRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_pubsublite(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPPubSubLitePath(path) {
		return false
	}

	switch r.Method {
	case http.MethodPost:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPPubSubLitePath(path string) bool {
	return strings.HasPrefix(path, "/gcp/google.cloud.pubsublite.v1.AdminService/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.pubsublite.v1.CursorService/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.pubsublite.v1.PartitionAssignmentService/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.pubsublite.v1.PublisherService/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.pubsublite.v1.SubscriberService/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.pubsublite.v1.TopicStatsService/")
}

func handleGCPContractProbe_pubsublite(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "pubsublite") {
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
			"name":     "projects/stackyard/locations/us-central1/pubsublite/sample",
			"service":  "pubsublite",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
