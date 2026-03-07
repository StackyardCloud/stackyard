package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	gcpStorageTransferReferenceTime      = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpStorageTransferProjectIDPattern   = regexp.MustCompile(`^[A-Za-z0-9-]{3,63}$`)
	gcpStorageTransferJobIDPattern       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	gcpStorageTransferAgentPoolIDPattern = regexp.MustCompile(`^[a-z](?:[a-z0-9._~-]*[a-z0-9])?$`)
)

func (s *Server) handleGCPStorageTransferRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_storagetransfer(w, r) {
		return true
	}

	path := normalizeGCPStorageTransferPath(rawRequestPath(r))
	if isGCPStorageTransferLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPStorageTransferListLocations(w, r, path) {
			return true
		}
		if handleGCPStorageTransferGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPStorageTransferPath(path, hasGCPStorageTransferHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPStorageTransferGetGoogleServiceAccount(w, path) {
			return true
		}
		if handleGCPStorageTransferListTransferJobs(w, r, path) {
			return true
		}
		if handleGCPStorageTransferGetTransferJob(w, r, path) {
			return true
		}
		if handleGCPStorageTransferListAgentPools(w, r, path) {
			return true
		}
		if handleGCPStorageTransferGetAgentPool(w, path) {
			return true
		}
		if handleGCPStorageTransferListOperations(w, r, path) {
			return true
		}
		if handleGCPStorageTransferGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPStorageTransferCreateTransferJob(w, r, path) {
			return true
		}
		if handleGCPStorageTransferRunTransferJob(w, r, path) {
			return true
		}
		if handleGCPStorageTransferPauseTransferOperation(w, r, path) {
			return true
		}
		if handleGCPStorageTransferResumeTransferOperation(w, r, path) {
			return true
		}
		if handleGCPStorageTransferCancelOperation(w, r, path) {
			return true
		}
		if handleGCPStorageTransferCreateAgentPool(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPStorageTransferUpdateTransferJob(w, r, path) {
			return true
		}
		if handleGCPStorageTransferUpdateAgentPool(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPStorageTransferDeleteTransferJob(w, r, path) {
			return true
		}
		if handleGCPStorageTransferDeleteAgentPool(w, path) {
			return true
		}
		if handleGCPStorageTransferDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPStorageTransferPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPStorageTransferHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "storagetransfer",
		"storagetransfer-apiv1",
		"storagetransfer_apiv1",
		"storage-transfer",
		"storage_transfer",
		"cloud-storage-transfer",
		"cloud_storage_transfer",
		"cloudstoragetransfer",
		"gcp-storage-transfer":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-storagetransfer-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/storagetransfer/apiv1")
}

func isGCPStorageTransferLocationRequest(r *http.Request, path string) bool {
	if !hasGCPStorageTransferHint(r) {
		return false
	}
	_, _, _, ok := parseGCPProjectLocationPath(path)
	return ok
}

func isGCPStorageTransferPath(path string, includeHint bool) bool {
	if _, ok := parseGCPStorageTransferGoogleServiceAccountPath(path); ok {
		return true
	}
	if parseGCPStorageTransferJobsCollectionPath(path) {
		return true
	}
	if _, ok := parseGCPStorageTransferJobPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPStorageTransferJobActionPath(path); ok {
		return true
	}
	if parseGCPStorageTransferOperationsCollectionPath(path) {
		return includeHint
	}
	if _, ok := parseGCPStorageTransferOperationPath(path); ok {
		return includeHint
	}
	if _, _, ok := parseGCPStorageTransferOperationActionPath(path); ok {
		return includeHint
	}
	if _, ok := parseGCPStorageTransferAgentPoolsCollectionPath(path); ok {
		return true
	}
	if _, _, ok := parseGCPStorageTransferAgentPoolPath(path); ok {
		return true
	}
	return false
}

func handleGCPStorageTransferListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPStorageTransferPagination(w, r, path, 256)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpStorageTransferLocation(project, "us-central1"),
		gcpStorageTransferLocation(project, "global"),
	}
	return respondGCPStorageTransferList(w, "locations", items, pageSize, start, path, false)
}

func handleGCPStorageTransferGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpStorageTransferLocation(project, location))
	return true
}

func handleGCPStorageTransferGetGoogleServiceAccount(w http.ResponseWriter, path string) bool {
	projectID, ok := parseGCPStorageTransferGoogleServiceAccountPath(path)
	if !ok {
		return false
	}
	if !isGCPStorageTransferProjectID(projectID) {
		respondGCPStorageTransferInvalidArgument(w, path, "projectId is invalid")
		return true
	}
	respondJSON(w, http.StatusOK, gcpStorageTransferGoogleServiceAccount(projectID))
	return true
}

