package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	batchsvc "github.com/stackyard/stackyard/internal/services/batch"
)

type batchError struct {
	Type    string `json:"__type,omitempty"`
	Message string `json:"message"`
}

func (s *Server) handleBatchJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isBatchJSONCandidate(r) {
		return false
	}

	ok, status, code, msg, _ := s.validateSigV4WithService(r, "batch")
	if !ok {
		respondBatchError(w, status, code, msg)
		return true
	}

	op, params, matched := matchBatchOperation(r.Method, r.URL.Path)
	if !matched {
		respondBatchError(w, http.StatusNotFound, "ClientException", "operation not found")
		return true
	}

	payload, err := parseBatchPayload(r)
	if err != nil {
		respondBatchError(w, http.StatusBadRequest, "ClientException", "invalid JSON body")
		return true
	}

	switch op.Name {
	case "CreateComputeEnvironment":
		name := batchString(payload["computeEnvironmentName"])
		ceType := batchString(payload["type"])
		state := batchString(payload["state"])
		unmanaged, _ := batchInt32(payload["unmanagedvCpus"])
		serviceRole := batchString(payload["serviceRole"])
		ce, err := s.batch.CreateComputeEnvironment(name, ceType, state, unmanaged, serviceRole, batchStringMap(payload["tags"]))
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{
			"computeEnvironmentArn":  ce.ARN,
			"computeEnvironmentName": ce.Name,
		})
		return true
	case "CreateConsumableResource":
		totalQuantity, _ := batchInt64(payload["totalQuantity"])
		cr, err := s.batch.CreateConsumableResource(
			batchString(payload["consumableResourceName"]),
			batchString(payload["resourceType"]),
			totalQuantity,
			batchStringMap(payload["tags"]),
		)
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{
			"consumableResourceArn":  cr.ARN,
			"consumableResourceName": cr.Name,
		})
		return true
	case "DescribeComputeEnvironments":
		ids := batchStringSlice(payload["computeEnvironments"])
		ces := s.batch.DescribeComputeEnvironments(ids)
		out := make([]map[string]any, 0, len(ces))
		for _, ce := range ces {
			out = append(out, batchComputeEnvironmentPayload(ce))
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{"computeEnvironments": out})
		return true
	case "DescribeConsumableResource":
		cr, err := s.batch.DescribeConsumableResource(batchString(payload["consumableResource"]))
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, batchConsumableResourceDetailPayload(cr))
		return true
	case "UpdateComputeEnvironment":
		id := batchString(payload["computeEnvironment"])
		state, hasState := batchOptionalString(payload, "state")
		serviceRole, hasServiceRole := batchOptionalString(payload, "serviceRole")
		unmanaged, hasUnmanaged := batchInt32(payload["unmanagedvCpus"])
		var statePtr *string
		if hasState {
			statePtr = &state
		}
		var serviceRolePtr *string
		if hasServiceRole {
			serviceRolePtr = &serviceRole
		}
		var unmanagedPtr *int32
		if hasUnmanaged {
			unmanagedPtr = &unmanaged
		}
		ce, err := s.batch.UpdateComputeEnvironment(id, statePtr, unmanagedPtr, serviceRolePtr)
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{
			"computeEnvironmentArn":  ce.ARN,
			"computeEnvironmentName": ce.Name,
		})
		return true
	case "UpdateConsumableResource":
		quantity, _ := batchInt64(payload["quantity"])
		cr, err := s.batch.UpdateConsumableResource(
			batchString(payload["consumableResource"]),
			batchString(payload["operation"]),
			quantity,
		)
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{
			"consumableResourceArn":  cr.ARN,
			"consumableResourceName": cr.Name,
			"totalQuantity":          cr.TotalQuantity,
		})
		return true
	case "DeleteComputeEnvironment":
		if err := s.batch.DeleteComputeEnvironment(batchString(payload["computeEnvironment"])); err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DeleteConsumableResource":
		if err := s.batch.DeleteConsumableResource(batchString(payload["consumableResource"])); err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{})
		return true
	case "CreateJobQueue":
		name := batchString(payload["jobQueueName"])
		priority, _ := batchInt32(payload["priority"])
		state := batchString(payload["state"])
		order := batchComputeEnvironmentOrders(payload["computeEnvironmentOrder"])
		schedArn := batchString(payload["schedulingPolicyArn"])
		jq, err := s.batch.CreateJobQueue(name, priority, state, order, schedArn, batchStringMap(payload["tags"]))
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{
			"jobQueueArn":  jq.ARN,
			"jobQueueName": jq.Name,
		})
		return true
	case "DescribeJobQueues":
		ids := batchStringSlice(payload["jobQueues"])
		queues := s.batch.DescribeJobQueues(ids)
		out := make([]map[string]any, 0, len(queues))
		for _, jq := range queues {
			out = append(out, batchJobQueuePayload(jq))
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{"jobQueues": out})
		return true
	case "UpdateJobQueue":
		id := batchString(payload["jobQueue"])
		state, hasState := batchOptionalString(payload, "state")
		priority, hasPriority := batchInt32(payload["priority"])
		ceOrder, hasOrder := batchOptionalComputeEnvironmentOrders(payload, "computeEnvironmentOrder")
		sched, hasSched := batchOptionalString(payload, "schedulingPolicyArn")

		var statePtr *string
		if hasState {
			statePtr = &state
		}
		var priorityPtr *int32
		if hasPriority {
			priorityPtr = &priority
		}
		var ceOrderInput []batchsvc.ComputeEnvironmentOrder
		if hasOrder {
			ceOrderInput = ceOrder
		}
		var schedPtr *string
		if hasSched {
			schedPtr = &sched
		}

		jq, err := s.batch.UpdateJobQueue(id, statePtr, priorityPtr, ceOrderInput, schedPtr)
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{
			"jobQueueArn":  jq.ARN,
			"jobQueueName": jq.Name,
		})
		return true
	case "DeleteJobQueue":
		if err := s.batch.DeleteJobQueue(batchString(payload["jobQueue"])); err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{})
		return true
	case "CreateSchedulingPolicy":
		sp, err := s.batch.CreateSchedulingPolicy(batchString(payload["name"]), batchStringMap(payload["tags"]))
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{"arn": sp.ARN, "name": sp.Name})
		return true
	case "CreateServiceEnvironment":
		se, err := s.batch.CreateServiceEnvironment(
			batchString(payload["serviceEnvironmentName"]),
			batchString(payload["serviceEnvironmentType"]),
			batchString(payload["state"]),
			batchServiceEnvironmentCapacities(payload["capacityLimits"]),
			batchStringMap(payload["tags"]),
		)
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{
			"serviceEnvironmentArn":  se.ARN,
			"serviceEnvironmentName": se.Name,
		})
		return true
	case "DescribeSchedulingPolicies":
		ids := batchStringSlice(payload["arns"])
		items := s.batch.DescribeSchedulingPolicies(ids)
		out := make([]map[string]any, 0, len(items))
		for _, sp := range items {
			out = append(out, batchSchedulingPolicyPayload(sp))
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{"schedulingPolicies": out})
		return true
	case "DescribeServiceEnvironments":
		items, nextToken := s.batch.DescribeServiceEnvironments(
			batchStringSlice(payload["serviceEnvironments"]),
			parseBatchMaxResults(payload),
			parseBatchNextToken(payload),
		)
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			out = append(out, batchServiceEnvironmentPayload(item))
		}
		resp := map[string]any{"serviceEnvironments": out}
		if nextToken != "" {
			resp["nextToken"] = nextToken
		}
		respondBatchJSON(w, http.StatusOK, resp)
		return true
	case "ListSchedulingPolicies":
		items := s.batch.DescribeSchedulingPolicies(nil)
		out := make([]map[string]any, 0, len(items))
		for _, sp := range items {
			out = append(out, batchSchedulingPolicyPayload(sp))
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{"schedulingPolicies": out})
		return true
	case "UpdateSchedulingPolicy":
		sp, err := s.batch.UpdateSchedulingPolicy(batchString(payload["arn"]), batchStringMap(payload["tags"]))
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{"arn": sp.ARN, "name": sp.Name})
		return true
	case "UpdateServiceEnvironment":
		state, hasState := batchOptionalString(payload, "state")
		var statePtr *string
		if hasState {
			statePtr = &state
		}
		capacity, hasCapacity := batchOptionalServiceEnvironmentCapacities(payload, "capacityLimits")
		var capInput []batchsvc.ServiceEnvironmentCapacity
		if hasCapacity {
			capInput = capacity
		}
		se, err := s.batch.UpdateServiceEnvironment(batchString(payload["serviceEnvironment"]), statePtr, capInput)
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{
			"serviceEnvironmentArn":  se.ARN,
			"serviceEnvironmentName": se.Name,
		})
		return true
	case "DeleteSchedulingPolicy":
		if err := s.batch.DeleteSchedulingPolicy(batchString(payload["arn"])); err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DeleteServiceEnvironment":
		if err := s.batch.DeleteServiceEnvironment(batchString(payload["serviceEnvironment"])); err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{})
		return true
	case "RegisterJobDefinition":
		jd, err := s.batch.RegisterJobDefinition(
			batchString(payload["jobDefinitionName"]),
			batchString(payload["type"]),
			batchStringMap(payload["parameters"]),
			batchStringMap(payload["tags"]),
		)
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{
			"jobDefinitionArn":  jd.ARN,
			"jobDefinitionName": jd.Name,
			"revision":          jd.Revision,
		})
		return true
	case "DescribeJobDefinitions":
		ids := batchStringSlice(payload["jobDefinitions"])
		status := batchString(payload["status"])
		defs := s.batch.DescribeJobDefinitions(ids, status)
		out := make([]map[string]any, 0, len(defs))
		for _, jd := range defs {
			out = append(out, batchJobDefinitionPayload(jd))
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{"jobDefinitions": out})
		return true
	case "DeregisterJobDefinition":
		if err := s.batch.DeregisterJobDefinition(batchString(payload["jobDefinition"])); err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{})
		return true
	case "SubmitJob":
		job, err := s.batch.SubmitJobWithOptions(
			batchString(payload["jobName"]),
			batchString(payload["jobQueue"]),
			batchString(payload["jobDefinition"]),
			batchStringMap(payload["parameters"]),
			batchStringMap(payload["tags"]),
			batchConsumableRequirements(payload["consumableResourcePropertiesOverride"]),
			batchString(payload["shareIdentifier"]),
			batchInt64OrZero(payload["quantity"]),
		)
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{"jobArn": job.ARN, "jobId": job.ID, "jobName": job.Name})
		return true
	case "SubmitServiceJob":
		retryStrategy := batchServiceJobRetryStrategy(payload["retryStrategy"])
		timeoutConfig := batchServiceJobTimeout(payload["timeoutConfig"])
		schedulingPrio, _ := batchInt32(payload["schedulingPriority"])
		job, err := s.batch.SubmitServiceJob(
			batchString(payload["jobName"]),
			batchString(payload["jobQueue"]),
			batchString(payload["serviceJobType"]),
			batchString(payload["serviceRequestPayload"]),
			schedulingPrio,
			batchString(payload["shareIdentifier"]),
			retryStrategy,
			timeoutConfig,
			batchStringMap(payload["tags"]),
		)
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{
			"jobArn":  job.ARN,
			"jobId":   job.ID,
			"jobName": job.Name,
		})
		return true
	case "DescribeJobs":
		ids := batchStringSlice(payload["jobs"])
		jobs := s.batch.DescribeJobs(ids)
		out := make([]map[string]any, 0, len(jobs))
		for _, job := range jobs {
			out = append(out, batchJobPayload(job))
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{"jobs": out})
		return true
	case "DescribeServiceJob":
		job, err := s.batch.DescribeServiceJob(batchString(payload["jobId"]))
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, batchServiceJobPayload(job))
		return true
	case "ListJobs":
		jobs := s.batch.ListJobs(batchString(payload["jobQueue"]), batchString(payload["jobStatus"]))
		out := make([]map[string]any, 0, len(jobs))
		for _, job := range jobs {
			out = append(out, batchJobSummaryPayload(job))
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{"jobSummaryList": out})
		return true
	case "ListJobsByConsumableResource":
		items, nextToken, err := s.batch.ListJobsByConsumableResource(
			batchString(payload["consumableResource"]),
			parseBatchMaxResults(payload),
			parseBatchNextToken(payload),
		)
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			out = append(out, batchListJobsByConsumablePayload(item))
		}
		resp := map[string]any{"jobs": out}
		if nextToken != "" {
			resp["nextToken"] = nextToken
		}
		respondBatchJSON(w, http.StatusOK, resp)
		return true
	case "ListConsumableResources":
		items, nextToken := s.batch.ListConsumableResources(parseBatchMaxResults(payload), parseBatchNextToken(payload))
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			out = append(out, batchConsumableResourceSummaryPayload(item))
		}
		resp := map[string]any{"consumableResources": out}
		if nextToken != "" {
			resp["nextToken"] = nextToken
		}
		respondBatchJSON(w, http.StatusOK, resp)
		return true
	case "ListServiceJobs":
		items, nextToken := s.batch.ListServiceJobs(
			batchString(payload["jobQueue"]),
			batchString(payload["jobStatus"]),
			parseBatchMaxResults(payload),
			parseBatchNextToken(payload),
		)
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			out = append(out, batchServiceJobSummaryPayload(item))
		}
		resp := map[string]any{"jobSummaryList": out}
		if nextToken != "" {
			resp["nextToken"] = nextToken
		}
		respondBatchJSON(w, http.StatusOK, resp)
		return true
	case "GetJobQueueSnapshot":
		snapshot, err := s.batch.GetJobQueueSnapshot(batchString(payload["jobQueue"]))
		if err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		jobs := make([]map[string]any, 0, len(snapshot.Jobs))
		for _, job := range snapshot.Jobs {
			jobs = append(jobs, map[string]any{
				"jobArn":                 job.JobARN,
				"earliestTimeAtPosition": batchTimeMillis(job.EarliestTimeAtPosition),
			})
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{
			"frontOfQueue": map[string]any{
				"jobs":          jobs,
				"lastUpdatedAt": batchTimeMillis(snapshot.LastUpdatedAt),
			},
		})
		return true
	case "CancelJob":
		if err := s.batch.CancelJob(batchString(payload["jobId"]), batchString(payload["reason"])); err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{})
		return true
	case "TerminateJob":
		if err := s.batch.TerminateJob(batchString(payload["jobId"]), batchString(payload["reason"])); err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{})
		return true
	case "TerminateServiceJob":
		if err := s.batch.TerminateServiceJob(batchString(payload["jobId"]), batchString(payload["reason"])); err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{})
		return true
	case "TagResource":
		if err := s.batch.TagResource(params["resourceArn"], batchStringMap(payload["tags"])); err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{})
		return true
	case "ListTagsForResource":
		tags, ok := s.batch.ListTagsForResource(params["resourceArn"])
		if !ok {
			respondBatchError(w, http.StatusNotFound, "ClientException", "resource not found")
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{"tags": tags})
		return true
	case "UntagResource":
		if err := s.batch.UntagResource(params["resourceArn"], batchTagKeys(r)); err != nil {
			respondBatchErrorForErr(w, err)
			return true
		}
		respondBatchJSON(w, http.StatusOK, map[string]any{})
		return true
	default:
		respondBatchError(w, http.StatusNotImplemented, "NotImplementedException", op.Name+" is not implemented")
		return true
	}
}

