package server

import (
	"net/http"
	"strings"
)

const azureDataLakeAnalytics20161101APIVersion = "2016-11-01"

// handleAzureDataLakeAnalytics20161101Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/datalakeanalytics-2016-11-01-plan.md.
func (s *Server) handleAzureDataLakeAnalytics20161101Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.datalakeanalytics/") &&
		!strings.HasPrefix(path, "/azure/catalog/") &&
		!strings.HasPrefix(path, "/azure/Jobs") &&
		!strings.HasPrefix(path, "/azure/pipelines") &&
		!strings.HasPrefix(path, "/azure/recurrences") {
		return false
	}
	return false
}
