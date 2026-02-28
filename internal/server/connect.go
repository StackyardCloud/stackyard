package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type connectError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleConnectRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isConnectRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "connect")
	if !ok {
		respondConnectError(w, status, code, msg)
		return true
	}

	action, pathParams := parseConnectRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondConnectError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := connectOperationByName[action]; !known {
		respondConnectError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseConnectPayload(r)
	if err != nil {
		respondConnectError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.connect.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondConnectJSON(w, http.StatusOK, response)
	return true
}

func isConnectRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "connect" {
		return false
	}
	if service == "connect" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if host == "connect.amazonaws.com" || strings.HasPrefix(host, "connect.") || strings.Contains(host, ".connect.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#connect") || strings.Contains(userAgent, " connect/") {
		return true
	}

	action, _ := parseConnectRoute(method, rawRequestPath(r))
	return action != ""
}

func parseConnectRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range connectOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := connectPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	return "", nil
}

func connectPathParams(template, actual string) (map[string]string, bool) {
	tSegs := connectSplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	aSegs := connectSplitPath(strings.SplitN(strings.TrimSpace(actual), "?", 2)[0])
	if len(tSegs) != len(aSegs) {
		return nil, false
	}

	params := map[string]string{}
	for i := range tSegs {
		t := tSegs[i]
		a := aSegs[i]
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "{"), "}"))
			if name == "" {
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

func connectSplitPath(path string) []string {
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

func parseConnectPayload(r *http.Request) (map[string]any, error) {
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

func respondConnectJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondConnectError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondConnectJSON(w, status, connectError{Type: code, Message: msg})
}
