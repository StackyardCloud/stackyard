package server

import (
	"net/http"
	"strings"
)

const azureEducation20211201PreviewAPIVersion = "2021-12-01-preview"

// handleAzureEducation20211201PreviewRouter is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/education-2021-12-01-preview-plan.md.
func (s *Server) handleAzureEducation20211201PreviewRouter(w http.ResponseWriter, r *http.Request) bool {
	path := strings.ToLower(rawRequestPath(r))
	if !strings.Contains(path, "/providers/microsoft.education/") {
		return false
	}
	return false
}
