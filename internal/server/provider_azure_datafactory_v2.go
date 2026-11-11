package server

import (
	"net/http"
	"strings"
)

const azureDataFactoryV2APIVersion = "2018-06-01"

// handleAzureDataFactoryV2Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/datafactory-v2-plan.md.
func (s *Server) handleAzureDataFactoryV2Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/Microsoft.DataFactory/") &&
		!strings.Contains(strings.ToLower(path), "/providers/microsoft.datafactory/") {
		return false
	}
	return false
}
