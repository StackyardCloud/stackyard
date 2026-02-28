package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type supportAppError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleSupportAppRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSupportAppRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "supportapp")
	if !ok {
		respondSupportAppError(w, status, code, msg)
		return true
	}

	action := parseSupportAppRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondSupportAppError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := supportAppOperationByName[action]; !known {
		respondSupportAppError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseSupportAppPayload(r)
	if err != nil {
		respondSupportAppError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.supportapp.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondSupportAppJSON(w, http.StatusOK, response)
	return true
}

func isSupportAppRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "supportapp" {
		return false
	}
	if service == "supportapp" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".supportapp.") ||
		strings.HasPrefix(host, "supportapp.") ||
		strings.Contains(host, ".support-app.") ||
		strings.HasPrefix(host, "support-app.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#support-app") || strings.Contains(userAgent, " support-app/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	return strings.HasPrefix(path, "/control/")
}

func parseSupportAppRoute(method, requestPath string) string {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range supportAppOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		if op.URI == path {
			return op.Name
		}
	}
	return ""
}

func parseSupportAppPayload(r *http.Request) (map[string]any, error) {
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

func respondSupportAppJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSupportAppError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSupportAppJSON(w, status, supportAppError{Type: code, Message: msg})
}
