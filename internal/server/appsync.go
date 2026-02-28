package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type appSyncError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleAppSyncRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isAppSyncRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "appsync")
	if !ok {
		respondAppSyncError(w, status, code, msg)
		return true
	}

	action, pathParams := parseAppSyncRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondAppSyncError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := appSyncOperationByName[action]; !known {
		respondAppSyncError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseAppSyncPayload(r)
	if err != nil {
		respondAppSyncError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.appsync.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondAppSyncJSON(w, http.StatusOK, response)
	return true
}

func isAppSyncRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "appsync" {
		return false
	}
	if service == "appsync" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".appsync-api.") || strings.HasPrefix(host, "appsync-api.") || strings.Contains(host, ".appsync.") || strings.HasPrefix(host, "appsync.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#appsync") || strings.Contains(userAgent, " appsync/") {
		return true
	}

	action, _ := parseAppSyncRoute(method, rawRequestPath(r))
	return action != ""
}

func parseAppSyncRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range appSyncOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := appSyncPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func appSyncPathParams(template, actual string) (map[string]string, bool) {
	tSegs := appSyncSplitPath(template)
	aSegs := appSyncSplitPath(actual)
	params := map[string]string{}
	i, j := 0, 0
	for i < len(tSegs) {
		t := tSegs[i]
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "{"), "}"))
			if name == "" {
				return nil, false
			}
			if i == len(tSegs)-1 {
				value, err := url.PathUnescape(strings.Join(aSegs[j:], "/"))
				if err != nil {
					value = strings.Join(aSegs[j:], "/")
				}
				params[name] = value
				i++
				j = len(aSegs)
				break
			}
			if j >= len(aSegs) {
				return nil, false
			}
			value, err := url.PathUnescape(aSegs[j])
			if err != nil {
				value = aSegs[j]
			}
			params[name] = value
			i++
			j++
			continue
		}
		if j >= len(aSegs) || t != aSegs[j] {
			return nil, false
		}
		i++
		j++
	}
	if i != len(tSegs) || j != len(aSegs) {
		return nil, false
	}
	return params, true
}

func appSyncSplitPath(path string) []string {
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

func parseAppSyncPayload(r *http.Request) (map[string]any, error) {
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

func respondAppSyncJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondAppSyncError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondAppSyncJSON(w, status, appSyncError{Type: code, Message: msg})
}
