package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type amplifyAdminUIError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleAmplifyAdminUIRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isAmplifyAdminUIRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "amplifybackend")
	if !ok {
		respondAmplifyAdminUIError(w, status, code, msg)
		return true
	}

	action, pathParams := parseAmplifyAdminUIRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondAmplifyAdminUIError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := amplifyAdminUIOperationByName[action]; !known {
		respondAmplifyAdminUIError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseAmplifyAdminUIPayload(r)
	if err != nil {
		respondAmplifyAdminUIError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.amplifyadminui.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondAmplifyAdminUIJSON(w, http.StatusOK, response)
	return true
}

func isAmplifyAdminUIRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodPost, http.MethodGet:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "amplifybackend" {
		return false
	}
	if service == "amplifybackend" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".amplifybackend.") || strings.HasPrefix(host, "amplifybackend.") || strings.Contains(host, "amplifybackend") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#amplifybackend") || strings.Contains(userAgent, " amplifybackend/") {
		return true
	}

	action, _ := parseAmplifyAdminUIRoute(method, rawRequestPath(r))
	return action != ""
}

func parseAmplifyAdminUIRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range amplifyAdminUIOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := amplifyAdminUIPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	return "", nil
}

func amplifyAdminUIPathParams(template, actual string) (map[string]string, bool) {
	tSegs := amplifyAdminUISplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	aSegs := amplifyAdminUISplitPath(strings.SplitN(strings.TrimSpace(actual), "?", 2)[0])
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

func amplifyAdminUISplitPath(path string) []string {
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

func parseAmplifyAdminUIPayload(r *http.Request) (map[string]any, error) {
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

func respondAmplifyAdminUIJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondAmplifyAdminUIError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondAmplifyAdminUIJSON(w, status, amplifyAdminUIError{Type: code, Message: msg})
}
