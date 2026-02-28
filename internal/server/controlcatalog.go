package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type controlCatalogError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleControlCatalogRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isControlCatalogRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "controlcatalog")
	if !ok {
		respondControlCatalogError(w, status, code, msg)
		return true
	}

	action, pathParams := parseControlCatalogRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondControlCatalogError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := controlCatalogOperationByName[action]; !known {
		respondControlCatalogError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseControlCatalogPayload(r)
	if err != nil {
		respondControlCatalogError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.controlcatalog.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondControlCatalogJSON(w, http.StatusOK, response)
	return true
}

func isControlCatalogRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodPost {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "controlcatalog" {
		return false
	}
	if service == "controlcatalog" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".controlcatalog.") || strings.HasPrefix(host, "controlcatalog.") || strings.Contains(host, "controlcatalog") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#controlcatalog") || strings.Contains(userAgent, " controlcatalog/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/get-control") ||
		strings.HasPrefix(path, "/common-controls") ||
		strings.HasPrefix(path, "/list-control-mappings") ||
		strings.HasPrefix(path, "/list-controls") ||
		strings.HasPrefix(path, "/domains") ||
		strings.HasPrefix(path, "/objectives")
}

func parseControlCatalogRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range controlCatalogOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := controlCatalogPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	return "", nil
}

func controlCatalogPathParams(template, actual string) (map[string]string, bool) {
	tSegs := controlCatalogSplitPath(template)
	aSegs := controlCatalogSplitPath(actual)
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

func controlCatalogSplitPath(path string) []string {
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

func parseControlCatalogPayload(r *http.Request) (map[string]any, error) {
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

func respondControlCatalogJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondControlCatalogError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondControlCatalogJSON(w, status, controlCatalogError{Type: code, Message: msg})
}
