package server

import (
	"net/http"
	"strings"
)

const azureDatabricks20260101APIVersion = "2026-01-01"

// handleAzureDatabricks20260101Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/databricks-2026-01-01-plan.md.
func (s *Server) handleAzureDatabricks20260101Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.databricks/") {
		return false
	}
	return false
}
