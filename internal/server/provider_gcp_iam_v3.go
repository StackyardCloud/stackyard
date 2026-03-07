package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPIAMV3Router(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPIAMV3Path(path) {
		return false
	}

	if strings.HasPrefix(path, "/gcp/google.iam.v3.PolicyBindings/") ||
		strings.HasPrefix(path, "/gcp/google.iam.v3.PrincipalAccessBoundaryPolicies/") {
		switch r.Method {
		case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		default:
			return false
		}
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPIAMV3ListPolicyBindings(w, r, path) {
			return true
		}
		if handleGCPIAMV3GetPolicyBinding(w, path) {
			return true
		}
		if handleGCPIAMV3SearchTargetPolicyBindings(w, r, path) {
			return true
		}
		if handleGCPIAMV3ListPrincipalAccessBoundaryPolicies(w, r, path) {
			return true
		}
		if handleGCPIAMV3GetPrincipalAccessBoundaryPolicy(w, path) {
			return true
		}
		if handleGCPIAMV3SearchPolicyBindings(w, r, path) {
			return true
		}
		if handleGCPIAMV3GetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPIAMV3CreatePolicyBinding(w, r, path) {
			return true
		}
		if handleGCPIAMV3CreatePrincipalAccessBoundaryPolicy(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPIAMV3UpdatePolicyBinding(w, r, path) {
			return true
		}
		if handleGCPIAMV3UpdatePrincipalAccessBoundaryPolicy(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPIAMV3DeletePolicyBinding(w, path) {
			return true
		}
		if handleGCPIAMV3DeletePrincipalAccessBoundaryPolicy(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func isGCPIAMV3Path(path string) bool {
	if strings.HasPrefix(path, "/gcp/google.iam.v3.PolicyBindings/") ||
		strings.HasPrefix(path, "/gcp/google.iam.v3.PrincipalAccessBoundaryPolicies/") {
		return true
	}
	if !strings.HasPrefix(path, "/gcp/v3/") {
		return false
	}
	if strings.HasPrefix(path, "/gcp/v3/operations/") {
		return true
	}

	return strings.Contains(path, "/policyBindings") ||
		strings.Contains(path, "/principalAccessBoundaryPolicies") ||
		strings.Contains(path, ":searchTargetPolicyBindings") ||
		strings.Contains(path, ":searchPolicyBindings")
}

func handleGCPIAMV3ListPolicyBindings(w http.ResponseWriter, r *http.Request, path string) bool {
	scopeType, scopeID, location, tail, ok := parseGCPIAMV3ScopeTail(path)
	if !ok || len(tail) != 1 || tail[0] != "policyBindings" {
		return false
	}
	pageSize, start, valid := parseGCPIAMV3Pagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpIAMV3PolicyBinding(scopeType, scopeID, location, "binding-a")}
	return respondGCPIAMV3List(w, "policyBindings", items, pageSize, start, path)
}

func handleGCPIAMV3GetPolicyBinding(w http.ResponseWriter, path string) bool {
	scopeType, scopeID, location, tail, ok := parseGCPIAMV3ScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "policyBindings" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpIAMV3PolicyBinding(scopeType, scopeID, location, tail[1]))
	return true
}

func handleGCPIAMV3CreatePolicyBinding(w http.ResponseWriter, r *http.Request, path string) bool {
	scopeType, scopeID, location, tail, ok := parseGCPIAMV3ScopeTail(path)
	if !ok || len(tail) != 1 || tail[0] != "policyBindings" {
		return false
	}
	body, valid := decodeGCPIAMV3JSONBody(w, r, path)
	if !valid {
		return true
	}
	binding := gcpIAMV3BodyMap(body, "policyBinding")
	if len(binding) == 0 {
		respondGCPIAMV3InvalidArgument(w, path, "policyBinding is required")
		return true
	}
	bindingID := strings.TrimSpace(stringFromMap(body, "policyBindingId"))
	if bindingID == "" {
		bindingID = strings.TrimSpace(r.URL.Query().Get("policyBindingId"))
	}
	if bindingID == "" {
		bindingID = "binding-a"
	}
	respondJSON(w, http.StatusOK, gcpIAMV3Operation("create-policy-binding-"+scopeType+"-"+scopeID+"-"+location+"-"+bindingID))
	return true
}

func handleGCPIAMV3UpdatePolicyBinding(w http.ResponseWriter, r *http.Request, path string) bool {
	scopeType, scopeID, location, tail, ok := parseGCPIAMV3ScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "policyBindings" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	body, valid := decodeGCPIAMV3JSONBody(w, r, path)
	if !valid {
		return true
	}
	binding := gcpIAMV3BodyMap(body, "policyBinding")
	if len(binding) == 0 {
		respondGCPIAMV3InvalidArgument(w, path, "policyBinding is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpIAMV3Operation("update-policy-binding-"+scopeType+"-"+scopeID+"-"+location+"-"+tail[1]))
	return true
}

func handleGCPIAMV3DeletePolicyBinding(w http.ResponseWriter, path string) bool {
	scopeType, scopeID, location, tail, ok := parseGCPIAMV3ScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "policyBindings" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpIAMV3Operation("delete-policy-binding-"+scopeType+"-"+scopeID+"-"+location+"-"+tail[1]))
	return true
}

func handleGCPIAMV3SearchTargetPolicyBindings(w http.ResponseWriter, r *http.Request, path string) bool {
	scopeType, scopeID, location, tail, ok := parseGCPIAMV3ScopeTail(path)
	if !ok || len(tail) != 1 {
		return false
	}
	segment, action, hasAction := strings.Cut(normalizeGCPIAMV3ActionSegment(tail[0]), ":")
	if !hasAction || segment != "policyBindings" || action != "searchTargetPolicyBindings" {
		return false
	}
	if strings.TrimSpace(r.URL.Query().Get("target")) == "" {
		respondGCPIAMV3InvalidArgument(w, path, "target is required")
		return true
	}
	pageSize, start, valid := parseGCPIAMV3Pagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpIAMV3PolicyBinding(scopeType, scopeID, location, "binding-a")}
	return respondGCPIAMV3List(w, "policyBindings", items, pageSize, start, path)
}

func handleGCPIAMV3ListPrincipalAccessBoundaryPolicies(w http.ResponseWriter, r *http.Request, path string) bool {
	scopeType, scopeID, location, tail, ok := parseGCPIAMV3ScopeTail(path)
	if !ok || len(tail) != 1 || tail[0] != "principalAccessBoundaryPolicies" {
		return false
	}
	pageSize, start, valid := parseGCPIAMV3Pagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpIAMV3PrincipalAccessBoundaryPolicy(scopeType, scopeID, location, "pab-a")}
	return respondGCPIAMV3List(w, "principalAccessBoundaryPolicies", items, pageSize, start, path)
}

func handleGCPIAMV3GetPrincipalAccessBoundaryPolicy(w http.ResponseWriter, path string) bool {
	scopeType, scopeID, location, tail, ok := parseGCPIAMV3ScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "principalAccessBoundaryPolicies" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpIAMV3PrincipalAccessBoundaryPolicy(scopeType, scopeID, location, tail[1]))
	return true
}

func handleGCPIAMV3CreatePrincipalAccessBoundaryPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	scopeType, scopeID, location, tail, ok := parseGCPIAMV3ScopeTail(path)
	if !ok || len(tail) != 1 || tail[0] != "principalAccessBoundaryPolicies" {
		return false
	}
	body, valid := decodeGCPIAMV3JSONBody(w, r, path)
	if !valid {
		return true
	}
	policy := gcpIAMV3BodyMap(body, "principalAccessBoundaryPolicy")
	if len(policy) == 0 {
		respondGCPIAMV3InvalidArgument(w, path, "principalAccessBoundaryPolicy is required")
		return true
	}
	policyID := strings.TrimSpace(stringFromMap(body, "principalAccessBoundaryPolicyId"))
	if policyID == "" {
		policyID = strings.TrimSpace(r.URL.Query().Get("principalAccessBoundaryPolicyId"))
	}
	if policyID == "" {
		policyID = "pab-a"
	}
	respondJSON(w, http.StatusOK, gcpIAMV3Operation("create-pab-policy-"+scopeType+"-"+scopeID+"-"+location+"-"+policyID))
	return true
}

