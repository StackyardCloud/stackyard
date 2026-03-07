package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const gcpSpannerAdminInstanceMaxPageSize = 1000

var gcpSpannerAdminInstanceReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPSpannerAdminInstanceRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_spanner_admin_instance(w, r) {
		return true
	}

	path := normalizeGCPSpannerAdminInstancePath(rawRequestPath(r))
	if !isGCPSpannerAdminInstancePath(path, hasGCPSpannerAdminInstanceHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPSpannerAdminInstanceListInstanceConfigs(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceGetInstanceConfig(w, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceListInstanceConfigOperations(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceListInstances(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceGetInstance(w, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceListInstancePartitions(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceGetInstancePartition(w, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceListInstancePartitionOperations(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceGetIAMPolicy(w, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceGetOperation(w, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceListOperations(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPSpannerAdminInstanceCreateInstanceConfig(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceCreateInstance(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceCreateInstancePartition(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceMoveInstance(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceSetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceTestIAMPermissions(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceGetIAMPolicy(w, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceCancelOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPSpannerAdminInstanceUpdateInstanceConfig(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceUpdateInstance(w, r, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceUpdateInstancePartition(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPSpannerAdminInstanceDeleteInstanceConfig(w, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceDeleteInstance(w, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceDeleteInstancePartition(w, path) {
			return true
		}
		if handleGCPSpannerAdminInstanceDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPSpannerAdminInstancePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPSpannerAdminInstanceHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "spanner_admin_instance", "spanner-admin-instance", "spanner-admin-instance-apiv1", "spanner_admin_instance_apiv1", "cloud-spanner-admin-instance", "cloud_spanner_admin_instance", "cloudspanneradmininstance", "gcp-cloud-spanner-admin-instance":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-spanner-admin-instance-apiv1") || strings.Contains(ua, "cloud.google.com/go/spanner/admin/instance")
}

func isGCPSpannerAdminInstancePath(path string, includeHint bool) bool {
	if _, _, ok := parseGCPSpannerAdminInstanceIAMActionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPSpannerAdminInstanceOperationPath(path); ok {
		return true
	}
	if _, scope, ok := parseGCPSpannerAdminInstanceOperationsCollectionPath(path); ok {
		if scope == "project" {
			return includeHint
		}
		return true
	}
	if _, _, _, ok := parseGCPSpannerAdminInstanceOperationActionPath(path, "cancel"); ok {
		return true
	}
	if _, _, _, ok := parseGCPSpannerAdminInstanceConfigPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPSpannerAdminInstancePath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPSpannerAdminInstancePartitionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPSpannerAdminInstanceConfigOperationsCollectionPath(path); ok {
		return true
	}
	if _, _, _, ok := parseGCPSpannerAdminInstancePartitionOperationsCollectionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPSpannerAdminInstanceMovePath(path); ok {
		return true
	}
	if includeHint {
		return strings.HasPrefix(path, "/gcp/v1/projects/") &&
			(strings.Contains(path, "/instances") || strings.Contains(path, "/instanceConfigs"))
	}
	return false
}

func handleGCPSpannerAdminInstanceListInstanceConfigs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, configID, list, ok := parseGCPSpannerAdminInstanceConfigPath(path)
	if !ok || !list || configID != "" {
		return false
	}
	pageSize, start, valid := parseGCPSpannerAdminInstancePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSpannerAdminInstanceConfigFixture(project, "custom-stackyard-primary"),
		gcpSpannerAdminInstanceConfigFixture(project, "custom-stackyard-analytics"),
	}
	return respondGCPSpannerAdminInstanceList(w, "instanceConfigs", items, pageSize, start, path)
}

func handleGCPSpannerAdminInstanceGetInstanceConfig(w http.ResponseWriter, path string) bool {
	project, configID, list, ok := parseGCPSpannerAdminInstanceConfigPath(path)
	if !ok || list || configID == "" {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, configID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "instance config not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerAdminInstanceConfigFixture(project, configID))
	return true
}

func handleGCPSpannerAdminInstanceCreateInstanceConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	project, configID, list, ok := parseGCPSpannerAdminInstanceConfigPath(path)
	if !ok || !list || configID != "" {
		return false
	}
	body, valid := decodeGCPSpannerAdminInstanceJSONBody(w, r, path)
	if !valid {
		return true
	}
	parent := strings.TrimSpace(gcpSpannerAdminInstanceString(body, "parent"))
	if parent != "" && parent != fmt.Sprintf("projects/%s", project) {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "parent must match requested project")
		return true
	}
	id := strings.TrimSpace(r.URL.Query().Get("instanceConfigId"))
	if id == "" {
		id = strings.TrimSpace(gcpSpannerAdminInstanceString(body, "instanceConfigId"))
	}
	if id == "" {
		id = strings.TrimSpace(gcpSpannerAdminInstanceString(body, "instance_config_id"))
	}
	if !isGCPSpannerAdminInstanceIdentifier(id, 64) {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "instanceConfigId is required")
		return true
	}
	instanceConfig := gcpSpannerAdminInstanceBodyMap(body, "instanceConfig")
	if len(instanceConfig) == 0 {
		instanceConfig = gcpSpannerAdminInstanceBodyMap(body, "instance_config")
	}
	if len(instanceConfig) == 0 {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "instanceConfig is required")
		return true
	}
	if isGCPSpannerAdminInstanceAlreadyExists(id) {
		respondGCPSpannerAdminInstanceAlreadyExists(w, path, "instance config already exists")
		return true
	}
	response := gcpSpannerAdminInstanceConfigFixture(project, id)
	if displayName := strings.TrimSpace(gcpSpannerAdminInstanceString(instanceConfig, "displayName")); displayName != "" {
		response["displayName"] = displayName
	}
	op := gcpSpannerAdminInstanceOperationFixture(project, "create-instance-config-"+id, response)
	op["metadata"] = map[string]any{
		"@type":          "type.googleapis.com/google.spanner.admin.instance.v1.CreateInstanceConfigMetadata",
		"instanceConfig": fmt.Sprintf("projects/%s/instanceConfigs/%s", project, id),
	}
	respondJSON(w, http.StatusOK, op)
	return true
}

func handleGCPSpannerAdminInstanceUpdateInstanceConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	project, configID, list, ok := parseGCPSpannerAdminInstanceConfigPath(path)
	if !ok || list || configID == "" {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, configID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "instance config not found")
		return true
	}
	body, valid := decodeGCPSpannerAdminInstanceJSONBody(w, r, path)
	if !valid {
		return true
	}
	instanceConfig := gcpSpannerAdminInstanceBodyMap(body, "instanceConfig")
	if len(instanceConfig) == 0 {
		instanceConfig = gcpSpannerAdminInstanceBodyMap(body, "instance_config")
	}
	if len(instanceConfig) == 0 {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "instanceConfig is required")
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/instanceConfigs/%s", project, configID)
	if name := strings.TrimSpace(gcpSpannerAdminInstanceString(instanceConfig, "name")); name == "" || name != expectedName {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "instanceConfig.name must match requested resource")
		return true
	}
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		updateMask = strings.TrimSpace(gcpSpannerAdminInstanceString(body, "updateMask"))
	}
	if updateMask == "" {
		updateMask = strings.TrimSpace(gcpSpannerAdminInstanceString(body, "update_mask"))
	}
	if updateMask == "" {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "updateMask is required")
		return true
	}
	response := gcpSpannerAdminInstanceConfigFixture(project, configID)
	if displayName := strings.TrimSpace(gcpSpannerAdminInstanceString(instanceConfig, "displayName")); displayName != "" {
		response["displayName"] = displayName
	}
	op := gcpSpannerAdminInstanceOperationFixture(project, "update-instance-config-"+configID, response)
	op["metadata"] = map[string]any{
		"@type":          "type.googleapis.com/google.spanner.admin.instance.v1.UpdateInstanceConfigMetadata",
		"instanceConfig": expectedName,
		"updateMask":     updateMask,
	}
	respondJSON(w, http.StatusOK, op)
	return true
}

func handleGCPSpannerAdminInstanceDeleteInstanceConfig(w http.ResponseWriter, path string) bool {
	project, configID, list, ok := parseGCPSpannerAdminInstanceConfigPath(path)
	if !ok || list || configID == "" {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, configID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "instance config not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSpannerAdminInstanceListInstanceConfigOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, configID, ok := parseGCPSpannerAdminInstanceConfigOperationsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSpannerAdminInstancePagination(w, r, path)
	if !valid {
		return true
	}
	opPrefix := "instance-config"
	if configID != "" {
		opPrefix = configID
	}
	items := []map[string]any{
		gcpSpannerAdminInstanceOperationFixture(project, "create-"+opPrefix, map[string]any{}),
		gcpSpannerAdminInstanceOperationFixture(project, "update-"+opPrefix, map[string]any{}),
	}
	return respondGCPSpannerAdminInstanceList(w, "operations", items, pageSize, start, path)
}

func handleGCPSpannerAdminInstanceListInstances(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instanceID, list, ok := parseGCPSpannerAdminInstancePath(path)
	if !ok || !list || instanceID != "" {
		return false
	}
	pageSize, start, valid := parseGCPSpannerAdminInstancePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSpannerAdminInstanceFixture(project, "stackyard-instance"),
		gcpSpannerAdminInstanceFixture(project, "analytics-instance"),
	}
	return respondGCPSpannerAdminInstanceList(w, "instances", items, pageSize, start, path)
}

func handleGCPSpannerAdminInstanceGetInstance(w http.ResponseWriter, path string) bool {
	project, instanceID, list, ok := parseGCPSpannerAdminInstancePath(path)
	if !ok || list || instanceID == "" {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, instanceID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "instance not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerAdminInstanceFixture(project, instanceID))
	return true
}

func handleGCPSpannerAdminInstanceCreateInstance(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instanceID, list, ok := parseGCPSpannerAdminInstancePath(path)
	if !ok || !list || instanceID != "" {
		return false
	}
	body, valid := decodeGCPSpannerAdminInstanceJSONBody(w, r, path)
	if !valid {
		return true
	}
	parent := strings.TrimSpace(gcpSpannerAdminInstanceString(body, "parent"))
	if parent != "" && parent != fmt.Sprintf("projects/%s", project) {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "parent must match requested project")
		return true
	}
	id := strings.TrimSpace(r.URL.Query().Get("instanceId"))
	if id == "" {
		id = strings.TrimSpace(gcpSpannerAdminInstanceString(body, "instanceId"))
	}
	if id == "" {
		id = strings.TrimSpace(gcpSpannerAdminInstanceString(body, "instance_id"))
	}
	if !isGCPSpannerAdminInstanceIdentifier(id, 64) {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "instanceId is required")
		return true
	}
	instance := gcpSpannerAdminInstanceBodyMap(body, "instance")
	if len(instance) == 0 {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "instance is required")
		return true
	}
	if cfg := strings.TrimSpace(gcpSpannerAdminInstanceString(instance, "config")); cfg == "" {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "instance.config is required")
		return true
	}
	if displayName := strings.TrimSpace(gcpSpannerAdminInstanceString(instance, "displayName")); displayName == "" {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "instance.displayName is required")
		return true
	}
	if isGCPSpannerAdminInstanceAlreadyExists(id) {
		respondGCPSpannerAdminInstanceAlreadyExists(w, path, "instance already exists")
		return true
	}
	response := gcpSpannerAdminInstanceFixture(project, id)
	if displayName := strings.TrimSpace(gcpSpannerAdminInstanceString(instance, "displayName")); displayName != "" {
		response["displayName"] = displayName
	}
	if nodeCount, ok := gcpSpannerAdminInstanceInt(instance["nodeCount"]); ok {
		response["nodeCount"] = nodeCount
	}
	op := gcpSpannerAdminInstanceOperationFixture(project, "create-instance-"+id, response)
	op["metadata"] = map[string]any{
		"@type":    "type.googleapis.com/google.spanner.admin.instance.v1.CreateInstanceMetadata",
		"instance": fmt.Sprintf("projects/%s/instances/%s", project, id),
	}
	respondJSON(w, http.StatusOK, op)
	return true
}

func handleGCPSpannerAdminInstanceUpdateInstance(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instanceID, list, ok := parseGCPSpannerAdminInstancePath(path)
	if !ok || list || instanceID == "" {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, instanceID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "instance not found")
		return true
	}
	body, valid := decodeGCPSpannerAdminInstanceJSONBody(w, r, path)
	if !valid {
		return true
	}
	instance := gcpSpannerAdminInstanceBodyMap(body, "instance")
	if len(instance) == 0 {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "instance is required")
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/instances/%s", project, instanceID)
	if name := strings.TrimSpace(gcpSpannerAdminInstanceString(instance, "name")); name == "" || name != expectedName {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "instance.name must match requested resource")
		return true
	}
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		updateMask = strings.TrimSpace(gcpSpannerAdminInstanceString(body, "fieldMask"))
	}
	if updateMask == "" {
		updateMask = strings.TrimSpace(gcpSpannerAdminInstanceString(body, "field_mask"))
	}
	if updateMask == "" {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "fieldMask is required")
		return true
	}
	response := gcpSpannerAdminInstanceFixture(project, instanceID)
	if displayName := strings.TrimSpace(gcpSpannerAdminInstanceString(instance, "displayName")); displayName != "" {
		response["displayName"] = displayName
	}
	if nodeCount, ok := gcpSpannerAdminInstanceInt(instance["nodeCount"]); ok {
		response["nodeCount"] = nodeCount
	}
	op := gcpSpannerAdminInstanceOperationFixture(project, "update-instance-"+instanceID, response)
	op["metadata"] = map[string]any{
		"@type":    "type.googleapis.com/google.spanner.admin.instance.v1.UpdateInstanceMetadata",
		"instance": expectedName,
	}
	respondJSON(w, http.StatusOK, op)
	return true
}

func handleGCPSpannerAdminInstanceDeleteInstance(w http.ResponseWriter, path string) bool {
	project, instanceID, list, ok := parseGCPSpannerAdminInstancePath(path)
	if !ok || list || instanceID == "" {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, instanceID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "instance not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSpannerAdminInstanceMoveInstance(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instanceID, ok := parseGCPSpannerAdminInstanceMovePath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, instanceID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "instance not found")
		return true
	}
	body, valid := decodeGCPSpannerAdminInstanceJSONBody(w, r, path)
	if !valid {
		return true
	}
	name := strings.TrimSpace(gcpSpannerAdminInstanceString(body, "name"))
	expectedName := fmt.Sprintf("projects/%s/instances/%s", project, instanceID)
	if name != "" && name != expectedName {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "name must match requested resource")
		return true
	}
	targetConfig := strings.TrimSpace(gcpSpannerAdminInstanceString(body, "targetConfig"))
	if targetConfig == "" {
		targetConfig = strings.TrimSpace(gcpSpannerAdminInstanceString(body, "target_config"))
	}
	if targetConfig == "" {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "targetConfig is required")
		return true
	}
	if strings.HasSuffix(targetConfig, "/custom-stackyard-primary") {
		respondGCPSpannerAdminInstanceFailedPrecondition(w, path, "instance already uses requested targetConfig")
		return true
	}
	op := gcpSpannerAdminInstanceOperationFixture(project, "move-instance-"+instanceID, map[string]any{})
	op["metadata"] = map[string]any{
		"@type":        "type.googleapis.com/google.spanner.admin.instance.v1.MoveInstanceMetadata",
		"source":       expectedName,
		"targetConfig": targetConfig,
	}
	respondJSON(w, http.StatusOK, op)
	return true
}

func handleGCPSpannerAdminInstanceListInstancePartitions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instanceID, partitionID, list, ok := parseGCPSpannerAdminInstancePartitionPath(path)
	if !ok || !list || partitionID != "" {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, instanceID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "instance not found")
		return true
	}
	pageSize, start, valid := parseGCPSpannerAdminInstancePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSpannerAdminInstancePartitionFixture(project, instanceID, "partition-a"),
		gcpSpannerAdminInstancePartitionFixture(project, instanceID, "partition-b"),
	}
	return respondGCPSpannerAdminInstanceList(w, "instancePartitions", items, pageSize, start, path)
}

