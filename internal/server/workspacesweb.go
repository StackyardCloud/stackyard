package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type workspacesWebError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleWorkSpacesWebRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isWorkSpacesWebRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "workspaces-web")
	if !ok {
		respondWorkSpacesWebError(w, status, code, msg)
		return true
	}

	action, pathParams := parseWorkSpacesWebRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondWorkSpacesWebError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := workspacesWebOperationByName[action]; !known {
		respondWorkSpacesWebError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseWorkSpacesWebPayload(r)
	if err != nil {
		respondWorkSpacesWebError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.workspacesweb.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondWorkSpacesWebJSON(w, http.StatusOK, response)
	return true
}

func isWorkSpacesWebRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "workspaces-web" {
		return false
	}
	if service == "workspaces-web" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".workspaces-web.") || strings.HasPrefix(host, "workspaces-web.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#workspaces-web") || strings.Contains(userAgent, "workspaces-web") {
		return true
	}

	action, _ := parseWorkSpacesWebRoute(method, rawRequestPath(r))
	return action != ""
}

func parseWorkSpacesWebRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	type routeMatch struct {
		action string
		params map[string]string
		score  workspacesWebRouteScore
	}

	var best *routeMatch

	for _, op := range workspacesWebOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := workspacesWebPathParams(op.URI, path)
		if ok {
			candidate := &routeMatch{
				action: op.Name,
				params: params,
				score:  scoreWorkSpacesWebRoute(op.URI),
			}
			if best == nil || candidate.score.betterThan(best.score) {
				best = candidate
			}
		}
	}

	if best != nil {
		return best.action, best.params
	}

	return "", nil
}

type workspacesWebRouteScore struct {
	literalSegments int
	greedySegments  int
	totalSegments   int
}

func (s workspacesWebRouteScore) betterThan(other workspacesWebRouteScore) bool {
	if s.literalSegments != other.literalSegments {
		return s.literalSegments > other.literalSegments
	}
	if s.greedySegments != other.greedySegments {
		return s.greedySegments < other.greedySegments
	}
	return s.totalSegments > other.totalSegments
}

func scoreWorkSpacesWebRoute(template string) workspacesWebRouteScore {
	segments := workspacesWebSplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	score := workspacesWebRouteScore{totalSegments: len(segments)}
	for _, seg := range segments {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(seg, "{"), "}"))
			if strings.HasSuffix(name, "+") {
				score.greedySegments++
			}
			continue
		}
		score.literalSegments++
	}
	return score
}

func workspacesWebPathParams(template, actual string) (map[string]string, bool) {
	tSegs := workspacesWebSplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	aSegs := workspacesWebSplitPath(strings.SplitN(strings.TrimSpace(actual), "?", 2)[0])

	params, ok := workspacesWebMatchPathSegments(tSegs, aSegs, 0, 0, map[string]string{})
	if !ok {
		return nil, false
	}
	return params, true
}

func workspacesWebMatchPathSegments(
	tSegs, aSegs []string,
	ti, ai int,
	params map[string]string,
) (map[string]string, bool) {
	if ti == len(tSegs) {
		if ai == len(aSegs) {
			return params, true
		}
		return nil, false
	}

	tSeg := tSegs[ti]
	if strings.HasPrefix(tSeg, "{") && strings.HasSuffix(tSeg, "}") {
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(tSeg, "{"), "}"))
		greedy := strings.HasSuffix(name, "+")
		if greedy {
			name = strings.TrimSuffix(name, "+")
		}
		if name == "" {
			return nil, false
		}

		if !greedy {
			if ai >= len(aSegs) || strings.TrimSpace(aSegs[ai]) == "" {
				return nil, false
			}
			raw := aSegs[ai]
			value, err := url.PathUnescape(raw)
			if err != nil {
				value = raw
			}
			next := make(map[string]string, len(params)+1)
			for k, v := range params {
				next[k] = v
			}
			next[name] = value
			return workspacesWebMatchPathSegments(tSegs, aSegs, ti+1, ai+1, next)
		}

		if ai >= len(aSegs) {
			return nil, false
		}
		for end := len(aSegs); end >= ai+1; end-- {
			raw := strings.Join(aSegs[ai:end], "/")
			value, err := url.PathUnescape(raw)
			if err != nil {
				value = raw
			}
			next := make(map[string]string, len(params)+1)
			for k, v := range params {
				next[k] = v
			}
			next[name] = value
			if matched, ok := workspacesWebMatchPathSegments(tSegs, aSegs, ti+1, end, next); ok {
				return matched, true
			}
		}
		return nil, false
	}

	if ai >= len(aSegs) || tSeg != aSegs[ai] {
		return nil, false
	}
	return workspacesWebMatchPathSegments(tSegs, aSegs, ti+1, ai+1, params)
}

func workspacesWebSplitPath(path string) []string {
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

func parseWorkSpacesWebPayload(r *http.Request) (map[string]any, error) {
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

func respondWorkSpacesWebJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondWorkSpacesWebError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondWorkSpacesWebJSON(w, status, workspacesWebError{Type: code, Message: msg})
}
