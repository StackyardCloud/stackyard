package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type codeDeployError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCodeDeployJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCodeDeployJSONCandidate(r) {
		return false
	}

	action := parseCodeDeployTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondCodeDeployError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := codeDeployOperationByName[action]; !known {
		respondCodeDeployError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "codedeploy")
	if !ok {
		respondCodeDeployError(w, status, code, msg)
		return true
	}

	payload, err := parseCodeDeployPayload(r)
	if err != nil {
		respondCodeDeployError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.codedeploy.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondCodeDeployJSON(w, http.StatusOK, response)
	return true
}

func isCodeDeployJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "CodeDeploy_20141006") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "CodeDeploy_20141006")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "codedeploy" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".codedeploy.") || strings.HasPrefix(host, "codedeploy.")
}

func parseCodeDeployTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "CodeDeploy_20141006.") {
		return strings.TrimPrefix(target, "CodeDeploy_20141006.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseCodeDeployPayload(r *http.Request) (map[string]any, error) {
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

func respondCodeDeployJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCodeDeployError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCodeDeployJSON(w, status, codeDeployError{Type: code, Message: msg})
}
