package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type workspacesError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleWorkSpacesJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isWorkSpacesJSONCandidate(r) {
		return false
	}

	action := parseWorkSpacesTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondWorkSpacesError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := workspacesOperationByName[action]; !known {
		respondWorkSpacesError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "workspaces")
	if !ok {
		respondWorkSpacesError(w, status, code, msg)
		return true
	}

	payload, err := parseWorkSpacesPayload(r)
	if err != nil {
		respondWorkSpacesError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.workspaces.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondWorkSpacesJSON(w, http.StatusOK, response)
	return true
}

func isWorkSpacesJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "WorkspacesService.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "WorkspacesService.")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "workspaces" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".workspaces.") || strings.HasPrefix(host, "workspaces.")
}

func parseWorkSpacesTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "WorkspacesService.") {
		return strings.TrimPrefix(target, "WorkspacesService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseWorkSpacesPayload(r *http.Request) (map[string]any, error) {
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

func respondWorkSpacesJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondWorkSpacesError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondWorkSpacesJSON(w, status, workspacesError{Type: code, Message: msg})
}
