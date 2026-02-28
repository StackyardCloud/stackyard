package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type arcRegionSwitchError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleARCRegionSwitchJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isARCRegionSwitchJSONCandidate(r) {
		return false
	}

	action := parseARCRegionSwitchTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondARCRegionSwitchError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := arcRegionSwitchOperationByName[action]; !known {
		respondARCRegionSwitchError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "arc-region-switch")
	if !ok {
		respondARCRegionSwitchError(w, status, code, msg)
		return true
	}

	payload, err := parseARCRegionSwitchPayload(r)
	if err != nil {
		respondARCRegionSwitchError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.arcregionswitch.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondARCRegionSwitchJSON(w, http.StatusOK, response)
	return true
}

func isARCRegionSwitchJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "ArcRegionSwitchService") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "ArcRegionSwitchService")
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "arc-region-switch" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".arc-region-switch.") || strings.HasPrefix(host, "arc-region-switch.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(userAgent, "command#arc-region-switch") || strings.Contains(userAgent, " arc-region-switch/")
}

func parseARCRegionSwitchTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "ArcRegionSwitchService.") {
		return strings.TrimPrefix(target, "ArcRegionSwitchService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseARCRegionSwitchPayload(r *http.Request) (map[string]any, error) {
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

func respondARCRegionSwitchJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondARCRegionSwitchError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondARCRegionSwitchJSON(w, status, arcRegionSwitchError{Type: code, Message: msg})
}
