package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) handleGCPMediaTranslationRouter(w http.ResponseWriter, r *http.Request) bool {
	path := strings.TrimSpace(r.URL.Path)
	if path == "" {
		path = rawRequestPath(r)
	}
	if !isGCPMediaTranslationPath(path) {
		return false
	}

	if r.Method != http.MethodPost {
		return false
	}

	if path != "/gcp/google.cloud.mediatranslation.v1beta1.SpeechTranslationService/StreamingTranslateSpeech" {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}

	body, valid := decodeGCPMediaTranslationJSONBody(w, r, path)
	if !valid {
		return true
	}

	if len(body) == 0 {
		respondGCPMediaTranslationInvalidArgument(w, path, "streaming request payload is required")
		return true
	}

	hasConfig := false
	if _, ok := body["streamingConfig"].(map[string]any); ok {
		hasConfig = true
	}
	if _, ok := body["streaming_request"].(map[string]any); ok {
		hasConfig = true
	}
	if !hasConfig {
		respondGCPMediaTranslationInvalidArgument(w, path, "streamingConfig is required before audio content")
		return true
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"result": map[string]any{
			"textTranslationResult": map[string]any{
				"translation":         "hola stackyard",
				"targetLanguageCode":  "es-ES",
				"sourceLanguageCode":  "en-US",
				"translationFinished": true,
			},
			"isFinal": true,
		},
		"speechEventType": "END_OF_SINGLE_UTTERANCE",
	})
	return true
}

func isGCPMediaTranslationPath(path string) bool {
	return strings.HasPrefix(path, "/gcp/google.cloud.mediatranslation.v1beta1.SpeechTranslationService/")
}

func decodeGCPMediaTranslationJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	var body map[string]any
	if r.Body == nil {
		return map[string]any{}, true
	}

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPMediaTranslationInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func respondGCPMediaTranslationInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
