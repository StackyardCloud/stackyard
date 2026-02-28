package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type notificationsContactsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleNotificationsContactsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isNotificationsContactsRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "notificationscontacts")
	if !ok {
		respondNotificationsContactsError(w, status, code, msg)
		return true
	}

	action, pathParams := parseNotificationsContactsRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondNotificationsContactsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := notificationsContactsOperationByName[action]; !known {
		respondNotificationsContactsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseNotificationsContactsPayload(r)
	if err != nil {
		respondNotificationsContactsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.notificationscontacts.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondNotificationsContactsJSON(w, http.StatusOK, response)
	return true
}

func isNotificationsContactsRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodGet && method != http.MethodPost && method != http.MethodPut && method != http.MethodDelete {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "notificationscontacts" {
		return false
	}
	if service == "notificationscontacts" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".notificationscontacts.") || strings.HasPrefix(host, "notificationscontacts.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#notificationscontacts") || strings.Contains(userAgent, " notificationscontacts/") {
		return true
	}

	action, _ := parseNotificationsContactsRoute(method, rawRequestPath(r))
	return action != ""
}

func parseNotificationsContactsRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range notificationsContactsOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := notificationsContactsPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func notificationsContactsPathParams(template, actual string) (map[string]string, bool) {
	tSegs := notificationsContactsSplitPath(template)
	aSegs := notificationsContactsSplitPath(actual)
	if len(tSegs) != len(aSegs) {
		return nil, false
	}

	params := map[string]string{}
	for i := range tSegs {
		t := tSegs[i]
		a := aSegs[i]
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "{"), "}"))
			if name == "" || strings.TrimSpace(a) == "" {
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

func notificationsContactsSplitPath(path string) []string {
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

func parseNotificationsContactsPayload(r *http.Request) (map[string]any, error) {
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

func respondNotificationsContactsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondNotificationsContactsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondNotificationsContactsJSON(w, status, notificationsContactsError{Type: code, Message: msg})
}
