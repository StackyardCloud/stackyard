package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type rekognitionError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleRekognitionJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isRekognitionJSONCandidate(r) {
		return false
	}

	action := parseRekognitionTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondRekognitionError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := rekognitionOperationByName[action]; !known {
		respondRekognitionError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "rekognition")
	if !ok {
		respondRekognitionError(w, status, code, msg)
		return true
	}

	payload, err := parseRekognitionPayload(r)
	if err != nil {
		respondRekognitionError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.rekognition.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondRekognitionJSON(w, http.StatusOK, response)
	return true
}

func isRekognitionJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "RekognitionService.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "RekognitionService.")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "rekognition" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".rekognition.") || strings.HasPrefix(host, "rekognition.")
}

func parseRekognitionTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "RekognitionService.") {
		return strings.TrimPrefix(target, "RekognitionService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseRekognitionPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}

	var payload map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&payload); err != nil {
		return nil, err
	}
	if payload == nil {
		payload = map[string]any{}
	}
	return payload, nil
}

func respondRekognitionJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondRekognitionError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondRekognitionJSON(w, status, rekognitionError{Type: code, Message: msg})
}