func handleGCPSpannerAdminInstanceGetInstancePartition(w http.ResponseWriter, path string) bool {
	project, instanceID, partitionID, list, ok := parseGCPSpannerAdminInstancePartitionPath(path)
	if !ok || list || partitionID == "" {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, instanceID, partitionID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "instance partition not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerAdminInstancePartitionFixture(project, instanceID, partitionID))
	return true
}

func handleGCPSpannerAdminInstanceCreateInstancePartition(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instanceID, partitionID, list, ok := parseGCPSpannerAdminInstancePartitionPath(path)
	if !ok || !list || partitionID != "" {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, instanceID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "instance not found")
		return true
	}
	body, valid := decodeGCPSpannerAdminInstanceJSONBody(w, r, path)
	if !valid {
		return true
	}
	parent := strings.TrimSpace(gcpSpannerAdminInstanceString(body, "parent"))
	if parent != "" && parent != fmt.Sprintf("projects/%s/instances/%s", project, instanceID) {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "parent must match requested instance")
		return true
	}
	id := strings.TrimSpace(r.URL.Query().Get("instancePartitionId"))
	if id == "" {
		id = strings.TrimSpace(gcpSpannerAdminInstanceString(body, "instancePartitionId"))
	}
	if id == "" {
		id = strings.TrimSpace(gcpSpannerAdminInstanceString(body, "instance_partition_id"))
	}
	if !isGCPSpannerAdminInstanceIdentifier(id, 64) {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "instancePartitionId is required")
		return true
	}
	partition := gcpSpannerAdminInstanceBodyMap(body, "instancePartition")
	if len(partition) == 0 {
		partition = gcpSpannerAdminInstanceBodyMap(body, "instance_partition")
	}
	if len(partition) == 0 {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "instancePartition is required")
		return true
	}
	if isGCPSpannerAdminInstanceAlreadyExists(id) {
		respondGCPSpannerAdminInstanceAlreadyExists(w, path, "instance partition already exists")
		return true
	}
	response := gcpSpannerAdminInstancePartitionFixture(project, instanceID, id)
	if displayName := strings.TrimSpace(gcpSpannerAdminInstanceString(partition, "displayName")); displayName != "" {
		response["displayName"] = displayName
	}
	op := gcpSpannerAdminInstanceOperationFixture(project, "create-instance-partition-"+id, response)
	op["metadata"] = map[string]any{
		"@type":             "type.googleapis.com/google.spanner.admin.instance.v1.CreateInstancePartitionMetadata",
		"instancePartition": fmt.Sprintf("projects/%s/instances/%s/instancePartitions/%s", project, instanceID, id),
	}
	respondJSON(w, http.StatusOK, op)
	return true
}

