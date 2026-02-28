package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type appFlowError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleAppFlowRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isAppFlowRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "appflow")
	if !ok {
		respondAppFlowError(w, status, code, msg)
		return true
	}

	action, pathParams := parseAppFlowRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondAppFlowError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := appFlowOperationByName[action]; !known {
		respondAppFlowError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseAppFlowPayload(r)
	if err != nil {
		respondAppFlowError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.appflow.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondAppFlowJSON(w, http.StatusOK, response)
	return true
}

func isAppFlowRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodPost, http.MethodGet, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "appflow" {
		return false
	}
	if service == "appflow" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".appflow.") || strings.HasPrefix(host, "appflow.") || strings.Contains(host, "appflow") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#appflow") || strings.Contains(userAgent, " appflow/") {
		return true
	}

	action, _ := parseAppFlowRoute(method, rawRequestPath(r))
	return action != ""
}

func parseAppFlowRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range appFlowOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := appFlowPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	return "", nil
}

func appFlowPathParams(template, actual string) (map[string]string, bool) {
	tSegs := appFlowSplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	aSegs := appFlowSplitPath(strings.SplitN(strings.TrimSpace(actual), "?", 2)[0])
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

func appFlowSplitPath(path string) []string {
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

func parseAppFlowPayload(r *http.Request) (map[string]any, error) {
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

func respondAppFlowJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondAppFlowError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondAppFlowJSON(w, status, appFlowError{Type: code, Message: msg})
}
