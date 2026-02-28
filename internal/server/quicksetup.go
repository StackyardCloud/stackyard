package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type quickSetupError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleQuickSetupRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isQuickSetupRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "ssm-quicksetup")
	if !ok {
		okAlt, _, _, _, _ := s.validateSigV4WithService(r, "quicksetup")
		if !okAlt {
			respondQuickSetupError(w, status, code, msg)
			return true
		}
	}

	action, pathParams := parseQuickSetupRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondQuickSetupError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := quickSetupOperationByName[action]; !known {
		respondQuickSetupError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseQuickSetupPayload(r)
	if err != nil {
		respondQuickSetupError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.quicksetup.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondQuickSetupJSON(w, http.StatusOK, response)
	return true
}

func isQuickSetupRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodPost && method != http.MethodGet && method != http.MethodDelete && method != http.MethodPut {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "ssm-quicksetup" && service != "quicksetup" {
		return false
	}
	if service == "ssm-quicksetup" || service == "quicksetup" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, "quicksetup") || strings.Contains(host, "ssm-quicksetup") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#ssm-quicksetup") || strings.Contains(userAgent, " ssm-quicksetup/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/configurationManager") ||
		strings.HasPrefix(path, "/getConfiguration") ||
		strings.HasPrefix(path, "/serviceSettings") ||
		strings.HasPrefix(path, "/listConfigurationManagers") ||
		strings.HasPrefix(path, "/listConfigurations") ||
		strings.HasPrefix(path, "/listQuickSetupTypes") ||
		strings.HasPrefix(path, "/tags/") ||
		strings.HasPrefix(path, "/configurationDefinition/")
}

func parseQuickSetupRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range quickSetupOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := quickSetupPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func quickSetupPathParams(template, actual string) (map[string]string, bool) {
	tSegs := quickSetupSplitPath(template)
	aSegs := quickSetupSplitPath(actual)
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

func quickSetupSplitPath(path string) []string {
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

func parseQuickSetupPayload(r *http.Request) (map[string]any, error) {
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

func respondQuickSetupJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondQuickSetupError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondQuickSetupJSON(w, status, quickSetupError{Type: code, Message: msg})
}
