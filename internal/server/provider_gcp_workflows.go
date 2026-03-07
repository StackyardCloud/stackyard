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

const (
	gcpWorkflowsGRPCPathPrefix           = "/gcp/google.cloud.workflows.v1.Workflows/"
	gcpWorkflowsLocationsGRPCPathPrefix  = "/gcp/google.cloud.location.Locations/"
	gcpWorkflowsOperationsGRPCPathPrefix = "/gcp/google.longrunning.Operations/"
)

var (
	gcpWorkflowsReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	gcpWorkflowsIDRegex       = regexp.MustCompile(`^[A-Za-z](?:[A-Za-z0-9_-]{0,62}[A-Za-z0-9])?$`)
	gcpWorkflowsRevisionRegex = regexp.MustCompile(`^\d{6}-[a-z0-9]{3}$`)
)

func (s *Server) handleGCPWorkflowsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_workflows(w, r) {
		return true
	}

	path := normalizeGCPWorkflowsPath(rawRequestPath(r))
	if strings.HasPrefix(path, gcpWorkflowsGRPCPathPrefix) ||
		strings.HasPrefix(path, gcpWorkflowsLocationsGRPCPathPrefix) ||
		strings.HasPrefix(path, gcpWorkflowsOperationsGRPCPathPrefix) {
		return handleGCPWorkflowsGRPCBridge(w, r, path)
	}

	if isGCPWorkflowsLocationRequest(r, path) {
		if r.Method != http.MethodGet {
			return false
		}
		if handleGCPWorkflowsListLocations(w, r, path) {
			return true
		}
		if handleGCPWorkflowsGetLocation(w, path) {
			return true
		}
		return false
	}

	includeHint := hasGCPWorkflowsHint(r)
	if !isGCPWorkflowsPath(path, includeHint, includeHint) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPWorkflowsListWorkflows(w, r, path) {
			return true
		}
		if handleGCPWorkflowsGetWorkflow(w, r, path) {
			return true
		}
		if handleGCPWorkflowsListWorkflowRevisions(w, r, path) {
			return true
		}
		if handleGCPWorkflowsListOperations(w, r, path) {
			return true
		}
		if handleGCPWorkflowsGetOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPWorkflowsCreateWorkflow(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPatch:
		if handleGCPWorkflowsUpdateWorkflow(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodDelete:
		if handleGCPWorkflowsDeleteWorkflow(w, path) {
			return true
		}
		if handleGCPWorkflowsDeleteOperation(w, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPWorkflowsPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPWorkflowsHint(r *http.Request) bool {
	service := strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service"))
	if service != "" && isGCPWorkflowsServiceHintValue(service) {
		return true
	}

	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-workflows-apiv1") || strings.Contains(ua, "cloud.google.com/go/workflows")
}

func isGCPWorkflowsServiceHintValue(service string) bool {
	switch strings.ToLower(strings.TrimSpace(service)) {
	case "workflows", "workflows-apiv1", "workflows_apiv1", "gcp-workflows", "gcp-workflows-apiv1":
		return true
	default:
		return false
	}
}

func isGCPWorkflowsLocationRequest(r *http.Request, path string) bool {
	return hasGCPWorkflowsHint(r) && isGCPProjectLocationDiscoveryPath(path)
}

func isGCPWorkflowsPath(path string, includeHint, allowAmbiguousOps bool) bool {
	if strings.HasPrefix(path, gcpWorkflowsGRPCPathPrefix) ||
		strings.HasPrefix(path, gcpWorkflowsLocationsGRPCPathPrefix) ||
		strings.HasPrefix(path, gcpWorkflowsOperationsGRPCPathPrefix) {
		return true
	}
	if _, _, tail, ok := parseGCPWorkflowsLocationTail(path); ok {
		if isGCPWorkflowsCollectionTail(tail) ||
			isGCPWorkflowResourceTail(tail) ||
			isGCPWorkflowListRevisionsTail(tail) {
			return true
		}
		if allowAmbiguousOps && (isGCPOperationsCollectionTail(tail) || isGCPOperationResourceTail(tail)) {
			return true
		}
	}
	return includeHint && strings.HasPrefix(path, "/gcp/v1/projects/")
}

func handleGCPWorkflowsListLocations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, _, list, ok := parseGCPProjectLocationPath(path)
	if !ok || !list {
		return false
	}

	pageSize, start, valid := parseGCPWorkflowsPaginationFromQuery(w, r, path, 1000)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpWorkflowsLocation(project, "us-central1"),
		gcpWorkflowsLocation(project, "global"),
	}
	return respondGCPWorkflowsList(w, "locations", items, pageSize, start, path)
}

func handleGCPWorkflowsGetLocation(w http.ResponseWriter, path string) bool {
	project, location, list, ok := parseGCPProjectLocationPath(path)
	if !ok || list {
		return false
	}
	respondJSON(w, http.StatusOK, gcpWorkflowsLocation(project, location))
	return true
}

func handleGCPWorkflowsListWorkflows(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPWorkflowsLocationTail(path)
	if !ok || !isGCPWorkflowsCollectionTail(tail) {
		return false
	}

	pageSize, start, valid := parseGCPWorkflowsPaginationFromQuery(w, r, path, 1000)
	if !valid {
		return true
	}

	filter := strings.TrimSpace(queryValue(r, "filter"))
	if !isGCPWorkflowsSupportedFilter(filter) {
		respondGCPWorkflowsInvalidArgument(w, path, "filter is unsupported")
		return true
	}
	orderBy := strings.TrimSpace(queryValue(r, "orderBy", "order_by"))
	if !isGCPWorkflowsSupportedOrderBy(orderBy) {
		respondGCPWorkflowsInvalidArgument(w, path, "orderBy is unsupported")
		return true
	}

	items := []map[string]any{
		gcpWorkflowsWorkflow(project, location, "workflow-1", "000001-a4d", "ACTIVE"),
		gcpWorkflowsWorkflow(project, location, "workflow-2", "000002-b5e", "UNAVAILABLE"),
	}

	if filter != "" {
		targetState := "ACTIVE"
		if strings.Contains(strings.ToUpper(filter), "UNAVAILABLE") {
			targetState = "UNAVAILABLE"
		}
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if strings.EqualFold(gcpWorkflowsString(item, "state"), targetState) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if strings.EqualFold(strings.TrimSpace(orderBy), "createTime desc") {
		sort.SliceStable(items, func(i, j int) bool {
			return gcpWorkflowsString(items[i], "createTime") > gcpWorkflowsString(items[j], "createTime")
		})
	}
	if strings.EqualFold(strings.TrimSpace(orderBy), "updateTime desc") {
		sort.SliceStable(items, func(i, j int) bool {
			return gcpWorkflowsString(items[i], "updateTime") > gcpWorkflowsString(items[j], "updateTime")
		})
	}

	if start > len(items) {
		respondGCPWorkflowsOutOfRange(w, path, "pageToken is out of range")
		return true
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"workflows":     items[start:end],
		"nextPageToken": next,
		"unreachable":   []string{},
	})
	return true
}

func handleGCPWorkflowsGetWorkflow(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, workflowID, ok := parseGCPWorkflowsWorkflowPath(path)
	if !ok {
		return false
	}

	revisionID := strings.TrimSpace(queryValue(r, "revisionId", "revision_id"))
	if revisionID != "" && !gcpWorkflowsRevisionRegex.MatchString(revisionID) {
		respondGCPWorkflowsInvalidArgument(w, path, "revisionId is invalid")
		return true
	}
	if revisionID == "" {
		revisionID = "000001-a4d"
	}

	respondJSON(w, http.StatusOK, gcpWorkflowsWorkflow(project, location, workflowID, revisionID, "ACTIVE"))
	return true
}

func handleGCPWorkflowsCreateWorkflow(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPWorkflowsLocationTail(path)
	if !ok || !isGCPWorkflowsCollectionTail(tail) {
		return false
	}

	workflowID := strings.TrimSpace(queryValue(r, "workflowId", "workflow_id"))
	if workflowID == "" {
		respondGCPWorkflowsInvalidArgument(w, path, "workflowId is required")
		return true
	}
	if !isGCPWorkflowsID(workflowID) {
		respondGCPWorkflowsInvalidArgument(w, path, "workflowId is invalid")
		return true
	}

	body, valid := decodeGCPWorkflowsJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	if parent := strings.TrimSpace(gcpWorkflowsString(body, "parent")); parent != "" {
		reqProject, reqLocation, parsed := parseGCPWorkflowsParent(parent)
		if !parsed || reqProject != project || reqLocation != location {
			respondGCPWorkflowsInvalidArgument(w, path, "parent must match requested resource")
			return true
		}
	}

	workflow := gcpWorkflowsWorkflowFromBody(body)
	if len(workflow) == 0 {
		respondGCPWorkflowsInvalidArgument(w, path, "workflow is required")
		return true
	}
	if !validateGCPWorkflowsWorkflow(w, path, workflow, false) {
		return true
	}

	expectedName := gcpWorkflowsWorkflowName(project, location, workflowID)
	if name := strings.TrimSpace(gcpWorkflowsString(workflow, "name")); name != "" && name != expectedName {
		respondGCPWorkflowsInvalidArgument(w, path, "workflow.name must match parent and workflowId")
		return true
	}

	respondJSON(w, http.StatusOK, gcpWorkflowsOperation(project, location, "createWorkflow."+workflowID, expectedName, "create", false))
	return true
}

func handleGCPWorkflowsDeleteWorkflow(w http.ResponseWriter, path string) bool {
	project, location, workflowID, ok := parseGCPWorkflowsWorkflowPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, gcpWorkflowsOperation(project, location, "deleteWorkflow."+workflowID, gcpWorkflowsWorkflowName(project, location, workflowID), "delete", false))
	return true
}

