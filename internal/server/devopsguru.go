package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type devOpsGuruError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleDevOpsGuruJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isDevOpsGuruJSONCandidate(r) {
		return false
	}

	action := parseDevOpsGuruTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondDevOpsGuruError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := devOpsGuruOperationByName[action]; !known {
		respondDevOpsGuruError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "devops-guru")
	if !ok {
		respondDevOpsGuruError(w, status, code, msg)
		return true
	}

	payload, err := parseDevOpsGuruPayload(r)
	if err != nil {
		respondDevOpsGuruError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.devopsguru.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondDevOpsGuruJSON(w, http.StatusOK, response)
	return true
}

func isDevOpsGuruJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "DevOpsGuru") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "DevOpsGuru")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "devops-guru" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".devops-guru.") || strings.HasPrefix(host, "devops-guru.")
}

func parseDevOpsGuruTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "DevOpsGuru_20200331.") {
		return strings.TrimPrefix(target, "DevOpsGuru_20200331.")
	}
	if strings.HasPrefix(target, "com.amazonaws.devopsguru.AmazonDevOpsGuru.") {
		return strings.TrimPrefix(target, "com.amazonaws.devopsguru.AmazonDevOpsGuru.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseDevOpsGuruPayload(r *http.Request) (map[string]any, error) {
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

func respondDevOpsGuruJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondDevOpsGuruError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondDevOpsGuruJSON(w, status, devOpsGuruError{Type: code, Message: msg})
}
