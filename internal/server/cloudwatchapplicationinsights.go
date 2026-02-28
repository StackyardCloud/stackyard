package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type cloudWatchApplicationInsightsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCloudWatchApplicationInsightsJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCloudWatchApplicationInsightsJSONCandidate(r) {
		return false
	}

	action := parseCloudWatchApplicationInsightsTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondCloudWatchApplicationInsightsError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := cloudWatchApplicationInsightsOperationByName[action]; !known {
		respondCloudWatchApplicationInsightsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "applicationinsights")
	if !ok {
		respondCloudWatchApplicationInsightsError(w, status, code, msg)
		return true
	}

	payload, err := parseCloudWatchApplicationInsightsPayload(r)
	if err != nil {
		respondCloudWatchApplicationInsightsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.cloudwatchapplicationinsights.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondCloudWatchApplicationInsightsJSON(w, http.StatusOK, response)
	return true
}

func isCloudWatchApplicationInsightsJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "EC2WindowsBarleyService.") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "EC2WindowsBarleyService.")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "applicationinsights" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".application-insights.") || strings.HasPrefix(host, "application-insights.")
}

func parseCloudWatchApplicationInsightsTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "EC2WindowsBarleyService.") {
		return strings.TrimPrefix(target, "EC2WindowsBarleyService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseCloudWatchApplicationInsightsPayload(r *http.Request) (map[string]any, error) {
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

func respondCloudWatchApplicationInsightsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCloudWatchApplicationInsightsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCloudWatchApplicationInsightsJSON(w, status, cloudWatchApplicationInsightsError{Type: code, Message: msg})
}
