package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type pcsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handlePCSJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isPCSJSONCandidate(r) {
		return false
	}

	action := parsePCSTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondPCSError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := pcsOperationByName[action]; !known {
		respondPCSError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "pcs")
	if !ok {
		respondPCSError(w, status, code, msg)
		return true
	}

	payload, err := parsePCSPayload(r)
	if err != nil {
		respondPCSError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.pcs.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondPCSJSON(w, http.StatusOK, response)
	return true
}

func isPCSJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "AWSParallelComputingService.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") && strings.Contains(target, "AWSParallelComputingService.") {
		return true
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "pcs" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".pcs.") || strings.HasPrefix(host, "pcs.")
}

func parsePCSTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSParallelComputingService.") {
		return strings.TrimPrefix(target, "AWSParallelComputingService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parsePCSPayload(r *http.Request) (map[string]any, error) {
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

func respondPCSJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondPCSError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondPCSJSON(w, status, pcsError{Type: code, Message: msg})
}
