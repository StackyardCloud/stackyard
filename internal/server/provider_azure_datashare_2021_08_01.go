package server

import (
	"net/http"
	"strings"
)

const azureDataShare20210801APIVersion = "2021-08-01"

// handleAzureDataShare20210801Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/datashare-2021-08-01-plan.md.
func (s *Server) handleAzureDataShare20210801Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.datashare/") {
		return false
	}
	return false
}
