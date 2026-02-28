package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type ebsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
	Reason  string `json:"Reason,omitempty"`
}

func (s *Server) handleEBSRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isEBSRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "ebs")
	if !ok {
		respondEBSError(w, status, code, msg, "")
		return true
	}

	action, pathParams := parseEBSRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondEBSError(w, http.StatusBadRequest, "ValidationException", "unknown action", "")
		return true
	}
	if _, known := ebsOperationByName[action]; !known {
		respondEBSError(w, http.StatusBadRequest, "ValidationException", "unknown action", "")
		return true
	}

	payload := map[string]any{}
	body := []byte{}
	var err error
	if action == "PutSnapshotBlock" {
		body, err = readBodyBytes(r)
		if err != nil {
			respondEBSError(w, http.StatusBadRequest, "ValidationException", "invalid block payload", "")
			return true
		}
	} else {
		payload, err = parseEBSPayload(r)
		if err != nil {
			respondEBSError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body", "")
			return true
		}
	}

	response := s.ebs.Handle(action, payload, pathParams, r.URL.Query(), r.Header, body)
	if response.ErrorCode != "" {
		respondEBSError(w, response.Status, response.ErrorCode, response.ErrorMessage, response.ErrorReason)
		return true
	}

	for k, v := range response.Headers {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		w.Header().Set(k, v)
	}

	statusCode := response.Status
	if statusCode <= 0 {
		statusCode = http.StatusOK
	}

	if response.Body != nil {
		if strings.TrimSpace(w.Header().Get("Content-Type")) == "" {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		w.WriteHeader(statusCode)
		_, _ = w.Write(response.Body)
		return true
	}

	if response.JSON != nil {
		respondEBSJSON(w, statusCode, response.JSON)
		return true
	}

	w.WriteHeader(statusCode)
	return true
}

func isEBSRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "ebs" {
		return false
	}
	if service == "ebs" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".ebs.") || strings.HasPrefix(host, "ebs.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#ebs") || strings.Contains(userAgent, " ebs/") {
		return true
	}

	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	if strings.HasPrefix(path, "/snapshots") {
		return true
	}

	action, _ := parseEBSRoute(method, rawRequestPath(r))
	return action != ""
}

func parseEBSRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range ebsOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := ebsPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func ebsPathParams(template, actual string) (map[string]string, bool) {
	tSegs := ebsSplitPath(template)
	aSegs := ebsSplitPath(actual)
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

func ebsSplitPath(path string) []string {
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

func parseEBSPayload(r *http.Request) (map[string]any, error) {
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

func respondEBSJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondEBSError(w http.ResponseWriter, status int, code, msg, reason string) {
	if status <= 0 {
		status = http.StatusBadRequest
	}
	if strings.TrimSpace(code) == "" {
		code = "ValidationException"
	}
	if strings.TrimSpace(msg) == "" {
		msg = "request failed"
	}
	w.Header().Set("X-Amzn-ErrorType", code)
	respondEBSJSON(w, status, ebsError{
		Type:    code,
		Message: msg,
		Reason:  strings.TrimSpace(reason),
	})
}
