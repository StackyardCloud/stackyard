package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type cloudMapError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleCloudMapJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isCloudMapJSONCandidate(r) {
		return false
	}

	action := parseCloudMapTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondCloudMapError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := cloudMapOperationByName[action]; !known {
		respondCloudMapError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "servicediscovery")
	if !ok {
		respondCloudMapError(w, status, code, msg)
		return true
	}

	payload, err := parseCloudMapPayload(r)
	if err != nil {
		respondCloudMapError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.cloudmap.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondCloudMapJSON(w, http.StatusOK, response)
	return true
}

func isCloudMapJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "Route53AutoNaming_v20170314") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "Route53AutoNaming_v20170314")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "servicediscovery" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".servicediscovery.") ||
		strings.HasPrefix(host, "servicediscovery.") ||
		strings.Contains(host, ".cloud-map.") ||
		strings.HasPrefix(host, "cloud-map.")
}

func parseCloudMapTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "Route53AutoNaming_v20170314.") {
		return strings.TrimPrefix(target, "Route53AutoNaming_v20170314.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseCloudMapPayload(r *http.Request) (map[string]any, error) {
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

func respondCloudMapJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondCloudMapError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondCloudMapJSON(w, status, cloudMapError{Type: code, Message: msg})
}
