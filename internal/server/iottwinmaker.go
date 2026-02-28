package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type iotTwinMakerError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleIoTTwinMakerRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isIoTTwinMakerRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "iottwinmaker")
	if !ok {
		respondIoTTwinMakerError(w, status, code, msg)
		return true
	}

	action, pathParams := parseIoTTwinMakerRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondIoTTwinMakerError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := iotTwinMakerOperationByName[action]; !known {
		respondIoTTwinMakerError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseIoTTwinMakerPayload(r)
	if err != nil {
		respondIoTTwinMakerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.iottwinmaker.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondIoTTwinMakerJSON(w, http.StatusOK, response)
	return true
}

func isIoTTwinMakerRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "iottwinmaker" {
		return false
	}
	if service == "iottwinmaker" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".iottwinmaker.") || strings.HasPrefix(host, "iottwinmaker.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#iottwinmaker") || strings.Contains(userAgent, " iottwinmaker/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/workspaces") ||
		strings.HasPrefix(path, "/metadata-transfer-jobs") ||
		strings.HasPrefix(path, "/sync-jobs") ||
		strings.HasPrefix(path, "/queries/") ||
		strings.HasPrefix(path, "/pricingplan") ||
		strings.HasPrefix(path, "/tags")
}

func parseIoTTwinMakerAction(method, requestPath string) string {
	action, _ := parseIoTTwinMakerRoute(method, requestPath)
	return action
}

func parseIoTTwinMakerRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(requestPath)
	if path == "" {
		path = "/"
	}
	for _, op := range iotTwinMakerOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := iotTwinMakerPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func iotTwinMakerPathMatches(template, actual string) bool {
	_, ok := iotTwinMakerPathParams(template, actual)
	return ok
}

func iotTwinMakerPathParams(template, actual string) (map[string]string, bool) {
	tSegs := iotTwinMakerSplitPathSegments(template)
	aSegs := iotTwinMakerSplitPathSegments(actual)
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

func iotTwinMakerSplitPathSegments(path string) []string {
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

func parseIoTTwinMakerPayload(r *http.Request) (map[string]any, error) {
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

func respondIoTTwinMakerJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondIoTTwinMakerError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondIoTTwinMakerJSON(w, status, iotTwinMakerError{Type: code, Message: msg})
}
