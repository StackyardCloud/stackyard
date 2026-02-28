package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type launchWizardError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleLaunchWizardRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isLaunchWizardRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "launchwizard")
	if !ok {
		respondLaunchWizardError(w, status, code, msg)
		return true
	}

	action := parseLaunchWizardRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondLaunchWizardError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := launchWizardOperationByName[action]; !known {
		respondLaunchWizardError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseLaunchWizardPayload(r)
	if err != nil {
		respondLaunchWizardError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.launchwizard.Handle(action, payload, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondLaunchWizardJSON(w, http.StatusOK, response)
	return true
}

func isLaunchWizardRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "launchwizard" {
		return false
	}
	if service == "launchwizard" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".launchwizard.") || strings.HasPrefix(host, "launchwizard.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#launch-wizard") || strings.Contains(userAgent, " launch-wizard/") {
		return true
	}

	action := parseLaunchWizardRoute(method, rawRequestPath(r))
	return action != ""
}

func parseLaunchWizardRoute(method, requestPath string) string {
	path := canonicalLaunchWizardPath(strings.SplitN(strings.TrimSpace(requestPath), "?", 2)[0])
	for _, op := range launchWizardOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		if canonicalLaunchWizardPath(op.URI) == path {
			return op.Name
		}
	}
	return ""
}

func canonicalLaunchWizardPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path == "/tags" || path == "/tags/" {
		return "/tags/"
	}
	if len(path) > 1 {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}

func parseLaunchWizardPayload(r *http.Request) (map[string]any, error) {
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

func respondLaunchWizardJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondLaunchWizardError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondLaunchWizardJSON(w, status, launchWizardError{Type: code, Message: msg})
}

func launchWizardLookupString(pathParams map[string]string, payload map[string]any, query url.Values, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(pathParams[key]); v != "" {
			return v
		}
		if payload != nil {
			if raw, ok := payload[key]; ok {
				if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
					return strings.TrimSpace(s)
				}
			}
		}
		if query != nil {
			for _, candidate := range []string{key, lowerFirst(key), upperFirst(key)} {
				if v := strings.TrimSpace(query.Get(candidate)); v != "" {
					return v
				}
			}
		}
	}
	return ""
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
