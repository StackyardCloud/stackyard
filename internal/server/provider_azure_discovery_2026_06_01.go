package server

import (
	"net/http"
	"strings"
)

const azureDiscovery20260601APIVersion = "2026-06-01"

// handleAzureDiscovery20260601Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/discovery-2026-06-01-plan.md.
func (s *Server) handleAzureDiscovery20260601Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.discovery/") {
		return false
	}
	return false
}
