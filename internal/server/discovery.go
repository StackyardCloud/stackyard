package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type discoveryError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleDiscoveryJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isDiscoveryJSONCandidate(r) {
		return false
	}

	action := parseDiscoveryTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondDiscoveryError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := discoveryOperationByName[action]; !known {
		respondDiscoveryError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "discovery")
	if !ok {
		respondDiscoveryError(w, status, code, msg)
		return true
	}

	payload, err := parseDiscoveryPayload(r)
	if err != nil {
		respondDiscoveryError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.discovery.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondDiscoveryJSON(w, http.StatusOK, response)
	return true
}

func isDiscoveryJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "AWSPoseidonService_V2015_11_01.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "AWSPoseidonService_V2015_11_01.")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "discovery" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".discovery.") || strings.HasPrefix(host, "discovery.")
}

func parseDiscoveryTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSPoseidonService_V2015_11_01.") {
		return strings.TrimPrefix(target, "AWSPoseidonService_V2015_11_01.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseDiscoveryPayload(r *http.Request) (map[string]any, error) {
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

func respondDiscoveryJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondDiscoveryError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondDiscoveryJSON(w, status, discoveryError{Type: code, Message: msg})
}
