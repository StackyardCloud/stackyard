package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type firehoseError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleFirehoseJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isFirehoseJSONCandidate(r) {
		return false
	}

	action := parseFirehoseTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondFirehoseError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := firehoseOperationByName[action]; !known {
		respondFirehoseError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "firehose")
	if !ok {
		respondFirehoseError(w, status, code, msg)
		return true
	}

	payload, err := parseFirehosePayload(r)
	if err != nil {
		respondFirehoseError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.firehose.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondFirehoseJSON(w, http.StatusOK, response)
	return true
}

func isFirehoseJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "Firehose_20150804") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "Firehose_20150804")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "firehose" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".firehose.") ||
		strings.HasPrefix(host, "firehose.") ||
		strings.Contains(host, ".kinesis.") ||
		strings.HasPrefix(host, "kinesis.")
}

func parseFirehoseTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "Firehose_20150804.") {
		return strings.TrimPrefix(target, "Firehose_20150804.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseFirehosePayload(r *http.Request) (map[string]any, error) {
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

func respondFirehoseJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondFirehoseError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondFirehoseJSON(w, status, firehoseError{Type: code, Message: msg})
}
