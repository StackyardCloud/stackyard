package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type comprehendMedicalError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleComprehendMedicalJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isComprehendMedicalJSONCandidate(r) {
		return false
	}

	action := parseComprehendMedicalTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondComprehendMedicalError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := comprehendMedicalOperationByName[action]; !known {
		respondComprehendMedicalError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "comprehendmedical")
	if !ok {
		respondComprehendMedicalError(w, status, code, msg)
		return true
	}

	payload, err := parseComprehendMedicalPayload(r)
	if err != nil {
		respondComprehendMedicalError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.comprehendmedical.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondComprehendMedicalJSON(w, http.StatusOK, response)
	return true
}

func isComprehendMedicalJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "ComprehendMedical_20181030") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "ComprehendMedical_20181030")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "comprehendmedical" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".comprehendmedical.") || strings.HasPrefix(host, "comprehendmedical.")
}

func parseComprehendMedicalTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "ComprehendMedical_20181030.") {
		return strings.TrimPrefix(target, "ComprehendMedical_20181030.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseComprehendMedicalPayload(r *http.Request) (map[string]any, error) {
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

func respondComprehendMedicalJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondComprehendMedicalError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondComprehendMedicalJSON(w, status, comprehendMedicalError{Type: code, Message: msg})
}
