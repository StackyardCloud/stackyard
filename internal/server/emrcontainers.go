package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type emrContainersError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleEMRContainersJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isEMRContainersJSONCandidate(r) {
		return false
	}

	action := parseEMRContainersTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondEMRContainersError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := emrContainersOperationByName[action]; !known {
		respondEMRContainersError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "emr-containers")
	if !ok {
		respondEMRContainersError(w, status, code, msg)
		return true
	}

	payload, err := parseEMRContainersPayload(r)
	if err != nil {
		respondEMRContainersError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.emrcontainers.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondEMRContainersJSON(w, http.StatusOK, response)
	return true
}

func isEMRContainersJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "EMRContainers") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "EMRContainers")
	}

	svc := strings.TrimSpace(sigV4ServiceHint(r))
	if svc == "emr-containers" || svc == "emrcontainers" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".emr-containers.") ||
		strings.HasPrefix(host, "emr-containers.") ||
		strings.Contains(host, ".emrcontainers.") ||
		strings.HasPrefix(host, "emrcontainers.")
}

func parseEMRContainersTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "EMRContainers.") {
		return strings.TrimPrefix(target, "EMRContainers.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseEMRContainersPayload(r *http.Request) (map[string]any, error) {
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

func respondEMRContainersJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondEMRContainersError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondEMRContainersJSON(w, status, emrContainersError{Type: code, Message: msg})
}
