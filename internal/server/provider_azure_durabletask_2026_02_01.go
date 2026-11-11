package server

import (
	"net/http"
	"strings"
)

const azureDurableTask20260201APIVersion = "2026-02-01"

// handleAzureDurableTask20260201Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/durabletask-2026-02-01-plan.md.
func (s *Server) handleAzureDurableTask20260201Router(w http.ResponseWriter, r *http.Request) bool {
	path := strings.ToLower(rawRequestPath(r))
	if !strings.Contains(path, "/providers/microsoft.durabletask/") {
		return false
	}
	return false
}
