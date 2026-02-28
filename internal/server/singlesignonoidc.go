package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type singleSignOnOIDCError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleSingleSignOnOIDCJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSingleSignOnOIDCJSONCandidate(r) {
		return false
	}

	action := parseSingleSignOnOIDCAction(r)
	if action == "" {
		respondSingleSignOnOIDCError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := singleSignOnOIDCOperationByName[action]; !known {
		respondSingleSignOnOIDCError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	// RegisterClient, StartDeviceAuthorization, and CreateToken are typically unsigned.
	// CreateTokenWithIAM is signed with sso-oauth by AWS CLI. Accept either when present.
	if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
		ok, _, _, _, _ := s.validateSigV4WithService(r, "sso-oauth")
		if !ok {
			ok2, status2, code2, msg2, _ := s.validateSigV4WithService(r, "sso-oidc")
			if !ok2 {
				respondSingleSignOnOIDCError(w, status2, code2, msg2)
				return true
			}
		}
	}

	payload, err := parseSingleSignOnOIDCPayload(r)
	if err != nil {
		respondSingleSignOnOIDCError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.singlesignonoidc.Handle(action, payload)
	if response == nil {
		response = map[string]any{}
	}
	respondSingleSignOnOIDCJSON(w, http.StatusOK, response)
	return true
}

func isSingleSignOnOIDCJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	if action := parseSingleSignOnOIDCAction(r); action != "" {
		return true
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service == "sso-oidc" || service == "sso-oauth" {
		return true
	}
	if service != "" && service != "sso-oidc" && service != "sso-oauth" {
		return false
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".sso-oidc.") || strings.HasPrefix(host, "sso-oidc.") || strings.Contains(host, ".sso.") || strings.HasPrefix(host, "sso.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(userAgent, "command#sso-oidc") || strings.Contains(userAgent, " sso-oidc/")
}

func parseSingleSignOnOIDCTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	const prefix = "AWSSSOOIDCService."
	if strings.HasPrefix(target, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(target, prefix))
	}

	// Some clients may send the bare action name without the service prefix.
	// Only accept known OIDC actions to avoid cross-service router collisions.
	if _, known := singleSignOnOIDCOperationByName[target]; known {
		return target
	}

	return ""
}

func parseSingleSignOnOIDCAction(r *http.Request) string {
	target := parseSingleSignOnOIDCTarget(strings.TrimSpace(r.Header.Get("X-Amz-Target")))
	if target != "" {
		return target
	}

	if r.Method != http.MethodPost {
		return ""
	}
	path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
	switch strings.TrimRight(path, "/") {
	case "/client/register":
		return "RegisterClient"
	case "/device_authorization":
		return "StartDeviceAuthorization"
	case "/token":
		if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("aws_iam")), "t") {
			return "CreateTokenWithIAM"
		}
		return "CreateToken"
	default:
		return ""
	}
}

func parseSingleSignOnOIDCPayload(r *http.Request) (map[string]any, error) {
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

func respondSingleSignOnOIDCJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSingleSignOnOIDCError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSingleSignOnOIDCJSON(w, status, singleSignOnOIDCError{Type: code, Message: msg})
}
