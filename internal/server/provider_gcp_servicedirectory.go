package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleGCPServiceDirectoryRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_servicedirectory(w, r) {
		return true
	}

	path := normalizeGCPServiceDirectoryPath(rawRequestPath(r))
	if isGCPServiceDirectoryLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPServiceDirectoryListLocations(w, r, path) {
			return true
		}
		if handleGCPServiceDirectoryGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPServiceDirectoryPath(path, hasGCPServiceDirectoryHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPServiceDirectoryListNamespaces(w, r, path) {
			return true
		}
		if handleGCPServiceDirectoryGetNamespace(w, path) {
			return true
		}
		if handleGCPServiceDirectoryListServices(w, r, path) {
			return true
		}
		if handleGCPServiceDirectoryGetService(w, path) {
			return true
		}
		if handleGCPServiceDirectoryListEndpoints(w, r, path) {
			return true
		}
		if handleGCPServiceDirectoryGetEndpoint(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPServiceDirectoryCreateNamespace(w, r, path) {
			return true
		}
		if handleGCPServiceDirectoryCreateService(w, r, path) {
			return true
		}
		if handleGCPServiceDirectoryCreateEndpoint(w, r, path) {
			return true
		}
		if handleGCPServiceDirectoryResolveService(w, r, path) {
			return true
		}
		if handleGCPServiceDirectoryGetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPServiceDirectorySetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPServiceDirectoryTestIAMPermissions(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPServiceDirectoryUpdateNamespace(w, r, path) {
			return true
		}
		if handleGCPServiceDirectoryUpdateService(w, r, path) {
			return true
		}
		if handleGCPServiceDirectoryUpdateEndpoint(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPServiceDirectoryDeleteNamespace(w, path) {
			return true
		}
		if handleGCPServiceDirectoryDeleteService(w, path) {
			return true
		}
		if handleGCPServiceDirectoryDeleteEndpoint(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPServiceDirectoryPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPServiceDirectoryHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "servicedirectory", "servicedirectory-apiv1", "servicedirectory_apiv1", "service-directory", "service_directory", "gcp-service-directory":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-servicedirectory-apiv1") || strings.Contains(ua, "cloud.google.com/go/servicedirectory")
}

func isGCPServiceDirectoryLocationRequest(r *http.Request, path string) bool {
	return isGCPProjectLocationDiscoveryPath(path) && hasGCPServiceDirectoryHint(r)
}

func isGCPServiceDirectoryPath(path string, includeHint bool) bool {
	if _, _, tail, ok := parseGCPServiceDirectoryLocationTail(path); ok {
		if isGCPServiceDirectoryNamespacesCollectionTail(tail) ||
			isGCPServiceDirectoryNamespaceTail(tail) ||
			isGCPServiceDirectoryServicesCollectionTail(tail) ||
			isGCPServiceDirectoryServiceTail(tail) ||
			isGCPServiceDirectoryServiceResolveTail(tail) ||
			isGCPServiceDirectoryEndpointsCollectionTail(tail) ||
			isGCPServiceDirectoryEndpointTail(tail) {
			return true
		}
	}
	if _, _, ok := parseGCPServiceDirectoryIAMActionPath(path); ok {
		return true
	}
	if isGCPServiceDirectoryGRPCPath(path) {
		return true
	}
	return includeHint && strings.Contains(path, "/namespaces/")
}

func isGCPServiceDirectoryGRPCPath(path string) bool {
	return strings.HasPrefix(path, "/gcp/google.cloud.servicedirectory.v1.RegistrationService/") ||
		strings.HasPrefix(path, "/gcp/google.cloud.servicedirectory.v1.LookupService/")
}

func handleGCPServiceDirectoryListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPServiceDirectoryPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpServiceDirectoryLocation(project, "us-central1"),
		gcpServiceDirectoryLocation(project, "global"),
	}
	return respondGCPServiceDirectoryList(w, "locations", items, pageSize, start, path)
}

func handleGCPServiceDirectoryGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpServiceDirectoryLocation(project, location))
	return true
}

