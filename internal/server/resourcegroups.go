package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type resourceGroupsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleResourceGroupsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isResourceGroupsRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "resource-groups")
	if !ok {
		respondResourceGroupsError(w, status, code, msg)
		return true
	}

	action, pathParams := parseResourceGroupsRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondResourceGroupsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := resourceGroupsOperationByName[action]; !known {
		respondResourceGroupsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseResourceGroupsPayload(r)
	if err != nil {
		respondResourceGroupsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.resourcegroups.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondResourceGroupsJSON(w, http.StatusOK, response)
	return true
}

func isResourceGroupsRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "resource-groups" {
		return false
	}
	if service == "resource-groups" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, "resource-groups") || strings.Contains(host, "resourcegroups") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#resource-groups") || strings.Contains(userAgent, " resource-groups/") {
		return true
	}

	action, _ := parseResourceGroupsRoute(method, rawRequestPath(r))
	return action != ""
}

func parseResourceGroupsRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range resourceGroupsOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := resourceGroupsPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	return "", nil
}

func resourceGroupsPathParams(template, actual string) (map[string]string, bool) {
	tSegs := resourceGroupsSplitPath(template)
	aSegs := resourceGroupsSplitPath(actual)
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

func resourceGroupsSplitPath(path string) []string {
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

func parseResourceGroupsPayload(r *http.Request) (map[string]any, error) {
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

func respondResourceGroupsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondResourceGroupsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondResourceGroupsJSON(w, status, resourceGroupsError{Type: code, Message: msg})
}
