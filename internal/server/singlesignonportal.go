package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type singleSignOnPortalError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleSingleSignOnPortalRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSingleSignOnPortalRESTCandidate(r) {
		return false
	}

	action := parseSingleSignOnPortalAction(r.Method, rawRequestPath(r))
	if action == "" {
		respondSingleSignOnPortalError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := singleSignOnPortalOperationByName[action]; !known {
		respondSingleSignOnPortalError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	// Portal APIs are bearer-token based, but accept signed requests when present.
	if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
		ok, status, code, msg, _ := s.validateSigV4WithService(r, "sso")
		if !ok {
			respondSingleSignOnPortalError(w, status, code, msg)
			return true
		}
	}

	payload, err := parseSingleSignOnPortalPayload(r)
	if err != nil {
		respondSingleSignOnPortalError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}
	applySingleSignOnPortalQuery(payload, r)
	applySingleSignOnPortalToken(payload, r)

	response, status, code, msg := s.singlesignonportal.Handle(action, payload)
	if code != "" {
		respondSingleSignOnPortalError(w, status, code, msg)
		return true
	}
	if response == nil {
		response = map[string]any{}
	}
	respondSingleSignOnPortalJSON(w, http.StatusOK, response)
	return true
}

func isSingleSignOnPortalRESTCandidate(r *http.Request) bool {
	action := parseSingleSignOnPortalAction(r.Method, rawRequestPath(r))
	if action != "" {
		return true
	}

	token := strings.TrimSpace(r.Header.Get("x-amz-sso_bearer_token"))
	if token == "" {
		token = strings.TrimSpace(r.Header.Get("X-Amz-Sso_Bearer_Token"))
	}
	if token != "" {
		path := strings.TrimSpace(strings.SplitN(rawRequestPath(r), "?", 2)[0])
		return strings.HasPrefix(path, "/assignment/") ||
			strings.HasPrefix(path, "/federation/") ||
			path == "/logout"
	}
	return false
}

func parseSingleSignOnPortalAction(method, requestPath string) string {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	switch {
	case method == http.MethodGet && path == "/assignment/accounts":
		return "ListAccounts"
	case method == http.MethodGet && path == "/assignment/roles":
		return "ListAccountRoles"
	case method == http.MethodGet && path == "/federation/credentials":
		return "GetRoleCredentials"
	case method == http.MethodPost && path == "/logout":
		return "Logout"
	default:
		return ""
	}
}

func parseSingleSignOnPortalPayload(r *http.Request) (map[string]any, error) {
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if contentType != "" && !strings.Contains(contentType, "json") {
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

func applySingleSignOnPortalQuery(payload map[string]any, r *http.Request) {
	if payload == nil {
		return
	}
	query := r.URL.Query()
	if accountID := strings.TrimSpace(query.Get("account_id")); accountID != "" {
		payload["accountId"] = accountID
	}
	if roleName := strings.TrimSpace(query.Get("role_name")); roleName != "" {
		payload["roleName"] = roleName
	}
	if accessToken := strings.TrimSpace(query.Get("access_token")); accessToken != "" {
		payload["accessToken"] = accessToken
	}
	if nextToken := strings.TrimSpace(query.Get("next_token")); nextToken != "" {
		payload["nextToken"] = nextToken
	}
	if maxResult := strings.TrimSpace(query.Get("max_result")); maxResult != "" {
		payload["maxResult"] = maxResult
	}
}

func applySingleSignOnPortalToken(payload map[string]any, r *http.Request) {
	if payload == nil {
		return
	}
	token := strings.TrimSpace(r.Header.Get("x-amz-sso_bearer_token"))
	if token == "" {
		token = strings.TrimSpace(r.Header.Get("X-Amz-Sso_Bearer_Token"))
	}
	if token == "" {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			token = strings.TrimSpace(auth[len("Bearer "):])
		}
	}
	if token != "" {
		payload["accessToken"] = token
	}
}

func respondSingleSignOnPortalJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSingleSignOnPortalError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSingleSignOnPortalJSON(w, status, singleSignOnPortalError{Type: code, Message: msg})
}
