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
	gcpStorageBatchOperationsReferenceTime    = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpStorageBatchOperationsJobIDPattern     = regexp.MustCompile(`^[a-z0-9](?:[-a-z0-9]{0,126}[a-z0-9])?$`)
	gcpStorageBatchOperationsRequestIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89aAbB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
)

func (s *Server) handleGCPStorageBatchOperationsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_storagebatchoperations(w, r) {
		return true
	}

	path := normalizeGCPStorageBatchOperationsPath(rawRequestPath(r))
	if isGCPStorageBatchOperationsLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPStorageBatchOperationsListLocations(w, r, path) {
			return true
		}
		if handleGCPStorageBatchOperationsGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPStorageBatchOperationsPath(path, hasGCPStorageBatchOperationsHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPStorageBatchOperationsListJobs(w, r, path) {
			return true
		}
		if handleGCPStorageBatchOperationsGetJob(w, path) {
			return true
		}
		if handleGCPStorageBatchOperationsListBucketOperations(w, r, path) {
			return true
		}
		if handleGCPStorageBatchOperationsGetBucketOperation(w, path) {
			return true
		}
		if handleGCPStorageBatchOperationsListOperations(w, r, path) {
			return true
		}
		if handleGCPStorageBatchOperationsGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPStorageBatchOperationsCreateJob(w, r, path) {
			return true
		}
		if handleGCPStorageBatchOperationsCancelJob(w, r, path) {
			return true
		}
		if handleGCPStorageBatchOperationsCancelOperation(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPStorageBatchOperationsDeleteJob(w, r, path) {
			return true
		}
		if handleGCPStorageBatchOperationsDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPStorageBatchOperationsPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPStorageBatchOperationsHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "storagebatchoperations",
		"storagebatchoperations-apiv1",
		"storagebatchoperations_apiv1",
		"storage-batch-operations",
		"storage_batch_operations",
		"cloud-storage-batch-operations",
		"cloud_storage_batch_operations",
		"gcp-storage-batch-operations":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-storagebatchoperations-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/storagebatchoperations/apiv1")
}

func isGCPStorageBatchOperationsLocationRequest(r *http.Request, path string) bool {
	if !hasGCPStorageBatchOperationsHint(r) {
		return false
	}
	_, _, _, ok := parseGCPProjectLocationPath(path)
	return ok
}

func isGCPStorageBatchOperationsPath(path string, includeAmbiguous bool) bool {
	if !includeAmbiguous {
		return false
	}
	_, location, tail, ok := parseGCPStorageBatchOperationsLocationTail(path)
	if !ok || location != "global" {
		return false
	}
	if isGCPStorageBatchOperationsJobsCollectionTail(tail) ||
		isGCPStorageBatchOperationsJobTail(tail) ||
		isGCPStorageBatchOperationsJobActionTail(tail, "cancel") ||
		isGCPStorageBatchOperationsBucketOperationsCollectionTail(tail) ||
		isGCPStorageBatchOperationsBucketOperationTail(tail) ||
		isGCPStorageBatchOperationsOperationsCollectionTail(tail) ||
		isGCPStorageBatchOperationsOperationTail(tail) ||
		isGCPStorageBatchOperationsOperationActionTail(tail, "cancel") {
		return true
	}
	return false
}

func handleGCPStorageBatchOperationsListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPStorageBatchOperationsPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpStorageBatchOperationsLocation(project, "global"),
	}
	return respondGCPStorageBatchOperationsList(w, "locations", items, pageSize, start, path, false)
}

func handleGCPStorageBatchOperationsGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	if location != "global" {
		respondGCPStorageBatchOperationsNotFound(w, path, "location not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpStorageBatchOperationsLocation(project, location))
	return true
}

func handleGCPStorageBatchOperationsListJobs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, tail, ok := parseGCPStorageBatchOperationsLocationTail(path)
	if !ok || !isGCPStorageBatchOperationsJobsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPStorageBatchOperationsPagination(w, r, path, 1000)
	if !valid {
		return true
	}

	orderBy := normalizeGCPStorageBatchOperationsOrderBy(r.URL.Query().Get("orderBy"))
	if !isGCPStorageBatchOperationsOrderBy(orderBy) {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "orderBy must be one of name, create_time, create_time asc, create_time desc")
		return true
	}

	filterState, filterValid := parseGCPStorageBatchOperationsStateFilter(r.URL.Query().Get("filter"))
	if !filterValid {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "filter must be state=<RUNNING|SUCCEEDED|CANCELED|FAILED|QUEUED>")
		return true
	}

	items := []map[string]any{
		gcpStorageBatchOperationsJob(project, "job-1", "RUNNING"),
		gcpStorageBatchOperationsJob(project, "job-succeeded", "SUCCEEDED"),
		gcpStorageBatchOperationsJob(project, "job-canceled", "CANCELED"),
	}
	if filterState != "" {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if strings.EqualFold(gcpStorageBatchOperationsString(item, "state"), filterState) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	items = sortGCPStorageBatchOperationsItems(items, orderBy)

	return respondGCPStorageBatchOperationsList(w, "jobs", items, pageSize, start, path, true)
}

