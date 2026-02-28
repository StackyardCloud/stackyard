package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type mediaPackageError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleMediaPackageRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isMediaPackageRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "mediapackagev2")
	if !ok {
		// Some clients still sign with mediapackage for v2 routes.
		ok, status, code, msg, _ = s.validateSigV4WithService(r, "mediapackage")
		if !ok {
			respondMediaPackageError(w, status, code, msg)
			return true
		}
	}

	action, pathParams := parseMediaPackageRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondMediaPackageError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := mediaPackageOperationByName[action]; !known {
		respondMediaPackageError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseMediaPackagePayload(r)
	if err != nil {
		respondMediaPackageError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.mediapackage.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondMediaPackageJSON(w, http.StatusOK, response)
	return true
}

func isMediaPackageRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "mediapackagev2" && service != "mediapackage" {
		return false
	}
	if service == "mediapackagev2" || service == "mediapackage" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".mediapackagev2.") ||
		strings.HasPrefix(host, "mediapackagev2.") ||
		strings.Contains(host, ".mediapackage.") ||
		strings.HasPrefix(host, "mediapackage.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#mediapackagev2") || strings.Contains(userAgent, " mediapackagev2/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	return strings.HasPrefix(path, "/channelGroup") || strings.HasPrefix(path, "/tags")
}

func parseMediaPackageRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range mediaPackageOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := mediaPackagePathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func mediaPackagePathParams(template, actual string) (map[string]string, bool) {
	tSegs := mediaPackageSplitPath(template)
	aSegs := mediaPackageSplitPath(actual)
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

func mediaPackageSplitPath(path string) []string {
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

func parseMediaPackagePayload(r *http.Request) (map[string]any, error) {
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

func respondMediaPackageJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondMediaPackageError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondMediaPackageJSON(w, status, mediaPackageError{Type: code, Message: msg})
}
