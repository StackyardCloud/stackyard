package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type lakeFormationError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleLakeFormationRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isLakeFormationRESTCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "lakeformation")
	if !ok {
		respondLakeFormationError(w, status, code, msg)
		return true
	}

	action := parseLakeFormationRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondLakeFormationError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := lakeFormationOperationByName[action]; !known {
		respondLakeFormationError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseLakeFormationPayload(r)
	if err != nil {
		respondLakeFormationError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.lakeformation.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondLakeFormationJSON(w, http.StatusOK, response)
	return true
}

func isLakeFormationRESTCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "lakeformation" {
		return false
	}
	if service == "lakeformation" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".lakeformation.") || strings.HasPrefix(host, "lakeformation.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#lakeformation") || strings.Contains(userAgent, " lakeformation/") {
		return true
	}

	action := parseLakeFormationRoute(r.Method, rawRequestPath(r))
	if action == "" {
		return false
	}
	_, known := lakeFormationOperationByName[action]
	return known
}

func parseLakeFormationRoute(method, requestPath string) string {
	if !strings.EqualFold(method, http.MethodPost) {
		return ""
	}

	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}

	if slash := strings.Index(path, "/"); slash >= 0 {
		path = path[:slash]
	}
	return strings.TrimSpace(path)
}

func parseLakeFormationPayload(r *http.Request) (map[string]any, error) {
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

func respondLakeFormationJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondLakeFormationError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondLakeFormationJSON(w, status, lakeFormationError{Type: code, Message: msg})
}
