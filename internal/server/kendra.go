package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type kendraError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleKendraJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isKendraJSONCandidate(r) {
		return false
	}

	action := parseKendraTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondKendraError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := kendraOperationByName[action]; !known {
		respondKendraError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "kendra")
	if !ok {
		respondKendraError(w, status, code, msg)
		return true
	}

	payload, err := parseKendraPayload(r)
	if err != nil {
		respondKendraError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.kendra.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondKendraJSON(w, http.StatusOK, response)
	return true
}

func isKendraJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AWSKendraFrontendService") || strings.Contains(target, "Kendra") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "Kendra")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "kendra" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".kendra.") || strings.HasPrefix(host, "kendra.")
}

func parseKendraTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSKendraFrontendService.") {
		return strings.TrimPrefix(target, "AWSKendraFrontendService.")
	}
	if strings.HasPrefix(target, "AmazonKendraFrontendService.") {
		return strings.TrimPrefix(target, "AmazonKendraFrontendService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseKendraPayload(r *http.Request) (map[string]any, error) {
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

func respondKendraJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondKendraError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondKendraJSON(w, status, kendraError{Type: code, Message: msg})
}
