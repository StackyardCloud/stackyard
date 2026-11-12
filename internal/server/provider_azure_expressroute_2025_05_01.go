package server

import (
	"net/http"
	"strings"
)

const azureExpressRoute20250501APIVersion = "2025-05-01"
const azureExpressRoute20250501Prefix = "/azure/expressroute/2025-05-01"

// handleAzureExpressRoute20250501Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/expressroute-2025-05-01-plan.md.
func (s *Server) handleAzureExpressRoute20250501Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureExpressRoute20250501Prefix) {
		return false
	}
	return false
}
