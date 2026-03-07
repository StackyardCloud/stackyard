package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPWorkspaceEventsSubscriptionsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_workspaceevents_subscriptions(w, r) {
		return true
	}

	path := rawRequestPath(r)

	// Route recognition for Google Workspace Events Subscriptions v1 resources:
	// - /v1/subscriptions
	// - /v1/subscriptions/{subscription}
	// - /v1/subscriptions/{subscription}:reactivate
	if path == "/gcp/v1/subscriptions" {
		switch r.Method {
		case http.MethodGet, http.MethodPost:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	if !strings.HasPrefix(path, "/gcp/v1/subscriptions/") {
		return false
	}

	switch r.Method {
	case http.MethodGet, http.MethodPatch, http.MethodDelete:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if strings.HasSuffix(path, ":reactivate") {
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
	}

	return false
}

func handleGCPContractProbe_workspaceevents_subscriptions(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "workspaceevents_subscriptions") {
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
			"name":     "projects/stackyard/locations/us-central1/workspaceevents_subscriptions/sample",
			"service":  "workspaceevents_subscriptions",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
