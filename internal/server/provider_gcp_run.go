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

var gcpRunReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func (s *Server) handleGCPRunRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_run(w, r) {
		return true
	}

	path := normalizeGCPRunPath(rawRequestPath(r))
	if isGCPRunLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPRunListLocations(w, r, path) {
			return true
		}
		if handleGCPRunGetLocation(w, path) {
			return true
		}
		return false
	}

	if !isGCPRunPath(path, hasGCPRunHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPRunListServices(w, r, path) {
			return true
		}
		if handleGCPRunGetService(w, path) {
			return true
		}
		if handleGCPRunListJobs(w, r, path) {
			return true
		}
		if handleGCPRunGetJob(w, path) {
			return true
		}
		if handleGCPRunListExecutions(w, r, path) {
			return true
		}
		if handleGCPRunGetExecution(w, path) {
			return true
		}
		if handleGCPRunListTasks(w, r, path) {
			return true
		}
		if handleGCPRunGetTask(w, path) {
			return true
		}
		if handleGCPRunListRevisions(w, r, path) {
			return true
		}
		if handleGCPRunGetRevision(w, path) {
			return true
		}
		if handleGCPRunListOperations(w, r, path) {
			return true
		}
		if handleGCPRunGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPRunCreateService(w, r, path) {
			return true
		}
		if handleGCPRunCreateJob(w, r, path) {
			return true
		}
		if handleGCPRunRunJob(w, r, path) {
			return true
		}
		if handleGCPRunCancelExecution(w, r, path) {
			return true
		}
		if handleGCPRunOperationAction(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPRunUpdateService(w, r, path) {
			return true
		}
		if handleGCPRunUpdateJob(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPRunDeleteService(w, path, r.URL.Query().Get("etag")) {
			return true
		}
		if handleGCPRunDeleteJob(w, path, r.URL.Query().Get("etag")) {
			return true
		}
		if handleGCPRunDeleteRevision(w, path, r.URL.Query().Get("etag")) {
			return true
		}
		if handleGCPRunDeleteExecution(w, path, r.URL.Query().Get("etag")) {
			return true
		}
		if handleGCPRunDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func hasGCPRunHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "run", "run-apiv2", "cloud-run-admin", "cloud_run_admin", "cloudrunadmin":
		return true
	}
	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-run-apiv2") || strings.Contains(ua, "cloud.google.com/go/run")
}

func normalizeGCPRunPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func isGCPRunLocationRequest(r *http.Request, path string) bool {
	if !hasGCPRunHint(r) {
		return false
	}
	_, _, _, ok := parseGCPRunProjectLocationPath(path)
	return ok
}

func isGCPRunPath(path string, includeOperations bool) bool {
	_, _, tail, ok := parseGCPRunLocationTail(path)
	if !ok || len(tail) == 0 {
		return false
	}
	if isGCPRunServicesCollectionTail(tail) ||
		isGCPRunServiceTail(tail) ||
		isGCPRunJobsCollectionTail(tail) ||
		isGCPRunJobTail(tail) ||
		isGCPRunJobRunActionTail(tail) ||
		isGCPRunExecutionsCollectionTail(tail) ||
		isGCPRunExecutionTail(tail) ||
		isGCPRunExecutionCancelActionTail(tail) ||
		isGCPRunTasksCollectionTail(tail) ||
		isGCPRunTaskTail(tail) ||
		isGCPRunRevisionsCollectionTail(tail) ||
		isGCPRunRevisionTail(tail) {
		return true
	}
	if !includeOperations {
		return false
	}
	return isGCPRunOperationsCollectionTail(tail) ||
		isGCPRunOperationTail(tail) ||
		isGCPRunOperationActionTail(tail, "cancel") ||
		isGCPRunOperationActionTail(tail, "wait") ||
		isGCPRunNestedOperationsCollectionTail(tail)
}

func handleGCPRunListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPRunProjectLocationPath(path)
	if !ok || !list {
		return false
	}
	pageSize, start, valid := parseGCPRunPagination(w, r, path)
	if !valid {
		return true
	}
	items := []map[string]any{
		gcpRunLocation(project, "us-central1"),
		gcpRunLocation(project, "global"),
	}
	return respondGCPRunList(w, "locations", items, pageSize, start, path)
}

func handleGCPRunGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPRunProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRunLocation(project, location))
	return true
}

func handleGCPRunListServices(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || !isGCPRunServicesCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPRunPagination(w, r, path)
	if !valid {
		return true
	}
	showDeleted, err := parseGCPRunOptionalBool(r.URL.Query().Get("showDeleted"))
	if err != nil {
		respondGCPRunInvalidArgument(w, path, "showDeleted must be a boolean")
		return true
	}
	items := []map[string]any{
		gcpRunService(project, location, "service-1"),
		gcpRunService(project, location, "service-2"),
	}
	if !showDeleted {
		items = items[:1]
	}
	return respondGCPRunList(w, "services", items, pageSize, start, path)
}

func handleGCPRunGetService(w http.ResponseWriter, path string) bool {
	project, location, serviceID, ok := parseGCPRunServicePath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRunService(project, location, serviceID))
	return true
}

