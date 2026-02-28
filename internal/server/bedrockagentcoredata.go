package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type bedrockAgentCoreDataError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleBedrockAgentCoreDataRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isBedrockAgentCoreDataRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "bedrock-agentcore")
	if !ok {
		respondBedrockAgentCoreDataError(w, status, code, msg)
		return true
	}

	action, pathParams := parseBedrockAgentCoreDataRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondBedrockAgentCoreDataError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := bedrockAgentCoreDataOperationByName[action]; !known {
		respondBedrockAgentCoreDataError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseBedrockAgentCoreDataPayload(r)
	if err != nil {
		respondBedrockAgentCoreDataError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.bedrockagentcoredata.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondBedrockAgentCoreDataJSON(w, http.StatusOK, response)
	return true
}

func isBedrockAgentCoreDataRESTCandidate(r *http.Request) bool {
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

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/memories") ||
		strings.HasPrefix(path, "/identities") ||
		strings.HasPrefix(path, "/runtimes") ||
		strings.HasPrefix(path, "/browsers") ||
		strings.HasPrefix(path, "/browser-profiles") ||
		strings.HasPrefix(path, "/code-interpreters") ||
		strings.HasPrefix(path, "/evaluations")
}

func parseBedrockAgentCoreDataRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range bedrockAgentCoreDataOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := bedrockAgentCoreDataPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	return "", nil
}

func bedrockAgentCoreDataPathParams(template, actual string) (map[string]string, bool) {
	tSegs := bedrockAgentCoreDataSplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	aSegs := bedrockAgentCoreDataSplitPath(strings.SplitN(strings.TrimSpace(actual), "?", 2)[0])
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

func bedrockAgentCoreDataSplitPath(path string) []string {
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

func parseBedrockAgentCoreDataPayload(r *http.Request) (map[string]any, error) {
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

func respondBedrockAgentCoreDataJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondBedrockAgentCoreDataError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondBedrockAgentCoreDataJSON(w, status, bedrockAgentCoreDataError{Type: code, Message: msg})
}
