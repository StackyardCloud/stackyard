package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPGenerativeLanguageRouter(w http.ResponseWriter, r *http.Request) bool {
	path := normalizeGCPGenerativeLanguagePath(rawRequestPath(r))
	if !isGCPGenerativeLanguagePath(path) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPGenerativeLanguageListModels(w, r, path) {
			return true
		}
		if handleGCPGenerativeLanguageGetModel(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPGenerativeLanguageModelAction(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPGenerativeLanguagePath(path string) bool {
	return path == "/gcp/v1/models" || strings.HasPrefix(path, "/gcp/v1/models/")
}

func normalizeGCPGenerativeLanguagePath(path string) string {
	normalized := strings.ReplaceAll(path, "%3A", ":")
	normalized = strings.ReplaceAll(normalized, "%3a", ":")
	return normalized
}

func handleGCPGenerativeLanguageListModels(w http.ResponseWriter, r *http.Request, path string) bool {
	if path != "/gcp/v1/models" {
		return false
	}

	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageSize must be a non-negative integer",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	start := 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = parseOptionalNonNegativeInt(pageToken)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "pageToken must be a non-negative integer offset",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
	}

	models := gcpGenerativeLanguageModels()
	if start > len(models) {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageToken is out of range",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}

	end := len(models)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}

	nextPageToken := ""
	if end < len(models) {
		nextPageToken = strconv.Itoa(end)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"models":        models[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func handleGCPGenerativeLanguageGetModel(w http.ResponseWriter, path string) bool {
	modelName, action, ok := parseGCPGenerativeLanguageModelPath(path)
	if !ok || action != "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpGenerativeLanguageModel(modelName))
	return true
}

func handleGCPGenerativeLanguageModelAction(w http.ResponseWriter, r *http.Request, path string) bool {
	modelName, action, ok := parseGCPGenerativeLanguageModelPath(path)
	if !ok || action == "" {
		return false
	}

	var reqBody map[string]any
	if r.Body != nil {
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		if err := decoder.Decode(&reqBody); err != nil && err.Error() != "EOF" {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "request body must be valid JSON",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
	}

	switch action {
	case "generateContent":
		if !hasNonEmptyArrayField(reqBody, "contents") {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "contents must be a non-empty array",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
		respondJSON(w, http.StatusOK, gcpGenerativeLanguageGenerateContentResponse(modelName))
		return true
	case "streamGenerateContent":
		if !hasNonEmptyArrayField(reqBody, "contents") {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "contents must be a non-empty array",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
		respondJSON(w, http.StatusOK, []any{
			gcpGenerativeLanguageGenerateContentResponse(modelName),
		})
		return true
	case "countTokens":
		if !hasNonEmptyArrayField(reqBody, "contents") {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "contents must be a non-empty array",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"totalTokens": 12,
		})
		return true
	case "embedContent":
		if _, ok := reqBody["content"].(map[string]any); !ok {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "content must be provided",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"embedding": map[string]any{
				"values": []float64{0.12, 0.34, 0.56},
			},
		})
		return true
	case "batchEmbedContents":
		if !hasNonEmptyArrayField(reqBody, "requests") {
			respondJSON(w, http.StatusBadRequest, map[string]any{
				"error":    "InvalidArgument",
				"message":  "requests must be a non-empty array",
				"provider": providerGCP,
				"path":     path,
			})
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"embeddings": []any{
				map[string]any{"values": []float64{0.12, 0.34, 0.56}},
				map[string]any{"values": []float64{0.22, 0.44, 0.66}},
			},
		})
		return true
	default:
		return false
	}
}

func parseGCPGenerativeLanguageModelPath(path string) (modelName, action string, ok bool) {
	const modelPrefix = "/gcp/v1/models/"
	if !strings.HasPrefix(path, modelPrefix) {
		return "", "", false
	}

	remainder := strings.TrimSpace(strings.TrimPrefix(path, modelPrefix))
	if remainder == "" {
		return "", "", false
	}

	modelID := remainder
	if strings.Contains(modelID, ":") {
		var found bool
		modelID, action, found = strings.Cut(remainder, ":")
		if !found {
			return "", "", false
		}
	}

	modelID = strings.TrimSpace(modelID)
	action = strings.TrimSpace(action)
	if modelID == "" || strings.Contains(modelID, "/") {
		return "", "", false
	}

	if action != "" {
		switch action {
		case "generateContent", "streamGenerateContent", "countTokens", "embedContent", "batchEmbedContents":
		default:
			return "", "", false
		}
	}

	return "models/" + modelID, action, true
}

func hasNonEmptyArrayField(body map[string]any, field string) bool {
	items, ok := body[field].([]any)
	return ok && len(items) > 0
}

func gcpGenerativeLanguageModels() []map[string]any {
	return []map[string]any{
		gcpGenerativeLanguageModel("models/gemini-2.0-flash"),
		gcpGenerativeLanguageModel("models/gemini-1.5-pro"),
	}
}

func gcpGenerativeLanguageModel(name string) map[string]any {
	return map[string]any{
		"name":                       name,
		"displayName":                strings.TrimPrefix(name, "models/"),
		"description":                "Stackyard staged generative model fixture",
		"inputTokenLimit":            30720,
		"outputTokenLimit":           2048,
		"supportedGenerationMethods": []string{"generateContent", "countTokens", "embedContent"},
	}
}

func gcpGenerativeLanguageGenerateContentResponse(model string) map[string]any {
	return map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"role": "model",
					"parts": []any{
						map[string]any{
							"text": "Hello from Stackyard.",
						},
					},
				},
				"finishReason": "STOP",
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     6,
			"candidatesTokenCount": 4,
			"totalTokenCount":      10,
		},
		"modelVersion": strings.TrimPrefix(model, "models/"),
	}
}
