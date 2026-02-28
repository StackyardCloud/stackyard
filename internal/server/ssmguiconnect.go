package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type ssmGUIConnectError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleSSMGUIConnectRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSSMGUIConnectRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "ssm-guiconnect")
	if !ok {
		okAlt, _, _, _, _ := s.validateSigV4WithService(r, "ssmguiconnect")
		if !okAlt {
			respondSSMGUIConnectError(w, status, code, msg)
			return true
		}
	}

	action, pathParams := parseSSMGUIConnectRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondSSMGUIConnectError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := ssmGUIConnectOperationByName[action]; !known {
		respondSSMGUIConnectError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseSSMGUIConnectPayload(r)
	if err != nil {
		respondSSMGUIConnectError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	for k, v := range pathParams {
		payload[k] = v
	}

	response := s.ssmguiconnect.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondSSMGUIConnectJSON(w, http.StatusOK, response)
	return true
}

func isSSMGUIConnectRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodPost {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "ssm-guiconnect" && service != "ssmguiconnect" {
		return false
	}
	if service == "ssm-guiconnect" || service == "ssmguiconnect" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, "ssm-guiconnect") || strings.Contains(host, "guiconnect") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#ssm-guiconnect") || strings.Contains(userAgent, " ssm-guiconnect/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/DeleteConnectionRecordingPreferences") ||
		strings.HasPrefix(path, "/GetConnectionRecordingPreferences") ||
		strings.HasPrefix(path, "/UpdateConnectionRecordingPreferences")
}

func parseSSMGUIConnectRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range ssmGUIConnectOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := ssmGUIConnectPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func ssmGUIConnectPathParams(template, actual string) (map[string]string, bool) {
	tSegs := ssmGUIConnectSplitPath(template)
	aSegs := ssmGUIConnectSplitPath(actual)
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

func ssmGUIConnectSplitPath(path string) []string {
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

func parseSSMGUIConnectPayload(r *http.Request) (map[string]any, error) {
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

func respondSSMGUIConnectJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSSMGUIConnectError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSSMGUIConnectJSON(w, status, ssmGUIConnectError{Type: code, Message: msg})
}
