package server

import (
	"net/http"
	"strings"
)

const azureDataReplication20260501APIVersion = "2026-05-01"

// handleAzureDataReplication20260501Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/datareplication-2026-05-01-plan.md.
func (s *Server) handleAzureDataReplication20260501Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.datareplication/") {
		return false
	}
	return false
}
