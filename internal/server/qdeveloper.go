package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type qDeveloperError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleQDeveloperRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isQDeveloperRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "chatbot")
	if !ok {
		respondQDeveloperError(w, status, code, msg)
		return true
	}

	action, pathParams := parseQDeveloperRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondQDeveloperError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := qDeveloperOperationByName[action]; !known {
		respondQDeveloperError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseQDeveloperPayload(r)
	if err != nil {
		respondQDeveloperError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.qdeveloper.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondQDeveloperJSON(w, http.StatusOK, response)
	return true
}

func isQDeveloperRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "chatbot" {
		return false
	}
	if service == "chatbot" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".chatbot.") || strings.HasPrefix(host, "chatbot.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#chatbot") || strings.Contains(userAgent, " chatbot/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	for _, op := range qDeveloperOperations {
		if strings.HasPrefix(path, op.URI) {
			return true
		}
	}
	return false
}

func parseQDeveloperRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range qDeveloperOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := qDeveloperPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func qDeveloperPathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	actualPath := strings.TrimSpace(strings.SplitN(actual, "?", 2)[0])
	tSegs := qDeveloperSplitPathSegments(templatePath)
	aSegs := qDeveloperSplitPathSegments(actualPath)
	if len(tSegs) != len(aSegs) {
		return nil, false
	}
	params := map[string]string{}
	for i := range tSegs {
		t := tSegs[i]
		a := aSegs[i]
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "{"), "}"))
			if name == "" || strings.TrimSpace(a) == "" {
				return nil, false
			}
			value, err := url.PathUnescape(a)
			if err != nil {
				value = a
			}
			params[name] = value
			continue
		}
		if t != a {
			return nil, false
		}
	}
	return params, true
}

func qDeveloperSplitPathSegments(path string) []string {
	path = strings.TrimSpace(strings.SplitN(path, "?", 2)[0])
	if path == "" || path == "/" {
		return nil
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseQDeveloperPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if contentType != "" && !strings.Contains(contentType, "json") {
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

func respondQDeveloperJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondQDeveloperError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondQDeveloperJSON(w, status, qDeveloperError{Type: code, Message: msg})
}
