package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type resourceExplorer2Error struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleResourceExplorer2RESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isResourceExplorer2RESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "resource-explorer-2")
	if !ok {
		respondResourceExplorer2Error(w, status, code, msg)
		return true
	}

	action, pathParams := parseResourceExplorer2Route(r.Method, rawRequestPath(r))
	if action == "" {
		respondResourceExplorer2Error(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := resourceExplorer2OperationByName[action]; !known {
		respondResourceExplorer2Error(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseResourceExplorer2Payload(r)
	if err != nil {
		respondResourceExplorer2Error(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.resourceexplorer2.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondResourceExplorer2JSON(w, http.StatusOK, response)
	return true
}

func isResourceExplorer2RESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "resource-explorer-2" {
		return false
	}
	if service == "resource-explorer-2" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, "resource-explorer") || strings.Contains(host, "resourceexplorer") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#resource-explorer-2") || strings.Contains(userAgent, " resource-explorer-2/") {
		return true
	}

	action, _ := parseResourceExplorer2Route(method, rawRequestPath(r))
	if action == "" {
		return false
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	if strings.HasPrefix(strings.ToLower(path), "/tags/") {
		return false
	}

	return true
}

func parseResourceExplorer2Route(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range resourceExplorer2Operations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := resourceExplorer2PathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	return "", nil
}

func resourceExplorer2PathParams(template, actual string) (map[string]string, bool) {
	tSegs := resourceExplorer2SplitPath(template)
	aSegs := resourceExplorer2SplitPath(actual)
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

func resourceExplorer2SplitPath(path string) []string {
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

func parseResourceExplorer2Payload(r *http.Request) (map[string]any, error) {
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

func respondResourceExplorer2JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondResourceExplorer2Error(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondResourceExplorer2JSON(w, status, resourceExplorer2Error{Type: code, Message: msg})
}
