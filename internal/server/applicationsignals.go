package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type applicationSignalsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleApplicationSignalsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isApplicationSignalsRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "application-signals")
	if !ok {
		respondApplicationSignalsError(w, status, code, msg)
		return true
	}

	action, pathParams := parseApplicationSignalsRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondApplicationSignalsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := applicationSignalsOperationByName[action]; !known {
		respondApplicationSignalsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseApplicationSignalsPayload(r)
	if err != nil {
		respondApplicationSignalsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.applicationsignals.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondApplicationSignalsJSON(w, http.StatusOK, response)
	return true
}

func isApplicationSignalsRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodPost && method != http.MethodGet && method != http.MethodDelete && method != http.MethodPatch && method != http.MethodPut {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "application-signals" {
		return false
	}
	if service == "application-signals" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".application-signals.") || strings.HasPrefix(host, "application-signals.") || strings.Contains(host, "applicationsignals") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#application-signals") || strings.Contains(userAgent, " application-signals/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/budget-report") ||
		strings.HasPrefix(path, "/exclusion-windows") ||
		strings.HasPrefix(path, "/slo") ||
		strings.HasPrefix(path, "/slos") ||
		strings.HasPrefix(path, "/service") ||
		strings.HasPrefix(path, "/services") ||
		strings.HasPrefix(path, "/service-dependencies") ||
		strings.HasPrefix(path, "/service-dependents") ||
		strings.HasPrefix(path, "/service-operations") ||
		strings.HasPrefix(path, "/grouping-configuration") ||
		strings.HasPrefix(path, "/grouping-attribute-definitions") ||
		strings.HasPrefix(path, "/auditFindings") ||
		strings.HasPrefix(path, "/events") ||
		strings.HasPrefix(path, "/start-discovery") ||
		strings.HasPrefix(path, "/tag-resource") ||
		strings.HasPrefix(path, "/untag-resource") ||
		strings.HasPrefix(path, "/tags")
}

func parseApplicationSignalsRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range applicationSignalsOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := applicationSignalsPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func applicationSignalsPathParams(template, actual string) (map[string]string, bool) {
	tSegs := applicationSignalsSplitPath(template)
	aSegs := applicationSignalsSplitPath(actual)
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

func applicationSignalsSplitPath(path string) []string {
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

func parseApplicationSignalsPayload(r *http.Request) (map[string]any, error) {
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

func respondApplicationSignalsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondApplicationSignalsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondApplicationSignalsJSON(w, status, applicationSignalsError{Type: code, Message: msg})
}
