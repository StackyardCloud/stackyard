package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type shieldAdvancedError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleShieldAdvancedJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isShieldAdvancedJSONCandidate(r) {
		return false
	}

	action := parseShieldAdvancedTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondShieldAdvancedError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := shieldAdvancedOperationByName[action]; !known {
		respondShieldAdvancedError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "shield")
	if !ok {
		respondShieldAdvancedError(w, status, code, msg)
		return true
	}

	payload, err := parseShieldAdvancedPayload(r)
	if err != nil {
		respondShieldAdvancedError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.shieldadvanced.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondShieldAdvancedJSON(w, http.StatusOK, response)
	return true
}

func isShieldAdvancedJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AWSShield_20160616.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") && strings.Contains(target, "AWSShield_20160616.") {
		return true
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "shield" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".shield.") || strings.HasPrefix(host, "shield.")
}

func parseShieldAdvancedTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSShield_20160616.") {
		return strings.TrimPrefix(target, "AWSShield_20160616.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseShieldAdvancedPayload(r *http.Request) (map[string]any, error) {
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

func respondShieldAdvancedJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondShieldAdvancedError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondShieldAdvancedJSON(w, status, shieldAdvancedError{Type: code, Message: msg})
}
