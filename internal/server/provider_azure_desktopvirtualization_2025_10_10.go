package server

import (
	"net/http"
	"strings"
)

const azureDesktopVirtualization20251010APIVersion = "2025-10-10"

// handleAzureDesktopVirtualization20251010Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/desktopvirtualization-2025-10-10-plan.md.
func (s *Server) handleAzureDesktopVirtualization20251010Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.desktopvirtualization/") {
		return false
	}
	return false
}
