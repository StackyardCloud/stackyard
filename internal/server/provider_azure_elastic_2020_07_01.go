package server

import (
	"net/http"
	"strings"
)

const azureElastic20200701APIVersion = "2020-07-01"

// handleAzureElastic20200701Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/elastic-2020-07-01-plan.md.
func (s *Server) handleAzureElastic20200701Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.elastic/") {
		return false
	}
	return false
}
