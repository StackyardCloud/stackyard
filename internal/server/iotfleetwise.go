package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type iotFleetWiseError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleIoTFleetWiseJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isIoTFleetWiseJSONCandidate(r) {
		return false
	}

	action := parseIoTFleetWiseTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondIoTFleetWiseError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := iotFleetWiseOperationByName[action]; !known {
		respondIoTFleetWiseError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "iotfleetwise")
	if !ok {
		respondIoTFleetWiseError(w, status, code, msg)
		return true
	}

	payload, err := parseIoTFleetWisePayload(r)
	if err != nil {
		respondIoTFleetWiseError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.iotfleetwise.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondIoTFleetWiseJSON(w, http.StatusOK, response)
	return true
}

func isIoTFleetWiseJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "IoTAutobahnControlPlane.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.Contains(target, "IoTAutobahnControlPlane.")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "iotfleetwise" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".iotfleetwise.") || strings.HasPrefix(host, "iotfleetwise.")
}

func parseIoTFleetWiseTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "IoTAutobahnControlPlane.") {
		return strings.TrimPrefix(target, "IoTAutobahnControlPlane.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseIoTFleetWisePayload(r *http.Request) (map[string]any, error) {
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

func respondIoTFleetWiseJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondIoTFleetWiseError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondIoTFleetWiseJSON(w, status, iotFleetWiseError{Type: code, Message: msg})
}
