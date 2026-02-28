package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type healthImagingError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleHealthImagingRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isHealthImagingRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "medical-imaging")
	if !ok {
		respondHealthImagingError(w, status, code, msg)
		return true
	}

	action, pathParams := parseHealthImagingRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondHealthImagingError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := healthImagingOperationByName[action]; !known {
		respondHealthImagingError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseHealthImagingPayload(r)
	if err != nil {
		respondHealthImagingError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.healthimaging.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondHealthImagingJSON(w, http.StatusOK, response)
	return true
}

func isHealthImagingRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "medical-imaging" {
		return false
	}
	if service == "medical-imaging" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".medical-imaging.") || strings.HasPrefix(host, "medical-imaging.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#medical-imaging") || strings.Contains(userAgent, " medical-imaging/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/datastore") ||
		strings.HasPrefix(path, "/listDICOMImportJobs") ||
		strings.HasPrefix(path, "/getDICOMImportJob") ||
		strings.HasPrefix(path, "/startDICOMImportJob") ||
		strings.HasPrefix(path, "/tags/")
}

func parseHealthImagingAction(method, requestPath string) string {
	action, _ := parseHealthImagingRoute(method, requestPath)
	return action
}

func parseHealthImagingRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(requestPath)
	if path == "" {
		path = "/"
	}
	for _, op := range healthImagingOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := healthImagingPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func healthImagingPathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	tSegs := healthImagingSplitPathSegments(templatePath)
	aSegs := healthImagingSplitPathSegments(actual)
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

func healthImagingSplitPathSegments(path string) []string {
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

func parseHealthImagingPayload(r *http.Request) (map[string]any, error) {
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

func respondHealthImagingJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondHealthImagingError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondHealthImagingJSON(w, status, healthImagingError{Type: code, Message: msg})
}
