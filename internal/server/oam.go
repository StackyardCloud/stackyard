package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type oamError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleOAMRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isOAMRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "oam")
	if !ok {
		respondOAMError(w, status, code, msg)
		return true
	}

	action, pathParams := parseOAMRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondOAMError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := oamOperationByName[action]; !known {
		respondOAMError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseOAMPayload(r)
	if err != nil {
		respondOAMError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.oam.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondOAMJSON(w, http.StatusOK, response)
	return true
}

func isOAMRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodPost && method != http.MethodGet && method != http.MethodPut && method != http.MethodDelete {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "oam" {
		return false
	}
	if service == "oam" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".oam.") || strings.HasPrefix(host, "oam.") || strings.Contains(host, "oam") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#oam") || strings.Contains(userAgent, " oam/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/Create") ||
		strings.HasPrefix(path, "/Delete") ||
		strings.HasPrefix(path, "/Get") ||
		strings.HasPrefix(path, "/List") ||
		strings.HasPrefix(path, "/Put") ||
		strings.HasPrefix(path, "/Update") ||
		strings.HasPrefix(path, "/tags/")
}

func parseOAMRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range oamOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := oamPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func oamPathParams(template, actual string) (map[string]string, bool) {
	tSegs := oamSplitPath(template)
	aSegs := oamSplitPath(actual)
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

func oamSplitPath(path string) []string {
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

func parseOAMPayload(r *http.Request) (map[string]any, error) {
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

func respondOAMJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondOAMError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondOAMJSON(w, status, oamError{Type: code, Message: msg})
}
