package server

import (
	"net/http"
	"strings"
)

const azureGraphServices20230413APIVersion = "2023-04-13"

// handleAzureGraphServices20230413Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/graphservices-2023-04-13-plan.md.
func (s *Server) handleAzureGraphServices20230413Router(w http.ResponseWriter, r *http.Request) bool {
	path := strings.ToLower(rawRequestPath(r))
	if !strings.Contains(path, "/providers/microsoft.graphservices/") {
		return false
	}

	// Stage-0 is intentionally non-owning.
	return false
}
