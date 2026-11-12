package server

import (
	"net/http"
	"strings"
)

const azureEventHub20260101APIVersion = "2026-01-01"
const azureEventHub20260101Prefix = "/azure/eventhub/2026-01-01"

// handleAzureEventHub20260101Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/eventhub-2026-01-01-plan.md.
func (s *Server) handleAzureEventHub20260101Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureEventHub20260101Prefix) {
		return false
	}
	return false
}
