package server

import (
	"net/http"
	"strings"
)

const azureDynatrace20210901APIVersion = "2021-09-01"

// handleAzureDynatrace20210901Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/dynatrace-2021-09-01-plan.md.
func (s *Server) handleAzureDynatrace20210901Router(w http.ResponseWriter, r *http.Request) bool {
	path := strings.ToLower(rawRequestPath(r))
	if !strings.Contains(path, "/providers/dynatrace.observability/") {
		return false
	}
	return false
}