func handleGCPSpannerAdminInstanceUpdateInstancePartition(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instanceID, partitionID, list, ok := parseGCPSpannerAdminInstancePartitionPath(path)
	if !ok || list || partitionID == "" {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, instanceID, partitionID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "instance partition not found")
		return true
	}
	body, valid := decodeGCPSpannerAdminInstanceJSONBody(w, r, path)
	if !valid {
		return true
	}
	partition := gcpSpannerAdminInstanceBodyMap(body, "instancePartition")
	if len(partition) == 0 {
		partition = gcpSpannerAdminInstanceBodyMap(body, "instance_partition")
	}
	if len(partition) == 0 {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "instancePartition is required")
		return true
	}
	expectedName := fmt.Sprintf("projects/%s/instances/%s/instancePartitions/%s", project, instanceID, partitionID)
	if name := strings.TrimSpace(gcpSpannerAdminInstanceString(partition, "name")); name == "" || name != expectedName {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "instancePartition.name must match requested resource")
		return true
	}
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		updateMask = strings.TrimSpace(gcpSpannerAdminInstanceString(body, "fieldMask"))
	}
	if updateMask == "" {
		updateMask = strings.TrimSpace(gcpSpannerAdminInstanceString(body, "field_mask"))
	}
	if updateMask == "" {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "fieldMask is required")
		return true
	}
	response := gcpSpannerAdminInstancePartitionFixture(project, instanceID, partitionID)
	if displayName := strings.TrimSpace(gcpSpannerAdminInstanceString(partition, "displayName")); displayName != "" {
		response["displayName"] = displayName
	}
	op := gcpSpannerAdminInstanceOperationFixture(project, "update-instance-partition-"+partitionID, response)
	op["metadata"] = map[string]any{
		"@type":             "type.googleapis.com/google.spanner.admin.instance.v1.UpdateInstancePartitionMetadata",
		"instancePartition": expectedName,
	}
	respondJSON(w, http.StatusOK, op)
	return true
}

