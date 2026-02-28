package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type globalAcceleratorError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleGlobalAcceleratorJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isGlobalAcceleratorJSONCandidate(r) {
		return false
	}

	action := parseGlobalAcceleratorTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondGlobalAcceleratorError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := globalAcceleratorOperationByName[action]; !known {
		respondGlobalAcceleratorError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "globalaccelerator")
	if !ok {
		respondGlobalAcceleratorError(w, status, code, msg)
		return true
	}

	payload, err := parseGlobalAcceleratorPayload(r)
	if err != nil {
		respondGlobalAcceleratorError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.globalaccelerator.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondGlobalAcceleratorJSON(w, http.StatusOK, response)
	return true
}

func isGlobalAcceleratorJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "GlobalAccelerator_V20180706") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "GlobalAccelerator_V20180706")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "globalaccelerator" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".globalaccelerator.") || strings.HasPrefix(host, "globalaccelerator.")
}

func parseGlobalAcceleratorTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "GlobalAccelerator_V20180706.") {
		return strings.TrimPrefix(target, "GlobalAccelerator_V20180706.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseGlobalAcceleratorPayload(r *http.Request) (map[string]any, error) {
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

func respondGlobalAcceleratorJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondGlobalAcceleratorError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondGlobalAcceleratorJSON(w, status, globalAcceleratorError{Type: code, Message: msg})
}
