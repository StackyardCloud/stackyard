package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type mediaTailorError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleMediaTailorRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isMediaTailorRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "mediatailor")
	if !ok {
		respondMediaTailorError(w, status, code, msg)
		return true
	}

	action, pathParams := parseMediaTailorRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondMediaTailorError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := mediaTailorOperationByName[action]; !known {
		respondMediaTailorError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseMediaTailorPayload(r)
	if err != nil {
		respondMediaTailorError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.mediatailor.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondMediaTailorJSON(w, http.StatusOK, response)
	return true
}

func isMediaTailorRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "mediatailor" {
		return false
	}
	if service == "mediatailor" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".mediatailor.") || strings.HasPrefix(host, "mediatailor.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#mediatailor") || strings.Contains(userAgent, " mediatailor/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	return strings.HasPrefix(path, "/channel/") ||
		strings.HasPrefix(path, "/channels") ||
		strings.HasPrefix(path, "/sourceLocation/") ||
		strings.HasPrefix(path, "/sourceLocations") ||
		strings.HasPrefix(path, "/playbackConfiguration") ||
		strings.HasPrefix(path, "/playbackConfigurations") ||
		strings.HasPrefix(path, "/prefetchSchedule/") ||
		strings.HasPrefix(path, "/alerts") ||
		strings.HasPrefix(path, "/tags/") ||
		strings.HasPrefix(path, "/configureLogs/")
}

func parseMediaTailorRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range mediaTailorOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := mediaTailorPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func mediaTailorPathParams(template, actual string) (map[string]string, bool) {
	tSegs := mediaTailorSplitPath(template)
	aSegs := mediaTailorSplitPath(actual)
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

func mediaTailorSplitPath(path string) []string {
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

func parseMediaTailorPayload(r *http.Request) (map[string]any, error) {
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

func respondMediaTailorJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondMediaTailorError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondMediaTailorJSON(w, status, mediaTailorError{Type: code, Message: msg})
}
