package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type supportError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleSupportJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSupportJSONCandidate(r) {
		return false
	}

	action := parseSupportTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondSupportError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := supportOperationByName[action]; !known {
		respondSupportError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "support")
	if !ok {
		respondSupportError(w, status, code, msg)
		return true
	}

	payload, err := parseSupportPayload(r)
	if err != nil {
		respondSupportError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.support.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondSupportJSON(w, http.StatusOK, response)
	return true
}

func isSupportJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AWSSupport_20130415") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "AWSSupport_20130415")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "support" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".support.") ||
		strings.HasPrefix(host, "support.") ||
		strings.Contains(host, ".awssupport.") ||
		strings.HasPrefix(host, "awssupport.")
}

func parseSupportTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSSupport_20130415.") {
		return strings.TrimPrefix(target, "AWSSupport_20130415.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseSupportPayload(r *http.Request) (map[string]any, error) {
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

func respondSupportJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSupportError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSupportJSON(w, status, supportError{Type: code, Message: msg})
}
