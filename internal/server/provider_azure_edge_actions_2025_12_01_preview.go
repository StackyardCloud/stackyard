package server

import (
	"net/http"
	"strings"
)

const azureEdgeActions20251201PreviewAPIVersion = "2025-12-01-preview"

// handleAzureEdgeActions20251201PreviewRouter is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/edge-actions-2025-12-01-preview-plan.md.
func (s *Server) handleAzureEdgeActions20251201PreviewRouter(w http.ResponseWriter, r *http.Request) bool {
	path := strings.ToLower(rawRequestPath(r))
	if !strings.Contains(path, "/providers/microsoft.cdn/edgeactions") {
		return false
	}

	// Stage-0 is intentionally non-owning to avoid changing existing CDN route behavior.
	return false
}