func handleGCPServiceDirectoryListNamespaces(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryNamespacesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPServiceDirectoryPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpServiceDirectoryNamespace(project, location, "ns-1"),
		gcpServiceDirectoryNamespace(project, location, "ns-2"),
	}
	return respondGCPServiceDirectoryList(w, "namespaces", items, pageSize, start, path)
}

func handleGCPServiceDirectoryGetNamespace(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryNamespaceTail(tail) {
		return false
	}
	namespaceID := strings.TrimSpace(tail[1])
	respondJSON(w, http.StatusOK, gcpServiceDirectoryNamespace(project, location, namespaceID))
	return true
}

func handleGCPServiceDirectoryCreateNamespace(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryNamespacesCollectionTail(tail) {
		return false
	}
	namespaceID := strings.TrimSpace(r.URL.Query().Get("namespaceId"))
	if !isGCPServiceDirectoryID(namespaceID) {
		respondGCPServiceDirectoryInvalidArgument(w, path, "namespaceId is required")
		return true
	}
	body, valid := decodeGCPServiceDirectoryJSONBody(w, r, path)
	if !valid {
		return true
	}
	if len(body) == 0 {
		respondGCPServiceDirectoryInvalidArgument(w, path, "namespace is required")
		return true
	}
	expectedName := gcpServiceDirectoryNamespaceName(project, location, namespaceID)
	if name := gcpServiceDirectoryString(body, "name"); name != "" && name != expectedName {
		respondGCPServiceDirectoryInvalidArgument(w, path, "namespace.name must match parent and namespaceId")
		return true
	}
	resp := gcpServiceDirectoryNamespace(project, location, namespaceID)
	if labels, ok := body["labels"].(map[string]any); ok {
		resp["labels"] = gcpServiceDirectoryStringMap(labels)
	}
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPServiceDirectoryUpdateNamespace(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryNamespaceTail(tail) {
		return false
	}
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		respondGCPServiceDirectoryInvalidArgument(w, path, "updateMask is required")
		return true
	}
	if !gcpServiceDirectoryValidUpdateMask(updateMask, map[string]struct{}{"labels": {}}) {
		respondGCPServiceDirectoryInvalidArgument(w, path, "updateMask has unsupported fields")
		return true
	}
	body, valid := decodeGCPServiceDirectoryJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpServiceDirectoryNamespaceName(project, location, strings.TrimSpace(tail[1]))
	if gcpServiceDirectoryString(body, "name") != expectedName {
		respondGCPServiceDirectoryInvalidArgument(w, path, "namespace.name must match requested resource")
		return true
	}
	resp := gcpServiceDirectoryNamespace(project, location, strings.TrimSpace(tail[1]))
	if labels, ok := body["labels"].(map[string]any); ok {
		resp["labels"] = gcpServiceDirectoryStringMap(labels)
	}
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPServiceDirectoryDeleteNamespace(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryNamespaceTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPServiceDirectoryListServices(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryServicesCollectionTail(tail) {
		return false
	}
	namespaceID := strings.TrimSpace(tail[1])
	pageSize, start, valid := parseGCPServiceDirectoryPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpServiceDirectoryService(project, location, namespaceID, "svc-1"),
		gcpServiceDirectoryService(project, location, namespaceID, "svc-2"),
	}
	return respondGCPServiceDirectoryList(w, "services", items, pageSize, start, path)
}

func handleGCPServiceDirectoryGetService(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryServiceTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpServiceDirectoryService(project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3])))
	return true
}

