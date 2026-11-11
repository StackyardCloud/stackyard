package server

import (
	"net/http"
	"strings"
)

const azureDellStorage20250321APIVersion = "2025-03-21"

// handleAzureDellStorage20250321Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/dellstorage-2025-03-21-plan.md.
func (s *Server) handleAzureDellStorage20250321Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/dell.storage/") {
		return false
	}
	return false
}
