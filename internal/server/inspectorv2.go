package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type inspectorV2Error struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleInspectorV2RESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isInspectorV2RESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "inspector2")
	if !ok {
		respondInspectorV2Error(w, status, code, msg)
		return true
	}

	action, pathParams := parseInspectorV2Route(r.Method, rawRequestPath(r))
	if action == "" {
		respondInspectorV2Error(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := inspectorV2OperationByName[action]; !known {
		respondInspectorV2Error(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseInspectorV2Payload(r)
	if err != nil {
		respondInspectorV2Error(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.inspectorv2.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondInspectorV2JSON(w, http.StatusOK, response)
	return true
}

func isInspectorV2RESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "inspector2" {
		return false
	}
	if service == "inspector2" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".inspector2.") || strings.HasPrefix(host, "inspector2.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#inspector2") || strings.Contains(userAgent, " inspector2/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	action, _ := parseInspectorV2Route(method, path)
	return action != ""
}

func parseInspectorV2Route(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range inspectorV2Operations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := inspectorV2PathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func inspectorV2PathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	actualPath := strings.TrimSpace(strings.SplitN(actual, "?", 2)[0])

	tSegs := inspectorV2SplitPath(templatePath)
	aSegs := inspectorV2SplitPath(actualPath)
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

func inspectorV2SplitPath(path string) []string {
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

func parseInspectorV2Payload(r *http.Request) (map[string]any, error) {
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

func respondInspectorV2JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondInspectorV2Error(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondInspectorV2JSON(w, status, inspectorV2Error{Type: code, Message: msg})
}
