package server

import (
	"net/http"
	"strings"
)

const azureEventGridDataPlane20180101APIVersion = "2018-01-01"
const azureEventGridDataPlane20180101Prefix = "/azure/eventgrid-dataplane/2018-01-01"

// handleAzureEventGridDataPlane20180101Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/eventgrid-dataplane-2018-01-01-plan.md.
func (s *Server) handleAzureEventGridDataPlane20180101Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureEventGridDataPlane20180101Prefix) {
		return false
	}
	return false
}
