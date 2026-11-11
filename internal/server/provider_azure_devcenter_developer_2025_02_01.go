package server

import (
	"net/http"
	"strings"
)

const azureDevCenterDeveloper20250201APIVersion = "2025-02-01"
const azureDevCenterDeveloper20250201Prefix = "/azure/devcenter-developer/2025-02-01"

// handleAzureDevCenterDeveloper20250201Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/devcenter-developer-2025-02-01-plan.md.
func (s *Server) handleAzureDevCenterDeveloper20250201Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureDevCenterDeveloper20250201Prefix) {
		return false
	}
	return false
}
