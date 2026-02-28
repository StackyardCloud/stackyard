package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type cloudWatchRUMError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCloudWatchRUMRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCloudWatchRUMRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "rum")
	if !ok {
		respondCloudWatchRUMError(w, status, code, msg)
		return true
	}

	payload, err := parseCloudWatchRUMPayload(r)
	if err != nil {
		respondCloudWatchRUMError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	action, pathParams := parseCloudWatchRUMRoute(r.Method, rawRequestPath(r), r.Header.Get("X-Amz-Target"), payload, r.URL.Query())
	if action == "" {
		respondCloudWatchRUMError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := cloudWatchRUMOperationByName[action]; !known {
		respondCloudWatchRUMError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	response := s.cloudwatchrum.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondCloudWatchRUMJSON(w, http.StatusOK, response)
	return true
}

func isCloudWatchRUMRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodPost && method != http.MethodGet && method != http.MethodDelete && method != http.MethodPatch && method != http.MethodPut {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "rum" {
		return false
	}
	if service == "rum" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".rum.") || strings.HasPrefix(host, "rum.") || strings.Contains(host, "cloudwatchrum") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#rum") || strings.Contains(userAgent, " rum/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/appmonitor") ||
		strings.HasPrefix(path, "/appmonitors") ||
		strings.HasPrefix(path, "/rummetrics") ||
		strings.HasPrefix(path, "/tags")
}

func parseCloudWatchRUMRoute(method, requestPath, targetHint string, payload map[string]any, query url.Values) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	type routeMatch struct {
		op     cloudWatchRUMOperation
		params map[string]string
	}

	matches := make([]routeMatch, 0, 2)
	for _, op := range cloudWatchRUMOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := cloudWatchRUMPathParams(op.URI, path)
		if !ok {
			continue
		}
		matches = append(matches, routeMatch{op: op, params: params})
	}
	if len(matches) == 0 {
		return "", nil
	}
	if len(matches) == 1 {
		return matches[0].op.Name, matches[0].params
	}

	hints := []string{
		targetHint,
		strings.TrimSpace(query.Get("operation")),
		strings.TrimSpace(query.Get("Action")),
		cloudWatchRUMDefaultString(payload, "operation", ""),
		cloudWatchRUMDefaultString(payload, "Action", ""),
	}
	for _, hint := range hints {
		n := cloudWatchRUMNormalizeToken(hint)
		if n == "" {
			continue
		}
		for _, m := range matches {
			op := cloudWatchRUMNormalizeToken(m.op.Name)
			if strings.Contains(n, op) || strings.Contains(op, n) {
				return m.op.Name, m.params
			}
		}
	}

	return matches[0].op.Name, matches[0].params
}

func cloudWatchRUMPathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	actualPath := strings.TrimSpace(strings.SplitN(actual, "?", 2)[0])
	tSegs := cloudWatchRUMSplitPathSegments(templatePath)
	aSegs := cloudWatchRUMSplitPathSegments(actualPath)
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

func cloudWatchRUMSplitPathSegments(path string) []string {
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

func parseCloudWatchRUMPayload(r *http.Request) (map[string]any, error) {
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

func respondCloudWatchRUMJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCloudWatchRUMError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCloudWatchRUMJSON(w, status, cloudWatchRUMError{Type: code, Message: msg})
}

func cloudWatchRUMNormalizeToken(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return ""
	}
	replacer := strings.NewReplacer("-", "", "_", "", ".", "", ":", "", " ", "", "/", "")
	return replacer.Replace(v)
}
