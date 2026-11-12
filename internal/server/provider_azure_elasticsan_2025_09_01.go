package server

import (
	"net/http"
	"strings"
)

const azureElasticSan20250901APIVersion = "2025-09-01"

// handleAzureElasticSan20250901Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/elasticsan-2025-09-01-plan.md.
func (s *Server) handleAzureElasticSan20250901Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.elasticsan/") {
		return false
	}
	return false
}
