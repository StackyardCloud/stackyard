package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type dataPipelineError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleDataPipelineJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isDataPipelineJSONCandidate(r) {
		return false
	}

	action := parseDataPipelineTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondDataPipelineError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := dataPipelineOperationByName[action]; !known {
		respondDataPipelineError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "datapipeline")
	if !ok {
		respondDataPipelineError(w, status, code, msg)
		return true
	}

	payload, err := parseDataPipelinePayload(r)
	if err != nil {
		respondDataPipelineError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.datapipeline.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondDataPipelineJSON(w, http.StatusOK, response)
	return true
}

func isDataPipelineJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "DataPipeline.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "DataPipeline.")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "datapipeline" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".datapipeline.") || strings.Contains(host, ".data-pipeline.") || strings.HasPrefix(host, "datapipeline.") || strings.HasPrefix(host, "data-pipeline.")
}

func parseDataPipelineTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "DataPipeline.") {
		return strings.TrimPrefix(target, "DataPipeline.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseDataPipelinePayload(r *http.Request) (map[string]any, error) {
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

func respondDataPipelineJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondDataPipelineError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondDataPipelineJSON(w, status, dataPipelineError{Type: code, Message: msg})
}
