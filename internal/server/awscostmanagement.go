package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type awsCostManagementError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleAWSCostManagementJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isAWSCostManagementJSONCandidate(r) {
		return false
	}

	action := parseAWSCostManagementTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if action == "" {
		respondAWSCostManagementError(w, http.StatusBadRequest, "ValidationException", "missing X-Amz-Target")
		return true
	}
	if _, known := awsCostManagementOperationByName[action]; !known {
		respondAWSCostManagementError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	ok, status, code, msg, _ := s.validateSigV4(r)
	if !ok {
		respondAWSCostManagementError(w, status, code, msg)
		return true
	}

	payload, err := parseAWSCostManagementPayload(r)
	if err != nil {
		respondAWSCostManagementError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.awscostmanagement.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondAWSCostManagementJSON(w, http.StatusOK, response)
	return true
}

func isAWSCostManagementJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	targetPrefix := target
	if dot := strings.Index(targetPrefix, "."); dot > 0 {
		targetPrefix = targetPrefix[:dot]
	}
	if strings.EqualFold(targetPrefix, "AWSBillingAndCostManagement") {
		return true
	}

	svc := strings.TrimSpace(sigV4ServiceHint(r))
	if svc != "" {
		switch svc {
		case "ce", "budgets", "cur", "pricing", "billing", "taxsettings", "invoicing", "freetier", "bcm-data-exports", "bcm-pricing-calculator", "cost-optimization-hub":
			return true
		default:
			return false
		}
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".ce.") ||
		strings.Contains(host, ".budgets.") ||
		strings.Contains(host, ".cur.") ||
		strings.Contains(host, ".pricing.") ||
		strings.Contains(host, ".billing.") ||
		strings.Contains(host, ".invoicing.") ||
		strings.Contains(host, ".freetier.") ||
		strings.Contains(host, ".taxsettings.") {
		return true
	}

	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(ua, "command#ce") ||
		strings.Contains(ua, "command#budgets") ||
		strings.Contains(ua, "command#cur") ||
		strings.Contains(ua, "command#pricing") ||
		strings.Contains(ua, "command#taxsettings") ||
		strings.Contains(ua, "command#invoicing") ||
		strings.Contains(ua, "command#billing") {
		return true
	}

	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(ct, "application/x-amz-json") && target != "" {
		return strings.Contains(target, ".")
	}

	return false
}

func parseAWSCostManagementTarget(target string) string {
	if target == "" {
		return ""
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return strings.TrimSpace(target[dot+1:])
	}
	return strings.TrimSpace(target)
}

func parseAWSCostManagementPayload(r *http.Request) (map[string]any, error) {
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

func respondAWSCostManagementJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondAWSCostManagementError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondAWSCostManagementJSON(w, status, awsCostManagementError{Type: code, Message: msg})
}
