package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type healthLakeError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleHealthLakeJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isHealthLakeJSONCandidate(r) {
		return false
	}

	action := parseHealthLakeTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondHealthLakeError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := healthLakeOperationByName[action]; !known {
		respondHealthLakeError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "healthlake")
	if !ok {
		respondHealthLakeError(w, status, code, msg)
		return true
	}

	payload, err := parseHealthLakePayload(r)
	if err != nil {
		respondHealthLakeError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.healthlake.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondHealthLakeJSON(w, http.StatusOK, response)
	return true
}

func isHealthLakeJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "HealthLake.") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(target, "HealthLake.")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "healthlake" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".healthlake.") || strings.HasPrefix(host, "healthlake.")
}

func parseHealthLakeTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "HealthLake.") {
		return strings.TrimPrefix(target, "HealthLake.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseHealthLakePayload(r *http.Request) (map[string]any, error) {
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

func respondHealthLakeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondHealthLakeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondHealthLakeJSON(w, status, healthLakeError{Type: code, Message: msg})
}
