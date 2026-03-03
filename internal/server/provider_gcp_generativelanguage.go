package server

import (
	"net/http"
	"strings"
)

func (s *Server) handleGCPGenerativeLanguageRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if path == "/gcp/v1/models" && r.Method == http.MethodGet {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
	if strings.HasPrefix(path, "/gcp/v1/models/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
	}
	return false
}
