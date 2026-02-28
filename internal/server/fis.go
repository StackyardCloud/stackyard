package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type fisError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleFISRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isFISRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "fis")
	if !ok {
		respondFISError(w, status, code, msg)
		return true
	}

	action, pathParams := parseFISRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondFISError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := fisOperationByName[action]; !known {
		respondFISError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseFISPayload(r)
	if err != nil {
		respondFISError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.fis.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondFISJSON(w, http.StatusOK, response)
	return true
}

func isFISRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "fis" {
		return false
	}
	if service == "fis" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".fis.") || strings.HasPrefix(host, "fis.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#fis") || strings.Contains(userAgent, " fis/") {
		return true
	}

	action, _ := parseFISRoute(method, rawRequestPath(r))
	return action != ""
}

func parseFISRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	type routeMatch struct {
		action string
		params map[string]string
		score  fisRouteScore
	}

	var best *routeMatch

	for _, op := range fisOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := fisPathParams(op.URI, path)
		if !ok {
			continue
		}
		candidate := &routeMatch{
			action: op.Name,
			params: params,
			score:  scoreFISRoute(op.URI),
		}
		if best == nil || candidate.score.betterThan(best.score) {
			best = candidate
		}
	}

	if best != nil {
		return best.action, best.params
	}
	return "", nil
}

type fisRouteScore struct {
	literalSegments int
	greedySegments  int
	totalSegments   int
}

func (s fisRouteScore) betterThan(other fisRouteScore) bool {
	if s.literalSegments != other.literalSegments {
		return s.literalSegments > other.literalSegments
	}
	if s.greedySegments != other.greedySegments {
		return s.greedySegments < other.greedySegments
	}
	return s.totalSegments > other.totalSegments
}

func scoreFISRoute(template string) fisRouteScore {
	segments := fisSplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	score := fisRouteScore{totalSegments: len(segments)}
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

func fisPathParams(template, actual string) (map[string]string, bool) {
	tSegs := fisSplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	aSegs := fisSplitPath(strings.SplitN(strings.TrimSpace(actual), "?", 2)[0])

	params, ok := fisMatchPathSegments(tSegs, aSegs, 0, 0, map[string]string{})
	if !ok {
		return nil, false
	}
	return params, true
}

func fisMatchPathSegments(tSegs, aSegs []string, ti, ai int, params map[string]string) (map[string]string, bool) {
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
			return fisMatchPathSegments(tSegs, aSegs, ti+1, ai+1, next)
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
			if matched, ok := fisMatchPathSegments(tSegs, aSegs, ti+1, end, next); ok {
				return matched, true
			}
		}
		return nil, false
	}

	if ai >= len(aSegs) || tSeg != aSegs[ai] {
		return nil, false
	}
	return fisMatchPathSegments(tSegs, aSegs, ti+1, ai+1, params)
}

func fisSplitPath(path string) []string {
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

func parseFISPayload(r *http.Request) (map[string]any, error) {
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

func respondFISJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondFISError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondFISJSON(w, status, fisError{Type: code, Message: msg})
}
