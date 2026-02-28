package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type grafanaError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleGrafanaRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isGrafanaRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "grafana")
	if !ok {
		respondGrafanaError(w, status, code, msg)
		return true
	}

	action, pathParams := parseGrafanaRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondGrafanaError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := grafanaOperationByName[action]; !known {
		respondGrafanaError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseGrafanaPayload(r)
	if err != nil {
		respondGrafanaError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.grafana.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondGrafanaJSON(w, http.StatusOK, response)
	return true
}

func isGrafanaRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodGet && method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch && method != http.MethodDelete {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "grafana" {
		return false
	}
	if service == "grafana" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".grafana.") || strings.HasPrefix(host, "grafana.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#grafana") || strings.Contains(userAgent, " grafana/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return path == "/versions" || strings.HasPrefix(path, "/workspaces") || strings.HasPrefix(path, "/tags/")
}

func parseGrafanaRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range grafanaOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := grafanaPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func grafanaPathParams(template, actual string) (map[string]string, bool) {
	tSegs := grafanaSplitPath(template)
	aSegs := grafanaSplitPath(actual)
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

func grafanaSplitPath(path string) []string {
	path = strings.TrimSpace(strings.SplitN(path, "?", 2)[0])
	if path == "" || path == "/" {
		return nil
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseGrafanaPayload(r *http.Request) (map[string]any, error) {
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

func respondGrafanaJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondGrafanaError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondGrafanaJSON(w, status, grafanaError{Type: code, Message: msg})
}