func handleGCPRunCreateService(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || !isGCPRunServicesCollectionTail(tail) {
		return false
	}
	serviceID := strings.TrimSpace(r.URL.Query().Get("serviceId"))
	if serviceID == "" {
		respondGCPRunInvalidArgument(w, path, "serviceId is required")
		return true
	}
	if !isGCPRunResourceID(serviceID) {
		respondGCPRunInvalidArgument(w, path, "serviceId is invalid")
		return true
	}
	service, valid := decodeGCPRunJSONBody(w, r, path)
	if !valid {
		return true
	}
	if len(service) == 0 {
		respondGCPRunInvalidArgument(w, path, "service is required")
		return true
	}
	if !hasGCPRunRevisionTemplate(service) {
		respondGCPRunInvalidArgument(w, path, "service.template.containers is required")
		return true
	}
	expectedName := gcpRunServiceName(project, location, serviceID)
	if name := gcpRunString(service, "name"); name != "" && name != expectedName {
		respondGCPRunInvalidArgument(w, path, "service.name must match parent and serviceId")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRunOperation(project, location, "createService."+serviceID, expectedName, "create", false))
	return true
}

func handleGCPRunUpdateService(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, serviceID, ok := parseGCPRunServicePath(path)
	if !ok {
		return false
	}
	service, valid := decodeGCPRunJSONBody(w, r, path)
	if !valid {
		return true
	}
	if len(service) == 0 {
		respondGCPRunInvalidArgument(w, path, "service is required")
		return true
	}
	expectedName := gcpRunServiceName(project, location, serviceID)
	if name := gcpRunString(service, "name"); name == "" || name != expectedName {
		respondGCPRunInvalidArgument(w, path, "service.name must match the requested resource")
		return true
	}
	if strings.TrimSpace(r.URL.Query().Get("updateMask")) == "" {
		respondGCPRunInvalidArgument(w, path, "updateMask is required")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRunOperation(project, location, "updateService."+serviceID, expectedName, "update", false))
	return true
}

func handleGCPRunDeleteService(w http.ResponseWriter, path, etag string) bool {
	project, location, serviceID, ok := parseGCPRunServicePath(path)
	if !ok {
		return false
	}
	if etag = strings.TrimSpace(etag); etag != "" && etag != gcpRunEtag(serviceID) {
		respondGCPRunFailedPrecondition(w, path, "etag mismatch")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRunOperation(project, location, "deleteService."+serviceID, gcpRunServiceName(project, location, serviceID), "delete", false))
	return true
}

func handleGCPRunListJobs(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || !isGCPRunJobsCollectionTail(tail) {
		return false
	}
	pageSize, start, valid := parseGCPRunPagination(w, r, path)
	if !valid {
		return true
	}
	showDeleted, err := parseGCPRunOptionalBool(r.URL.Query().Get("showDeleted"))
	if err != nil {
		respondGCPRunInvalidArgument(w, path, "showDeleted must be a boolean")
		return true
	}
	items := []map[string]any{
		gcpRunJob(project, location, "job-1"),
		gcpRunJob(project, location, "job-2"),
	}
	if !showDeleted {
		items = items[:1]
	}
	return respondGCPRunList(w, "jobs", items, pageSize, start, path)
}

func handleGCPRunGetJob(w http.ResponseWriter, path string) bool {
	project, location, jobID, ok := parseGCPRunJobPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRunJob(project, location, jobID))
	return true
}

func handleGCPRunCreateJob(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || !isGCPRunJobsCollectionTail(tail) {
		return false
	}
	jobID := strings.TrimSpace(r.URL.Query().Get("jobId"))
	if jobID == "" {
		respondGCPRunInvalidArgument(w, path, "jobId is required")
		return true
	}
	if !isGCPRunResourceID(jobID) {
		respondGCPRunInvalidArgument(w, path, "jobId is invalid")
		return true
	}
	job, valid := decodeGCPRunJSONBody(w, r, path)
	if !valid {
		return true
	}
	if len(job) == 0 {
		respondGCPRunInvalidArgument(w, path, "job is required")
		return true
	}
	if !hasGCPRunExecutionTemplate(job) {
		respondGCPRunInvalidArgument(w, path, "job.template.template.containers is required")
		return true
	}
	expectedName := gcpRunJobName(project, location, jobID)
	if name := gcpRunString(job, "name"); name != "" && name != expectedName {
		respondGCPRunInvalidArgument(w, path, "job.name must match parent and jobId")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRunOperation(project, location, "createJob."+jobID, expectedName, "create", false))
	return true
}

func handleGCPRunUpdateJob(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, jobID, ok := parseGCPRunJobPath(path)
	if !ok {
		return false
	}
	job, valid := decodeGCPRunJSONBody(w, r, path)
	if !valid {
		return true
	}
	if len(job) == 0 {
		respondGCPRunInvalidArgument(w, path, "job is required")
		return true
	}
	expectedName := gcpRunJobName(project, location, jobID)
	if name := gcpRunString(job, "name"); name == "" || name != expectedName {
		respondGCPRunInvalidArgument(w, path, "job.name must match the requested resource")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRunOperation(project, location, "updateJob."+jobID, expectedName, "update", false))
	return true
}

func handleGCPRunDeleteJob(w http.ResponseWriter, path, etag string) bool {
	project, location, jobID, ok := parseGCPRunJobPath(path)
	if !ok {
		return false
	}
	if etag = strings.TrimSpace(etag); etag != "" && etag != gcpRunEtag(jobID) {
		respondGCPRunFailedPrecondition(w, path, "etag mismatch")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRunOperation(project, location, "deleteJob."+jobID, gcpRunJobName(project, location, jobID), "delete", false))
	return true
}