func handleGCPStorageBatchOperationsGetJob(w http.ResponseWriter, path string) bool {
	project, _, jobID, ok := parseGCPStorageBatchOperationsJobPath(path)
	if !ok {
		return false
	}
	if !isGCPStorageBatchOperationsJobID(jobID) {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "job name is invalid")
		return true
	}
	if isGCPStorageBatchOperationsMissingID(jobID) {
		respondGCPStorageBatchOperationsNotFound(w, path, "job not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpStorageBatchOperationsJob(project, jobID, gcpStorageBatchOperationsStateForJobID(jobID)))
	return true
}

func handleGCPStorageBatchOperationsCreateJob(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, tail, ok := parseGCPStorageBatchOperationsLocationTail(path)
	if !ok || !isGCPStorageBatchOperationsJobsCollectionTail(tail) {
		return false
	}

	jobID := strings.TrimSpace(r.URL.Query().Get("jobId"))
	if jobID == "" {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "jobId is required")
		return true
	}
	if !isGCPStorageBatchOperationsJobID(jobID) {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "jobId is invalid")
		return true
	}
	if isGCPStorageBatchOperationsMissingID(jobID) || strings.Contains(strings.ToLower(jobID), "existing") {
		respondGCPStorageBatchOperationsAlreadyExists(w, path, "job already exists")
		return true
	}

	requestID := strings.TrimSpace(r.URL.Query().Get("requestId"))
	if requestID != "" && !isGCPStorageBatchOperationsRequestID(requestID) {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "requestId must be a valid non-zero UUID")
		return true
	}

	body, valid := decodeGCPStorageBatchOperationsJSONBody(w, r, path)
	if !valid {
		return true
	}
	job := gcpStorageBatchOperationsBodyMap(body, "job")
	if len(job) == 0 {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "job is required")
		return true
	}
	if !validateGCPStorageBatchOperationsJob(w, path, job) {
		return true
	}

	if providedName := gcpStorageBatchOperationsString(job, "name"); providedName != "" {
		p, location, id, parsed := parseGCPStorageBatchOperationsJobName(providedName)
		if !parsed || location != "global" || p != project || id != jobID {
			respondGCPStorageBatchOperationsInvalidArgument(w, path, "job.name must match parent and jobId")
			return true
		}
	}

	created := gcpStorageBatchOperationsJobFromRequest(project, jobID, "RUNNING", job)
	operationID := "createJob." + jobID
	respondJSON(w, http.StatusOK, gcpStorageBatchOperationsOperation(project, operationID, created, false, false))
	return true
}

