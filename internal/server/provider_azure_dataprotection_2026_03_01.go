package server

import (
	"net/http"
	"strings"
)

const azureDataProtection20260301APIVersion = "2026-03-01"

// handleAzureDataProtection20260301Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/dataprotection-2026-03-01-plan.md.
func (s *Server) handleAzureDataProtection20260301Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.dataprotection/") {
		return false
	}
	return false
}
