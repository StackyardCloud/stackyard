package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type auditManagerError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleAuditManagerRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isAuditManagerRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "auditmanager")
	if !ok {
		respondAuditManagerError(w, status, code, msg)
		return true
	}

	action, pathParams := parseAuditManagerRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondAuditManagerError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := auditManagerOperationByName[action]; !known {
		respondAuditManagerError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseAuditManagerPayload(r)
	if err != nil {
		respondAuditManagerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.auditmanager.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondAuditManagerJSON(w, http.StatusOK, response)
	return true
}

func isAuditManagerRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "auditmanager" {
		return false
	}
	if service == "auditmanager" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".auditmanager.") || strings.HasPrefix(host, "auditmanager.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#auditmanager") || strings.Contains(userAgent, " auditmanager/") {
		return true
	}

	action, _ := parseAuditManagerRoute(method, rawRequestPath(r))
	return action != ""
}

func parseAuditManagerRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range auditManagerOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		uriPath := strings.TrimSpace(strings.SplitN(op.URI, "?", 2)[0])
		params, ok := auditManagerPathParams(uriPath, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func auditManagerPathParams(template, actual string) (map[string]string, bool) {
	tSegs := auditManagerSplitPath(template)
	aSegs := auditManagerSplitPath(actual)
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

func auditManagerSplitPath(path string) []string {
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

func parseAuditManagerPayload(r *http.Request) (map[string]any, error) {
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

func respondAuditManagerJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondAuditManagerError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondAuditManagerJSON(w, status, auditManagerError{Type: code, Message: msg})
}
