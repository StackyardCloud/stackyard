package server

import (
	"net/http"
	"strings"
)

const azureDataLakeStore20161101APIVersion = "2016-11-01"

// handleAzureDataLakeStore20161101Router is intentionally a stage-0 skeleton.
// The staged implementation plan lives in docs/datalakestore-2016-11-01-plan.md.
func (s *Server) handleAzureDataLakeStore20161101Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.Contains(strings.ToLower(path), "/providers/microsoft.datalakestore/") {
		return false
	}
	return false
}
