package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type drsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleDRSRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isDRSRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "drs")
	if !ok {
		respondDRSError(w, status, code, msg)
		return true
	}

	action, pathParams := parseDRSRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondDRSError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := drsOperationByName[action]; !known {
		respondDRSError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseDRSPayload(r)
	if err != nil {
		respondDRSError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}
	for key, value := range pathParams {
		if _, exists := payload[key]; !exists {
			payload[key] = value
		}
	}
	for key, values := range r.URL.Query() {
		if len(values) == 0 {
			continue
		}
		if _, exists := payload[key]; exists {
			continue
		}
		if len(values) == 1 {
			payload[key] = values[0]
			continue
		}
		items := make([]any, 0, len(values))
		for _, value := range values {
			items = append(items, value)
		}
		payload[key] = items
	}

	response := s.drs.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondDRSJSON(w, http.StatusOK, response)
	return true
}

func isDRSRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "drs" {
		return false
	}
	if service == "drs" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".drs.") || strings.HasPrefix(host, "drs.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#drs") || strings.Contains(userAgent, " drs/") {
		return true
	}

	action, _ := parseDRSRoute(r.Method, rawRequestPath(r))
	return action != ""
}

func parseDRSRoute(method, requestPath string) (string, map[string]string) {
	normalizedMethod := strings.ToUpper(strings.TrimSpace(method))
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range drsOperations {
		if !strings.EqualFold(op.Method, normalizedMethod) {
			continue
		}
		params, ok := drsPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func drsPathParams(template, actual string) (map[string]string, bool) {
	tSegments := drsSplitPath(template)
	aSegments := drsSplitPath(actual)
	if len(tSegments) != len(aSegments) {
		return nil, false
	}

	params := map[string]string{}
	for i := range tSegments {
		t := tSegments[i]
		a := aSegments[i]
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

func drsSplitPath(path string) []string {
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

func parseDRSPayload(r *http.Request) (map[string]any, error) {
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

func respondDRSJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondDRSError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondDRSJSON(w, status, drsError{Type: code, Message: msg})
}
