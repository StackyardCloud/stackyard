package server

import (
	"net/http"
	"strings"
)

const azureDatadog20250611APIVersion = "2025-06-11"

// handleAzureDatadog20250611Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/datadog-2025-06-11-plan.md.
func (s *Server) handleAzureDatadog20250611Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.datadog/") {
		return false
	}
	return false
}
