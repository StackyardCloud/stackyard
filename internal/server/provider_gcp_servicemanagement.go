package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	gcpServiceManagementReferenceTime    = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpServiceManagementServiceNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`)
	gcpServiceManagementProjectIDRegex   = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
)

func (s *Server) handleGCPServiceManagementRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_servicemanagement(w, r) {
		return true
	}

	path := normalizeGCPServiceManagementPath(rawRequestPath(r))
	if !isGCPServiceManagementPath(path, hasGCPServiceManagementHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPServiceManagementListServices(w, r, path) {
			return true
		}
		if handleGCPServiceManagementGetService(w, path) {
			return true
		}
		if handleGCPServiceManagementListServiceConfigs(w, r, path) {
			return true
		}
		if handleGCPServiceManagementGetServiceConfig(w, r, path) {
			return true
		}
		if handleGCPServiceManagementListServiceRollouts(w, r, path) {
			return true
		}
		if handleGCPServiceManagementGetServiceRollout(w, path) {
			return true
		}
		if handleGCPServiceManagementListOperations(w, r, path) {
			return true
		}
		if handleGCPServiceManagementGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPServiceManagementCreateService(w, r, path) {
			return true
		}
		if handleGCPServiceManagementUndeleteService(w, path) {
			return true
		}
		if handleGCPServiceManagementCreateServiceConfig(w, r, path) {
			return true
		}
		if handleGCPServiceManagementSubmitConfigSource(w, r, path) {
			return true
		}
		if handleGCPServiceManagementCreateServiceRollout(w, r, path) {
			return true
		}
		if handleGCPServiceManagementGenerateConfigReport(w, r, path) {
			return true
		}
		if handleGCPServiceManagementGetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPServiceManagementSetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPServiceManagementTestIAMPermissions(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPServiceManagementDeleteService(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPServiceManagementPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPServiceManagementHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "servicemanagement",
		"servicemanagement-apiv1",
		"servicemanagement_apiv1",
		"service-management",
		"service_management",
		"gcp-service-management":
		return true
	}

	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-servicemanagement-apiv1") || strings.Contains(ua, "cloud.google.com/go/servicemanagement")
}

func isGCPServiceManagementPath(path string, includeHint bool) bool {
	if isGCPServiceManagementGRPCPath(path) {
		return true
	}
	if parseGCPServiceManagementServicesCollectionPath(path) {
		return true
	}
	if _, ok := parseGCPServiceManagementServicePath(path); ok {
		return true
	}
	if _, action, ok := parseGCPServiceManagementServiceActionPath(path); ok && action == "undelete" {
		return true
	}
	if _, _, _, ok := parseGCPServiceManagementServiceConfigsPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPServiceManagementServiceRolloutsPath(path); ok {
		return true
	}
	if parseGCPServiceManagementGenerateConfigReportPath(path) {
		return true
	}
	if _, _, ok := parseGCPServiceManagementIAMActionPath(path); ok {
		return true
	}
	if parseGCPServiceManagementOperationsCollectionPath(path) {
		return includeHint
	}
	if operationName, ok := parseGCPServiceManagementOperationPath(path); ok {
		return includeHint || strings.HasPrefix(operationName, "servicemanagement-")
	}
	return includeHint && strings.HasPrefix(path, "/gcp/v1/services")
}

func isGCPServiceManagementGRPCPath(path string) bool {
	return strings.HasPrefix(strings.TrimSpace(path), "/gcp/google.api.servicemanagement.v1.ServiceManager/")
}

func handleGCPServiceManagementListServices(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPServiceManagementServicesCollectionPath(path) {
		return false
	}

	pageSize, start, ok := parseGCPServiceManagementPagination(w, r, path, 500)
	if !ok {
		return true
	}

	consumerID := strings.TrimSpace(r.URL.Query().Get("consumerId"))
	if consumerID != "" && !strings.HasPrefix(consumerID, "project:") {
		respondGCPServiceManagementInvalidArgument(w, path, "consumerId must have project: prefix")
		return true
	}
	producerProjectID := strings.TrimSpace(r.URL.Query().Get("producerProjectId"))
	if producerProjectID != "" && !gcpServiceManagementProjectIDRegex.MatchString(producerProjectID) {
		respondGCPServiceManagementInvalidArgument(w, path, "producerProjectId is invalid")
		return true
	}

	items := []map[string]any{
		gcpServiceManagementManagedService("stackyard.googleapis.com"),
		gcpServiceManagementManagedService("aux.stackyard.googleapis.com"),
	}
	return respondGCPServiceManagementList(w, "services", items, pageSize, start, path)
}

func handleGCPServiceManagementGetService(w http.ResponseWriter, path string) bool {
	serviceName, ok := parseGCPServiceManagementServicePath(path)
	if !ok {
		return false
	}
	if !isGCPServiceManagementServiceName(serviceName) {
		respondGCPServiceManagementInvalidArgument(w, path, "serviceName is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpServiceManagementManagedService(serviceName))
	return true
}

func handleGCPServiceManagementCreateService(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPServiceManagementServicesCollectionPath(path) {
		return false
	}
	body, ok := decodeGCPServiceManagementJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}

	serviceName := strings.TrimSpace(gcpServiceManagementString(body, "serviceName"))
	if serviceName == "" {
		respondGCPServiceManagementInvalidArgument(w, path, "service.serviceName is required")
		return true
	}
	if !isGCPServiceManagementServiceName(serviceName) {
		respondGCPServiceManagementInvalidArgument(w, path, "service.serviceName is invalid")
		return true
	}

	respondJSON(w, http.StatusOK, gcpServiceManagementOperation("servicemanagement-create-service", "services/"+serviceName, "create", false))
	return true
}

func handleGCPServiceManagementDeleteService(w http.ResponseWriter, path string) bool {
	serviceName, ok := parseGCPServiceManagementServicePath(path)
	if !ok {
		return false
	}
	if !isGCPServiceManagementServiceName(serviceName) {
		respondGCPServiceManagementInvalidArgument(w, path, "serviceName is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpServiceManagementOperation("servicemanagement-delete-service", "services/"+serviceName, "delete", false))
	return true
}

func handleGCPServiceManagementUndeleteService(w http.ResponseWriter, path string) bool {
	serviceName, action, ok := parseGCPServiceManagementServiceActionPath(path)
	if !ok || action != "undelete" {
		return false
	}
	if !isGCPServiceManagementServiceName(serviceName) {
		respondGCPServiceManagementInvalidArgument(w, path, "serviceName is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpServiceManagementOperation("servicemanagement-undelete-service", "services/"+serviceName, "undelete", false))
	return true
}

func handleGCPServiceManagementListServiceConfigs(w http.ResponseWriter, r *http.Request, path string) bool {
	serviceName, configID, submit, ok := parseGCPServiceManagementServiceConfigsPath(path)
	if !ok || configID != "" || submit {
		return false
	}
	if !isGCPServiceManagementServiceName(serviceName) {
		respondGCPServiceManagementInvalidArgument(w, path, "serviceName is invalid")
		return true
	}

	pageSize, start, ok := parseGCPServiceManagementPagination(w, r, path, 100)
	if !ok {
		return true
	}
	items := []map[string]any{
		gcpServiceManagementServiceConfig(serviceName, "2026-01-01r0", false),
		gcpServiceManagementServiceConfig(serviceName, "2026-01-02r0", false),
	}
	return respondGCPServiceManagementList(w, "serviceConfigs", items, pageSize, start, path)
}

func handleGCPServiceManagementGetServiceConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	serviceName, configID, submit, ok := parseGCPServiceManagementServiceConfigsPath(path)
	if !ok || configID == "" || submit {
		return false
	}
	if !isGCPServiceManagementServiceName(serviceName) {
		respondGCPServiceManagementInvalidArgument(w, path, "serviceName is invalid")
		return true
	}
	full, valid := parseGCPServiceManagementConfigView(r.URL.Query().Get("view"))
	if !valid {
		respondGCPServiceManagementInvalidArgument(w, path, "view must be BASIC, FULL, or 0..1")
		return true
	}
	respondJSON(w, http.StatusOK, gcpServiceManagementServiceConfig(serviceName, configID, full))
	return true
}

func handleGCPServiceManagementCreateServiceConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	serviceName, configID, submit, ok := parseGCPServiceManagementServiceConfigsPath(path)
	if !ok || configID != "" || submit {
		return false
	}
	if !isGCPServiceManagementServiceName(serviceName) {
		respondGCPServiceManagementInvalidArgument(w, path, "serviceName is invalid")
		return true
	}
	body, ok := decodeGCPServiceManagementJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	if name := strings.TrimSpace(gcpServiceManagementString(body, "name")); name != "" && name != serviceName {
		respondGCPServiceManagementInvalidArgument(w, path, "serviceConfig.name must match serviceName")
		return true
	}
	configID = strings.TrimSpace(gcpServiceManagementString(body, "id"))
	if configID == "" {
		configID = "2026-01-03r0"
	}
	respondJSON(w, http.StatusOK, gcpServiceManagementServiceConfig(serviceName, configID, true))
	return true
}

func handleGCPServiceManagementSubmitConfigSource(w http.ResponseWriter, r *http.Request, path string) bool {
	serviceName, configID, submit, ok := parseGCPServiceManagementServiceConfigsPath(path)
	if !ok || configID != "" || !submit {
		return false
	}
	if !isGCPServiceManagementServiceName(serviceName) {
		respondGCPServiceManagementInvalidArgument(w, path, "serviceName is invalid")
		return true
	}

	body, ok := decodeGCPServiceManagementJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	if bodyServiceName := strings.TrimSpace(gcpServiceManagementString(body, "serviceName")); bodyServiceName != "" && bodyServiceName != serviceName {
		respondGCPServiceManagementInvalidArgument(w, path, "serviceName must match requested service")
		return true
	}
	configSource, _ := body["configSource"].(map[string]any)
	if len(configSource) == 0 {
		respondGCPServiceManagementInvalidArgument(w, path, "configSource is required")
		return true
	}
	if files, present := configSource["files"]; present {
		rawFiles, ok := files.([]any)
		if !ok || len(rawFiles) == 0 {
			respondGCPServiceManagementInvalidArgument(w, path, "configSource.files must include at least one entry")
			return true
		}
		for idx, raw := range rawFiles {
			file, ok := raw.(map[string]any)
			if !ok {
				respondGCPServiceManagementInvalidArgument(w, path, fmt.Sprintf("configSource.files[%d] must be an object", idx))
				return true
			}
			if strings.TrimSpace(gcpServiceManagementString(file, "filePath")) == "" {
				respondGCPServiceManagementInvalidArgument(w, path, fmt.Sprintf("configSource.files[%d].filePath is required", idx))
				return true
			}
		}
	}

	respondJSON(w, http.StatusOK, gcpServiceManagementOperation("servicemanagement-submit-config-source", "services/"+serviceName+"/configs/2026-01-04r0", "submitConfigSource", false))
	return true
}

func handleGCPServiceManagementListServiceRollouts(w http.ResponseWriter, r *http.Request, path string) bool {
	serviceName, rolloutID, ok := parseGCPServiceManagementServiceRolloutsPath(path)
	if !ok || rolloutID != "" {
		return false
	}
	if !isGCPServiceManagementServiceName(serviceName) {
		respondGCPServiceManagementInvalidArgument(w, path, "serviceName is invalid")
		return true
	}

	pageSize, start, ok := parseGCPServiceManagementPagination(w, r, path, 100)
	if !ok {
		return true
	}
	filter := strings.TrimSpace(r.URL.Query().Get("filter"))
	if filter == "" {
		respondGCPServiceManagementInvalidArgument(w, path, "filter is required")
		return true
	}
	if !isGCPServiceManagementRolloutFilter(filter) {
		respondGCPServiceManagementInvalidArgument(w, path, "filter is malformed")
		return true
	}

	items := []map[string]any{
		gcpServiceManagementRollout(serviceName, "2026-01-01r0"),
		gcpServiceManagementRollout(serviceName, "2026-01-02r0"),
	}
	return respondGCPServiceManagementList(w, "rollouts", items, pageSize, start, path)
}

func handleGCPServiceManagementGetServiceRollout(w http.ResponseWriter, path string) bool {
	serviceName, rolloutID, ok := parseGCPServiceManagementServiceRolloutsPath(path)
	if !ok || rolloutID == "" {
		return false
	}
	if !isGCPServiceManagementServiceName(serviceName) {
		respondGCPServiceManagementInvalidArgument(w, path, "serviceName is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpServiceManagementRollout(serviceName, rolloutID))
	return true
}

func handleGCPServiceManagementCreateServiceRollout(w http.ResponseWriter, r *http.Request, path string) bool {
	serviceName, rolloutID, ok := parseGCPServiceManagementServiceRolloutsPath(path)
	if !ok || rolloutID != "" {
		return false
	}
	if !isGCPServiceManagementServiceName(serviceName) {
		respondGCPServiceManagementInvalidArgument(w, path, "serviceName is invalid")
		return true
	}
	body, ok := decodeGCPServiceManagementJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	if strings.TrimSpace(gcpServiceManagementString(body, "rolloutId")) == "" {
		respondGCPServiceManagementInvalidArgument(w, path, "rollout.rolloutId is required")
		return true
	}
	_, hasTraffic := body["trafficPercentStrategy"]
	_, hasDelete := body["deleteServiceStrategy"]
	if !hasTraffic && !hasDelete {
		respondGCPServiceManagementInvalidArgument(w, path, "rollout strategy is required")
		return true
	}
	if hasTraffic && hasDelete {
		respondGCPServiceManagementInvalidArgument(w, path, "rollout must define only one strategy")
		return true
	}
	respondJSON(w, http.StatusOK, gcpServiceManagementOperation("servicemanagement-create-rollout", "services/"+serviceName+"/rollouts/"+strings.TrimSpace(gcpServiceManagementString(body, "rolloutId")), "createRollout", false))
	return true
}

func handleGCPServiceManagementGenerateConfigReport(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPServiceManagementGenerateConfigReportPath(path) {
		return false
	}
	body, ok := decodeGCPServiceManagementJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	if _, exists := body["newConfig"]; !exists {
		respondGCPServiceManagementInvalidArgument(w, path, "newConfig is required")
		return true
	}
	serviceName := "stackyard.googleapis.com"
	if oldConfig, ok := body["oldConfig"].(map[string]any); ok {
		if name := strings.TrimSpace(gcpServiceManagementString(oldConfig, "name")); isGCPServiceManagementServiceName(name) {
			serviceName = name
		}
	}
	respondJSON(w, http.StatusOK, gcpServiceManagementGenerateConfigReport(serviceName))
	return true
}

func handleGCPServiceManagementGetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	resource, action, ok := parseGCPServiceManagementIAMActionPath(path)
	if !ok || action != "getIamPolicy" {
		return false
	}
	body, ok := decodeGCPServiceManagementJSONBodyOptional(w, r, path)
	if !ok {
		return true
	}
	if bodyResource := strings.TrimSpace(gcpServiceManagementString(body, "resource")); bodyResource != "" && bodyResource != resource {
		respondGCPServiceManagementInvalidArgument(w, path, "resource must match requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpServiceManagementIAMPolicy(resource))
	return true
}

func handleGCPServiceManagementSetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	resource, action, ok := parseGCPServiceManagementIAMActionPath(path)
	if !ok || action != "setIamPolicy" {
		return false
	}
	body, ok := decodeGCPServiceManagementJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	if bodyResource := strings.TrimSpace(gcpServiceManagementString(body, "resource")); bodyResource != "" && bodyResource != resource {
		respondGCPServiceManagementInvalidArgument(w, path, "resource must match requested resource")
		return true
	}
	policy, ok := body["policy"].(map[string]any)
	if !ok || len(policy) == 0 {
		respondGCPServiceManagementInvalidArgument(w, path, "policy is required")
		return true
	}
	resp := gcpServiceManagementIAMPolicy(resource)
	if version, ok := policy["version"]; ok {
		resp["version"] = version
	}
	if bindings, ok := policy["bindings"].([]any); ok {
		resp["bindings"] = bindings
	}
	if etag := strings.TrimSpace(gcpServiceManagementString(policy, "etag")); etag != "" {
		resp["etag"] = etag
	}
	respondJSON(w, http.StatusOK, resp)
	return true
}

func handleGCPServiceManagementTestIAMPermissions(w http.ResponseWriter, r *http.Request, path string) bool {
	resource, action, ok := parseGCPServiceManagementIAMActionPath(path)
	if !ok || action != "testIamPermissions" {
		return false
	}
	body, ok := decodeGCPServiceManagementJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	if bodyResource := strings.TrimSpace(gcpServiceManagementString(body, "resource")); bodyResource != "" && bodyResource != resource {
		respondGCPServiceManagementInvalidArgument(w, path, "resource must match requested resource")
		return true
	}
	rawPermissions, _ := body["permissions"].([]any)
	if len(rawPermissions) == 0 {
		respondGCPServiceManagementInvalidArgument(w, path, "permissions must include at least one entry")
		return true
	}
	permissions := make([]string, 0, len(rawPermissions))
	for idx, raw := range rawPermissions {
		permission, ok := raw.(string)
		if !ok || strings.TrimSpace(permission) == "" {
			respondGCPServiceManagementInvalidArgument(w, path, fmt.Sprintf("permissions[%d] must be a non-empty string", idx))
			return true
		}
		permissions = append(permissions, strings.TrimSpace(permission))
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"permissions": permissions,
	})
	return true
}

func handleGCPServiceManagementListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPServiceManagementOperationsCollectionPath(path) {
		return false
	}

	pageSize, start, ok := parseGCPServiceManagementPagination(w, r, path, 100)
	if !ok {
		return true
	}
	if filter := strings.TrimSpace(r.URL.Query().Get("filter")); filter != "" && isGCPServiceManagementMalformedFilter(filter) {
		respondGCPServiceManagementInvalidArgument(w, path, "filter is malformed")
		return true
	}
	if name := strings.TrimSpace(r.URL.Query().Get("name")); name != "" && !strings.HasPrefix(name, "operations/") {
		respondGCPServiceManagementInvalidArgument(w, path, "name must start with operations/")
		return true
	}

	items := []map[string]any{
		gcpServiceManagementOperation("servicemanagement-create-service", "services/stackyard.googleapis.com", "create", true),
		gcpServiceManagementOperation("servicemanagement-create-rollout", "services/stackyard.googleapis.com/rollouts/2026-01-01r0", "createRollout", true),
	}
	return respondGCPServiceManagementList(w, "operations", items, pageSize, start, path)
}

func handleGCPServiceManagementGetOperation(w http.ResponseWriter, path string) bool {
	operationName, ok := parseGCPServiceManagementOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpServiceManagementOperation(operationName, "services/stackyard.googleapis.com", "poll", true))
	return true
}

func parseGCPServiceManagementServicesCollectionPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "services"
}

func parseGCPServiceManagementServicePath(path string) (serviceName string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "services" {
		return "", false
	}
	serviceName = strings.TrimSpace(parts[3])
	if strings.Contains(serviceName, ":") || serviceName == "" {
		return "", false
	}
	return serviceName, true
}

func parseGCPServiceManagementServiceActionPath(path string) (serviceName, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "services" {
		return "", "", false
	}
	serviceName, action, ok = strings.Cut(strings.TrimSpace(parts[3]), ":")
	if !ok || strings.TrimSpace(serviceName) == "" || strings.TrimSpace(action) == "" {
		return "", "", false
	}
	return strings.TrimSpace(serviceName), strings.TrimSpace(action), true
}

func parseGCPServiceManagementServiceConfigsPath(path string) (serviceName, configID string, submit bool, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || len(parts) > 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "services" {
		return "", "", false, false
	}
	serviceName = strings.TrimSpace(parts[3])
	if serviceName == "" {
		return "", "", false, false
	}
	if len(parts) == 5 {
		if parts[4] == "configs" {
			return serviceName, "", false, true
		}
		if parts[4] == "configs:submit" {
			return serviceName, "", true, true
		}
		return "", "", false, false
	}
	if parts[4] != "configs" {
		return "", "", false, false
	}
	configID = strings.TrimSpace(parts[5])
	if configID == "" {
		return "", "", false, false
	}
	return serviceName, configID, false, true
}

func parseGCPServiceManagementServiceRolloutsPath(path string) (serviceName, rolloutID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 || len(parts) > 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "services" {
		return "", "", false
	}
	serviceName = strings.TrimSpace(parts[3])
	if serviceName == "" || parts[4] != "rollouts" {
		return "", "", false
	}
	if len(parts) == 5 {
		return serviceName, "", true
	}
	rolloutID = strings.TrimSpace(parts[5])
	if rolloutID == "" {
		return "", "", false
	}
	return serviceName, rolloutID, true
}

func parseGCPServiceManagementGenerateConfigReportPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "services:generateConfigReport"
}

func parseGCPServiceManagementIAMActionPath(path string) (resource, action string, ok bool) {
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
	return resource, action, true
}

func parseGCPServiceManagementOperationsCollectionPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "operations"
}

func parseGCPServiceManagementOperationPath(path string) (operationName string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "operations" {
		return "", false
	}
	operationName = strings.TrimSpace(parts[3])
	if operationName == "" {
		return "", false
	}
	return operationName, true
}

func parseGCPServiceManagementPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0
	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			respondGCPServiceManagementInvalidArgument(w, path, "pageSize must be an integer")
			return 0, 0, false
		}
		if value < 1 || value > maxPageSize {
			respondGCPServiceManagementOutOfRange(w, path, fmt.Sprintf("pageSize must be between 1 and %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = value
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPServiceManagementInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = value
	}
	return pageSize, start, true
}

func respondGCPServiceManagementList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPServiceManagementOutOfRange(w, path, "pageToken is out of range")
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

func parseGCPServiceManagementConfigView(raw string) (full bool, ok bool) {
	value := strings.TrimSpace(raw)
	switch strings.ToUpper(value) {
	case "", "BASIC", "CONFIG_VIEW_UNSPECIFIED", "0":
		return false, true
	case "FULL", "1":
		return true, true
	default:
		return false, false
	}
}

func isGCPServiceManagementRolloutFilter(filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" || strings.Contains(filter, "==") || strings.Contains(filter, "!!") {
		return false
	}
	if strings.HasPrefix(filter, "status=") {
		switch strings.TrimSpace(strings.TrimPrefix(filter, "status=")) {
		case "ROLLOUT_STATUS_UNSPECIFIED", "IN_PROGRESS", "SUCCESS", "CANCELLED", "FAILED", "PENDING", "FAILED_ROLLED_BACK":
			return true
		default:
			return false
		}
	}
	if strings.HasPrefix(filter, "strategy=") {
		switch strings.TrimSpace(strings.TrimPrefix(filter, "strategy=")) {
		case "TrafficPercentStrategy", "DeleteServiceStrategy":
			return true
		default:
			return false
		}
	}
	return false
}

func isGCPServiceManagementMalformedFilter(filter string) bool {
	filter = strings.TrimSpace(filter)
	return strings.Contains(filter, "==") || strings.Contains(filter, "!!")
}

func isGCPServiceManagementServiceName(serviceName string) bool {
	serviceName = strings.TrimSpace(serviceName)
	return serviceName != "" &&
		strings.Contains(serviceName, ".") &&
		!strings.Contains(serviceName, "/") &&
		!strings.Contains(serviceName, " ") &&
		gcpServiceManagementServiceNameRegex.MatchString(serviceName)
}

func decodeGCPServiceManagementJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		respondGCPServiceManagementInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPServiceManagementInvalidArgument(w, path, "request body could not be read")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		respondGCPServiceManagementInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	body := map[string]any{}
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPServiceManagementInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func decodeGCPServiceManagementJSONBodyOptional(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPServiceManagementInvalidArgument(w, path, "request body could not be read")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	body := map[string]any{}
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPServiceManagementInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func gcpServiceManagementManagedService(serviceName string) map[string]any {
	return map[string]any{
		"serviceName":       serviceName,
		"producerProjectId": "stackyard-project",
	}
}

func gcpServiceManagementServiceConfig(serviceName, configID string, full bool) map[string]any {
	cfg := map[string]any{
		"name":              serviceName,
		"id":                configID,
		"title":             "Stackyard Service Config",
		"producerProjectId": "stackyard-project",
		"apis": []map[string]any{
			{
				"name": serviceName,
			},
		},
		"documentation": map[string]any{
			"summary": "Stackyard staged service configuration",
		},
	}
	if full {
		cfg["sourceInfo"] = map[string]any{
			"sourceFiles": []map[string]any{
				{
					"path":     "service.yaml",
					"contents": "Y29uZmlnVmVyc2lvbjogMw==",
				},
			},
		}
	}
	return cfg
}

func gcpServiceManagementRollout(serviceName, rolloutID string) map[string]any {
	return map[string]any{
		"rolloutId":  rolloutID,
		"createTime": gcpServiceManagementReferenceTime.Add(10 * time.Minute).Format(time.RFC3339),
		"createdBy":  "stackyard@example.com",
		"status":     "SUCCESS",
		"trafficPercentStrategy": map[string]any{
			"percentages": map[string]any{
				"2026-01-01r0": 100,
			},
		},
		"serviceName": serviceName,
	}
}

func gcpServiceManagementOperation(operationID, target, action string, done bool) map[string]any {
	operation := map[string]any{
		"name": fmt.Sprintf("operations/%s", operationID),
		"done": done,
		"metadata": map[string]any{
			"@type":              "type.googleapis.com/google.api.servicemanagement.v1.OperationMetadata",
			"resourceNames":      []string{target},
			"progressPercentage": 100,
			"startTime":          gcpServiceManagementReferenceTime.Format(time.RFC3339),
			"steps": []map[string]any{
				{
					"description": "staged emulation " + action,
					"status":      "DONE",
				},
			},
		},
	}
	if done {
		operation["response"] = map[string]any{
			"@type": "type.googleapis.com/google.protobuf.Empty",
		}
	}
	return operation
}

func gcpServiceManagementGenerateConfigReport(serviceName string) map[string]any {
	return map[string]any{
		"serviceName": serviceName,
		"id":          "report-2026-01-01",
		"changeReports": []map[string]any{
			{
				"configChanges": []map[string]any{
					{
						"element":    "apis[0].name",
						"oldValue":   "old.stackyard.googleapis.com",
						"newValue":   serviceName,
						"changeType": "MODIFIED",
					},
				},
			},
		},
		"diagnostics": []map[string]any{
			{
				"location": "service.yaml:1",
				"kind":     "WARNING",
				"message":  "staged configuration warning",
			},
		},
	}
}

func gcpServiceManagementIAMPolicy(resource string) map[string]any {
	_ = resource
	return map[string]any{
		"version": 1,
		"etag":    "c3RhY2t5YXJk",
		"bindings": []map[string]any{
			{
				"role":    "roles/viewer",
				"members": []string{"user:tester@example.com"},
			},
		},
	}
}

func gcpServiceManagementString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func respondGCPServiceManagementInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPServiceManagementError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPServiceManagementFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPServiceManagementError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPServiceManagementOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPServiceManagementError(w, http.StatusBadRequest, "OutOfRange", path, message)
}

func respondGCPServiceManagementError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_servicemanagement(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if r.URL.Query().Get("stackyard_contract_probe") != "1" {
		return false
	}

	path := normalizeGCPServiceManagementPath(rawRequestPath(r))
	if !isGCPServiceManagementPath(path, true) {
		return false
	}

	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPServiceManagementInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") != "1" {
		return false
	}

	payload := gcpServiceManagementManagedService("stackyard.googleapis.com")
	payload["service"] = "servicemanagement"
	payload["provider"] = providerGCP
	payload["path"] = path
	payload["name"] = "services/stackyard.googleapis.com"
	respondJSON(w, http.StatusOK, payload)
	return true
}
