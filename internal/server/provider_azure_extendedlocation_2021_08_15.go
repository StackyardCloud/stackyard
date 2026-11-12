package server

import (
	"net/http"
	"strings"
)

const azureExtendedLocation20210815APIVersion = "2021-08-15"
const azureExtendedLocation20210815Prefix = "/azure/extendedlocation/2021-08-15"

// handleAzureExtendedLocation20210815Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/extendedlocation-2021-08-15-plan.md.
func (s *Server) handleAzureExtendedLocation20210815Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureExtendedLocation20210815Prefix) {
		return false
	}
	return false
}