func handleGCPWorkflowsUpdateWorkflow(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, workflowID, ok := parseGCPWorkflowsWorkflowPath(path)
	if !ok {
		return false
	}

	body, valid := decodeGCPWorkflowsJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	workflow := gcpWorkflowsWorkflowFromBody(body)
	if len(workflow) == 0 {
		respondGCPWorkflowsInvalidArgument(w, path, "workflow is required")
		return true
	}
	if !validateGCPWorkflowsWorkflow(w, path, workflow, true) {
		return true
	}

	expectedName := gcpWorkflowsWorkflowName(project, location, workflowID)
	if name := strings.TrimSpace(gcpWorkflowsString(workflow, "name")); name != expectedName {
		respondGCPWorkflowsInvalidArgument(w, path, "workflow.name must match requested resource")
		return true
	}

	if rawMask, exists := gcpWorkflowsLookup(body, "updateMask", "update_mask"); exists {
		if strings.TrimSpace(gcpWorkflowsToString(rawMask)) == "" {
			respondGCPWorkflowsInvalidArgument(w, path, "updateMask is invalid")
			return true
		}
	}
	if rawMask := strings.TrimSpace(queryValue(r, "updateMask", "update_mask")); rawMask != "" && rawMask == "," {
		respondGCPWorkflowsInvalidArgument(w, path, "updateMask is invalid")
		return true
	}

	respondJSON(w, http.StatusOK, gcpWorkflowsOperation(project, location, "updateWorkflow."+workflowID, expectedName, "update", false))
	return true
}