func handleGCPServiceDirectoryCreateService(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryServicesCollectionTail(tail) {
		return false
	}
	namespaceID := strings.TrimSpace(tail[1])
	serviceID := strings.TrimSpace(r.URL.Query().Get("serviceId"))
	if !isGCPServiceDirectoryID(serviceID) {
		respondGCPServiceDirectoryInvalidArgument(w, path, "serviceId is required")
		return true
	}
	body, valid := decodeGCPServiceDirectoryJSONBody(w, r, path)
	if !valid {
		return true
	}
	if len(body) == 0 {
		respondGCPServiceDirectoryInvalidArgument(w, path, "service is required")
		return true
	}
	expectedName := gcpServiceDirectoryServiceName(project, location, namespaceID, serviceID)
	if name := gcpServiceDirectoryString(body, "name"); name != "" && name != expectedName {
		respondGCPServiceDirectoryInvalidArgument(w, path, "service.name must match parent and serviceId")
		return true
	}
	resp := gcpServiceDirectoryService(project, location, namespaceID, serviceID)
	if annotations, ok := body["annotations"].(map[string]any); ok {
		resp["annotations"] = gcpServiceDirectoryStringMap(annotations)
	}
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPServiceDirectoryUpdateService(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryServiceTail(tail) {
		return false
	}
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		respondGCPServiceDirectoryInvalidArgument(w, path, "updateMask is required")
		return true
	}
	if !gcpServiceDirectoryValidUpdateMask(updateMask, map[string]struct{}{"annotations": {}}) {
		respondGCPServiceDirectoryInvalidArgument(w, path, "updateMask has unsupported fields")
		return true
	}
	body, valid := decodeGCPServiceDirectoryJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpServiceDirectoryServiceName(project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]))
	if gcpServiceDirectoryString(body, "name") != expectedName {
		respondGCPServiceDirectoryInvalidArgument(w, path, "service.name must match requested resource")
		return true
	}
	resp := gcpServiceDirectoryService(project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]))
	if annotations, ok := body["annotations"].(map[string]any); ok {
		resp["annotations"] = gcpServiceDirectoryStringMap(annotations)
	}
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPServiceDirectoryDeleteService(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryServiceTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPServiceDirectoryListEndpoints(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryEndpointsCollectionTail(tail) {
		return false
	}
	namespaceID := strings.TrimSpace(tail[1])
	serviceID := strings.TrimSpace(tail[3])
	pageSize, start, valid := parseGCPServiceDirectoryPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpServiceDirectoryEndpoint(project, location, namespaceID, serviceID, "ep-1"),
		gcpServiceDirectoryEndpoint(project, location, namespaceID, serviceID, "ep-2"),
	}
	return respondGCPServiceDirectoryList(w, "endpoints", items, pageSize, start, path)
}

func handleGCPServiceDirectoryGetEndpoint(w http.ResponseWriter, path string) bool {
	project, location, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryEndpointTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, gcpServiceDirectoryEndpoint(project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), strings.TrimSpace(tail[5])))
	return true
}

