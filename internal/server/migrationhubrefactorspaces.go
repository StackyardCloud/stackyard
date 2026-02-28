package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type migrationHubRefactorSpacesError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleMigrationHubRefactorSpacesRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isMigrationHubRefactorSpacesRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "refactor-spaces")
	if !ok {
		respondMigrationHubRefactorSpacesError(w, status, code, msg)
		return true
	}

	action, pathParams := parseMigrationHubRefactorSpacesRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondMigrationHubRefactorSpacesError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := migrationHubRefactorSpacesOperationByName[action]; !known {
		respondMigrationHubRefactorSpacesError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseMigrationHubRefactorSpacesPayload(r)
	if err != nil {
		respondMigrationHubRefactorSpacesError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.migrationhubrefactorspaces.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondMigrationHubRefactorSpacesJSON(w, http.StatusOK, response)
	return true
}

func isMigrationHubRefactorSpacesRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "refactor-spaces" && service != "migrationhub-refactor-spaces" {
		return false
	}
	if service == "refactor-spaces" || service == "migrationhub-refactor-spaces" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".refactor-spaces.") || strings.HasPrefix(host, "refactor-spaces.") || strings.Contains(host, "migrationhub-refactor-spaces") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#migrationhub-refactor-spaces") ||
		strings.Contains(userAgent, " migrationhub-refactor-spaces/") ||
		strings.Contains(userAgent, " refactor-spaces/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	return strings.HasPrefix(path, "/environments") ||
		strings.HasPrefix(path, "/resourcepolicy") ||
		strings.HasPrefix(path, "/tags") ||
		strings.HasPrefix(path, "/applications") ||
		strings.HasPrefix(path, "/services") ||
		strings.HasPrefix(path, "/routes")
}

func parseMigrationHubRefactorSpacesRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range migrationHubRefactorSpacesOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := migrationHubRefactorSpacesPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func migrationHubRefactorSpacesPathParams(template, actual string) (map[string]string, bool) {
	tSegs := migrationHubRefactorSpacesSplitPath(template)
	aSegs := migrationHubRefactorSpacesSplitPath(actual)
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

func migrationHubRefactorSpacesSplitPath(path string) []string {
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

func parseMigrationHubRefactorSpacesPayload(r *http.Request) (map[string]any, error) {
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

func respondMigrationHubRefactorSpacesJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondMigrationHubRefactorSpacesError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondMigrationHubRefactorSpacesJSON(w, status, migrationHubRefactorSpacesError{
		Type:    code,
		Message: msg,
	})
}
