package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type protonError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleProtonJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isProtonJSONCandidate(r) {
		return false
	}

	action := parseProtonTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondProtonError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := protonOperationByName[action]; !known {
		respondProtonError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "proton")
	if !ok {
		respondProtonError(w, status, code, msg)
		return true
	}

	payload, err := parseProtonPayload(r)
	if err != nil {
		respondProtonError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.proton.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondProtonJSON(w, http.StatusOK, response)
	return true
}

func isProtonJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AwsProton20200720") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.Contains(target, "AwsProton20200720")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "proton" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".proton.") || strings.HasPrefix(host, "proton.")
}

func parseProtonTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AwsProton20200720.") {
		return strings.TrimPrefix(target, "AwsProton20200720.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseProtonPayload(r *http.Request) (map[string]any, error) {
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

func respondProtonJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondProtonError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondProtonJSON(w, status, protonError{Type: code, Message: msg})
}
