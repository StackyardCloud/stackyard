package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type computeOptimizerError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleComputeOptimizerJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isComputeOptimizerJSONCandidate(r) {
		return false
	}

	action := parseComputeOptimizerTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondComputeOptimizerError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := computeOptimizerOperationByName[action]; !known {
		respondComputeOptimizerError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "compute-optimizer")
	if !ok {
		respondComputeOptimizerError(w, status, code, msg)
		return true
	}

	payload, err := parseComputeOptimizerPayload(r)
	if err != nil {
		respondComputeOptimizerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.computeoptimizer.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondComputeOptimizerJSON(w, http.StatusOK, response)
	return true
}

func isComputeOptimizerJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "ComputeOptimizerService.") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(target, "ComputeOptimizerService.")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "compute-optimizer" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".compute-optimizer.") || strings.HasPrefix(host, "compute-optimizer.")
}

func parseComputeOptimizerTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "ComputeOptimizerService.") {
		return strings.TrimPrefix(target, "ComputeOptimizerService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseComputeOptimizerPayload(r *http.Request) (map[string]any, error) {
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

func respondComputeOptimizerJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondComputeOptimizerError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondComputeOptimizerJSON(w, status, computeOptimizerError{Type: code, Message: msg})
}