func handleGCPRunRunJob(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, jobID, ok := parseGCPRunJobRunPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPRunJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpRunJobName(project, location, jobID)
	if got := gcpRunString(body, "name"); got != "" && got != expectedName {
		respondGCPRunInvalidArgument(w, path, "name must match the requested resource")
		return true
	}
	if etag := gcpRunString(body, "etag"); etag != "" && etag != gcpRunEtag(jobID) {
		respondGCPRunFailedPrecondition(w, path, "etag mismatch")
		return true
	}
	executionName := gcpRunExecutionName(project, location, jobID, "execution-1")
	respondJSON(w, http.StatusOK, gcpRunOperation(project, location, "runJob."+jobID, executionName, "run", false))
	return true
}

func handleGCPRunListExecutions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, jobID, ok := parseGCPRunExecutionsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRunPagination(w, r, path)
	if !valid {
		return true
	}
	showDeleted, err := parseGCPRunOptionalBool(r.URL.Query().Get("showDeleted"))
	if err != nil {
		respondGCPRunInvalidArgument(w, path, "showDeleted must be a boolean")
		return true
	}
	items := []map[string]any{
		gcpRunExecution(project, location, jobID, "execution-1"),
		gcpRunExecution(project, location, jobID, "execution-2"),
	}
	if !showDeleted {
		items = items[:1]
	}
	return respondGCPRunList(w, "executions", items, pageSize, start, path)
}

func handleGCPRunGetExecution(w http.ResponseWriter, path string) bool {
	project, location, jobID, executionID, ok := parseGCPRunExecutionPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRunExecution(project, location, jobID, executionID))
	return true
}

func handleGCPRunDeleteExecution(w http.ResponseWriter, path, etag string) bool {
	project, location, jobID, executionID, ok := parseGCPRunExecutionPath(path)
	if !ok {
		return false
	}
	if etag = strings.TrimSpace(etag); etag != "" && etag != gcpRunEtag(executionID) {
		respondGCPRunFailedPrecondition(w, path, "etag mismatch")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRunOperation(project, location, "deleteExecution."+executionID, gcpRunExecutionName(project, location, jobID, executionID), "delete", false))
	return true
}

func handleGCPRunCancelExecution(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, jobID, executionID, ok := parseGCPRunExecutionCancelPath(path)
	if !ok {
		return false
	}
	body, valid := decodeGCPRunJSONBody(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpRunExecutionName(project, location, jobID, executionID)
	if got := gcpRunString(body, "name"); got != "" && got != expectedName {
		respondGCPRunInvalidArgument(w, path, "name must match the requested resource")
		return true
	}
	if etag := gcpRunString(body, "etag"); etag != "" && etag != gcpRunEtag(executionID) {
		respondGCPRunFailedPrecondition(w, path, "etag mismatch")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRunOperation(project, location, "cancelExecution."+executionID, expectedName, "cancel", false))
	return true
}

func handleGCPRunListTasks(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, jobID, executionID, ok := parseGCPRunTasksCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRunPagination(w, r, path)
	if !valid {
		return true
	}
	showDeleted, err := parseGCPRunOptionalBool(r.URL.Query().Get("showDeleted"))
	if err != nil {
		respondGCPRunInvalidArgument(w, path, "showDeleted must be a boolean")
		return true
	}
	items := []map[string]any{
		gcpRunTask(project, location, jobID, executionID, "task-1"),
		gcpRunTask(project, location, jobID, executionID, "task-2"),
	}
	if !showDeleted {
		items = items[:1]
	}
	return respondGCPRunList(w, "tasks", items, pageSize, start, path)
}

func handleGCPRunGetTask(w http.ResponseWriter, path string) bool {
	project, location, jobID, executionID, taskID, ok := parseGCPRunTaskPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRunTask(project, location, jobID, executionID, taskID))
	return true
}

func handleGCPRunListRevisions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, serviceID, ok := parseGCPRunRevisionsCollectionPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRunPagination(w, r, path)
	if !valid {
		return true
	}
	showDeleted, err := parseGCPRunOptionalBool(r.URL.Query().Get("showDeleted"))
	if err != nil {
		respondGCPRunInvalidArgument(w, path, "showDeleted must be a boolean")
		return true
	}
	items := []map[string]any{
		gcpRunRevision(project, location, serviceID, serviceID+"-00001"),
		gcpRunRevision(project, location, serviceID, serviceID+"-00002"),
	}
	if !showDeleted {
		items = items[:1]
	}
	return respondGCPRunList(w, "revisions", items, pageSize, start, path)
}

func handleGCPRunGetRevision(w http.ResponseWriter, path string) bool {
	project, location, serviceID, revisionID, ok := parseGCPRunRevisionPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRunRevision(project, location, serviceID, revisionID))
	return true
}

func handleGCPRunDeleteRevision(w http.ResponseWriter, path, etag string) bool {
	project, location, serviceID, revisionID, ok := parseGCPRunRevisionPath(path)
	if !ok {
		return false
	}
	if etag = strings.TrimSpace(etag); etag != "" && etag != gcpRunEtag(revisionID) {
		respondGCPRunFailedPrecondition(w, path, "etag mismatch")
		return true
	}
	respondJSON(w, http.StatusOK, gcpRunOperation(project, location, "deleteRevision."+revisionID, gcpRunRevisionName(project, location, serviceID, revisionID), "delete", false))
	return true
}

func handleGCPRunListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, scope, ok := parseGCPRunOperationsListScope(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPRunPagination(w, r, path)
	if !valid {
		return true
	}
	scopeSuffix := "location"
	if parts := strings.Split(strings.Trim(scope, "/"), "/"); len(parts) > 0 {
		scopeSuffix = parts[len(parts)-1]
	}
	items := []map[string]any{
		gcpRunOperation(project, location, "listOperations."+scopeSuffix, scope, "list", true),
	}
	return respondGCPRunList(w, "operations", items, pageSize, start, path)
}

func handleGCPRunGetOperation(w http.ResponseWriter, path string) bool {
	project, location, operationID, ok := parseGCPRunOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpRunOperation(project, location, operationID, fmt.Sprintf("projects/%s/locations/%s", project, location), "poll", true))
	return true
}

func handleGCPRunDeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, _, ok := parseGCPRunOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPRunOperationAction(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, operationID, action, ok := parseGCPRunOperationActionPath(path)
	if !ok {
		return false
	}
	switch action {
	case "cancel":
		if _, valid := decodeGCPRunJSONBody(w, r, path); !valid {
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{})
		return true
	case "wait":
		if _, valid := decodeGCPRunJSONBody(w, r, path); !valid {
			return true
		}
		respondJSON(w, http.StatusOK, gcpRunOperation(project, location, operationID, fmt.Sprintf("projects/%s/locations/%s", project, location), "wait", true))
		return true
	default:
		return false
	}
}

func parseGCPRunProjectLocationPath(path string) (project, location string, list, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 5 && len(parts) != 6 {
		return "", "", false, false
	}
	if parts[0] != "gcp" || (parts[1] != "v1" && parts[1] != "v2") || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", false, false
	}
	project = strings.TrimSpace(parts[3])
	if project == "" {
		return "", "", false, false
	}
	if len(parts) == 5 {
		return project, "", true, true
	}
	location = strings.TrimSpace(parts[5])
	if location == "" {
		return "", "", false, false
	}
	return project, location, false, true
}

func parseGCPRunLocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 {
		return "", "", nil, false
	}
	if parts[0] != "gcp" || parts[1] != "v2" || parts[2] != "projects" || parts[4] != "locations" {
		return "", "", nil, false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	if project == "" || location == "" {
		return "", "", nil, false
	}
	for _, part := range parts[6:] {
		if strings.TrimSpace(part) == "" {
			return "", "", nil, false
		}
	}
	return project, location, parts[6:], true
}

func parseGCPRunServicePath(path string) (project, location, serviceID string, ok bool) {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || !isGCPRunServiceTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPRunJobPath(path string) (project, location, jobID string, ok bool) {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || !isGCPRunJobTail(tail) {
		return "", "", "", false
	}
	return project, location, strings.TrimSpace(tail[1]), true
}

func parseGCPRunJobRunPath(path string) (project, location, jobID string, ok bool) {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || !isGCPRunJobRunActionTail(tail) {
		return "", "", "", false
	}
	jobID, _, _ = gcpRunResourceActionSegment(tail[1])
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return "", "", "", false
	}
	return project, location, jobID, true
}

func parseGCPRunExecutionsCollectionPath(path string) (project, location, jobID string, ok bool) {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || !isGCPRunExecutionsCollectionTail(tail) {
		return "", "", "", false
	}
	jobID = strings.TrimSpace(tail[1])
	if jobID == "" {
		return "", "", "", false
	}
	return project, location, jobID, true
}

func parseGCPRunExecutionPath(path string) (project, location, jobID, executionID string, ok bool) {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || !isGCPRunExecutionTail(tail) {
		return "", "", "", "", false
	}
	jobID = strings.TrimSpace(tail[1])
	executionID = strings.TrimSpace(tail[3])
	if jobID == "" || executionID == "" {
		return "", "", "", "", false
	}
	return project, location, jobID, executionID, true
}

func parseGCPRunExecutionCancelPath(path string) (project, location, jobID, executionID string, ok bool) {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || !isGCPRunExecutionCancelActionTail(tail) {
		return "", "", "", "", false
	}
	jobID = strings.TrimSpace(tail[1])
	executionID, _, _ = gcpRunResourceActionSegment(tail[3])
	executionID = strings.TrimSpace(executionID)
	if jobID == "" || executionID == "" {
		return "", "", "", "", false
	}
	return project, location, jobID, executionID, true
}

func parseGCPRunTasksCollectionPath(path string) (project, location, jobID, executionID string, ok bool) {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || !isGCPRunTasksCollectionTail(tail) {
		return "", "", "", "", false
	}
	jobID = strings.TrimSpace(tail[1])
	executionID = strings.TrimSpace(tail[3])
	if jobID == "" || executionID == "" {
		return "", "", "", "", false
	}
	return project, location, jobID, executionID, true
}

func parseGCPRunTaskPath(path string) (project, location, jobID, executionID, taskID string, ok bool) {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || !isGCPRunTaskTail(tail) {
		return "", "", "", "", "", false
	}
	jobID = strings.TrimSpace(tail[1])
	executionID = strings.TrimSpace(tail[3])
	taskID = strings.TrimSpace(tail[5])
	if jobID == "" || executionID == "" || taskID == "" {
		return "", "", "", "", "", false
	}
	return project, location, jobID, executionID, taskID, true
}

func parseGCPRunRevisionsCollectionPath(path string) (project, location, serviceID string, ok bool) {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || !isGCPRunRevisionsCollectionTail(tail) {
		return "", "", "", false
	}
	serviceID = strings.TrimSpace(tail[1])
	if serviceID == "" {
		return "", "", "", false
	}
	return project, location, serviceID, true
}

