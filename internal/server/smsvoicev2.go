package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type smsVoiceV2Error struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleSMSVoiceV2JSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSMSVoiceV2JSONCandidate(r) {
		return false
	}

	action := parseSMSVoiceV2Target(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondSMSVoiceV2Error(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := smsVoiceV2OperationByName[action]; !known {
		respondSMSVoiceV2Error(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "sms-voice")
	if !ok {
		respondSMSVoiceV2Error(w, status, code, msg)
		return true
	}

	payload, err := parseSMSVoiceV2Payload(r)
	if err != nil {
		respondSMSVoiceV2Error(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.smsvoicev2.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondSMSVoiceV2JSON(w, http.StatusOK, response)
	return true
}

func isSMSVoiceV2JSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "PinpointSMSVoiceV2.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.Contains(target, "PinpointSMSVoiceV2")
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "sms-voice" || service == "sms-voice-v2" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".sms-voice.") || strings.HasPrefix(host, "sms-voice.")
}

func parseSMSVoiceV2Target(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "PinpointSMSVoiceV2.") {
		return strings.TrimPrefix(target, "PinpointSMSVoiceV2.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseSMSVoiceV2Payload(r *http.Request) (map[string]any, error) {
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

func respondSMSVoiceV2JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSMSVoiceV2Error(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSMSVoiceV2JSON(w, status, smsVoiceV2Error{Type: code, Message: msg})
}
