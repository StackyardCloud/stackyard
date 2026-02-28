package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type tnbError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleTNBRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isTNBRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "tnb")
	if !ok {
		respondTNBError(w, status, code, msg)
		return true
	}

	action, pathParams := parseTNBRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondTNBError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := tnbOperationByName[action]; !known {
		respondTNBError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseTNBPayload(r)
	if err != nil {
		respondTNBError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.tnb.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondTNBJSON(w, http.StatusOK, response)
	return true
}

func isTNBRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "tnb" {
		return false
	}
	if service == "tnb" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".tnb.") || strings.HasPrefix(host, "tnb.") || strings.Contains(host, "telco") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#tnb") || strings.Contains(userAgent, " tnb/") {
		return true
	}

	path := rawRequestPath(r)
	return strings.HasPrefix(path, "/sol/")
}

func parseTNBRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range tnbOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := tnbPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	return "", nil
}

func tnbPathParams(template, actual string) (map[string]string, bool) {
	tSegs := tnbSplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	aSegs := tnbSplitPath(strings.SplitN(strings.TrimSpace(actual), "?", 2)[0])
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

func tnbSplitPath(path string) []string {
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

func parseTNBPayload(r *http.Request) (map[string]any, error) {
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

func respondTNBJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondTNBError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondTNBJSON(w, status, tnbError{Type: code, Message: msg})
}
