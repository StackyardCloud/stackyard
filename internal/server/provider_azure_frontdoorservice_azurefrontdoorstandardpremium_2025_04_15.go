package server

import (
	"net/http"
	"strings"
)

const azureFrontDoorServiceAzureFrontDoorStandardPremium20250415APIVersion = "2025-04-15"

// handleAzureFrontDoorServiceAzureFrontDoorStandardPremium20250415Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/frontdoorservice-azurefrontdoorstandardpremium-2025-04-15-plan.md.
func (s *Server) handleAzureFrontDoorServiceAzureFrontDoorStandardPremium20250415Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.cdn/") {
		return false
	}

	// Stage-0 is intentionally non-owning to avoid changing existing CDN routing behavior.
	return false
}
