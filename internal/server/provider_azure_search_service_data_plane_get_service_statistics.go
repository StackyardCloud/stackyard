package server

import (
	"net/http"
	"strings"
)

const azureSearchServiceGetServiceStatisticsPath = "/azure/servicestats"

func (s *Server) handleAzureSearchServiceDataPlaneGetServiceStatisticsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if path != azureSearchServiceGetServiceStatisticsPath && !strings.HasPrefix(path, azureSearchServiceGetServiceStatisticsPath+"/") {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	respondAzureImplemented(w, path)
	return true
}
