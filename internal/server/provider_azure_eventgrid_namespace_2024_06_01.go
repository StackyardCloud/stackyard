package server

import (
	"net/http"
	"strings"
)

const azureEventGridNamespace20240601APIVersion = "2024-06-01"
const azureEventGridNamespace20240601Prefix = "/azure/eventgrid-namespace/2024-06-01"

// handleAzureEventGridNamespace20240601Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/eventgrid-namespace-2024-06-01-plan.md.
func (s *Server) handleAzureEventGridNamespace20240601Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureEventGridNamespace20240601Prefix) {
		return false
	}
	return false
}
