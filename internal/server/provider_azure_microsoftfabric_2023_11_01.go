package server

import (
	"net/http"
	"strings"
)

const azureMicrosoftFabric20231101APIVersion = "2023-11-01"
const azureMicrosoftFabric20231101Prefix = "/azure/microsoftfabric/2023-11-01"

// handleAzureMicrosoftFabric20231101Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/microsoftfabric-2023-11-01-plan.md.
func (s *Server) handleAzureMicrosoftFabric20231101Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureMicrosoftFabric20231101Prefix) {
		return false
	}
	return false
}
