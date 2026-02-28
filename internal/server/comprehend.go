package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type comprehendError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleComprehendJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isComprehendJSONCandidate(r) {
		return false
	}

	action := parseComprehendTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondComprehendError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := comprehendOperationByName[action]; !known {
		respondComprehendError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "comprehend")
	if !ok {
		respondComprehendError(w, status, code, msg)
		return true
	}

	payload, err := parseComprehendPayload(r)
	if err != nil {
		respondComprehendError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.comprehend.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondComprehendJSON(w, http.StatusOK, response)
	return true
}

func isComprehendJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "Comprehend_20171127") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "Comprehend_20171127")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "comprehend" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".comprehend.") || strings.HasPrefix(host, "comprehend.")
}

func parseComprehendTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "Comprehend_20171127.") {
		return strings.TrimPrefix(target, "Comprehend_20171127.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseComprehendPayload(r *http.Request) (map[string]any, error) {
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

func respondComprehendJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondComprehendError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondComprehendJSON(w, status, comprehendError{Type: code, Message: msg})
}
