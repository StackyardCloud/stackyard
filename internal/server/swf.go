package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/stackyard/stackyard/internal/services/swf"
)

type swfError struct {
	Type    string `json:"__type"`
	Message string `json:"message"`
}

type swfDomainInfo struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Description string `json:"description,omitempty"`
	Arn         string `json:"arn,omitempty"`
}

type swfDomainConfiguration struct {
	WorkflowExecutionRetentionPeriodInDays string `json:"workflowExecutionRetentionPeriodInDays"`
}

type swfActivityType struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type swfWorkflowType struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type swfActivityTypeInfo struct {
	ActivityType   swfActivityType `json:"activityType"`
	Status         string          `json:"status"`
	Description    string          `json:"description,omitempty"`
	CreationDate   float64         `json:"creationDate,omitempty"`
	DeprecatedDate float64         `json:"deprecationDate,omitempty"`
}

type swfWorkflowTypeInfo struct {
	WorkflowType   swfWorkflowType `json:"workflowType"`
	Status         string          `json:"status"`
	Description    string          `json:"description,omitempty"`
	CreationDate   float64         `json:"creationDate,omitempty"`
	DeprecatedDate float64         `json:"deprecationDate,omitempty"`
}

type swfExecution struct {
	WorkflowId string `json:"workflowId"`
	RunId      string `json:"runId"`
}

type swfWorkflowExecutionInfo struct {
	Execution       swfExecution    `json:"execution"`
	WorkflowType    swfWorkflowType `json:"workflowType"`
	StartTimestamp  float64         `json:"startTimestamp"`
	CloseTimestamp  float64         `json:"closeTimestamp,omitempty"`
	CloseStatus     string          `json:"closeStatus,omitempty"`
	ExecutionStatus string          `json:"executionStatus"`
	ParentExecution *swfExecution   `json:"parentExecution,omitempty"`
	TagList         []string        `json:"tagList,omitempty"`
	CancelRequested bool            `json:"cancelRequested,omitempty"`
}

type swfTaskList struct {
	Name string `json:"name"`
}

type swfResourceTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (s *Server) handleSWFJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if !isSWFJSONCandidate(r) {
		return false
	}
	ok, status, code, msg, _ := s.validateSigV4WithService(r, "swf")
	if !ok {
		respondSWFJSONError(w, status, code, msg)
		return true
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	action := parseSWFTarget(target)
	if action == "" {
		respondSWFJSONError(w, http.StatusBadRequest, "InvalidAction", "missing X-Amz-Target")
		return true
	}

	body, err := readBodyBytes(r)
	if err != nil {
		respondSWFJSONError(w, http.StatusBadRequest, "InvalidRequest", "unable to read request body")
		return true
	}
	if len(bytes.TrimSpace(body)) == 0 {
		body = []byte("{}")
	}

	switch action {
	case "RegisterDomain":
		var input struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Retention   string `json:"workflowExecutionRetentionPeriodInDays"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if err := s.swf.RegisterDomain(input.Name, input.Description, input.Retention); err != nil {
			switch err {
			case swf.ErrDomainAlreadyExists:
				respondSWFJSONError(w, http.StatusBadRequest, "DomainAlreadyExistsFault", err.Error())
			case swf.ErrInvalidParameter:
				respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", err.Error())
			default:
				respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", err.Error())
			}
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DescribeDomain":
		var input struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(body, &input)
		domain, err := s.swf.DescribeDomain(input.Name)
		if err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "DomainDoesNotExistFault", "domain not found")
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{
			"domainInfo": swfDomainInfo{
				Name:        domain.Name,
				Status:      domain.Status,
				Description: domain.Description,
				Arn:         swfDomainArn(domain.Name),
			},
			"configuration": swfDomainConfiguration{
				WorkflowExecutionRetentionPeriodInDays: domain.Retention,
			},
		})
		return true
	case "ListDomains":
		var input struct {
			Status string `json:"registrationStatus"`
		}
		_ = json.Unmarshal(body, &input)
		if strings.TrimSpace(input.Status) == "" {
			respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", "registrationStatus is required")
			return true
		}
		domains := s.swf.ListDomains(strings.TrimSpace(input.Status))
		infos := make([]swfDomainInfo, 0, len(domains))
		for _, domain := range domains {
			infos = append(infos, swfDomainInfo{
				Name:        domain.Name,
				Status:      domain.Status,
				Description: domain.Description,
				Arn:         swfDomainArn(domain.Name),
			})
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{"domainInfos": infos})
		return true
	case "DeprecateDomain":
		var input struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(body, &input)
		if err := s.swf.DeprecateDomain(input.Name); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "DomainDoesNotExistFault", "domain not found")
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "UndeprecateDomain":
		var input struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(body, &input)
		if err := s.swf.UndeprecateDomain(input.Name); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "DomainDoesNotExistFault", "domain not found")
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "RegisterActivityType":
		var input struct {
			Domain      string `json:"domain"`
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if err := s.swf.RegisterActivityType(input.Domain, input.Name, input.Version, input.Description); err != nil {
			switch err {
			case swf.ErrDomainNotFound:
				respondSWFJSONError(w, http.StatusBadRequest, "DomainDoesNotExistFault", "domain not found")
			case swf.ErrActivityTypeAlreadyExists:
				respondSWFJSONError(w, http.StatusBadRequest, "TypeAlreadyExistsFault", "activity type exists")
			default:
				respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", err.Error())
			}
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DescribeActivityType":
		var input struct {
			Domain       string          `json:"domain"`
			ActivityType swfActivityType `json:"activityType"`
		}
		_ = json.Unmarshal(body, &input)
		activity, err := s.swf.DescribeActivityType(input.Domain, input.ActivityType.Name, input.ActivityType.Version)
		if err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "TypeDoesNotExistFault", "activity type not found")
			return true
		}
		info := swfActivityTypeInfo{
			ActivityType: swfActivityType{Name: activity.Name, Version: activity.Version},
			Status:       activity.Status,
			Description:  activity.Description,
			CreationDate: swfSyntheticCreationDate(activity.Name, activity.Version),
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{
			"typeInfo": info,
			"configuration": map[string]any{
				"defaultTaskHeartbeatTimeout":       "60",
				"defaultTaskList":                   swfTaskList{Name: swfDefaultTaskListName(activity.Domain)},
				"defaultTaskScheduleToCloseTimeout": "120",
				"defaultTaskScheduleToStartTimeout": "60",
				"defaultTaskStartToCloseTimeout":    "60",
			},
		})
		return true
	case "ListActivityTypes":
		var input struct {
			Domain string `json:"domain"`
			Status string `json:"registrationStatus"`
		}
		_ = json.Unmarshal(body, &input)
		if strings.TrimSpace(input.Domain) == "" || strings.TrimSpace(input.Status) == "" {
			respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", "domain and registrationStatus are required")
			return true
		}
		types := s.swf.ListActivityTypes(input.Domain, input.Status)
		out := make([]swfActivityTypeInfo, 0, len(types))
		for _, activity := range types {
			out = append(out, swfActivityTypeInfo{
				ActivityType: swfActivityType{Name: activity.Name, Version: activity.Version},
				Status:       activity.Status,
				Description:  activity.Description,
				CreationDate: swfSyntheticCreationDate(activity.Name, activity.Version),
			})
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{"typeInfos": out})
		return true
	case "DeprecateActivityType":
		var input struct {
			Domain       string          `json:"domain"`
			ActivityType swfActivityType `json:"activityType"`
		}
		_ = json.Unmarshal(body, &input)
		if err := s.swf.DeprecateActivityType(input.Domain, input.ActivityType.Name, input.ActivityType.Version); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "TypeDoesNotExistFault", "activity type not found")
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "UndeprecateActivityType":
		var input struct {
			Domain       string          `json:"domain"`
			ActivityType swfActivityType `json:"activityType"`
		}
		_ = json.Unmarshal(body, &input)
		if err := s.swf.UndeprecateActivityType(input.Domain, input.ActivityType.Name, input.ActivityType.Version); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "TypeDoesNotExistFault", "activity type not found")
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DeleteActivityType":
		var input struct {
			Domain       string          `json:"domain"`
			ActivityType swfActivityType `json:"activityType"`
		}
		_ = json.Unmarshal(body, &input)
		if err := s.swf.DeleteActivityType(input.Domain, input.ActivityType.Name, input.ActivityType.Version); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "TypeDoesNotExistFault", "activity type not found")
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "RegisterWorkflowType":
		var input struct {
			Domain      string `json:"domain"`
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if err := s.swf.RegisterWorkflowType(input.Domain, input.Name, input.Version, input.Description); err != nil {
			switch err {
			case swf.ErrDomainNotFound:
				respondSWFJSONError(w, http.StatusBadRequest, "DomainDoesNotExistFault", "domain not found")
			case swf.ErrWorkflowTypeAlreadyExists:
				respondSWFJSONError(w, http.StatusBadRequest, "TypeAlreadyExistsFault", "workflow type exists")
			default:
				respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", err.Error())
			}
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DescribeWorkflowType":
		var input struct {
			Domain       string          `json:"domain"`
			WorkflowType swfWorkflowType `json:"workflowType"`
		}
		_ = json.Unmarshal(body, &input)
		workflow, err := s.swf.DescribeWorkflowType(input.Domain, input.WorkflowType.Name, input.WorkflowType.Version)
		if err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "TypeDoesNotExistFault", "workflow type not found")
			return true
		}
		info := swfWorkflowTypeInfo{
			WorkflowType: swfWorkflowType{Name: workflow.Name, Version: workflow.Version},
			Status:       workflow.Status,
			Description:  workflow.Description,
			CreationDate: swfSyntheticCreationDate(workflow.Name, workflow.Version),
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{
			"typeInfo":      info,
			"configuration": swfWorkflowExecutionConfiguration(swfDefaultTaskListName(workflow.Domain)),
		})
		return true
	case "ListWorkflowTypes":
		var input struct {
			Domain string `json:"domain"`
			Status string `json:"registrationStatus"`
		}
		_ = json.Unmarshal(body, &input)
		if strings.TrimSpace(input.Domain) == "" || strings.TrimSpace(input.Status) == "" {
			respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", "domain and registrationStatus are required")
			return true
		}
		types := s.swf.ListWorkflowTypes(input.Domain, input.Status)
		out := make([]swfWorkflowTypeInfo, 0, len(types))
		for _, workflow := range types {
			out = append(out, swfWorkflowTypeInfo{
				WorkflowType: swfWorkflowType{Name: workflow.Name, Version: workflow.Version},
				Status:       workflow.Status,
				Description:  workflow.Description,
				CreationDate: swfSyntheticCreationDate(workflow.Name, workflow.Version),
			})
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{"typeInfos": out})
		return true
	case "DeprecateWorkflowType":
		var input struct {
			Domain       string          `json:"domain"`
			WorkflowType swfWorkflowType `json:"workflowType"`
		}
		_ = json.Unmarshal(body, &input)
		if err := s.swf.DeprecateWorkflowType(input.Domain, input.WorkflowType.Name, input.WorkflowType.Version); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "TypeDoesNotExistFault", "workflow type not found")
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "UndeprecateWorkflowType":
		var input struct {
			Domain       string          `json:"domain"`
			WorkflowType swfWorkflowType `json:"workflowType"`
		}
		_ = json.Unmarshal(body, &input)
		if err := s.swf.UndeprecateWorkflowType(input.Domain, input.WorkflowType.Name, input.WorkflowType.Version); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "TypeDoesNotExistFault", "workflow type not found")
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "DeleteWorkflowType":
		var input struct {
			Domain       string          `json:"domain"`
			WorkflowType swfWorkflowType `json:"workflowType"`
		}
		_ = json.Unmarshal(body, &input)
		if err := s.swf.DeleteWorkflowType(input.Domain, input.WorkflowType.Name, input.WorkflowType.Version); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "TypeDoesNotExistFault", "workflow type not found")
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "StartWorkflowExecution":
		var input struct {
			Domain       string          `json:"domain"`
			WorkflowID   string          `json:"workflowId"`
			WorkflowType swfWorkflowType `json:"workflowType"`
			TaskList     swfTaskList     `json:"taskList"`
			TagList      []string        `json:"tagList"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		exec, err := s.swf.StartWorkflowExecution(input.Domain, input.WorkflowID, input.WorkflowType.Name, input.WorkflowType.Version, input.TaskList.Name, input.TagList)
		if err != nil {
			switch err {
			case swf.ErrDomainNotFound:
				respondSWFJSONError(w, http.StatusBadRequest, "DomainDoesNotExistFault", "domain not found")
			case swf.ErrWorkflowTypeNotFound:
				respondSWFJSONError(w, http.StatusBadRequest, "TypeDoesNotExistFault", "workflow type not found")
			case swf.ErrWorkflowExecutionAlreadyStarted:
				respondSWFJSONError(w, http.StatusBadRequest, "WorkflowExecutionAlreadyStartedFault", "workflow already started")
			default:
				respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", err.Error())
			}
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{"runId": exec.RunID})
		return true
	case "DescribeWorkflowExecution":
		var input struct {
			Domain    string       `json:"domain"`
			Execution swfExecution `json:"execution"`
		}
		_ = json.Unmarshal(body, &input)
		exec, err := s.swf.DescribeWorkflowExecution(input.Domain, input.Execution.WorkflowId, input.Execution.RunId)
		if err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "UnknownResourceFault", "workflow execution not found")
			return true
		}
		info := swfWorkflowExecutionInfo{
			Execution:       swfExecution{WorkflowId: exec.WorkflowID, RunId: exec.RunID},
			WorkflowType:    swfWorkflowType{Name: exec.WorkflowTypeName, Version: exec.WorkflowTypeVersion},
			StartTimestamp:  float64(exec.StartTime.Unix()),
			CloseTimestamp:  closeTimestamp(exec),
			CloseStatus:     exec.CloseStatus,
			ExecutionStatus: exec.Status,
			TagList:         exec.Tags,
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{
			"executionInfo":          info,
			"executionConfiguration": swfWorkflowExecutionConfiguration(exec.TaskList),
			"openCounts": map[string]any{
				"openActivityTasks":           0,
				"openDecisionTasks":           0,
				"openTimers":                  0,
				"openChildWorkflowExecutions": 0,
			},
		})
		return true
	case "ListOpenWorkflowExecutions":
		var input struct {
			Domain string `json:"domain"`
		}
		_ = json.Unmarshal(body, &input)
		execs := s.swf.ListWorkflowExecutions(input.Domain, swf.StatusOpen)
		respondSWFJSON(w, http.StatusOK, map[string]any{
			"executionInfos": buildExecutionInfos(execs),
		})
		return true
	case "ListClosedWorkflowExecutions":
		var input struct {
			Domain string `json:"domain"`
		}
		_ = json.Unmarshal(body, &input)
		execs := s.swf.ListWorkflowExecutions(input.Domain, swf.StatusClosed)
		respondSWFJSON(w, http.StatusOK, map[string]any{
			"executionInfos": buildExecutionInfos(execs),
		})
		return true
	case "CountOpenWorkflowExecutions":
		var input struct {
			Domain string `json:"domain"`
		}
		_ = json.Unmarshal(body, &input)
		count := s.swf.CountWorkflowExecutions(input.Domain, swf.StatusOpen)
		respondSWFJSON(w, http.StatusOK, map[string]any{"count": count, "truncated": false})
		return true
	case "CountClosedWorkflowExecutions":
		var input struct {
			Domain string `json:"domain"`
		}
		_ = json.Unmarshal(body, &input)
		count := s.swf.CountWorkflowExecutions(input.Domain, swf.StatusClosed)
		respondSWFJSON(w, http.StatusOK, map[string]any{"count": count, "truncated": false})
		return true
	case "GetWorkflowExecutionHistory":
		var input struct {
			Domain    string       `json:"domain"`
			Execution swfExecution `json:"execution"`
		}
		_ = json.Unmarshal(body, &input)
		_, err := s.swf.DescribeWorkflowExecution(input.Domain, input.Execution.WorkflowId, input.Execution.RunId)
		if err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "UnknownResourceFault", "workflow execution not found")
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{"events": []any{}})
		return true
	case "RequestCancelWorkflowExecution":
		var input struct {
			Domain     string `json:"domain"`
			WorkflowID string `json:"workflowId"`
			RunID      string `json:"runId"`
		}
		_ = json.Unmarshal(body, &input)
		if err := s.swf.CloseWorkflowExecution(input.Domain, input.WorkflowID, input.RunID, swf.CloseCanceled); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "UnknownResourceFault", "workflow execution not found")
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "SignalWorkflowExecution":
		var input struct {
			Domain     string `json:"domain"`
			WorkflowID string `json:"workflowId"`
			RunID      string `json:"runId"`
		}
		_ = json.Unmarshal(body, &input)
		if _, err := s.swf.DescribeWorkflowExecution(input.Domain, input.WorkflowID, input.RunID); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "UnknownResourceFault", "workflow execution not found")
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "TerminateWorkflowExecution":
		var input struct {
			Domain     string `json:"domain"`
			WorkflowID string `json:"workflowId"`
			RunID      string `json:"runId"`
		}
		_ = json.Unmarshal(body, &input)
		if err := s.swf.CloseWorkflowExecution(input.Domain, input.WorkflowID, input.RunID, swf.CloseTerminated); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "UnknownResourceFault", "workflow execution not found")
			return true
		}
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "CountPendingActivityTasks":
		respondSWFJSON(w, http.StatusOK, map[string]any{"count": 0, "truncated": false})
		return true
	case "CountPendingDecisionTasks":
		respondSWFJSON(w, http.StatusOK, map[string]any{"count": 0, "truncated": false})
		return true
	case "PollForActivityTask":
		var input struct {
			Domain   string      `json:"domain"`
			TaskList swfTaskList `json:"taskList"`
		}
		_ = json.Unmarshal(body, &input)
		execution := s.swfPollExecution(input.Domain)
		activityType := s.swfPollActivityType(input.Domain)
		respondSWFJSON(w, http.StatusOK, map[string]any{
			"activityId":        "stackyard-activity-1",
			"activityType":      activityType,
			"startedEventId":    int64(1),
			"taskToken":         "stackyard-activity-task-token",
			"workflowExecution": execution,
			"input":             "{}",
		})
		return true
	case "PollForDecisionTask":
		var input struct {
			Domain   string      `json:"domain"`
			TaskList swfTaskList `json:"taskList"`
		}
		_ = json.Unmarshal(body, &input)
		execution := s.swfPollExecution(input.Domain)
		workflowType := s.swfPollWorkflowType(input.Domain)
		eventTimestamp := s.swfPollExecutionStartTimestamp(input.Domain)
		respondSWFJSON(w, http.StatusOK, map[string]any{
			"events": []any{
				map[string]any{
					"eventId":        int64(1),
					"eventTimestamp": eventTimestamp,
					"eventType":      "WorkflowExecutionStarted",
					"workflowExecutionStartedEventAttributes": map[string]any{
						"childPolicy":                  "TERMINATE",
						"executionStartToCloseTimeout": "3600",
						"input":                        "{}",
						"taskList":                     map[string]any{"name": swfDefaultTaskListName(input.TaskList.Name)},
						"taskStartToCloseTimeout":      "300",
						"workflowType":                 workflowType,
					},
				},
			},
			"startedEventId":         int64(1),
			"taskToken":              "stackyard-decision-task-token",
			"workflowExecution":      execution,
			"workflowType":           workflowType,
			"previousStartedEventId": int64(0),
		})
		return true
	case "RecordActivityTaskHeartbeat":
		respondSWFJSON(w, http.StatusOK, map[string]any{"cancelRequested": false})
		return true
	case "RespondActivityTaskCompleted", "RespondActivityTaskFailed", "RespondActivityTaskCanceled", "RespondDecisionTaskCompleted":
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "TagResource":
		var input struct {
			ResourceArn string           `json:"resourceArn"`
			Tags        []swfResourceTag `json:"tags"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if strings.TrimSpace(input.ResourceArn) == "" {
			respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", "resourceArn is required")
			return true
		}
		tagMap := map[string]string{}
		for _, tag := range input.Tags {
			if tag.Key == "" {
				continue
			}
			tagMap[tag.Key] = tag.Value
		}
		s.swf.TagResource(input.ResourceArn, tagMap)
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "UntagResource":
		var input struct {
			ResourceArn string   `json:"resourceArn"`
			TagKeys     []string `json:"tagKeys"`
		}
		if err := json.Unmarshal(body, &input); err != nil {
			respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", "invalid JSON")
			return true
		}
		if strings.TrimSpace(input.ResourceArn) == "" {
			respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", "resourceArn is required")
			return true
		}
		s.swf.UntagResource(input.ResourceArn, input.TagKeys)
		respondSWFJSON(w, http.StatusOK, map[string]any{})
		return true
	case "ListTagsForResource":
		var input struct {
			ResourceArn string `json:"resourceArn"`
		}
		_ = json.Unmarshal(body, &input)
		if strings.TrimSpace(input.ResourceArn) == "" {
			respondSWFJSONError(w, http.StatusBadRequest, "ValidationException", "resourceArn is required")
			return true
		}
		tags := s.swf.ListTagsForResource(input.ResourceArn)
		tagList := make([]swfResourceTag, 0, len(tags))
		for k, v := range tags {
			tagList = append(tagList, swfResourceTag{Key: k, Value: v})
		}
		sort.Slice(tagList, func(i, j int) bool {
			return tagList[i].Key < tagList[j].Key
		})
		respondSWFJSON(w, http.StatusOK, map[string]any{"tags": tagList})
		return true
	default:
		respondSWFJSONError(w, http.StatusNotImplemented, "NotImplemented", "operation not implemented")
		return true
	}
}

func respondSWFJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondSWFJSONError(w http.ResponseWriter, status int, code, msg string) {
	respondSWFJSON(w, status, swfError{Type: code, Message: msg})
}

func isSWFJSONCandidate(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	target := strings.TrimSpace(r.Header.Get("X-Amz-Target"))
	if strings.HasPrefix(target, "SimpleWorkflowService") {
		return true
	}
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/x-amz-json-1.0") || strings.Contains(contentType, "application/x-amz-json-1.1") {
		return strings.HasPrefix(target, "SimpleWorkflowService")
	}
	return false
}

func parseSWFTarget(target string) string {
	if target == "" {
		return ""
	}
	if strings.HasPrefix(target, "SimpleWorkflowService.") {
		return strings.TrimPrefix(target, "SimpleWorkflowService.")
	}
	if dot := strings.LastIndex(target, "."); dot >= 0 && dot+1 < len(target) {
		return target[dot+1:]
	}
	return target
}

func swfDomainArn(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	return "arn:aws:swf:" + swf.DefaultRegion + ":" + swf.DefaultAccountID + ":domain/" + name
}

func buildExecutionInfos(execs []swf.WorkflowExecution) []swfWorkflowExecutionInfo {
	out := make([]swfWorkflowExecutionInfo, 0, len(execs))
	for _, exec := range execs {
		out = append(out, swfWorkflowExecutionInfo{
			Execution:       swfExecution{WorkflowId: exec.WorkflowID, RunId: exec.RunID},
			WorkflowType:    swfWorkflowType{Name: exec.WorkflowTypeName, Version: exec.WorkflowTypeVersion},
			StartTimestamp:  float64(exec.StartTime.Unix()),
			CloseTimestamp:  closeTimestamp(exec),
			CloseStatus:     exec.CloseStatus,
			ExecutionStatus: exec.Status,
			TagList:         exec.Tags,
		})
	}
	return out
}

func closeTimestamp(exec swf.WorkflowExecution) float64 {
	if exec.CloseTime.IsZero() {
		return 0
	}
	return float64(exec.CloseTime.Unix())
}

func swfSyntheticCreationDate(name, version string) float64 {
	base := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	offset := time.Duration(len(strings.TrimSpace(name))+len(strings.TrimSpace(version))) * time.Minute
	return float64(base.Add(offset).Unix())
}

func swfDefaultTaskListName(domain string) string {
	if strings.TrimSpace(domain) == "" {
		return "stackyard-swf-tasklist"
	}
	return strings.TrimSpace(domain) + "-tasklist"
}

func swfWorkflowExecutionConfiguration(taskList string) map[string]any {
	if strings.TrimSpace(taskList) == "" {
		taskList = swfDefaultTaskListName("")
	}
	return map[string]any{
		"childPolicy":                  "TERMINATE",
		"executionStartToCloseTimeout": "600",
		"taskList":                     swfTaskList{Name: taskList},
		"taskStartToCloseTimeout":      "60",
	}
}

func (s *Server) swfPollExecution(domain string) swfExecution {
	executions := s.swf.ListWorkflowExecutions(domain, swf.StatusOpen)
	if len(executions) == 0 {
		executions = s.swf.ListWorkflowExecutions(domain, "")
	}
	if len(executions) == 0 {
		return swfExecution{WorkflowId: "stackyard-swf-workflow-id", RunId: "stackyard-swf-run-id"}
	}
	exec := executions[0]
	return swfExecution{WorkflowId: exec.WorkflowID, RunId: exec.RunID}
}

func (s *Server) swfPollExecutionStartTimestamp(domain string) float64 {
	executions := s.swf.ListWorkflowExecutions(domain, swf.StatusOpen)
	if len(executions) == 0 {
		executions = s.swf.ListWorkflowExecutions(domain, "")
	}
	if len(executions) == 0 || executions[0].StartTime.IsZero() {
		return swfSyntheticCreationDate(strings.TrimSpace(domain), "poll")
	}
	return float64(executions[0].StartTime.Unix())
}

func (s *Server) swfPollActivityType(domain string) swfActivityType {
	types := s.swf.ListActivityTypes(domain, "")
	if len(types) == 0 {
		return swfActivityType{Name: "stackyard-swf-activity", Version: "1"}
	}
	return swfActivityType{Name: types[0].Name, Version: types[0].Version}
}

func (s *Server) swfPollWorkflowType(domain string) swfWorkflowType {
	types := s.swf.ListWorkflowTypes(domain, "")
	if len(types) == 0 {
		return swfWorkflowType{Name: "stackyard-swf-workflow", Version: "1"}
	}
	return swfWorkflowType{Name: types[0].Name, Version: types[0].Version}
}
