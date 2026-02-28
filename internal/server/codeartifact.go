package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type codeArtifactError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCodeArtifactRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCodeArtifactRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "codeartifact")
	if !ok {
		respondCodeArtifactError(w, status, code, msg)
		return true
	}

	payload, err := parseCodeArtifactPayload(r)
	if err != nil {
		respondCodeArtifactError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.codeartifact.Handle(r, payload)
	if response.Status == 0 {
		response.Status = http.StatusOK
	}
	if strings.TrimSpace(response.ContentType) == "" {
		response.ContentType = "application/json"
	}
	w.Header().Set("Content-Type", response.ContentType)
	w.WriteHeader(response.Status)
	if len(response.RawBody) > 0 {
		_, _ = w.Write(response.RawBody)
		return true
	}
	if response.Body == nil {
		response.Body = map[string]any{}
	}
	_ = json.NewEncoder(w).Encode(response.Body)
	return true
}

func isCodeArtifactRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "codeartifact" {
		return false
	}
	if service == "codeartifact" {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	if path == "/v1" || strings.HasPrefix(path, "/v1/") {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".codeartifact.") || strings.HasPrefix(host, "codeartifact.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#codeartifact") || strings.Contains(userAgent, " codeartifact/") {
		return true
	}

	amzUserAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Amz-User-Agent")))
	return strings.Contains(amzUserAgent, "command#codeartifact") || strings.Contains(amzUserAgent, " codeartifact/")
}

func parseCodeArtifactPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if !strings.Contains(contentType, "json") {
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

func respondCodeArtifactJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCodeArtifactError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCodeArtifactJSON(w, status, codeArtifactError{Type: code, Message: msg})
}