func handleGCPStorageBatchOperationsDeleteJob(w http.ResponseWriter, r *http.Request, path string) bool {
	_, _, jobID, ok := parseGCPStorageBatchOperationsJobPath(path)
	if !ok {
		return false
	}
	if !isGCPStorageBatchOperationsJobID(jobID) {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "job name is invalid")
		return true
	}
	if isGCPStorageBatchOperationsMissingID(jobID) {
		respondGCPStorageBatchOperationsNotFound(w, path, "job not found")
		return true
	}

	requestID := strings.TrimSpace(r.URL.Query().Get("requestId"))
	if requestID != "" && !isGCPStorageBatchOperationsRequestID(requestID) {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "requestId must be a valid non-zero UUID")
		return true
	}
	if forceRaw := strings.TrimSpace(r.URL.Query().Get("force")); forceRaw != "" {
		if _, err := strconv.ParseBool(forceRaw); err != nil {
			respondGCPStorageBatchOperationsInvalidArgument(w, path, "force must be a boolean")
			return true
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPStorageBatchOperationsCancelJob(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, jobID, ok := parseGCPStorageBatchOperationsJobActionPath(path, "cancel")
	if !ok {
		return false
	}
	if !isGCPStorageBatchOperationsJobID(jobID) {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "job name is invalid")
		return true
	}
	if isGCPStorageBatchOperationsMissingID(jobID) {
		respondGCPStorageBatchOperationsNotFound(w, path, "job not found")
		return true
	}

	body, valid := decodeGCPStorageBatchOperationsJSONBody(w, r, path)
	if !valid {
		return true
	}
	if name := gcpStorageBatchOperationsString(body, "name"); name != "" {
		p, location, id, parsed := parseGCPStorageBatchOperationsJobName(name)
		if !parsed || location != "global" || p != project || id != jobID {
			respondGCPStorageBatchOperationsInvalidArgument(w, path, "name must match requested job")
			return true
		}
	}
	if requestID := gcpStorageBatchOperationsString(body, "requestId"); requestID != "" && !isGCPStorageBatchOperationsRequestID(requestID) {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "requestId must be a valid non-zero UUID")
		return true
	}

	state := gcpStorageBatchOperationsStateForJobID(jobID)
	if state == "SUCCEEDED" || state == "FAILED" || state == "CANCELED" {
		respondGCPStorageBatchOperationsFailedPrecondition(w, path, "job is already in terminal state")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPStorageBatchOperationsListBucketOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, jobID, tail, ok := parseGCPStorageBatchOperationsBucketOperationsCollectionPath(path)
	if !ok || !isGCPStorageBatchOperationsBucketOperationsCollectionTail(tail) {
		return false
	}
	if !isGCPStorageBatchOperationsJobID(jobID) {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "parent is invalid")
		return true
	}
	pageSize, start, valid := parseGCPStorageBatchOperationsPagination(w, r, path, 1000)
	if !valid {
		return true
	}

	orderBy := normalizeGCPStorageBatchOperationsOrderBy(r.URL.Query().Get("orderBy"))
	if !isGCPStorageBatchOperationsOrderBy(orderBy) {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "orderBy must be one of name, create_time, create_time asc, create_time desc")
		return true
	}

	filterState, filterValid := parseGCPStorageBatchOperationsStateFilter(r.URL.Query().Get("filter"))
	if !filterValid {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "filter must be state=<RUNNING|SUCCEEDED|CANCELED|FAILED|QUEUED>")
		return true
	}

	items := []map[string]any{
		gcpStorageBatchOperationsBucketOperation(project, jobID, "bucket-op-1", "RUNNING"),
		gcpStorageBatchOperationsBucketOperation(project, jobID, "bucket-op-2", "SUCCEEDED"),
	}
	if filterState != "" {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if strings.EqualFold(gcpStorageBatchOperationsString(item, "state"), filterState) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	items = sortGCPStorageBatchOperationsItems(items, orderBy)

	return respondGCPStorageBatchOperationsList(w, "bucketOperations", items, pageSize, start, path, true)
}

