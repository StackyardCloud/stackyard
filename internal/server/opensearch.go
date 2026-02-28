package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type opensearchError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleOpenSearchRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isOpenSearchRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "es")
	if !ok {
		respondOpenSearchError(w, status, code, msg)
		return true
	}

	action, pathParams := parseOpenSearchRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondOpenSearchError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := opensearchOperationByName[action]; !known {
		respondOpenSearchError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseOpenSearchPayload(r)
	if err != nil {
		respondOpenSearchError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.opensearch.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondOpenSearchJSON(w, http.StatusOK, response)
	return true
}

func isOpenSearchRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "es" && service != "opensearch" {
		return false
	}
	if service == "es" || service == "opensearch" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".es.") || strings.HasPrefix(host, "es.") || strings.Contains(host, "opensearch") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#opensearch") || strings.Contains(userAgent, " opensearch/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	if strings.HasPrefix(path, "/2021-01-01") {
		return true
	}

	action, _ := parseOpenSearchRoute(method, path)
	return action != ""
}

func parseOpenSearchRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range opensearchOperations {
		if op.URI == "" || op.Method == "" {
			continue
		}
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := opensearchPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	return "", nil
}

func opensearchPathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	tSegs := opensearchSplitPathSegments(templatePath)
	aSegs := opensearchSplitPathSegments(actual)
	if len(tSegs) != len(aSegs) {
		return nil, false
	}

	params := map[string]string{}
	for i := range tSegs {
		t := tSegs[i]
		a := aSegs[i]
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			if strings.TrimSpace(a) == "" {
				return nil, false
			}
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

func opensearchSplitPathSegments(path string) []string {
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

func parseOpenSearchPayload(r *http.Request) (map[string]any, error) {
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

func respondOpenSearchJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondOpenSearchError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondOpenSearchJSON(w, status, opensearchError{Type: code, Message: msg})
}
