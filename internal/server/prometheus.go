package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type prometheusError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handlePrometheusRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isPrometheusRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "aps")
	if !ok {
		respondPrometheusError(w, status, code, msg)
		return true
	}

	action, pathParams := parsePrometheusRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondPrometheusError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := prometheusOperationByName[action]; !known {
		respondPrometheusError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parsePrometheusPayload(r)
	if err != nil {
		respondPrometheusError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.prometheus.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondPrometheusJSON(w, http.StatusOK, response)
	return true
}

func isPrometheusRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodGet && method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch && method != http.MethodDelete {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "aps" {
		return false
	}
	if service == "aps" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".aps.") || strings.HasPrefix(host, "aps.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#amp") || strings.Contains(userAgent, " amp/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return path == "/scraperconfiguration" ||
		strings.HasPrefix(path, "/workspaces/") ||
		path == "/workspaces" ||
		strings.HasPrefix(path, "/scrapers/") ||
		path == "/scrapers" ||
		strings.HasPrefix(path, "/tags/")
}

func parsePrometheusRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range prometheusOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := prometheusPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func prometheusPathParams(template, actual string) (map[string]string, bool) {
	tSegs := prometheusSplitPath(template)
	aSegs := prometheusSplitPath(actual)
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

func prometheusSplitPath(path string) []string {
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

func parsePrometheusPayload(r *http.Request) (map[string]any, error) {
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

func respondPrometheusJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondPrometheusError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondPrometheusJSON(w, status, prometheusError{Type: code, Message: msg})
}
