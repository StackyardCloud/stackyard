package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type routingControlError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleRoutingControlJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isRoutingControlJSONCandidate(r) {
		return false
	}

	action := parseRoutingControlTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondRoutingControlError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := routingControlOperationByName[action]; !known {
		respondRoutingControlError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "route53-recovery-cluster")
	if !ok {
		respondRoutingControlError(w, status, code, msg)
		return true
	}

	payload, err := parseRoutingControlPayload(r)
	if err != nil {
		respondRoutingControlError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.routingcontrol.Handle(action, payload, nil, nil)
	if response == nil {
		response = map[string]any{}
	}
	respondRoutingControlJSON(w, http.StatusOK, response)
	return true
}

func isRoutingControlJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "ToggleCustomerAPI") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") || strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "ToggleCustomerAPI")
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "route53-recovery-cluster" {
		return false
	}
	if service == "route53-recovery-cluster" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, "route53-recovery-cluster") || strings.Contains(host, "routing-control") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#route53-recovery-cluster") || strings.Contains(userAgent, " route53-recovery-cluster/") {
		return true
	}
	return false
}

func parseRoutingControlTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "ToggleCustomerAPI.") {
		return strings.TrimPrefix(target, "ToggleCustomerAPI.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseRoutingControlPayload(r *http.Request) (map[string]any, error) {
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

func respondRoutingControlJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondRoutingControlError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondRoutingControlJSON(w, status, routingControlError{Type: code, Message: msg})
}
