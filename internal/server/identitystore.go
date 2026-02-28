package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type identityStoreError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleIdentityStoreJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isIdentityStoreJSONCandidate(r) {
		return false
	}

	action := parseIdentityStoreTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondIdentityStoreError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := identityStoreOperationByName[action]; !known {
		respondIdentityStoreError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "identitystore")
	if !ok {
		respondIdentityStoreError(w, status, code, msg)
		return true
	}

	payload, err := parseIdentityStorePayload(r)
	if err != nil {
		respondIdentityStoreError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.identitystore.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondIdentityStoreJSON(w, http.StatusOK, response)
	return true
}

func isIdentityStoreJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "AWSIdentityStore.") {
		return true
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "identitystore" {
		return false
	}
	if service == "identitystore" {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") && strings.Contains(target, "AWSIdentityStore") {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".identitystore.") || strings.HasPrefix(host, "identitystore.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(userAgent, "command#identitystore") || strings.Contains(userAgent, " identitystore/")
}

func parseIdentityStoreTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSIdentityStore.") {
		return strings.TrimPrefix(target, "AWSIdentityStore.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseIdentityStorePayload(r *http.Request) (map[string]any, error) {
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

func respondIdentityStoreJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondIdentityStoreError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondIdentityStoreJSON(w, status, identityStoreError{Type: code, Message: msg})
}
