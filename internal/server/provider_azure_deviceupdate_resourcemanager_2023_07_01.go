package server

import (
	"net/http"
	"strings"
)

const azureDeviceUpdateResourceManager20230701APIVersion = "2023-07-01"

// handleAzureDeviceUpdateResourceManager20230701Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/deviceupdate-resourcemanager-2023-07-01-plan.md.
func (s *Server) handleAzureDeviceUpdateResourceManager20230701Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.deviceupdate/") {
		return false
	}
	return false
}
