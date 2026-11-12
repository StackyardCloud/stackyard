package server

import (
	"net/http"
	"strings"
)

const azureOpenAIPrefix = "/azure/openai/"

func (s *Server) handleAzureOpenAIRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, azureOpenAIPrefix) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}

	relative := strings.TrimPrefix(path, azureOpenAIPrefix)
	normalized := strings.Trim(strings.TrimSpace(relative), "/")
	if normalized == "" {
		return false
	}

	segments := splitPathSegments(normalized)
	if len(segments) == 0 {
		return false
	}

	switch segments[0] {
	case "batches":
		if handleAzureOpenAIBatchesRoutes(w, r, path, segments) {
			return true
		}
	case "files":
		if handleAzureOpenAIFilesRoutes(w, r, path, segments) {
			return true
		}
	case "fine_tuning":
		if handleAzureOpenAIFineTuningRoutes(w, r, path, segments) {
			return true
		}
	case "models":
		if handleAzureOpenAIModelsRoutes(w, r, path, segments) {
			return true
		}
	case "uploads":
		if handleAzureOpenAIUploadRoutes(w, r, path, segments) {
			return true
		}
	}

	// Keep staged ownership for the full OpenAI data-plane prefix.
	respondAzureImplemented(w, path)
	return true
}

func handleAzureOpenAIBatchesRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	switch {
	case len(segments) == 1 && (r.Method == http.MethodPost || r.Method == http.MethodGet):
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 2 && segments[1] != "" && r.Method == http.MethodGet:
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 3 && segments[1] != "" && segments[2] == "cancel" && r.Method == http.MethodPost:
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureOpenAIFilesRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	switch {
	case len(segments) == 1 && (r.Method == http.MethodPost || r.Method == http.MethodGet):
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 2 && segments[1] == "import" && r.Method == http.MethodPost:
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 2 && segments[1] != "" && (r.Method == http.MethodGet || r.Method == http.MethodDelete):
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 3 && segments[1] != "" && segments[2] == "content" && r.Method == http.MethodGet:
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureOpenAIFineTuningRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	if len(segments) < 2 || segments[1] != "jobs" {
		return false
	}

	switch {
	case len(segments) == 2 && (r.Method == http.MethodPost || r.Method == http.MethodGet):
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 3 && segments[2] != "" && (r.Method == http.MethodGet || r.Method == http.MethodDelete):
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 4 && segments[2] != "" && segments[3] == "cancel" && r.Method == http.MethodPost:
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 4 && segments[2] != "" && segments[3] == "checkpoints" && r.Method == http.MethodGet:
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 4 && segments[2] != "" && segments[3] == "events" && r.Method == http.MethodGet:
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureOpenAIModelsRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	switch {
	case len(segments) == 1 && r.Method == http.MethodGet:
		respondAzureImplemented(w, path)
		return true
	case len(segments) == 2 && segments[1] != "" && r.Method == http.MethodGet:
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}

func handleAzureOpenAIUploadRoutes(w http.ResponseWriter, r *http.Request, path string, segments []string) bool {
	if len(segments) != 3 || strings.TrimSpace(segments[1]) == "" {
		return false
	}
	if r.Method != http.MethodPost {
		return false
	}

	switch segments[2] {
	case "cancel", "complete", "parts":
		respondAzureImplemented(w, path)
		return true
	default:
		return false
	}
}
