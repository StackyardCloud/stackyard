package server

import (
	"net/http"
	"strings"
)

const azureEventGridControlPlane20250215APIVersion = "2025-02-15"

// handleAzureEventGridControlPlane20250215Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/eventgrid-controlplane-2025-02-15-plan.md.
func (s *Server) handleAzureEventGridControlPlane20250215Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.eventgrid/") {
		return false
	}
	return false
}
