package server

import (
	"net/http"
	"strings"
)

const azureDeviceRegistry20260401APIVersion = "2026-04-01"

// handleAzureDeviceRegistry20260401Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/deviceregistry-2026-04-01-plan.md.
func (s *Server) handleAzureDeviceRegistry20260401Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.deviceregistry/") {
		return false
	}
	return false
}
