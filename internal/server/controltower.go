package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type controlTowerError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleControlTowerRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isControlTowerRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "controltower")
	if !ok {
		respondControlTowerError(w, status, code, msg)
		return true
	}

	action, pathParams := parseControlTowerRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondControlTowerError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := controlTowerOperationByName[action]; !known {
		respondControlTowerError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseControlTowerPayload(r)
	if err != nil {
		respondControlTowerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.controltower.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondControlTowerJSON(w, http.StatusOK, response)
	return true
}

func isControlTowerRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodPost, http.MethodGet, http.MethodDelete, http.MethodPatch, http.MethodPut:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "controltower" {
		return false
	}
	if service == "controltower" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".controltower.") || strings.HasPrefix(host, "controltower.") || strings.Contains(host, "controltower") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#controltower") || strings.Contains(userAgent, " controltower/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/create-") ||
		strings.HasPrefix(path, "/delete-") ||
		strings.HasPrefix(path, "/disable-") ||
		strings.HasPrefix(path, "/enable-") ||
		strings.HasPrefix(path, "/get-") ||
		strings.HasPrefix(path, "/list-") ||
		strings.HasPrefix(path, "/reset-") ||
		strings.HasPrefix(path, "/update-") ||
		strings.HasPrefix(path, "/tags/")
}

func parseControlTowerRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range controlTowerOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := controlTowerPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	// Fallback for unknown static route variations.
	if action := controlTowerActionFromPath(method, path); action != "" {
		return action, map[string]string{}
	}

	return "", nil
}

func controlTowerActionFromPath(method, path string) string {
	trimmed := strings.Trim(strings.SplitN(path, "?", 2)[0], "/")
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "tags/") {
		switch strings.ToUpper(strings.TrimSpace(method)) {
		case http.MethodGet:
			return "ListTagsForResource"
		case http.MethodPost:
			return "TagResource"
		case http.MethodDelete:
			return "UntagResource"
		default:
			return ""
		}
	}

	parts := strings.Split(trimmed, "/")
	if len(parts) != 1 {
		return ""
	}
	candidate := controlTowerKebabToPascal(parts[0])
	norm := controlTowerNormalizeAction(candidate)
	return controlTowerOperationByNormalizedName[norm]
}

func controlTowerPathParams(template, actual string) (map[string]string, bool) {
	tSegs := controlTowerSplitPath(template)
	aSegs := controlTowerSplitPath(actual)
	if len(tSegs) != len(aSegs) {
		return nil, false
	}

	params := map[string]string{}
	for i := range tSegs {
		t := tSegs[i]
		a := aSegs[i]
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "{"), "}"))
			greedy := strings.HasSuffix(name, "+")
			if greedy {
				name = strings.TrimSuffix(name, "+")
			}
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

func controlTowerSplitPath(path string) []string {
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

func parseControlTowerPayload(r *http.Request) (map[string]any, error) {
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

func respondControlTowerJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondControlTowerError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondControlTowerJSON(w, status, controlTowerError{Type: code, Message: msg})
}

var controlTowerActionNormalizeRegex = regexp.MustCompile(`[^a-z0-9]+`)

func controlTowerNormalizeAction(action string) string {
	return controlTowerActionNormalizeRegex.ReplaceAllString(strings.ToLower(strings.TrimSpace(action)), "")
}

func controlTowerKebabToPascal(name string) string {
	parts := strings.Split(strings.TrimSpace(name), "-")
	out := strings.Builder{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if len(part) == 1 {
			out.WriteString(strings.ToUpper(part))
			continue
		}
		out.WriteString(strings.ToUpper(part[:1]))
		out.WriteString(part[1:])
	}
	return out.String()
}

var controlTowerOperationByNormalizedName = func() map[string]string {
	out := make(map[string]string, len(controlTowerOperations))
	for _, op := range controlTowerOperations {
		out[controlTowerNormalizeAction(op.Name)] = op.Name
	}
	return out
}()
