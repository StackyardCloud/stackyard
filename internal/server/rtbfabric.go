package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type rtbFabricError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleRTBFabricRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isRTBFabricRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "rtb-fabric")
	if !ok {
		respondRTBFabricError(w, status, code, msg)
		return true
	}

	action, pathParams := parseRTBFabricRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondRTBFabricError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := rtbFabricOperationByName[action]; !known {
		respondRTBFabricError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseRTBFabricPayload(r)
	if err != nil {
		respondRTBFabricError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.rtbfabric.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondRTBFabricJSON(w, http.StatusOK, response)
	return true
}

func isRTBFabricRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "rtb-fabric" && service != "rtbfabric" {
		return false
	}
	if service == "rtb-fabric" || service == "rtbfabric" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, "rtb-fabric") || strings.Contains(host, "rtbfabric") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#rtbfabric") || strings.Contains(userAgent, "command#rtb-fabric") || strings.Contains(userAgent, " rtbfabric/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	return strings.HasPrefix(path, "/gateway/") ||
		strings.HasPrefix(path, "/requester-gateway") ||
		strings.HasPrefix(path, "/responder-gateway") ||
		strings.HasPrefix(path, "/requester-gateways") ||
		strings.HasPrefix(path, "/responder-gateways") ||
		strings.HasPrefix(path, "/tags/")
}

func parseRTBFabricRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range rtbFabricOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := rtbFabricPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func rtbFabricPathParams(template, actual string) (map[string]string, bool) {
	tSegs := rtbFabricSplitPath(strings.SplitN(template, "?", 2)[0])
	aSegs := rtbFabricSplitPath(actual)
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

func rtbFabricSplitPath(path string) []string {
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

func parseRTBFabricPayload(r *http.Request) (map[string]any, error) {
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

func respondRTBFabricJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondRTBFabricError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondRTBFabricJSON(w, status, rtbFabricError{Type: code, Message: msg})
}
