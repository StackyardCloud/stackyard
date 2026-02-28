package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type migrationHubOrchestratorError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleMigrationHubOrchestratorRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isMigrationHubOrchestratorRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "migrationhub-orchestrator")
	if !ok {
		respondMigrationHubOrchestratorError(w, status, code, msg)
		return true
	}

	action, pathParams := parseMigrationHubOrchestratorRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondMigrationHubOrchestratorError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := migrationHubOrchestratorOperationByName[action]; !known {
		respondMigrationHubOrchestratorError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseMigrationHubOrchestratorPayload(r)
	if err != nil {
		respondMigrationHubOrchestratorError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.migrationhuborchestrator.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondMigrationHubOrchestratorJSON(w, http.StatusOK, response)
	return true
}

func isMigrationHubOrchestratorRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "migrationhub-orchestrator" && service != "migrationhuborchestrator" {
		return false
	}
	if service == "migrationhub-orchestrator" || service == "migrationhuborchestrator" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".migrationhub-orchestrator.") || strings.HasPrefix(host, "migrationhub-orchestrator.") || strings.Contains(host, "migrationhuborchestrator") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#migrationhuborchestrator") ||
		strings.Contains(userAgent, " migrationhuborchestrator/") ||
		strings.Contains(userAgent, " migrationhub-orchestrator/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	return strings.HasPrefix(path, "/migrationworkflow") ||
		strings.HasPrefix(path, "/migrationworkflows") ||
		strings.HasPrefix(path, "/migrationworkflowtemplate") ||
		strings.HasPrefix(path, "/migrationworkflowtemplates") ||
		strings.HasPrefix(path, "/workflow") ||
		strings.HasPrefix(path, "/workflowstep") ||
		strings.HasPrefix(path, "/workflowstepgroup") ||
		strings.HasPrefix(path, "/workflowstepgroups") ||
		strings.HasPrefix(path, "/plugins") ||
		strings.HasPrefix(path, "/template") ||
		strings.HasPrefix(path, "/templates") ||
		strings.HasPrefix(path, "/templatestep") ||
		strings.HasPrefix(path, "/templatesteps") ||
		strings.HasPrefix(path, "/templatestepgroups") ||
		strings.HasPrefix(path, "/retryworkflowstep") ||
		strings.HasPrefix(path, "/tags")
}

func parseMigrationHubOrchestratorRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range migrationHubOrchestratorOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := migrationHubOrchestratorPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func migrationHubOrchestratorPathParams(template, actual string) (map[string]string, bool) {
	tSegs := migrationHubOrchestratorSplitPath(template)
	aSegs := migrationHubOrchestratorSplitPath(actual)
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

func migrationHubOrchestratorSplitPath(path string) []string {
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

func parseMigrationHubOrchestratorPayload(r *http.Request) (map[string]any, error) {
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

func respondMigrationHubOrchestratorJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondMigrationHubOrchestratorError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondMigrationHubOrchestratorJSON(w, status, migrationHubOrchestratorError{Type: code, Message: msg})
}
