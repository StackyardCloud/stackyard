package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPPubSubRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_pubsub(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPPubSubPath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPPubSubPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.pubsub.v1.Publisher/") ||
		strings.HasPrefix(path, "/gcp/google.pubsub.v1.Subscriber/") ||
		strings.HasPrefix(path, "/gcp/google.pubsub.v1.SchemaService/") {
		return true
	}

	if !strings.HasPrefix(path, "/gcp/v1/") || !strings.Contains(path, "/projects/") {
		return false
	}

	return strings.Contains(path, "/topics") ||
		strings.Contains(path, "/subscriptions") ||
		strings.Contains(path, "/snapshots") ||
		strings.Contains(path, "/schemas") ||
		strings.Contains(path, ":publish") ||
		strings.Contains(path, ":pull") ||
		strings.Contains(path, ":acknowledge") ||
		strings.Contains(path, ":modifyAckDeadline") ||
		strings.Contains(path, ":modifyPushConfig") ||
		strings.Contains(path, ":seek") ||
		strings.Contains(path, ":detach") ||
		strings.Contains(path, ":validate") ||
		strings.Contains(path, ":validateMessage") ||
		strings.Contains(path, ":commit") ||
		strings.Contains(path, ":rollback") ||
		strings.Contains(path, ":deleteRevision")
}

func handleGCPContractProbe_pubsub(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "pubsub") {
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
			"name":     "projects/stackyard/locations/us-central1/pubsub/sample",
			"service":  "pubsub",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
