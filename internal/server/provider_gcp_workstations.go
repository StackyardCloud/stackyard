package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const gcpWorkstationsGRPCPathPrefix = "/gcp/google.cloud.workstations.v1.Workstations/"

var (
	gcpWorkstationsReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpWorkstationsIDPattern     = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
)

func (s *Server) handleGCPWorkstationsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_workstations(w, r) {
		return true
	}

	path := normalizeGCPWorkstationsPath(rawRequestPath(r))
	if strings.HasPrefix(path, gcpWorkstationsGRPCPathPrefix) {
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
	if !isGCPWorkstationsPath(path, hasGCPWorkstationsHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPWorkstationsListWorkstationClusters(w, r, path) {
			return true
		}
		if handleGCPWorkstationsGetWorkstationCluster(w, path) {
			return true
		}
		if handleGCPWorkstationsListWorkstationConfigs(w, r, path) {
			return true
		}
		if handleGCPWorkstationsListUsableWorkstationConfigs(w, r, path) {
			return true
		}
		if handleGCPWorkstationsGetWorkstationConfig(w, path) {
			return true
		}
		if handleGCPWorkstationsListWorkstations(w, r, path) {
			return true
		}
		if handleGCPWorkstationsListUsableWorkstations(w, r, path) {
			return true
		}
		if handleGCPWorkstationsGetWorkstation(w, path) {
			return true
		}
		if handleGCPWorkstationsListOperations(w, r, path) {
			return true
		}
		if handleGCPWorkstationsGetOperation(w, path) {
			return true
		}
		if handleGCPWorkstationsGetIAMPolicy(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPWorkstationsCreateWorkstationCluster(w, r, path) {
			return true
		}
		if handleGCPWorkstationsCreateWorkstationConfig(w, r, path) {
			return true
		}
		if handleGCPWorkstationsCreateWorkstation(w, r, path) {
			return true
		}
		if handleGCPWorkstationsStartWorkstation(w, r, path) {
			return true
		}
		if handleGCPWorkstationsStopWorkstation(w, r, path) {
			return true
		}
		if handleGCPWorkstationsGenerateAccessToken(w, r, path) {
			return true
		}
		if handleGCPWorkstationsSetIAMPolicy(w, r, path) {
			return true
		}
		if handleGCPWorkstationsTestIAMPermissions(w, r, path) {
			return true
		}
		if handleGCPWorkstationsCancelOperation(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPWorkstationsUpdateWorkstationCluster(w, r, path) {
			return true
		}
		if handleGCPWorkstationsUpdateWorkstationConfig(w, r, path) {
			return true
		}
		if handleGCPWorkstationsUpdateWorkstation(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPWorkstationsDeleteWorkstationCluster(w, r, path) {
			return true
		}
		if handleGCPWorkstationsDeleteWorkstationConfig(w, r, path) {
			return true
		}
		if handleGCPWorkstationsDeleteWorkstation(w, path) {
			return true
		}
		if handleGCPWorkstationsDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPWorkstationsPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPWorkstationsHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "workstations",
		"workstations-apiv1",
		"workstations_apiv1",
		"workstation",
		"cloud-workstations",
		"cloud_workstations",
		"gcp-workstations",
		"gcp-workstations-apiv1":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-workstations-apiv1") || strings.Contains(ua, "cloud.google.com/go/workstations")
}

func isGCPWorkstationsPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, gcpWorkstationsGRPCPathPrefix) {
		return true
	}

	_, _, tail, ok := parseGCPWorkstationsLocationTail(path)
	if !ok {
		return false
	}
	if len(tail) == 0 {
		return includeHint
	}
	if !includeHint {
		return false
	}

	return isGCPWorkstationsClustersCollectionTail(tail) ||
		isGCPWorkstationsClusterTail(tail) ||
		isGCPWorkstationsClusterActionTail(tail, "getIamPolicy") ||
		isGCPWorkstationsClusterActionTail(tail, "setIamPolicy") ||
		isGCPWorkstationsClusterActionTail(tail, "testIamPermissions") ||
		isGCPWorkstationsConfigsCollectionTail(tail) ||
		isGCPWorkstationsConfigTail(tail) ||
		isGCPWorkstationsConfigsActionCollectionTail(tail, "listUsable") ||
		isGCPWorkstationsConfigActionTail(tail, "getIamPolicy") ||
		isGCPWorkstationsConfigActionTail(tail, "setIamPolicy") ||
		isGCPWorkstationsConfigActionTail(tail, "testIamPermissions") ||
		isGCPWorkstationsWorkstationsCollectionTail(tail) ||
		isGCPWorkstationsWorkstationTail(tail) ||
		isGCPWorkstationsWorkstationsActionCollectionTail(tail, "listUsable") ||
		isGCPWorkstationsWorkstationActionTail(tail, "start") ||
		isGCPWorkstationsWorkstationActionTail(tail, "stop") ||
		isGCPWorkstationsWorkstationActionTail(tail, "generateAccessToken") ||
		isGCPWorkstationsWorkstationActionTail(tail, "getIamPolicy") ||
		isGCPWorkstationsWorkstationActionTail(tail, "setIamPolicy") ||
		isGCPWorkstationsWorkstationActionTail(tail, "testIamPermissions") ||
		isGCPWorkstationsOperationsCollectionTail(tail) ||
		isGCPWorkstationsOperationTail(tail) ||
		isGCPWorkstationsOperationActionTail(tail, "cancel")
}

func handleGCPWorkstationsListWorkstationClusters(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPWorkstationsClustersCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPWorkstationsPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpWorkstationsWorkstationClusterFixture(project, location, "cluster-1"),
		gcpWorkstationsWorkstationClusterFixture(project, location, "cluster-2"),
	}
	return respondGCPWorkstationsList(w, "workstationClusters", items, pageSize, start, path)
}

func handleGCPWorkstationsGetWorkstationCluster(w http.ResponseWriter, path string) bool {
	project, location, clusterID, ok := parseGCPWorkstationsClusterPath(path)
	if !ok {
		return false
	}
	if strings.Contains(strings.ToLower(clusterID), "missing") {
		respondGCPWorkstationsNotFound(w, path, "workstation cluster not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpWorkstationsWorkstationClusterFixture(project, location, clusterID))
	return true
}

func handleGCPWorkstationsCreateWorkstationCluster(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPWorkstationsClustersCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPWorkstationsJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	cluster := gcpWorkstationsBodyMap(body, "workstationCluster")
	if len(cluster) == 0 {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationCluster is required")
		return true
	}
	clusterID := strings.TrimSpace(r.URL.Query().Get("workstationClusterId"))
	if clusterID == "" {
		if name := gcpWorkstationsString(cluster, "name"); name != "" {
			_, _, parsedID, parsed := parseGCPWorkstationsClusterName(name)
			if parsed {
				clusterID = parsedID
			}
		}
	}
	if clusterID == "" {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationClusterId is required")
		return true
	}
	if !gcpWorkstationsIDPattern.MatchString(clusterID) {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationClusterId is invalid")
		return true
	}
	expectedName := gcpWorkstationsClusterName(project, location, clusterID)
	if name := gcpWorkstationsString(cluster, "name"); name != "" && name != expectedName {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationCluster.name must match parent and workstationClusterId")
		return true
	}
	if strings.TrimSpace(gcpWorkstationsString(cluster, "network")) == "" {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationCluster.network is required")
		return true
	}
	if strings.Contains(strings.ToLower(clusterID), "existing") {
		respondGCPWorkstationsAlreadyExists(w, path, "workstation cluster already exists")
		return true
	}
	operationID := "createWorkstationCluster." + clusterID
	respondJSON(w, http.StatusOK, gcpWorkstationsOperationFixture(project, location, operationID, expectedName, "create", false))
	return true
}

func handleGCPWorkstationsUpdateWorkstationCluster(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, ok := parseGCPWorkstationsClusterPath(path)
	if !ok {
		return false
	}
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		respondGCPWorkstationsInvalidArgument(w, path, "updateMask is required")
		return true
	}
	maskPaths, ok := parseGCPWorkstationsUpdateMask(updateMask)
	if !ok || len(maskPaths) == 0 {
		respondGCPWorkstationsInvalidArgument(w, path, "updateMask contains unsupported paths")
		return true
	}
	body, valid := decodeGCPWorkstationsJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	cluster := gcpWorkstationsBodyMap(body, "workstationCluster")
	if len(cluster) == 0 {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationCluster is required")
		return true
	}
	expectedName := gcpWorkstationsClusterName(project, location, clusterID)
	if name := gcpWorkstationsString(cluster, "name"); name == "" || name != expectedName {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationCluster.name must match requested resource")
		return true
	}
	operationID := "updateWorkstationCluster." + clusterID
	respondJSON(w, http.StatusOK, gcpWorkstationsOperationFixture(project, location, operationID, expectedName, "update", false))
	return true
}

func handleGCPWorkstationsDeleteWorkstationCluster(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, ok := parseGCPWorkstationsClusterPath(path)
	if !ok {
		return false
	}
	force, err := parseOptionalBool(r.URL.Query().Get("force"), false)
	if err != nil {
		respondGCPWorkstationsInvalidArgument(w, path, "force must be a boolean")
		return true
	}
	if !force && strings.Contains(strings.ToLower(clusterID), "inuse") {
		respondGCPWorkstationsFailedPrecondition(w, path, "cluster contains workstation configurations; set force=true")
		return true
	}
	name := gcpWorkstationsClusterName(project, location, clusterID)
	operationID := "deleteWorkstationCluster." + clusterID
	respondJSON(w, http.StatusOK, gcpWorkstationsOperationFixture(project, location, operationID, name, "delete", false))
	return true
}

func handleGCPWorkstationsListWorkstationConfigs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, ok := parseGCPWorkstationsConfigsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPWorkstationsPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpWorkstationsWorkstationConfigFixture(project, location, clusterID, "config-1"),
		gcpWorkstationsWorkstationConfigFixture(project, location, clusterID, "config-2"),
	}
	return respondGCPWorkstationsList(w, "workstationConfigs", items, pageSize, start, path)
}

