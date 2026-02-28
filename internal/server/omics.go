package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type omicsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleOmicsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isOmicsRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "omics")
	if !ok {
		respondOmicsError(w, status, code, msg)
		return true
	}

	action, pathParams := parseOmicsRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondOmicsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := omicsOperationByName[action]; !known {
		respondOmicsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseOmicsPayload(r)
	if err != nil {
		respondOmicsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.omics.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondOmicsJSON(w, http.StatusOK, response)
	return true
}

func isOmicsRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "omics" {
		return false
	}
	if service == "omics" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".omics.") || strings.HasPrefix(host, "omics.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#omics") || strings.Contains(userAgent, " omics/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/annotationStore") ||
		strings.HasPrefix(path, "/annotationStores") ||
		strings.HasPrefix(path, "/import/") ||
		strings.HasPrefix(path, "/referencestore") ||
		strings.HasPrefix(path, "/referencestores") ||
		strings.HasPrefix(path, "/run") ||
		strings.HasPrefix(path, "/runCache") ||
		strings.HasPrefix(path, "/runGroup") ||
		strings.HasPrefix(path, "/s3accesspolicy") ||
		strings.HasPrefix(path, "/sequencestore") ||
		strings.HasPrefix(path, "/sequencestores") ||
		strings.HasPrefix(path, "/share") ||
		strings.HasPrefix(path, "/shares") ||
		strings.HasPrefix(path, "/tags") ||
		strings.HasPrefix(path, "/variantStore") ||
		strings.HasPrefix(path, "/variantStores") ||
		strings.HasPrefix(path, "/workflow")
}

func parseOmicsAction(method, requestPath string) string {
	action, _ := parseOmicsRoute(method, requestPath)
	return action
}

func parseOmicsRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(requestPath)
	if path == "" {
		path = "/"
	}
	for _, op := range omicsOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := omicsPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func omicsPathMatches(template, actual string) bool {
	_, ok := omicsPathParams(template, actual)
	return ok
}

func omicsPathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	tSegs := omicsSplitPathSegments(templatePath)
	aSegs := omicsSplitPathSegments(actual)
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

func omicsSplitPathSegments(path string) []string {
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

func parseOmicsPayload(r *http.Request) (map[string]any, error) {
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

func respondOmicsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondOmicsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondOmicsJSON(w, status, omicsError{Type: code, Message: msg})
}