func parseGCPRunRevisionPath(path string) (project, location, serviceID, revisionID string, ok bool) {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || !isGCPRunRevisionTail(tail) {
		return "", "", "", "", false
	}
	serviceID = strings.TrimSpace(tail[1])
	revisionID = strings.TrimSpace(tail[3])
	if serviceID == "" || revisionID == "" {
		return "", "", "", "", false
	}
	return project, location, serviceID, revisionID, true
}

func parseGCPRunOperationsListScope(path string) (project, location, scope string, ok bool) {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || len(tail) == 0 {
		return "", "", "", false
	}
	if tail[len(tail)-1] != "operations" {
		return "", "", "", false
	}
	scopeParts := append([]string{"projects", project, "locations", location}, tail[:len(tail)-1]...)
	return project, location, strings.Join(scopeParts, "/"), true
}

func parseGCPRunOperationPath(path string) (project, location, operationID string, ok bool) {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || len(tail) < 2 {
		return "", "", "", false
	}
	for i := 0; i < len(tail); i++ {
		if tail[i] != "operations" {
			continue
		}
		if i+1 >= len(tail) || i+2 != len(tail) {
			return "", "", "", false
		}
		operationID, action, hasAction := gcpRunResourceActionSegment(tail[i+1])
		if strings.TrimSpace(operationID) == "" || (hasAction && strings.TrimSpace(action) != "") {
			return "", "", "", false
		}
		return project, location, strings.TrimSpace(operationID), true
	}
	return "", "", "", false
}

func parseGCPRunOperationActionPath(path string) (project, location, operationID, action string, ok bool) {
	project, location, tail, ok := parseGCPRunLocationTail(path)
	if !ok || len(tail) < 2 {
		return "", "", "", "", false
	}
	for i := 0; i < len(tail); i++ {
		if tail[i] != "operations" {
			continue
		}
		if i+1 >= len(tail) || i+2 != len(tail) {
			return "", "", "", "", false
		}
		operationID, action, hasAction := gcpRunResourceActionSegment(tail[i+1])
		if !hasAction || strings.TrimSpace(operationID) == "" || strings.TrimSpace(action) == "" {
			return "", "", "", "", false
		}
		return project, location, strings.TrimSpace(operationID), strings.TrimSpace(action), true
	}
	return "", "", "", "", false
}

func isGCPRunServicesCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "services"
}

func isGCPRunServiceTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "services" {
		return false
	}
	resource, action, hasAction := gcpRunResourceActionSegment(tail[1])
	return !hasAction && strings.TrimSpace(resource) != "" && strings.TrimSpace(action) == ""
}

func isGCPRunJobsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "jobs"
}

func isGCPRunJobTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "jobs" {
		return false
	}
	resource, action, hasAction := gcpRunResourceActionSegment(tail[1])
	return !hasAction && strings.TrimSpace(resource) != "" && strings.TrimSpace(action) == ""
}

func isGCPRunJobRunActionTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "jobs" {
		return false
	}
	resource, action, hasAction := gcpRunResourceActionSegment(tail[1])
	return hasAction && strings.TrimSpace(resource) != "" && strings.EqualFold(strings.TrimSpace(action), "run")
}

func isGCPRunExecutionsCollectionTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "jobs" && strings.TrimSpace(tail[1]) != "" && tail[2] == "executions"
}

func isGCPRunExecutionTail(tail []string) bool {
	if len(tail) != 4 || tail[0] != "jobs" || tail[2] != "executions" {
		return false
	}
	if strings.TrimSpace(tail[1]) == "" {
		return false
	}
	resource, action, hasAction := gcpRunResourceActionSegment(tail[3])
	return !hasAction && strings.TrimSpace(resource) != "" && strings.TrimSpace(action) == ""
}

func isGCPRunExecutionCancelActionTail(tail []string) bool {
	if len(tail) != 4 || tail[0] != "jobs" || tail[2] != "executions" {
		return false
	}
	if strings.TrimSpace(tail[1]) == "" {
		return false
	}
	resource, action, hasAction := gcpRunResourceActionSegment(tail[3])
	return hasAction && strings.TrimSpace(resource) != "" && strings.EqualFold(strings.TrimSpace(action), "cancel")
}

func isGCPRunTasksCollectionTail(tail []string) bool {
	return len(tail) == 5 && tail[0] == "jobs" && strings.TrimSpace(tail[1]) != "" && tail[2] == "executions" && strings.TrimSpace(tail[3]) != "" && tail[4] == "tasks"
}

func isGCPRunTaskTail(tail []string) bool {
	if len(tail) != 6 || tail[0] != "jobs" || tail[2] != "executions" || tail[4] != "tasks" {
		return false
	}
	if strings.TrimSpace(tail[1]) == "" || strings.TrimSpace(tail[3]) == "" {
		return false
	}
	resource, action, hasAction := gcpRunResourceActionSegment(tail[5])
	return !hasAction && strings.TrimSpace(resource) != "" && strings.TrimSpace(action) == ""
}

func isGCPRunRevisionsCollectionTail(tail []string) bool {
	return len(tail) == 3 && tail[0] == "services" && strings.TrimSpace(tail[1]) != "" && tail[2] == "revisions"
}

