package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type mgnError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleMGNRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isMGNRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "mgn")
	if !ok {
		respondMGNError(w, status, code, msg)
		return true
	}

	action, pathParams := parseMGNRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondMGNError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := mgnOperationByName[action]; !known {
		respondMGNError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseMGNPayload(r)
	if err != nil {
		respondMGNError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}
	for key, value := range pathParams {
		if _, exists := payload[key]; !exists {
			payload[key] = value
		}
	}
	for key, values := range r.URL.Query() {
		if len(values) == 0 {
			continue
		}
		if _, exists := payload[key]; !exists {
			if len(values) == 1 {
				payload[key] = values[0]
			} else {
				items := make([]any, 0, len(values))
				for _, value := range values {
					items = append(items, value)
				}
				payload[key] = items
			}
		}
	}

	response := s.mgn.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondMGNJSON(w, http.StatusOK, response)
	return true
}

func isMGNRESTRouterCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "mgn" {
		return false
	}
	if service == "mgn" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".mgn.") || strings.HasPrefix(host, "mgn.") {
		return true
	}

	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(ua, "command#mgn") || strings.Contains(ua, " mgn/") {
		return true
	}

	action, _ := parseMGNRoute(r.Method, rawRequestPath(r))
	return action != ""
}

func parseMGNRoute(method, requestPath string) (string, map[string]string) {
	normalizedMethod := strings.ToUpper(strings.TrimSpace(method))
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	for _, op := range mgnOperations {
		if !strings.EqualFold(op.Method, normalizedMethod) {
			continue
		}
		params, ok := mgnPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func mgnPathParams(template, actual string) (map[string]string, bool) {
	tSegments := mgnSplitPath(template)
	aSegments := mgnSplitPath(actual)
	if len(tSegments) != len(aSegments) {
		return nil, false
	}

	params := map[string]string{}
	for i := range tSegments {
		t := tSegments[i]
		a := aSegments[i]
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

func mgnSplitPath(path string) []string {
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

func parseMGNPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
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

func respondMGNJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondMGNError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondMGNJSON(w, status, mgnError{Type: code, Message: msg})
}
