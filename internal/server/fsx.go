package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type fsxError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleFSxJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isFSxJSONCandidate(r) {
		return false
	}

	action := parseFSxTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondFSxError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := fsxOperationByName[action]; !known {
		respondFSxError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "fsx")
	if !ok {
		respondFSxError(w, status, code, msg)
		return true
	}

	payload, err := parseFSxPayload(r)
	if err != nil {
		respondFSxError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	resp := s.fsx.Handle(action, payload)
	if resp == nil {
		resp = map[string]any{}
	}
	respondFSxJSON(w, http.StatusOK, resp)
	return true
}

func isFSxJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AWSFSx.") || strings.Contains(target, "AWSSimbaAPIService_v20180301.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "AWSFSx.") || strings.Contains(target, "AWSSimbaAPIService_v20180301.")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "fsx" {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#fsx") || strings.Contains(userAgent, " fsx/") {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".fsx.") || strings.HasPrefix(host, "fsx.")
}

func parseFSxTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSSimbaAPIService_v20180301.") {
		return strings.TrimPrefix(target, "AWSSimbaAPIService_v20180301.")
	}
	if strings.HasPrefix(target, "AWSFSx.") {
		return strings.TrimPrefix(target, "AWSFSx.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseFSxPayload(r *http.Request) (map[string]any, error) {
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

func respondFSxJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondFSxError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondFSxJSON(w, status, fsxError{Type: code, Message: msg})
}
