package server

import (
	"net/http"
	"strings"
)

const azureDataMigration20220330PreviewAPIVersion = "2022-03-30-preview"

// handleAzureDataMigration20220330PreviewRouter is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/datamigration-2022-03-30-preview-plan.md.
func (s *Server) handleAzureDataMigration20220330PreviewRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.datamigration/") {
		return false
	}
	return false
}