func handleGCPStorageBatchOperationsGetBucketOperation(w http.ResponseWriter, path string) bool {
	project, _, jobID, bucketOperationID, ok := parseGCPStorageBatchOperationsBucketOperationPath(path)
	if !ok {
		return false
	}
	if !isGCPStorageBatchOperationsJobID(jobID) {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "name is invalid")
		return true
	}
	if isGCPStorageBatchOperationsMissingID(jobID) || isGCPStorageBatchOperationsMissingID(bucketOperationID) {
		respondGCPStorageBatchOperationsNotFound(w, path, "bucket operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, gcpStorageBatchOperationsBucketOperation(project, jobID, bucketOperationID, "SUCCEEDED"))
	return true
}

func handleGCPStorageBatchOperationsListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, tail, ok := parseGCPStorageBatchOperationsLocationTail(path)
	if !ok || !isGCPStorageBatchOperationsOperationsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPStorageBatchOperationsPagination(w, r, path, 1000)
	if !valid {
		return true
	}
	wantDoneSet, wantDone, filterValid := parseGCPStorageBatchOperationsDoneFilter(r.URL.Query().Get("filter"))
	if !filterValid {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "filter must be done=true or done=false")
		return true
	}
	if returnPartialRaw := strings.TrimSpace(r.URL.Query().Get("returnPartialSuccess")); returnPartialRaw != "" {
		if _, err := strconv.ParseBool(returnPartialRaw); err != nil {
			respondGCPStorageBatchOperationsInvalidArgument(w, path, "returnPartialSuccess must be a boolean")
			return true
		}
	}

	items := []map[string]any{
		gcpStorageBatchOperationsOperation(project, "createJob.job-1", gcpStorageBatchOperationsJob(project, "job-1", "RUNNING"), false, false),
		gcpStorageBatchOperationsOperation(project, "createJob.job-succeeded", gcpStorageBatchOperationsJob(project, "job-succeeded", "SUCCEEDED"), true, false),
	}
	if wantDoneSet {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			done, _ := item["done"].(bool)
			if done == wantDone {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	sort.Slice(items, func(i, j int) bool {
		return gcpStorageBatchOperationsString(items[i], "name") < gcpStorageBatchOperationsString(items[j], "name")
	})

	return respondGCPStorageBatchOperationsList(w, "operations", items, pageSize, start, path, false)
}

func handleGCPStorageBatchOperationsGetOperation(w http.ResponseWriter, path string) bool {
	project, _, operationID, ok := parseGCPStorageBatchOperationsOperationPath(path)
	if !ok {
		return false
	}
	if isGCPStorageBatchOperationsMissingID(operationID) {
		respondGCPStorageBatchOperationsNotFound(w, path, "operation not found")
		return true
	}
	jobID := gcpStorageBatchOperationsJobIDFromOperationID(operationID)
	done := !strings.Contains(strings.ToLower(operationID), "running")
	respondJSON(w, http.StatusOK, gcpStorageBatchOperationsOperation(project, operationID, gcpStorageBatchOperationsJob(project, jobID, gcpStorageBatchOperationsStateForJobID(jobID)), done, strings.Contains(strings.ToLower(operationID), "cancel")))
	return true
}

func handleGCPStorageBatchOperationsCancelOperation(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, operationID, ok := parseGCPStorageBatchOperationsOperationActionPath(path, "cancel")
	if !ok {
		return false
	}
	if isGCPStorageBatchOperationsMissingID(operationID) {
		respondGCPStorageBatchOperationsNotFound(w, path, "operation not found")
		return true
	}

	body, valid := decodeGCPStorageBatchOperationsJSONBody(w, r, path)
	if !valid {
		return true
	}
	if name := gcpStorageBatchOperationsString(body, "name"); name != "" {
		p, location, id, parsed := parseGCPStorageBatchOperationsOperationName(name)
		if !parsed || location != "global" || p != project || id != operationID {
			respondGCPStorageBatchOperationsInvalidArgument(w, path, "name must match requested operation")
			return true
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPStorageBatchOperationsDeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, operationID, ok := parseGCPStorageBatchOperationsOperationPath(path)
	if !ok {
		return false
	}
	if isGCPStorageBatchOperationsMissingID(operationID) {
		respondGCPStorageBatchOperationsNotFound(w, path, "operation not found")
		return true
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func parseGCPStorageBatchOperationsLocationTail(path string) (project, location string, tail []string, ok bool) {
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

func isGCPStorageBatchOperationsJobsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "jobs"
}

func isGCPStorageBatchOperationsJobTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "jobs" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPStorageBatchOperationsJobActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "jobs" {
		return false
	}
	jobID, parsedAction, found := splitGCPStorageBatchOperationsActionSegment(tail[1])
	return found && strings.TrimSpace(jobID) != "" && parsedAction == action
}

func isGCPStorageBatchOperationsBucketOperationsCollectionTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "jobs" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":") && tail[2] == "bucketOperations"
}

func isGCPStorageBatchOperationsBucketOperationTail(tail []string) bool {
	return len(tail) == 4 && tail[0] == "jobs" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":") && tail[2] == "bucketOperations" && strings.TrimSpace(tail[3]) != "" && !strings.Contains(tail[3], ":")
}

func isGCPStorageBatchOperationsOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPStorageBatchOperationsOperationTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPStorageBatchOperationsOperationActionTail(tail []string, action string) bool {
	if len(tail) != 2 || tail[0] != "operations" {
		return false
	}
	operationID, parsedAction, found := splitGCPStorageBatchOperationsActionSegment(tail[1])
	return found && strings.TrimSpace(operationID) != "" && parsedAction == action
}

func splitGCPStorageBatchOperationsActionSegment(segment string) (resourceID, action string, ok bool) {
	segment = strings.TrimSpace(segment)
	resourceID, action, ok = strings.Cut(segment, ":")
	if !ok {
		return "", "", false
	}
	resourceID = strings.TrimSpace(resourceID)
	action = strings.TrimSpace(action)
	if resourceID == "" || action == "" {
		return "", "", false
	}
	return resourceID, action, true
}

func parseGCPStorageBatchOperationsJobPath(path string) (project, location, jobID string, ok bool) {
	project, location, tail, ok := parseGCPStorageBatchOperationsLocationTail(path)
	if !ok || !isGCPStorageBatchOperationsJobTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPStorageBatchOperationsJobActionPath(path, action string) (project, location, jobID string, ok bool) {
	project, location, tail, ok := parseGCPStorageBatchOperationsLocationTail(path)
	if !ok || !isGCPStorageBatchOperationsJobActionTail(tail, action) {
		return "", "", "", false
	}
	jobID, _, _ = splitGCPStorageBatchOperationsActionSegment(tail[1])
	return project, location, jobID, true
}

func parseGCPStorageBatchOperationsBucketOperationsCollectionPath(path string) (project, location, jobID string, tail []string, ok bool) {
	project, location, tail, ok = parseGCPStorageBatchOperationsLocationTail(path)
	if !ok || !isGCPStorageBatchOperationsBucketOperationsCollectionTail(tail) {
		return "", "", "", nil, false
	}
	return project, location, strings.TrimSpace(tail[1]), tail, true
}

func parseGCPStorageBatchOperationsBucketOperationPath(path string) (project, location, jobID, bucketOperationID string, ok bool) {
	project, location, tail, ok := parseGCPStorageBatchOperationsLocationTail(path)
	if !ok || !isGCPStorageBatchOperationsBucketOperationTail(tail) {
		return "", "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), strings.TrimSpace(tail[3]), true
}

func parseGCPStorageBatchOperationsOperationPath(path string) (project, location, operationID string, ok bool) {
	project, location, tail, ok := parseGCPStorageBatchOperationsLocationTail(path)
	if !ok || !isGCPStorageBatchOperationsOperationTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPStorageBatchOperationsOperationActionPath(path, action string) (project, location, operationID string, ok bool) {
	project, location, tail, ok := parseGCPStorageBatchOperationsLocationTail(path)
	if !ok || !isGCPStorageBatchOperationsOperationActionTail(tail, action) {
		return "", "", "", false
	}
	operationID, _, _ = splitGCPStorageBatchOperationsActionSegment(tail[1])
	return project, location, operationID, true
}

func parseGCPStorageBatchOperationsParent(parent string) (project, location string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 4 || parts[0] != "projects" || parts[2] != "locations" {
		return "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	if project == "" || location == "" {
		return "", "", false
	}
	return project, location, true
}

func parseGCPStorageBatchOperationsJobName(name string) (project, location, jobID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "jobs" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	jobID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || jobID == "" {
		return "", "", "", false
	}
	return project, location, jobID, true
}

func parseGCPStorageBatchOperationsOperationName(name string) (project, location, operationID string, ok bool) {
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

func parseGCPStorageBatchOperationsBucketOperationName(name string) (project, location, jobID, bucketOperationID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "jobs" || parts[6] != "bucketOperations" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	jobID = strings.TrimSpace(parts[5])
	bucketOperationID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || jobID == "" || bucketOperationID == "" {
		return "", "", "", "", false
	}
	return project, location, jobID, bucketOperationID, true
}

func parseGCPStorageBatchOperationsPagination(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize > maxPageSize {
		respondGCPStorageBatchOperationsOutOfRange(w, path, fmt.Sprintf("pageSize cannot exceed %d", maxPageSize))
		return 0, 0, false
	}
	start = 0
	pageToken := strings.TrimSpace(r.URL.Query().Get("pageToken"))
	if pageToken != "" {
		start, err = strconv.Atoi(pageToken)
		if err != nil || start < 0 {
			respondGCPStorageBatchOperationsInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func parseGCPStorageBatchOperationsStateFilter(raw string) (state string, ok bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", true
	}
	normalized := strings.ToLower(strings.ReplaceAll(value, " ", ""))
	switch {
	case strings.HasPrefix(normalized, "state="):
		state = strings.ToUpper(strings.TrimSpace(normalized[len("state="):]))
	case strings.HasPrefix(normalized, "state:"):
		state = strings.ToUpper(strings.TrimSpace(normalized[len("state:"):]))
	default:
		return "", false
	}
	switch state {
	case "RUNNING", "SUCCEEDED", "CANCELED", "FAILED", "QUEUED":
		return state, true
	default:
		return "", false
	}
}

func parseGCPStorageBatchOperationsDoneFilter(raw string) (set, done, ok bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false, false, true
	}
	normalized := strings.ToLower(strings.ReplaceAll(value, " ", ""))
	switch {
	case strings.HasPrefix(normalized, "done="):
		normalized = normalized[len("done="):]
	case strings.HasPrefix(normalized, "done:"):
		normalized = normalized[len("done:"):]
	default:
		return false, false, false
	}
	switch normalized {
	case "true":
		return true, true, true
	case "false":
		return true, false, true
	default:
		return false, false, false
	}
}

func normalizeGCPStorageBatchOperationsOrderBy(raw string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(raw))), " ")
}

func isGCPStorageBatchOperationsOrderBy(orderBy string) bool {
	switch orderBy {
	case "", "name", "create_time", "create_time asc", "create_time desc":
		return true
	default:
		return false
	}
}

func sortGCPStorageBatchOperationsItems(items []map[string]any, orderBy string) []map[string]any {
	if len(items) <= 1 {
		return items
	}
	sorted := append([]map[string]any(nil), items...)
	switch orderBy {
	case "name":
		sort.Slice(sorted, func(i, j int) bool {
			return gcpStorageBatchOperationsString(sorted[i], "name") < gcpStorageBatchOperationsString(sorted[j], "name")
		})
	case "create_time":
		sort.Slice(sorted, func(i, j int) bool {
			return gcpStorageBatchOperationsString(sorted[i], "createTime") < gcpStorageBatchOperationsString(sorted[j], "createTime")
		})
	case "create_time desc":
		sort.Slice(sorted, func(i, j int) bool {
			return gcpStorageBatchOperationsString(sorted[i], "createTime") > gcpStorageBatchOperationsString(sorted[j], "createTime")
		})
	case "create_time asc":
		sort.Slice(sorted, func(i, j int) bool {
			return gcpStorageBatchOperationsString(sorted[i], "createTime") < gcpStorageBatchOperationsString(sorted[j], "createTime")
		})
	}
	return sorted
}

func respondGCPStorageBatchOperationsList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string, includeUnreachable bool) bool {
	if start > len(items) {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "pageToken is out of range")
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
	resp := map[string]any{
		key:             items[start:end],
		"nextPageToken": nextPageToken,
	}
	if includeUnreachable {
		resp["unreachable"] = []any{}
	}
	respondJSON(w, http.StatusOK, resp)
	return true
}

func decodeGCPStorageBatchOperationsJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	payload, err := io.ReadAll(io.LimitReader(http.MaxBytesReader(w, r.Body, 1<<20), 1<<20))
	if err != nil {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "request body must be readable")
		return nil, false
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpStorageBatchOperationsBodyMap(body map[string]any, key string) map[string]any {
	if nested, ok := body[key].(map[string]any); ok && len(nested) > 0 {
		return nested
	}
	return body
}

func gcpStorageBatchOperationsString(body map[string]any, key string) string {
	value, _ := body[key].(string)
	return strings.TrimSpace(value)
}

func validateGCPStorageBatchOperationsJob(w http.ResponseWriter, path string, job map[string]any) bool {
	sourceKeys := []string{"bucketList"}
	sourceCount := 0
	for _, key := range sourceKeys {
		if _, ok := job[key]; ok {
			sourceCount++
		}
	}
	if sourceCount == 0 {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "job.bucketList is required")
		return false
	}
	if sourceCount > 1 {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "job source oneof is invalid")
		return false
	}

	bucketList, ok := job["bucketList"].(map[string]any)
	if !ok || len(bucketList) == 0 {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "job.bucketList must be an object")
		return false
	}
	rawBuckets, ok := bucketList["buckets"].([]any)
	if !ok || len(rawBuckets) == 0 {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "job.bucketList.buckets must include at least one bucket")
		return false
	}
	for idx, rawBucket := range rawBuckets {
		bucket, ok := rawBucket.(map[string]any)
		if !ok {
			respondGCPStorageBatchOperationsInvalidArgument(w, path, fmt.Sprintf("job.bucketList.buckets[%d] must be an object", idx))
			return false
		}
		bucketName := gcpStorageBatchOperationsString(bucket, "bucket")
		if bucketName == "" {
			respondGCPStorageBatchOperationsInvalidArgument(w, path, fmt.Sprintf("job.bucketList.buckets[%d].bucket is required", idx))
			return false
		}
	}

	transformationKeys := []string{"putObjectHold", "deleteObject", "putMetadata", "rewriteObject", "updateObjectCustomContext"}
	transformationCount := 0
	unsupportedTransformation := false
	for _, key := range transformationKeys {
		if _, ok := job[key]; ok {
			transformationCount++
			if key == "updateObjectCustomContext" {
				unsupportedTransformation = true
			}
		}
	}
	if transformationCount == 0 {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "job transformation is required")
		return false
	}
	if transformationCount > 1 {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "job transformation oneof is invalid")
		return false
	}
	if unsupportedTransformation {
		respondGCPStorageBatchOperationsInvalidArgument(w, path, "job.updateObjectCustomContext is not supported in staged emulation")
		return false
	}
	return true
}

