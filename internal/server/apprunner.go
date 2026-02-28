package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type appRunnerError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleAppRunnerJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isAppRunnerJSONCandidate(r) {
		return false
	}

	action := parseAppRunnerTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondAppRunnerError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := appRunnerOperationByName[action]; !known {
		respondAppRunnerError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "apprunner")
	if !ok {
		respondAppRunnerError(w, status, code, msg)
		return true
	}

	payload, err := parseAppRunnerPayload(r)
	if err != nil {
		respondAppRunnerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.apprunner.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondAppRunnerJSON(w, http.StatusOK, response)
	return true
}

func isAppRunnerJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "AppRunner.") {
		return true
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "apprunner" {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(target, "AppRunner.") || target == ""
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".apprunner.") || strings.HasPrefix(host, "apprunner.")
}

func parseAppRunnerTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AppRunner.") {
		return strings.TrimPrefix(target, "AppRunner.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseAppRunnerPayload(r *http.Request) (map[string]any, error) {
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

func respondAppRunnerJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondAppRunnerError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondAppRunnerJSON(w, status, appRunnerError{Type: code, Message: msg})
}
