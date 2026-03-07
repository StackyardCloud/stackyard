package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) handleGCPIAMRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPIAMPath(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.iam.v1.IAMPolicy/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	if r.Method != http.MethodPost {
		return false
	}

	resource, action, ok := parseGCPIAMActionPath(path)
	if !ok {
		return false
	}

	body, valid := decodeGCPIAMJSONBody(w, r, path)
	if !valid {
		return true
	}

	switch action {
	case "getIamPolicy":
		respondJSON(w, http.StatusOK, gcpIAMPolicy(resource))
		return true
	case "setIamPolicy":
		policy := gcpIAMBodyMap(body, "policy")
		if len(policy) == 0 {
			respondGCPIAMInvalidArgument(w, path, "policy is required")
			return true
		}
		respondJSON(w, http.StatusOK, gcpIAMPolicy(resource))
		return true
	case "testIamPermissions":
		permissions, _ := body["permissions"].([]any)
		if len(permissions) == 0 {
			respondGCPIAMInvalidArgument(w, path, "permissions is required")
			return true
		}
		result := make([]string, 0, len(permissions))
		for _, permission := range permissions {
			if asText, ok := permission.(string); ok && strings.TrimSpace(asText) != "" {
				result = append(result, asText)
			}
		}
		if len(result) == 0 {
			respondGCPIAMInvalidArgument(w, path, "permissions is required")
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{"permissions": result})
		return true
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func isGCPIAMPath(path string) bool {
	if _, _, _, ok := parseGCPSecretManagerSecretIAMActionPath(normalizeGCPSecretManagerPath(path)); ok {
		return false
	}
	if _, _, _, _, ok := parseGCPSecureSourceManagerRepositoryIAMPath(normalizeGCPSecureSourceManagerPath(path)); ok {
		return false
	}
	if strings.HasPrefix(path, "/gcp/google.iam.v1.IAMPolicy/") {
		return true
	}
	resource, action, ok := parseGCPIAMActionPath(path)
	if !ok {
		return false
	}
	switch action {
	case "getIamPolicy", "setIamPolicy", "testIamPermissions":
		return isGCPIAMCoreResource(resource)
	default:
		return false
	}
}

func parseGCPIAMActionPath(path string) (resource, action string, ok bool) {
	trimmed := strings.TrimPrefix(path, "/gcp/v1/")
	if trimmed == path || strings.TrimSpace(trimmed) == "" {
		return "", "", false
	}
	resourcePart, actionPart, hasAction := strings.Cut(normalizeGCPIAMActionSegment(trimmed), ":")
	if !hasAction || strings.TrimSpace(resourcePart) == "" || strings.TrimSpace(actionPart) == "" {
		return "", "", false
	}
	return resourcePart, actionPart, true
}

func decodeGCPIAMJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPIAMInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpIAMBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func normalizeGCPIAMActionSegment(raw string) string {
	segment := strings.TrimSpace(raw)
	segment = strings.ReplaceAll(segment, "%3A", ":")
	segment = strings.ReplaceAll(segment, "%3a", ":")
	return segment
}

func isGCPIAMCoreResource(resource string) bool {
	resource = strings.Trim(resource, "/")
	parts := strings.Split(resource, "/")
	if len(parts) != 2 {
		return false
	}
	if strings.TrimSpace(parts[1]) == "" {
		return false
	}
	switch parts[0] {
	case "projects", "folders", "organizations":
		return true
	default:
		return false
	}
}

func gcpIAMPolicy(resource string) map[string]any {
	return map[string]any{
		"version": 1,
		"bindings": []any{
			map[string]any{
				"role":    "roles/viewer",
				"members": []string{"user:alice@example.com"},
			},
		},
		"etag":     "c3RhY2t5YXJk",
		"resource": resource,
	}
}

func respondGCPIAMInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