func handleGCPStorageTransferCreateTransferJob(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPStorageTransferJobsCollectionPath(path) {
		return false
	}
	body, valid := decodeGCPStorageTransferJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	transferJob := gcpStorageTransferBodyMap(body, "transferJob")
	if len(transferJob) == 0 {
		respondGCPStorageTransferInvalidArgument(w, path, "transferJob is required")
		return true
	}
	projectID := strings.TrimSpace(gcpStorageTransferString(transferJob, "projectId"))
	if projectID == "" {
		projectID = strings.TrimSpace(gcpStorageTransferString(body, "projectId"))
	}
	if !isGCPStorageTransferProjectID(projectID) {
		respondGCPStorageTransferInvalidArgument(w, path, "transferJob.projectId is required")
		return true
	}
	transferSpec, _ := transferJob["transferSpec"].(map[string]any)
	if len(transferSpec) == 0 {
		respondGCPStorageTransferInvalidArgument(w, path, "transferJob.transferSpec is required")
		return true
	}
	status := strings.TrimSpace(gcpStorageTransferString(transferJob, "status"))
	if status == "" {
		status = "ENABLED"
	}
	if !isGCPStorageTransferJobStatus(status) {
		respondGCPStorageTransferInvalidArgument(w, path, "transferJob.status is invalid")
		return true
	}

	jobID := "job-1"
	if providedName := strings.TrimSpace(gcpStorageTransferString(transferJob, "name")); providedName != "" {
		parsedJobID, ok := parseGCPStorageTransferJobResourceName(providedName)
		if !ok {
			respondGCPStorageTransferInvalidArgument(w, path, "transferJob.name is invalid")
			return true
		}
		jobID = parsedJobID
	}
	if !isGCPStorageTransferJobID(jobID) {
		respondGCPStorageTransferInvalidArgument(w, path, "transfer job name is invalid")
		return true
	}
	if isGCPStorageTransferMissingID(jobID) || strings.Contains(strings.ToLower(jobID), "existing") {
		respondGCPStorageTransferAlreadyExists(w, path, "transfer job already exists")
		return true
	}

	created := gcpStorageTransferTransferJob(projectID, jobID, status)
	applyGCPStorageTransferTransferJobOverrides(created, transferJob)
	respondJSON(w, http.StatusOK, created)
	return true
}

func handleGCPStorageTransferUpdateTransferJob(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.Method != http.MethodPatch {
		return false
	}

	jobID, ok := parseGCPStorageTransferJobPath(path)
	if !ok {
		return false
	}

	body, valid := decodeGCPStorageTransferJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	projectID := strings.TrimSpace(gcpStorageTransferString(body, "projectId"))
	if !isGCPStorageTransferProjectID(projectID) {
		respondGCPStorageTransferInvalidArgument(w, path, "projectId is required")
		return true
	}
	if isGCPStorageTransferMissingID(jobID) {
		respondGCPStorageTransferNotFound(w, path, "transfer job not found")
		return true
	}

	updateMask := strings.TrimSpace(r.URL.Query().Get("updateTransferJobFieldMask"))
	if updateMask == "" {
		updateMask = strings.TrimSpace(gcpStorageTransferString(body, "updateTransferJobFieldMask"))
	}
	if !validateGCPStorageTransferTransferJobUpdateMask(updateMask) {
		respondGCPStorageTransferInvalidArgument(w, path, "updateTransferJobFieldMask is invalid")
		return true
	}

	transferJob := gcpStorageTransferBodyMap(body, "transferJob")
	if len(transferJob) == 0 {
		respondGCPStorageTransferInvalidArgument(w, path, "transferJob is required")
		return true
	}
	expectedName := gcpStorageTransferTransferJobName(jobID)
	if strings.TrimSpace(gcpStorageTransferString(transferJob, "name")) != expectedName {
		respondGCPStorageTransferInvalidArgument(w, path, "transferJob.name must match requested resource")
		return true
	}
	if specRaw, ok := transferJob["transferSpec"]; ok {
		transferSpec, _ := specRaw.(map[string]any)
		if len(transferSpec) == 0 {
			respondGCPStorageTransferInvalidArgument(w, path, "transferJob.transferSpec cannot be empty")
			return true
		}
	}
	if status := strings.TrimSpace(gcpStorageTransferString(transferJob, "status")); status != "" && !isGCPStorageTransferJobStatus(status) {
		respondGCPStorageTransferInvalidArgument(w, path, "transferJob.status is invalid")
		return true
	}

	updated := gcpStorageTransferTransferJob(projectID, jobID, gcpStorageTransferJobStatusFromID(jobID))
	applyGCPStorageTransferTransferJobOverrides(updated, transferJob)
	respondJSON(w, http.StatusOK, updated)
	return true
}