func respondBatchJSON(w http.ResponseWriter, status int, body any) {
	respondJSON(w, status, body)
}

func respondBatchError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("X-Amzn-ErrorType", code)
	respondJSON(w, status, batchError{Type: code, Message: msg})
}

func respondBatchErrorForErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, batchsvc.ErrInvalidParameter):
		respondBatchError(w, http.StatusBadRequest, "ClientException", err.Error())
	case errors.Is(err, batchsvc.ErrAlreadyExists):
		respondBatchError(w, http.StatusBadRequest, "ClientException", err.Error())
	case errors.Is(err, batchsvc.ErrNotFound):
		respondBatchError(w, http.StatusNotFound, "ClientException", err.Error())
	default:
		respondBatchError(w, http.StatusBadRequest, "ClientException", err.Error())
	}
}

func isBatchJSONCandidate(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodPost, http.MethodDelete:
	default:
		return false
	}
	path := normalizeBatchPath(r.URL.Path)
	return strings.HasPrefix(path, "/v1/")
}

func matchBatchOperation(method, path string) (batchOperation, map[string]string, bool) {
	for _, op := range batchOperations {
		if method != op.Method {
			continue
		}
		if params, ok := matchBatchPathPattern(op.Pattern, path); ok {
			return op, params, true
		}
	}
	return batchOperation{}, nil, false
}