func handleGCPIAMV3UpdatePrincipalAccessBoundaryPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	scopeType, scopeID, location, tail, ok := parseGCPIAMV3ScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "principalAccessBoundaryPolicies" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	body, valid := decodeGCPIAMV3JSONBody(w, r, path)
	if !valid {
		return true
	}
	policy := gcpIAMV3BodyMap(body, "principalAccessBoundaryPolicy")
	if len(policy) == 0 {
		respondGCPIAMV3InvalidArgument(w, path, "principalAccessBoundaryPolicy is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpIAMV3Operation("update-pab-policy-"+scopeType+"-"+scopeID+"-"+location+"-"+tail[1]))
	return true
}

func handleGCPIAMV3DeletePrincipalAccessBoundaryPolicy(w http.ResponseWriter, path string) bool {
	scopeType, scopeID, location, tail, ok := parseGCPIAMV3ScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "principalAccessBoundaryPolicies" || strings.TrimSpace(tail[1]) == "" || strings.Contains(tail[1], ":") {
		return false
	}
	respondJSON(w, http.StatusOK, gcpIAMV3Operation("delete-pab-policy-"+scopeType+"-"+scopeID+"-"+location+"-"+tail[1]))
	return true
}

func handleGCPIAMV3SearchPolicyBindings(w http.ResponseWriter, r *http.Request, path string) bool {
	scopeType, scopeID, location, tail, ok := parseGCPIAMV3ScopeTail(path)
	if !ok || len(tail) != 2 || tail[0] != "principalAccessBoundaryPolicies" {
		return false
	}
	policyID, action, hasAction := strings.Cut(normalizeGCPIAMV3ActionSegment(tail[1]), ":")
	if !hasAction || strings.TrimSpace(policyID) == "" || action != "searchPolicyBindings" {
		return false
	}
	pageSize, start, valid := parseGCPIAMV3Pagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{gcpIAMV3PolicyBinding(scopeType, scopeID, location, "binding-a")}
	return respondGCPIAMV3List(w, "policyBindings", items, pageSize, start, path)
}

func handleGCPIAMV3GetOperation(w http.ResponseWriter, path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v3" || parts[2] != "operations" || strings.TrimSpace(parts[3]) == "" {
		return false
	}
	respondJSON(w, http.StatusOK, gcpIAMV3Operation(parts[3]))
	return true
}

