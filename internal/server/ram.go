package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type ramError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleRAMJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isRAMJSONCandidate(r) {
		return false
	}

	action := parseRAMTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondRAMError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := ramOperationByName[action]; !known {
		respondRAMError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "ram")
	if !ok {
		respondRAMError(w, status, code, msg)
		return true
	}

	payload, err := parseRAMPayload(r)
	if err != nil {
		respondRAMError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.ram.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondRAMJSON(w, http.StatusOK, response)
	return true
}

func isRAMJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AWSRamFrontEndService") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "AWSRamFrontEndService")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "ram" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".ram.") || strings.HasPrefix(host, "ram.")
}

func parseRAMTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSRamFrontEndService.") {
		return strings.TrimPrefix(target, "AWSRamFrontEndService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseRAMPayload(r *http.Request) (map[string]any, error) {
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

func respondRAMJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondRAMError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondRAMJSON(w, status, ramError{Type: code, Message: msg})
}
