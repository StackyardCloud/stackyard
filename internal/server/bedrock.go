package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type bedrockError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleBedrockJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isBedrockJSONCandidate(r) {
		return false
	}

	action := parseBedrockTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondBedrockError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := bedrockOperationByName[action]; !known {
		respondBedrockError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "bedrock")
	if !ok {
		respondBedrockError(w, status, code, msg)
		return true
	}

	payload, err := parseBedrockPayload(r)
	if err != nil {
		respondBedrockError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.bedrock.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondBedrockJSON(w, http.StatusOK, response)
	return true
}

func isBedrockJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "Bedrock") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "Bedrock")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "bedrock" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".bedrock.") || strings.HasPrefix(host, "bedrock.")
}

func parseBedrockTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "Bedrock_20230420.") {
		return strings.TrimPrefix(target, "Bedrock_20230420.")
	}
	if strings.HasPrefix(target, "AmazonBedrockControlPlaneService.") {
		return strings.TrimPrefix(target, "AmazonBedrockControlPlaneService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseBedrockPayload(r *http.Request) (map[string]any, error) {
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

func respondBedrockJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondBedrockError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondBedrockJSON(w, status, bedrockError{Type: code, Message: msg})
}
