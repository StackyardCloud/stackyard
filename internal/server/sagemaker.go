package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type sagemakerError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleSageMakerJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSageMakerJSONCandidate(r) {
		return false
	}

	action := parseSageMakerTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondSageMakerError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := sagemakerOperationByName[action]; !known {
		respondSageMakerError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "sagemaker")
	if !ok {
		respondSageMakerError(w, status, code, msg)
		return true
	}

	payload, err := parseSageMakerPayload(r)
	if err != nil {
		respondSageMakerError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.sagemaker.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondSageMakerJSON(w, http.StatusOK, response)
	return true
}

func isSageMakerJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "SageMaker.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") && strings.Contains(target, "SageMaker.") {
		return true
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "sagemaker" && strings.Contains(target, "SageMaker.") {
		return true
	}

	return false
}

func parseSageMakerTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "SageMaker.") {
		return strings.TrimPrefix(target, "SageMaker.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseSageMakerPayload(r *http.Request) (map[string]any, error) {
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

func respondSageMakerJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSageMakerError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSageMakerJSON(w, status, sagemakerError{Type: code, Message: msg})
}