func handleGCPStorageTransferGetTransferJob(w http.ResponseWriter, r *http.Request, path string) bool {
	jobID, ok := parseGCPStorageTransferJobPath(path)
	if !ok {
		return false
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("projectId"))
	if !isGCPStorageTransferProjectID(projectID) {
		respondGCPStorageTransferInvalidArgument(w, path, "projectId is required")
		return true
	}
	if !isGCPStorageTransferJobID(jobID) {
		respondGCPStorageTransferInvalidArgument(w, path, "transfer job name is invalid")
		return true
	}
	if isGCPStorageTransferMissingID(jobID) {
		respondGCPStorageTransferNotFound(w, path, "transfer job not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpStorageTransferTransferJob(projectID, jobID, gcpStorageTransferJobStatusFromID(jobID)))
	return true
}

func handleGCPStorageTransferListTransferJobs(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPStorageTransferJobsCollectionPath(path) {
		return false
	}
	pageSize, start, valid := parseGCPStorageTransferPagination(w, r, path, 256)
	if !valid {
		return true
	}
	filterRaw := strings.TrimSpace(r.URL.Query().Get("filter"))
	filter, err := parseGCPStorageTransferFilterJSON(filterRaw)
	if err != nil {
		respondGCPStorageTransferInvalidArgument(w, path, "filter must be valid JSON")
		return true
	}
	projectID := strings.TrimSpace(gcpStorageTransferString(filter, "projectId"))
	if projectID == "" {
		projectID = strings.TrimSpace(gcpStorageTransferString(filter, "project_id"))
	}
	if !isGCPStorageTransferProjectID(projectID) {
		respondGCPStorageTransferInvalidArgument(w, path, "filter.projectId is required")
		return true
	}

	jobNames := gcpStorageTransferAnyStringSet(filter["jobNames"])
	if len(jobNames) == 0 {
		jobNames = gcpStorageTransferAnyStringSet(filter["job_names"])
	}
	jobStatuses := gcpStorageTransferAnyStringSet(filter["jobStatuses"])
	if len(jobStatuses) == 0 {
		jobStatuses = gcpStorageTransferAnyStringSet(filter["job_statuses"])
	}

	items := []map[string]any{
		gcpStorageTransferTransferJob(projectID, "job-1", "ENABLED"),
		gcpStorageTransferTransferJob(projectID, "job-disabled", "DISABLED"),
	}
	if len(jobNames) > 0 {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			name := strings.TrimSpace(gcpStorageTransferString(item, "name"))
			jobID := strings.TrimPrefix(name, "transferJobs/")
			if _, ok := jobNames[name]; ok {
				filtered = append(filtered, item)
				continue
			}
			if _, ok := jobNames[jobID]; ok {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if len(jobStatuses) > 0 {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			status := strings.ToUpper(strings.TrimSpace(gcpStorageTransferString(item, "status")))
			if _, ok := jobStatuses[status]; ok {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	sort.Slice(items, func(i, j int) bool {
		return gcpStorageTransferString(items[i], "name") < gcpStorageTransferString(items[j], "name")
	})

	return respondGCPStorageTransferList(w, "transferJobs", items, pageSize, start, path, false)
}

func handleGCPStorageTransferDeleteTransferJob(w http.ResponseWriter, r *http.Request, path string) bool {
	jobID, ok := parseGCPStorageTransferJobPath(path)
	if !ok {
		return false
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("projectId"))
	if !isGCPStorageTransferProjectID(projectID) {
		respondGCPStorageTransferInvalidArgument(w, path, "projectId is required")
		return true
	}
	if !isGCPStorageTransferJobID(jobID) {
		respondGCPStorageTransferInvalidArgument(w, path, "transfer job name is invalid")
		return true
	}
	if isGCPStorageTransferMissingID(jobID) {
		respondGCPStorageTransferNotFound(w, path, "transfer job not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPStorageTransferRunTransferJob(w http.ResponseWriter, r *http.Request, path string) bool {
	jobID, action, ok := parseGCPStorageTransferJobActionPath(path)
	if !ok || action != "run" {
		return false
	}
	body, valid := decodeGCPStorageTransferJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	projectID := strings.TrimSpace(gcpStorageTransferString(body, "projectId"))
	if !isGCPStorageTransferProjectID(projectID) {
		respondGCPStorageTransferInvalidArgument(w, path, "projectId is required")
		return true
	}
	if isGCPStorageTransferMissingID(jobID) {
		respondGCPStorageTransferNotFound(w, path, "transfer job not found")
		return true
	}
	if strings.Contains(strings.ToLower(jobID), "running") {
		respondGCPStorageTransferFailedPrecondition(w, path, "transfer job already has an active run")
		return true
	}

	operationID := "run." + jobID
	respondJSON(w, http.StatusOK, gcpStorageTransferOperationEnvelope(projectID, operationID, jobID, "IN_PROGRESS", false))
	return true
}

func handleGCPStorageTransferPauseTransferOperation(w http.ResponseWriter, r *http.Request, path string) bool {
	operationID, action, ok := parseGCPStorageTransferOperationActionPath(path)
	if !ok || action != "pause" {
		return false
	}
	if _, valid := decodeGCPStorageTransferJSONBody(w, r, path); !valid {
		return true
	}
	if isGCPStorageTransferMissingID(operationID) {
		respondGCPStorageTransferNotFound(w, path, "transfer operation not found")
		return true
	}
	if strings.Contains(strings.ToLower(operationID), "paused") {
		respondGCPStorageTransferFailedPrecondition(w, path, "transfer operation is already paused")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPStorageTransferResumeTransferOperation(w http.ResponseWriter, r *http.Request, path string) bool {
	operationID, action, ok := parseGCPStorageTransferOperationActionPath(path)
	if !ok || action != "resume" {
		return false
	}
	if _, valid := decodeGCPStorageTransferJSONBody(w, r, path); !valid {
		return true
	}
	if isGCPStorageTransferMissingID(operationID) {
		respondGCPStorageTransferNotFound(w, path, "transfer operation not found")
		return true
	}
	if strings.Contains(strings.ToLower(operationID), "running") || strings.Contains(strings.ToLower(operationID), "active") {
		respondGCPStorageTransferFailedPrecondition(w, path, "transfer operation is not paused")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPStorageTransferCancelOperation(w http.ResponseWriter, r *http.Request, path string) bool {
	operationID, action, ok := parseGCPStorageTransferOperationActionPath(path)
	if !ok || action != "cancel" {
		return false
	}
	if _, valid := decodeGCPStorageTransferJSONBody(w, r, path); !valid {
		return true
	}
	if isGCPStorageTransferMissingID(operationID) {
		respondGCPStorageTransferNotFound(w, path, "transfer operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPStorageTransferGetOperation(w http.ResponseWriter, path string) bool {
	operationID, ok := parseGCPStorageTransferOperationPath(path)
	if !ok {
		return false
	}
	if isGCPStorageTransferMissingID(operationID) {
		respondGCPStorageTransferNotFound(w, path, "transfer operation not found")
		return true
	}
	projectID := "stackyard"
	jobID := gcpStorageTransferJobIDFromOperationID(operationID)
	status := gcpStorageTransferOperationStatusFromID(operationID)
	done := gcpStorageTransferOperationDone(status)
	respondJSON(w, http.StatusOK, gcpStorageTransferOperationEnvelope(projectID, operationID, jobID, status, done))
	return true
}

func handleGCPStorageTransferListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	if !parseGCPStorageTransferOperationsCollectionPath(path) {
		return false
	}
	pageSize, start, valid := parseGCPStorageTransferPagination(w, r, path, 256)
	if !valid {
		return true
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name != "" && name != "transferOperations" && !strings.HasPrefix(name, "transferOperations/") {
		respondGCPStorageTransferInvalidArgument(w, path, "name must be transferOperations or transferOperations/*")
		return true
	}
	projectID := "stackyard"
	items := []map[string]any{
		gcpStorageTransferOperationEnvelope(projectID, "run.job-1", "job-1", "IN_PROGRESS", false),
		gcpStorageTransferOperationEnvelope(projectID, "run.job-succeeded", "job-succeeded", "SUCCESS", true),
	}
	if name != "" {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			operationName := strings.TrimSpace(gcpStorageTransferString(item, "name"))
			if strings.HasPrefix(operationName, name) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	return respondGCPStorageTransferList(w, "operations", items, pageSize, start, path, false)
}

func handleGCPStorageTransferDeleteOperation(w http.ResponseWriter, path string) bool {
	operationID, ok := parseGCPStorageTransferOperationPath(path)
	if !ok {
		return false
	}
	if isGCPStorageTransferMissingID(operationID) {
		respondGCPStorageTransferNotFound(w, path, "transfer operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPStorageTransferCreateAgentPool(w http.ResponseWriter, r *http.Request, path string) bool {
	projectID, ok := parseGCPStorageTransferAgentPoolsCollectionPath(path)
	if !ok {
		return false
	}
	if !isGCPStorageTransferProjectID(projectID) {
		respondGCPStorageTransferInvalidArgument(w, path, "projectId is invalid")
		return true
	}

	body, valid := decodeGCPStorageTransferJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	agentPool := gcpStorageTransferBodyMap(body, "agentPool")
	if len(agentPool) == 0 {
		respondGCPStorageTransferInvalidArgument(w, path, "agentPool is required")
		return true
	}
	agentPoolID := strings.TrimSpace(r.URL.Query().Get("agentPoolId"))
	if agentPoolID == "" {
		agentPoolID = strings.TrimSpace(gcpStorageTransferString(body, "agentPoolId"))
	}
	if agentPoolID == "" {
		if providedName := strings.TrimSpace(gcpStorageTransferString(agentPool, "name")); providedName != "" {
			parsedProjectID, parsedAgentPoolID, parsed := parseGCPStorageTransferAgentPoolResourceName(providedName)
			if !parsed || parsedProjectID != projectID {
				respondGCPStorageTransferInvalidArgument(w, path, "agentPool.name is invalid")
				return true
			}
			agentPoolID = parsedAgentPoolID
		}
	}
	if !isGCPStorageTransferAgentPoolID(agentPoolID) {
		respondGCPStorageTransferInvalidArgument(w, path, "agentPoolId is required")
		return true
	}
	if isGCPStorageTransferMissingID(agentPoolID) || strings.Contains(strings.ToLower(agentPoolID), "existing") {
		respondGCPStorageTransferAlreadyExists(w, path, "agent pool already exists")
		return true
	}
	expectedName := gcpStorageTransferAgentPoolName(projectID, agentPoolID)
	if providedName := strings.TrimSpace(gcpStorageTransferString(agentPool, "name")); providedName != "" && providedName != expectedName {
		respondGCPStorageTransferInvalidArgument(w, path, "agentPool.name must match parent and agentPoolId")
		return true
	}

	created := gcpStorageTransferAgentPool(projectID, agentPoolID, "CREATED")
	applyGCPStorageTransferAgentPoolOverrides(created, agentPool)
	respondJSON(w, http.StatusOK, created)
	return true
}

func handleGCPStorageTransferGetAgentPool(w http.ResponseWriter, path string) bool {
	projectID, agentPoolID, ok := parseGCPStorageTransferAgentPoolPath(path)
	if !ok {
		return false
	}
	if !isGCPStorageTransferProjectID(projectID) {
		respondGCPStorageTransferInvalidArgument(w, path, "projectId is invalid")
		return true
	}
	if !isGCPStorageTransferAgentPoolID(agentPoolID) {
		respondGCPStorageTransferInvalidArgument(w, path, "agent pool name is invalid")
		return true
	}
	if isGCPStorageTransferMissingID(agentPoolID) {
		respondGCPStorageTransferNotFound(w, path, "agent pool not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpStorageTransferAgentPool(projectID, agentPoolID, gcpStorageTransferAgentPoolStateFromID(agentPoolID)))
	return true
}

func handleGCPStorageTransferListAgentPools(w http.ResponseWriter, r *http.Request, path string) bool {
	projectID, ok := parseGCPStorageTransferAgentPoolsCollectionPath(path)
	if !ok {
		return false
	}
	if !isGCPStorageTransferProjectID(projectID) {
		respondGCPStorageTransferInvalidArgument(w, path, "projectId is invalid")
		return true
	}
	pageSize, start, valid := parseGCPStorageTransferPagination(w, r, path, 256)
	if !valid {
		return true
	}
	filterRaw := strings.TrimSpace(r.URL.Query().Get("filter"))
	filter := map[string]any{}
	if filterRaw != "" {
		parsed, err := parseGCPStorageTransferFilterJSON(filterRaw)
		if err != nil {
			respondGCPStorageTransferInvalidArgument(w, path, "filter must be valid JSON")
			return true
		}
		filter = parsed
		if projectInFilter := strings.TrimSpace(gcpStorageTransferString(filter, "projectId")); projectInFilter != "" && projectInFilter != projectID {
			respondGCPStorageTransferInvalidArgument(w, path, "filter.projectId must match parent project")
			return true
		}
	}

	items := []map[string]any{
		gcpStorageTransferAgentPool(projectID, "agentpool-1", "CREATED"),
		gcpStorageTransferAgentPool(projectID, "agentpool-2", "CREATING"),
	}
	nameFilter := gcpStorageTransferAnyStringSet(filter["agentPoolNames"])
	if len(nameFilter) == 0 {
		nameFilter = gcpStorageTransferAnyStringSet(filter["agent_pool_names"])
	}
	if len(nameFilter) > 0 {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if _, ok := nameFilter[gcpStorageTransferString(item, "name")]; ok {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	return respondGCPStorageTransferList(w, "agentPools", items, pageSize, start, path, false)
}

func handleGCPStorageTransferUpdateAgentPool(w http.ResponseWriter, r *http.Request, path string) bool {
	projectID, agentPoolID, ok := parseGCPStorageTransferAgentPoolPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPStorageTransferJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	updateMask := strings.TrimSpace(r.URL.Query().Get("updateMask"))
	if updateMask == "" {
		updateMask = strings.TrimSpace(gcpStorageTransferString(body, "updateMask"))
	}
	if !validateGCPStorageTransferAgentPoolUpdateMask(updateMask) {
		respondGCPStorageTransferInvalidArgument(w, path, "updateMask is invalid")
		return true
	}
	agentPool := gcpStorageTransferBodyMap(body, "agentPool")
	if len(agentPool) == 0 {
		respondGCPStorageTransferInvalidArgument(w, path, "agentPool is required")
		return true
	}
	expectedName := gcpStorageTransferAgentPoolName(projectID, agentPoolID)
	if strings.TrimSpace(gcpStorageTransferString(agentPool, "name")) != expectedName {
		respondGCPStorageTransferInvalidArgument(w, path, "agentPool.name must match requested resource")
		return true
	}
	if isGCPStorageTransferMissingID(agentPoolID) {
		respondGCPStorageTransferNotFound(w, path, "agent pool not found")
		return true
	}

	updated := gcpStorageTransferAgentPool(projectID, agentPoolID, gcpStorageTransferAgentPoolStateFromID(agentPoolID))
	applyGCPStorageTransferAgentPoolOverrides(updated, agentPool)
	respondJSON(w, http.StatusOK, updated)
	return true
}

func handleGCPStorageTransferDeleteAgentPool(w http.ResponseWriter, path string) bool {
	_, agentPoolID, ok := parseGCPStorageTransferAgentPoolPath(path)
	if !ok {
		return false
	}
	if isGCPStorageTransferMissingID(agentPoolID) {
		respondGCPStorageTransferNotFound(w, path, "agent pool not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPStorageTransferGoogleServiceAccountPath(path string) (projectID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "googleServiceAccounts" {
		return "", false
	}
	projectID = strings.TrimSpace(parts[3])
	if projectID == "" {
		return "", false
	}
	return projectID, true
}

func parseGCPStorageTransferJobsCollectionPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "transferJobs"
}

func parseGCPStorageTransferJobPath(path string) (jobID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "transferJobs" {
		return "", false
	}
	jobID = strings.TrimSpace(parts[3])
	if jobID == "" || strings.Contains(jobID, ":") {
		return "", false
	}
	return jobID, true
}

func parseGCPStorageTransferJobActionPath(path string) (jobID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "transferJobs" {
		return "", "", false
	}
	jobID, action, ok = strings.Cut(strings.TrimSpace(parts[3]), ":")
	if !ok || strings.TrimSpace(jobID) == "" {
		return "", "", false
	}
	action = strings.TrimSpace(action)
	if action == "" {
		return "", "", false
	}
	return strings.TrimSpace(jobID), action, true
}

func parseGCPStorageTransferOperationsCollectionPath(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 3 && parts[0] == "gcp" && parts[1] == "v1" && parts[2] == "transferOperations"
}

func parseGCPStorageTransferOperationPath(path string) (operationID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "transferOperations" {
		return "", false
	}
	operationID = strings.TrimSpace(parts[3])
	if operationID == "" || strings.Contains(operationID, ":") {
		return "", false
	}
	return operationID, true
}

func parseGCPStorageTransferOperationActionPath(path string) (operationID, action string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 4 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "transferOperations" {
		return "", "", false
	}
	operationID, action, ok = strings.Cut(strings.TrimSpace(parts[3]), ":")
	if !ok || strings.TrimSpace(operationID) == "" {
		return "", "", false
	}
	action = strings.TrimSpace(action)
	if action == "" {
		return "", "", false
	}
	return strings.TrimSpace(operationID), action, true
}

func parseGCPStorageTransferAgentPoolsCollectionPath(path string) (projectID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "agentPools" {
		return "", false
	}
	projectID = strings.TrimSpace(parts[3])
	if projectID == "" {
		return "", false
	}
	return projectID, true
}

func parseGCPStorageTransferAgentPoolPath(path string) (projectID, agentPoolID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "agentPools" {
		return "", "", false
	}
	projectID = strings.TrimSpace(parts[3])
	agentPoolID = strings.TrimSpace(parts[5])
	if projectID == "" || agentPoolID == "" || strings.Contains(agentPoolID, ":") {
		return "", "", false
	}
	return projectID, agentPoolID, true
}

func parseGCPStorageTransferJobResourceName(name string) (jobID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 2 || parts[0] != "transferJobs" {
		return "", false
	}
	jobID = strings.TrimSpace(parts[1])
	if jobID == "" {
		return "", false
	}
	return jobID, true
}

func parseGCPStorageTransferAgentPoolResourceName(name string) (projectID, agentPoolID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "agentPools" {
		return "", "", false
	}
	projectID = strings.TrimSpace(parts[1])
	agentPoolID = strings.TrimSpace(parts[3])
	if projectID == "" || agentPoolID == "" {
		return "", "", false
	}
	return projectID, agentPoolID, true
}

func parseGCPStorageTransferPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPStorageTransferInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize > maxPageSize {
		respondGCPStorageTransferOutOfRange(w, path, fmt.Sprintf("pageSize must be <= %d", maxPageSize))
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = parseOptionalNonNegativeInt(pageToken)
		if err != nil {
			respondGCPStorageTransferInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func respondGCPStorageTransferList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string, includeUnreachable bool) bool {
	if start > len(items) {
		respondGCPStorageTransferInvalidArgument(w, path, "pageToken is out of range")
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
	response := map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	}
	if includeUnreachable {
		response["unreachable"] = []string{}
	}
	respondJSON(w, http.StatusOK, response)
	return true
}

func decodeGCPStorageTransferJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPStorageTransferInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPStorageTransferInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func decodeGCPStorageTransferJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, ok := decodeGCPStorageTransferJSONBody(w, r, path)
	if !ok {
		return nil, false
	}
	if len(body) == 0 {
		respondGCPStorageTransferInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func parseGCPStorageTransferFilterJSON(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty")
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func validateGCPStorageTransferTransferJobUpdateMask(mask string) bool {
	mask = strings.TrimSpace(mask)
	if mask == "" {
		return false
	}
	allowed := map[string]struct{}{
		"description":         {},
		"transferSpec":        {},
		"transfer_spec":       {},
		"notificationConfig":  {},
		"notification_config": {},
		"loggingConfig":       {},
		"logging_config":      {},
		"status":              {},
		"schedule":            {},
	}
	parts := strings.Split(mask, ",")
	for _, part := range parts {
		if _, ok := allowed[strings.TrimSpace(part)]; !ok {
			return false
		}
	}
	return true
}

func validateGCPStorageTransferAgentPoolUpdateMask(mask string) bool {
	mask = strings.TrimSpace(mask)
	if mask == "" {
		return false
	}
	allowed := map[string]struct{}{
		"display_name":    {},
		"displayName":     {},
		"bandwidth_limit": {},
		"bandwidthLimit":  {},
	}
	parts := strings.Split(mask, ",")
	for _, part := range parts {
		if _, ok := allowed[strings.TrimSpace(part)]; !ok {
			return false
		}
	}
	return true
}

func isGCPStorageTransferProjectID(projectID string) bool {
	projectID = strings.TrimSpace(projectID)
	return projectID != "" && gcpStorageTransferProjectIDPattern.MatchString(projectID)
}

func isGCPStorageTransferJobID(jobID string) bool {
	jobID = strings.TrimSpace(jobID)
	return jobID != "" && gcpStorageTransferJobIDPattern.MatchString(jobID)
}

func isGCPStorageTransferAgentPoolID(agentPoolID string) bool {
	agentPoolID = strings.TrimSpace(agentPoolID)
	return agentPoolID != "" &&
		!strings.HasPrefix(strings.ToLower(agentPoolID), "goog") &&
		gcpStorageTransferAgentPoolIDPattern.MatchString(agentPoolID)
}

func isGCPStorageTransferJobStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "ENABLED", "DISABLED", "DELETED":
		return true
	default:
		return false
	}
}

func isGCPStorageTransferMissingID(id string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	return id == "" || strings.Contains(id, "missing")
}

func gcpStorageTransferBodyMap(body map[string]any, key string) map[string]any {
	nested, _ := body[key].(map[string]any)
	if len(nested) > 0 {
		return nested
	}
	return map[string]any{}
}

func gcpStorageTransferAnyStringSet(raw any) map[string]struct{} {
	out := map[string]struct{}{}
	switch typed := raw.(type) {
	case []any:
		for _, item := range typed {
			value, _ := item.(string)
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			out[value] = struct{}{}
			out[strings.ToUpper(value)] = struct{}{}
		}
	}
	return out
}

func gcpStorageTransferString(m map[string]any, key string) string {
	value, _ := m[key].(string)
	return strings.TrimSpace(value)
}

func gcpStorageTransferLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Storage Transfer " + location,
		"metadata": map[string]any{
			"agentPoolsAvailable":   true,
			"transferJobsAvailable": true,
		},
	}
}

func gcpStorageTransferGoogleServiceAccount(projectID string) map[string]any {
	return map[string]any{
		"accountEmail": fmt.Sprintf("project-%s@storage-transfer-service.iam.gserviceaccount.com", projectID),
		"subjectId":    fmt.Sprintf("subject-%s", projectID),
	}
}

func gcpStorageTransferTransferJobName(jobID string) string {
	return "transferJobs/" + strings.TrimSpace(jobID)
}

func gcpStorageTransferTransferJob(projectID, jobID, status string) map[string]any {
	return map[string]any{
		"name":                 gcpStorageTransferTransferJobName(jobID),
		"description":          "Stackyard Storage Transfer job " + jobID,
		"projectId":            projectID,
		"status":               strings.ToUpper(strings.TrimSpace(status)),
		"creationTime":         gcpStorageTransferReferenceTime.Format(time.RFC3339),
		"lastModificationTime": gcpStorageTransferReferenceTime.Add(2 * time.Hour).Format(time.RFC3339),
		"latestOperationName":  "transferOperations/run." + jobID,
		"transferSpec": map[string]any{
			"gcsDataSource": map[string]any{
				"bucketName": "stackyard-source-bucket",
			},
			"gcsDataSink": map[string]any{
				"bucketName": "stackyard-destination-bucket",
			},
			"objectConditions": map[string]any{
				"includePrefixes": []string{"incoming/"},
			},
			"transferOptions": map[string]any{
				"overwriteObjectsAlreadyExistingInSink": true,
				"deleteObjectsUniqueInSink":             false,
			},
		},
		"schedule": map[string]any{
			"scheduleStartDate": map[string]any{
				"year":  2026,
				"month": 1,
				"day":   1,
			},
			"startTimeOfDay": map[string]any{
				"hours":   2,
				"minutes": 0,
				"seconds": 0,
			},
			"repeatInterval": "86400s",
		},
		"notificationConfig": map[string]any{
			"pubsubTopic":   fmt.Sprintf("projects/%s/topics/stackyard-storage-transfer", projectID),
			"eventTypes":    []string{"TRANSFER_OPERATION_SUCCESS"},
			"payloadFormat": "JSON",
		},
		"loggingConfig": map[string]any{
			"logActions":      []string{"FIND", "TRANSFER"},
			"logActionStates": []string{"SUCCEEDED", "FAILED"},
		},
	}
}

func applyGCPStorageTransferTransferJobOverrides(target map[string]any, overrides map[string]any) {
	if description := strings.TrimSpace(gcpStorageTransferString(overrides, "description")); description != "" {
		target["description"] = description
	}
	if status := strings.TrimSpace(gcpStorageTransferString(overrides, "status")); status != "" {
		target["status"] = strings.ToUpper(status)
	}
	if transferSpec, ok := overrides["transferSpec"].(map[string]any); ok && len(transferSpec) > 0 {
		target["transferSpec"] = transferSpec
	}
	if schedule, ok := overrides["schedule"].(map[string]any); ok && len(schedule) > 0 {
		target["schedule"] = schedule
	}
	if notificationConfig, ok := overrides["notificationConfig"].(map[string]any); ok && len(notificationConfig) > 0 {
		target["notificationConfig"] = notificationConfig
	}
	if loggingConfig, ok := overrides["loggingConfig"].(map[string]any); ok && len(loggingConfig) > 0 {
		target["loggingConfig"] = loggingConfig
	}
}

func gcpStorageTransferAgentPoolName(projectID, agentPoolID string) string {
	return fmt.Sprintf("projects/%s/agentPools/%s", projectID, agentPoolID)
}

func gcpStorageTransferAgentPool(projectID, agentPoolID, state string) map[string]any {
	return map[string]any{
		"name":        gcpStorageTransferAgentPoolName(projectID, agentPoolID),
		"displayName": "Stackyard Agent Pool " + agentPoolID,
		"state":       strings.ToUpper(strings.TrimSpace(state)),
		"bandwidthLimit": map[string]any{
			"limitMbps": 100,
		},
	}
}

func applyGCPStorageTransferAgentPoolOverrides(target map[string]any, overrides map[string]any) {
	if displayName := strings.TrimSpace(gcpStorageTransferString(overrides, "displayName")); displayName != "" {
		target["displayName"] = displayName
	}
	if displayName := strings.TrimSpace(gcpStorageTransferString(overrides, "display_name")); displayName != "" {
		target["displayName"] = displayName
	}
	if bandwidthLimit, ok := overrides["bandwidthLimit"].(map[string]any); ok && len(bandwidthLimit) > 0 {
		target["bandwidthLimit"] = bandwidthLimit
	}
	if bandwidthLimit, ok := overrides["bandwidth_limit"].(map[string]any); ok && len(bandwidthLimit) > 0 {
		target["bandwidthLimit"] = bandwidthLimit
	}
}

func gcpStorageTransferOperationEnvelope(projectID, operationID, jobID, state string, done bool) map[string]any {
	metadata := map[string]any{
		"@type":           "type.googleapis.com/google.storagetransfer.v1.TransferOperation",
		"name":            "transferOperations/" + operationID,
		"projectId":       projectID,
		"transferJobName": gcpStorageTransferTransferJobName(jobID),
		"status":          strings.ToUpper(strings.TrimSpace(state)),
		"startTime":       gcpStorageTransferReferenceTime.Add(5 * time.Minute).Format(time.RFC3339),
	}
	if done {
		metadata["endTime"] = gcpStorageTransferReferenceTime.Add(35 * time.Minute).Format(time.RFC3339)
	}
	operation := map[string]any{
		"name":     "transferOperations/" + operationID,
		"done":     done,
		"metadata": metadata,
	}
	if done {
		operation["response"] = map[string]any{
			"@type": "type.googleapis.com/google.protobuf.Empty",
		}
	}
	return operation
}

func gcpStorageTransferJobStatusFromID(jobID string) string {
	id := strings.ToLower(strings.TrimSpace(jobID))
	switch {
	case strings.Contains(id, "disabled"):
		return "DISABLED"
	case strings.Contains(id, "deleted"):
		return "DELETED"
	default:
		return "ENABLED"
	}
}

func gcpStorageTransferAgentPoolStateFromID(agentPoolID string) string {
	id := strings.ToLower(strings.TrimSpace(agentPoolID))
	switch {
	case strings.Contains(id, "creating"):
		return "CREATING"
	case strings.Contains(id, "deleting"):
		return "DELETING"
	default:
		return "CREATED"
	}
}

func gcpStorageTransferOperationStatusFromID(operationID string) string {
	id := strings.ToLower(strings.TrimSpace(operationID))
	switch {
	case strings.Contains(id, "paused"):
		return "PAUSED"
	case strings.Contains(id, "success"), strings.Contains(id, "succeeded"):
		return "SUCCESS"
	case strings.Contains(id, "failed"):
		return "FAILED"
	case strings.Contains(id, "aborted"), strings.Contains(id, "cancel"):
		return "ABORTED"
	default:
		return "IN_PROGRESS"
	}
}

func gcpStorageTransferOperationDone(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "SUCCESS", "FAILED", "ABORTED":
		return true
	default:
		return false
	}
}

func gcpStorageTransferJobIDFromOperationID(operationID string) string {
	trimmed := strings.TrimSpace(operationID)
	if strings.HasPrefix(trimmed, "run.") {
		trimmed = strings.TrimPrefix(trimmed, "run.")
	}
	if strings.HasPrefix(trimmed, "transferOperations/") {
		trimmed = strings.TrimPrefix(trimmed, "transferOperations/")
	}
	if trimmed == "" {
		return "job-1"
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) >= 2 {
		last := strings.TrimSpace(parts[len(parts)-1])
		if last != "" {
			return last
		}
	}
	return "job-1"
}

func respondGCPStorageTransferInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPStorageTransferError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPStorageTransferNotFound(w http.ResponseWriter, path, message string) {
	respondGCPStorageTransferError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPStorageTransferAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPStorageTransferError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPStorageTransferFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPStorageTransferError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPStorageTransferOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPStorageTransferError(w, http.StatusBadRequest, "OutOfRange", path, message)
}

func respondGCPStorageTransferError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_storagetransfer(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "storagetransfer") {
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
			"name":     "projects/stackyard/locations/us-central1/storagetransfer/sample",
			"service":  "storagetransfer",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
