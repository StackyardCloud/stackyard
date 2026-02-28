package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type securityHubError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

var securityHubCLIActionPattern = regexp.MustCompile(`(?i)\bcommand#securityhub\.([a-z0-9-]+)`)

func (s *Server) handleSecurityHubRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSecurityHubRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "securityhub")
	if !ok {
		respondSecurityHubError(w, status, code, msg)
		return true
	}

	action := parseSecurityHubAction(r.Method, rawRequestPath(r), r.Header.Get("User-Agent"))
	if action == "" {
		respondSecurityHubError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := securityHubOperationByName[action]; !known {
		respondSecurityHubError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseSecurityHubPayload(r)
	if err != nil {
		respondSecurityHubError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.securityhub.Handle(action, payload, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondSecurityHubJSON(w, http.StatusOK, response)
	return true
}

func isSecurityHubRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
	default:
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "securityhub" {
		return false
	}
	if service == "securityhub" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".securityhub.") || strings.HasPrefix(host, "securityhub.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#securityhub.") || strings.Contains(userAgent, " securityhub/") {
		return true
	}

	action := parseSecurityHubAction(method, rawRequestPath(r), userAgent)
	return action != ""
}

func parseSecurityHubAction(method, requestPath, userAgent string) string {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])

	if strings.HasPrefix(path, "/_securityhub/") {
		action := strings.TrimSpace(strings.TrimPrefix(path, "/_securityhub/"))
		if action != "" {
			return action
		}
	}

	if action := parseSecurityHubPathAction(method, path); action != "" {
		return action
	}

	if matches := securityHubCLIActionPattern.FindStringSubmatch(strings.ToLower(userAgent)); len(matches) == 2 {
		return kebabToPascal(matches[1])
	}

	return ""
}

func parseSecurityHubPathAction(method, path string) string {
	switch {
	case strings.EqualFold(method, http.MethodGet) && path == "/accounts":
		return "DescribeHub"
	case strings.EqualFold(method, http.MethodGet) && path == "/productSubscriptions":
		return "ListEnabledProductsForImport"
	case strings.EqualFold(method, http.MethodGet) && path == "/organization/admin":
		return "ListOrganizationAdminAccounts"
	case strings.EqualFold(method, http.MethodPost) && path == "/standards/get":
		return "GetEnabledStandards"
	default:
		return ""
	}
}

func parseSecurityHubPayload(r *http.Request) (map[string]any, error) {
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

func respondSecurityHubJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSecurityHubError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondSecurityHubJSON(w, status, securityHubError{Type: code, Message: msg})
}

func kebabToPascal(name string) string {
	parts := strings.Split(strings.TrimSpace(name), "-")
	var out strings.Builder
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out.WriteString(strings.ToUpper(p[:1]))
		if len(p) > 1 {
			out.WriteString(p[1:])
		}
	}
	return out.String()
}

func securityHubStringFromQuery(values url.Values, key string) string {
	if values == nil {
		return ""
	}
	if v := strings.TrimSpace(values.Get(key)); v != "" {
		return v
	}
	for k, vals := range values {
		if !strings.EqualFold(k, key) || len(vals) == 0 {
			continue
		}
		if v := strings.TrimSpace(vals[0]); v != "" {
			return v
		}
	}
	return ""
}
