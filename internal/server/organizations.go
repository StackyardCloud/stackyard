package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type organizationsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleOrganizationsJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isOrganizationsJSONCandidate(r) {
		return false
	}

	action := parseOrganizationsTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondOrganizationsError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := organizationsOperationByName[action]; !known {
		respondOrganizationsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "organizations")
	if !ok {
		respondOrganizationsError(w, status, code, msg)
		return true
	}

	payload, err := parseOrganizationsPayload(r)
	if err != nil {
		respondOrganizationsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.organizations.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondOrganizationsJSON(w, http.StatusOK, response)
	return true
}

func isOrganizationsJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AWSOrganizationsV20161128") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "AWSOrganizationsV20161128")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "organizations" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".organizations.") || strings.HasPrefix(host, "organizations.")
}

func parseOrganizationsTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSOrganizationsV20161128.") {
		return strings.TrimPrefix(target, "AWSOrganizationsV20161128.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseOrganizationsPayload(r *http.Request) (map[string]any, error) {
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

func respondOrganizationsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondOrganizationsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondOrganizationsJSON(w, status, organizationsError{Type: code, Message: msg})
}
