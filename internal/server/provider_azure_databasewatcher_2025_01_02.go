package server

import (
	"net/http"
	"strings"
)

const azureDatabaseWatcher20250102APIVersion = "2025-01-02"

// handleAzureDatabaseWatcher20250102Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/databasewatcher-2025-01-02-plan.md.
func (s *Server) handleAzureDatabaseWatcher20250102Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.databasewatcher/") {
		return false
	}
	return false
}
