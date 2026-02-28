package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type directConnectError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleDirectConnectJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isDirectConnectJSONCandidate(r) {
		return false
	}

	action := parseDirectConnectTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondDirectConnectError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := directConnectOperationByName[action]; !known {
		respondDirectConnectError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "directconnect")
	if !ok {
		respondDirectConnectError(w, status, code, msg)
		return true
	}

	payload, err := parseDirectConnectPayload(r)
	if err != nil {
		respondDirectConnectError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.directconnect.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondDirectConnectJSON(w, http.StatusOK, response)
	return true
}

func isDirectConnectJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "OvertureService.") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "OvertureService.")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "directconnect" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".directconnect.") || strings.HasPrefix(host, "directconnect.")
}

func parseDirectConnectTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "OvertureService.") {
		return strings.TrimPrefix(target, "OvertureService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseDirectConnectPayload(r *http.Request) (map[string]any, error) {
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

func respondDirectConnectJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondDirectConnectError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondDirectConnectJSON(w, status, directConnectError{Type: code, Message: msg})
}
