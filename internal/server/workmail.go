package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type workmailError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleWorkMailJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isWorkMailJSONCandidate(r) {
		return false
	}

	action := parseWorkMailTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondWorkMailError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := workmailOperationByName[action]; !known {
		respondWorkMailError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "workmail")
	if !ok {
		respondWorkMailError(w, status, code, msg)
		return true
	}

	payload, err := parseWorkMailPayload(r)
	if err != nil {
		respondWorkMailError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.workmail.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondWorkMailJSON(w, http.StatusOK, response)
	return true
}

func isWorkMailJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "WorkMailService.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "WorkMailService.")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "workmail" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".workmail.") || strings.HasPrefix(host, "workmail.")
}

func parseWorkMailTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "WorkMailService.") {
		return strings.TrimPrefix(target, "WorkMailService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseWorkMailPayload(r *http.Request) (map[string]any, error) {
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

func respondWorkMailJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondWorkMailError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondWorkMailJSON(w, status, workmailError{Type: code, Message: msg})
}
