package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type groundStationError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleGroundStationRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isGroundStationRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "groundstation")
	if !ok {
		respondGroundStationError(w, status, code, msg)
		return true
	}

	action, pathParams := parseGroundStationRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondGroundStationError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := groundStationOperationByName[action]; !known {
		respondGroundStationError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseGroundStationPayload(r)
	if err != nil {
		respondGroundStationError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.groundstation.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondGroundStationJSON(w, http.StatusOK, response)
	return true
}

func isGroundStationRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "groundstation" {
		return false
	}
	if service == "groundstation" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".groundstation.") || strings.HasPrefix(host, "groundstation.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#groundstation") || strings.Contains(userAgent, " groundstation/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	return strings.HasPrefix(path, "/contact") ||
		strings.HasPrefix(path, "/config") ||
		strings.HasPrefix(path, "/dataflowEndpointGroup") ||
		strings.HasPrefix(path, "/dataflowEndpointGroupV2") ||
		strings.HasPrefix(path, "/ephemeris") ||
		strings.HasPrefix(path, "/missionprofile") ||
		strings.HasPrefix(path, "/agent") ||
		strings.HasPrefix(path, "/agentResponseUrl") ||
		strings.HasPrefix(path, "/minute-usage") ||
		strings.HasPrefix(path, "/contacts") ||
		strings.HasPrefix(path, "/ephemerides") ||
		strings.HasPrefix(path, "/groundstation") ||
		strings.HasPrefix(path, "/satellite") ||
		strings.HasPrefix(path, "/tags")
}

func parseGroundStationRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range groundStationOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := groundStationPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	return "", nil
}

func groundStationPathParams(template, actual string) (map[string]string, bool) {
	tSegs := groundStationSplitPath(template)
	aSegs := groundStationSplitPath(actual)
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

func groundStationSplitPath(path string) []string {
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

func parseGroundStationPayload(r *http.Request) (map[string]any, error) {
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

func respondGroundStationJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondGroundStationError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondGroundStationJSON(w, status, groundStationError{Type: code, Message: msg})
}
