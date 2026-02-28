package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type iotEventsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleIoTEventsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isIoTEventsRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "iotevents")
	if !ok {
		respondIoTEventsError(w, status, code, msg)
		return true
	}

	action, pathParams := parseIoTEventsRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondIoTEventsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := iotEventsOperationByName[action]; !known {
		respondIoTEventsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseIoTEventsPayload(r)
	if err != nil {
		respondIoTEventsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.iotevents.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondIoTEventsJSON(w, http.StatusOK, response)
	return true
}

func isIoTEventsRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "iotevents" {
		return false
	}
	if service == "iotevents" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".iotevents.") || strings.HasPrefix(host, "iotevents.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#iotevents") || strings.Contains(userAgent, " iotevents/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/alarm-models") ||
		strings.HasPrefix(path, "/detector-models") ||
		strings.HasPrefix(path, "/analysis/") ||
		strings.HasPrefix(path, "/inputs") ||
		strings.HasPrefix(path, "/input-routings") ||
		strings.HasPrefix(path, "/logging") ||
		strings.HasPrefix(path, "/tags")
}

func parseIoTEventsAction(method, requestPath string) string {
	action, _ := parseIoTEventsRoute(method, requestPath)
	return action
}

func parseIoTEventsRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(requestPath)
	if path == "" {
		path = "/"
	}
	for _, op := range iotEventsOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := iotEventsPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func iotEventsPathMatches(template, actual string) bool {
	_, ok := iotEventsPathParams(template, actual)
	return ok
}

func iotEventsPathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	tSegs := iotEventsSplitPathSegments(templatePath)
	aSegs := iotEventsSplitPathSegments(actual)
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

func iotEventsSplitPathSegments(path string) []string {
	path = strings.TrimSpace(path)
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

func parseIoTEventsPayload(r *http.Request) (map[string]any, error) {
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

func respondIoTEventsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondIoTEventsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondIoTEventsJSON(w, status, iotEventsError{Type: code, Message: msg})
}