func matchBatchPathPattern(pattern, actual string) (map[string]string, bool) {
	patternSegs := splitBatchPath(pattern)
	actualSegs := splitBatchPath(actual)
	params := map[string]string{}
	pi := 0
	ai := 0
	for pi < len(patternSegs) {
		if ai >= len(actualSegs) {
			return nil, false
		}
		seg := patternSegs[pi]
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") && len(seg) > 2 {
			name := seg[1 : len(seg)-1]
			raw := actualSegs[ai]
			if pi == len(patternSegs)-1 {
				raw = strings.Join(actualSegs[ai:], "/")
				ai = len(actualSegs)
			} else {
				ai++
			}
			value, err := url.PathUnescape(raw)
			if err != nil {
				value = raw
			}
			params[name] = value
			pi++
			continue
		}
		if seg != actualSegs[ai] {
			return nil, false
		}
		pi++
		ai++
	}
	if ai != len(actualSegs) {
		return nil, false
	}
	return params, true
}

func normalizeBatchPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
		if path == "" {
			return "/"
		}
	}
	return path
}

func splitBatchPath(path string) []string {
	path = normalizeBatchPath(path)
	if path == "/" {
		return nil
	}
	return strings.Split(strings.TrimPrefix(path, "/"), "/")
}

func parseBatchPayload(r *http.Request) (map[string]any, error) {
	if r.Method == http.MethodGet || r.Method == http.MethodDelete {
		return map[string]any{}, nil
	}
	body, err := readBodyBytes(r)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}, nil
	}
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return nil, err
	}
	obj, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}
	return obj, nil
}

