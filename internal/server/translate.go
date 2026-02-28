package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type translateError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleTranslateJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isTranslateJSONCandidate(r) {
		return false
	}

	action := parseTranslateTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondTranslateError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := translateOperationByName[action]; !known {
		respondTranslateError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "translate")
	if !ok {
		respondTranslateError(w, status, code, msg)
		return true
	}

	payload, err := parseTranslatePayload(r)
	if err != nil {
		respondTranslateError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.translate.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondTranslateJSON(w, http.StatusOK, response)
	return true
}

func isTranslateJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "AWSShineFrontendService_20170701.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "AWSShineFrontendService_20170701.")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "translate" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".translate.") || strings.HasPrefix(host, "translate.")
}

func parseTranslateTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSShineFrontendService_20170701.") {
		return strings.TrimPrefix(target, "AWSShineFrontendService_20170701.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseTranslatePayload(r *http.Request) (map[string]any, error) {
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

func respondTranslateJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondTranslateError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondTranslateJSON(w, status, translateError{Type: code, Message: msg})
}