func handleGCPServiceDirectoryCreateEndpoint(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryEndpointsCollectionTail(tail) {
		return false
	}
	namespaceID := strings.TrimSpace(tail[1])
	serviceID := strings.TrimSpace(tail[3])
	endpointID := strings.TrimSpace(r.URL.Query().Get("endpointId"))
	if !isGCPServiceDirectoryID(endpointID) {
		respondGCPServiceDirectoryInvalidArgument(w, path, "endpointId is required")
		return true
	}
	body, valid := decodeGCPServiceDirectoryJSONBody(w, r, path)
	if !valid {
		return true
	}
	if len(body) == 0 {
		respondGCPServiceDirectoryInvalidArgument(w, path, "endpoint is required")
		return true
	}
	expectedName := gcpServiceDirectoryEndpointName(project, location, namespaceID, serviceID, endpointID)
	if name := gcpServiceDirectoryString(body, "name"); name != "" && name != expectedName {
		respondGCPServiceDirectoryInvalidArgument(w, path, "endpoint.name must match parent and endpointId")
		return true
	}
	if !gcpServiceDirectoryValidateEndpointFields(w, path, body) {
		return true
	}
	resp := gcpServiceDirectoryEndpoint(project, location, namespaceID, serviceID, endpointID)
	gcpServiceDirectoryApplyEndpointOverrides(resp, body)
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPServiceDirectoryUpdateEndpoint(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryEndpointTail(tail) {
		return false
	}
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		respondGCPServiceDirectoryInvalidArgument(w, path, "updateMask is required")
		return true
	}
	if !gcpServiceDirectoryValidUpdateMask(updateMask, map[string]struct{}{"address": {}, "port": {}, "annotations": {}, "network": {}}) {
		respondGCPServiceDirectoryInvalidArgument(w, path, "updateMask has unsupported fields")
		return true
	}
	body, valid := decodeGCPServiceDirectoryJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpServiceDirectoryEndpointName(project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), strings.TrimSpace(tail[5]))
	if gcpServiceDirectoryString(body, "name") != expectedName {
		respondGCPServiceDirectoryInvalidArgument(w, path, "endpoint.name must match requested resource")
		return true
	}
	if !gcpServiceDirectoryValidateEndpointFields(w, path, body) {
		return true
	}
	resp := gcpServiceDirectoryEndpoint(project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), strings.TrimSpace(tail[5]))
	gcpServiceDirectoryApplyEndpointOverrides(resp, body)
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPServiceDirectoryDeleteEndpoint(w http.ResponseWriter, path string) bool {
	_, _, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryEndpointTail(tail) {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPServiceDirectoryResolveService(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPServiceDirectoryLocationTail(path)
	if !ok || !isGCPServiceDirectoryServiceResolveTail(tail) {
		return false
	}
	namespaceID := strings.TrimSpace(tail[1])
	serviceID := strings.TrimSpace(strings.TrimSuffix(tail[3], ":resolve"))
	body, valid := decodeGCPServiceDirectoryJSONBodyOptional(w, r, path)
	if !valid {
		return true
	}
	name := gcpServiceDirectoryServiceName(project, location, namespaceID, serviceID)
	if providedName := gcpServiceDirectoryString(body, "name"); providedName != "" && providedName != name {
		respondGCPServiceDirectoryInvalidArgument(w, path, "name must match requested service")
		return true
	}
	maxEndpoints := 0
	if raw, ok := body["maxEndpoints"]; ok {
		value, ok := gcpServiceDirectoryInt(raw)
		if !ok || value < 0 || value > 1000 {
			respondGCPServiceDirectoryInvalidArgument(w, path, "maxEndpoints must be between 0 and 1000")
			return true
		}
		maxEndpoints = value
	}
	if endpointFilter := strings.TrimSpace(gcpServiceDirectoryString(body, "endpointFilter")); endpointFilter != "" {
		if strings.Contains(endpointFilter, "!!") || strings.Contains(endpointFilter, "\n") {
			respondGCPServiceDirectoryInvalidArgument(w, path, "endpointFilter is invalid")
			return true
		}
	}
	endpoints := []map[string]any{
		gcpServiceDirectoryEndpoint(project, location, namespaceID, serviceID, "ep-1"),
		gcpServiceDirectoryEndpoint(project, location, namespaceID, serviceID, "ep-2"),
	}
	if maxEndpoints > 0 && maxEndpoints < len(endpoints) {
		endpoints = endpoints[:maxEndpoints]
	}
	service := gcpServiceDirectoryService(project, location, namespaceID, serviceID)
	service["endpoints"] = endpoints
	respondJSON(w, http.StatusOK, map[string]any{
		"service": service,
	})
	return true
}

func handleGCPServiceDirectoryGetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	resource, action, ok := parseGCPServiceDirectoryIAMActionPath(path)
	if !ok || action != "getIamPolicy" {
		return false
	}
	body, valid := decodeGCPServiceDirectoryJSONBodyOptional(w, r, path)
	if !valid {
		return true
	}
	if bodyResource := strings.TrimSpace(gcpServiceDirectoryString(body, "resource")); bodyResource != "" && bodyResource != resource {
		respondGCPServiceDirectoryInvalidArgument(w, path, "resource must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpServiceDirectoryIAMPolicy(resource, nil))
	return true
}

func handleGCPServiceDirectorySetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	resource, action, ok := parseGCPServiceDirectoryIAMActionPath(path)
	if !ok || action != "setIamPolicy" {
		return false
	}
	body, valid := decodeGCPServiceDirectoryJSONBody(w, r, path)
	if !valid {
		return true
	}
	if bodyResource := strings.TrimSpace(gcpServiceDirectoryString(body, "resource")); bodyResource != "" && bodyResource != resource {
		respondGCPServiceDirectoryInvalidArgument(w, path, "resource must match requested resource")
		return true
	}
	policy, _ := body["policy"].(map[string]any)
	if len(policy) == 0 {
		respondGCPServiceDirectoryInvalidArgument(w, path, "policy is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpServiceDirectoryIAMPolicy(resource, policy))
	return true
}

func handleGCPServiceDirectoryTestIAMPermissions(w http.ResponseWriter, r *http.Request, path string) bool {
	resource, action, ok := parseGCPServiceDirectoryIAMActionPath(path)
	if !ok || action != "testIamPermissions" {
		return false
	}
	body, valid := decodeGCPServiceDirectoryJSONBody(w, r, path)
	if !valid {
		return true
	}
	if bodyResource := strings.TrimSpace(gcpServiceDirectoryString(body, "resource")); bodyResource != "" && bodyResource != resource {
		respondGCPServiceDirectoryInvalidArgument(w, path, "resource must match requested resource")
		return true
	}
	rawPermissions, _ := body["permissions"].([]any)
	if len(rawPermissions) == 0 {
		respondGCPServiceDirectoryInvalidArgument(w, path, "permissions must include at least one entry")
		return true
	}
	permissions := make([]string, 0, len(rawPermissions))
	for idx, raw := range rawPermissions {
		permission, ok := raw.(string)
		if !ok || strings.TrimSpace(permission) == "" {
			respondGCPServiceDirectoryInvalidArgument(w, path, fmt.Sprintf("permissions[%d] must be a non-empty string", idx))
			return true
		}
		permissions = append(permissions, strings.TrimSpace(permission))
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"permissions": permissions,
	})
	return true
}

func parseGCPServiceDirectoryLocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", nil, false
	}
	return project, location, parts[6:], true
}

func parseGCPServiceDirectoryIAMActionPath(path string) (resource, action string, ok bool) {
	trimmed := strings.Trim(path, "/")
	const prefix = "gcp/v1/"
	if !strings.HasPrefix(trimmed, prefix) {
		return "", "", false
	}
	target := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
	idx := strings.LastIndex(target, ":")
	if idx <= 0 || idx >= len(target)-1 {
		return "", "", false
	}
	resource = strings.TrimSpace(target[:idx])
	action = strings.TrimSpace(target[idx+1:])
	if resource == "" || action == "" {
		return "", "", false
	}
	if action != "getIamPolicy" && action != "setIamPolicy" && action != "testIamPermissions" {
		return "", "", false
	}
	parts := strings.Split(resource, "/")
	if len(parts) == 6 && parts[0] == "projects" && parts[2] == "locations" && parts[4] == "namespaces" {
		return resource, action, strings.TrimSpace(parts[1]) != "" && strings.TrimSpace(parts[3]) != "" && isGCPServiceDirectoryID(parts[5])
	}
	if len(parts) == 8 && parts[0] == "projects" && parts[2] == "locations" && parts[4] == "namespaces" && parts[6] == "services" {
		return resource, action, strings.TrimSpace(parts[1]) != "" && strings.TrimSpace(parts[3]) != "" && isGCPServiceDirectoryID(parts[5]) && isGCPServiceDirectoryID(parts[7])
	}
	return "", "", false
}

func isGCPServiceDirectoryNamespacesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "namespaces"
}

func isGCPServiceDirectoryNamespaceTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "namespaces" && isGCPServiceDirectoryID(tail[1])
}

func isGCPServiceDirectoryServicesCollectionTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "namespaces" && isGCPServiceDirectoryID(tail[1]) && tail[2] == "services"
}

func isGCPServiceDirectoryServiceTail(tail []string) bool {
	return len(tail) == 4 && tail[0] == "namespaces" && isGCPServiceDirectoryID(tail[1]) && tail[2] == "services" && isGCPServiceDirectoryID(tail[3])
}

func isGCPServiceDirectoryServiceResolveTail(tail []string) bool {
	if len(tail) != 4 || tail[0] != "namespaces" || !isGCPServiceDirectoryID(tail[1]) || tail[2] != "services" {
		return false
	}
	serviceID, action, found := strings.Cut(strings.TrimSpace(tail[3]), ":")
	return found && action == "resolve" && isGCPServiceDirectoryID(serviceID)
}

func isGCPServiceDirectoryEndpointsCollectionTail(tail []string) bool {
	return len(tail) == 5 && tail[0] == "namespaces" && isGCPServiceDirectoryID(tail[1]) && tail[2] == "services" && isGCPServiceDirectoryID(tail[3]) && tail[4] == "endpoints"
}

func isGCPServiceDirectoryEndpointTail(tail []string) bool {
	return len(tail) == 6 && tail[0] == "namespaces" && isGCPServiceDirectoryID(tail[1]) && tail[2] == "services" && isGCPServiceDirectoryID(tail[3]) && tail[4] == "endpoints" && isGCPServiceDirectoryID(tail[5])
}

func decodeGCPServiceDirectoryJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPServiceDirectoryInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		respondGCPServiceDirectoryInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	body := map[string]any{}
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPServiceDirectoryInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func decodeGCPServiceDirectoryJSONBodyOptional(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPServiceDirectoryInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	body := map[string]any{}
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPServiceDirectoryInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func parseGCPServiceDirectoryPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0
	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		value, err := parseOptionalNonNegativeInt(raw)
		if err != nil {
			respondGCPServiceDirectoryInvalidArgument(w, path, "pageSize must be a non-negative integer")
			return 0, 0, false
		}
		pageSize = value
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		value, err := parseOptionalNonNegativeInt(raw)
		if err != nil {
			respondGCPServiceDirectoryInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = value
	}
	return pageSize, start, true
}

func respondGCPServiceDirectoryList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPServiceDirectoryInvalidArgument(w, path, "pageToken is out of range")
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

func gcpServiceDirectoryValidateEndpointFields(w http.ResponseWriter, path string, endpoint map[string]any) bool {
	if rawPort, ok := endpoint["port"]; ok {
		port, ok := gcpServiceDirectoryInt(rawPort)
		if !ok || port < 0 || port > 65535 {
			respondGCPServiceDirectoryInvalidArgument(w, path, "endpoint.port must be between 0 and 65535")
			return false
		}
	}
	if address := strings.TrimSpace(gcpServiceDirectoryString(endpoint, "address")); strings.Contains(address, "/") {
		respondGCPServiceDirectoryInvalidArgument(w, path, "endpoint.address is invalid")
		return false
	}
	return true
}

func gcpServiceDirectoryApplyEndpointOverrides(out, in map[string]any) {
	if address := strings.TrimSpace(gcpServiceDirectoryString(in, "address")); address != "" {
		out["address"] = address
	}
	if rawPort, ok := in["port"]; ok {
		if port, ok := gcpServiceDirectoryInt(rawPort); ok {
			out["port"] = port
		}
	}
	if annotations, ok := in["annotations"].(map[string]any); ok {
		out["annotations"] = gcpServiceDirectoryStringMap(annotations)
	}
	if network := strings.TrimSpace(gcpServiceDirectoryString(in, "network")); network != "" {
		out["network"] = network
	}
}

func gcpServiceDirectoryValidUpdateMask(mask string, allowed map[string]struct{}) bool {
	for _, raw := range strings.Split(mask, ",") {
		field := strings.TrimSpace(raw)
		if field == "" {
			return false
		}
		if _, ok := allowed[field]; !ok {
			return false
		}
	}
	return true
}

func gcpServiceDirectoryInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), float64(int(v)) == v
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		return parsed, err == nil
	default:
		return 0, false
	}
}

func gcpServiceDirectoryString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func gcpServiceDirectoryStringMap(in map[string]any) map[string]string {
	out := map[string]string{}
	for key, raw := range in {
		value, ok := raw.(string)
		if !ok {
			continue
		}
		out[key] = value
	}
	return out
}

