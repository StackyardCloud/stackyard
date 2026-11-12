package server

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const azureComputeScheduleAPIVersion = "2025-05-01"

var (
	azureComputeScheduleSubscriptionPattern = regexp.MustCompile(`^[0-9a-fA-F-]{36}$`)
	azureComputeScheduleVMResourcePattern   = regexp.MustCompile(`(?i)^/subscriptions/[0-9a-f-]+/resourcegroups/[^/]+/providers/microsoft\.compute/virtualmachines/[^/]+$`)
)

type azureComputeScheduleOperation struct {
	OperationID     string
	CorrelationID   string
	SubscriptionID  string
	Location        string
	OperationType   string
	ResourceIDs     []string
	State           string
	Schedule        map[string]any
	ExecutionParams map[string]any
	Errors          []map[string]any
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CompletedAt     time.Time
	PollCount       int
	IsPreview       bool
}

type azureComputeScheduleActionRequest struct {
	CorrelationID   string
	ResourceIDs     []string
	Schedule        map[string]any
	ExecutionParams map[string]any
	OperationIDs    []string
}

func (s *Server) handleAzureComputeScheduleRouter(w http.ResponseWriter, r *http.Request) bool {
	path := rawRequestPath(r)
	if !strings.HasPrefix(path, "/azure/providers/") && !strings.HasPrefix(path, "/azure/subscriptions/") {
		return false
	}

	segments := splitPathSegments(strings.TrimPrefix(path, "/azure/"))
	if len(segments) == 0 || !azureComputeScheduleIsPath(segments) {
		return false
	}
	if hasAzureInvalidAPIVersion(r) {
		respondAzureInvalidRequest(w, path, "invalid api-version query parameter")
		return true
	}
	if !azureComputeScheduleAllowedMethod(r.Method) {
		return false
	}

	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ComputeSchedule") &&
		strings.EqualFold(segments[2], "operations") {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return true
		}
		respondJSON(w, http.StatusOK, map[string]any{"value": azureComputeScheduleOperationsFixture()})
		return true
	}

	subscriptionID, location, action, ok := azureComputeScheduleRouteContext(segments)
	if !ok {
		respondAzureImplemented(w, path)
		return true
	}
	if !azureComputeScheduleSubscriptionPattern.MatchString(strings.TrimSpace(subscriptionID)) {
		respondAzureInvalidRequest(w, path, "subscriptionId is invalid")
		return true
	}
	if strings.TrimSpace(location) == "" {
		respondAzureInvalidRequest(w, path, "locationparameter is required")
		return true
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}

	switch strings.ToLower(action) {
	case "virtualmachinessubmitstart":
		s.handleAzureComputeScheduleAction(w, r, path, subscriptionID, location, "submitStart", false)
	case "virtualmachinessubmitdeallocate":
		s.handleAzureComputeScheduleAction(w, r, path, subscriptionID, location, "submitDeallocate", false)
	case "virtualmachinessubmithibernate":
		s.handleAzureComputeScheduleAction(w, r, path, subscriptionID, location, "submitHibernate", false)
	case "virtualmachinesexecutestart":
		s.handleAzureComputeScheduleAction(w, r, path, subscriptionID, location, "executeStart", false)
	case "virtualmachinesexecutedeallocate":
		s.handleAzureComputeScheduleAction(w, r, path, subscriptionID, location, "executeDeallocate", false)
	case "virtualmachinesexecutehibernate":
		s.handleAzureComputeScheduleAction(w, r, path, subscriptionID, location, "executeHibernate", false)
	case "virtualmachinesexecutecreate":
		s.handleAzureComputeScheduleAction(w, r, path, subscriptionID, location, "executeCreate", true)
	case "virtualmachinesexecutedelete":
		s.handleAzureComputeScheduleAction(w, r, path, subscriptionID, location, "executeDelete", true)
	case "virtualmachinesgetoperationstatus":
		s.handleAzureComputeScheduleGetOperationStatus(w, r, path, subscriptionID, location)
	case "virtualmachinesgetoperationerrors":
		s.handleAzureComputeScheduleGetOperationErrors(w, r, path, subscriptionID, location)
	case "virtualmachinescanceloperations":
		s.handleAzureComputeScheduleCancelOperations(w, r, path, subscriptionID, location)
	default:
		respondAzureImplemented(w, path)
	}
	return true
}