func parseGCPIAMV3ScopeTail(path string) (scopeType, scopeID, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v3" || parts[4] != "locations" {
		return "", "", "", nil, false
	}
	scopeType = strings.TrimSpace(parts[2])
	scopeID = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if (scopeType != "projects" && scopeType != "organizations") || scopeID == "" || location == "" {
		return "", "", "", nil, false
	}
	if len(parts) == 6 {
		return scopeType, scopeID, location, nil, true
	}
	return scopeType, scopeID, location, parts[6:], true
}

func parseGCPIAMV3Pagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPIAMV3InvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken == "" {
		return pageSize, 0, true
	}
	start, err = parseOptionalNonNegativeInt(pageToken)
	if err != nil {
		respondGCPIAMV3InvalidArgument(w, path, "pageToken must be a non-negative integer offset")
		return 0, 0, false
	}
	return pageSize, start, true
}

func respondGCPIAMV3List(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPIAMV3InvalidArgument(w, path, "pageToken is out of range")
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

func decodeGCPIAMV3JSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	var body map[string]any
	if err := decoder.Decode(&body); err != nil && err.Error() != "EOF" {
		respondGCPIAMV3InvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpIAMV3BodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return body
}

func normalizeGCPIAMV3ActionSegment(raw string) string {
	segment := strings.TrimSpace(raw)
	segment = strings.ReplaceAll(segment, "%3A", ":")
	segment = strings.ReplaceAll(segment, "%3a", ":")
	return segment
}

func gcpIAMV3PolicyBinding(scopeType, scopeID, location, bindingID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("%s/%s/locations/%s/policyBindings/%s", scopeType, scopeID, location, bindingID),
		"displayName": "stackyard policy binding",
		"policy":      fmt.Sprintf("organizations/%s/locations/%s/principalAccessBoundaryPolicies/pab-a", scopeID, location),
		"target": map[string]any{
			"principalSet": "//cloudresourcemanager.googleapis.com/projects/stackyard",
		},
	}
}

func gcpIAMV3PrincipalAccessBoundaryPolicy(scopeType, scopeID, location, policyID string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("%s/%s/locations/%s/principalAccessBoundaryPolicies/%s", scopeType, scopeID, location, policyID),
		"displayName": "stackyard pab policy",
		"details": map[string]any{
			"rules": []any{
				map[string]any{
					"description": "restrict access to stackyard project",
					"resources":   []string{"//cloudresourcemanager.googleapis.com/projects/stackyard"},
				},
			},
		},
	}
}

func gcpIAMV3Operation(operationID string) map[string]any {
	return map[string]any{
		"name": fmt.Sprintf("operations/%s", operationID),
		"done": true,
	}
}

func respondGCPIAMV3InvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}
