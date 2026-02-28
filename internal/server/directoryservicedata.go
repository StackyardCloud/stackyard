package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type directoryServiceDataError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleDirectoryServiceDataRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isDirectoryServiceDataRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "ds-data")
	if !ok {
		respondDirectoryServiceDataError(w, status, code, msg)
		return true
	}

	action, pathParams := parseDirectoryServiceDataRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondDirectoryServiceDataError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := directoryServiceDataOperationByName[action]; !known {
		respondDirectoryServiceDataError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseDirectoryServiceDataPayload(r)
	if err != nil {
		respondDirectoryServiceDataError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.directoryservicedata.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondDirectoryServiceDataJSON(w, http.StatusOK, response)
	return true
}

func isDirectoryServiceDataRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodPost {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "ds-data" && service != "directoryservicedata" {
		return false
	}
	if service == "ds-data" || service == "directoryservicedata" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".ds-data.") || strings.HasPrefix(host, "ds-data.") || strings.Contains(host, "directoryservicedata") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#ds-data") ||
		strings.Contains(userAgent, " ds-data/") ||
		strings.Contains(userAgent, " directoryservicedata/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	return strings.HasPrefix(path, "/Users/") ||
		strings.HasPrefix(path, "/Groups/") ||
		strings.HasPrefix(path, "/GroupMemberships/")
}

func parseDirectoryServiceDataRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range directoryServiceDataOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := directoryServiceDataPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func directoryServiceDataPathParams(template, actual string) (map[string]string, bool) {
	tSegs := directoryServiceDataSplitPath(template)
	aSegs := directoryServiceDataSplitPath(actual)
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

func directoryServiceDataSplitPath(path string) []string {
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

func parseDirectoryServiceDataPayload(r *http.Request) (map[string]any, error) {
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

func respondDirectoryServiceDataJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondDirectoryServiceDataError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondDirectoryServiceDataJSON(w, status, directoryServiceDataError{Type: code, Message: msg})
}
