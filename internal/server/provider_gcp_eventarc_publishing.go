package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPEventarcPublishingRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_eventarc_publishing(w, r) {
		return true
	}

	path := rawRequestPath(r)
	if !isGCPEventarcPublishingPath(path) {
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

func isGCPEventarcPublishingPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.cloud.eventarc.publishing.v1.Publisher/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/projects/") {
		return false
	}
	if !strings.Contains(path, "/locations/") {
		return false
	}

	return (strings.Contains(path, "/channelConnections/") && strings.Contains(path, ":publishEvents")) ||
		(strings.Contains(path, "/channels/") && strings.Contains(path, ":publishEvents")) ||
		(strings.Contains(path, "/messageBuses/") && strings.Contains(path, ":publish"))
}

func handleGCPContractProbe_eventarc_publishing(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "eventarc_publishing") {
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
			"name":     "projects/stackyard/locations/us-central1/eventarc_publishing/sample",
			"service":  "eventarc_publishing",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
