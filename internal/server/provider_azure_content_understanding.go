package server

import (
	"net/http"
	"strings"
)

const azureContentUnderstandingPrefix = "/azure/contentunderstanding/"

func (s *Server) handleAzureContentUnderstandingRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureContentUnderstandingPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureContentUnderstandingPrefix)
	normalized := strings.Trim(strings.TrimSpace(relative), "/")
	if normalized == "" {
		return false
	}

	segments := splitPathSegments(normalized)
	if len(segments) == 0 {
		return false
	}

	if handleAzureContentUnderstandingAnalyzerRoutes(w, r, path, segments) {
		return true
	}
	if handleAzureContentUnderstandingResultRoutes(w, r, path, segments) {
		return true
	}
	if handleAzureContentUnderstandingDefaultsRoutes(w, r, path, segments) {
		return true
	}

	// Keep staged ownership for the full content understanding prefix.
	respondAzureImplemented(w, path)
	return true
}

func handleAzureContentUnderstandingAnalyzerRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	if !strings.EqualFold(segments[0], "analyzers") {
		return false
	}

	switch {
	case len(segments) == 1 && r.Method == http.MethodGet:
		// List analyzers
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 2 && segments[1] != "":
		analyzerSegment := strings.TrimSpace(segments[1])
		lower := strings.ToLower(analyzerSegment)
		if strings.Contains(lower, ":") {
			switch {
			case strings.HasSuffix(lower, ":analyze") && r.Method == http.MethodPost:
				respondAzureImplemented(w, path)
				return true
			case strings.HasSuffix(lower, ":analyzebinary") && r.Method == http.MethodPost:
				respondAzureImplemented(w, path)
				return true
			case strings.HasSuffix(lower, ":copy") && r.Method == http.MethodPost:
				respondAzureImplemented(w, path)
				return true
			case strings.HasSuffix(lower, ":grantcopyauthorization") && r.Method == http.MethodPost:
				respondAzureImplemented(w, path)
				return true
			default:
				return false
			}
		}

		switch r.Method {
		case http.MethodPut, http.MethodPatch, http.MethodGet, http.MethodDelete:
			respondAzureImplemented(w, path)
			return true
		default:
			return false
		}
	case len(segments) == 4 &&
		segments[1] != "" &&
		strings.EqualFold(segments[2], "operations") &&
		segments[3] != "" &&
		r.Method == http.MethodGet:
		// Get analyzer operation status
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureContentUnderstandingResultRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	if !strings.EqualFold(segments[0], "analyzerResults") {
		return false
	}

	switch {
	case len(segments) == 2 && segments[1] != "" && (r.Method == http.MethodGet || r.Method == http.MethodDelete):
		// Get/Delete analyzer result
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 4 &&
		segments[1] != "" &&
		strings.EqualFold(segments[2], "files") &&
		segments[3] != "" &&
		r.Method == http.MethodGet:
		// Get analyzer result file
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureContentUnderstandingDefaultsRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	if len(segments) != 1 || !strings.EqualFold(segments[0], "defaults") {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPatch {
		return false
	}
	respondAzureImplemented(w, path)
	return true
}
