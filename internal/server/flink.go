package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type flinkError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleFlinkJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isFlinkJSONCandidate(r) {
		return false
	}

	action := parseFlinkTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondFlinkError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := flinkOperationByName[action]; !known {
		respondFlinkError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "kinesisanalytics")
	if !ok {
		respondFlinkError(w, status, code, msg)
		return true
	}

	payload, err := parseFlinkPayload(r)
	if err != nil {
		respondFlinkError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.flink.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondFlinkJSON(w, http.StatusOK, response)
	return true
}

func isFlinkJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "KinesisAnalytics_20180523") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "KinesisAnalytics")
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "kinesisanalytics" || service == "kinesisanalyticsv2" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".kinesisanalytics.") || strings.HasPrefix(host, "kinesisanalytics.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(userAgent, "command#kinesisanalyticsv2") || strings.Contains(userAgent, " kinesisanalyticsv2/")
}

func parseFlinkTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "KinesisAnalytics_20180523.") {
		return strings.TrimPrefix(target, "KinesisAnalytics_20180523.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseFlinkPayload(r *http.Request) (map[string]any, error) {
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

func respondFlinkJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondFlinkError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondFlinkJSON(w, status, flinkError{Type: code, Message: msg})
}
