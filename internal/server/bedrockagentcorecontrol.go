package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type bedrockAgentCoreControlError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleBedrockAgentCoreControlRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isBedrockAgentCoreControlRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "bedrock-agentcore")
	if !ok {
		respondBedrockAgentCoreControlError(w, status, code, msg)
		return true
	}

	action, pathParams := parseBedrockAgentCoreControlRoute(r.Method, rawRequestPath(r))
	if action == "" {
		// AgentCore data-plane and control-plane share service signing name and
		// overlapping prefixes. Let other routers attempt a match first.
		return false
	}
	if _, known := bedrockAgentCoreControlOperationByName[action]; !known {
		respondBedrockAgentCoreControlError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseBedrockAgentCoreControlPayload(r)
	if err != nil {
		respondBedrockAgentCoreControlError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.bedrockagentcorecontrol.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondBedrockAgentCoreControlJSON(w, http.StatusOK, response)
	return true
}

func isBedrockAgentCoreControlRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodPost, http.MethodGet, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "bedrock-agentcore" {
		return false
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	if path == "" {
		path = "/"
	}
	looksLikeControlRoute := strings.HasPrefix(path, "/runtimes") ||
		strings.HasPrefix(path, "/identities") ||
		strings.HasPrefix(path, "/browsers") ||
		strings.HasPrefix(path, "/browser-profiles") ||
		strings.HasPrefix(path, "/code-interpreters") ||
		strings.HasPrefix(path, "/evaluators") ||
		strings.HasPrefix(path, "/gateways") ||
		strings.HasPrefix(path, "/memories") ||
		strings.HasPrefix(path, "/online-evaluation-configs") ||
		strings.HasPrefix(path, "/policy-engines") ||
		strings.HasPrefix(path, "/resourcepolicy") ||
		strings.HasPrefix(path, "/tags")
	if !looksLikeControlRoute {
		return false
	}

	if service == "bedrock-agentcore" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".bedrock-agentcore.") || strings.HasPrefix(host, "bedrock-agentcore.") || strings.Contains(host, "bedrock-agentcore") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#bedrock-agentcore") || strings.Contains(userAgent, " bedrock-agentcore/") {
		return true
	}

	action, _ := parseBedrockAgentCoreControlRoute(method, rawRequestPath(r))
	return action != ""
}

func parseBedrockAgentCoreControlRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range bedrockAgentCoreControlOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := bedrockAgentCoreControlPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	return "", nil
}

func bedrockAgentCoreControlPathParams(template, actual string) (map[string]string, bool) {
	tSegs := bedrockAgentCoreControlSplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	aSegs := bedrockAgentCoreControlSplitPath(strings.SplitN(strings.TrimSpace(actual), "?", 2)[0])
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

func bedrockAgentCoreControlSplitPath(path string) []string {
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

func parseBedrockAgentCoreControlPayload(r *http.Request) (map[string]any, error) {
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

func respondBedrockAgentCoreControlJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondBedrockAgentCoreControlError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondBedrockAgentCoreControlJSON(w, status, bedrockAgentCoreControlError{Type: code, Message: msg})
}
