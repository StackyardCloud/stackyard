package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type b2biError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleB2BIJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isB2BIJSONCandidate(r) {
		return false
	}

	action := parseB2BITarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondB2BIError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := b2biOperationByName[action]; !known {
		respondB2BIError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "b2bi")
	if !ok {
		respondB2BIError(w, status, code, msg)
		return true
	}

	payload, err := parseB2BIPayload(r)
	if err != nil {
		respondB2BIError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.b2bi.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondB2BIJSON(w, http.StatusOK, response)
	return true
}

func isB2BIJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "B2BI.") {
		return true
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "b2bi" {
		return false
	}
	if service == "b2bi" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".b2bi.") || strings.HasPrefix(host, "b2bi.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#b2bi") || strings.Contains(userAgent, " b2bi/") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") {
		if target == "" {
			return service == "b2bi" || strings.Contains(host, "b2bi")
		}
		return strings.Contains(target, ".")
	}

	return false
}

func parseB2BITarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "B2BI.") {
		return strings.TrimPrefix(target, "B2BI.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseB2BIPayload(r *http.Request) (map[string]any, error) {
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

func respondB2BIJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondB2BIError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondB2BIJSON(w, status, b2biError{Type: code, Message: msg})
}
