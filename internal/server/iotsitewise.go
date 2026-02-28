package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type iotSiteWiseError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleIoTSiteWiseRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isIoTSiteWiseRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "iotsitewise")
	if !ok {
		respondIoTSiteWiseError(w, status, code, msg)
		return true
	}

	action, pathParams := parseIoTSiteWiseRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondIoTSiteWiseError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := iotSiteWiseOperationByName[action]; !known {
		respondIoTSiteWiseError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseIoTSiteWisePayload(r)
	if err != nil {
		respondIoTSiteWiseError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.iotsitewise.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondIoTSiteWiseJSON(w, http.StatusOK, response)
	return true
}

func isIoTSiteWiseRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "iotsitewise" {
		return false
	}
	if service == "iotsitewise" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".iotsitewise.") || strings.HasPrefix(host, "iotsitewise.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#iotsitewise") || strings.Contains(userAgent, " iotsitewise/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/assets") ||
		strings.HasPrefix(path, "/asset-models") ||
		strings.HasPrefix(path, "/projects") ||
		strings.HasPrefix(path, "/portals") ||
		strings.HasPrefix(path, "/dashboards") ||
		strings.HasPrefix(path, "/datasets") ||
		strings.HasPrefix(path, "/jobs") ||
		strings.HasPrefix(path, "/actions") ||
		strings.HasPrefix(path, "/assistant") ||
		strings.HasPrefix(path, "/queries") ||
		strings.HasPrefix(path, "/timeseries") ||
		strings.HasPrefix(path, "/properties") ||
		strings.HasPrefix(path, "/logging") ||
		strings.HasPrefix(path, "/configuration") ||
		strings.HasPrefix(path, "/executions") ||
		strings.HasPrefix(path, "/computation-models") ||
		strings.HasPrefix(path, "/access-policies") ||
		strings.HasPrefix(path, "/tags") ||
		strings.HasPrefix(path, "/20200301/gateways") ||
		strings.HasPrefix(path, "/interface/")
}

func parseIoTSiteWiseAction(method, requestPath string) string {
	action, _ := parseIoTSiteWiseRoute(method, requestPath)
	return action
}

func parseIoTSiteWiseRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(requestPath)
	if path == "" {
		path = "/"
	}
	for _, op := range iotSiteWiseOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := iotSiteWisePathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func iotSiteWisePathMatches(template, actual string) bool {
	_, ok := iotSiteWisePathParams(template, actual)
	return ok
}

func iotSiteWisePathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	tSegs := iotSiteWiseSplitPathSegments(templatePath)
	aSegs := iotSiteWiseSplitPathSegments(actual)
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

func iotSiteWiseSplitPathSegments(path string) []string {
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

func parseIoTSiteWisePayload(r *http.Request) (map[string]any, error) {
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

func respondIoTSiteWiseJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondIoTSiteWiseError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondIoTSiteWiseJSON(w, status, iotSiteWiseError{Type: code, Message: msg})
}
