package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type emrError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleEMRJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isEMRJSONCandidate(r) {
		return false
	}

	action := parseEMRTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondEMRError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := emrOperationByName[action]; !known {
		respondEMRError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "elasticmapreduce")
	if !ok {
		respondEMRError(w, status, code, msg)
		return true
	}

	payload, err := parseEMRPayload(r)
	if err != nil {
		respondEMRError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.emr.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondEMRJSON(w, http.StatusOK, response)
	return true
}

func isEMRJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "ElasticMapReduce") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "ElasticMapReduce")
	}

	svc := strings.TrimSpace(sigV4ServiceHint(r))
	if svc == "elasticmapreduce" || svc == "emr" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".elasticmapreduce.") ||
		strings.HasPrefix(host, "elasticmapreduce.") ||
		strings.Contains(host, ".emr.") ||
		strings.HasPrefix(host, "emr.")
}

func parseEMRTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "ElasticMapReduce.") {
		return strings.TrimPrefix(target, "ElasticMapReduce.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseEMRPayload(r *http.Request) (map[string]any, error) {
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

func respondEMRJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondEMRError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondEMRJSON(w, status, emrError{Type: code, Message: msg})
}
