package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type mskConnectError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleMSKConnectRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isMSKConnectRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "kafkaconnect")
	if !ok {
		respondMSKConnectError(w, status, code, msg)
		return true
	}

	action, pathParams := parseMSKConnectRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondMSKConnectError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := mskConnectOperationByName[action]; !known {
		respondMSKConnectError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseMSKConnectPayload(r)
	if err != nil {
		respondMSKConnectError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.mskconnect.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondMSKConnectJSON(w, http.StatusOK, response)
	return true
}

func isMSKConnectRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "kafkaconnect" {
		return false
	}
	if service == "kafkaconnect" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".kafkaconnect.") || strings.HasPrefix(host, "kafkaconnect.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#kafkaconnect") || strings.Contains(userAgent, " kafkaconnect/") {
		return true
	}

	action, _ := parseMSKConnectRoute(method, rawRequestPath(r))
	return action != ""
}

func parseMSKConnectRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range mskConnectOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := mskConnectPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	return "", nil
}

func mskConnectPathParams(template, actual string) (map[string]string, bool) {
	tSegs := mskConnectSplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	aSegs := mskConnectSplitPath(strings.SplitN(strings.TrimSpace(actual), "?", 2)[0])
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

func mskConnectSplitPath(path string) []string {
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

func parseMSKConnectPayload(r *http.Request) (map[string]any, error) {
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

func respondMSKConnectJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondMSKConnectError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondMSKConnectJSON(w, status, mskConnectError{Type: code, Message: msg})
}
