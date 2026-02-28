package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type appConfigError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleAppConfigRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isAppConfigRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "appconfig")
	if !ok {
		respondAppConfigError(w, status, code, msg)
		return true
	}

	action, pathParams := parseAppConfigRoute(r.Method, rawRequestPath(r), r.Header.Get("X-Amz-Target"))
	if action == "" {
		respondAppConfigError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := appConfigOperationByName[action]; !known {
		respondAppConfigError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseAppConfigPayload(r)
	if err != nil {
		respondAppConfigError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.appconfig.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondAppConfigJSON(w, http.StatusOK, response)
	return true
}

func isAppConfigRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "appconfig" {
		return false
	}
	if service == "appconfig" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".appconfig.") || strings.HasPrefix(host, "appconfig.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#appconfig") || strings.Contains(userAgent, " appconfig/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/applications") ||
		strings.HasPrefix(path, "/deploymentstrategies") ||
		strings.HasPrefix(path, "/deployementstrategies") ||
		strings.HasPrefix(path, "/extensions") ||
		strings.HasPrefix(path, "/extensionassociations") ||
		strings.HasPrefix(path, "/settings") ||
		strings.HasPrefix(path, "/tags")
}

func parseAppConfigRoute(method, requestPath, target string) (string, map[string]string) {
	target = strings.TrimSpace(target)
	if target != "" {
		action := parseAppConfigTarget(target)
		if action != "" {
			return action, map[string]string{}
		}
	}

	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range appConfigOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := appConfigPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func parseAppConfigTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AppConfig_20191009.") {
		return strings.TrimPrefix(target, "AppConfig_20191009.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func appConfigPathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	actualPath := strings.TrimSpace(strings.SplitN(actual, "?", 2)[0])
	tSegs := appConfigSplitPathSegments(templatePath)
	aSegs := appConfigSplitPathSegments(actualPath)
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

func appConfigSplitPathSegments(path string) []string {
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

func parseAppConfigPayload(r *http.Request) (map[string]any, error) {
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

func respondAppConfigJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondAppConfigError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondAppConfigJSON(w, status, appConfigError{Type: code, Message: msg})
}
