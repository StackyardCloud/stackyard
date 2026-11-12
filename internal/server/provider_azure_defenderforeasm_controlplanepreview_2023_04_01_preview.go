package server

import (
	"net/http"
	"strings"
)

const azureDefenderForEASMControlPlanePreview20230401APIVersion = "2023-04-01-preview"

// handleAzureDefenderForEasmControlPlanePreview20230401Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/defenderforeasm-controlplanepreview-2023-04-01-preview-plan.md.
func (s *Server) handleAzureDefenderForEasmControlPlanePreview20230401Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.easm/") {
		return false
	}
	return false
}
