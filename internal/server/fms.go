package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type fmsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleFMSJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isFMSJSONCandidate(r) {
		return false
	}

	action := parseFMSTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondFMSError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := fmsOperationByName[action]; !known {
		respondFMSError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "fms")
	if !ok {
		respondFMSError(w, status, code, msg)
		return true
	}

	payload, err := parseFMSPayload(r)
	if err != nil {
		respondFMSError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.fms.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondFMSJSON(w, http.StatusOK, response)
	return true
}

func isFMSJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AWSFMS_20180101") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "AWSFMS_20180101")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "fms" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".fms.") || strings.HasPrefix(host, "fms.")
}

func parseFMSTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSFMS_20180101.") {
		return strings.TrimPrefix(target, "AWSFMS_20180101.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseFMSPayload(r *http.Request) (map[string]any, error) {
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

func respondFMSJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondFMSError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondFMSJSON(w, status, fmsError{Type: code, Message: msg})
}