func handleGCPWorkstationsListUsableWorkstationConfigs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, ok := parseGCPWorkstationsConfigsActionCollectionPath(path, "listUsable")
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPWorkstationsPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpWorkstationsWorkstationConfigFixture(project, location, clusterID, "config-1"),
	}
	return respondGCPWorkstationsList(w, "workstationConfigs", items, pageSize, start, path)
}

func handleGCPWorkstationsGetWorkstationConfig(w http.ResponseWriter, path string) bool {
	project, location, clusterID, configID, ok := parseGCPWorkstationsConfigPath(path)
	if !ok {
		return false
	}
	if strings.Contains(strings.ToLower(configID), "missing") {
		respondGCPWorkstationsNotFound(w, path, "workstation config not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpWorkstationsWorkstationConfigFixture(project, location, clusterID, configID))
	return true
}

func handleGCPWorkstationsCreateWorkstationConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, ok := parseGCPWorkstationsConfigsCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPWorkstationsJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	config := gcpWorkstationsBodyMap(body, "workstationConfig")
	if len(config) == 0 {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationConfig is required")
		return true
	}
	configID := strings.TrimSpace(r.URL.Query().Get("workstationConfigId"))
	if configID == "" {
		if name := gcpWorkstationsString(config, "name"); name != "" {
			_, _, _, parsedID, parsed := parseGCPWorkstationsConfigName(name)
			if parsed {
				configID = parsedID
			}
		}
	}
	if configID == "" {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationConfigId is required")
		return true
	}
	if !gcpWorkstationsIDPattern.MatchString(configID) {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationConfigId is invalid")
		return true
	}
	expectedName := gcpWorkstationsConfigName(project, location, clusterID, configID)
	if name := gcpWorkstationsString(config, "name"); name != "" && name != expectedName {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationConfig.name must match parent and workstationConfigId")
		return true
	}
	if _, ok := config["host"]; !ok {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationConfig.host is required")
		return true
	}
	if strings.Contains(strings.ToLower(configID), "existing") {
		respondGCPWorkstationsAlreadyExists(w, path, "workstation config already exists")
		return true
	}
	operationID := "createWorkstationConfig." + configID
	respondJSON(w, http.StatusOK, gcpWorkstationsOperationFixture(project, location, operationID, expectedName, "create", false))
	return true
}

func handleGCPWorkstationsUpdateWorkstationConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, configID, ok := parseGCPWorkstationsConfigPath(path)
	if !ok {
		return false
	}
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		respondGCPWorkstationsInvalidArgument(w, path, "updateMask is required")
		return true
	}
	maskPaths, ok := parseGCPWorkstationsUpdateMask(updateMask)
	if !ok || len(maskPaths) == 0 {
		respondGCPWorkstationsInvalidArgument(w, path, "updateMask contains unsupported paths")
		return true
	}
	body, valid := decodeGCPWorkstationsJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	config := gcpWorkstationsBodyMap(body, "workstationConfig")
	if len(config) == 0 {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationConfig is required")
		return true
	}
	expectedName := gcpWorkstationsConfigName(project, location, clusterID, configID)
	if name := gcpWorkstationsString(config, "name"); name == "" || name != expectedName {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationConfig.name must match requested resource")
		return true
	}
	operationID := "updateWorkstationConfig." + configID
	respondJSON(w, http.StatusOK, gcpWorkstationsOperationFixture(project, location, operationID, expectedName, "update", false))
	return true
}

