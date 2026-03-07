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
	gcpServiceUsageReferenceTime    = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpServiceUsageProjectIDRegex   = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	gcpServiceUsageServiceIDRegex   = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]*[a-z0-9]$`)
	gcpServiceUsageOperationIDRegex = regexp.MustCompile(`^serviceusage-[a-z0-9.-]+$`)
)

func (s *Server) handleGCPServiceUsageRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_serviceusage(w, r) {
		return true
	}

	path := normalizeGCPServiceUsagePath(rawRequestPath(r))
	if !isGCPServiceUsagePath(path, hasGCPServiceUsageHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPServiceUsageListServices(w, r, path) {
			return true
		}
		if handleGCPServiceUsageGetService(w, path) {
			return true
		}
		if handleGCPServiceUsageBatchGetServices(w, r, path) {
			return true
		}
		if handleGCPServiceUsageListOperations(w, r, path) {
			return true
		}
		if handleGCPServiceUsageGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPServiceUsageEnableService(w, r, path) {
			return true
		}
		if handleGCPServiceUsageDisableService(w, r, path) {
			return true
		}
		if handleGCPServiceUsageBatchEnableServices(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPServiceUsagePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPServiceUsageHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "serviceusage",
		"serviceusage-apiv1",
		"serviceusage_apiv1",
		"service-usage",
		"service_usage",
		"gcp-service-usage":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-serviceusage-apiv1") || strings.Contains(ua, "cloud.google.com/go/serviceusage")
}

func isGCPServiceUsagePath(path string, includeHint bool) bool {
	if isGCPServiceUsageGRPCPath(path, includeHint) {
		return true
	}
	if _, ok := parseGCPServiceUsageServicesCollectionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPServiceUsageServicePath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPServiceUsageServiceActionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPServiceUsageBatchActionPath(path); ok {
		return true
	}
	if parseGCPServiceUsageOperationsCollectionPath(path) {
		return includeHint
	}
	if operationID, ok := parseGCPServiceUsageOperationPath(path); ok {
		return includeHint || gcpServiceUsageOperationIDRegex.MatchString(operationID)
	}
	return includeHint && strings.HasPrefix(path, "/gcp/v1/projects/")
}

func isGCPServiceUsageGRPCPath(path string, includeHint bool) bool {
	trimmed := strings.TrimSpace(path)
	if strings.HasPrefix(trimmed, "/gcp/google.api.serviceusage.v1.ServiceUsage/") {
		return true
	}
	if includeHint && strings.HasPrefix(trimmed, "/gcp/google.longrunning.Operations/") {
		return true
	}
	return false
}

func handleGCPServiceUsageListServices(w http.ResponseWriter, r *http.Request, path string) bool {
	project, ok := parseGCPServiceUsageServicesCollectionPath(path)
	if !ok {
		return false
	}

	pageSize, start, ok := parseGCPServiceUsagePagination(w, r, path, 200)
	if !ok {
		return true
	}
	filter := strings.TrimSpace(r.URL.Query().Get("filter"))
	if filter != "" && !isGCPServiceUsageFilter(filter) {
		respondGCPServiceUsageInvalidArgument(w, path, "filter must be state:ENABLED or state:DISABLED")
		return true
	}

	items := gcpServiceUsageDefaultServices(project)
	if filter == "state:ENABLED" {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if strings.TrimSpace(gcpServiceUsageString(item, "state")) == "ENABLED" {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if filter == "state:DISABLED" {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if strings.TrimSpace(gcpServiceUsageString(item, "state")) == "DISABLED" {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	return respondGCPServiceUsageList(w, "services", items, pageSize, start, path)
}

func handleGCPServiceUsageGetService(w http.ResponseWriter, path string) bool {
	project, serviceID, ok := parseGCPServiceUsageServicePath(path)
	if !ok {
		return false
	}
	if !isGCPServiceUsageProjectID(project) {
		respondGCPServiceUsageInvalidArgument(w, path, "project is invalid")
		return true
	}
	if !isGCPServiceUsageServiceID(serviceID) {
		respondGCPServiceUsageInvalidArgument(w, path, "service name is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpServiceUsageService(project, serviceID, gcpServiceUsageDefaultState(serviceID)))
	return true
}

func handleGCPServiceUsageEnableService(w http.ResponseWriter, r *http.Request, path string) bool {
	project, serviceID, action, ok := parseGCPServiceUsageServiceActionPath(path)
	if !ok || action != "enable" {
		return false
	}

	body, ok := decodeGCPServiceUsageJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/services/%s", project, serviceID)
	name := strings.TrimSpace(gcpServiceUsageString(body, "name"))
	if name == "" {
		respondGCPServiceUsageInvalidArgument(w, path, "name is required")
		return true
	}
	if name != expectedName {
		respondGCPServiceUsageInvalidArgument(w, path, "name must match requested resource")
		return true
	}

	operationID := "serviceusage-enable-" + serviceID
	service := gcpServiceUsageService(project, serviceID, "ENABLED")
	respondJSON(w, http.StatusOK, gcpServiceUsageOperation(operationID, []string{expectedName}, map[string]any{
		"@type":   "type.googleapis.com/google.api.serviceusage.v1.EnableServiceResponse",
		"service": service,
	}))
	return true
}

func handleGCPServiceUsageDisableService(w http.ResponseWriter, r *http.Request, path string) bool {
	project, serviceID, action, ok := parseGCPServiceUsageServiceActionPath(path)
	if !ok || action != "disable" {
		return false
	}

	body, ok := decodeGCPServiceUsageJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/services/%s", project, serviceID)
	name := strings.TrimSpace(gcpServiceUsageString(body, "name"))
	if name == "" {
		respondGCPServiceUsageInvalidArgument(w, path, "name is required")
		return true
	}
	if name != expectedName {
		respondGCPServiceUsageInvalidArgument(w, path, "name must match requested resource")
		return true
	}
	if rawCheck, exists := body["checkIfServiceHasUsage"]; exists && !isGCPServiceUsageDisableUsageCheckValue(rawCheck) {
		respondGCPServiceUsageInvalidArgument(w, path, "checkIfServiceHasUsage must be UNSPECIFIED, SKIP, CHECK, or 0..2")
		return true
	}

	operationID := "serviceusage-disable-" + serviceID
	service := gcpServiceUsageService(project, serviceID, "DISABLED")
	respondJSON(w, http.StatusOK, gcpServiceUsageOperation(operationID, []string{expectedName}, map[string]any{
		"@type":   "type.googleapis.com/google.api.serviceusage.v1.DisableServiceResponse",
		"service": service,
	}))
	return true
}

func handleGCPServiceUsageBatchEnableServices(w http.ResponseWriter, r *http.Request, path string) bool {
	project, action, ok := parseGCPServiceUsageBatchActionPath(path)
	if !ok || action != "batchEnable" {
		return false
	}

	body, ok := decodeGCPServiceUsageJSONBodyRequired(w, r, path)
	if !ok {
		return true
	}
	parent := strings.TrimSpace(gcpServiceUsageString(body, "parent"))
	expectedParent := "projects/" + project
	if parent == "" {
		respondGCPServiceUsageInvalidArgument(w, path, "parent is required")
		return true
	}
	if parent != expectedParent {
		respondGCPServiceUsageInvalidArgument(w, path, "parent must match requested resource")
		return true
	}

	rawServiceIDs, ok := body["serviceIds"].([]any)
	if !ok || len(rawServiceIDs) == 0 {
		respondGCPServiceUsageInvalidArgument(w, path, "serviceIds must include at least one entry")
		return true
	}
	if len(rawServiceIDs) > 20 {
		respondGCPServiceUsageOutOfRange(w, path, "serviceIds cannot include more than 20 entries")
		return true
	}

	services := make([]map[string]any, 0, len(rawServiceIDs))
	resourceNames := make([]string, 0, len(rawServiceIDs))
	for idx, rawServiceID := range rawServiceIDs {
		serviceID, ok := rawServiceID.(string)
		serviceID = strings.TrimSpace(serviceID)
		if !ok || !isGCPServiceUsageServiceID(serviceID) {
			respondGCPServiceUsageInvalidArgument(w, path, fmt.Sprintf("serviceIds[%d] is invalid", idx))
			return true
		}
		resourceName := fmt.Sprintf("projects/%s/services/%s", project, serviceID)
		resourceNames = append(resourceNames, resourceName)
		services = append(services, gcpServiceUsageService(project, serviceID, "ENABLED"))
	}

	operationID := "serviceusage-batch-enable-" + project
	respondJSON(w, http.StatusOK, gcpServiceUsageOperation(operationID, resourceNames, map[string]any{
		"@type":    "type.googleapis.com/google.api.serviceusage.v1.BatchEnableServicesResponse",
		"services": services,
		"failures": []map[string]any{},
	}))
	return true
}

func handleGCPServiceUsageBatchGetServices(w http.ResponseWriter, r *http.Request, path string) bool {
	project, action, ok := parseGCPServiceUsageBatchActionPath(path)
	if !ok || action != "batchGet" {
		return false
	}

	names := r.URL.Query()["names"]
	if len(names) == 0 {
		respondGCPServiceUsageInvalidArgument(w, path, "names must include at least one entry")
		return true
	}
	if len(names) > 30 {
		respondGCPServiceUsageOutOfRange(w, path, "names cannot include more than 30 entries")
		return true
	}

	items := make([]map[string]any, 0, len(names))
	for idx, name := range names {
		nameProject, serviceID, valid := parseGCPServiceUsageServiceResourceName(name)
		if !valid || nameProject != project {
			respondGCPServiceUsageInvalidArgument(w, path, fmt.Sprintf("names[%d] must match parent project", idx))
			return true
		}
		items = append(items, gcpServiceUsageService(project, serviceID, gcpServiceUsageDefaultState(serviceID)))
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"services": items,
	})
	return true
}

func handleGCPServiceUsageListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPServiceUsageOperationsCollectionPath(path) {
		return false
	}

	pageSize, start, ok := parseGCPServiceUsagePagination(w, r, path, 100)
	if !ok {
		return true
	}
	if filter := strings.TrimSpace(r.URL.Query().Get("filter")); filter != "" && isGCPServiceUsageMalformedFilter(filter) {
		respondGCPServiceUsageInvalidArgument(w, path, "filter is malformed")
		return true
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name != "" && name != "operations" && !strings.HasPrefix(name, "operations/") {
		respondGCPServiceUsageInvalidArgument(w, path, "name must be operations or start with operations/")
		return true
	}

	items := []map[string]any{
		gcpServiceUsageOperationFromID("stackyard", "serviceusage-enable-serviceusage.googleapis.com"),
		gcpServiceUsageOperationFromID("stackyard", "serviceusage-batch-enable-stackyard"),
	}
	if name != "" {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			operationName := strings.TrimSpace(gcpServiceUsageString(item, "name"))
			if strings.HasPrefix(operationName, name) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	return respondGCPServiceUsageList(w, "operations", items, pageSize, start, path)
}

func handleGCPServiceUsageGetOperation(w http.ResponseWriter, path string) bool {
	operationID, ok := parseGCPServiceUsageOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpServiceUsageOperationFromID("stackyard", operationID))
	return true
}

func parseGCPServiceUsageServicesCollectionPath(path string) (project string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "services" {
		return "", false
	}
	project = strings.TrimSpace(parts[3])
	if !isGCPServiceUsageProjectID(project) {
		return "", false
	}
	return project, true
}

func parseGCPServiceUsageServicePath(path string) (project, serviceID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "services" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	serviceID = strings.TrimSpace(parts[5])
	if !isGCPServiceUsageProjectID(project) || !isGCPServiceUsageServiceID(serviceID) || strings.Contains(serviceID, ":") {
		return "", "", false
	}
	return project, serviceID, true
}

func parseGCPServiceUsageServiceActionPath(path string) (project, serviceID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "services" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	serviceAction := strings.TrimSpace(parts[5])
	serviceID, action, ok = strings.Cut(serviceAction, ":")
	if !ok {
		return "", "", "", false
	}
	serviceID = strings.TrimSpace(serviceID)
	action = strings.TrimSpace(action)
	if !isGCPServiceUsageProjectID(project) || !isGCPServiceUsageServiceID(serviceID) || action == "" {
		return "", "", "", false
	}
	return project, serviceID, action, true
}

func parseGCPServiceUsageBatchActionPath(path string) (project, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	if !isGCPServiceUsageProjectID(project) {
		return "", "", false
	}
	if parts[4] == "services:batchEnable" {
		return project, "batchEnable", true
	}
	if parts[4] == "services:batchGet" {
		return project, "batchGet", true
	}
	return "", "", false
}

func parseGCPServiceUsageOperationsCollectionPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "operations"
}

func parseGCPServiceUsageOperationPath(path string) (operationID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "operations" {
		return "", false
	}
	operationID = strings.TrimSpace(parts[3])
	if operationID == "" {
		return "", false
	}
	return operationID, true
}

func parseGCPServiceUsagePagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0
	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			respondGCPServiceUsageInvalidArgument(w, path, "pageSize must be an integer")
			return 0, 0, false
		}
		if value < 1 || value > maxPageSize {
			respondGCPServiceUsageOutOfRange(w, path, fmt.Sprintf("pageSize must be between 1 and %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = value
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPServiceUsageInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = value
	}
	return pageSize, start, true
}

func decodeGCPServiceUsageJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		respondGCPServiceUsageInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPServiceUsageInvalidArgument(w, path, "request body could not be read")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		respondGCPServiceUsageInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	body := map[string]any{}
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPServiceUsageInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func respondGCPServiceUsageList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPServiceUsageOutOfRange(w, path, "pageToken is out of range")
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

func parseGCPServiceUsageParent(parent string) (project string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 2 || parts[0] != "projects" {
		return "", false
	}
	project = strings.TrimSpace(parts[1])
	if !isGCPServiceUsageProjectID(project) {
		return "", false
	}
	return project, true
}

func parseGCPServiceUsageServiceResourceName(name string) (project, serviceID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "services" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	serviceID = strings.TrimSpace(parts[3])
	if !isGCPServiceUsageProjectID(project) || !isGCPServiceUsageServiceID(serviceID) {
		return "", "", false
	}
	return project, serviceID, true
}

func isGCPServiceUsageProjectID(project string) bool {
	return gcpServiceUsageProjectIDRegex.MatchString(strings.TrimSpace(project))
}

func isGCPServiceUsageServiceID(serviceID string) bool {
	serviceID = strings.TrimSpace(serviceID)
	return strings.Contains(serviceID, ".") && gcpServiceUsageServiceIDRegex.MatchString(serviceID)
}

func isGCPServiceUsageFilter(filter string) bool {
	switch strings.TrimSpace(filter) {
	case "state:ENABLED", "state:DISABLED":
		return true
	default:
		return false
	}
}

func isGCPServiceUsageMalformedFilter(filter string) bool {
	filter = strings.TrimSpace(filter)
	return strings.Contains(filter, "==") || strings.Contains(filter, "!!")
}

func isGCPServiceUsageDisableUsageCheckValue(value any) bool {
	switch v := value.(type) {
	case float64:
		intVal := int(v)
		return float64(intVal) == v && intVal >= 0 && intVal <= 2
	case string:
		switch strings.ToUpper(strings.TrimSpace(v)) {
		case "", "CHECK_IF_SERVICE_HAS_USAGE_UNSPECIFIED", "SKIP", "CHECK", "0", "1", "2":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func gcpServiceUsageDefaultServices(project string) []map[string]any {
	return []map[string]any{
		gcpServiceUsageService(project, "serviceusage.googleapis.com", "ENABLED"),
		gcpServiceUsageService(project, "stackyard.googleapis.com", "ENABLED"),
		gcpServiceUsageService(project, "compute.googleapis.com", "DISABLED"),
	}
}

func gcpServiceUsageDefaultState(serviceID string) string {
	switch strings.TrimSpace(serviceID) {
	case "compute.googleapis.com":
		return "DISABLED"
	default:
		return "ENABLED"
	}
}

func gcpServiceUsageService(project, serviceID, state string) map[string]any {
	name := fmt.Sprintf("projects/%s/services/%s", project, serviceID)
	return map[string]any{
		"name":   name,
		"parent": "projects/" + project,
		"state":  state,
		"config": map[string]any{
			"name":  serviceID,
			"title": "Stackyard " + serviceID,
			"documentation": map[string]any{
				"summary": "Stackyard staged service configuration",
			},
			"apis": []map[string]any{
				{
					"name":    serviceID,
					"version": "v1",
				},
			},
			"endpoints": []map[string]any{
				{
					"name": serviceID,
				},
			},
		},
	}
}

func gcpServiceUsageOperation(operationID string, resourceNames []string, response map[string]any) map[string]any {
	operation := map[string]any{
		"name": "operations/" + operationID,
		"done": true,
		"metadata": map[string]any{
			"@type":         "type.googleapis.com/google.api.serviceusage.v1.OperationMetadata",
			"resourceNames": resourceNames,
		},
	}
	if len(response) > 0 {
		operation["response"] = response
	}
	return operation
}

func gcpServiceUsageOperationFromID(project, operationID string) map[string]any {
	switch {
	case strings.HasPrefix(operationID, "serviceusage-enable-"):
		serviceID := strings.TrimSpace(strings.TrimPrefix(operationID, "serviceusage-enable-"))
		if !isGCPServiceUsageServiceID(serviceID) {
			serviceID = "serviceusage.googleapis.com"
		}
		service := gcpServiceUsageService(project, serviceID, "ENABLED")
		return gcpServiceUsageOperation(operationID, []string{fmt.Sprintf("projects/%s/services/%s", project, serviceID)}, map[string]any{
			"@type":   "type.googleapis.com/google.api.serviceusage.v1.EnableServiceResponse",
			"service": service,
		})
	case strings.HasPrefix(operationID, "serviceusage-disable-"):
		serviceID := strings.TrimSpace(strings.TrimPrefix(operationID, "serviceusage-disable-"))
		if !isGCPServiceUsageServiceID(serviceID) {
			serviceID = "serviceusage.googleapis.com"
		}
		service := gcpServiceUsageService(project, serviceID, "DISABLED")
		return gcpServiceUsageOperation(operationID, []string{fmt.Sprintf("projects/%s/services/%s", project, serviceID)}, map[string]any{
			"@type":   "type.googleapis.com/google.api.serviceusage.v1.DisableServiceResponse",
			"service": service,
		})
	case strings.HasPrefix(operationID, "serviceusage-batch-enable-"):
		services := []map[string]any{
			gcpServiceUsageService(project, "serviceusage.googleapis.com", "ENABLED"),
			gcpServiceUsageService(project, "stackyard.googleapis.com", "ENABLED"),
		}
		resourceNames := []string{
			fmt.Sprintf("projects/%s/services/serviceusage.googleapis.com", project),
			fmt.Sprintf("projects/%s/services/stackyard.googleapis.com", project),
		}
		return gcpServiceUsageOperation(operationID, resourceNames, map[string]any{
			"@type":    "type.googleapis.com/google.api.serviceusage.v1.BatchEnableServicesResponse",
			"services": services,
			"failures": []map[string]any{},
		})
	default:
		return gcpServiceUsageOperation(operationID, []string{
			fmt.Sprintf("projects/%s/services/serviceusage.googleapis.com", project),
		}, map[string]any{
			"@type": "type.googleapis.com/google.protobuf.Empty",
		})
	}
}

func gcpServiceUsageString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func respondGCPServiceUsageInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPServiceUsageError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPServiceUsageFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPServiceUsageError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPServiceUsageOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPServiceUsageError(w, http.StatusBadRequest, "OutOfRange", path, message)
}

func respondGCPServiceUsageError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_serviceusage(w http.ResponseWriter, r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	if r.URL.Query().Get("stackyard_contract_probe") != "1" {
		return false
	}

	path := normalizeGCPServiceUsagePath(rawRequestPath(r))
	if !isGCPServiceUsagePath(path, true) {
		return false
	}

	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPServiceUsageInvalidArgument(w, path, "pageSize must be an integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") != "1" {
		return false
	}

	payload := gcpServiceUsageService("stackyard", "serviceusage.googleapis.com", "ENABLED")
	payload["service"] = "serviceusage"
	payload["provider"] = providerGCP
	payload["path"] = path
	respondJSON(w, http.StatusOK, payload)
	return true
}
