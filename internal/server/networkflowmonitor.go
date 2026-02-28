package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type networkFlowMonitorError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleNetworkFlowMonitorRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isNetworkFlowMonitorRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "networkflowmonitor")
	if !ok {
		respondNetworkFlowMonitorError(w, status, code, msg)
		return true
	}

	action, pathParams := parseNetworkFlowMonitorRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondNetworkFlowMonitorError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := networkFlowMonitorOperationByName[action]; !known {
		respondNetworkFlowMonitorError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseNetworkFlowMonitorPayload(r)
	if err != nil {
		respondNetworkFlowMonitorError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.networkflowmonitor.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondNetworkFlowMonitorJSON(w, http.StatusOK, response)
	return true
}

func isNetworkFlowMonitorRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodPost && method != http.MethodGet && method != http.MethodDelete && method != http.MethodPatch {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "networkflowmonitor" {
		return false
	}
	if service == "networkflowmonitor" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".networkflowmonitor.") || strings.HasPrefix(host, "networkflowmonitor.") || strings.Contains(host, "networkflowmonitor") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#networkflowmonitor") || strings.Contains(userAgent, " networkflowmonitor/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/monitors") ||
		strings.HasPrefix(path, "/scopes") ||
		strings.HasPrefix(path, "/workloadInsights") ||
		strings.HasPrefix(path, "/tags")
}

func parseNetworkFlowMonitorRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range networkFlowMonitorOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := networkFlowMonitorPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func networkFlowMonitorPathParams(template, actual string) (map[string]string, bool) {
	tSegs := networkFlowMonitorSplitPath(template)
	aSegs := networkFlowMonitorSplitPath(actual)
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

func networkFlowMonitorSplitPath(path string) []string {
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

func parseNetworkFlowMonitorPayload(r *http.Request) (map[string]any, error) {
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

func respondNetworkFlowMonitorJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondNetworkFlowMonitorError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondNetworkFlowMonitorJSON(w, status, networkFlowMonitorError{Type: code, Message: msg})
}
