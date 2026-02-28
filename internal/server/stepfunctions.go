package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type stepFunctionsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleStepFunctionsJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isStepFunctionsJSONCandidate(r) {
		return false
	}

	action := parseStepFunctionsTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondStepFunctionsError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := stepFunctionsOperationByName[action]; !known {
		respondStepFunctionsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "states")
	if !ok {
		respondStepFunctionsError(w, status, code, msg)
		return true
	}

	payload, err := parseStepFunctionsPayload(r)
	if err != nil {
		respondStepFunctionsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.stepfunctions.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondStepFunctionsJSON(w, http.StatusOK, response)
	return true
}

func isStepFunctionsJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AWSStepFunctions.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") || strings.Contains(contentType, "application/x-amz-json-1.1") {
		if target == "" {
			return strings.TrimSpace(sigV4ServiceHint(r)) == "states"
		}
		return strings.Contains(target, "AWSStepFunctions.")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "states" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".states.") || strings.HasPrefix(host, "states.")
}

func parseStepFunctionsTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSStepFunctions.") {
		return strings.TrimPrefix(target, "AWSStepFunctions.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseStepFunctionsPayload(r *http.Request) (map[string]any, error) {
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

func respondStepFunctionsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondStepFunctionsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondStepFunctionsJSON(w, status, stepFunctionsError{Type: code, Message: msg})
}
