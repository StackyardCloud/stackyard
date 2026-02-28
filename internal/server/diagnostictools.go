package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type diagnosticToolsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleDiagnosticToolsJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isDiagnosticToolsJSONCandidate(r) {
		return false
	}

	action := parseDiagnosticToolsTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondDiagnosticToolsError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := diagnosticToolsOperationByName[action]; !known {
		respondDiagnosticToolsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "troubleshooting")
	if !ok {
		respondDiagnosticToolsError(w, status, code, msg)
		return true
	}

	payload, err := parseDiagnosticToolsPayload(r)
	if err != nil {
		respondDiagnosticToolsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.diagnostictools.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondDiagnosticToolsJSON(w, http.StatusOK, response)
	return true
}

func isDiagnosticToolsJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "Troubleshooting.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.Contains(target, "Troubleshooting.")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "troubleshooting" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".diagnostic-tools.") || strings.Contains(host, ".troubleshooting.")
}

func parseDiagnosticToolsTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "Troubleshooting.") {
		return strings.TrimPrefix(target, "Troubleshooting.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseDiagnosticToolsPayload(r *http.Request) (map[string]any, error) {
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

func respondDiagnosticToolsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondDiagnosticToolsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondDiagnosticToolsJSON(w, status, diagnosticToolsError{Type: code, Message: msg})
}
