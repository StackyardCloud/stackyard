package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type paymentCryptographyError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handlePaymentCryptographyJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isPaymentCryptographyJSONCandidate(r) {
		return false
	}

	action := parsePaymentCryptographyTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondPaymentCryptographyError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := paymentCryptographyOperationByName[action]; !known {
		respondPaymentCryptographyError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "payment-cryptography")
	if !ok {
		respondPaymentCryptographyError(w, status, code, msg)
		return true
	}

	payload, err := parsePaymentCryptographyPayload(r)
	if err != nil {
		respondPaymentCryptographyError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.paymentcryptography.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondPaymentCryptographyJSON(w, http.StatusOK, response)
	return true
}

func isPaymentCryptographyJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "PaymentCryptographyControlPlane.") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(target, "PaymentCryptographyControlPlane.")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "payment-cryptography" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".payment-cryptography.") || strings.HasPrefix(host, "payment-cryptography.")
}

func parsePaymentCryptographyTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "PaymentCryptographyControlPlane.") {
		return strings.TrimPrefix(target, "PaymentCryptographyControlPlane.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parsePaymentCryptographyPayload(r *http.Request) (map[string]any, error) {
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

func respondPaymentCryptographyJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondPaymentCryptographyError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondPaymentCryptographyJSON(w, status, paymentCryptographyError{Type: code, Message: msg})
}
