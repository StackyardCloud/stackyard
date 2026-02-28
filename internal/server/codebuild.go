package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type codeBuildError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCodeBuildJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCodeBuildJSONCandidate(r) {
		return false
	}

	action := parseCodeBuildTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondCodeBuildError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := codeBuildOperationByName[action]; !known {
		respondCodeBuildError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "codebuild")
	if !ok {
		respondCodeBuildError(w, status, code, msg)
		return true
	}

	payload, err := parseCodeBuildPayload(r)
	if err != nil {
		respondCodeBuildError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.codebuild.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondCodeBuildJSON(w, http.StatusOK, response)
	return true
}

func isCodeBuildJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "CodeBuild_20161006") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "CodeBuild_20161006")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "codebuild" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".codebuild.") || strings.HasPrefix(host, "codebuild.")
}

func parseCodeBuildTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "CodeBuild_20161006.") {
		return strings.TrimPrefix(target, "CodeBuild_20161006.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseCodeBuildPayload(r *http.Request) (map[string]any, error) {
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

func respondCodeBuildJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCodeBuildError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCodeBuildJSON(w, status, codeBuildError{Type: code, Message: msg})
}
