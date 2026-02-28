package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type gameliftStreamsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleGameLiftStreamsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isGameLiftStreamsRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "gameliftstreams")
	if !ok {
		respondGameLiftStreamsError(w, status, code, msg)
		return true
	}

	action, pathParams := parseGameLiftStreamsRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondGameLiftStreamsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := gameliftStreamsOperationByName[action]; !known {
		respondGameLiftStreamsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseGameLiftStreamsPayload(r)
	if err != nil {
		respondGameLiftStreamsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.gameliftstreams.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondGameLiftStreamsJSON(w, http.StatusOK, response)
	return true
}

func isGameLiftStreamsRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "gameliftstreams" {
		return false
	}
	if service == "gameliftstreams" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".gameliftstreams.") || strings.HasPrefix(host, "gameliftstreams.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#gameliftstreams") || strings.Contains(userAgent, " gameliftstreams/") {
		return true
	}

	action, _ := parseGameLiftStreamsRoute(method, rawRequestPath(r))
	return action != ""
}

func parseGameLiftStreamsRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range gameliftStreamsOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := gameliftStreamsPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func gameliftStreamsPathParams(template, actual string) (map[string]string, bool) {
	tSegs := gameliftStreamsSplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	aSegs := gameliftStreamsSplitPath(strings.SplitN(strings.TrimSpace(actual), "?", 2)[0])
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

func gameliftStreamsSplitPath(path string) []string {
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

func parseGameLiftStreamsPayload(r *http.Request) (map[string]any, error) {
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

func respondGameLiftStreamsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondGameLiftStreamsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondGameLiftStreamsJSON(w, status, gameliftStreamsError{Type: code, Message: msg})
}