func handleGCPSpannerAdminInstanceDeleteInstancePartition(w http.ResponseWriter, path string) bool {
	project, instanceID, partitionID, list, ok := parseGCPSpannerAdminInstancePartitionPath(path)
	if !ok || list || partitionID == "" {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, instanceID, partitionID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "instance partition not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSpannerAdminInstanceListInstancePartitionOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, instanceID, partitionID, ok := parseGCPSpannerAdminInstancePartitionOperationsCollectionPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, instanceID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "instance not found")
		return true
	}
	pageSize, start, valid := parseGCPSpannerAdminInstancePagination(w, r, path)
	if !valid {
		return true
	}
	opSuffix := instanceID
	if partitionID != "" {
		opSuffix = partitionID
	}
	items := []map[string]any{
		gcpSpannerAdminInstanceOperationFixture(project, "create-partition-"+opSuffix, map[string]any{}),
		gcpSpannerAdminInstanceOperationFixture(project, "update-partition-"+opSuffix, map[string]any{}),
	}
	return respondGCPSpannerAdminInstanceList(w, "operations", items, pageSize, start, path)
}

func handleGCPSpannerAdminInstanceSetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	resource, action, ok := parseGCPSpannerAdminInstanceIAMActionPath(path)
	if !ok || action != "setIamPolicy" {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(resource) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "resource not found")
		return true
	}
	body, valid := decodeGCPSpannerAdminInstanceJSONBody(w, r, path)
	if !valid {
		return true
	}
	policy := gcpSpannerAdminInstanceBodyMap(body, "policy")
	if len(policy) == 0 {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "policy is required")
		return true
	}
	response := gcpSpannerAdminInstancePolicyFixture(resource)
	if bindings, ok := policy["bindings"].([]any); ok {
		response["bindings"] = bindings
	}
	if etag := strings.TrimSpace(gcpSpannerAdminInstanceString(policy, "etag")); etag != "" {
		response["etag"] = etag
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func handleGCPSpannerAdminInstanceGetIAMPolicy(w http.ResponseWriter, path string) bool {
	resource, action, ok := parseGCPSpannerAdminInstanceIAMActionPath(path)
	if !ok || action != "getIamPolicy" {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(resource) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "resource not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerAdminInstancePolicyFixture(resource))
	return true
}

func handleGCPSpannerAdminInstanceTestIAMPermissions(w http.ResponseWriter, r *http.Request, path string) bool {
	resource, action, ok := parseGCPSpannerAdminInstanceIAMActionPath(path)
	if !ok || action != "testIamPermissions" {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(resource) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "resource not found")
		return true
	}
	body, valid := decodeGCPSpannerAdminInstanceJSONBody(w, r, path)
	if !valid {
		return true
	}
	permissions := gcpSpannerAdminInstanceStringSlice(body["permissions"])
	if len(permissions) == 0 {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "permissions is required")
		return true
	}
	filtered := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		if strings.Contains(permission, "spanner") {
			filtered = append(filtered, permission)
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"permissions": filtered})
	return true
}