func isGCPServiceDirectoryID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 63 || strings.Contains(id, "/") || strings.Contains(id, ":") {
		return false
	}
	return true
}

func gcpServiceDirectoryNamespaceName(project, location, namespaceID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/namespaces/%s", project, location, namespaceID)
}

func gcpServiceDirectoryServiceName(project, location, namespaceID, serviceID string) string {
	return fmt.Sprintf("%s/services/%s", gcpServiceDirectoryNamespaceName(project, location, namespaceID), serviceID)
}

func gcpServiceDirectoryEndpointName(project, location, namespaceID, serviceID, endpointID string) string {
	return fmt.Sprintf("%s/endpoints/%s", gcpServiceDirectoryServiceName(project, location, namespaceID, serviceID), endpointID)
}

func gcpServiceDirectoryLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": strings.ToUpper(location),
		"labels": map[string]any{
			"stackyard": "true",
		},
	}
}

func gcpServiceDirectoryNamespace(project, location, namespaceID string) map[string]any {
	return map[string]any{
		"name": gcpServiceDirectoryNamespaceName(project, location, namespaceID),
		"labels": map[string]any{
			"env": "stackyard",
		},
		"uid": fmt.Sprintf("namespace-%s", namespaceID),
	}
}

func gcpServiceDirectoryService(project, location, namespaceID, serviceID string) map[string]any {
	return map[string]any{
		"name": gcpServiceDirectoryServiceName(project, location, namespaceID, serviceID),
		"annotations": map[string]any{
			"owner": "stackyard",
		},
		"uid": fmt.Sprintf("service-%s", serviceID),
	}
}

func gcpServiceDirectoryEndpoint(project, location, namespaceID, serviceID, endpointID string) map[string]any {
	return map[string]any{
		"name":    gcpServiceDirectoryEndpointName(project, location, namespaceID, serviceID, endpointID),
		"address": "10.10.0.8",
		"port":    8080,
		"annotations": map[string]any{
			"backend": "primary",
		},
		"network": "projects/1234567890/locations/global/networks/default",
		"uid":     fmt.Sprintf("endpoint-%s", endpointID),
	}
}

func gcpServiceDirectoryIAMPolicy(resource string, requested map[string]any) map[string]any {
	policy := map[string]any{
		"version": 1,
		"etag":    fmt.Sprintf("etag-%s", strings.ReplaceAll(resource, "/", "-")),
		"bindings": []map[string]any{
			{
				"role":    "roles/servicedirectory.viewer",
				"members": []string{"user:stackyard@example.com"},
			},
		},
	}
	if requested != nil {
		if version, ok := requested["version"].(float64); ok {
			policy["version"] = int(version)
		}
		if bindings, ok := requested["bindings"].([]any); ok {
			policy["bindings"] = bindings
		}
		if etag, ok := requested["etag"].(string); ok && strings.TrimSpace(etag) != "" {
			policy["etag"] = etag
		}
	}
	return policy
}

func respondGCPServiceDirectoryInvalidArgument(w http.ResponseWriter, path, message string) {
	respondJSON(w, http.StatusBadRequest, map[string]any{
		"error":    "InvalidArgument",
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_servicedirectory(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if r.URL.Query().Get("stackyard_contract_probe") != "1" {
		return false
	}

	path := normalizeGCPServiceDirectoryPath(rawRequestPath(r))
	project, location, serviceToken, ok := parseGCPServiceDirectoryProbePath(path)
	if !ok {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPServiceDirectoryInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") != "1" {
		return false
	}

	payload := gcpServiceDirectoryNamespace(project, location, "ns-1")
	payload["service"] = serviceToken
	payload["provider"] = providerGCP
	payload["path"] = path
	respondJSON(w, http.StatusOK, payload)
	return true
}

func parseGCPServiceDirectoryProbePath(path string) (project, location, service string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	service = strings.TrimSpace(parts[6])
	if project == "" || location == "" || service != "servicedirectory" {
		return "", "", "", false
	}
	return project, location, service, true
}
