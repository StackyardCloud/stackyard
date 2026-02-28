package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type mpaError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleMPARESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isMPARESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "mpa")
	if !ok {
		respondMPAError(w, status, code, msg)
		return true
	}

	action, pathParams := parseMPARoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondMPAError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := mpaOperationByName[action]; !known {
		respondMPAError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseMPAPayload(r)
	if err != nil {
		respondMPAError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.mpa.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondMPAJSON(w, http.StatusOK, response)
	return true
}

func isMPARESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodGet &&
		method != http.MethodPost &&
		method != http.MethodPut &&
		method != http.MethodPatch &&
		method != http.MethodDelete {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "mpa" {
		return false
	}
	if service == "mpa" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	hostMatch := strings.Contains(host, ".mpa.") || strings.HasPrefix(host, "mpa.") || strings.Contains(host, "mpa")

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	uaMatch := strings.Contains(userAgent, "command#mpa") || strings.Contains(userAgent, " mpa/")
	if hostMatch || uaMatch {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/approval-teams") ||
		strings.HasPrefix(path, "/identity-sources") ||
		strings.HasPrefix(path, "/sessions") ||
		strings.HasPrefix(path, "/policies") ||
		strings.HasPrefix(path, "/policy-versions") ||
		strings.HasPrefix(path, "/resource-policies") ||
		strings.HasPrefix(path, "/GetResourcePolicy")
}

func parseMPARoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range mpaOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := mpaPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func mpaPathParams(template, actual string) (map[string]string, bool) {
	templatePath := mpaCanonicalPath(template)
	actualPath := mpaCanonicalPath(actual)
	templateHasTrailingSlash := templatePath != "/" && strings.HasSuffix(templatePath, "/")
	actualHasTrailingSlash := actualPath != "/" && strings.HasSuffix(actualPath, "/")
	if templateHasTrailingSlash != actualHasTrailingSlash {
		return nil, false
	}

	tSegs := mpaSplitPath(templatePath)
	aSegs := mpaSplitPath(actualPath)
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

func mpaCanonicalPath(path string) string {
	path = strings.TrimSpace(strings.SplitN(path, "?", 2)[0])
	if path == "" {
		return "/"
	}
	return path
}

func mpaSplitPath(path string) []string {
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

func parseMPAPayload(r *http.Request) (map[string]any, error) {
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

func respondMPAJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondMPAError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondMPAJSON(w, status, mpaError{Type: code, Message: msg})
}