func handleGCPSpannerAdminInstanceGetOperation(w http.ResponseWriter, path string) bool {
	project, _, opID, ok := parseGCPSpannerAdminInstanceOperationPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, opID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpSpannerAdminInstanceOperationFixture(project, opID, map[string]any{}))
	return true
}

func handleGCPSpannerAdminInstanceListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, scope, ok := parseGCPSpannerAdminInstanceOperationsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPSpannerAdminInstancePagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpSpannerAdminInstanceOperationFixture(project, "create-"+scope, map[string]any{}),
		gcpSpannerAdminInstanceOperationFixture(project, "update-"+scope, map[string]any{}),
	}
	return respondGCPSpannerAdminInstanceList(w, "operations", items, pageSize, start, path)
}

func handleGCPSpannerAdminInstanceCancelOperation(w http.ResponseWriter, path string) bool {
	project, _, opID, ok := parseGCPSpannerAdminInstanceOperationActionPath(path, "cancel")
	if !ok {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, opID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPSpannerAdminInstanceDeleteOperation(w http.ResponseWriter, path string) bool {
	project, _, opID, ok := parseGCPSpannerAdminInstanceOperationPath(path)
	if !ok {
		return false
	}
	if isGCPSpannerAdminInstanceMissingResource(project, opID) {
		respondGCPSpannerAdminInstanceNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPSpannerAdminInstanceConfigPath(path string) (project, configID string, list, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 {
		return "", "", false, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return "", "", false, false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" || parts[4] != "instanceConfigs" {
		return "", "", false, false
	}
	if len(parts) == 5 {
		return project, "", true, true
	}
	if len(parts) == 6 {
		id := strings.TrimSpace(parts[5])
		if id == "" || strings.Contains(id, ":") {
			return "", "", false, false
		}
		return project, id, false, true
	}
	return "", "", false, false
}

func parseGCPSpannerAdminInstancePath(path string) (project, instanceID string, list, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 5 {
		return "", "", false, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" {
		return "", "", false, false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" || parts[4] != "instances" {
		return "", "", false, false
	}
	if len(parts) == 5 {
		return project, "", true, true
	}
	if len(parts) == 6 {
		id := strings.TrimSpace(parts[5])
		if id == "" || strings.Contains(id, ":") {
			return "", "", false, false
		}
		return project, id, false, true
	}
	return "", "", false, false
}

func parseGCPSpannerAdminInstanceMovePath(path string) (project, instanceID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "instances" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[3])
	raw := strings.TrimSpace(parts[5])
	instanceID, action, found := strings.Cut(raw, ":")
	if !found || strings.TrimSpace(action) != "move" {
		return "", "", false
	}
	instanceID = strings.TrimSpace(instanceID)
	if project == "" || instanceID == "" {
		return "", "", false
	}
	return project, instanceID, true
}

func parseGCPSpannerAdminInstancePartitionPath(path string) (project, instanceID, partitionID string, list, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 {
		return "", "", "", false, false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "instances" {
		return "", "", "", false, false
	}
	project = strings.TrimSpace(parts[3])
	instanceID = strings.TrimSpace(parts[5])
	if project == "" || instanceID == "" || parts[6] != "instancePartitions" {
		return "", "", "", false, false
	}
	if len(parts) == 7 {
		return project, instanceID, "", true, true
	}
	if len(parts) == 8 {
		partitionID = strings.TrimSpace(parts[7])
		if partitionID == "" || strings.Contains(partitionID, ":") {
			return "", "", "", false, false
		}
		return project, instanceID, partitionID, false, true
	}
	return "", "", "", false, false
}

func parseGCPSpannerAdminInstanceIAMActionPath(path string) (resource, action string, ok bool) {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	if !strings.HasPrefix(trimmed, "gcp/v1/") {
		return "", "", false
	}
	rest := strings.TrimPrefix(trimmed, "gcp/v1/")
	resource, action, ok = strings.Cut(rest, ":")
	if !ok {
		return "", "", false
	}
	resource = strings.TrimSpace(resource)
	action = strings.TrimSpace(action)
	if resource == "" {
		return "", "", false
	}
	switch strings.ToLower(action) {
	case "setiampolicy":
		action = "setIamPolicy"
	case "getiampolicy":
		action = "getIamPolicy"
	case "testiampermissions":
		action = "testIamPermissions"
	default:
		return "", "", false
	}
	if _, _, ok := parseGCPSpannerInstanceName(resource); ok {
		return resource, action, true
	}
	return "", "", false
}

func parseGCPSpannerAdminInstanceConfigOperationsCollectionPath(path string) (project, configID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 5 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "projects" && strings.TrimSpace(parts[3]) != "" && parts[4] == "instanceConfigOperations" {
		return strings.TrimSpace(parts[3]), "", true
	}
	if len(parts) == 7 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "projects" && strings.TrimSpace(parts[3]) != "" && parts[4] == "instanceConfigs" && strings.TrimSpace(parts[5]) != "" && parts[6] == "operations" {
		return strings.TrimSpace(parts[3]), strings.TrimSpace(parts[5]), true
	}
	return "", "", false
}

func parseGCPSpannerAdminInstancePartitionOperationsCollectionPath(path string) (project, instanceID, partitionID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 7 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "projects" && strings.TrimSpace(parts[3]) != "" && parts[4] == "instances" && strings.TrimSpace(parts[5]) != "" && parts[6] == "instancePartitionOperations" {
		return strings.TrimSpace(parts[3]), strings.TrimSpace(parts[5]), "", true
	}
	if len(parts) == 9 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "projects" && strings.TrimSpace(parts[3]) != "" && parts[4] == "instances" && strings.TrimSpace(parts[5]) != "" && parts[6] == "instancePartitions" && strings.TrimSpace(parts[7]) != "" && parts[8] == "operations" {
		return strings.TrimSpace(parts[3]), strings.TrimSpace(parts[5]), strings.TrimSpace(parts[7]), true
	}
	return "", "", "", false
}

func parseGCPSpannerAdminInstanceOperationsCollectionPath(path string) (project, scope string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 5 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "projects" && strings.TrimSpace(parts[3]) != "" && parts[4] == "operations" {
		return strings.TrimSpace(parts[3]), "project", true
	}
	if len(parts) == 7 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "projects" && strings.TrimSpace(parts[3]) != "" && parts[4] == "instances" && strings.TrimSpace(parts[5]) != "" && parts[6] == "operations" {
		return strings.TrimSpace(parts[3]), "instance-" + strings.TrimSpace(parts[5]), true
	}
	return "", "", false
}

func parseGCPSpannerAdminInstanceOperationPath(path string) (project, instanceID, operationID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 {
		return "", "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "instances" || parts[6] != "operations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	instanceID = strings.TrimSpace(parts[5])
	operationID = strings.TrimSpace(parts[7])
	if project == "" || instanceID == "" || operationID == "" || strings.Contains(operationID, ":") {
		return "", "", "", false
	}
	return project, instanceID, operationID, true
}

func parseGCPSpannerAdminInstanceOperationActionPath(path, action string) (project, instanceID, operationID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 8 {
		return "", "", "", false
	}
	if parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "instances" || parts[6] != "operations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	instanceID = strings.TrimSpace(parts[5])
	operationID = strings.TrimSpace(parts[7])
	id, parsedAction, found := strings.Cut(operationID, ":")
	if !found || strings.TrimSpace(parsedAction) != action {
		return "", "", "", false
	}
	id = strings.TrimSpace(id)
	if project == "" || instanceID == "" || id == "" {
		return "", "", "", false
	}
	return project, instanceID, id, true
}

func parseGCPSpannerAdminInstanceProjectName(name string) (project string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 2 || parts[0] != "projects" {
		return "", false
	}
	project = strings.TrimSpace(parts[1])
	return project, project != ""
}

func parseGCPSpannerAdminInstanceConfigName(name string) (project, configID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "instanceConfigs" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	configID = strings.TrimSpace(parts[3])
	if project == "" || configID == "" || strings.Contains(configID, ":") {
		return "", "", false
	}
	return project, configID, true
}

func parseGCPSpannerInstanceName(name string) (project, instanceID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "instances" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	instanceID = strings.TrimSpace(parts[3])
	if project == "" || instanceID == "" || strings.Contains(instanceID, ":") {
		return "", "", false
	}
	return project, instanceID, true
}

func parseGCPSpannerAdminInstancePartitionName(name string) (project, instanceID, partitionID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "instances" || parts[4] != "instancePartitions" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	instanceID = strings.TrimSpace(parts[3])
	partitionID = strings.TrimSpace(parts[5])
	if project == "" || instanceID == "" || partitionID == "" || strings.Contains(partitionID, ":") {
		return "", "", "", false
	}
	return project, instanceID, partitionID, true
}

func parseGCPSpannerAdminInstanceOperationName(name string) (project, instanceID, operationID string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "instances" || parts[4] != "operations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	instanceID = strings.TrimSpace(parts[3])
	operationID = strings.TrimSpace(parts[5])
	if project == "" || instanceID == "" || operationID == "" || strings.Contains(operationID, ":") {
		return "", "", "", false
	}
	return project, instanceID, operationID, true
}

func parseGCPSpannerAdminInstanceOperationsCollectionName(name string) (project string, ok bool) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(name), "/"), "/")
	if len(parts) != 3 || parts[0] != "projects" || parts[2] != "operations" {
		return "", false
	}
	project = strings.TrimSpace(parts[1])
	return project, project != ""
}

func parseGCPSpannerAdminInstancePagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize = 0
	start = 0

	if raw := strings.TrimSpace(r.URL.Query().Get("pageSize")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 || parsed > gcpSpannerAdminInstanceMaxPageSize {
			respondGCPSpannerAdminInstanceInvalidArgument(w, path, "pageSize must be a non-negative integer up to 1000")
			return 0, 0, false
		}
		pageSize = parsed
	}

	if raw := strings.TrimSpace(r.URL.Query().Get("pageToken")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			respondGCPSpannerAdminInstanceInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = parsed
	}

	return pageSize, start, true
}

func respondGCPSpannerAdminInstanceList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "pageToken is out of range")
		return false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	resp := map[string]any{
		field: items[start:end],
	}
	if end < len(items) {
		resp["nextPageToken"] = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, resp)
	return true
}

func decodeGCPSpannerAdminInstanceJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "failed to read request body")
		return nil, false
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func gcpSpannerAdminInstanceBodyMap(body map[string]any, key string) map[string]any {
	val, ok := body[key]
	if !ok {
		return nil
	}
	m, _ := val.(map[string]any)
	return m
}

func gcpSpannerAdminInstanceString(body map[string]any, key string) string {
	val, ok := body[key]
	if !ok {
		return ""
	}
	s, _ := val.(string)
	return s
}

func gcpSpannerAdminInstanceStringSlice(v any) []string {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func gcpSpannerAdminInstanceInt(v any) (int, bool) {
	switch typed := v.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func gcpSpannerAdminInstanceConfigFixture(project, configID string) map[string]any {
	if strings.TrimSpace(configID) == "" {
		configID = "custom-stackyard-primary"
	}
	return map[string]any{
		"name":                          fmt.Sprintf("projects/%s/instanceConfigs/%s", project, configID),
		"displayName":                   "Stackyard User Managed Config",
		"configType":                    "USER_MANAGED",
		"baseConfig":                    fmt.Sprintf("projects/%s/instanceConfigs/regional-us-central1", project),
		"etag":                          "etag-instance-config-" + configID,
		"leaderOptions":                 []string{"default", "regional-us-central1"},
		"reconciling":                   false,
		"state":                         "READY",
		"quorumType":                    "REGION",
		"storageLimitPerProcessingUnit": "4096",
	}
}

func gcpSpannerAdminInstanceFixture(project, instanceID string) map[string]any {
	if strings.TrimSpace(instanceID) == "" {
		instanceID = "stackyard-instance"
	}
	return map[string]any{
		"name":                      fmt.Sprintf("projects/%s/instances/%s", project, instanceID),
		"config":                    fmt.Sprintf("projects/%s/instanceConfigs/custom-stackyard-primary", project),
		"displayName":               "Stackyard Instance",
		"nodeCount":                 1,
		"state":                     "READY",
		"endpointUris":              []string{"spanner.googleapis.com"},
		"createTime":                gcpSpannerAdminInstanceReferenceTime.Add(2 * time.Minute).Format(time.RFC3339),
		"updateTime":                gcpSpannerAdminInstanceReferenceTime.Add(4 * time.Minute).Format(time.RFC3339),
		"edition":                   "ENTERPRISE",
		"defaultBackupScheduleType": "AUTOMATIC",
	}
}

func gcpSpannerAdminInstancePartitionFixture(project, instanceID, partitionID string) map[string]any {
	if strings.TrimSpace(partitionID) == "" {
		partitionID = "partition-a"
	}
	return map[string]any{
		"name":            fmt.Sprintf("projects/%s/instances/%s/instancePartitions/%s", project, instanceID, partitionID),
		"config":          fmt.Sprintf("projects/%s/instanceConfigs/custom-stackyard-primary", project),
		"displayName":     "Stackyard Partition",
		"processingUnits": 1000,
		"state":           "READY",
		"createTime":      gcpSpannerAdminInstanceReferenceTime.Add(3 * time.Minute).Format(time.RFC3339),
		"updateTime":      gcpSpannerAdminInstanceReferenceTime.Add(5 * time.Minute).Format(time.RFC3339),
		"etag":            "etag-instance-partition-" + partitionID,
	}
}

func gcpSpannerAdminInstancePolicyFixture(resource string) map[string]any {
	return map[string]any{
		"version": 1,
		"etag":    "etag-spanner-admin-instance-policy",
		"bindings": []map[string]any{
			{
				"role":    "roles/spanner.admin",
				"members": []string{"user:stackyard@example.com"},
			},
		},
		"resource": resource,
	}
}

func gcpSpannerAdminInstanceOperationFixture(project, operationID string, response any) map[string]any {
	if strings.TrimSpace(operationID) == "" {
		operationID = "operation-1"
	}
	return map[string]any{
		"name": fmt.Sprintf("projects/%s/instances/stackyard-instance/operations/%s", project, operationID),
		"done": true,
		"metadata": map[string]any{
			"@type":           "type.googleapis.com/google.spanner.admin.instance.v1.OperationProgress",
			"startTime":       gcpSpannerAdminInstanceReferenceTime.Format(time.RFC3339),
			"progressPercent": 100,
		},
		"response": response,
	}
}

func isGCPSpannerAdminInstanceIdentifier(id string, maxLen int) bool {
	id = strings.TrimSpace(id)
	if len(id) < 2 || len(id) > maxLen {
		return false
	}
	for idx, ch := range id {
		if idx == 0 {
			if ch < 'a' || ch > 'z' {
				return false
			}
			continue
		}
		if idx == len(id)-1 {
			if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')) {
				return false
			}
			continue
		}
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-') {
			return false
		}
	}
	return true
}

func isGCPSpannerAdminInstanceMissingResource(parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(strings.ToLower(strings.TrimSpace(part)), "missing") {
			return true
		}
	}
	return false
}

func isGCPSpannerAdminInstanceAlreadyExists(parts ...string) bool {
	for _, part := range parts {
		if strings.Contains(strings.ToLower(strings.TrimSpace(part)), "existing") {
			return true
		}
	}
	return false
}

func respondGCPSpannerAdminInstanceInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPSpannerAdminInstanceError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPSpannerAdminInstanceNotFound(w http.ResponseWriter, path, message string) {
	respondGCPSpannerAdminInstanceError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPSpannerAdminInstanceFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPSpannerAdminInstanceError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPSpannerAdminInstanceAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPSpannerAdminInstanceError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPSpannerAdminInstanceError(w http.ResponseWriter, status int, err, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    err,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_spanner_admin_instance(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "spanner_admin_instance") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPSpannerAdminInstanceInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/instances/stackyard-instance",
			"service":  "spanner_admin_instance",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
