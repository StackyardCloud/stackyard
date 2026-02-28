package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type codeGuruProfilerError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCodeGuruProfilerRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCodeGuruProfilerRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "codeguru-profiler")
	if !ok {
		respondCodeGuruProfilerError(w, status, code, msg)
		return true
	}

	action, pathParams := parseCodeGuruProfilerRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondCodeGuruProfilerError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := codeGuruProfilerOperationByName[action]; !known {
		respondCodeGuruProfilerError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseCodeGuruProfilerPayload(r)
	if err != nil {
		respondCodeGuruProfilerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.codeguruprofiler.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondCodeGuruProfilerJSON(w, http.StatusOK, response)
	return true
}

func isCodeGuruProfilerRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "codeguru-profiler" {
		return false
	}
	if service == "codeguru-profiler" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".codeguru-profiler.") || strings.HasPrefix(host, "codeguru-profiler.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#codeguruprofiler") ||
		strings.Contains(userAgent, " codeguruprofiler/") ||
		strings.Contains(userAgent, "command#codeguru-profiler") ||
		strings.Contains(userAgent, " codeguru-profiler/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/profilingGroups") ||
		strings.HasPrefix(path, "/internal/findingsReports") ||
		strings.HasPrefix(path, "/internal/profilingGroups") ||
		strings.HasPrefix(path, "/tags")
}

func parseCodeGuruProfilerAction(method, requestPath string) string {
	action, _ := parseCodeGuruProfilerRoute(method, requestPath)
	return action
}

func parseCodeGuruProfilerRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range codeGuruProfilerOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := codeGuruProfilerPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func codeGuruProfilerPathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	actualPath := strings.TrimSpace(strings.SplitN(actual, "?", 2)[0])
	tSegs := codeGuruProfilerSplitPathSegments(templatePath)
	aSegs := codeGuruProfilerSplitPathSegments(actualPath)
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

func codeGuruProfilerSplitPathSegments(path string) []string {
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

func parseCodeGuruProfilerPayload(r *http.Request) (map[string]any, error) {
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

func respondCodeGuruProfilerJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCodeGuruProfilerError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCodeGuruProfilerJSON(w, status, codeGuruProfilerError{Type: code, Message: msg})
}
