package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type codeGuruError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCodeGuruRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCodeGuruRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "codeguru-reviewer")
	if !ok {
		respondCodeGuruError(w, status, code, msg)
		return true
	}

	action, pathParams := parseCodeGuruRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondCodeGuruError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := codeGuruOperationByName[action]; !known {
		respondCodeGuruError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseCodeGuruPayload(r)
	if err != nil {
		respondCodeGuruError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.codeguru.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondCodeGuruJSON(w, http.StatusOK, response)
	return true
}

func isCodeGuruRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "codeguru-reviewer" {
		return false
	}
	if service == "codeguru-reviewer" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".codeguru-reviewer.") || strings.HasPrefix(host, "codeguru-reviewer.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#codeguru-reviewer") || strings.Contains(userAgent, " codeguru-reviewer/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/associations") ||
		strings.HasPrefix(path, "/codereviews") ||
		strings.HasPrefix(path, "/createCodeReviewInternal") ||
		strings.HasPrefix(path, "/feedback") ||
		strings.HasPrefix(path, "/metrics") ||
		strings.HasPrefix(path, "/tags") ||
		strings.HasPrefix(path, "/thirdPartyRepositories")
}

func parseCodeGuruAction(method, requestPath string) string {
	action, _ := parseCodeGuruRoute(method, requestPath)
	return action
}

func parseCodeGuruRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range codeGuruOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := codeGuruPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func codeGuruPathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	actualPath := strings.TrimSpace(strings.SplitN(actual, "?", 2)[0])
	tSegs := codeGuruSplitPathSegments(templatePath)
	aSegs := codeGuruSplitPathSegments(actualPath)
	if len(tSegs) != len(aSegs) {
		return nil, false
	}
	params := map[string]string{}
	for i := range tSegs {
		t := tSegs[i]
		a := aSegs[i]
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			if strings.TrimSpace(a) == "" {
				return nil, false
			}
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

func codeGuruSplitPathSegments(path string) []string {
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

func parseCodeGuruPayload(r *http.Request) (map[string]any, error) {
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

func respondCodeGuruJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCodeGuruError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCodeGuruJSON(w, status, codeGuruError{Type: code, Message: msg})
}
