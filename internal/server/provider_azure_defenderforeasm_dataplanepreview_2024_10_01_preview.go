package server

import (
	"net/http"
	"strings"
)

const azureDefenderForEASMDataPlanePreview20241001APIVersion = "2024-10-01-preview"
const azureDefenderForEASMDataPlanePreview20241001Prefix = "/azure/defenderforeasm-dataplanepreview/2024-10-01-preview"

// handleAzureDefenderForEasmDataPlanePreview20241001Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/defenderforeasm-dataplanepreview-2024-10-01-preview-plan.md.
func (s *Server) handleAzureDefenderForEasmDataPlanePreview20241001Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureDefenderForEASMDataPlanePreview20241001Prefix) {
		return false
	}
	return false
}
