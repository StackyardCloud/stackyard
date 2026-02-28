package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type trustedAdvisorError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleTrustedAdvisorRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isTrustedAdvisorRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "trustedadvisor")
	if !ok {
		respondTrustedAdvisorError(w, status, code, msg)
		return true
	}

	action, pathParams := parseTrustedAdvisorRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondTrustedAdvisorError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := trustedAdvisorOperationByName[action]; !known {
		respondTrustedAdvisorError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseTrustedAdvisorPayload(r)
	if err != nil {
		respondTrustedAdvisorError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.trustedadvisor.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondTrustedAdvisorJSON(w, http.StatusOK, response)
	return true
}

func isTrustedAdvisorRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodGet && method != http.MethodPut {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "trustedadvisor" {
		return false
	}
	if service == "trustedadvisor" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".trustedadvisor.") || strings.HasPrefix(host, "trustedadvisor.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#trustedadvisor") || strings.Contains(userAgent, " trustedadvisor/") {
		return true
	}

	action, _ := parseTrustedAdvisorRoute(method, rawRequestPath(r))
	return action != ""
}

func parseTrustedAdvisorRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range trustedAdvisorOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := trustedAdvisorPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func trustedAdvisorPathParams(template, actual string) (map[string]string, bool) {
	tSegs := trustedAdvisorSplitPath(template)
	aSegs := trustedAdvisorSplitPath(actual)
	if len(tSegs) != len(aSegs) {
		return nil, false
	}

	params := map[string]string{}
	for i := range tSegs {
		t := tSegs[i]
		a := aSegs[i]
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "{"), "}"))
			if name == "" || strings.TrimSpace(a) == "" {
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

func trustedAdvisorSplitPath(path string) []string {
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

func parseTrustedAdvisorPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
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

func respondTrustedAdvisorJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondTrustedAdvisorError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondTrustedAdvisorJSON(w, status, trustedAdvisorError{Type: code, Message: msg})
}
