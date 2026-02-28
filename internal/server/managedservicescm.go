package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type managedServicesCMError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleManagedServicesCMJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isManagedServicesCMJSONCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "amscm")
	if !ok {
		respondManagedServicesCMError(w, status, code, msg)
		return true
	}

	action := parseManagedServicesCMTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondManagedServicesCMError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := managedServicesCMOperationByName[action]; !known {
		respondManagedServicesCMError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseManagedServicesCMPayload(r)
	if err != nil {
		respondManagedServicesCMError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.managedservicescm.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondManagedServicesCMJSON(w, http.StatusOK, response)
	return true
}

func isManagedServicesCMJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if action := parseManagedServicesCMTarget(target); action != "" {
		if _, known := managedServicesCMOperationByName[action]; known {
			return true
		}
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "amscm" {
		return false
	}
	if service == "amscm" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".managedservices.") || strings.HasPrefix(host, "managedservices.") || strings.HasPrefix(host, "amscm.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") && strings.HasPrefix(target, "AWSManagedServicesChangeManagement.") {
		return true
	}
	return false
}

func parseManagedServicesCMTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return strings.TrimSpace(target[dot+1:])
	}
	return target
}

func parseManagedServicesCMPayload(r *http.Request) (map[string]any, error) {
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

func respondManagedServicesCMJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondManagedServicesCMError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondManagedServicesCMJSON(w, status, managedServicesCMError{Type: code, Message: msg})
}
