package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type snowballError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleSnowballJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSnowballJSONCandidate(r) {
		return false
	}

	action := parseSnowballTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondSnowballError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, ok := snowballOperationByName[action]; !ok {
		respondSnowballError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "snowball")
	if !ok {
		respondSnowballError(w, status, code, msg)
		return true
	}

	payload, err := parseSnowballPayload(r)
	if err != nil {
		respondSnowballError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.snowball.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondSnowballJSON(w, http.StatusOK, response)
	return true
}

func isSnowballJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AWSIESnowballJobManagementService.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "AWSIESnowballJobManagementService.")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "snowball" {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#snowball") || strings.Contains(userAgent, " snowball/") {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".snowball.") || strings.HasPrefix(host, "snowball.")
}

func parseSnowballTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSIESnowballJobManagementService.") {
		return strings.TrimPrefix(target, "AWSIESnowballJobManagementService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseSnowballPayload(r *http.Request) (map[string]any, error) {
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

func respondSnowballJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSnowballError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSnowballJSON(w, status, snowballError{Type: code, Message: msg})
}