func batchComputeEnvironmentPayload(ce batchsvc.ComputeEnvironment) map[string]any {
	return map[string]any{
		"computeEnvironmentArn":  ce.ARN,
		"computeEnvironmentName": ce.Name,
		"type":                   ce.Type,
		"state":                  ce.State,
		"status":                 ce.Status,
		"statusReason":           ce.StatusReason,
		"serviceRole":            ce.ServiceRole,
		"unmanagedvCpus":         ce.UnmanagedVCPUs,
		"tags":                   ce.Tags,
	}
}

func batchJobQueuePayload(jq batchsvc.JobQueue) map[string]any {
	order := make([]map[string]any, 0, len(jq.ComputeEnvironmentOrder))
	for _, item := range jq.ComputeEnvironmentOrder {
		order = append(order, map[string]any{
			"order":              item.Order,
			"computeEnvironment": item.ComputeEnvironment,
		})
	}
	return map[string]any{
		"jobQueueArn":             jq.ARN,
		"jobQueueName":            jq.Name,
		"state":                   jq.State,
		"status":                  jq.Status,
		"statusReason":            jq.StatusReason,
		"priority":                jq.Priority,
		"computeEnvironmentOrder": order,
		"schedulingPolicyArn":     jq.SchedulingPolicyARN,
		"tags":                    jq.Tags,
	}
}