func handleGCPWorkstationsDeleteWorkstationConfig(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, configID, ok := parseGCPWorkstationsConfigPath(path)
	if !ok {
		return false
	}
	force, err := parseOptionalBool(r.URL.Query().Get("force"), false)
	if err != nil {
		respondGCPWorkstationsInvalidArgument(w, path, "force must be a boolean")
		return true
	}
	if !force && strings.Contains(strings.ToLower(configID), "inuse") {
		respondGCPWorkstationsFailedPrecondition(w, path, "configuration contains workstations; set force=true")
		return true
	}
	name := gcpWorkstationsConfigName(project, location, clusterID, configID)
	operationID := "deleteWorkstationConfig." + configID
	respondJSON(w, http.StatusOK, gcpWorkstationsOperationFixture(project, location, operationID, name, "delete", false))
	return true
}

func handleGCPWorkstationsListWorkstations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, configID, ok := parseGCPWorkstationsWorkstationsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPWorkstationsPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpWorkstationsWorkstationFixture(project, location, clusterID, configID, "workstation-running"),
		gcpWorkstationsWorkstationFixture(project, location, clusterID, configID, "workstation-stopped"),
	}
	return respondGCPWorkstationsList(w, "workstations", items, pageSize, start, path)
}

func handleGCPWorkstationsListUsableWorkstations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, configID, ok := parseGCPWorkstationsWorkstationsActionCollectionPath(path, "listUsable")
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPWorkstationsPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpWorkstationsWorkstationFixture(project, location, clusterID, configID, "workstation-running"),
	}
	return respondGCPWorkstationsList(w, "workstations", items, pageSize, start, path)
}

