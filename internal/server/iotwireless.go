package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type iotWirelessError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleIoTWirelessRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isIoTWirelessRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "iotwireless")
	if !ok {
		respondIoTWirelessError(w, status, code, msg)
		return true
	}

	action, pathParams := parseIoTWirelessRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondIoTWirelessError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := iotWirelessOperationByName[action]; !known {
		respondIoTWirelessError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseIoTWirelessPayload(r)
	if err != nil {
		respondIoTWirelessError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.iotwireless.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondIoTWirelessJSON(w, http.StatusOK, response)
	return true
}

func isIoTWirelessRESTCandidate(r *http.Request) bool {
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "iotwireless" {
		return false
	}
	if service == "iotwireless" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".iotwireless.") || strings.HasPrefix(host, "iotwireless.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#iotwireless") || strings.Contains(userAgent, " iotwireless/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/wireless-devices") ||
		strings.HasPrefix(path, "/wireless-gateways") ||
		strings.HasPrefix(path, "/destinations") ||
		strings.HasPrefix(path, "/device-profiles") ||
		strings.HasPrefix(path, "/service-profiles") ||
		strings.HasPrefix(path, "/fuota-tasks") ||
		strings.HasPrefix(path, "/multicast-groups") ||
		strings.HasPrefix(path, "/network-analyzer-configurations") ||
		strings.HasPrefix(path, "/partner-accounts") ||
		strings.HasPrefix(path, "/position") ||
		strings.HasPrefix(path, "/resource-positions") ||
		strings.HasPrefix(path, "/service-endpoint") ||
		strings.HasPrefix(path, "/tags")
}

func parseIoTWirelessAction(method, requestPath string) string {
	action, _ := parseIoTWirelessRoute(method, requestPath)
	return action
}

func parseIoTWirelessRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(requestPath)
	if path == "" {
		path = "/"
	}
	for _, op := range iotWirelessOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := iotWirelessPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func iotWirelessPathMatches(template, actual string) bool {
	_, ok := iotWirelessPathParams(template, actual)
	return ok
}

func iotWirelessPathParams(template, actual string) (map[string]string, bool) {
	templatePath := strings.TrimSpace(strings.SplitN(template, "?", 2)[0])
	tSegs := iotWirelessSplitPathSegments(templatePath)
	aSegs := iotWirelessSplitPathSegments(actual)
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

func iotWirelessSplitPathSegments(path string) []string {
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

func parseIoTWirelessPayload(r *http.Request) (map[string]any, error) {
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

func respondIoTWirelessJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondIoTWirelessError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondIoTWirelessJSON(w, status, iotWirelessError{Type: code, Message: msg})
}