func batchSchedulingPolicyPayload(sp batchsvc.SchedulingPolicy) map[string]any {
	return map[string]any{
		"arn":  sp.ARN,
		"name": sp.Name,
		"tags": sp.Tags,
	}
}

func batchJobDefinitionPayload(jd batchsvc.JobDefinition) map[string]any {
	return map[string]any{
		"jobDefinitionArn":  jd.ARN,
		"jobDefinitionName": jd.Name,
		"revision":          jd.Revision,
		"status":            jd.Status,
		"type":              strings.ToLower(jd.Type),
		"parameters":        jd.Parameters,
		"tags":              jd.Tags,
	}
}

func batchJobPayload(job batchsvc.Job) map[string]any {
	out := map[string]any{
		"jobArn":        job.ARN,
		"jobId":         job.ID,
		"jobName":       job.Name,
		"jobDefinition": job.Definition,
		"jobQueue":      job.Queue,
		"status":        job.Status,
		"statusReason":  job.StatusReason,
		"createdAt":     batchTimeMillis(job.CreatedAt),
		"parameters":    job.Parameters,
		"tags":          job.Tags,
	}
	startedAt := job.CreatedAt
	if job.StartedAt != nil {
		startedAt = *job.StartedAt
	}
	out["startedAt"] = batchTimeMillis(startedAt)
	if job.StoppedAt != nil {
		out["stoppedAt"] = batchTimeMillis(*job.StoppedAt)
	}
	return out
}

