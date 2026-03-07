package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPIAMV2Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPIAMV2Path(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.iam.v2.Policies/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPIAMV2ListPolicies(w, r, path) {
			return true
		}
		if handleGCPIAMV2GetPolicy(w, path) {
			return true
		}
		if handleGCPIAMV2GetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPIAMV2CreatePolicy(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPut:
		if handleGCPIAMV2UpdatePolicy(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPIAMV2DeletePolicy(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPIAMV2Path(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.iam.v2.Policies/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v2/") {
		return false
	}
	if strings.HasPrefix(path, "/gcp/v2/operations/") {
		return true
	}
	return strings.HasPrefix(path, "/gcp/v2/policies/") && strings.Contains(path, "/denypolicies")
}

func handleGCPIAMV2ListPolicies(w http.ResponseWriter, r *http.Request, path string) bool {
	attachment, policyID, ok := parseGCPIAMV2PolicyPath(path)
	if !ok || policyID != "" {
		return false
	}
	pageSize, start, valid := parseGCPIAMV2Pagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpIAMV2Policy(attachment, "deny-read-access")}
	return respondGCPIAMV2List(w, "policies", items, pageSize, start, path)
}

func handleGCPIAMV2GetPolicy(w http.ResponseWriter, path string) bool {
	attachment, policyID, ok := parseGCPIAMV2PolicyPath(path)
	if !ok || policyID == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpIAMV2Policy(attachment, policyID))
	return true
}

func handleGCPIAMV2CreatePolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	attachment, policyID, ok := parseGCPIAMV2PolicyPath(path)
	if !ok || policyID != "" {
		return false
	}
	body, valid := decodeGCPIAMV2JSONBody(w, r, path)
	if !valid {
		return true
	}
	policy := gcpIAMV2BodyMap(body, "policy")
	if len(policy) == 0 {
		respondGCPIAMV2InvalidArgument(w, path, "policy is required")
		return true
	}
	policyID = strings.TrimSpace(r.URL.Query().Get("policyId"))
	if policyID == "" {
		respondGCPIAMV2InvalidArgument(w, path, "policyId is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpIAMV2Operation("create-policy-"+attachment+"-"+policyID))
	return true
}

func handleGCPIAMV2UpdatePolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	attachment, policyID, ok := parseGCPIAMV2PolicyPath(path)
	if !ok || policyID == "" {
		return false
	}
	body, valid := decodeGCPIAMV2JSONBody(w, r, path)
	if !valid {
		return true
	}
	policy := gcpIAMV2BodyMap(body, "policy")
	if len(policy) == 0 {
		respondGCPIAMV2InvalidArgument(w, path, "policy is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpIAMV2Operation("update-policy-"+attachment+"-"+policyID))
	return true
}

func handleGCPIAMV2DeletePolicy(w http.ResponseWriter, path string) bool {
	attachment, policyID, ok := parseGCPIAMV2PolicyPath(path)
	if !ok || policyID == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpIAMV2Operation("delete-policy-"+attachment+"-"+policyID))
	return true
}

func handleGCPIAMV2GetOperation(w http.ResponseWriter, path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "operations" || strings.TrimSpace(parts[3]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpIAMV2Operation(parts[3]))
	return true
}

func parseGCPIAMV2PolicyPath(path string) (attachment, policyID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "policies" {
		return "", "", false
	}
	deniesIndex := -1
	for i := 3; i < len(parts); i++ {
		if parts[i] == "denypolicies" {
			deniesIndex = i
			break
		}
	}
	if deniesIndex < 0 {
		return "", "", false
	}
	attachment = strings.Join(parts[3:deniesIndex], "/")
	if strings.TrimSpace(attachment) == "" {
		return "", "", false
	}
	if len(parts) == deniesIndex+1 {
		return attachment, "", true
	}
	if len(parts) == deniesIndex+2 && strings.TrimSpace(parts[deniesIndex+1]) != "" {
		return attachment, parts[deniesIndex+1], true
	}
	return "", "", false
}

func parseGCPIAMV2Pagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPIAMV2InvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPIAMV2InvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPIAMV2List(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPIAMV2InvalidArgument(w, path, "pageToken is out of range")
		return true
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func decodeGCPIAMV2JSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPIAMV2InvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpIAMV2BodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func gcpIAMV2Policy(attachment, policyID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("policies/%s/denypolicies/%s", attachment, policyID),
		"displayName": "stackyard deny policy",
		"uid":         "deny-uid-1",
		"kind":        "DenyPolicy",
		"rules": []any{
			map[string]any{
				"description":       "deny broad project read",
				"deniedPrincipals":  []string{"principalSet://goog/public:all"},
				"deniedPermissions": []string{"resourcemanager.googleapis.com/projects.get"},
			},
		},
	}
}

func gcpIAMV2Operation(operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("operations/%s", operationID),
		"done": true,
	}
}

func respondGCPIAMV2InvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