func azureComputeScheduleIsPath(segments []string) bool {
	if len(segments) == 3 &&
		strings.EqualFold(segments[0], "providers") &&
		strings.EqualFold(segments[1], "Microsoft.ComputeSchedule") &&
		strings.EqualFold(segments[2], "operations") {
		return true
	}
	_, _, action, ok := azureComputeScheduleRouteContext(segments)
	return ok && strings.TrimSpace(action) != ""
}

func azureComputeScheduleAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost:
		return true
	default:
		return false
	}
}

func azureComputeScheduleRouteContext(segments []string) (subscriptionID, location, action string, ok bool) {
	if len(segments) < 7 || !strings.EqualFold(segments[0], "subscriptions") {
		return "", "", "", false
	}
	subscriptionID = strings.TrimSpace(segments[1])
	for idx := 2; idx+4 < len(segments); idx++ {
		if strings.EqualFold(segments[idx], "providers") &&
			strings.EqualFold(segments[idx+1], "Microsoft.ComputeSchedule") &&
			strings.EqualFold(segments[idx+2], "locations") {
			location = strings.TrimSpace(segments[idx+3])
			if idx+4 < len(segments) {
				action = strings.TrimSpace(segments[idx+4])
				return subscriptionID, location, action, true
			}
		}
	}
	return "", "", "", false
}

func (s *Server) handleAzureComputeScheduleAction(w http.ResponseWriter, r *http.Request, path, subscriptionID, location, operationType string, preview bool) {
	_, payload, ok := azureComputeScheduleReadRequestBody(w, r, path)
	if !ok {
		return
	}
	request, err := azureComputeScheduleParseActionRequest(payload, strings.HasPrefix(strings.ToLower(operationType), "submit"), strings.EqualFold(operationType, "executeCreate") || strings.EqualFold(operationType, "executeDelete"))
	if err != nil {
		respondAzureInvalidRequest(w, path, err.Error())
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	now := time.Now().UTC()
	results := make([]map[string]any, 0, len(request.ResourceIDs))
	for _, resourceID := range request.ResourceIDs {
		op := s.azureComputeScheduleUpsertOperation(subscriptionID, location, operationType, request, resourceID, preview, now)
		results = append(results, azureComputeScheduleOperationResultFixture(resourceID, op, nil))
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"correlationid": request.CorrelationID,
		"results":       results,
	})
}