func batchJobSummaryPayload(job batchsvc.Job) map[string]any {
	out := map[string]any{
		"jobArn":       job.ARN,
		"jobId":        job.ID,
		"jobName":      job.Name,
		"status":       job.Status,
		"statusReason": job.StatusReason,
		"createdAt":    batchTimeMillis(job.CreatedAt),
	}
	startedAt := job.CreatedAt
	if job.StartedAt != nil {
		startedAt = *job.StartedAt
	}
	out["startedAt"] = batchTimeMillis(startedAt)
	if job.StoppedAt != nil {
		out["stoppedAt"] = batchTimeMillis(*job.StoppedAt)
	}
	return out
}

func batchConsumableResourceDetailPayload(cr batchsvc.ConsumableResource) map[string]any {
	available := cr.TotalQuantity - cr.InUseQuantity
	if available < 0 {
		available = 0
	}
	return map[string]any{
		"consumableResourceArn":  cr.ARN,
		"consumableResourceName": cr.Name,
		"resourceType":           cr.ResourceType,
		"totalQuantity":          cr.TotalQuantity,
		"inUseQuantity":          cr.InUseQuantity,
		"availableQuantity":      available,
		"createdAt":              batchTimeMillis(cr.CreatedAt),
		"tags":                   cr.Tags,
	}
}

func batchConsumableResourceSummaryPayload(cr batchsvc.ConsumableResource) map[string]any {
	return map[string]any{
		"consumableResourceArn":  cr.ARN,
		"consumableResourceName": cr.Name,
		"resourceType":           cr.ResourceType,
		"totalQuantity":          cr.TotalQuantity,
		"inUseQuantity":          cr.InUseQuantity,
	}
}

func batchServiceEnvironmentPayload(se batchsvc.ServiceEnvironment) map[string]any {
	capacity := make([]map[string]any, 0, len(se.CapacityLimits))
	for _, item := range se.CapacityLimits {
		capacity = append(capacity, map[string]any{
			"capacityUnit": item.CapacityUnit,
			"maxCapacity":  item.MaxCapacity,
		})
	}
	return map[string]any{
		"serviceEnvironmentArn":  se.ARN,
		"serviceEnvironmentName": se.Name,
		"serviceEnvironmentType": se.Type,
		"state":                  se.State,
		"status":                 se.Status,
		"capacityLimits":         capacity,
		"tags":                   se.Tags,
	}
}

func batchServiceJobPayload(job batchsvc.ServiceJob) map[string]any {
	out := map[string]any{
		"jobArn":                job.ARN,
		"jobId":                 job.ID,
		"jobName":               job.Name,
		"jobQueue":              job.Queue,
		"serviceJobType":        job.ServiceJobType,
		"serviceRequestPayload": job.ServiceReqPayload,
		"shareIdentifier":       job.ShareID,
		"schedulingPriority":    job.SchedulingPrio,
		"retryStrategy": map[string]any{
			"attempts":       job.RetryStrategy.Attempts,
			"evaluateOnExit": batchServiceJobEvaluateOnExitPayload(job.RetryStrategy.EvaluateOnExit),
		},
		"timeoutConfig": map[string]any{
			"attemptDurationSeconds": job.TimeoutConfig.AttemptDurationSeconds,
		},
		"attempts":     batchServiceJobAttemptsPayload(job.Attempts),
		"isTerminated": job.IsTerminated,
		"status":       job.Status,
		"statusReason": job.StatusReason,
		"createdAt":    batchTimeMillis(job.CreatedAt),
		"tags":         job.Tags,
	}
	if latest := batchLatestServiceJobAttempt(job.Attempts); latest != nil {
		out["latestAttempt"] = latest
	}
	if job.StartedAt != nil {
		out["startedAt"] = batchTimeMillis(*job.StartedAt)
	}
	if job.StoppedAt != nil {
		out["stoppedAt"] = batchTimeMillis(*job.StoppedAt)
	}
	return out
}

