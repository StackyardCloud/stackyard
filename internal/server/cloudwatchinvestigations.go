package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type cloudWatchInvestigationsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCloudWatchInvestigationsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCloudWatchInvestigationsRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "aiops")
	if !ok {
		respondCloudWatchInvestigationsError(w, status, code, msg)
		return true
	}

	action, pathParams := parseCloudWatchInvestigationsRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondCloudWatchInvestigationsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := cloudWatchInvestigationsOperationByName[action]; !known {
		respondCloudWatchInvestigationsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseCloudWatchInvestigationsPayload(r)
	if err != nil {
		respondCloudWatchInvestigationsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.cloudwatchinvestigations.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondCloudWatchInvestigationsJSON(w, http.StatusOK, response)
	return true
}

func isCloudWatchInvestigationsRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "aiops" {
		return false
	}
	if service == "aiops" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".aiops.") || strings.HasPrefix(host, "aiops.") {
		return true
	}
	if strings.Contains(host, "cloudwatchinvestigations") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#aiops") || strings.Contains(userAgent, " aiops/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/investigationGroups") || strings.HasPrefix(path, "/tags")
}

func parseCloudWatchInvestigationsRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range cloudWatchInvestigationsOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := cloudWatchInvestigationsPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func cloudWatchInvestigationsPathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	actualPath := strings.TrimSpace(strings.SplitN(actual, "?", 2)[0])
	tSegs := cloudWatchInvestigationsSplitPathSegments(templatePath)
	aSegs := cloudWatchInvestigationsSplitPathSegments(actualPath)
	if len(tSegs) != len(aSegs) {
		return nil, false
	}

	params := map[string]string{}
	for i := range tSegs {
		t := tSegs[i]
		a := aSegs[i]
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "{"), "}"))
			if name == "" || strings.TrimSpace(a) == "" {
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

func cloudWatchInvestigationsSplitPathSegments(path string) []string {
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

func parseCloudWatchInvestigationsPayload(r *http.Request) (map[string]any, error) {
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

func respondCloudWatchInvestigationsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCloudWatchInvestigationsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCloudWatchInvestigationsJSON(w, status, cloudWatchInvestigationsError{Type: code, Message: msg})
}
