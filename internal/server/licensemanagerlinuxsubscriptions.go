package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type licenseManagerLinuxSubscriptionsError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleLicenseManagerLinuxSubscriptionsRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isLicenseManagerLinuxSubscriptionsRESTRouterCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "license-manager-linux-subscriptions")
	if !ok {
		respondLicenseManagerLinuxSubscriptionsError(w, status, code, msg)
		return true
	}

	action, pathParams := parseLicenseManagerLinuxSubscriptionsRoute(r.Method, rawRequestPath(r))
	if action == "" {
		respondLicenseManagerLinuxSubscriptionsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}
	if _, known := licenseManagerLinuxSubscriptionsOperationByName[action]; !known {
		respondLicenseManagerLinuxSubscriptionsError(w, http.StatusBadRequest, "ValidationException", "unknown action")
		return true
	}

	payload, err := parseLicenseManagerLinuxSubscriptionsPayload(r)
	if err != nil {
		respondLicenseManagerLinuxSubscriptionsError(w, http.StatusBadRequest, "ValidationException", "invalid JSON body")
		return true
	}

	response := s.licensemanagerlinuxsubscriptions.Handle(action, payload, pathParams, r.URL.Query())
	if response == nil {
		response = map[string]any{}
	}
	respondLicenseManagerLinuxSubscriptionsJSON(w, http.StatusOK, response)
	return true
}

func isLicenseManagerLinuxSubscriptionsRESTRouterCandidate(r *http.Request) bool {
	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method != http.MethodPost && method != http.MethodGet && method != http.MethodPut && method != http.MethodDelete {
		return false
	}

	service := strings.TrimSpace(sigV4ServiceHint(r))
	if service != "" && service != "license-manager-linux-subscriptions" {
		return false
	}
	if service == "license-manager-linux-subscriptions" {
		return true
	}

	host := strings.ToLower(strings.TrimSpace(r.Host))
	if strings.Contains(host, ".license-manager-linux-subscriptions.") || strings.HasPrefix(host, "license-manager-linux-subscriptions.") {
		return true
	}

	userAgent := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	if strings.Contains(userAgent, "command#license-manager-linux-subscriptions") || strings.Contains(userAgent, " license-manager-linux-subscriptions/") {
		return true
	}

	path := strings.TrimSpace(rawRequestPath(r))
	return strings.HasPrefix(path, "/subscription/") || strings.HasPrefix(path, "/tags/")
}

func parseLicenseManagerLinuxSubscriptionsRoute(method, requestPath string) (string, map[string]string) {
	path := strings.TrimSpace(strings.SplitN(requestPath, "?", 2)[0])
	if path == "" {
		path = "/"
	}
	for _, op := range licenseManagerLinuxSubscriptionsOperations {
		if !strings.EqualFold(op.Method, method) {
			continue
		}
		params, ok := licenseManagerLinuxSubscriptionsPathParams(op.URI, path)
		if ok {
			return op.Name, params
		}
	}
	return "", nil
}

func licenseManagerLinuxSubscriptionsPathParams(template, actual string) (map[string]string, bool) {
	tSegs := licenseManagerLinuxSubscriptionsSplitPath(template)
	aSegs := licenseManagerLinuxSubscriptionsSplitPath(actual)
	if len(tSegs) != len(aSegs) {
		return nil, false
	}

	params := map[string]string{}
	for i := range tSegs {
		t := tSegs[i]
		a := aSegs[i]
		if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(t, "{"), "}"))
			if name == "" {
				return nil, false
			}
			value, err := url.PathUnescape(a)
			if err != nil {
				value = a
			}
			params[name] = value
			continue
		}
		if t != a {
			return nil, false
		}
	}
	return params, true
}

func licenseManagerLinuxSubscriptionsSplitPath(path string) []string {
	path = strings.TrimSpace(strings.SplitN(path, "?", 2)[0])
	if path == "" || path == "/" {
		return nil
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseLicenseManagerLinuxSubscriptionsPayload(r *http.Request) (map[string]any, error) {
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

func respondLicenseManagerLinuxSubscriptionsJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondLicenseManagerLinuxSubscriptionsError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondLicenseManagerLinuxSubscriptionsJSON(w, status, licenseManagerLinuxSubscriptionsError{Type: code, Message: msg})
}