func batchServiceJobSummaryPayload(job batchsvc.ServiceJob) map[string]any {
	out := map[string]any{
		"jobArn":          job.ARN,
		"jobId":           job.ID,
		"jobName":         job.Name,
		"serviceJobType":  job.ServiceJobType,
		"shareIdentifier": job.ShareID,
		"status":          job.Status,
		"statusReason":    job.StatusReason,
		"createdAt":       batchTimeMillis(job.CreatedAt),
	}
	if latest := batchLatestServiceJobAttempt(job.Attempts); latest != nil {
		out["latestAttempt"] = latest
	}
	if job.StartedAt != nil {
		out["startedAt"] = batchTimeMillis(*job.StartedAt)
	}
	if job.StoppedAt != nil {
		out["stoppedAt"] = batchTimeMillis(*job.StoppedAt)
	}
	return out
}

func batchListJobsByConsumablePayload(item batchsvc.ListJobsByConsumableResourceSummary) map[string]any {
	consumables := make([]map[string]any, 0, len(item.Consumables))
	for _, req := range item.Consumables {
		consumables = append(consumables, map[string]any{
			"consumableResource": req.ConsumableResource,
			"quantity":           req.Quantity,
		})
	}
	out := map[string]any{
		"consumableResourceProperties": map[string]any{
			"consumableResourceList": consumables,
		},
		"createdAt":        batchTimeMillis(item.CreatedAt),
		"jobArn":           item.JobARN,
		"jobDefinitionArn": item.JobDefARN,
		"jobName":          item.JobName,
		"jobQueueArn":      item.JobQueueARN,
		"jobStatus":        item.JobStatus,
		"quantity":         item.Quantity,
		"shareIdentifier":  item.ShareID,
		"statusReason":     item.StatusReason,
	}
	if item.StartedAt != nil {
		out["startedAt"] = batchTimeMillis(*item.StartedAt)
	}
	return out
}

func batchServiceJobEvaluateOnExitPayload(in []batchsvc.ServiceJobEvaluateOnExit) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, item := range in {
		out = append(out, map[string]any{
			"action":         item.Action,
			"onStatusReason": item.OnStatusReason,
		})
	}
	return out
}

func batchServiceJobAttemptsPayload(in []batchsvc.ServiceJobAttempt) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, attempt := range in {
		entry := map[string]any{
			"statusReason": attempt.StatusReason,
		}
		if attempt.ServiceResourceName != "" || attempt.ServiceResourceValue != "" {
			entry["serviceResourceId"] = map[string]any{
				"name":  attempt.ServiceResourceName,
				"value": attempt.ServiceResourceValue,
			}
		}
		if attempt.StartedAt != nil {
			entry["startedAt"] = batchTimeMillis(*attempt.StartedAt)
		}
		if attempt.StoppedAt != nil {
			entry["stoppedAt"] = batchTimeMillis(*attempt.StoppedAt)
		}
		out = append(out, entry)
	}
	return out
}

func batchLatestServiceJobAttempt(in []batchsvc.ServiceJobAttempt) map[string]any {
	if len(in) == 0 {
		return nil
	}
	latest := in[len(in)-1]
	if latest.ServiceResourceName == "" && latest.ServiceResourceValue == "" {
		return nil
	}
	return map[string]any{
		"serviceResourceId": map[string]any{
			"name":  latest.ServiceResourceName,
			"value": latest.ServiceResourceValue,
		},
	}
}

func batchComputeEnvironmentOrders(value any) []batchsvc.ComputeEnvironmentOrder {
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]batchsvc.ComputeEnvironmentOrder, 0, len(list))
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		order, _ := batchInt32(entry["order"])
		ce := batchString(entry["computeEnvironment"])
		if ce == "" {
			continue
		}
		out = append(out, batchsvc.ComputeEnvironmentOrder{Order: order, ComputeEnvironment: ce})
	}
	return out
}

func batchOptionalComputeEnvironmentOrders(payload map[string]any, key string) ([]batchsvc.ComputeEnvironmentOrder, bool) {
	value, ok := payload[key]
	if !ok {
		return nil, false
	}
	return batchComputeEnvironmentOrders(value), true
}

func batchOptionalString(payload map[string]any, key string) (string, bool) {
	value, ok := payload[key]
	if !ok {
		return "", false
	}
	return batchString(value), true
}

func batchMap(value any) map[string]any {
	obj, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return obj
}

