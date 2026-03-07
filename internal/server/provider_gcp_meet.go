package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPMeetRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_meet(w, r) {
		return true
	}

	path := rawRequestPath(r)

	if path == "/gcp/v2/spaces" {
		if r.Method == http.MethodPost {
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
		return false
	}

	if strings.HasPrefix(path, "/gcp/v2/spaces/") {
		switch r.Method {
		case http.MethodGet, http.MethodPatch:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		case http.MethodPost:
			if strings.HasSuffix(path, ":endActiveConference") {
				respondProviderNotImplemented(w, providerGCP, path)
				return true
			}
		}
		return false
	}

	if path == "/gcp/v2/conferenceRecords" {
		if r.Method == http.MethodGet {
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
		return false
	}

	if strings.HasPrefix(path, "/gcp/v2/conferenceRecords/") {
		switch r.Method {
		case http.MethodGet:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	return false
}

func handleGCPContractProbe_meet(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "meet") {
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
			"name":     "projects/stackyard/locations/us-central1/meet/sample",
			"service":  "meet",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	return false
}
