package server

import (
	"net/http"
	"strings"
)

const azureDeviceUpdateDataPlane20221001APIVersion = "2022-10-01"
const azureDeviceUpdateDataPlane20221001Prefix = "/azure/deviceupdate-dataplane/2022-10-01/"

// handleAzureDeviceUpdateDataPlane20221001Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/deviceupdate-dataplane-2022-10-01-plan.md.
func (s *Server) handleAzureDeviceUpdateDataPlane20221001Router(w http.ResponseWriter, r *http.Request) bool {
	path := strings.ToLower(rawRequestPath(r))
	if !strings.HasPrefix(path, azureDeviceUpdateDataPlane20221001Prefix) {
		return false
	}
	return false
}
