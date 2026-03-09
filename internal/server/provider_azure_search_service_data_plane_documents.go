package server

import (
	"net/http"
	"strings"
)

const (
	azureSearchServiceIndexesPrefix = "/azure/indexes('"
	azureSearchServiceDocsMarker    = "')/docs"
)

func (s *Server) handleAzureSearchServiceDataPlaneDocumentsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureSearchServiceIndexesPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	markerIndex := strings.Index(path, azureSearchServiceDocsMarker)
	if markerIndex < 0 {
		return false
	}

	suffix := path[markerIndex+len(azureSearchServiceDocsMarker):]
	switch {
	case suffix == "":
		respondAzureImplemented(w, path)
		return true
	case strings.HasPrefix(suffix, "?"),
		strings.HasPrefix(suffix, "("),
		strings.HasPrefix(suffix, "/"),
		strings.HasPrefix(suffix, "$"),
		strings.HasPrefix(suffix, "."):
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}
