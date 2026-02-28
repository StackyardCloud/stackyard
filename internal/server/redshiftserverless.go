package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type redshiftServerlessError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleRedshiftServerlessJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isRedshiftServerlessJSONCandidate(r) {
		return false
	}

	action := parseRedshiftServerlessTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondRedshiftServerlessError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := redshiftServerlessOperationByName[action]; !known {
		respondRedshiftServerlessError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "redshift-serverless")
	if !ok {
		okAlt, _, _, _, _ := s.validateSigV4WithService(r, "redshiftserverless")
		if okAlt {
			ok = true
		}
	}
	if !ok {
		okAlt, _, _, _, _ := s.validateSigV4WithService(r, "redshift")
		if okAlt {
			ok = true
		}
	}
	if !ok {
		respondRedshiftServerlessError(w, status, code, msg)
		return true
	}

	payload, err := parseRedshiftServerlessPayload(r)
	if err != nil {
		respondRedshiftServerlessError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.redshiftserverless.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondRedshiftServerlessJSON(w, http.StatusOK, response)
	return true
}

func isRedshiftServerlessJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "RedshiftServerless.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "RedshiftServerless.")
	}

	hint := strings.TrimSpace(sigV4ServiceHint(r))
	if hint == "redshift-serverless" || hint == "redshiftserverless" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".redshift-serverless.") ||
		strings.HasPrefix(host, "redshift-serverless.") ||
		strings.Contains(host, "redshift-serverless")
}

func parseRedshiftServerlessTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "RedshiftServerless.") {
		return strings.TrimPrefix(target, "RedshiftServerless.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseRedshiftServerlessPayload(r *http.Request) (map[string]any, error) {
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

func respondRedshiftServerlessJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondRedshiftServerlessError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondRedshiftServerlessJSON(w, status, redshiftServerlessError{Type: code, Message: msg})
}
