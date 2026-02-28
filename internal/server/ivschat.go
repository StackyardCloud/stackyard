package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type ivsChatError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleIVSChatRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isIVSChatRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "ivs-chat")
	if !ok {
		okAlt, _, _, _, _ := s.validateSigV4WithService(r, "ivschat")
		if !okAlt {
			respondIVSChatError(w, status, code, msg)
			return true
		}
	}

	action, pathParams := parseIVSChatRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondIVSChatError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := ivsChatOperationByName[action]; !known {
		respondIVSChatError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseIVSChatPayload(r)
	if err != nil {
		respondIVSChatError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.ivschat.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondIVSChatJSON(w, http.StatusOK, response)
	return true
}

func isIVSChatRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodPost, http.MethodGet, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "ivs-chat" && service != "ivschat" {
		return false
	}
	if service == "ivs-chat" || service == "ivschat" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".ivs-chat.") ||
		strings.HasPrefix(host, "ivs-chat.") ||
		strings.Contains(host, ".ivschat.") ||
		strings.HasPrefix(host, "ivschat.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#ivs-chat") || strings.Contains(userAgent, " ivs-chat/") {
		return true
	}

	action, _ := parseIVSChatRoute(method, rawRequestPath(r))
	return action != ""
}

func parseIVSChatRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range ivsChatOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := ivsChatPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func ivsChatPathParams(template, actual string) (map[string]string, bool) {
	tSegs := ivsChatSplitPath(template)
	aSegs := ivsChatSplitPath(actual)
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

func ivsChatSplitPath(path string) []string {
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

func parseIVSChatPayload(r *http.Request) (map[string]any, error) {
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

func respondIVSChatJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondIVSChatError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondIVSChatJSON(w, status, ivsChatError{Type: code, Message: msg})
}