func isGCPRunRevisionTail(tail []string) bool {
	if len(tail) != 4 || tail[0] != "services" || tail[2] != "revisions" {
		return false
	}
	if strings.TrimSpace(tail[1]) == "" {
		return false
	}
	resource, action, hasAction := gcpRunResourceActionSegment(tail[3])
	return !hasAction && strings.TrimSpace(resource) != "" && strings.TrimSpace(action) == ""
}

func isGCPRunOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPRunNestedOperationsCollectionTail(tail []string) bool {
	if len(tail) < 2 {
		return false
	}
	return tail[len(tail)-1] == "operations"
}

func isGCPRunOperationTail(tail []string) bool {
	if len(tail) < 2 {
		return false
	}
	for i := 0; i < len(tail); i++ {
		if tail[i] != "operations" {
			continue
		}
		if i+1 >= len(tail) || i+2 != len(tail) {
			return false
		}
		resource, action, hasAction := gcpRunResourceActionSegment(tail[i+1])
		return !hasAction && strings.TrimSpace(resource) != "" && strings.TrimSpace(action) == ""
	}
	return false
}

func isGCPRunOperationActionTail(tail []string, expectedAction string) bool {
	if len(tail) < 2 {
		return false
	}
	for i := 0; i < len(tail); i++ {
		if tail[i] != "operations" {
			continue
		}
		if i+1 >= len(tail) || i+2 != len(tail) {
			return false
		}
		resource, action, hasAction := gcpRunResourceActionSegment(tail[i+1])
		return hasAction && strings.TrimSpace(resource) != "" && strings.EqualFold(strings.TrimSpace(action), expectedAction)
	}
	return false
}

func gcpRunResourceActionSegment(segment string) (resource, action string, hasAction bool) {
	segment = strings.TrimSpace(segment)
	resource, action, hasAction = strings.Cut(segment, ":")
	if !hasAction {
		return segment, "", false
	}
	return strings.TrimSpace(resource), strings.TrimSpace(action), true
}

func parseGCPRunPagination(w http.ResponseWriter, r *http.Request, path string) (pageSize, start int, ok bool) {
	pageSize, err := parseOptionalNonNegativeInt(r.URL.Query().Get("pageSize"))
	if err != nil {
		respondGCPRunInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return 0, 0, false
	}
	if pageSize > 1000 {
		respondGCPRunInvalidArgument(w, path, "pageSize must be less than or equal to 1000")
		return 0, 0, false
	}
	start = 0
	if token := strings.TrimSpace(r.URL.Query().Get("pageToken")); token != "" {
		start, err = parseOptionalNonNegativeInt(token)
		if err != nil {
			respondGCPRunInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
	}
	return pageSize, start, true
}

func parseGCPRunOptionalBool(raw string) (bool, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false, nil
	}
	if strings.EqualFold(value, "true") {
		return true, nil
	}
	if strings.EqualFold(value, "false") {
		return false, nil
	}
	return false, fmt.Errorf("expected bool")
}

func respondGCPRunList(w http.ResponseWriter, key string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPRunInvalidArgument(w, path, "pageToken is out of range")
		return false
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	out := make([]any, 0, end-start)
	for _, item := range items[start:end] {
		out = append(out, item)
	}
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		key:             out,
		"nextPageToken": nextPageToken,
	})
	return true
}

