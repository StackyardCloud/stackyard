package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type arcZonalShiftError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleARCZonalShiftJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isARCZonalShiftJSONCandidate(r) {
		return false
	}

	action := parseARCZonalShiftTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondARCZonalShiftError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := arcZonalShiftOperationByName[action]; !known {
		respondARCZonalShiftError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "arc-zonal-shift")
	if !ok {
		respondARCZonalShiftError(w, status, code, msg)
		return true
	}

	payload, err := parseARCZonalShiftPayload(r)
	if err != nil {
		respondARCZonalShiftError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.arczonalshift.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondARCZonalShiftJSON(w, http.StatusOK, response)
	return true
}

func isARCZonalShiftJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "ArcZonalShiftService") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "ArcZonalShiftService")
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "arc-zonal-shift" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".arc-zonal-shift.") || strings.HasPrefix(host, "arc-zonal-shift.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(userAgent, "command#arc-zonal-shift") || strings.Contains(userAgent, " arc-zonal-shift/")
}

func parseARCZonalShiftTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "ArcZonalShiftService.") {
		return strings.TrimPrefix(target, "ArcZonalShiftService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseARCZonalShiftPayload(r *http.Request) (map[string]any, error) {
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

func respondARCZonalShiftJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondARCZonalShiftError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondARCZonalShiftJSON(w, status, arcZonalShiftError{Type: code, Message: msg})
}