func handleGCPWorkstationsGetWorkstation(w http.ResponseWriter, path string) bool {
	project, location, clusterID, configID, workstationID, ok := parseGCPWorkstationsWorkstationPath(path)
	if !ok {
		return false
	}
	if strings.Contains(strings.ToLower(workstationID), "missing") {
		respondGCPWorkstationsNotFound(w, path, "workstation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpWorkstationsWorkstationFixture(project, location, clusterID, configID, workstationID))
	return true
}

func handleGCPWorkstationsCreateWorkstation(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, configID, ok := parseGCPWorkstationsWorkstationsCollectionPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPWorkstationsJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	workstation := gcpWorkstationsBodyMap(body, "workstation")
	if len(workstation) == 0 {
		respondGCPWorkstationsInvalidArgument(w, path, "workstation is required")
		return true
	}
	workstationID := strings.TrimSpace(r.URL.Query().Get("workstationId"))
	if workstationID == "" {
		if name := gcpWorkstationsString(workstation, "name"); name != "" {
			_, _, _, _, parsedID, parsed := parseGCPWorkstationsWorkstationName(name)
			if parsed {
				workstationID = parsedID
			}
		}
	}
	if workstationID == "" {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationId is required")
		return true
	}
	if !gcpWorkstationsIDPattern.MatchString(workstationID) {
		respondGCPWorkstationsInvalidArgument(w, path, "workstationId is invalid")
		return true
	}
	expectedName := gcpWorkstationsWorkstationName(project, location, clusterID, configID, workstationID)
	if name := gcpWorkstationsString(workstation, "name"); name != "" && name != expectedName {
		respondGCPWorkstationsInvalidArgument(w, path, "workstation.name must match parent and workstationId")
		return true
	}
	if strings.Contains(strings.ToLower(workstationID), "existing") {
		respondGCPWorkstationsAlreadyExists(w, path, "workstation already exists")
		return true
	}
	operationID := "createWorkstation." + workstationID
	respondJSON(w, http.StatusOK, gcpWorkstationsOperationFixture(project, location, operationID, expectedName, "create", false))
	return true
}

func handleGCPWorkstationsUpdateWorkstation(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, configID, workstationID, ok := parseGCPWorkstationsWorkstationPath(path)
	if !ok {
		return false
	}
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		respondGCPWorkstationsInvalidArgument(w, path, "updateMask is required")
		return true
	}
	maskPaths, ok := parseGCPWorkstationsUpdateMask(updateMask)
	if !ok || len(maskPaths) == 0 {
		respondGCPWorkstationsInvalidArgument(w, path, "updateMask contains unsupported paths")
		return true
	}
	body, valid := decodeGCPWorkstationsJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	workstation := gcpWorkstationsBodyMap(body, "workstation")
	if len(workstation) == 0 {
		respondGCPWorkstationsInvalidArgument(w, path, "workstation is required")
		return true
	}
	expectedName := gcpWorkstationsWorkstationName(project, location, clusterID, configID, workstationID)
	if name := gcpWorkstationsString(workstation, "name"); name == "" || name != expectedName {
		respondGCPWorkstationsInvalidArgument(w, path, "workstation.name must match requested resource")
		return true
	}
	operationID := "updateWorkstation." + workstationID
	respondJSON(w, http.StatusOK, gcpWorkstationsOperationFixture(project, location, operationID, expectedName, "update", false))
	return true
}

func handleGCPWorkstationsDeleteWorkstation(w http.ResponseWriter, path string) bool {
	project, location, clusterID, configID, workstationID, ok := parseGCPWorkstationsWorkstationPath(path)
	if !ok {
		return false
	}
	name := gcpWorkstationsWorkstationName(project, location, clusterID, configID, workstationID)
	operationID := "deleteWorkstation." + workstationID
	respondJSON(w, http.StatusOK, gcpWorkstationsOperationFixture(project, location, operationID, name, "delete", false))
	return true
}

func handleGCPWorkstationsStartWorkstation(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, configID, workstationID, ok := parseGCPWorkstationsWorkstationActionPath(path, "start")
	if !ok {
		return false
	}
	body, valid := decodeGCPWorkstationsJSONBody(w, r, path, false)
	if !valid {
		return true
	}
	expectedName := gcpWorkstationsWorkstationName(project, location, clusterID, configID, workstationID)
	if name := strings.TrimSpace(gcpWorkstationsString(body, "name")); name != "" && name != expectedName {
		respondGCPWorkstationsInvalidArgument(w, path, "name must match requested workstation")
		return true
	}
	switch gcpWorkstationsStateForID(workstationID) {
	case "STATE_RUNNING", "STATE_STARTING":
		respondGCPWorkstationsFailedPrecondition(w, path, "workstation must be stopped before start")
		return true
	}
	operationID := "startWorkstation." + workstationID
	respondJSON(w, http.StatusOK, gcpWorkstationsOperationFixture(project, location, operationID, expectedName, "start", false))
	return true
}

func handleGCPWorkstationsStopWorkstation(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, configID, workstationID, ok := parseGCPWorkstationsWorkstationActionPath(path, "stop")
	if !ok {
		return false
	}
	body, valid := decodeGCPWorkstationsJSONBody(w, r, path, false)
	if !valid {
		return true
	}
	expectedName := gcpWorkstationsWorkstationName(project, location, clusterID, configID, workstationID)
	if name := strings.TrimSpace(gcpWorkstationsString(body, "name")); name != "" && name != expectedName {
		respondGCPWorkstationsInvalidArgument(w, path, "name must match requested workstation")
		return true
	}
	switch gcpWorkstationsStateForID(workstationID) {
	case "STATE_STOPPED", "STATE_STOPPING":
		respondGCPWorkstationsFailedPrecondition(w, path, "workstation must be running before stop")
		return true
	}
	operationID := "stopWorkstation." + workstationID
	respondJSON(w, http.StatusOK, gcpWorkstationsOperationFixture(project, location, operationID, expectedName, "stop", false))
	return true
}

func handleGCPWorkstationsGenerateAccessToken(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, clusterID, configID, workstationID, ok := parseGCPWorkstationsWorkstationActionPath(path, "generateAccessToken")
	if !ok {
		return false
	}
	body, valid := decodeGCPWorkstationsJSONBody(w, r, path, false)
	if !valid {
		return true
	}
	expectedName := gcpWorkstationsWorkstationName(project, location, clusterID, configID, workstationID)
	if workstation := strings.TrimSpace(gcpWorkstationsString(body, "workstation")); workstation != "" && workstation != expectedName {
		respondGCPWorkstationsInvalidArgument(w, path, "workstation must match requested workstation")
		return true
	}
	state := gcpWorkstationsStateForID(workstationID)
	if state != "STATE_RUNNING" {
		respondGCPWorkstationsFailedPrecondition(w, path, "workstation must be running to generate access token")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"accessToken": "ya29.stackyard-workstations-" + workstationID,
		"expireTime":  gcpWorkstationsReferenceTime.Add(30 * time.Minute).Format(time.RFC3339),
	})
	return true
}

func handleGCPWorkstationsListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, ok := parseGCPWorkstationsOperationsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPWorkstationsPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpWorkstationsOperationFixture(project, location, "operations/op-1", gcpWorkstationsClusterName(project, location, "cluster-1"), "create", true),
		gcpWorkstationsOperationFixture(project, location, "operations/op-2", gcpWorkstationsWorkstationName(project, location, "cluster-1", "config-1", "workstation-running"), "start", false),
	}
	return respondGCPWorkstationsList(w, "operations", items, pageSize, start, path)
}

func handleGCPWorkstationsGetOperation(w http.ResponseWriter, path string) bool {
	project, location, operationID, ok := parseGCPWorkstationsOperationPath(path)
	if !ok {
		return false
	}
	done := strings.Contains(strings.ToLower(operationID), "done")
	respondJSON(w, http.StatusOK, gcpWorkstationsOperationFixture(project, location, "operations/"+operationID, gcpWorkstationsClusterName(project, location, "cluster-1"), "get", done))
	return true
}

func handleGCPWorkstationsCancelOperation(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, _, ok := parseGCPWorkstationsOperationActionPath(path, "cancel")
	if !ok {
		return false
	}
	if _, valid := decodeGCPWorkstationsJSONBody(w, r, path, false); !valid {
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPWorkstationsDeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, _, ok := parseGCPWorkstationsOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPWorkstationsGetIAMPolicy(w http.ResponseWriter, path string) bool {
	resource, ok := parseGCPWorkstationsIAMResourcePath(path, "getIamPolicy")
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpWorkstationsPolicyFixture(resource, nil))
	return true
}

func handleGCPWorkstationsSetIAMPolicy(w http.ResponseWriter, r *http.Request, path string) bool {
	resource, ok := parseGCPWorkstationsIAMResourcePath(path, "setIamPolicy")
	if !ok {
		return false
	}
	body, valid := decodeGCPWorkstationsJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	policy, _ := body["policy"].(map[string]any)
	if len(policy) == 0 {
		respondGCPWorkstationsInvalidArgument(w, path, "policy is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpWorkstationsPolicyFixture(resource, policy))
	return true
}

func handleGCPWorkstationsTestIAMPermissions(w http.ResponseWriter, r *http.Request, path string) bool {
	_, ok := parseGCPWorkstationsIAMResourcePath(path, "testIamPermissions")
	if !ok {
		return false
	}
	body, valid := decodeGCPWorkstationsJSONBody(w, r, path, true)
	if !valid {
		return true
	}
	permissions := gcpWorkstationsStringSlice(body["permissions"])
	if len(permissions) == 0 {
		respondGCPWorkstationsInvalidArgument(w, path, "permissions are required")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"permissions": permissions,
	})
	return true
}

func parseGCPWorkstationsLocationTail(path string) (project, location string, tail []string, ok bool) {
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

func isGCPWorkstationsClustersCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "workstationClusters"
}

func isGCPWorkstationsClusterTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "workstationClusters" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPWorkstationsClusterActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "workstationClusters" {
		return false
	}
	id, parsedAction, ok := splitGCPWorkstationsActionSegment(tail[1])
	return ok && strings.TrimSpace(id) != "" && parsedAction == action
}

func isGCPWorkstationsConfigsCollectionTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "workstationClusters" && strings.TrimSpace(tail[1]) != "" && tail[2] == "workstationConfigs"
}

func isGCPWorkstationsConfigTail(tail []string) bool {
	return len(tail) == 4 &&
		tail[0] == "workstationClusters" &&
		strings.TrimSpace(tail[1]) != "" &&
		tail[2] == "workstationConfigs" &&
		strings.TrimSpace(tail[3]) != "" &&
		!strings.Contains(tail[3], ":")
}

func isGCPWorkstationsConfigsActionCollectionTail(tail []string, action string) bool {
	if len(tail) != 3 || tail[0] != "workstationClusters" || strings.TrimSpace(tail[1]) == "" || tail[2] != "workstationConfigs:"+action {
		return false
	}
	return true
}

func isGCPWorkstationsConfigActionTail(tail []string, action string) bool {
	if len(tail) != 4 || tail[0] != "workstationClusters" || strings.TrimSpace(tail[1]) == "" || tail[2] != "workstationConfigs" {
		return false
	}
	id, parsedAction, ok := splitGCPWorkstationsActionSegment(tail[3])
	return ok && strings.TrimSpace(id) != "" && parsedAction == action
}

func isGCPWorkstationsWorkstationsCollectionTail(tail []string) bool {
	return len(tail) == 5 &&
		tail[0] == "workstationClusters" &&
		strings.TrimSpace(tail[1]) != "" &&
		tail[2] == "workstationConfigs" &&
		strings.TrimSpace(tail[3]) != "" &&
		tail[4] == "workstations"
}

func isGCPWorkstationsWorkstationTail(tail []string) bool {
	return len(tail) == 6 &&
		tail[0] == "workstationClusters" &&
		strings.TrimSpace(tail[1]) != "" &&
		tail[2] == "workstationConfigs" &&
		strings.TrimSpace(tail[3]) != "" &&
		tail[4] == "workstations" &&
		strings.TrimSpace(tail[5]) != "" &&
		!strings.Contains(tail[5], ":")
}

func isGCPWorkstationsWorkstationsActionCollectionTail(tail []string, action string) bool {
	return len(tail) == 5 &&
		tail[0] == "workstationClusters" &&
		strings.TrimSpace(tail[1]) != "" &&
		tail[2] == "workstationConfigs" &&
		strings.TrimSpace(tail[3]) != "" &&
		tail[4] == "workstations:"+action
}

