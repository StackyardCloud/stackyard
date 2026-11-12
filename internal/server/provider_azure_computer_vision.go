package server

import (
	"net/http"
	"strings"
)

const azureComputerVisionPrefix = "/azure/computervision/v4.0-preview/2023-04-01/"

func (s *Server) handleAzureComputerVisionRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureComputerVisionPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureComputerVisionPrefix)
	normalized := strings.Trim(strings.TrimSpace(relative), "/")
	if normalized == "" {
		return false
	}

	segments := splitPathSegments(normalized)
	if len(segments) == 0 {
		return false
	}
	first := strings.ToLower(strings.TrimSpace(segments[0]))
	switch {
	case first == "datasets":
		if handleAzureComputerVisionDatasetsRoutes(w, r, path, segments) {
			return true
		}
	case strings.HasPrefix(first, "imageanalysis:"):
		if handleAzureComputerVisionImageAnalysisRoutes(w, r, path, first) {
			return true
		}
	case strings.HasPrefix(first, "imagecomposition:"):
		if handleAzureComputerVisionImageCompositionRoutes(w, r, path, first) {
			return true
		}
	case strings.HasPrefix(first, "imageretrieval:"):
		if handleAzureComputerVisionImageRetrievalRoutes(w, r, path, first) {
			return true
		}
	case first == "modelevaluations":
		if handleAzureComputerVisionModelEvaluationRoutes(w, r, path, segments) {
			return true
		}
	case first == "models":
		if handleAzureComputerVisionModelRoutes(w, r, path, segments) {
			return true
		}
	case strings.HasPrefix(first, "planogramcompliance:"):
		if first == "planogramcompliance:match" && r.Method == http.MethodPost {
			respondAzureImplemented(w, path)
			return true
		}
	case first == "productrecognition":
		if handleAzureComputerVisionProductRecognitionRoutes(w, r, path, segments) {
			return true
		}
	}

	// Keep staged ownership for the full computer vision prefix.
	respondAzureImplemented(w, path)
	return true
}

func handleAzureComputerVisionDatasetsRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	switch {
	case len(segments) == 1 && r.Method == http.MethodGet:
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 2 && segments[1] != "" && (r.Method == http.MethodPut || r.Method == http.MethodGet || r.Method == http.MethodPatch || r.Method == http.MethodDelete):
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureComputerVisionImageAnalysisRoutes(w http.ResponseWriter, r *http.Request, path, first string) bool {
	if r.Method != http.MethodPost {
		return false
	}
	switch first {
	case "imageanalysis:analyze", "imageanalysis:segment":
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureComputerVisionImageCompositionRoutes(w http.ResponseWriter, r *http.Request, path, first string) bool {
	if r.Method != http.MethodPost {
		return false
	}
	switch first {
	case "imagecomposition:rectify", "imagecomposition:stitch":
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureComputerVisionImageRetrievalRoutes(w http.ResponseWriter, r *http.Request, path, first string) bool {
	if r.Method != http.MethodPost {
		return false
	}
	switch first {
	case "imageretrieval:vectorizeimage", "imageretrieval:vectorizestream", "imageretrieval:vectorizetext":
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureComputerVisionModelEvaluationRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	switch {
	case len(segments) == 1 && r.Method == http.MethodGet:
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 2 && segments[1] != "" && (r.Method == http.MethodPut || r.Method == http.MethodGet || r.Method == http.MethodDelete):
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureComputerVisionModelRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	switch {
	case len(segments) == 1 && r.Method == http.MethodGet:
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 2 && segments[1] != "" && (r.Method == http.MethodPut || r.Method == http.MethodGet || r.Method == http.MethodDelete):
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 2 && strings.HasSuffix(strings.ToLower(segments[1]), ":cancel") && r.Method == http.MethodPost:
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureComputerVisionProductRecognitionRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	switch {
	case len(segments) == 2 && strings.EqualFold(segments[1], "runs") && (r.Method == http.MethodPost || r.Method == http.MethodGet):
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 3 && strings.EqualFold(segments[1], "runs") && segments[2] != "" && (r.Method == http.MethodGet || r.Method == http.MethodDelete):
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}
