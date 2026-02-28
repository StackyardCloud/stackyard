package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type m2Error struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleM2RESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isM2RESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "m2")
	if !ok {
		respondM2Error(w, status, code, msg)
		return true
	}

	action, pathParams := parseM2Route(r.Method, rawRequestPath(r))
	if action == "" {
		respondM2Error(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := m2OperationByName[action]; !known {
		respondM2Error(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseM2Payload(r)
	if err != nil {
		respondM2Error(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.m2.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondM2JSON(w, http.StatusOK, response)
	return true
}

func isM2RESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPatch:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "m2" {
		return false
	}
	if service == "m2" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".m2.") || strings.HasPrefix(host, "m2.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#m2") || strings.Contains(userAgent, " m2/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	return strings.HasPrefix(path, "/applications") ||
		strings.HasPrefix(path, "/environments") ||
		strings.HasPrefix(path, "/engine-versions") ||
		strings.HasPrefix(path, "/signed-bi-url") ||
		strings.HasPrefix(path, "/tags")
}

func parseM2Route(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range m2Operations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := m2PathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func m2PathParams(template, actual string) (map[string]string, bool) {
	tSegs := m2SplitPath(template)
	aSegs := m2SplitPath(actual)
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

func m2SplitPath(path string) []string {
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

func parseM2Payload(r *http.Request) (map[string]any, error) {
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

func respondM2JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondM2Error(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondM2JSON(w, status, m2Error{Type: code, Message: msg})
}
