package server

import (
	"net/http"
	"strings"
)

const azureGuestConfiguration20240405APIVersion = "2024-04-05"

// handleAzureGuestConfiguration20240405Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/guestconfiguration-2024-04-05-plan.md.
func (s *Server) handleAzureGuestConfiguration20240405Router(w http.ResponseWriter, r *http.Request) bool {
	path := strings.ToLower(rawRequestPath(r))
	if !strings.Contains(path, "/providers/microsoft.guestconfiguration/") {
		return false
	}

	// Stage-0 is intentionally non-owning.
	return false
}
