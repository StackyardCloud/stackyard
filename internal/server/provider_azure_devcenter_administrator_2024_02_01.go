package server

import (
	"net/http"
	"strings"
)

const azureDevCenterAdministrator20240201APIVersion = "2024-02-01"

// handleAzureDevCenterAdministrator20240201Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/devcenter-administrator-2024-02-01-plan.md.
func (s *Server) handleAzureDevCenterAdministrator20240201Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.devcenter/") {
		return false
	}
	return false
}
