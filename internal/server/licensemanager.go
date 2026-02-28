package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type licenseManagerError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleLicenseManagerJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isLicenseManagerJSONCandidate(r) {
		return false
	}

	action := parseLicenseManagerTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondLicenseManagerError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := licenseManagerOperationByName[action]; !known {
		respondLicenseManagerError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "license-manager")
	if !ok {
		respondLicenseManagerError(w, status, code, msg)
		return true
	}

	payload, err := parseLicenseManagerPayload(r)
	if err != nil {
		respondLicenseManagerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.licensemanager.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondLicenseManagerJSON(w, http.StatusOK, response)
	return true
}

func isLicenseManagerJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AWSLicenseManager") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "AWSLicenseManager")
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "license-manager" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".license-manager.") ||
		strings.HasPrefix(host, "license-manager.") ||
		strings.Contains(host, ".licensemanager.") ||
		strings.HasPrefix(host, "licensemanager.")
}

func parseLicenseManagerTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSLicenseManager.") {
		return strings.TrimPrefix(target, "AWSLicenseManager.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseLicenseManagerPayload(r *http.Request) (map[string]any, error) {
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

func respondLicenseManagerJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondLicenseManagerError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondLicenseManagerJSON(w, status, licenseManagerError{Type: code, Message: msg})
}
