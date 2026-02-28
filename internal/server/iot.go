package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type iotError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleIoTRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isIoTRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "iot")
	if !ok {
		respondIoTError(w, status, code, msg)
		return true
	}

	action, pathParams := parseIoTRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondIoTError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := iotOperationByName[action]; !known {
		respondIoTError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseIoTPayload(r)
	if err != nil {
		respondIoTError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.iot.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondIoTJSON(w, http.StatusOK, response)
	return true
}

func isIoTRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "iot" {
		return false
	}
	if service == "iot" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".iot.") || strings.HasPrefix(host, "iot.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#iot") || strings.Contains(userAgent, " iot/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/things") ||
		strings.HasPrefix(path, "/thing-types") ||
		strings.HasPrefix(path, "/thing-groups") ||
		strings.HasPrefix(path, "/billing-groups") ||
		strings.HasPrefix(path, "/policies") ||
		strings.HasPrefix(path, "/certificates") ||
		strings.HasPrefix(path, "/jobs") ||
		strings.HasPrefix(path, "/audit") ||
		strings.HasPrefix(path, "/security-profiles") ||
		strings.HasPrefix(path, "/rules") ||
		strings.HasPrefix(path, "/tags") ||
		strings.HasPrefix(path, "/commands")
}

func parseIoTAction(method, requestPath string) string {
	action, _ := parseIoTRoute(method, requestPath)
	return action
}

func parseIoTRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(requestPath)
	if path == "" {
		path = "/"
	}
	for _, op := range iotOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := iotPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func iotPathMatches(template, actual string) bool {
	_, ok := iotPathParams(template, actual)
	return ok
}

func iotPathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	tSegs := iotSplitPathSegments(templatePath)
	aSegs := iotSplitPathSegments(actual)

	params := map[string]string{}
	ti := 0
	ai := 0
	for ti < len(tSegs) {
		t := tSegs[ti]
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "{"), "}"))
			greedy := strings.HasSuffix(name, "+")
			if greedy {
				name = strings.TrimSuffix(name, "+")
			}
			if name == "" {
				return nil, false
			}
			if greedy {
				if ti != len(tSegs)-1 {
					return nil, false
				}
				if ai >= len(aSegs) {
					return nil, false
				}
				raw := strings.Join(aSegs[ai:], "/")
				value, err := url.PathUnescape(raw)
				if err != nil {
					value = raw
				}
				params[name] = value
				ai = len(aSegs)
				ti++
				break
			}
			if ai >= len(aSegs) || strings.TrimSpace(aSegs[ai]) == "" {
				return nil, false
			}
			value, err := url.PathUnescape(aSegs[ai])
			if err != nil {
				value = aSegs[ai]
			}
			params[name] = value
			ai++
			ti++
			continue
		}
		if ai >= len(aSegs) || t != aSegs[ai] {
			return nil, false
		}
		ai++
		ti++
	}

	if ai != len(aSegs) {
		return nil, false
	}
	return params, true
}

func iotSplitPathSegments(path string) []string {
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

func parseIoTPayload(r *http.Request) (map[string]any, error) {
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

func respondIoTJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondIoTError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondIoTJSON(w, status, iotError{Type: code, Message: msg})
}
