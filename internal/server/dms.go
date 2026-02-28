package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type dmsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleDMSJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isDMSJSONCandidate(r) {
		return false
	}

	action := parseDMSTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondDMSError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, ok := dmsOperationByName[action]; !ok {
		respondDMSError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "dms")
	if !ok {
		respondDMSError(w, status, code, msg)
		return true
	}

	payload, err := parseDMSPayload(r)
	if err != nil {
		respondDMSError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.dms.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondDMSJSON(w, http.StatusOK, response)
	return true
}

func isDMSJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AmazonDMSv20160101.") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "AmazonDMSv20160101.")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "dms" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".dms.") || strings.HasPrefix(host, "dms.")
}

func parseDMSTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AmazonDMSv20160101.") {
		return strings.TrimPrefix(target, "AmazonDMSv20160101.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseDMSPayload(r *http.Request) (map[string]any, error) {
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

func respondDMSJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondDMSError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondDMSJSON(w, status, dmsError{Type: code, Message: msg})
}
