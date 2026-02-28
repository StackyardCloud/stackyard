package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type partnerCentralSellingError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handlePartnerCentralSellingJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isPartnerCentralSellingJSONCandidate(r) {
		return false
	}

	action := parsePartnerCentralSellingTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondPartnerCentralSellingError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := partnerCentralSellingOperationByName[action]; !known {
		respondPartnerCentralSellingError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "partnercentral-selling")
	if !ok {
		respondPartnerCentralSellingError(w, status, code, msg)
		return true
	}

	payload, err := parsePartnerCentralSellingPayload(r)
	if err != nil {
		respondPartnerCentralSellingError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.partnercentralselling.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondPartnerCentralSellingJSON(w, http.StatusOK, response)
	return true
}

func isPartnerCentralSellingJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "PartnerCentralSelling") || strings.Contains(target, "PartnerCentralAccount") {
		return true
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "partnercentral-selling" {
		return false
	}
	if service == "partnercentral-selling" {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") && target != "" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".partnercentral-selling.") || strings.HasPrefix(host, "partnercentral-selling.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(userAgent, "command#partnercentral-selling") || strings.Contains(userAgent, " partnercentral-selling/")
}

func parsePartnerCentralSellingTarget(target string) string {
	if target == "" {
		return ""
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parsePartnerCentralSellingPayload(r *http.Request) (map[string]any, error) {
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

func respondPartnerCentralSellingJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondPartnerCentralSellingError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondPartnerCentralSellingJSON(w, status, partnerCentralSellingError{Type: code, Message: msg})
}
