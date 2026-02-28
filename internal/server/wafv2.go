package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type wafv2Error struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleWAFV2JSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isWAFV2JSONCandidate(r) {
		return false
	}

	action := parseWAFV2Target(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondWAFV2Error(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := wafv2OperationByName[action]; !known {
		respondWAFV2Error(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "wafv2")
	if !ok {
		respondWAFV2Error(w, status, code, msg)
		return true
	}

	payload, err := parseWAFV2Payload(r)
	if err != nil {
		respondWAFV2Error(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.wafv2.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondWAFV2JSON(w, http.StatusOK, response)
	return true
}

func isWAFV2JSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "AWSWAF_20190729") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "AWSWAF_20190729")
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "wafv2" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".wafv2.") || strings.HasPrefix(host, "wafv2.")
}

func parseWAFV2Target(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "AWSWAF_20190729.") {
		return strings.TrimPrefix(target, "AWSWAF_20190729.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseWAFV2Payload(r *http.Request) (map[string]any, error) {
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

func respondWAFV2JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondWAFV2Error(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondWAFV2JSON(w, status, wafv2Error{Type: code, Message: msg})
}
