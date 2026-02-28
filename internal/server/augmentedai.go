package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type augmentedAIError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleAugmentedAIRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isAugmentedAIRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "sagemaker")
	if !ok {
		respondAugmentedAIError(w, status, code, msg)
		return true
	}

	action, pathParams := parseAugmentedAIRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondAugmentedAIError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := augmentedAIOperationByName[action]; !known {
		respondAugmentedAIError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseAugmentedAIPayload(r)
	if err != nil {
		respondAugmentedAIError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.augmentedai.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondAugmentedAIJSON(w, http.StatusOK, response)
	return true
}

func isAugmentedAIRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "sagemaker" {
		return false
	}
	if service == "sagemaker" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".sagemaker.") || strings.HasPrefix(host, "sagemaker.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#sagemaker-a2i-runtime") || strings.Contains(userAgent, " sagemaker-a2i-runtime/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/human-loops")
}

func parseAugmentedAIRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(requestPath)
	if path == "" {
		path = "/"
	}
	for _, op := range augmentedAIOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := augmentedAIPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func augmentedAIPathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	tSegs := augmentedAISplitPathSegments(templatePath)
	aSegs := augmentedAISplitPathSegments(actual)
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

func augmentedAISplitPathSegments(path string) []string {
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

func parseAugmentedAIPayload(r *http.Request) (map[string]any, error) {
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

func respondAugmentedAIJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondAugmentedAIError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondAugmentedAIJSON(w, status, augmentedAIError{Type: code, Message: msg})
}