func isGCPStorageBatchOperationsJobID(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && gcpStorageBatchOperationsJobIDPattern.MatchString(value)
}

func isGCPStorageBatchOperationsRequestID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || !gcpStorageBatchOperationsRequestIDPattern.MatchString(value) {
		return false
	}
	return strings.ToLower(value) != "00000000-0000-0000-0000-000000000000"
}

func isGCPStorageBatchOperationsMissingID(value string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(value)), "missing")
}

func gcpStorageBatchOperationsStateForJobID(jobID string) string {
	normalized := strings.ToLower(strings.TrimSpace(jobID))
	switch {
	case strings.Contains(normalized, "succeeded"):
		return "SUCCEEDED"
	case strings.Contains(normalized, "failed"):
		return "FAILED"
	case strings.Contains(normalized, "canceled"), strings.Contains(normalized, "cancelled"):
		return "CANCELED"
	case strings.Contains(normalized, "queued"):
		return "QUEUED"
	default:
		return "RUNNING"
	}
}

func gcpStorageBatchOperationsJobIDFromOperationID(operationID string) string {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return "job-1"
	}
	if _, after, ok := strings.Cut(operationID, "."); ok && isGCPStorageBatchOperationsJobID(after) {
		return after
	}
	return "job-1"
}

func gcpStorageBatchOperationsLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": "Storage Batch Operations " + location,
		"labels": map[string]string{
			"service":  "storagebatchoperations",
			"provider": providerGCP,
		},
		"metadata": map[string]any{
			"stagedEmulation": true,
		},
	}
}

