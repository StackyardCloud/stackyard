package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type mwaaServerlessError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleMWAAServerlessJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isMWAAServerlessJSONCandidate(r) {
		return false
	}

	action := parseMWAAServerlessTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondMWAAServerlessError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := mwaaServerlessOperationByName[action]; !known {
		respondMWAAServerlessError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "mwaa-serverless")
	if !ok {
		respondMWAAServerlessError(w, status, code, msg)
		return true
	}

	payload, err := parseMWAAServerlessPayload(r)
	if err != nil {
		respondMWAAServerlessError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.mwaaserverless.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondMWAAServerlessJSON(w, http.StatusOK, response)
	return true
}

func isMWAAServerlessJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AmazonMWAAServerless") || strings.Contains(target, "MWAAServerless") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "MWAA")
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "mwaa-serverless" || service == "mwaaserverless" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".mwaa-serverless.") || strings.HasPrefix(host, "mwaa-serverless.")
}

func parseMWAAServerlessTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AmazonMWAAServerless.") {
		return strings.TrimPrefix(target, "AmazonMWAAServerless.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseMWAAServerlessPayload(r *http.Request) (map[string]any, error) {
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

func respondMWAAServerlessJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondMWAAServerlessError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondMWAAServerlessJSON(w, status, mwaaServerlessError{Type: code, Message: msg})
}
