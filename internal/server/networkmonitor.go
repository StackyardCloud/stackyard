package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type networkMonitorError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleNetworkMonitorRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isNetworkMonitorRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "networkmonitor")
	if !ok {
		respondNetworkMonitorError(w, status, code, msg)
		return true
	}

	action, pathParams := parseNetworkMonitorRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondNetworkMonitorError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := networkMonitorOperationByName[action]; !known {
		respondNetworkMonitorError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseNetworkMonitorPayload(r)
	if err != nil {
		respondNetworkMonitorError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.networkmonitor.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondNetworkMonitorJSON(w, http.StatusOK, response)
	return true
}

func isNetworkMonitorRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodPost && method != http.MethodGet && method != http.MethodDelete && method != http.MethodPatch {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "networkmonitor" {
		return false
	}
	if service == "networkmonitor" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".networkmonitor.") || strings.HasPrefix(host, "networkmonitor.") || strings.Contains(host, "networkmonitor") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#networkmonitor") || strings.Contains(userAgent, " networkmonitor/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/monitors") || strings.HasPrefix(path, "/tags/")
}

func parseNetworkMonitorRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range networkMonitorOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := networkMonitorPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func networkMonitorPathParams(template, actual string) (map[string]string, bool) {
	tSegs := networkMonitorSplitPath(template)
	aSegs := networkMonitorSplitPath(actual)
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

func networkMonitorSplitPath(path string) []string {
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

func parseNetworkMonitorPayload(r *http.Request) (map[string]any, error) {
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

func respondNetworkMonitorJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondNetworkMonitorError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondNetworkMonitorJSON(w, status, networkMonitorError{Type: code, Message: msg})
}
