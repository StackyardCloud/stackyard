package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type ivsMultitrackError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleIVSMultitrackRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isIVSMultitrackRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "ivs")
	if !ok {
		okAlt, _, _, _, _ := s.validateSigV4WithService(r, "ivsmultitrack")
		if !okAlt {
			respondIVSMultitrackError(w, status, code, msg)
			return true
		}
	}

	action, pathParams := parseIVSMultitrackRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondIVSMultitrackError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := ivsMultitrackOperationByName[action]; !known {
		respondIVSMultitrackError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseIVSMultitrackPayload(r)
	if err != nil {
		respondIVSMultitrackError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.ivsmultitrack.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondIVSMultitrackJSON(w, http.StatusOK, response)
	return true
}

func isIVSMultitrackRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodPost, http.MethodGet:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "ivs" && service != "ivsmultitrack" {
		return false
	}

	action, _ := parseIVSMultitrackRoute(method, rawRequestPath(r))
	if action != "" {
		return true
	}

	if service == "ivsmultitrack" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, "ingest.contribute.live-video.net") || strings.HasPrefix(host, "ingest.")
}

func parseIVSMultitrackRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range ivsMultitrackOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := ivsMultitrackPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func ivsMultitrackPathParams(template, actual string) (map[string]string, bool) {
	tSegs := ivsMultitrackSplitPath(template)
	aSegs := ivsMultitrackSplitPath(actual)
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

func ivsMultitrackSplitPath(path string) []string {
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

func parseIVSMultitrackPayload(r *http.Request) (map[string]any, error) {
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

func respondIVSMultitrackJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondIVSMultitrackError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondIVSMultitrackJSON(w, status, ivsMultitrackError{Type: code, Message: msg})
}
