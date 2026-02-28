package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type recoveryClusterError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleRecoveryClusterRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isRecoveryClusterRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "route53-recovery-control-config")
	if !ok {
		respondRecoveryClusterError(w, status, code, msg)
		return true
	}

	action, pathParams := parseRecoveryClusterRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondRecoveryClusterError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := recoveryClusterOperationByName[action]; !known {
		respondRecoveryClusterError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseRecoveryClusterPayload(r)
	if err != nil {
		respondRecoveryClusterError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.recoverycluster.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondRecoveryClusterJSON(w, http.StatusOK, response)
	return true
}

func isRecoveryClusterRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "route53-recovery-control-config" {
		return false
	}
	if service == "route53-recovery-control-config" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, "route53-recovery-control-config") || strings.Contains(host, "recovery-cluster") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#route53-recovery-control-config") || strings.Contains(userAgent, " route53-recovery-control-config/") {
		return true
	}

	path := strings.ToLower(strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0]))
	prefixes := []string{
		"/cluster",
		"/controlpanel",
		"/controlpanels",
		"/routingcontrol",
		"/safetyrule",
		"/resourcepolicy",
		"/tags/",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func parseRecoveryClusterRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range recoveryClusterOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := recoveryClusterPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func recoveryClusterPathParams(template, actual string) (map[string]string, bool) {
	tSegs := recoveryClusterSplitPath(template)
	aSegs := recoveryClusterSplitPath(actual)
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

func recoveryClusterSplitPath(path string) []string {
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

func parseRecoveryClusterPayload(r *http.Request) (map[string]any, error) {
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

func respondRecoveryClusterJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondRecoveryClusterError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondRecoveryClusterJSON(w, status, recoveryClusterError{Type: code, Message: msg})
}
