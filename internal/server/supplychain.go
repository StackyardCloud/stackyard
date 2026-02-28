package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type supplyChainError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleSupplyChainRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSupplyChainRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "scn")
	if !ok {
		respondSupplyChainError(w, status, code, msg)
		return true
	}

	action, pathParams := parseSupplyChainRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondSupplyChainError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := supplyChainOperationByName[action]; !known {
		respondSupplyChainError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseSupplyChainPayload(r)
	if err != nil {
		respondSupplyChainError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.supplychain.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondSupplyChainJSON(w, http.StatusOK, response)
	return true
}

func isSupplyChainRESTCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "scn" {
		return false
	}
	if service == "scn" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".scn.") || strings.HasPrefix(host, "scn.") || strings.Contains(host, "supplychain") || strings.Contains(host, "supply-chain") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#supplychain") || strings.Contains(userAgent, " supplychain/") {
		return true
	}

	action, _ := parseSupplyChainRoute(method, rawRequestPath(r))
	return action != ""
}

func parseSupplyChainRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range supplyChainOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := supplyChainPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}

	return "", nil
}

func supplyChainPathParams(template, actual string) (map[string]string, bool) {
	tSegs := supplyChainSplitPath(strings.SplitN(strings.TrimSpace(template), "?", 2)[0])
	aSegs := supplyChainSplitPath(strings.SplitN(strings.TrimSpace(actual), "?", 2)[0])
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

func supplyChainSplitPath(path string) []string {
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

func parseSupplyChainPayload(r *http.Request) (map[string]any, error) {
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

func respondSupplyChainJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSupplyChainError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSupplyChainJSON(w, status, supplyChainError{Type: code, Message: msg})
}