func isGCPWorkstationsWorkstationActionTail(tail []string, action string) bool {
	if len(tail) != 6 ||
		tail[0] != "workstationClusters" ||
		strings.TrimSpace(tail[1]) == "" ||
		tail[2] != "workstationConfigs" ||
		strings.TrimSpace(tail[3]) == "" ||
		tail[4] != "workstations" {
		return false
	}
	id, parsedAction, ok := splitGCPWorkstationsActionSegment(tail[5])
	return ok && strings.TrimSpace(id) != "" && parsedAction == action
}

func isGCPWorkstationsOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPWorkstationsOperationTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPWorkstationsOperationActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	id, parsedAction, ok := splitGCPWorkstationsActionSegment(tail[1])
	return ok && strings.TrimSpace(id) != "" && parsedAction == action
}

func parseGCPWorkstationsClustersCollectionPath(path string) (project, location string, ok bool) {
	project, location, tail, ok := parseGCPWorkstationsLocationTail(path)
	if !ok || !isGCPWorkstationsClustersCollectionTail(tail) {
		return "", "", false
	}
	return project, location, true
}

func parseGCPWorkstationsClusterPath(path string) (project, location, clusterID string, ok bool) {
	project, location, tail, ok := parseGCPWorkstationsLocationTail(path)
	if !ok || !isGCPWorkstationsClusterTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPWorkstationsConfigsCollectionPath(path string) (project, location, clusterID string, ok bool) {
	project, location, tail, ok := parseGCPWorkstationsLocationTail(path)
	if !ok || !isGCPWorkstationsConfigsCollectionTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPWorkstationsConfigsActionCollectionPath(path, action string) (project, location, clusterID string, ok bool) {
	project, location, tail, ok := parseGCPWorkstationsLocationTail(path)
	if !ok || !isGCPWorkstationsConfigsActionCollectionTail(tail, action) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPWorkstationsConfigPath(path string) (project, location, clusterID, configID string, ok bool) {
	project, location, tail, ok := parseGCPWorkstationsLocationTail(path)
	if !ok || !isGCPWorkstationsConfigTail(tail) {
		return "", "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), true
}

func parseGCPWorkstationsWorkstationsCollectionPath(path string) (project, location, clusterID, configID string, ok bool) {
	project, location, tail, ok := parseGCPWorkstationsLocationTail(path)
	if !ok || !isGCPWorkstationsWorkstationsCollectionTail(tail) {
		return "", "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), true
}

func parseGCPWorkstationsWorkstationsActionCollectionPath(path, action string) (project, location, clusterID, configID string, ok bool) {
	project, location, tail, ok := parseGCPWorkstationsLocationTail(path)
	if !ok || !isGCPWorkstationsWorkstationsActionCollectionTail(tail, action) {
		return "", "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), true
}

func parseGCPWorkstationsWorkstationPath(path string) (project, location, clusterID, configID, workstationID string, ok bool) {
	project, location, tail, ok := parseGCPWorkstationsLocationTail(path)
	if !ok || !isGCPWorkstationsWorkstationTail(tail) {
		return "", "", "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), strings.TrimSpace(tail[5]), true
}

func parseGCPWorkstationsWorkstationActionPath(path, action string) (project, location, clusterID, configID, workstationID string, ok bool) {
	project, location, tail, ok := parseGCPWorkstationsLocationTail(path)
	if !ok || !isGCPWorkstationsWorkstationActionTail(tail, action) {
		return "", "", "", "", "", false
	}
	workstationID, parsedAction, parsed := splitGCPWorkstationsActionSegment(tail[5])
	if !parsed || parsedAction != action {
		return "", "", "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), strings.TrimSpace(workstationID), true
}

func parseGCPWorkstationsOperationsCollectionPath(path string) (project, location string, ok bool) {
	project, location, tail, ok := parseGCPWorkstationsLocationTail(path)
	if !ok || !isGCPWorkstationsOperationsCollectionTail(tail) {
		return "", "", false
	}
	return project, location, true
}

func parseGCPWorkstationsOperationPath(path string) (project, location, operationID string, ok bool) {
	project, location, tail, ok := parseGCPWorkstationsLocationTail(path)
	if !ok || !isGCPWorkstationsOperationTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPWorkstationsOperationActionPath(path, action string) (project, location, operationID string, ok bool) {
	project, location, tail, ok := parseGCPWorkstationsLocationTail(path)
	if !ok || !isGCPWorkstationsOperationActionTail(tail, action) {
		return "", "", "", false
	}
	operationID, parsedAction, parsed := splitGCPWorkstationsActionSegment(tail[1])
	if !parsed || parsedAction != action {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(operationID), true
}

func parseGCPWorkstationsIAMResourcePath(path, action string) (resource string, ok bool) {
	project, location, tail, ok := parseGCPWorkstationsLocationTail(path)
	if !ok {
		return "", false
	}
	switch {
	case isGCPWorkstationsClusterActionTail(tail, action):
		clusterID, _, parsed := splitGCPWorkstationsActionSegment(tail[1])
		if !parsed {
			return "", false
		}
		return gcpWorkstationsClusterName(project, location, strings.TrimSpace(clusterID)), true
	case isGCPWorkstationsConfigActionTail(tail, action):
		configID, _, parsed := splitGCPWorkstationsActionSegment(tail[3])
		if !parsed {
			return "", false
		}
		return gcpWorkstationsConfigName(project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(configID)), true
	case isGCPWorkstationsWorkstationActionTail(tail, action):
		workstationID, _, parsed := splitGCPWorkstationsActionSegment(tail[5])
		if !parsed {
			return "", false
		}
		return gcpWorkstationsWorkstationName(project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), strings.TrimSpace(workstationID)), true
	default:
		return "", false
	}
}

func splitGCPWorkstationsActionSegment(raw string) (id, action string, ok bool) {
	segment := strings.TrimSpace(raw)
	if segment == "" {
		return "", "", false
	}
	if decoded, err := url.PathUnescape(segment); err == nil {
		segment = decoded
	}
	id, action, ok = strings.Cut(segment, ":")
	if !ok {
		return "", "", false
	}
	id = strings.TrimSpace(id)
	action = strings.TrimSpace(action)
	if id == "" || action == "" {
		return "", "", false
	}
	return id, action, true
}

func parseGCPWorkstationsPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPWorkstationsInvalidArgument(w, path, fmt.Sprintf("pageSize must be a non-negative integer <= %d", maxPageSize))
		return 0, 0, false
	}
	if pageSize > maxPageSize {
		respondGCPWorkstationsOutOfRange(w, path, fmt.Sprintf("pageSize must be less than or equal to %d", maxPageSize))
		return 0, 0, false
	}
	start = 0
	if token := strings.TrimSpace(r.URL.Query().Get("pageToken")); token != "" {
		start, err = strconv.Atoi(token)
		if err != nil || start < 0 {
			respondGCPWorkstationsInvalidArgument(w, path, "pageToken must be a non-negative integer")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPWorkstationsList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPWorkstationsInvalidArgument(w, path, "pageToken out of range")
		return false
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
		"unreachable":   []any{},
	})
	return true
}

func decodeGCPWorkstationsJSONBody(w http.ResponseWriter, r *http.Request, path string, requireBody bool) (map[string]any, bool) {
	if r.Body == nil {
		if requireBody {
			respondGCPWorkstationsInvalidArgument(w, path, "request body is required")
			return nil, false
		}
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPWorkstationsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		if requireBody {
			respondGCPWorkstationsInvalidArgument(w, path, "request body is required")
			return nil, false
		}
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPWorkstationsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	return body, true
}

func parseGCPWorkstationsUpdateMask(raw string) ([]string, bool) {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	allowed := map[string]struct{}{
		"displayname":    {},
		"annotations":    {},
		"labels":         {},
		"network":        {},
		"subnetwork":     {},
		"idletimeout":    {},
		"runningtimeout": {},
		"host":           {},
		"container":      {},
		"etag":           {},
	}
	for _, part := range parts {
		normalized := strings.TrimSpace(part)
		if normalized == "" {
			continue
		}
		normalized = strings.ToLower(strings.ReplaceAll(normalized, "_", ""))
		normalized = strings.ToLower(strings.ReplaceAll(normalized, ".", ""))
		if _, ok := allowed[normalized]; !ok {
			return nil, false
		}
		out = append(out, strings.TrimSpace(part))
	}
	return out, len(out) > 0
}

func gcpWorkstationsBodyMap(body map[string]any, key string) map[string]any {
	if body == nil {
		return nil
	}
	if nested, ok := body[key].(map[string]any); ok {
		return nested
	}
	return body
}

func gcpWorkstationsString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	value, _ := m[key].(string)
	return strings.TrimSpace(value)
}

func gcpWorkstationsStringSlice(raw any) []string {
	switch v := raw.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				if trimmed := strings.TrimSpace(s); trimmed != "" {
					out = append(out, trimmed)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func gcpWorkstationsClusterName(project, location, clusterID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/workstationClusters/%s", project, location, clusterID)
}

func gcpWorkstationsConfigName(project, location, clusterID, configID string) string {
	return fmt.Sprintf("%s/workstationConfigs/%s", gcpWorkstationsClusterName(project, location, clusterID), configID)
}

func gcpWorkstationsWorkstationName(project, location, clusterID, configID, workstationID string) string {
	return fmt.Sprintf("%s/workstations/%s", gcpWorkstationsConfigName(project, location, clusterID, configID), workstationID)
}

func gcpWorkstationsOperationName(project, location, operationID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID)
}

func gcpWorkstationsStateForID(workstationID string) string {
	normalized := strings.ToLower(strings.TrimSpace(workstationID))
	switch {
	case strings.Contains(normalized, "starting"):
		return "STATE_STARTING"
	case strings.Contains(normalized, "stopping"):
		return "STATE_STOPPING"
	case strings.Contains(normalized, "stopped"):
		return "STATE_STOPPED"
	case strings.Contains(normalized, "running"):
		return "STATE_RUNNING"
	default:
		return "STATE_RUNNING"
	}
}

func gcpWorkstationsWorkstationClusterFixture(project, location, clusterID string) map[string]any {
	name := gcpWorkstationsClusterName(project, location, clusterID)
	return map[string]any{
		"name":           name,
		"displayName":    strings.ToUpper(clusterID[:1]) + clusterID[1:],
		"uid":            "wsc-" + clusterID,
		"reconciling":    false,
		"annotations":    map[string]any{"stackyard.dev/stage": "staged"},
		"labels":         map[string]any{"env": "staged"},
		"createTime":     gcpWorkstationsReferenceTime.Format(time.RFC3339),
		"updateTime":     gcpWorkstationsReferenceTime.Add(2 * time.Minute).Format(time.RFC3339),
		"etag":           "etag-" + clusterID,
		"network":        fmt.Sprintf("projects/%s/global/networks/default", project),
		"subnetwork":     fmt.Sprintf("projects/%s/regions/%s/subnetworks/default", project, location),
		"controlPlaneIp": "10.10.0.10",
		"degraded":       false,
		"conditions":     []any{},
	}
}

func gcpWorkstationsWorkstationConfigFixture(project, location, clusterID, configID string) map[string]any {
	name := gcpWorkstationsConfigName(project, location, clusterID, configID)
	return map[string]any{
		"name":        name,
		"displayName": "Config " + configID,
		"uid":         "wscfg-" + configID,
		"reconciling": false,
		"annotations": map[string]any{
			"stackyard.dev/stage": "staged",
		},
		"labels": map[string]any{
			"env": "staged",
		},
		"createTime":     gcpWorkstationsReferenceTime.Add(5 * time.Minute).Format(time.RFC3339),
		"updateTime":     gcpWorkstationsReferenceTime.Add(6 * time.Minute).Format(time.RFC3339),
		"etag":           "etag-" + configID,
		"idleTimeout":    "1200s",
		"runningTimeout": "43200s",
		"host": map[string]any{
			"gceInstance": map[string]any{
				"machineType":    "e2-standard-4",
				"poolSize":       1,
				"bootDiskSizeGb": 50,
			},
		},
		"container": map[string]any{
			"image":      "us-docker.pkg.dev/cloud-workstations-images/predefined/code-oss:latest",
			"workingDir": "/home/workstation",
		},
		"degraded":   false,
		"conditions": []any{},
	}
}

func gcpWorkstationsWorkstationFixture(project, location, clusterID, configID, workstationID string) map[string]any {
	name := gcpWorkstationsWorkstationName(project, location, clusterID, configID, workstationID)
	state := gcpWorkstationsStateForID(workstationID)
	out := map[string]any{
		"name":        name,
		"displayName": "Workstation " + workstationID,
		"uid":         "ws-" + workstationID,
		"reconciling": false,
		"annotations": map[string]any{
			"stackyard.dev/stage": "staged",
		},
		"labels": map[string]any{
			"env": "staged",
		},
		"createTime": gcpWorkstationsReferenceTime.Add(10 * time.Minute).Format(time.RFC3339),
		"updateTime": gcpWorkstationsReferenceTime.Add(12 * time.Minute).Format(time.RFC3339),
		"startTime":  gcpWorkstationsReferenceTime.Add(15 * time.Minute).Format(time.RFC3339),
		"etag":       "etag-" + workstationID,
		"state":      state,
		"host":       workstationID + ".ws.stackyard.local",
	}
	return out
}

func gcpWorkstationsOperationFixture(project, location, operationID, target, verb string, done bool) map[string]any {
	id := strings.TrimSpace(operationID)
	if strings.HasPrefix(id, "operations/") {
		id = strings.TrimPrefix(id, "operations/")
	}
	name := gcpWorkstationsOperationName(project, location, id)
	return map[string]any{
		"name": name,
		"metadata": map[string]any{
			"@type":                 "type.googleapis.com/google.cloud.workstations.v1.OperationMetadata",
			"createTime":            gcpWorkstationsReferenceTime.Add(20 * time.Minute).Format(time.RFC3339),
			"endTime":               gcpWorkstationsReferenceTime.Add(21 * time.Minute).Format(time.RFC3339),
			"target":                target,
			"verb":                  verb,
			"statusMessage":         "staged",
			"requestedCancellation": false,
			"apiVersion":            "v1",
		},
		"done": done,
		"response": map[string]any{
			"@type": "type.googleapis.com/google.protobuf.Empty",
		},
	}
}

func gcpWorkstationsPolicyFixture(resource string, in map[string]any) map[string]any {
	policy := map[string]any{
		"version": 1,
		"bindings": []any{
			map[string]any{
				"role":    "roles/workstations.user",
				"members": []any{"user:stackyard@example.com"},
			},
		},
		"etag": "policy-etag",
	}
	for _, key := range []string{"version", "bindings", "etag"} {
		if in != nil {
			if value, ok := in[key]; ok {
				policy[key] = value
			}
		}
	}
	policy["resource"] = resource
	return policy
}

func parseGCPWorkstationsClusterName(name string) (project, location, clusterID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "workstationClusters" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	clusterID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || clusterID == "" {
		return "", "", "", false
	}
	return project, location, clusterID, true
}

func parseGCPWorkstationsConfigName(name string) (project, location, clusterID, configID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "workstationClusters" || parts[6] != "workstationConfigs" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	clusterID = strings.TrimSpace(parts[5])
	configID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || clusterID == "" || configID == "" {
		return "", "", "", "", false
	}
	return project, location, clusterID, configID, true
}

func parseGCPWorkstationsWorkstationName(name string) (project, location, clusterID, configID, workstationID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 10 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "workstationClusters" || parts[6] != "workstationConfigs" || parts[8] != "workstations" {
		return "", "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	clusterID = strings.TrimSpace(parts[5])
	configID = strings.TrimSpace(parts[7])
	workstationID = strings.TrimSpace(parts[9])
	if project == "" || location == "" || clusterID == "" || configID == "" || workstationID == "" {
		return "", "", "", "", "", false
	}
	return project, location, clusterID, configID, workstationID, true
}

func parseGCPWorkstationsOperationName(name string) (project, location, operationID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "operations" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	operationID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || operationID == "" {
		return "", "", "", false
	}
	return project, location, operationID, true
}

func respondGCPWorkstationsInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPWorkstationsError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPWorkstationsOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPWorkstationsError(w, http.StatusBadRequest, "OutOfRange", path, message)
}

func respondGCPWorkstationsFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPWorkstationsError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPWorkstationsNotFound(w http.ResponseWriter, path, message string) {
	respondGCPWorkstationsError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPWorkstationsAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPWorkstationsError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPWorkstationsError(w http.ResponseWriter, status int, err, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    err,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_workstations(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "workstations") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":    "InvalidArgument",
			"message":  "pageSize must be a non-negative integer",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	if r.URL.Query().Get("typedSuccess") == "1" {
		respondJSON(w, http.StatusOK, map[string]any{
			"name":     "projects/stackyard/locations/us-central1/workstationClusters/cluster-1",
			"service":  "workstations",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
