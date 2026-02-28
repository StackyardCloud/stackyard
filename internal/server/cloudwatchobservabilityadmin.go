package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type cloudWatchObservabilityAdminError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCloudWatchObservabilityAdminJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCloudWatchObservabilityAdminJSONRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "observabilityadmin")
	if !ok {
		respondCloudWatchObservabilityAdminError(w, status, code, msg)
		return true
	}

	payload, err := parseCloudWatchObservabilityAdminPayload(r)
	if err != nil {
		respondCloudWatchObservabilityAdminError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	action := parseCloudWatchObservabilityAdminAction(r.Method, rawRequestPath(r), r.Header.Get("X-Amz-Target"), payload)
	if action == "" {
		respondCloudWatchObservabilityAdminError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := cloudWatchObservabilityAdminOperationByName[action]; !known {
		respondCloudWatchObservabilityAdminError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	response := s.cloudwatchobservabilityadmin.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondCloudWatchObservabilityAdminJSON(w, http.StatusOK, response)
	return true
}

func isCloudWatchObservabilityAdminJSONRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodPost {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "observabilityadmin" {
		return false
	}
	if service == "observabilityadmin" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".observabilityadmin.") || strings.HasPrefix(host, "observabilityadmin.") || strings.Contains(host, "cloudwatchobservabilityadmin") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#observabilityadmin") || strings.Contains(userAgent, " observabilityadmin/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/Create") ||
		strings.HasPrefix(path, "/Delete") ||
		strings.HasPrefix(path, "/Get") ||
		strings.HasPrefix(path, "/List") ||
		strings.HasPrefix(path, "/Start") ||
		strings.HasPrefix(path, "/Stop") ||
		strings.HasPrefix(path, "/Tag") ||
		strings.HasPrefix(path, "/Untag") ||
		strings.HasPrefix(path, "/Update") ||
		strings.HasPrefix(path, "/Test") ||
		strings.HasPrefix(path, "/Validate")
}

func parseCloudWatchObservabilityAdminAction(method, requestPath, targetHint string, payload map[string]any) string {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}

	for _, op := range cloudWatchObservabilityAdminOperations {
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
		for _, op := range cloudWatchObservabilityAdminOperations {
			if strings.EqualFold(op.Name, hint) {
				return op.Name
			}
		}
	}

	for _, key := range []string{"Action", "action", "Operation", "operation"} {
		value := cloudWatchObservabilityAdminString(payload, key)
		if value == "" {
			continue
		}
		for _, op := range cloudWatchObservabilityAdminOperations {
			if strings.EqualFold(op.Name, value) {
				return op.Name
			}
		}
	}

	return ""
}

func parseCloudWatchObservabilityAdminPayload(r *http.Request) (map[string]any, error) {
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

func respondCloudWatchObservabilityAdminJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCloudWatchObservabilityAdminError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCloudWatchObservabilityAdminJSON(w, status, cloudWatchObservabilityAdminError{Type: code, Message: msg})
}
