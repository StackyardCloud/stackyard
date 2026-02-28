package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type chimeError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleChimeRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isChimeRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "chime")
	if !ok {
		respondChimeError(w, status, code, msg)
		return true
	}

	action, pathParams := parseChimeRoute(r.Method, rawRequestPath(r), r.URL.Query())
	if action == "" {
		respondChimeError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := chimeOperationByName[action]; !known {
		respondChimeError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseChimePayload(r)
	if err != nil {
		respondChimeError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.chime.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondChimeJSON(w, http.StatusOK, response)
	return true
}

func isChimeRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodPost, http.MethodGet, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "chime" {
		return false
	}
	if service == "chime" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".chime.") || strings.HasPrefix(host, "chime.") || strings.Contains(host, "chime") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#chime") || strings.Contains(userAgent, " chime/") {
		return true
	}

	action, _ := parseChimeRoute(method, rawRequestPath(r), r.URL.Query())
	return action != ""
}

func parseChimeRoute(method, requestPath string, query url.Values) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	bestAction := ""
	bestParams := map[string]string(nil)
	bestScore := -1
	for _, op := range chimeOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, score, ok := chimeMatchTemplate(op.URI, path, query)
		if ok {
			if score > bestScore {
				bestAction = op.Name
				bestParams = params
				bestScore = score
			}
		}
	}
	return bestAction, bestParams
}

func chimeMatchTemplate(template, actualPath string, actualQuery url.Values) (map[string]string, int, bool) {
	templateParts := strings.SplitN(strings.TrimSpace(template), "?", 2)
	templatePath := strings.TrimSpace(templateParts[0])
	if templatePath == "" {
		templatePath = "/"
	}

	tSegs := chimeSplitPath(templatePath)
	aSegs := chimeSplitPath(actualPath)
	if len(tSegs) != len(aSegs) {
		return nil, 0, false
	}

	params := map[string]string{}
	for i := range tSegs {
		t := tSegs[i]
		a := aSegs[i]
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "{"), "}"))
			if name == "" {
				return nil, 0, false
			}
			value, err := url.PathUnescape(a)
			if err != nil {
				value = a
			}
			params[name] = value
			continue
		}
		if t != a {
			return nil, 0, false
		}
	}

	if len(templateParts) == 1 || strings.TrimSpace(templateParts[1]) == "" {
		return params, 0, true
	}

	templateQuery, err := url.ParseQuery(templateParts[1])
	if err != nil {
		return nil, 0, false
	}
	for key, values := range templateQuery {
		actualValues := actualQuery[key]
		if len(actualValues) == 0 {
			return nil, 0, false
		}
		expected := values[len(values)-1]
		actual := actualValues[len(actualValues)-1]
		if strings.HasPrefix(expected, "{") && strings.HasSuffix(expected, "}") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(expected, "{"), "}"))
			if name != "" {
				params[name] = actual
			}
			continue
		}
		if expected != actual {
			return nil, 0, false
		}
	}

	return params, len(templateQuery), true
}

func chimeSplitPath(path string) []string {
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

func parseChimePayload(r *http.Request) (map[string]any, error) {
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

func respondChimeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondChimeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondChimeJSON(w, status, chimeError{Type: code, Message: msg})
}
