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

const gcpWorkflowExecutionsGRPCPathPrefix = "/gcp/google.cloud.workflows.executions.v1.Executions/"

var (
	gcpWorkflowExecutionsReferenceTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	gcpWorkflowExecutionsIDRegex         = regexp.MustCompile(`^[A-Za-z](?:[A-Za-z0-9_-]{0,62}[A-Za-z0-9])?$`)
	gcpWorkflowExecutionsLabelKeyRegex   = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,62}$`)
	gcpWorkflowExecutionsLabelValueRegex = regexp.MustCompile(`^[a-z0-9_-]{0,63}$`)
	gcpWorkflowExecutionsFilterRegex     = regexp.MustCompile(`^state\s*=\s*"([A-Z_]+)"$`)
)

func (s *Server) handleGCPWorkflowExecutionsRouter(w http.ResponseWriter, r *http.Request) bool {
	if handleGCPContractProbe_workflow_executions(w, r) {
		return true
	}

	path := normalizeGCPWorkflowExecutionsPath(rawRequestPath(r))
	if strings.HasPrefix(path, gcpWorkflowExecutionsGRPCPathPrefix) {
		return handleGCPWorkflowExecutionsGRPCBridge(w, r, path)
	}

	if !isGCPWorkflowExecutionsPath(path, hasGCPWorkflowExecutionsHint(r)) {
		return false
	}

	switch r.Method {
	case http.MethodGet:
		if handleGCPWorkflowExecutionsListExecutions(w, r, path) {
			return true
		}
		if handleGCPWorkflowExecutionsGetExecution(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	case http.MethodPost:
		if handleGCPWorkflowExecutionsCreateExecution(w, r, path) {
			return true
		}
		if handleGCPWorkflowExecutionsCancelExecution(w, r, path) {
			return true
		}
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	default:
		return false
	}
}

func normalizeGCPWorkflowExecutionsPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "%3A", ":")
	path = strings.ReplaceAll(path, "%3a", ":")
	return path
}

func hasGCPWorkflowExecutionsHint(r *http.Request) bool {
	service := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Stackyard-GCP-Service")))
	switch service {
	case "workflow-executions",
		"workflow_executions",
		"workflow-executions-apiv1",
		"workflow_executions_apiv1",
		"workflowexecutions",
		"workflowexecutions-apiv1",
		"workflows-executions",
		"workflows_executions",
		"gcp-workflow-executions",
		"gcp-workflow-executions-apiv1":
		return true
	}

	ua := strings.ToLower(strings.TrimSpace(r.Header.Get("User-Agent")))
	return strings.Contains(ua, "stackyard-workflow-executions-apiv1") ||
		strings.Contains(ua, "cloud.google.com/go/workflows/executions")
}

func isGCPWorkflowExecutionsPath(path string, includeHint bool) bool {
	if strings.HasPrefix(path, gcpWorkflowExecutionsGRPCPathPrefix) {
		return true
	}
	if _, _, _, ok := parseGCPWorkflowExecutionsCollectionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPWorkflowExecutionPath(path); ok {
		return true
	}
	if _, _, _, _, ok := parseGCPWorkflowExecutionCancelPath(path); ok {
		return true
	}
	return includeHint && strings.Contains(path, "/workflows/") && strings.Contains(path, "/executions")
}

func handleGCPWorkflowExecutionsListExecutions(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, workflowID, ok := parseGCPWorkflowExecutionsCollectionPath(path)
	if !ok {
		return false
	}

	view, maxPageSize, valid := parseGCPWorkflowExecutionsListView(queryValue(r, "view"))
	if !valid {
		respondGCPWorkflowExecutionsInvalidArgument(w, path, "view is invalid")
		return true
	}

	pageSize, start, valid := parseGCPWorkflowExecutionsPaginationFromQuery(w, r, path, maxPageSize)
	if !valid {
		return true
	}

	filterState, valid := parseGCPWorkflowExecutionsStateFilter(queryValue(r, "filter"))
	if !valid {
		respondGCPWorkflowExecutionsInvalidArgument(w, path, "filter is unsupported")
		return true
	}
	orderBy := strings.TrimSpace(queryValue(r, "orderBy", "order_by"))
	if !isGCPWorkflowExecutionsSupportedOrderBy(orderBy) {
		respondGCPWorkflowExecutionsInvalidArgument(w, path, "orderBy is unsupported")
		return true
	}

	items := gcpWorkflowExecutionsExecutionFixtures(project, location, workflowID, view == "FULL")
	if filterState != "" {
		filtered := make([]map[string]any, 0, len(items))
		for _, item := range items {
			if strings.EqualFold(gcpWorkflowExecutionsString(item, "state"), filterState) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	applyGCPWorkflowExecutionsOrdering(items, orderBy)

	return respondGCPWorkflowExecutionsList(w, items, pageSize, start, path)
}

func handleGCPWorkflowExecutionsGetExecution(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, workflowID, executionID, ok := parseGCPWorkflowExecutionPath(path)
	if !ok {
		return false
	}

	view, valid := parseGCPWorkflowExecutionsGetView(queryValue(r, "view"))
	if !valid {
		respondGCPWorkflowExecutionsInvalidArgument(w, path, "view is invalid")
		return true
	}
	if strings.Contains(strings.ToLower(executionID), "missing") {
		respondGCPWorkflowExecutionsNotFound(w, path, "execution not found")
		return true
	}

	state := gcpWorkflowExecutionsStateForExecutionID(executionID)
	respondJSON(w, http.StatusOK, gcpWorkflowExecutionsExecution(project, location, workflowID, executionID, state, view == "FULL"))
	return true
}

func handleGCPWorkflowExecutionsCreateExecution(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, workflowID, ok := parseGCPWorkflowExecutionsCollectionPath(path)
	if !ok {
		return false
	}

	body, valid := decodeGCPWorkflowExecutionsJSONBodyRequired(w, r, path)
	if !valid {
		return true
	}
	execution := gcpWorkflowExecutionsExecutionFromBody(body)
	if len(execution) == 0 {
		respondGCPWorkflowExecutionsInvalidArgument(w, path, "execution is required")
		return true
	}

	executionID, argument, callLogLevel, labels, valid := validateGCPWorkflowExecutionsCreateRequest(w, path, project, location, workflowID, execution)
	if !valid {
		return true
	}
	if strings.Contains(strings.ToLower(executionID), "existing") {
		respondGCPWorkflowExecutionsAlreadyExists(w, path, "execution already exists")
		return true
	}

	payload := gcpWorkflowExecutionsExecution(project, location, workflowID, executionID, "ACTIVE", true)
	if argument != "" {
		payload["argument"] = argument
	}
	if callLogLevel != "" {
		payload["callLogLevel"] = callLogLevel
	}
	if labels != nil {
		payload["labels"] = labels
	}
	respondJSON(w, http.StatusOK, payload)
	return true
}

func handleGCPWorkflowExecutionsCancelExecution(w http.ResponseWriter, r *http.Request, path string) bool {
	project, location, workflowID, executionID, ok := parseGCPWorkflowExecutionCancelPath(path)
	if !ok {
		return false
	}

	body, valid := decodeGCPWorkflowExecutionsJSONBodyOptional(w, r, path)
	if !valid {
		return true
	}
	expectedName := gcpWorkflowExecutionsExecutionName(project, location, workflowID, executionID)
	if got := gcpWorkflowExecutionsString(body, "name"); got != "" && got != expectedName {
		respondGCPWorkflowExecutionsInvalidArgument(w, path, "name must match requested resource")
		return true
	}
	if strings.Contains(strings.ToLower(executionID), "missing") {
		respondGCPWorkflowExecutionsNotFound(w, path, "execution not found")
		return true
	}

	state := gcpWorkflowExecutionsStateForExecutionID(executionID)
	switch state {
	case "SUCCEEDED", "FAILED", "CANCELLED":
		respondGCPWorkflowExecutionsFailedPrecondition(w, path, "execution is not cancellable in terminal state")
		return true
	}

	respondJSON(w, http.StatusOK, gcpWorkflowExecutionsExecution(project, location, workflowID, executionID, "CANCELLED", true))
	return true
}

func handleGCPWorkflowExecutionsGRPCBridge(w http.ResponseWriter, r *http.Request, path string) bool {
	if r.Method != http.MethodPost {
		return false
	}

	body, valid := decodeGCPWorkflowExecutionsJSONBodyOptional(w, r, path)
	if !valid {
		return true
	}

	method := strings.TrimPrefix(path, gcpWorkflowExecutionsGRPCPathPrefix)
	switch method {
	case "ListExecutions":
		project, location, workflowID, ok := parseGCPWorkflowExecutionsParent(gcpWorkflowExecutionsString(body, "parent"))
		if !ok {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "parent is required")
			return true
		}
		view, maxPageSize, ok := parseGCPWorkflowExecutionsListView(gcpWorkflowExecutionsToString(body["view"]))
		if !ok {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "view is invalid")
			return true
		}
		pageSize, start, ok := parseGCPWorkflowExecutionsPaginationFromBody(w, path, body, maxPageSize)
		if !ok {
			return true
		}
		filterState, ok := parseGCPWorkflowExecutionsStateFilter(gcpWorkflowExecutionsString(body, "filter"))
		if !ok {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "filter is unsupported")
			return true
		}
		orderBy := gcpWorkflowExecutionsString(body, "orderBy", "order_by")
		if !isGCPWorkflowExecutionsSupportedOrderBy(orderBy) {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "orderBy is unsupported")
			return true
		}
		items := gcpWorkflowExecutionsExecutionFixtures(project, location, workflowID, view == "FULL")
		if filterState != "" {
			filtered := make([]map[string]any, 0, len(items))
			for _, item := range items {
				if strings.EqualFold(gcpWorkflowExecutionsString(item, "state"), filterState) {
					filtered = append(filtered, item)
				}
			}
			items = filtered
		}
		applyGCPWorkflowExecutionsOrdering(items, orderBy)
		return respondGCPWorkflowExecutionsList(w, items, pageSize, start, path)
	case "CreateExecution":
		project, location, workflowID, ok := parseGCPWorkflowExecutionsParent(gcpWorkflowExecutionsString(body, "parent"))
		if !ok {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "parent is required")
			return true
		}
		execution := gcpWorkflowExecutionsExecutionFromBody(body)
		if len(execution) == 0 {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "execution is required")
			return true
		}
		executionID, argument, callLogLevel, labels, ok := validateGCPWorkflowExecutionsCreateRequest(w, path, project, location, workflowID, execution)
		if !ok {
			return true
		}
		if strings.Contains(strings.ToLower(executionID), "existing") {
			respondGCPWorkflowExecutionsAlreadyExists(w, path, "execution already exists")
			return true
		}
		payload := gcpWorkflowExecutionsExecution(project, location, workflowID, executionID, "ACTIVE", true)
		if argument != "" {
			payload["argument"] = argument
		}
		if callLogLevel != "" {
			payload["callLogLevel"] = callLogLevel
		}
		if labels != nil {
			payload["labels"] = labels
		}
		respondJSON(w, http.StatusOK, payload)
		return true
	case "GetExecution":
		project, location, workflowID, executionID, ok := parseGCPWorkflowExecutionsName(gcpWorkflowExecutionsString(body, "name"))
		if !ok {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "name is required")
			return true
		}
		view, ok := parseGCPWorkflowExecutionsGetView(gcpWorkflowExecutionsToString(body["view"]))
		if !ok {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "view is invalid")
			return true
		}
		if strings.Contains(strings.ToLower(executionID), "missing") {
			respondGCPWorkflowExecutionsNotFound(w, path, "execution not found")
			return true
		}
		respondJSON(w, http.StatusOK, gcpWorkflowExecutionsExecution(project, location, workflowID, executionID, gcpWorkflowExecutionsStateForExecutionID(executionID), view == "FULL"))
		return true
	case "CancelExecution":
		project, location, workflowID, executionID, ok := parseGCPWorkflowExecutionsName(gcpWorkflowExecutionsString(body, "name"))
		if !ok {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "name is required")
			return true
		}
		if strings.Contains(strings.ToLower(executionID), "missing") {
			respondGCPWorkflowExecutionsNotFound(w, path, "execution not found")
			return true
		}
		state := gcpWorkflowExecutionsStateForExecutionID(executionID)
		switch state {
		case "SUCCEEDED", "FAILED", "CANCELLED":
			respondGCPWorkflowExecutionsFailedPrecondition(w, path, "execution is not cancellable in terminal state")
			return true
		}
		respondJSON(w, http.StatusOK, gcpWorkflowExecutionsExecution(project, location, workflowID, executionID, "CANCELLED", true))
		return true
	default:
		respondProviderNotImplemented(w, providerGCP, path)
		return true
	}
}

func parseGCPWorkflowExecutionsCollectionPath(path string) (project, location, workflowID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 9 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "workflows" || parts[8] != "executions" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	workflowID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || !isGCPWorkflowExecutionsID(workflowID) {
		return "", "", "", false
	}
	return project, location, workflowID, true
}

func parseGCPWorkflowExecutionPath(path string) (project, location, workflowID, executionID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "workflows" || parts[8] != "executions" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	workflowID = strings.TrimSpace(parts[7])
	executionID = strings.TrimSpace(parts[9])
	if project == "" || location == "" || !isGCPWorkflowExecutionsID(workflowID) || !isGCPWorkflowExecutionsID(executionID) {
		return "", "", "", "", false
	}
	return project, location, workflowID, executionID, true
}

func parseGCPWorkflowExecutionCancelPath(path string) (project, location, workflowID, executionID string, ok bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 10 || parts[0] != "gcp" || parts[1] != "v1" || parts[2] != "projects" || parts[4] != "locations" || parts[6] != "workflows" || parts[8] != "executions" {
		return "", "", "", "", false
	}
	rawTail := strings.TrimSpace(parts[9])
	id, action, hasAction := strings.Cut(rawTail, ":")
	if !hasAction || action != "cancel" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[3])
	location = strings.TrimSpace(parts[5])
	workflowID = strings.TrimSpace(parts[7])
	executionID = strings.TrimSpace(id)
	if project == "" || location == "" || !isGCPWorkflowExecutionsID(workflowID) || !isGCPWorkflowExecutionsID(executionID) {
		return "", "", "", "", false
	}
	return project, location, workflowID, executionID, true
}

func parseGCPWorkflowExecutionsParent(parent string) (project, location, workflowID string, ok bool) {
	parts := strings.Split(strings.Trim(parent, "/"), "/")
	if len(parts) != 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "workflows" {
		return "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	workflowID = strings.TrimSpace(parts[5])
	if project == "" || location == "" || !isGCPWorkflowExecutionsID(workflowID) {
		return "", "", "", false
	}
	return project, location, workflowID, true
}

func parseGCPWorkflowExecutionsName(name string) (project, location, workflowID, executionID string, ok bool) {
	parts := strings.Split(strings.Trim(name, "/"), "/")
	if len(parts) != 8 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "workflows" || parts[6] != "executions" {
		return "", "", "", "", false
	}
	project = strings.TrimSpace(parts[1])
	location = strings.TrimSpace(parts[3])
	workflowID = strings.TrimSpace(parts[5])
	executionID = strings.TrimSpace(parts[7])
	if project == "" || location == "" || !isGCPWorkflowExecutionsID(workflowID) || !isGCPWorkflowExecutionsID(executionID) {
		return "", "", "", "", false
	}
	return project, location, workflowID, executionID, true
}

func parseGCPWorkflowExecutionsListView(raw string) (view string, maxPageSize int, ok bool) {
	switch strings.TrimSpace(strings.ToUpper(raw)) {
	case "", "0", "EXECUTION_VIEW_UNSPECIFIED", "1", "BASIC":
		return "BASIC", 1000, true
	case "2", "FULL":
		return "FULL", 100, true
	default:
		return "", 0, false
	}
}

func parseGCPWorkflowExecutionsGetView(raw string) (view string, ok bool) {
	switch strings.TrimSpace(strings.ToUpper(raw)) {
	case "", "0", "EXECUTION_VIEW_UNSPECIFIED", "2", "FULL":
		return "FULL", true
	case "1", "BASIC":
		return "BASIC", true
	default:
		return "", false
	}
}

func parseGCPWorkflowExecutionsPaginationFromQuery(w http.ResponseWriter, r *http.Request, path string, maxPageSize int) (pageSize, start int, ok bool) {
	if raw := strings.TrimSpace(queryValue(r, "pageSize", "page_size")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 || value > maxPageSize {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, fmt.Sprintf("pageSize must be a non-negative integer <= %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = value
	}
	if raw := strings.TrimSpace(queryValue(r, "pageToken", "page_token")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
			return 0, 0, false
		}
		start = value
	}
	return pageSize, start, true
}

func parseGCPWorkflowExecutionsPaginationFromBody(w http.ResponseWriter, path string, body map[string]any, maxPageSize int) (pageSize, start int, ok bool) {
	if raw, exists := gcpWorkflowExecutionsLookup(body, "pageSize", "page_size"); exists {
		value, valid := gcpWorkflowExecutionsNumberToInt(raw)
		if !valid || value < 0 || value > maxPageSize {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, fmt.Sprintf("pageSize must be a non-negative integer <= %d", maxPageSize))
			return 0, 0, false
		}
		pageSize = value
	}
	if raw, exists := gcpWorkflowExecutionsLookup(body, "pageToken", "page_token"); exists {
		token := strings.TrimSpace(gcpWorkflowExecutionsToString(raw))
		if token != "" {
			value, err := strconv.Atoi(token)
			if err != nil || value < 0 {
				respondGCPWorkflowExecutionsInvalidArgument(w, path, "pageToken must be a non-negative integer offset")
				return 0, 0, false
			}
			start = value
		}
	}
	return pageSize, start, true
}

func decodeGCPWorkflowExecutionsJSONBodyOptional(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	if r.Body == nil {
		return map[string]any{}, true
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		respondGCPWorkflowExecutionsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, true
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		respondGCPWorkflowExecutionsInvalidArgument(w, path, "request body must be valid JSON")
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}

func decodeGCPWorkflowExecutionsJSONBodyRequired(w http.ResponseWriter, r *http.Request, path string) (map[string]any, bool) {
	body, ok := decodeGCPWorkflowExecutionsJSONBodyOptional(w, r, path)
	if !ok {
		return nil, false
	}
	if len(body) == 0 {
		respondGCPWorkflowExecutionsInvalidArgument(w, path, "request body is required")
		return nil, false
	}
	return body, true
}

func gcpWorkflowExecutionsExecutionFromBody(body map[string]any) map[string]any {
	if nested, ok := body["execution"].(map[string]any); ok && len(nested) > 0 {
		return nested
	}
	return body
}

func validateGCPWorkflowExecutionsCreateRequest(w http.ResponseWriter, path, project, location, workflowID string, execution map[string]any) (executionID, argument, callLogLevel string, labels map[string]any, ok bool) {
	name := strings.TrimSpace(gcpWorkflowExecutionsString(execution, "name"))
	if name != "" {
		reqProject, reqLocation, reqWorkflowID, reqExecutionID, parsed := parseGCPWorkflowExecutionsName(name)
		if !parsed {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "execution.name is invalid")
			return "", "", "", nil, false
		}
		if reqProject != project || reqLocation != location || reqWorkflowID != workflowID {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "execution.name must match parent")
			return "", "", "", nil, false
		}
		executionID = reqExecutionID
	}
	if executionID == "" {
		executionID = "execution-created"
	}

	argument = gcpWorkflowExecutionsString(execution, "argument")
	if len(argument) > 32768 {
		respondGCPWorkflowExecutionsInvalidArgument(w, path, "execution.argument must be <= 32768 characters")
		return "", "", "", nil, false
	}

	if labelsRaw, exists := execution["labels"]; exists {
		labelMap, valid := labelsRaw.(map[string]any)
		if !valid {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "execution.labels must be an object")
			return "", "", "", nil, false
		}
		if len(labelMap) > 64 {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "execution.labels supports at most 64 entries")
			return "", "", "", nil, false
		}
		labels = map[string]any{}
		for key, value := range labelMap {
			trimmedKey := strings.TrimSpace(key)
			trimmedValue := strings.TrimSpace(gcpWorkflowExecutionsToString(value))
			if !gcpWorkflowExecutionsLabelKeyRegex.MatchString(trimmedKey) {
				respondGCPWorkflowExecutionsInvalidArgument(w, path, "execution.labels key is invalid")
				return "", "", "", nil, false
			}
			if !gcpWorkflowExecutionsLabelValueRegex.MatchString(trimmedValue) {
				respondGCPWorkflowExecutionsInvalidArgument(w, path, "execution.labels value is invalid")
				return "", "", "", nil, false
			}
			labels[trimmedKey] = trimmedValue
		}
	}

	rawCallLogLevel := gcpWorkflowExecutionsString(execution, "callLogLevel", "call_log_level")
	if rawCallLogLevel != "" {
		normalized, valid := normalizeGCPWorkflowExecutionsCallLogLevel(rawCallLogLevel)
		if !valid {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "execution.callLogLevel is invalid")
			return "", "", "", nil, false
		}
		callLogLevel = normalized
	}
	if callLogLevel == "" {
		callLogLevel = "LOG_ERRORS_ONLY"
	}

	if raw, exists := gcpWorkflowExecutionsLookup(execution, "state"); exists {
		state := strings.TrimSpace(strings.ToUpper(gcpWorkflowExecutionsToString(raw)))
		if state != "" && state != "STATE_UNSPECIFIED" && state != "0" && state != "ACTIVE" && state != "1" {
			respondGCPWorkflowExecutionsInvalidArgument(w, path, "execution.state is output only")
			return "", "", "", nil, false
		}
	}
	if strings.TrimSpace(gcpWorkflowExecutionsString(execution, "result")) != "" {
		respondGCPWorkflowExecutionsInvalidArgument(w, path, "execution.result is output only")
		return "", "", "", nil, false
	}
	if _, exists := execution["error"]; exists {
		respondGCPWorkflowExecutionsInvalidArgument(w, path, "execution.error is output only")
		return "", "", "", nil, false
	}

	return executionID, argument, callLogLevel, labels, true
}

func parseGCPWorkflowExecutionsStateFilter(filter string) (state string, ok bool) {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return "", true
	}
	matches := gcpWorkflowExecutionsFilterRegex.FindStringSubmatch(filter)
	if len(matches) != 2 {
		return "", false
	}
	state = strings.TrimSpace(strings.ToUpper(matches[1]))
	switch state {
	case "ACTIVE", "SUCCEEDED", "FAILED", "CANCELLED", "QUEUED", "UNAVAILABLE":
		return state, true
	default:
		return "", false
	}
}

func isGCPWorkflowExecutionsSupportedOrderBy(orderBy string) bool {
	switch strings.TrimSpace(strings.ToLower(orderBy)) {
	case "", "starttime", "starttime desc", "starttime asc", "endtime", "endtime desc", "endtime asc", "state", "state desc", "state asc":
		return true
	default:
		return false
	}
}

func applyGCPWorkflowExecutionsOrdering(items []map[string]any, orderBy string) {
	normalized := strings.TrimSpace(strings.ToLower(orderBy))
	if normalized == "" {
		normalized = "starttime desc"
	}
	parts := strings.Fields(normalized)
	field := parts[0]
	desc := len(parts) > 1 && parts[1] == "desc"

	sort.SliceStable(items, func(i, j int) bool {
		cmp := compareGCPWorkflowExecutionsItems(items[i], items[j], field)
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
}

func compareGCPWorkflowExecutionsItems(left, right map[string]any, field string) int {
	switch field {
	case "state":
		return strings.Compare(gcpWorkflowExecutionsString(left, "state"), gcpWorkflowExecutionsString(right, "state"))
	case "endtime":
		return strings.Compare(gcpWorkflowExecutionsString(left, "endTime"), gcpWorkflowExecutionsString(right, "endTime"))
	default:
		return strings.Compare(gcpWorkflowExecutionsString(left, "startTime"), gcpWorkflowExecutionsString(right, "startTime"))
	}
}

func isGCPWorkflowExecutionsID(id string) bool {
	return gcpWorkflowExecutionsIDRegex.MatchString(strings.TrimSpace(id))
}

func normalizeGCPWorkflowExecutionsCallLogLevel(raw string) (string, bool) {
	switch strings.TrimSpace(strings.ToUpper(raw)) {
	case "", "0", "CALL_LOG_LEVEL_UNSPECIFIED":
		return "LOG_ERRORS_ONLY", true
	case "1", "LOG_ALL_CALLS":
		return "LOG_ALL_CALLS", true
	case "2", "LOG_ERRORS_ONLY":
		return "LOG_ERRORS_ONLY", true
	case "3", "LOG_NONE":
		return "LOG_NONE", true
	default:
		return "", false
	}
}

func gcpWorkflowExecutionsExecutionFixtures(project, location, workflowID string, full bool) []map[string]any {
	return []map[string]any{
		gcpWorkflowExecutionsExecution(project, location, workflowID, "execution-1", "ACTIVE", full),
		gcpWorkflowExecutionsExecution(project, location, workflowID, "execution-2", "SUCCEEDED", full),
		gcpWorkflowExecutionsExecution(project, location, workflowID, "execution-3", "FAILED", full),
	}
}

func gcpWorkflowExecutionsExecutionName(project, location, workflowID, executionID string) string {
	return fmt.Sprintf("projects/%s/locations/%s/workflows/%s/executions/%s", project, location, workflowID, executionID)
}

func gcpWorkflowExecutionsStateForExecutionID(executionID string) string {
	id := strings.ToLower(strings.TrimSpace(executionID))
	switch {
	case strings.Contains(id, "succeeded"), strings.Contains(id, "success"), id == "execution-2":
		return "SUCCEEDED"
	case strings.Contains(id, "failed"), id == "execution-3":
		return "FAILED"
	case strings.Contains(id, "cancelled"), strings.Contains(id, "canceled"):
		return "CANCELLED"
	case strings.Contains(id, "queued"):
		return "QUEUED"
	case strings.Contains(id, "unavailable"):
		return "UNAVAILABLE"
	default:
		return "ACTIVE"
	}
}

func gcpWorkflowExecutionsExecution(project, location, workflowID, executionID, state string, full bool) map[string]any {
	index := 1
	switch executionID {
	case "execution-1":
		index = 1
	case "execution-2":
		index = 2
	case "execution-3":
		index = 3
	default:
		index = 4
	}

	start := gcpWorkflowExecutionsReferenceTime.Add(time.Duration(index) * time.Minute)
	end := start.Add(45 * time.Second)
	name := gcpWorkflowExecutionsExecutionName(project, location, workflowID, executionID)

	payload := map[string]any{
		"name":               name,
		"startTime":          start.Format(time.RFC3339),
		"endTime":            end.Format(time.RFC3339),
		"duration":           "45s",
		"state":              state,
		"workflowRevisionId": "000001-a4d",
		"callLogLevel":       "LOG_ERRORS_ONLY",
	}

	if !full {
		return payload
	}

	payload["argument"] = fmt.Sprintf(`{"input":"%s"}`, executionID)
	payload["result"] = ""
	payload["error"] = map[string]any{
		"payload": "",
		"context": "",
		"stackTrace": map[string]any{
			"elements": []any{},
		},
	}
	payload["status"] = map[string]any{
		"currentSteps": []any{
			map[string]any{
				"routine": "main",
				"step":    "run-step",
			},
		},
	}
	payload["labels"] = map[string]any{
		"env":      "staged",
		"service":  "workflow_executions",
		"provider": providerGCP,
	}

	switch state {
	case "SUCCEEDED":
		payload["result"] = `{"status":"ok"}`
	case "FAILED":
		payload["error"] = map[string]any{
			"payload": `{"message":"workflow failed"}`,
			"context": "main.run-step",
			"stackTrace": map[string]any{
				"elements": []any{
					map[string]any{
						"routine": "main",
						"step":    "run-step",
						"position": map[string]any{
							"line":   4,
							"column": 3,
							"length": 8,
						},
					},
				},
			},
		}
	case "CANCELLED":
		payload["error"] = map[string]any{
			"payload": `{"message":"execution cancelled"}`,
			"context": "cancelled by user",
			"stackTrace": map[string]any{
				"elements": []any{},
			},
		}
	case "UNAVAILABLE":
		payload["stateError"] = map[string]any{
			"details": "KMS key permission revoked",
			"type":    "KMS_ERROR",
		}
	}

	return payload
}

func respondGCPWorkflowExecutionsList(w http.ResponseWriter, items []map[string]any, pageSize, start int, path string) bool {
	if start > len(items) {
		respondGCPWorkflowExecutionsOutOfRange(w, path, "pageToken is out of range")
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
		"executions":    items[start:end],
		"nextPageToken": nextPageToken,
	})
	return true
}

func respondGCPWorkflowExecutionsInvalidArgument(w http.ResponseWriter, path, message string) {
	respondGCPWorkflowExecutionsError(w, http.StatusBadRequest, "InvalidArgument", path, message)
}

func respondGCPWorkflowExecutionsOutOfRange(w http.ResponseWriter, path, message string) {
	respondGCPWorkflowExecutionsError(w, http.StatusBadRequest, "OutOfRange", path, message)
}

func respondGCPWorkflowExecutionsNotFound(w http.ResponseWriter, path, message string) {
	respondGCPWorkflowExecutionsError(w, http.StatusNotFound, "NotFound", path, message)
}

func respondGCPWorkflowExecutionsAlreadyExists(w http.ResponseWriter, path, message string) {
	respondGCPWorkflowExecutionsError(w, http.StatusConflict, "AlreadyExists", path, message)
}

func respondGCPWorkflowExecutionsFailedPrecondition(w http.ResponseWriter, path, message string) {
	respondGCPWorkflowExecutionsError(w, http.StatusBadRequest, "FailedPrecondition", path, message)
}

func respondGCPWorkflowExecutionsError(w http.ResponseWriter, statusCode int, code, path, message string) {
	respondJSON(w, statusCode, map[string]any{
		"error":    code,
		"message":  message,
		"provider": providerGCP,
		"path":     path,
	})
}

func handleGCPContractProbe_workflow_executions(w http.ResponseWriter, r *http.Request) bool {
	path := normalizeGCPWorkflowExecutionsPath(rawRequestPath(r))
	if !isGCPContractProbeRequestForService(r, path, "workflow-executions") &&
		!isGCPContractProbeRequestForService(r, path, "workflow_executions") {
		return false
	}

	if r.URL.Query().Get("pageSize") == "bad" {
		respondGCPWorkflowExecutionsInvalidArgument(w, path, "pageSize must be a non-negative integer")
		return true
	}
	if r.URL.Query().Get("typedSuccess") != "1" {
		return false
	}

	payload := gcpWorkflowExecutionsExecution("stackyard", "us-central1", "workflow-probe", "execution-probe", "SUCCEEDED", true)
	payload["service"] = "workflow_executions"
	payload["provider"] = providerGCP
	payload["path"] = path
	respondJSON(w, http.StatusOK, payload)
	return true
}

func gcpWorkflowExecutionsLookup(m map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func gcpWorkflowExecutionsString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return strings.TrimSpace(gcpWorkflowExecutionsToString(value))
		}
	}
	return ""
}

func gcpWorkflowExecutionsToString(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	case float64:
		if typed == float64(int(typed)) {
			return strconv.Itoa(int(typed))
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int32:
		return strconv.Itoa(int(typed))
	case int64:
		return strconv.FormatInt(typed, 10)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func gcpWorkflowExecutionsNumberToInt(raw any) (int, bool) {
	switch number := raw.(type) {
	case float64:
		if number != float64(int(number)) {
			return 0, false
		}
		return int(number), true
	case int:
		return number, true
	case int32:
		return int(number), true
	case int64:
		return int(number), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(number))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}
