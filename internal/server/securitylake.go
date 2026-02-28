package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type securityLakeError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleSecurityLakeRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSecurityLakeRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "securitylake")
	if !ok {
		respondSecurityLakeError(w, status, code, msg)
		return true
	}

	action, pathParams := parseSecurityLakeRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondSecurityLakeError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := securityLakeOperationByName[action]; !known {
		respondSecurityLakeError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseSecurityLakePayload(r)
	if err != nil {
		respondSecurityLakeError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.securitylake.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondSecurityLakeJSON(w, http.StatusOK, response)
	return true
}

func isSecurityLakeRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "securitylake" && service != "security-lake" {
		return false
	}
	if service == "securitylake" || service == "security-lake" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".securitylake.") || strings.HasPrefix(host, "securitylake.") || strings.Contains(host, ".security-lake.") || strings.HasPrefix(host, "security-lake.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#securitylake.") || strings.Contains(userAgent, " securitylake/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	return strings.HasPrefix(path, "/v1/datalake") ||
		path == "/v1/datalakes" ||
		strings.HasPrefix(path, "/v1/subscribers") ||
		strings.HasPrefix(path, "/v1/tags/")
}

func parseSecurityLakeRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range securityLakeOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := securityLakePathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	return "", nil
}

func securityLakePathParams(template, actual string) (map[string]string, bool) {
	tSegs := securityLakeSplitPath(template)
	aSegs := securityLakeSplitPath(actual)
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

func securityLakeSplitPath(path string) []string {
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

func parseSecurityLakePayload(r *http.Request) (map[string]any, error) {
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

func respondSecurityLakeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSecurityLakeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSecurityLakeJSON(w, status, securityLakeError{Type: code, Message: msg})
}
