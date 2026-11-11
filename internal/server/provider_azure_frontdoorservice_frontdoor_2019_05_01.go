package server

import (
	"net/http"
	"strings"
)

const azureFrontDoorServiceFrontDoor20190501APIVersion = "2019-05-01"

// handleAzureFrontDoorServiceFrontDoor20190501Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/frontdoorservice-frontdoor-2019-05-01-plan.md.
func (s *Server) handleAzureFrontDoorServiceFrontDoor20190501Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	lower := strings.ToLower(path)
	if !strings.Contains(lower, "/providers/microsoft.network/frontdoors/") &&
		!strings.Contains(lower, "/providers/microsoft.network/checkfrontdoornameavailability") {
		return false
	}

	// Stage-0 is intentionally non-owning.
	return false
}
