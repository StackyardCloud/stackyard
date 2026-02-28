package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type workspacesAppStream2Error struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleWorkSpacesAppStream2JSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isWorkSpacesAppStream2JSONCandidate(r) {
		return false
	}

	action := parseWorkSpacesAppStream2Target(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondWorkSpacesAppStream2Error(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := workspacesAppStream2OperationByName[action]; !known {
		respondWorkSpacesAppStream2Error(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "appstream")
	if !ok {
		respondWorkSpacesAppStream2Error(w, status, code, msg)
		return true
	}

	payload, err := parseWorkSpacesAppStream2Payload(r)
	if err != nil {
		respondWorkSpacesAppStream2Error(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.workspacesappstream2.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondWorkSpacesAppStream2JSON(w, http.StatusOK, response)
	return true
}

func isWorkSpacesAppStream2JSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "PhotonAdminProxyService.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "PhotonAdminProxyService.")
	}

	if strings.TrimSpace(sigV4ServiceHint(r)) == "appstream" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".appstream.") || strings.HasPrefix(host, "appstream.")
}

func parseWorkSpacesAppStream2Target(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "PhotonAdminProxyService.") {
		return strings.TrimPrefix(target, "PhotonAdminProxyService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseWorkSpacesAppStream2Payload(r *http.Request) (map[string]any, error) {
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

func respondWorkSpacesAppStream2JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondWorkSpacesAppStream2Error(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondWorkSpacesAppStream2JSON(w, status, workspacesAppStream2Error{Type: code, Message: msg})
}