func (s *Server) handleAzureComputeScheduleGetOperationStatus(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string) {
	_, payload, ok := azureComputeScheduleReadRequestBody(w, r, path)
	if !ok {
		return
	}
	request, err := azureComputeScheduleParseOperationLookupRequest(payload)
	if err != nil {
		respondAzureInvalidRequest(w, path, err.Error())
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	results := make([]map[string]any, 0, len(request.OperationIDs))
	for _, operationID := range request.OperationIDs {
		op := s.azureComputeScheduleOperations[azureComputeScheduleOperationKey(subscriptionID, location, operationID)]
		if op == nil {
			results = append(results, map[string]any{
				"operationId": operationID,
				"error": map[string]any{
					"code":    "OperationNotFound",
					"message": "operation not found",
				},
			})
			continue
		}
		azureComputeScheduleAdvanceOperation(op)
		for _, resourceID := range op.ResourceIDs {
			results = append(results, azureComputeScheduleOperationResultFixture(resourceID, op, nil))
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"correlationid": request.CorrelationID,
		"results":       results,
	})
}

func (s *Server) handleAzureComputeScheduleGetOperationErrors(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string) {
	_, payload, ok := azureComputeScheduleReadRequestBody(w, r, path)
	if !ok {
		return
	}
	request, err := azureComputeScheduleParseOperationLookupRequest(payload)
	if err != nil {
		respondAzureInvalidRequest(w, path, err.Error())
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	results := make([]map[string]any, 0, len(request.OperationIDs))
	for _, operationID := range request.OperationIDs {
		op := s.azureComputeScheduleOperations[azureComputeScheduleOperationKey(subscriptionID, location, operationID)]
		if op == nil {
			results = append(results, map[string]any{
				"operationId": operationID,
				"errors": []map[string]any{{
					"code":    "OperationNotFound",
					"message": "operation not found",
				}},
			})
			continue
		}
		results = append(results, map[string]any{
			"operationId": operationID,
			"errors":      azureComputeScheduleErrorsFixture(op),
		})
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"correlationid": request.CorrelationID,
		"results":       results,
	})
}

func (s *Server) handleAzureComputeScheduleCancelOperations(w http.ResponseWriter, r *http.Request, path, subscriptionID, location string) {
	_, payload, ok := azureComputeScheduleReadRequestBody(w, r, path)
	if !ok {
		return
	}
	request, err := azureComputeScheduleParseOperationLookupRequest(payload)
	if err != nil {
		respondAzureInvalidRequest(w, path, err.Error())
		return
	}

	s.providerStorageMu.Lock()
	defer s.providerStorageMu.Unlock()

	results := make([]map[string]any, 0, len(request.OperationIDs))
	now := time.Now().UTC()
	for _, operationID := range request.OperationIDs {
		op := s.azureComputeScheduleOperations[azureComputeScheduleOperationKey(subscriptionID, location, operationID)]
		if op == nil {
			results = append(results, map[string]any{
				"operationId": operationID,
				"errorCode":   "OperationNotFound",
				"errorDetails": []map[string]any{{
					"code":    "OperationNotFound",
					"message": "operation not found",
				}},
			})
			continue
		}
		op.State = "Canceled"
		op.UpdatedAt = now
		op.CompletedAt = now
		for _, resourceID := range op.ResourceIDs {
			results = append(results, azureComputeScheduleOperationResultFixture(resourceID, op, nil))
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"correlationid": request.CorrelationID,
		"results":       results,
	})
}

func (s *Server) azureComputeScheduleUpsertOperation(subscriptionID, location, operationType string, request azureComputeScheduleActionRequest, resourceID string, preview bool, now time.Time) *azureComputeScheduleOperation {
	resourceKey := azureComputeScheduleHash(strings.ToLower(strings.TrimSpace(resourceID)))
	operationID := azureComputeScheduleHash(subscriptionID, location, request.CorrelationID, operationType, resourceKey)
	key := azureComputeScheduleOperationKey(subscriptionID, location, operationID)
	op := s.azureComputeScheduleOperations[key]
	if op == nil {
		op = &azureComputeScheduleOperation{
			OperationID:     operationID,
			CorrelationID:   request.CorrelationID,
			SubscriptionID:  subscriptionID,
			Location:        location,
			OperationType:   operationType,
			ResourceIDs:     []string{resourceID},
			Schedule:        azureComputeScheduleCloneMap(request.Schedule),
			ExecutionParams: azureComputeScheduleCloneMap(request.ExecutionParams),
			CreatedAt:       now,
			UpdatedAt:       now,
			IsPreview:       preview,
		}
		switch {
		case strings.HasPrefix(strings.ToLower(operationType), "submit"):
			op.State = "Scheduled"
		case strings.HasPrefix(strings.ToLower(operationType), "execute"):
			op.State = "InProgress"
		default:
			op.State = "Scheduled"
		}
		if strings.Contains(strings.ToLower(resourceID), "fail") {
			op.Errors = []map[string]any{{
				"resourceId": resourceID,
				"code":       "SyntheticFailure",
				"message":    "synthetic resource failure seeded by resource id",
			}}
		}
		s.azureComputeScheduleOperations[key] = op
		return op
	}
	if !azureComputeScheduleContains(op.ResourceIDs, resourceID) {
		op.ResourceIDs = append(op.ResourceIDs, resourceID)
		sort.Strings(op.ResourceIDs)
	}
	op.UpdatedAt = now
	return op
}

func azureComputeScheduleAdvanceOperation(op *azureComputeScheduleOperation) {
	op.PollCount++
	op.UpdatedAt = time.Now().UTC()
	if op.State == "Canceled" || op.State == "Failed" || op.State == "Succeeded" {
		if op.CompletedAt.IsZero() && (op.State == "Canceled" || op.State == "Failed" || op.State == "Succeeded") {
			op.CompletedAt = op.UpdatedAt
		}
		return
	}
	if len(op.Errors) > 0 {
		op.State = "Failed"
		op.CompletedAt = op.UpdatedAt
		return
	}
	switch op.State {
	case "Scheduled":
		if op.PollCount >= 1 {
			op.State = "InProgress"
		}
	case "InProgress":
		if op.PollCount >= 2 {
			op.State = "Succeeded"
			op.CompletedAt = op.UpdatedAt
		}
	}
}

func azureComputeScheduleOperationsFixture() []map[string]any {
	return []map[string]any{
		azureComputeScheduleOperationFixture("Microsoft.ComputeSchedule/operations/read", "Operations", "List Compute Schedule operations"),
		azureComputeScheduleOperationFixture("Microsoft.ComputeSchedule/virtualMachinesSubmitStart/action", "Scheduled Actions", "Submit virtual machines start"),
		azureComputeScheduleOperationFixture("Microsoft.ComputeSchedule/virtualMachinesSubmitDeallocate/action", "Scheduled Actions", "Submit virtual machines deallocate"),
		azureComputeScheduleOperationFixture("Microsoft.ComputeSchedule/virtualMachinesSubmitHibernate/action", "Scheduled Actions", "Submit virtual machines hibernate"),
		azureComputeScheduleOperationFixture("Microsoft.ComputeSchedule/virtualMachinesExecuteStart/action", "Scheduled Actions", "Execute virtual machines start"),
		azureComputeScheduleOperationFixture("Microsoft.ComputeSchedule/virtualMachinesExecuteDeallocate/action", "Scheduled Actions", "Execute virtual machines deallocate"),
		azureComputeScheduleOperationFixture("Microsoft.ComputeSchedule/virtualMachinesExecuteHibernate/action", "Scheduled Actions", "Execute virtual machines hibernate"),
		azureComputeScheduleOperationFixture("Microsoft.ComputeSchedule/virtualMachinesCancelOperations/action", "Scheduled Actions", "Cancel virtual machine operations"),
		azureComputeScheduleOperationFixture("Microsoft.ComputeSchedule/virtualMachinesGetOperationStatus/action", "Scheduled Actions", "Get virtual machine operation status"),
		azureComputeScheduleOperationFixture("Microsoft.ComputeSchedule/virtualMachinesGetOperationErrors/action", "Scheduled Actions", "Get virtual machine operation errors"),
		azureComputeScheduleOperationFixture("Microsoft.ComputeSchedule/virtualMachinesExecuteCreate/action", "Scheduled Actions", "Execute virtual machine create [Private Preview]"),
		azureComputeScheduleOperationFixture("Microsoft.ComputeSchedule/virtualMachinesExecuteDelete/action", "Scheduled Actions", "Execute virtual machine delete [Private Preview]"),
	}
}

func azureComputeScheduleOperationFixture(name, resource, operation string) map[string]any {
	return map[string]any{
		"name":       name,
		"origin":     "system",
		"actionType": "Internal",
		"display": map[string]any{
			"provider":  "Microsoft Compute Schedule",
			"resource":  resource,
			"operation": operation,
		},
		"isDataAction": false,
	}
}

func azureComputeScheduleOperationResultFixture(resourceID string, op *azureComputeScheduleOperation, explicitError map[string]any) map[string]any {
	result := map[string]any{
		"resourceId": resourceID,
		"operation": map[string]any{
			"operationId":   op.OperationID,
			"operationType": op.OperationType,
			"state":         op.State,
			"createdAt":     op.CreatedAt.Format(time.RFC3339),
			"updatedAt":     op.UpdatedAt.Format(time.RFC3339),
		},
	}
	if !op.CompletedAt.IsZero() {
		result["operation"].(map[string]any)["completedAt"] = op.CompletedAt.Format(time.RFC3339)
	}
	if op.IsPreview {
		result["operation"].(map[string]any)["preview"] = true
	}
	if explicitError != nil {
		result["errorCode"] = azureComputeScheduleString(explicitError["code"])
		result["errorDetails"] = []map[string]any{explicitError}
	} else if err := azureComputeScheduleResourceError(op, resourceID); err != nil {
		result["errorCode"] = azureComputeScheduleString(err["code"])
		result["errorDetails"] = []map[string]any{err}
	}
	return result
}

func azureComputeScheduleErrorsFixture(op *azureComputeScheduleOperation) []map[string]any {
	if len(op.Errors) == 0 {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(op.Errors))
	for _, err := range op.Errors {
		out = append(out, azureComputeScheduleCloneMap(err))
	}
	return out
}

func azureComputeScheduleResourceError(op *azureComputeScheduleOperation, resourceID string) map[string]any {
	for _, err := range op.Errors {
		if strings.EqualFold(azureComputeScheduleString(err["resourceId"]), resourceID) {
			return azureComputeScheduleCloneMap(err)
		}
	}
	return nil
}

func azureComputeScheduleParseActionRequest(payload map[string]any, requireSchedule bool, allowNonVMResources bool) (azureComputeScheduleActionRequest, error) {
	request := azureComputeScheduleActionRequest{
		CorrelationID:   azureComputeScheduleCorrelationID(payload),
		ExecutionParams: azureComputeScheduleObject(payload["executionParameters"]),
		Schedule:        azureComputeScheduleObject(payload["schedule"]),
	}
	if request.CorrelationID == "" {
		return request, fmt.Errorf("correlationid is required")
	}
	resourceContainer := azureComputeScheduleObject(payload["resources"])
	request.ResourceIDs = azureComputeScheduleStringSlice(resourceContainer["ids"])
	if len(request.ResourceIDs) == 0 {
		return request, fmt.Errorf("resources.ids must include at least one resource id")
	}
	for _, resourceID := range request.ResourceIDs {
		if !allowNonVMResources && !azureComputeScheduleVMResourcePattern.MatchString(resourceID) {
			return request, fmt.Errorf("resources.ids must contain Microsoft.Compute virtual machine resource ids")
		}
	}
	if requireSchedule {
		if len(request.Schedule) == 0 {
			return request, fmt.Errorf("schedule is required")
		}
		if azureComputeScheduleScheduleField(request.Schedule, "deadline") == "" {
			return request, fmt.Errorf("schedule.deadline is required")
		}
	}
	return request, nil
}

func azureComputeScheduleParseOperationLookupRequest(payload map[string]any) (azureComputeScheduleActionRequest, error) {
	request := azureComputeScheduleActionRequest{
		CorrelationID: azureComputeScheduleCorrelationID(payload),
		OperationIDs:  azureComputeScheduleStringSlice(payload["operationIds"]),
	}
	if request.CorrelationID == "" {
		return request, fmt.Errorf("correlationid is required")
	}
	if len(request.OperationIDs) == 0 {
		return request, fmt.Errorf("operationIds must include at least one operation id")
	}
	return request, nil
}

func azureComputeScheduleReadRequestBody(w http.ResponseWriter, r *http.Request, path string) ([]byte, map[string]any, bool) {
	if r.Body == nil {
		return nil, map[string]any{}, true
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondAzureInvalidRequest(w, path, "failed to read request body")
		return nil, nil, false
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return body, map[string]any{}, true
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		respondAzureInvalidRequest(w, path, "request body must be valid JSON")
		return nil, nil, false
	}
	return body, payload, true
}

func azureComputeScheduleOperationKey(subscriptionID, location, operationID string) string {
	return strings.ToLower(strings.TrimSpace(subscriptionID)) + "::" +
		strings.ToLower(strings.TrimSpace(location)) + "::" +
		strings.ToLower(strings.TrimSpace(operationID))
}

func azureComputeScheduleCorrelationID(payload map[string]any) string {
	for _, key := range []string{"correlationid", "correlationId"} {
		if value := azureComputeScheduleString(payload[key]); value != "" {
			return value
		}
	}
	return ""
}

func azureComputeScheduleScheduleField(schedule map[string]any, key string) string {
	if value := azureComputeScheduleString(schedule[key]); value != "" {
		return value
	}
	switch strings.ToLower(key) {
	case "deadline":
		for _, candidate := range []string{"deadline", "deadlineTime", "requestedDeadline"} {
			if value := azureComputeScheduleString(schedule[candidate]); value != "" {
				return value
			}
		}
	}
	return ""
}

func azureComputeScheduleObject(value any) map[string]any {
	if object, ok := value.(map[string]any); ok {
		return object
	}
	return map[string]any{}
}

func azureComputeScheduleString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func azureComputeScheduleStringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, item := range raw {
		s := azureComputeScheduleString(item)
		if s == "" {
			continue
		}
		key := strings.ToLower(s)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}
	return out
}

func azureComputeScheduleCloneMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}

func azureComputeScheduleContains(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func azureComputeScheduleHash(parts ...string) string {
	h := fnv.New64a()
	for _, part := range parts {
		_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(part))))
		_, _ = h.Write([]byte{0})
	}
	return "op-" + strconv.FormatUint(h.Sum64(), 16)
}