func handleGCPWorkflowsListWorkflowRevisions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, workflowID, ok := parseGCPWorkflowsListRevisionsPath(path)
	if !ok {
		return false
	}
	pageSize, start, valid := parseGCPWorkflowsPaginationFromQuery(w, r, path, 100)
	if !valid {
		return true
	}

	items := []map[string]any{
		gcpWorkflowsWorkflow(project, location, workflowID, "000003-c6f", "ACTIVE"),
		gcpWorkflowsWorkflow(project, location, workflowID, "000002-b5e", "ACTIVE"),
		gcpWorkflowsWorkflow(project, location, workflowID, "000001-a4d", "ACTIVE"),
	}
	return respondGCPWorkflowsList(w, "workflows", items, pageSize, start, path)
}

func handleGCPWorkflowsListOperations(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, tail, ok := parseGCPWorkflowsLocationTail(path)
	if !ok || !isGCPOperationsCollectionTail(tail) {
		return false
	}

	pageSize, start, valid := parseGCPWorkflowsPaginationFromQuery(w, r, path, 1000)
	if !valid {
		return true
	}

	filter := strings.TrimSpace(queryValue(r, "filter"))
	if !isGCPWorkflowsSupportedOperationsFilter(filter) {
		respondGCPWorkflowsInvalidArgument(w, path, "filter is malformed")
		return true
	}

	items := []map[string]any{
		gcpWorkflowsOperation(project, location, "op-1", gcpWorkflowsWorkflowName(project, location, "workflow-1"), "create", false),
		gcpWorkflowsOperation(project, location, "op-2", gcpWorkflowsWorkflowName(project, location, "workflow-2"), "update", true),
	}
	if filter == "done=true" {
		items = []map[string]any{items[1]}
	}
	if filter == "done=false" {
		items = []map[string]any{items[0]}
	}
	return respondGCPWorkflowsList(w, "operations", items, pageSize, start, path)
}

func handleGCPWorkflowsGetOperation(w http.ResponseWriter, path string) bool {
	project, location, operationID, ok := parseGCPWorkflowsOperationPath(path)
	if !ok {
		return false
	}
	target := gcpWorkflowsWorkflowName(project, location, "workflow-1")
	verb := "get"
	done := true
	switch {
	case strings.HasPrefix(operationID, "createWorkflow."):
		workflowID := strings.TrimPrefix(operationID, "createWorkflow.")
		target = gcpWorkflowsWorkflowName(project, location, workflowID)
		verb = "create"
		done = false
	case strings.HasPrefix(operationID, "updateWorkflow."):
		workflowID := strings.TrimPrefix(operationID, "updateWorkflow.")
		target = gcpWorkflowsWorkflowName(project, location, workflowID)
		verb = "update"
		done = false
	case strings.HasPrefix(operationID, "deleteWorkflow."):
		workflowID := strings.TrimPrefix(operationID, "deleteWorkflow.")
		target = gcpWorkflowsWorkflowName(project, location, workflowID)
		verb = "delete"
		done = false
	}
	respondJSON(w, http.StatusOK, gcpWorkflowsOperation(project, location, operationID, target, verb, done))
	return true
}

