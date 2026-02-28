package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type directoryServiceError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleDirectoryServiceJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isDirectoryServiceJSONCandidate(r) {
		return false
	}

	action := parseDirectoryServiceTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondDirectoryServiceError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := directoryServiceOperationByName[action]; !known {
		respondDirectoryServiceError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "ds")
	if !ok {
		respondDirectoryServiceError(w, status, code, msg)
		return true
	}

	payload, err := parseDirectoryServicePayload(r)
	if err != nil {
		respondDirectoryServiceError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.directoryservice.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondDirectoryServiceJSON(w, http.StatusOK, response)
	return true
}

func isDirectoryServiceJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "DirectoryService_20150416.") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "DirectoryService_20150416.")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "ds" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".ds.") || strings.HasPrefix(host, "ds.")
}

func parseDirectoryServiceTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "DirectoryService_20150416.") {
		return strings.TrimPrefix(target, "DirectoryService_20150416.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseDirectoryServicePayload(r *http.Request) (map[string]any, error) {
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

func respondDirectoryServiceJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondDirectoryServiceError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondDirectoryServiceJSON(w, status, directoryServiceError{Type: code, Message: msg})
}
