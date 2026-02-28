package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type mskv1Error struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleMSKV1RESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isMSKV1RESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "kafka")
	if !ok {
		respondMSKV1Error(w, status, code, msg)
		return true
	}

	action, pathParams := parseMSKV1Route(r.Method, rawRequestPath(r))
	if action == "" {
		respondMSKV1Error(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := mskv1OperationByName[action]; !known {
		respondMSKV1Error(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseMSKV1Payload(r)
	if err != nil {
		respondMSKV1Error(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.mskv1.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondMSKV1JSON(w, http.StatusOK, response)
	return true
}

func isMSKV1RESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "kafka" {
		return false
	}
	if action, _ := parseMSKV1Route(method, path); action != "" {
		return true
	}
	if service == "kafka" && strings.HasPrefix(path, "/replication/v1/") {
		return true
	}
	return false
}

func parseMSKV1Route(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range mskv1Operations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := mskv1PathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func mskv1PathParams(template, actual string) (map[string]string, bool) {
	tSegs := mskv1SplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	aSegs := mskv1SplitPath(strings.SplitN(strings.TrimSpace(actual), "?", 2)[0])
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

func mskv1SplitPath(path string) []string {
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

func parseMSKV1Payload(r *http.Request) (map[string]any, error) {
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

func respondMSKV1JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondMSKV1Error(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondMSKV1JSON(w, status, mskv1Error{Type: code, Message: msg})
}
