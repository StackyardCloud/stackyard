package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type networkFirewallError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleNetworkFirewallJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isNetworkFirewallJSONCandidate(r) {
		return false
	}

	action := parseNetworkFirewallTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondNetworkFirewallError(w, http.StatusBadRequest, "InvalidRequestException", "missing X-Amz-Target")
		return true
	}
	if _, known := networkFirewallOperationByName[action]; !known {
		respondNetworkFirewallError(w, http.StatusBadRequest, "InvalidRequestException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "network-firewall")
	if !ok {
		respondNetworkFirewallError(w, status, code, msg)
		return true
	}

	payload, err := parseNetworkFirewallPayload(r)
	if err != nil {
		respondNetworkFirewallError(w, http.StatusBadRequest, "InvalidRequestException", "invalid JSON body")
		return true
	}

	response := s.networkfirewall.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondNetworkFirewallJSON(w, http.StatusOK, response)
	return true
}

func isNetworkFirewallJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "NetworkFirewall_20201112.") {
		return true
	}
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(contentType, "application/x-amz-json-1.0") {
		return strings.HasPrefix(target, "NetworkFirewall_20201112.")
	}
	if strings.TrimSpace(sigV4ServiceHint(r)) == "network-firewall" {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(r.Host))
	return strings.Contains(host, ".network-firewall.") || strings.HasPrefix(host, "network-firewall.")
}

func parseNetworkFirewallTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "NetworkFirewall_20201112.") {
		return strings.TrimPrefix(target, "NetworkFirewall_20201112.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func parseNetworkFirewallPayload(r *http.Request) (map[string]any, error) {
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

func respondNetworkFirewallJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondNetworkFirewallError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondNetworkFirewallJSON(w, status, networkFirewallError{Type: code, Message: msg})
}
