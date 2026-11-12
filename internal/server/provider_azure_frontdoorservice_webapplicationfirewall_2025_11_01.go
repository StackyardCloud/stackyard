package server

import (
	"net/http"
	"strings"
)

const azureFrontDoorServiceWebApplicationFirewall20251101APIVersion = "2025-11-01"

// handleAzureFrontDoorServiceWebApplicationFirewall20251101Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/frontdoorservice-webapplicationfirewall-2025-11-01-plan.md.
func (s *Server) handleAzureFrontDoorServiceWebApplicationFirewall20251101Router(w http.ResponseWriter, r *http.Request) bool {
	path := strings.ToLower(rawRequestPath(r))
	if !strings.Contains(path, "/providers/microsoft.network/frontdoors") &&
		!strings.Contains(path, "/providers/microsoft.network/checkfrontdoornameavailability") &&
		!strings.Contains(path, "/providers/microsoft.network/frontdoorwebapplicationfirewallpolicies") &&
		!strings.Contains(path, "/providers/microsoft.network/frontdoorwebapplicationfirewallmanagedrulesets") &&
		!strings.Contains(path, "/providers/microsoft.network/networkexperimentprofiles") {
		return false
	}

	// Stage-0 is intentionally non-owning to avoid changing existing Front Door routing behavior.
	return false
}
