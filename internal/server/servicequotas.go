package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type serviceQuotasError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleServiceQuotasJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isServiceQuotasJSONCandidate(r) {
		return false
	}

	action := parseServiceQuotasTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondServiceQuotasError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := serviceQuotasOperationByName[action]; !known {
		respondServiceQuotasError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "servicequotas")
	if !ok {
		respondServiceQuotasError(w, status, code, msg)
		return true
	}

	payload, err := parseServiceQuotasPayload(r)
	if err != nil {
		respondServiceQuotasError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.servicequotas.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondServiceQuotasJSON(w, http.StatusOK, response)
	return true
}

func isServiceQuotasJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "ServiceQuotasV20190624") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "ServiceQuotasV20190624")
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "servicequotas" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".servicequotas.") ||
		strings.HasPrefix(host, "servicequotas.") ||
		strings.Contains(host, ".service-quotas.") ||
		strings.HasPrefix(host, "service-quotas.")
}

func parseServiceQuotasTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "ServiceQuotasV20190624.") {
		return strings.TrimPrefix(target, "ServiceQuotasV20190624.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseServiceQuotasPayload(r *http.Request) (map[string]any, error) {
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

func respondServiceQuotasJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondServiceQuotasError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondServiceQuotasJSON(w, status, serviceQuotasError{Type: code, Message: msg})
}