func batchString(value any) string {
	if value == nil {
		return ""
	}
	v, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

func batchStringSlice(value any) []string {
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		v := batchString(item)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func batchStringMap(value any) map[string]string {
	obj := batchMap(value)
	if len(obj) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(obj))
	for k, v := range obj {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = batchString(v)
	}
	return out
}

func batchInt32(value any) (int32, bool) {
	switch v := value.(type) {
	case float64:
		return int32(v), true
	case int:
		return int32(v), true
	case int32:
		return v, true
	case int64:
		return int32(v), true
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int32(n), true
	default:
		return 0, false
	}
}

func batchInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case float64:
		return int64(v), true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func batchInt64OrZero(value any) int64 {
	v, _ := batchInt64(value)
	return v
}

func batchConsumableRequirements(value any) []batchsvc.ConsumableResourceRequirement {
	obj := batchMap(value)
	list, _ := obj["consumableResourceList"].([]any)
	out := make([]batchsvc.ConsumableResourceRequirement, 0, len(list))
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := batchString(entry["consumableResource"])
		if name == "" {
			continue
		}
		qty, _ := batchInt64(entry["quantity"])
		out = append(out, batchsvc.ConsumableResourceRequirement{
			ConsumableResource: name,
			Quantity:           qty,
		})
	}
	return out
}

func batchServiceEnvironmentCapacities(value any) []batchsvc.ServiceEnvironmentCapacity {
	list, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]batchsvc.ServiceEnvironmentCapacity, 0, len(list))
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		unit := batchString(entry["capacityUnit"])
		if unit == "" {
			continue
		}
		maxCapacity, _ := batchInt64(entry["maxCapacity"])
		out = append(out, batchsvc.ServiceEnvironmentCapacity{
			CapacityUnit: unit,
			MaxCapacity:  maxCapacity,
		})
	}
	return out
}

func batchOptionalServiceEnvironmentCapacities(payload map[string]any, key string) ([]batchsvc.ServiceEnvironmentCapacity, bool) {
	value, ok := payload[key]
	if !ok {
		return nil, false
	}
	return batchServiceEnvironmentCapacities(value), true
}

func batchServiceJobRetryStrategy(value any) batchsvc.ServiceJobRetryStrategy {
	obj := batchMap(value)
	attempts, _ := batchInt32(obj["attempts"])
	rulesRaw, _ := obj["evaluateOnExit"].([]any)
	rules := make([]batchsvc.ServiceJobEvaluateOnExit, 0, len(rulesRaw))
	for _, raw := range rulesRaw {
		entry, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		rules = append(rules, batchsvc.ServiceJobEvaluateOnExit{
			Action:         batchString(entry["action"]),
			OnStatusReason: batchString(entry["onStatusReason"]),
		})
	}
	return batchsvc.ServiceJobRetryStrategy{
		Attempts:       attempts,
		EvaluateOnExit: rules,
	}
}

func batchServiceJobTimeout(value any) batchsvc.ServiceJobTimeout {
	obj := batchMap(value)
	attemptDurationSeconds, _ := batchInt64(obj["attemptDurationSeconds"])
	return batchsvc.ServiceJobTimeout{
		AttemptDurationSeconds: attemptDurationSeconds,
	}
}

func batchTagKeys(r *http.Request) []string {
	values := r.URL.Query()["tagKeys"]
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, key := range strings.Split(value, ",") {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func batchTimeMillis(t time.Time) int64 {
	return t.UTC().UnixMilli()
}

func parseBatchOpPaths() map[string]string {
	out := make(map[string]string, len(batchOperations))
	for _, op := range batchOperations {
		out[op.Name] = op.Method + " " + op.Pattern
	}
	return out
}

func parseBatchMaxResults(payload map[string]any) int {
	raw, ok := payload["maxResults"]
	if !ok {
		return 0
	}
	v, ok := batchInt32(raw)
	if !ok {
		return 0
	}
	if v < 0 {
		return 0
	}
	return int(v)
}

func parseBatchNextToken(payload map[string]any) int {
	tok := batchString(payload["nextToken"])
	if !strings.HasPrefix(tok, "token-") {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimPrefix(tok, "token-"))
	if err != nil || n < 0 {
		return 0
	}
	return n
}
