package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type awsHealthError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleAWSHealthJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isAWSHealthJSONCandidate(r) {
		return false
	}

	action := parseAWSHealthTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondAWSHealthError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := healthOperationByName[action]; !known {
		respondAWSHealthError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "health")
	if !ok {
		respondAWSHealthError(w, status, code, msg)
		return true
	}

	payload, err := parseAWSHealthPayload(r)
	if err != nil {
		respondAWSHealthError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.health.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondAWSHealthJSON(w, http.StatusOK, response)
	return true
}

func isAWSHealthJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AWSHealth_20160804") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "AWSHealth_20160804")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "health" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".health.") || strings.HasPrefix(host, "health.")
}

func parseAWSHealthTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSHealth_20160804.") {
		return strings.TrimPrefix(target, "AWSHealth_20160804.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseAWSHealthPayload(r *http.Request) (map[string]any, error) {
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

func respondAWSHealthJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondAWSHealthError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondAWSHealthJSON(w, status, awsHealthError{Type: code, Message: msg})
}
