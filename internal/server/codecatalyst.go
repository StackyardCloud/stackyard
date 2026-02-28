package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type codeCatalystError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCodeCatalystRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCodeCatalystRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "codecatalyst")
	if !ok {
		respondCodeCatalystError(w, status, code, msg)
		return true
	}

	action, pathParams := parseCodeCatalystRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondCodeCatalystError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := codeCatalystOperationByName[action]; !known {
		respondCodeCatalystError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseCodeCatalystPayload(r)
	if err != nil {
		respondCodeCatalystError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.codecatalyst.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondCodeCatalystJSON(w, http.StatusOK, response)
	return true
}

func isCodeCatalystRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "codecatalyst" {
		return false
	}
	if service == "codecatalyst" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".codecatalyst.") || strings.HasPrefix(host, "codecatalyst.") || strings.Contains(host, "codecatalyst") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#codecatalyst") || strings.Contains(userAgent, " codecatalyst/") {
		return true
	}

	action, _ := parseCodeCatalystRoute(method, rawRequestPath(r))
	return action != ""
}

func parseCodeCatalystRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range codeCatalystOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := codeCatalystPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	return "", nil
}

func codeCatalystPathParams(template, actual string) (map[string]string, bool) {
	tSegs := codeCatalystSplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	aSegs := codeCatalystSplitPath(strings.SplitN(strings.TrimSpace(actual), "?", 2)[0])
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

func codeCatalystSplitPath(path string) []string {
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

func parseCodeCatalystPayload(r *http.Request) (map[string]any, error) {
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

func respondCodeCatalystJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCodeCatalystError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCodeCatalystJSON(w, status, codeCatalystError{Type: code, Message: msg})
}
