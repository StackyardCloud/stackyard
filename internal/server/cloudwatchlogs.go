package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"unicode"
)

type cloudWatchLogsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCloudWatchLogsJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCloudWatchLogsJSONRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "logs")
	if !ok {
		respondCloudWatchLogsError(w, status, code, msg)
		return true
	}

	payload, err := parseCloudWatchLogsPayload(r)
	if err != nil {
		respondCloudWatchLogsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	action := parseCloudWatchLogsAction(r.Method, rawRequestPath(r), r.Header.Get("X-Amz-Target"), payload)
	if action == "" {
		respondCloudWatchLogsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := cloudWatchLogsOperationByName[action]; !known {
		respondCloudWatchLogsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	response := s.cloudwatchlogs.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondCloudWatchLogsJSON(w, http.StatusOK, response)
	return true
}

func isCloudWatchLogsJSONRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodPost {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "logs" {
		return false
	}
	if service == "logs" {
		return true
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "Logs_20140328.") {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".logs.") || strings.HasPrefix(host, "logs.") || strings.Contains(host, "cloudwatchlogs") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#logs") || strings.Contains(userAgent, " logs/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	if path == "/" {
		return target != ""
	}
	if len(path) > 1 && path[0] == '/' {
		r := []rune(path[1:])
		if len(r) > 0 && unicode.IsUpper(r[0]) {
			return true
		}
	}
	return false
}

func parseCloudWatchLogsAction(method, requestPath, targetHint string, payload map[string]any) string {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range cloudWatchLogsOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		if path == op.URI {
			return op.Name
		}
	}

	targetHint = strings.TrimSpace(targetHint)
	if targetHint != "" {
		parts := strings.Split(targetHint, ".")
		hint := parts[len(parts)-1]
		for _, op := range cloudWatchLogsOperations {
			if strings.EqualFold(op.Name, hint) {
				return op.Name
			}
		}
	}

	for _, key := range []string{"Action", "action", "Operation", "operation"} {
		value := cloudWatchLogsString(payload, key)
		if value == "" {
			continue
		}
		for _, op := range cloudWatchLogsOperations {
			if strings.EqualFold(op.Name, value) {
				return op.Name
			}
		}
	}

	return ""
}

func parseCloudWatchLogsPayload(r *http.Request) (map[string]any, error) {
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

func respondCloudWatchLogsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCloudWatchLogsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCloudWatchLogsJSON(w, status, cloudWatchLogsError{Type: code, Message: msg})
}

func cloudWatchLogsString(payload map[string]any, key string) string {
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		if s, ok := v.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