func gcpStorageBatchOperationsJob(project, jobID, state string) map[string]any {
	offsetMinutes := (len(jobID) % 5) + 1
	createTime := gcpStorageBatchOperationsReferenceTime.Add(time.Duration(offsetMinutes) * time.Minute)
	scheduleTime := createTime.Add(30 * time.Second)
	completeTime := scheduleTime.Add(2 * time.Minute)

	counters := map[string]any{
		"totalObjectCount":     120,
		"succeededObjectCount": 116,
		"failedObjectCount":    4,
		"totalBytesFound":      104857600,
	}
	errorSummaries := []any{}
	switch state {
	case "RUNNING", "QUEUED":
		counters["succeededObjectCount"] = 44
		counters["failedObjectCount"] = 1
	case "CANCELED":
		counters["succeededObjectCount"] = 57
		counters["failedObjectCount"] = 2
	case "FAILED":
		counters["succeededObjectCount"] = 12
		counters["failedObjectCount"] = 108
		errorSummaries = []any{
			map[string]any{
				"errorCode":  13,
				"errorCount": 3,
				"errorLogEntries": []any{
					map[string]any{
						"objectUri":    "gs://stackyard-source-bucket/incoming/broken-object.txt",
						"errorDetails": []any{"staged emulation failure"},
					},
				},
			},
		}
	}

	job := map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/global/jobs/%s", project, jobID),
		"description": "Stackyard Storage Batch Operations job " + jobID,
		"bucketList": map[string]any{
			"buckets": []any{
				map[string]any{
					"bucket": "stackyard-source-bucket",
					"prefixList": map[string]any{
						"includedObjectPrefixes": []any{"incoming/"},
					},
				},
			},
		},
		"deleteObject": map[string]any{
			"permanentObjectDeletionEnabled": true,
		},
		"createTime":       createTime.Format(time.RFC3339Nano),
		"scheduleTime":     scheduleTime.Format(time.RFC3339Nano),
		"counters":         counters,
		"errorSummaries":   errorSummaries,
		"state":            state,
		"dryRun":           false,
		"isMultiBucketJob": false,
	}
	if state != "RUNNING" && state != "QUEUED" {
		job["completeTime"] = completeTime.Format(time.RFC3339Nano)
	}
	return job
}

