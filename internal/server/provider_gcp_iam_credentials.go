package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) handleGCPIAMCredentialsRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPIAMCredentialsPath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.iam.credentials.v1.IAMCredentials/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	if r.Method != http.MethodPost {
		return false
	}

	serviceAccount, action, ok := parseGCPIAMCredentialsActionPath(path)
	if !ok {
		return false
	}

	body, valid := decodeGCPIAMCredentialsJSONBody(w, r, path)
	if !valid {
		return true
	}

	switch action {
	case "generateAccessToken":
		scope, _ := body["scope"].([]any)
		if len(scope) == 0 {
			respondGCPIAMCredentialsInvalidArgument(w, path, "scope is required")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"accessToken": "ya29.stackyard.access.token",
			"expireTime":  "2030-01-01T00:00:00Z",
		})
		return true
	case "generateIdToken":
		if strings.TrimSpace(stringFromMap(body, "audience")) == "" {
			respondGCPIAMCredentialsInvalidArgument(w, path, "audience is required")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.stackyard",
		})
		return true
	case "signBlob":
		if strings.TrimSpace(stringFromMap(body, "payload")) == "" {
			respondGCPIAMCredentialsInvalidArgument(w, path, "payload is required")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"keyId":      "key-1",
			"signedBlob": "c2lnbmVkLWJ5LXN0YWNreWFyZA==",
			"name":       serviceAccount,
		})
		return true
	case "signJwt":
		if strings.TrimSpace(stringFromMap(body, "payload")) == "" {
			respondGCPIAMCredentialsInvalidArgument(w, path, "payload is required")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{
			"keyId":     "key-1",
			"signedJwt": "header.payload.signature",
			"name":      serviceAccount,
		})
		return true
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func isGCPIAMCredentialsPath(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.iam.credentials.v1.IAMCredentials/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v1/") {
		return false
	}

	return strings.Contains(path, ":generateAccessToken") ||
		strings.Contains(path, ":generateIdToken") ||
		strings.Contains(path, ":signBlob") ||
		strings.Contains(path, ":signJwt")
}

func parseGCPIAMCredentialsActionPath(path string) (serviceAccount, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return "", "", false
	}
	if parts[4] != "serviceAccounts" {
		return "", "", false
	}
	accountAction := normalizeGCPIAMCredentialsActionSegment(parts[5])
	account, op, hasAction := strings.Cut(accountAction, ":")
	if !hasAction || strings.TrimSpace(account) == "" || strings.TrimSpace(op) == "" {
		return "", "", false
	}
	serviceAccount = "projects/" + strings.TrimSpace(parts[3]) + "/serviceAccounts/" + account
	if len(parts) > 6 {
		return "", "", false
	}
	return serviceAccount, op, true
}

func decodeGCPIAMCredentialsJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPIAMCredentialsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func normalizeGCPIAMCredentialsActionSegment(raw string) string {
	segment := strings.TrimSpace(raw)
	segment = strings.ReplaceAll(segment, "%3A", ":")
	segment = strings.ReplaceAll(segment, "%3a", ":")
	return segment
}

func respondGCPIAMCredentialsInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