func handleGCPWorkflowsDeleteOperation(w http.ResponseWriter, path string) bool {
	_, _, _, ok := parseGCPWorkflowsOperationPath(path)
	if !ok {
		return false
	}
	respondJSON(w, http.StatusOK, map[string]any{})
	return true
}

func handleGCPWorkflowsGRPCBridge(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.Method != http.MethodPost {
		return false
	}

	body, valid := decodeGCPWorkflowsJSONBodyOptional(w, r, path)
	if !valid {
		return true
	}

	switch {
	case strings.HasPrefix(path, gcpWorkflowsGRPCPathPrefix):
		method := strings.TrimPrefix(path, gcpWorkflowsGRPCPathPrefix)
		switch method {
		case "ListWorkflows":
			project, location, ok := parseGCPWorkflowsParent(gcpWorkflowsString(body, "parent"))
			if !ok {
				respondGCPWorkflowsInvalidArgument(w, path, "parent is required")
				return true
			}
			pageSize, start, valid := parseGCPWorkflowsPaginationFromBody(w, path, body, 1000)
			if !valid {
				return true
			}
			items := []map[string]any{
				gcpWorkflowsWorkflow(project, location, "workflow-1", "000001-a4d", "ACTIVE"),
				gcpWorkflowsWorkflow(project, location, "workflow-2", "000002-b5e", "UNAVAILABLE"),
			}
			return respondGCPWorkflowsList(w, "workflows", items, pageSize, start, path)
		case "GetWorkflow":
			project, location, workflowID, ok := parseGCPWorkflowsWorkflowName(gcpWorkflowsString(body, "name"))
			if !ok {
				respondGCPWorkflowsInvalidArgument(w, path, "name is required")
				return true
			}
			revisionID := strings.TrimSpace(gcpWorkflowsString(body, "revisionId", "revision_id"))
			if revisionID != "" && !gcpWorkflowsRevisionRegex.MatchString(revisionID) {
				respondGCPWorkflowsInvalidArgument(w, path, "revisionId is invalid")
				return true
			}
			if revisionID == "" {
				revisionID = "000001-a4d"
			}
			respondJSON(w, http.StatusOK, gcpWorkflowsWorkflow(project, location, workflowID, revisionID, "ACTIVE"))
			return true
		case "CreateWorkflow":
			project, location, ok := parseGCPWorkflowsParent(gcpWorkflowsString(body, "parent"))
			if !ok {
				respondGCPWorkflowsInvalidArgument(w, path, "parent is required")
				return true
			}
			workflowID := strings.TrimSpace(gcpWorkflowsString(body, "workflowId", "workflow_id"))
			if workflowID == "" || !isGCPWorkflowsID(workflowID) {
				respondGCPWorkflowsInvalidArgument(w, path, "workflowId is invalid")
				return true
			}
			workflow := gcpWorkflowsWorkflowFromBody(body)
			if len(workflow) == 0 || !validateGCPWorkflowsWorkflow(w, path, workflow, false) {
				if len(workflow) == 0 {
					respondGCPWorkflowsInvalidArgument(w, path, "workflow is required")
				}
				return true
			}
			expectedName := gcpWorkflowsWorkflowName(project, location, workflowID)
			respondJSON(w, http.StatusOK, gcpWorkflowsOperation(project, location, "createWorkflow."+workflowID, expectedName, "create", false))
			return true
		case "DeleteWorkflow":
			project, location, workflowID, ok := parseGCPWorkflowsWorkflowName(gcpWorkflowsString(body, "name"))
			if !ok {
				respondGCPWorkflowsInvalidArgument(w, path, "name is required")
				return true
			}
			respondJSON(w, http.StatusOK, gcpWorkflowsOperation(project, location, "deleteWorkflow."+workflowID, gcpWorkflowsWorkflowName(project, location, workflowID), "delete", false))
			return true
		case "UpdateWorkflow":
			workflow := gcpWorkflowsWorkflowFromBody(body)
			if len(workflow) == 0 {
				respondGCPWorkflowsInvalidArgument(w, path, "workflow is required")
				return true
			}
			if !validateGCPWorkflowsWorkflow(w, path, workflow, true) {
				return true
			}
			project, location, workflowID, ok := parseGCPWorkflowsWorkflowName(gcpWorkflowsString(workflow, "name"))
			if !ok {
				respondGCPWorkflowsInvalidArgument(w, path, "workflow.name is required")
				return true
			}
			respondJSON(w, http.StatusOK, gcpWorkflowsOperation(project, location, "updateWorkflow."+workflowID, gcpWorkflowsWorkflowName(project, location, workflowID), "update", false))
			return true
		case "ListWorkflowRevisions":
			project, location, workflowID, ok := parseGCPWorkflowsWorkflowName(gcpWorkflowsString(body, "name"))
			if !ok {
				respondGCPWorkflowsInvalidArgument(w, path, "name is required")
				return true
			}
			pageSize, start, valid := parseGCPWorkflowsPaginationFromBody(w, path, body, 100)
			if !valid {
				return true
			}
			items := []map[string]any{
				gcpWorkflowsWorkflow(project, location, workflowID, "000003-c6f", "ACTIVE"),
				gcpWorkflowsWorkflow(project, location, workflowID, "000002-b5e", "ACTIVE"),
				gcpWorkflowsWorkflow(project, location, workflowID, "000001-a4d", "ACTIVE"),
			}
			return respondGCPWorkflowsList(w, "workflows", items, pageSize, start, path)
		default:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
	case strings.HasPrefix(path, gcpWorkflowsLocationsGRPCPathPrefix):
		method := strings.TrimPrefix(path, gcpWorkflowsLocationsGRPCPathPrefix)
		switch method {
		case "GetLocation":
			project, location, ok := parseGCPStage4LocationName(gcpWorkflowsString(body, "name"))
			if !ok {
				respondGCPWorkflowsInvalidArgument(w, path, "name is required")
				return true
			}
			respondJSON(w, http.StatusOK, gcpWorkflowsLocation(project, location))
			return true
		case "ListLocations":
			project, ok := parseGCPStage4ProjectParent(gcpWorkflowsString(body, "name"))
			if !ok {
				respondGCPWorkflowsInvalidArgument(w, path, "name is required")
				return true
			}
			pageSize, start, valid := parseGCPWorkflowsPaginationFromBody(w, path, body, 1000)
			if !valid {
				return true
			}
			items := []map[string]any{
				gcpWorkflowsLocation(project, "us-central1"),
				gcpWorkflowsLocation(project, "global"),
			}
			return respondGCPWorkflowsList(w, "locations", items, pageSize, start, path)
		default:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
	case strings.HasPrefix(path, gcpWorkflowsOperationsGRPCPathPrefix):
		method := strings.TrimPrefix(path, gcpWorkflowsOperationsGRPCPathPrefix)
		switch method {
		case "GetOperation":
			name := strings.TrimSpace(gcpWorkflowsString(body, "name"))
			project, location, operationID, ok := parseGCPWorkflowsOperationName(name)
			if !ok {
				respondGCPWorkflowsInvalidArgument(w, path, "name is required")
				return true
			}
			respondJSON(w, http.StatusOK, gcpWorkflowsOperation(project, location, operationID, gcpWorkflowsWorkflowName(project, location, "workflow-1"), "get", true))
			return true
		case "ListOperations":
			project, location, ok := parseGCPWorkflowsParent(gcpWorkflowsString(body, "name"))
			if !ok {
				respondGCPWorkflowsInvalidArgument(w, path, "name is required")
				return true
			}
			pageSize, start, valid := parseGCPWorkflowsPaginationFromBody(w, path, body, 1000)
			if !valid {
				return true
			}
			items := []map[string]any{
				gcpWorkflowsOperation(project, location, "op-1", gcpWorkflowsWorkflowName(project, location, "workflow-1"), "create", false),
				gcpWorkflowsOperation(project, location, "op-2", gcpWorkflowsWorkflowName(project, location, "workflow-2"), "update", true),
			}
			return respondGCPWorkflowsList(w, "operations", items, pageSize, start, path)
		case "DeleteOperation":
			if _, _, _, ok := parseGCPWorkflowsOperationName(gcpWorkflowsString(body, "name")); !ok {
				respondGCPWorkflowsInvalidArgument(w, path, "name is required")
				return true
			}
			respondJSON(w, http.StatusOK, map[string]any{})
			return true
		default:
			respondProviderNotImplemented(w, providerGCP, path)
			return true
		}
	default:
		return false
	}
}

func parseGCPWorkflowsLocationTail(path string) (project, location string, tail []string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 7 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" {
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

func isGCPWorkflowsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "workflows"
}

func isGCPWorkflowResourceTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "workflows" && strings.TrimSpace(tail[1]) != "" && !strings.Contains(tail[1], ":")
}

func isGCPWorkflowListRevisionsTail(tail []string) bool {
	if len(tail) != 2 || tail[0] != "workflows" {
		return false
	}
	workflowID, action, ok := strings.Cut(strings.TrimSpace(tail[1]), ":")
	return ok && strings.TrimSpace(workflowID) != "" && strings.TrimSpace(action) == "listRevisions"
}

func isGCPOperationsCollectionTail(tail []string) bool {
	return len(tail) == 1 && tail[0] == "operations"
}

func isGCPOperationResourceTail(tail []string) bool {
	return len(tail) == 2 && tail[0] == "operations" && strings.TrimSpace(tail[1]) != ""
}

func parseGCPWorkflowsWorkflowPath(path string) (project, location, workflowID string, ok bool) {
	project, location, tail, ok := parseGCPWorkflowsLocationTail(path)
	if !ok || !isGCPWorkflowResourceTail(tail) {
		return "", "", "", false
	}
	workflowID = strings.TrimSpace(tail[1])
	if !isGCPWorkflowsID(workflowID) {
		return "", "", "", false
	}
	return project, location, workflowID, true
}

func parseGCPWorkflowsListRevisionsPath(path string) (project, location, workflowID string, ok bool) {
	project, location, tail, ok := parseGCPWorkflowsLocationTail(path)
	if !ok || !isGCPWorkflowListRevisionsTail(tail) {
		return "", "", "", false
	}
	workflowID, _, _ = strings.Cut(strings.TrimSpace(tail[1]), ":")
	if !isGCPWorkflowsID(workflowID) {
		return "", "", "", false
	}
	return project, location, workflowID, true
}

func parseGCPWorkflowsOperationPath(path string) (project, location, operationID string, ok bool) {
	project, location, tail, ok := parseGCPWorkflowsLocationTail(path)
	if !ok || !isGCPOperationResourceTail(tail) {
		return "", "", "", false
	}
	operationID = strings.TrimSpace(tail[1])
	if operationID == "" {
		return "", "", "", false
	}
	return project, location, operationID, true
}

func parseGCPWorkflowsParent(parent string) (project, location string, ok bool) {
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

func parseGCPWorkflowsWorkflowName(name string) (project, location, workflowID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "workflows" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	workflowID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || !isGCPWorkflowsID(workflowID) {
		return "", "", "", false
	}
	return project, location, workflowID, true
}

func parseGCPWorkflowsOperationName(name string) (project, location, operationID string, ok bool) {
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

func parseGCPWorkflowsPaginationFromQuery(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	if raw := strings.TrimSpace(queryValue(r, "pageSize", "page_size")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 || n > maxPageSize {
			respondGCPWorkflowsInvalidArgument(w, path, fmt.Sprintf("pageSize must be a non-negative integer <= %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = n
	}
	if raw := strings.TrimSpace(queryValue(r, "pageToken", "page_token")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			respondGCPWorkflowsInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = n
	}
	return pageSize, start, true
}

func parseGCPWorkflowsPaginationFromBody(w http.ResponseWriter, path string, body map[string]any, maxPageSize int) (pageSize, start int, ok bool) {
	if raw, exists := gcpWorkflowsLookup(body, "pageSize", "page_size"); exists {
		n, valid := gcpWorkflowsNumberToInt(raw)
		if !valid || n < 0 || n > maxPageSize {
			respondGCPWorkflowsInvalidArgument(w, path, fmt.Sprintf("pageSize must be a non-negative integer <= %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = n
	}
	if raw, exists := gcpWorkflowsLookup(body, "pageToken", "page_token"); exists {
		token := strings.TrimSpace(gcpWorkflowsToString(raw))
		if token != "" {
			n, err := strconv.Atoi(token)
			if err != nil || n < 0 {
				respondGCPWorkflowsInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
				return 0, 0, false
			}
			start = n
		}
	}
	return pageSize, start, true
}

func decodeGCPWorkflowsJSONBodyOptional(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPWorkflowsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPWorkflowsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func decodeGCPWorkflowsJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, ok := decodeGCPWorkflowsJSONBodyOptional(w, r, path)
	if !ok {
		return nil, false
	}
	if len(body) == 0 {
		respondGCPWorkflowsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func gcpWorkflowsWorkflowFromBody(body map[string]any) map[string]any {
	if nested, ok := body["workflow"].(map[string]any); ok && len(nested) > 0 {
		return nested
	}
	return body
}

func validateGCPWorkflowsWorkflow(w http.ResponseWriter, path string, workflow map[string]any, requireName bool) bool {
	name := strings.TrimSpace(gcpWorkflowsString(workflow, "name"))
	if requireName && name == "" {
		respondGCPWorkflowsInvalidArgument(w, path, "workflow.name is required")
		return false
	}
	if name != "" {
		if _, _, _, ok := parseGCPWorkflowsWorkflowName(name); !ok {
			respondGCPWorkflowsInvalidArgument(w, path, "workflow.name is invalid")
			return false
		}
	}

	if desc := gcpWorkflowsString(workflow, "description"); len(desc) > 1000 {
		respondGCPWorkflowsInvalidArgument(w, path, "workflow.description must be <= 1000 characters")
		return false
	}

	sourceContents := strings.TrimSpace(gcpWorkflowsString(workflow, "sourceContents", "source_contents"))
	if sourceContents == "" {
		respondGCPWorkflowsInvalidArgument(w, path, "workflow.sourceContents is required")
		return false
	}

	if labels, exists := workflow["labels"]; exists {
		labelMap, ok := labels.(map[string]any)
		if !ok {
			respondGCPWorkflowsInvalidArgument(w, path, "workflow.labels must be an object")
			return false
		}
		if len(labelMap) > 64 {
			respondGCPWorkflowsInvalidArgument(w, path, "workflow.labels supports at most 64 entries")
			return false
		}
		for key, rawValue := range labelMap {
			key = strings.TrimSpace(key)
			if key == "" || len(key) > 63 {
				respondGCPWorkflowsInvalidArgument(w, path, "workflow.labels key is invalid")
				return false
			}
			if len(strings.TrimSpace(gcpWorkflowsToString(rawValue))) > 63 {
				respondGCPWorkflowsInvalidArgument(w, path, "workflow.labels value is invalid")
				return false
			}
		}
	}

	if raw, exists := gcpWorkflowsLookup(workflow, "callLogLevel", "call_log_level"); exists {
		if !isGCPWorkflowsCallLogLevel(raw) {
			respondGCPWorkflowsInvalidArgument(w, path, "workflow.callLogLevel is invalid")
			return false
		}
	}
	if raw, exists := gcpWorkflowsLookup(workflow, "executionHistoryLevel", "execution_history_level"); exists {
		if !isGCPWorkflowsExecutionHistoryLevel(raw) {
			respondGCPWorkflowsInvalidArgument(w, path, "workflow.executionHistoryLevel is invalid")
			return false
		}
	}

	return true
}

func isGCPWorkflowsCallLogLevel(raw any) bool {
	switch strings.TrimSpace(strings.ToUpper(gcpWorkflowsToString(raw))) {
	case "", "0", "1", "2", "3", "CALL_LOG_LEVEL_UNSPECIFIED", "LOG_ALL_CALLS", "LOG_ERRORS_ONLY", "LOG_NONE":
		return true
	default:
		return false
	}
}

func isGCPWorkflowsExecutionHistoryLevel(raw any) bool {
	switch strings.TrimSpace(strings.ToUpper(gcpWorkflowsToString(raw))) {
	case "", "0", "1", "2", "EXECUTION_HISTORY_LEVEL_UNSPECIFIED", "EXECUTION_HISTORY_BASIC", "EXECUTION_HISTORY_DETAILED":
		return true
	default:
		return false
	}
}

func isGCPWorkflowsSupportedFilter(filter string) bool {
	filter = strings.TrimSpace(filter)
	switch filter {
	case "", `state="ACTIVE"`, `state="UNAVAILABLE"`:
		return true
	default:
		return false
	}
}

func isGCPWorkflowsSupportedOrderBy(orderBy string) bool {
	orderBy = strings.TrimSpace(orderBy)
	switch orderBy {
	case "", "createTime", "createTime desc", "updateTime", "updateTime desc":
		return true
	default:
		return false
	}
}

func isGCPWorkflowsSupportedOperationsFilter(filter string) bool {
	filter = strings.TrimSpace(strings.ToLower(filter))
	switch filter {
	case "", "done=true", "done=false":
		return true
	default:
		return false
	}
}

func isGCPWorkflowsID(id string) bool {
	return gcpWorkflowsIDRegex.MatchString(strings.TrimSpace(id))
}

func gcpWorkflowsLocation(project, location string) map[string]any {
	return map[string]any{
		"name":        fmt.Sprintf("projects/%s/locations/%s", project, location),
		"locationId":  location,
		"displayName": fmt.Sprintf("Workflows %s", strings.ToUpper(strings.ReplaceAll(location, "-", " "))),
		"metadata": map[string]any{
			"service":  "workflows",
			"provider": providerGCP,
		},
	}
}

func gcpWorkflowsWorkflowName(project, location, workflowID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/workflows/%s", project, location, workflowID)
}

func gcpWorkflowsWorkflow(project, location, workflowID, revisionID, state string) map[string]any {
	if revisionID == "" {
		revisionID = "000001-a4d"
	}
	create := gcpWorkflowsReferenceTime
	update := create.Add(10 * time.Minute)
	revisionCreate := update
	name := gcpWorkflowsWorkflowName(project, location, workflowID)
	return map[string]any{
		"name":               name,
		"description":        "Stackyard staged workflow " + workflowID,
		"state":              state,
		"revisionId":         revisionID,
		"createTime":         create.Format(time.RFC3339),
		"updateTime":         update.Format(time.RFC3339),
		"revisionCreateTime": revisionCreate.Format(time.RFC3339),
		"labels": map[string]any{
			"env":      "staged",
			"service":  "workflows",
			"provider": providerGCP,
		},
		"serviceAccount":        fmt.Sprintf("workflow-sa@%s.iam.gserviceaccount.com", project),
		"sourceContents":        "main:\n  params: [input]\n  steps:\n  - return_output:\n      return: ${input}",
		"callLogLevel":          "LOG_ERRORS_ONLY",
		"executionHistoryLevel": "EXECUTION_HISTORY_BASIC",
		"userEnvVars": map[string]any{
			"STACKYARD": "true",
		},
	}
}

func gcpWorkflowsOperation(project, location, operationID, target, verb string, done bool) map[string]any {
	name := fmt.Sprintf("projects/%s/locations/%s/operations/%s", project, location, operationID)
	payload := map[string]any{
		"name": name,
		"metadata": map[string]any{
			"@type":      "type.googleapis.com/google.cloud.workflows.v1.OperationMetadata",
			"createTime": gcpWorkflowsReferenceTime.Format(time.RFC3339),
			"endTime":    gcpWorkflowsReferenceTime.Add(2 * time.Minute).Format(time.RFC3339),
			"target":     target,
			"verb":       verb,
			"apiVersion": "v1",
		},
		"done": done,
	}
	if done {
		switch verb {
		case "delete":
			payload["response"] = map[string]any{
				"@type": "type.googleapis.com/google.protobuf.Empty",
			}
		default:
			payload["response"] = map[string]any{
				"@type": "type.googleapis.com/google.cloud.workflows.v1.Workflow",
				"name":  target,
			}
		}
	}
	return payload
}

func respondGCPWorkflowsList(w http.ResponseWriter, field string, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPWorkflowsOutOfRange(w, path, "pageToken is out of range")
		return true
	}
	end := len(items)
	if pageSize > 0 && start+pageSize < end {
		end = start + pageSize
	}
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	respondJSON(w, http.StatusOK, map[string]any{
		field:           items[start:end],
		"nextPageToken": next,
	})
	return true
}

func respondGCPWorkflowsInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPWorkflowsError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPWorkflowsOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPWorkflowsError(w, http.StatusBadRequest, "OutOfRange", path, message)
}

func respondGCPWorkflowsNotFound(w http.ResponseWriter, path, message string) {
	respondGCPWorkflowsError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPWorkflowsAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPWorkflowsError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPWorkflowsFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPWorkflowsError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPWorkflowsError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_workflows(w http.ResponseWriter, r *http.Request) bool {
	path := normalizeGCPWorkflowsPath(rawRequestPath(r))
	if !isGCPContractProbeRequestForService(r, path, "workflows") {
		return false
	}
	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPWorkflowsInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") != "1" {
		return false
	}
	payload := gcpWorkflowsWorkflow("stackyard", "us-central1", "workflow-probe", "000001-a4d", "ACTIVE")
	payload["service"] = "workflows"
	payload["provider"] = providerGCP
	payload["path"] = path
	respondJSON(w, http.StatusOK, payload)
	return true
}

func gcpWorkflowsLookup(m map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			return val, true
		}
	}
	return nil, false
}

func gcpWorkflowsString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if val, ok := m[key]; ok {
			return strings.TrimSpace(gcpWorkflowsToString(val))
		}
	}
	return ""
}

func gcpWorkflowsToString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int(t)) {
			return strconv.Itoa(int(t))
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int32:
		return strconv.Itoa(int(t))
	case int64:
		return strconv.FormatInt(t, 10)
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func gcpWorkflowsNumberToInt(raw any) (int, bool) {
	switch n := raw.(type) {
	case float64:
		if n != float64(int(n)) {
			return 0, false
		}
		return int(n), true
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(n))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func queryValue(r *http.Request, keys ...string) string {
	if r == nil || r.URL == nil {
		return ""
	}
	q := r.URL.Query()
	for _, key := range keys {
		if value := strings.TrimSpace(q.Get(key)); value != "" {
			return value
		}
	}
	return ""
}
