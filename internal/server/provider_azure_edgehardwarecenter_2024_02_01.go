package server

import (
	"net/http"
	"strings"
)

const azureEdgeHardwareCenter20240201APIVersion = "2024-02-01"

// handleAzureEdgeHardwareCenter20240201Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/edgehardwarecenter-2024-02-01-plan.md.
func (s *Server) handleAzureEdgeHardwareCenter20240201Router(w http.ResponseWriter, r *http.Request) bool {
	path := strings.ToLower(rawRequestPath(r))
	if !strings.Contains(path, "/providers/microsoft.edgeorder/") {
		return false
	}
	return false
}
