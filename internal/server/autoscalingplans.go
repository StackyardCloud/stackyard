package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type autoScalingPlansError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleAutoScalingPlansJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isAutoScalingPlansJSONCandidate(r) {
		return false
	}

	action := parseAutoScalingPlansTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondAutoScalingPlansError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := autoScalingPlansOperationByName[action]; !known {
		respondAutoScalingPlansError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "autoscaling-plans")
	if !ok {
		respondAutoScalingPlansError(w, status, code, msg)
		return true
	}

	payload, err := parseAutoScalingPlansPayload(r)
	if err != nil {
		respondAutoScalingPlansError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.autoscalingplans.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondAutoScalingPlansJSON(w, http.StatusOK, response)
	return true
}

func isAutoScalingPlansJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "AnyScaleScalingPlannerFrontendService.") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.HasPrefix(target, "AnyScaleScalingPlannerFrontendService.")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "autoscaling-plans" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".autoscaling-plans.") || strings.HasPrefix(host, "autoscaling-plans.")
}

func parseAutoScalingPlansTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AnyScaleScalingPlannerFrontendService.") {
		return strings.TrimPrefix(target, "AnyScaleScalingPlannerFrontendService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseAutoScalingPlansPayload(r *http.Request) (map[string]any, error) {
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

func respondAutoScalingPlansJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondAutoScalingPlansError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondAutoScalingPlansJSON(w, status, autoScalingPlansError{Type: code, Message: msg})
}