func decodeGCPRunJSONBody(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.UseNumber()
	var body map[string]any
	if err := decoder.Decode(&body); err != nil {
		if err == io.EOF {
			return map[string]any{}, true
		}
		respondGCPRunInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func gcpRunString(body map[string]any, key string) string {
	if body == nil {
		return ""
	}
	raw, ok := body[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func hasGCPRunRevisionTemplate(service map[string]any) bool {
	template, ok := service["template"].(map[string]any)
	if !ok {
		return false
	}
	containers, ok := template["containers"].([]any)
	return ok && len(containers) > 0
}

func hasGCPRunExecutionTemplate(job map[string]any) bool {
	template, ok := job["template"].(map[string]any)
	if !ok {
		return false
	}
	taskTemplate, ok := template["template"].(map[string]any)
	if !ok {
		return false
	}
	containers, ok := taskTemplate["containers"].([]any)
	return ok && len(containers) > 0
}

func isGCPRunResourceID(id string) bool {
	if len(id) == 0 || len(id) > 49 {
		return false
	}
	for i := 0; i < len(id); i++ {
		ch := id[i]
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= '0' && ch <= '9':
		case ch == '-':
		default:
			return false
		}
		if i == 0 && !(ch >= 'a' && ch <= 'z') {
			return false
		}
		if i == len(id)-1 && ch == '-' {
			return false
		}
	}
	return true
}

func gcpRunLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": strings.ToUpper(location),
		"labels": map[string]any{
			"cloud.googleapis.com/region": location,
		},
		"metadata": map[string]any{
			"service": "run.googleapis.com",
		},
	}
}

func gcpRunService(project, location, serviceID string) map[string]any {
	name := gcpRunServiceName(project, location, serviceID)
	revisionID := serviceID + "-00001"
	revisionName := gcpRunRevisionName(project, location, serviceID, revisionID)
	uri := fmt.Sprintf("https://%s-%s-%s.a.run.app", serviceID, project, location)
	transition := gcpRunReferenceTime.Add(5 * time.Minute).Format(time.RFC3339)
	return map[string]any{
		"name":               name,
		"uid":                fmt.Sprintf("run-service-%s", serviceID),
		"generation":         "3",
		"labels":             map[string]any{"env": "staged"},
		"annotations":        map[string]any{"stackyard.dev/example": "run"},
		"createTime":         gcpRunReferenceTime.Format(time.RFC3339),
		"updateTime":         gcpRunReferenceTime.Add(2 * time.Hour).Format(time.RFC3339),
		"creator":            "stackyard@example.com",
		"lastModifier":       "stackyard@example.com",
		"template":           map[string]any{"revision": revisionID, "containers": []any{map[string]any{"name": "app", "image": "us-docker.pkg.dev/cloudrun/container/hello"}}},
		"traffic":            []any{map[string]any{"type": "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST", "percent": 100}},
		"observedGeneration": "3",
		"terminalCondition": map[string]any{
			"type":               "Ready",
			"state":              "CONDITION_SUCCEEDED",
			"lastTransitionTime": transition,
		},
		"conditions": []any{
			map[string]any{
				"type":               "Ready",
				"state":              "CONDITION_SUCCEEDED",
				"lastTransitionTime": transition,
			},
		},
		"latestReadyRevision":   revisionName,
		"latestCreatedRevision": revisionName,
		"trafficStatuses": []any{
			map[string]any{"type": "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST", "percent": 100, "revision": revisionName, "uri": uri},
		},
		"uri":         uri,
		"urls":        []any{uri},
		"reconciling": false,
		"etag":        gcpRunEtag(serviceID),
	}
}

func gcpRunJob(project, location, jobID string) map[string]any {
	name := gcpRunJobName(project, location, jobID)
	executionName := gcpRunExecutionName(project, location, jobID, "execution-1")
	transition := gcpRunReferenceTime.Add(10 * time.Minute).Format(time.RFC3339)
	return map[string]any{
		"name":               name,
		"uid":                fmt.Sprintf("run-job-%s", jobID),
		"generation":         "2",
		"labels":             map[string]any{"env": "staged"},
		"annotations":        map[string]any{"stackyard.dev/example": "run"},
		"createTime":         gcpRunReferenceTime.Format(time.RFC3339),
		"updateTime":         gcpRunReferenceTime.Add(3 * time.Hour).Format(time.RFC3339),
		"creator":            "stackyard@example.com",
		"lastModifier":       "stackyard@example.com",
		"template":           map[string]any{"parallelism": 1, "taskCount": 1, "template": map[string]any{"containers": []any{map[string]any{"name": "job", "image": "us-docker.pkg.dev/cloudrun/container/job"}}, "maxRetries": 3, "timeout": "600s"}},
		"observedGeneration": "2",
		"terminalCondition": map[string]any{
			"type":               "Ready",
			"state":              "CONDITION_SUCCEEDED",
			"lastTransitionTime": transition,
		},
		"conditions": []any{
			map[string]any{
				"type":               "Ready",
				"state":              "CONDITION_SUCCEEDED",
				"lastTransitionTime": transition,
			},
		},
		"executionCount": 1,
		"latestCreatedExecution": map[string]any{
			"name":             executionName,
			"createTime":       gcpRunReferenceTime.Add(4 * time.Hour).Format(time.RFC3339),
			"completionTime":   gcpRunReferenceTime.Add(4*time.Hour + 5*time.Minute).Format(time.RFC3339),
			"completionStatus": "EXECUTION_SUCCEEDED",
		},
		"reconciling": false,
		"etag":        gcpRunEtag(jobID),
	}
}

func gcpRunExecution(project, location, jobID, executionID string) map[string]any {
	name := gcpRunExecutionName(project, location, jobID, executionID)
	jobName := gcpRunJobName(project, location, jobID)
	transition := gcpRunReferenceTime.Add(15 * time.Minute).Format(time.RFC3339)
	return map[string]any{
		"name":               name,
		"uid":                fmt.Sprintf("run-execution-%s", executionID),
		"creator":            "stackyard@example.com",
		"generation":         "1",
		"labels":             map[string]any{"job": jobID},
		"annotations":        map[string]any{"stackyard.dev/example": "run"},
		"createTime":         gcpRunReferenceTime.Add(4 * time.Hour).Format(time.RFC3339),
		"startTime":          gcpRunReferenceTime.Add(4*time.Hour + 30*time.Second).Format(time.RFC3339),
		"completionTime":     gcpRunReferenceTime.Add(4*time.Hour + 5*time.Minute).Format(time.RFC3339),
		"updateTime":         gcpRunReferenceTime.Add(4*time.Hour + 5*time.Minute).Format(time.RFC3339),
		"job":                jobName,
		"parallelism":        1,
		"taskCount":          1,
		"template":           map[string]any{"containers": []any{map[string]any{"name": "job", "image": "us-docker.pkg.dev/cloudrun/container/job"}}, "maxRetries": 3, "timeout": "600s"},
		"reconciling":        false,
		"conditions":         []any{map[string]any{"type": "Completed", "state": "CONDITION_SUCCEEDED", "lastTransitionTime": transition}},
		"observedGeneration": "1",
		"runningCount":       0,
		"succeededCount":     1,
		"failedCount":        0,
		"cancelledCount":     0,
		"retriedCount":       0,
		"logUri":             fmt.Sprintf("https://console.cloud.google.com/run/jobs/details/%s/%s/executions/%s/logs", location, jobID, executionID),
		"etag":               gcpRunEtag(executionID),
	}
}

func gcpRunTask(project, location, jobID, executionID, taskID string) map[string]any {
	name := gcpRunTaskName(project, location, jobID, executionID, taskID)
	executionName := gcpRunExecutionName(project, location, jobID, executionID)
	jobName := gcpRunJobName(project, location, jobID)
	transition := gcpRunReferenceTime.Add(20 * time.Minute).Format(time.RFC3339)
	return map[string]any{
		"name":               name,
		"uid":                fmt.Sprintf("run-task-%s", taskID),
		"generation":         "1",
		"labels":             map[string]any{"execution": executionID},
		"annotations":        map[string]any{"stackyard.dev/example": "run"},
		"createTime":         gcpRunReferenceTime.Add(4*time.Hour + 10*time.Second).Format(time.RFC3339),
		"scheduledTime":      gcpRunReferenceTime.Add(4*time.Hour + 20*time.Second).Format(time.RFC3339),
		"startTime":          gcpRunReferenceTime.Add(4*time.Hour + 30*time.Second).Format(time.RFC3339),
		"completionTime":     gcpRunReferenceTime.Add(4*time.Hour + 3*time.Minute).Format(time.RFC3339),
		"updateTime":         gcpRunReferenceTime.Add(4*time.Hour + 3*time.Minute).Format(time.RFC3339),
		"job":                jobName,
		"execution":          executionName,
		"containers":         []any{map[string]any{"name": "task", "image": "us-docker.pkg.dev/cloudrun/container/job"}},
		"maxRetries":         3,
		"timeout":            "600s",
		"serviceAccount":     "stackyard@stackyard.iam.gserviceaccount.com",
		"reconciling":        false,
		"conditions":         []any{map[string]any{"type": "Completed", "state": "CONDITION_SUCCEEDED", "lastTransitionTime": transition}},
		"observedGeneration": "1",
		"index":              0,
		"retried":            0,
		"lastAttemptResult":  map[string]any{"status": "TASK_ATTEMPT_SUCCEEDED", "exitCode": 0},
		"logUri":             fmt.Sprintf("https://console.cloud.google.com/run/jobs/details/%s/%s/executions/%s/tasks/%s/logs", location, jobID, executionID, taskID),
		"etag":               gcpRunEtag(taskID),
	}
}

func gcpRunRevision(project, location, serviceID, revisionID string) map[string]any {
	name := gcpRunRevisionName(project, location, serviceID, revisionID)
	serviceName := gcpRunServiceName(project, location, serviceID)
	transition := gcpRunReferenceTime.Add(25 * time.Minute).Format(time.RFC3339)
	return map[string]any{
		"name":                          name,
		"uid":                           fmt.Sprintf("run-revision-%s", revisionID),
		"generation":                    "1",
		"labels":                        map[string]any{"service": serviceID},
		"annotations":                   map[string]any{"stackyard.dev/example": "run"},
		"createTime":                    gcpRunReferenceTime.Format(time.RFC3339),
		"updateTime":                    gcpRunReferenceTime.Add(2 * time.Hour).Format(time.RFC3339),
		"service":                       serviceName,
		"maxInstanceRequestConcurrency": 80,
		"timeout":                       "300s",
		"serviceAccount":                "stackyard@stackyard.iam.gserviceaccount.com",
		"containers":                    []any{map[string]any{"name": "app", "image": "us-docker.pkg.dev/cloudrun/container/hello"}},
		"reconciling":                   false,
		"conditions":                    []any{map[string]any{"type": "Ready", "state": "CONDITION_SUCCEEDED", "lastTransitionTime": transition}},
		"observedGeneration":            "1",
		"logUri":                        fmt.Sprintf("https://console.cloud.google.com/run/detail/%s/%s/revisions/%s/logs", location, serviceID, revisionID),
		"creator":                       "stackyard@example.com",
		"etag":                          gcpRunEtag(revisionID),
	}
}

func gcpRunOperation(project, location, operationID, target, verb string, done bool) map[string]any {
	created := gcpRunReferenceTime.Add(30 * time.Minute).Format(time.RFC3339)
	response := map[string]any{
		"name": fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID),
		"metadata": map[string]any{
			"@type":      "type.googleapis.com/google.cloud.run.v2.OperationMetadata",
			"target":     target,
			"verb":       verb,
			"createTime": created,
		},
		"done": done,
	}
	if done {
		response["response"] = map[string]any{
			"@type": "type.googleapis.com/google.protobuf.Empty",
		}
	}
	return response
}

func gcpRunServiceName(project, location, serviceID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/services/%s", project, location, serviceID)
}

func gcpRunJobName(project, location, jobID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/jobs/%s", project, location, jobID)
}

func gcpRunExecutionName(project, location, jobID, executionID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/jobs/%s/executions/%s", project, location, jobID, executionID)
}

func gcpRunTaskName(project, location, jobID, executionID, taskID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/jobs/%s/executions/%s/tasks/%s", project, location, jobID, executionID, taskID)
}

func gcpRunRevisionName(project, location, serviceID, revisionID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/services/%s/revisions/%s", project, location, serviceID, revisionID)
}

func gcpRunEtag(id string) string {
	return fmt.Sprintf("etag-%s", strings.TrimSpace(id))
}

func respondGCPRunInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPRunError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPRunFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPRunError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPRunError(w http.ResponseWriter, status int, err, path, message string) {
	respondJSON(w, status, map[string]any{
		"error":    err,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_run(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !isGCPContractProbeRequestForService(r, path, "run") {
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
			"name":     "projects/stackyard/locations/us-central1/services/sample-service",
			"service":  "run",
			"provider": providerGCP,
			"path":     path,
		})
		return true
	}
	return false
}
