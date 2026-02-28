package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type securityIRError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleSecurityIRRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSecurityIRRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "security-ir")
	if !ok {
		// Some clients/service models can surface the service as "securityir" in hints.
		ok, status, code, msg, _ = s.validateSigV4WithService(r, "securityir")
		if !ok {
			respondSecurityIRError(w, status, code, msg)
			return true
		}
	}

	action, pathParams := parseSecurityIRRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondSecurityIRError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := securityIROperationByName[action]; !known {
		respondSecurityIRError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseSecurityIRPayload(r)
	if err != nil {
		respondSecurityIRError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.securityir.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondSecurityIRJSON(w, http.StatusOK, response)
	return true
}

func isSecurityIRRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "security-ir" && service != "securityir" {
		return false
	}
	if service == "security-ir" || service == "securityir" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".security-ir.") || strings.HasPrefix(host, "security-ir.") || strings.Contains(host, "securityir") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#security-ir.") ||
		strings.Contains(userAgent, "command#securityir.") ||
		strings.Contains(userAgent, " security-ir/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	return strings.HasPrefix(path, "/v1/cases/") ||
		strings.HasPrefix(path, "/v1/membership") ||
		path == "/v1/list-cases" ||
		path == "/v1/memberships" ||
		strings.HasPrefix(path, "/v1/tags/") ||
		path == "/v1/create-case"
}

func parseSecurityIRRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range securityIROperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := securityIRPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func securityIRPathParams(template, actual string) (map[string]string, bool) {
	tSegs := securityIRSplitPath(template)
	aSegs := securityIRSplitPath(actual)
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

func securityIRSplitPath(path string) []string {
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

func parseSecurityIRPayload(r *http.Request) (map[string]any, error) {
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

func respondSecurityIRJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSecurityIRError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSecurityIRJSON(w, status, securityIRError{Type: code, Message: msg})
}
