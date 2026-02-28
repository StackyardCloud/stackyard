package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type elementalInferenceError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleElementalInferenceRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isElementalInferenceRESTCandidate(r) {
		return false
	}

	payload, err := parseElementalInferencePayload(r)
	if err != nil {
		respondElementalInferenceError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	ok := false
	status := http.StatusForbidden
	code := "AuthFailure"
	msg := "invalid signature"
	for _, serviceName := range []string{"elemental-inference", "elastic-inference", "elementalinference"} {
		accepted, sStatus, sCode, sMsg, _ := s.validateSigV4WithService(r, serviceName)
		if accepted {
			ok = true
			break
		}
		status = sStatus
		code = sCode
		msg = sMsg
	}
	if !ok {
		respondElementalInferenceError(w, status, code, msg)
		return true
	}

	action, pathParams := parseElementalInferenceRoute(r.Method, rawRequestPath(r), r.Header.Get("X-Amz-Target"), payload, r.URL.Query())
	if action == "" {
		respondElementalInferenceError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := elementalInferenceOperationByName[action]; !known {
		respondElementalInferenceError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	response := s.elementalinference.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondElementalInferenceJSON(w, http.StatusOK, response)
	return true
}

func isElementalInferenceRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "elemental-inference" && service != "elastic-inference" && service != "elementalinference" {
		return false
	}
	if service == "elemental-inference" || service == "elastic-inference" || service == "elementalinference" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".elemental-inference.") || strings.HasPrefix(host, "elemental-inference.") {
		return true
	}
	if strings.Contains(host, ".elastic-inference.") || strings.HasPrefix(host, "elastic-inference.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#elemental-inference") || strings.Contains(userAgent, " elemental-inference/") {
		return true
	}
	if strings.Contains(userAgent, "command#elastic-inference") || strings.Contains(userAgent, " elastic-inference/") {
		return true
	}

	action, _ := parseElementalInferenceRoute(method, rawRequestPath(r), "", nil, nil)
	return action != ""
}

func parseElementalInferenceRoute(
	method, requestPath, targetHint string,
	payload map[string]any,
	query url.Values,
) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	type routeMatch struct {
		op     elementalInferenceOperation
		params map[string]string
	}

	matches := make([]routeMatch, 0, 2)
	for _, op := range elementalInferenceOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := elementalInferencePathParams(op.URI, path)
		if !ok {
			continue
		}
		matches = append(matches, routeMatch{op: op, params: params})
	}
	if len(matches) == 0 {
		return "", nil
	}
	if len(matches) == 1 {
		return matches[0].op.Name, matches[0].params
	}

	hints := []string{
		targetHint,
		strings.TrimSpace(query.Get("operation")),
		strings.TrimSpace(query.Get("action")),
		elementalInferencePayloadString(payload, "operation", "Operation", "action", "Action"),
	}
	for _, hint := range hints {
		nHint := elementalInferenceNormalizeToken(hint)
		if nHint == "" {
			continue
		}
		for _, match := range matches {
			nAction := elementalInferenceNormalizeToken(match.op.Name)
			if strings.Contains(nHint, nAction) || strings.Contains(nAction, nHint) {
				return match.op.Name, match.params
			}
		}
	}

	// Associate/Disassociate share the same method+URI.
	if len(matches) == 2 {
		hasAssociationHint := strings.TrimSpace(elementalInferencePayloadString(
			payload,
			"flowArn",
			"FlowArn",
			"resourceArn",
			"ResourceArn",
			"associatedResourceArn",
			"AssociatedResourceArn",
		)) != ""
		if hasAssociationHint {
			for _, match := range matches {
				if match.op.Name == "AssociateFeed" {
					return match.op.Name, match.params
				}
			}
		}
		for _, match := range matches {
			if match.op.Name == "DisassociateFeed" {
				return match.op.Name, match.params
			}
		}
	}

	return matches[0].op.Name, matches[0].params
}

func elementalInferencePathParams(template, actual string) (map[string]string, bool) {
	tSegs := elementalInferenceSplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	aSegs := elementalInferenceSplitPath(strings.SplitN(strings.TrimSpace(actual), "?", 2)[0])
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

func elementalInferenceSplitPath(path string) []string {
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

func parseElementalInferencePayload(r *http.Request) (map[string]any, error) {
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

func respondElementalInferenceJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondElementalInferenceError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondElementalInferenceJSON(w, status, elementalInferenceError{Type: code, Message: msg})
}

func elementalInferenceNormalizeToken(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return ""
	}
	replacer := strings.NewReplacer("-", "", "_", "", ".", "", ":", "", " ", "", "/", "")
	return replacer.Replace(v)
}

func elementalInferencePayloadString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if payload == nil {
			continue
		}
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		if s, ok := raw.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return s
			}
		}
	}
	return ""
}
