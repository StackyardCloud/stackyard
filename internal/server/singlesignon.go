package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type singleSignOnError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleSingleSignOnJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSingleSignOnJSONCandidate(r) {
		return false
	}

	action := parseSingleSignOnTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondSingleSignOnError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := singleSignOnOperationByName[action]; !known {
		respondSingleSignOnError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "sso")
	if !ok {
		respondSingleSignOnError(w, status, code, msg)
		return true
	}

	payload, err := parseSingleSignOnPayload(r)
	if err != nil {
		respondSingleSignOnError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.singlesignon.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondSingleSignOnJSON(w, http.StatusOK, response)
	return true
}

func isSingleSignOnJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "SWBExternalService") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "SWBExternalService")
	}
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "sso" || service == "sso-admin" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".sso.") || strings.HasPrefix(host, "sso.")
}

func parseSingleSignOnTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "SWBExternalService.") {
		return strings.TrimPrefix(target, "SWBExternalService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseSingleSignOnPayload(r *http.Request) (map[string]any, error) {
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

func respondSingleSignOnJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSingleSignOnError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSingleSignOnJSON(w, status, singleSignOnError{Type: code, Message: msg})
}