func gcpStorageBatchOperationsJobFromRequest(project, jobID, state string, req map[string]any) map[string]any {
	job := gcpStorageBatchOperationsJob(project, jobID, state)
	if description := gcpStorageBatchOperationsString(req, "description"); description != "" {
		job["description"] = description
	}
	mergeKeys := []string{
		"bucketList",
		"putObjectHold",
		"deleteObject",
		"putMetadata",
		"rewriteObject",
		"loggingConfig",
		"dryRun",
	}
	for _, key := range mergeKeys {
		if value, ok := req[key]; ok {
			job[key] = value
		}
	}
	if _, hasDelete := req["deleteObject"]; !hasDelete {
		delete(job, "deleteObject")
	}
	return job
}

func gcpStorageBatchOperationsBucketOperation(project, jobID, bucketOperationID, state string) map[string]any {
	createTime := gcpStorageBatchOperationsReferenceTime.Add(3 * time.Minute)
	startTime := createTime.Add(15 * time.Second)
	completeTime := startTime.Add(45 * time.Second)

	item := map[string]any{
		"name":       fmt.Sprintf("projects/%s/locations/global/jobs/%s/bucketOperations/%s", project, jobID, bucketOperationID),
		"bucketName": "stackyard-source-bucket",
		"prefixList": map[string]any{
			"includedObjectPrefixes": []any{"incoming/"},
		},
		"deleteObject": map[string]any{
			"permanentObjectDeletionEnabled": true,
		},
		"createTime": createTime.Format(time.RFC3339Nano),
		"startTime":  startTime.Format(time.RFC3339Nano),
		"counters": map[string]any{
			"totalObjectCount":     60,
			"succeededObjectCount": 58,
			"failedObjectCount":    2,
		},
		"errorSummaries": []any{},
		"state":          state,
	}
	if state != "RUNNING" && state != "QUEUED" {
		item["completeTime"] = completeTime.Format(time.RFC3339Nano)
	}
	return item
}

