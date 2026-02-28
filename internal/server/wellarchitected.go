package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type wellArchitectedError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleWellArchitectedRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isWellArchitectedRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "wellarchitected")
	if !ok {
		respondWellArchitectedError(w, status, code, msg)
		return true
	}

	action, pathParams := parseWellArchitectedRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondWellArchitectedError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := wellArchitectedOperationByName[action]; !known {
		respondWellArchitectedError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseWellArchitectedPayload(r)
	if err != nil {
		respondWellArchitectedError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.wellarchitected.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondWellArchitectedJSON(w, http.StatusOK, response)
	return true
}

func isWellArchitectedRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "wellarchitected" {
		return false
	}
	if service == "wellarchitected" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".wellarchitected.") || strings.HasPrefix(host, "wellarchitected.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#wellarchitected") || strings.Contains(userAgent, " wellarchitected/") {
		return true
	}

	action, _ := parseWellArchitectedRoute(method, rawRequestPath(r))
	return action != ""
}

func parseWellArchitectedRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	type routeMatch struct {
		action        string
		params        map[string]string
		literalCount  int
		segmentCount  int
		paramSegments int
	}
	var best *routeMatch

	for _, op := range wellArchitectedOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := wellArchitectedPathParams(op.URI, path)
		if !ok {
			continue
		}
		segments := wellArchitectedSplitPath(op.URI)
		literalCount := 0
		paramSegments := 0
		for _, seg := range segments {
			if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
				paramSegments++
			} else {
				literalCount++
			}
		}
		candidate := &routeMatch{
			action:        op.Name,
			params:        params,
			literalCount:  literalCount,
			segmentCount:  len(segments),
			paramSegments: paramSegments,
		}
		if best == nil ||
			candidate.literalCount > best.literalCount ||
			(candidate.literalCount == best.literalCount && candidate.segmentCount > best.segmentCount) ||
			(candidate.literalCount == best.literalCount && candidate.segmentCount == best.segmentCount && candidate.paramSegments < best.paramSegments) {
			best = candidate
		}
	}

	if best == nil {
		return "", nil
	}
	return best.action, best.params
}

func wellArchitectedPathParams(template, actual string) (map[string]string, bool) {
	tSegs := wellArchitectedSplitPath(template)
	aSegs := wellArchitectedSplitPath(actual)
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

func wellArchitectedSplitPath(path string) []string {
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

func parseWellArchitectedPayload(r *http.Request) (map[string]any, error) {
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

func respondWellArchitectedJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondWellArchitectedError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondWellArchitectedJSON(w, status, wellArchitectedError{Type: code, Message: msg})
}
