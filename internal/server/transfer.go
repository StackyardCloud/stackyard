package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type transferError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleTransferJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isTransferJSONCandidate(r) {
		return false
	}

	action := parseTransferTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondTransferError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, ok := transferOperationByName[action]; !ok {
		respondTransferError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "transfer")
	if !ok {
		altOK, _, _, _, _ := s.validateSigV4WithService(r, "transferservice")
		if !altOK {
			respondTransferError(w, status, code, msg)
			return true
		}
	}

	payload, err := parseTransferPayload(r)
	if err != nil {
		respondTransferError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.transfer.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondTransferJSON(w, http.StatusOK, response)
	return true
}

func isTransferJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.Contains(target, "TransferService.") {
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.1") {
		if target == "" {
			service := strings.TrimSpace(sigV4ServiceHint(r))
			return service == "transfer" || service == "transferservice"
		}
		return strings.Contains(target, "TransferService.")
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "transfer" || service == "transferservice" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".transfer.") || strings.HasPrefix(host, "transfer.") {
		return true
	}

	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "command#transfer") || strings.Contains(ua, " transfer/")
}

func parseTransferTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "TransferService.") {
		return strings.TrimPrefix(target, "TransferService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseTransferPayload(r *http.Request) (map[string]any, error) {
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

func respondTransferJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondTransferError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondTransferJSON(w, status, transferError{Type: code, Message: msg})
}