func gcpStorageBatchOperationsOperation(project, operationID string, job map[string]any, done, requestedCancellation bool) map[string]any {
	operationName := fmt.Sprintf("projects/%s/locations/global/operations/%s", project, operationID)
	metadata := map[string]any{
		"@type":                 "type.googleapis.com/google.cloud.storagebatchoperations.v1.OperationMetadata",
		"operation":             operationName,
		"createTime":            gcpStorageBatchOperationsReferenceTime.Format(time.RFC3339Nano),
		"apiVersion":            "v1",
		"requestedCancellation": requestedCancellation,
		"job":                   job,
	}
	if done {
		metadata["endTime"] = gcpStorageBatchOperationsReferenceTime.Add(2 * time.Minute).Format(time.RFC3339Nano)
	}
	operation := map[string]any{
		"name":     operationName,
		"done":     done,
		"metadata": metadata,
	}
	if done {
		response := map[string]any{
			"@type": "type.googleapis.com/google.cloud.storagebatchoperations.v1.Job",
		}
		for key, value := range job {
			response[key] = value
		}
		operation["response"] = response
	}
	return operation
}

func respondGCPStorageBatchOperationsInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPStorageBatchOperationsError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPStorageBatchOperationsNotFound(w http.ResponseWriter, path, message string) {
	respondGCPStorageBatchOperationsError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPStorageBatchOperationsAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPStorageBatchOperationsError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPStorageBatchOperationsFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPStorageBatchOperationsError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPStorageBatchOperationsOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPStorageBatchOperationsError(w, http.StatusBadRequest, "OutOfRange", path, message)
}

func respondGCPStorageBatchOperationsError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_storagebatchoperations(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "storagebatchoperations") {
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
			"name":     "projects/stackyard/locations/global/storagebatchoperations/sample",
			"service":  "storagebatchoperations",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
