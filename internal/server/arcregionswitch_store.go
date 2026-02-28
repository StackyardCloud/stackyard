package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type arcRegionSwitchStore struct {
	mu sync.Mutex

	nextPlanID      int64
	nextExecutionID int64

	plans             map[string]map[string]any
	planExecutions    map[string]map[string]any
	executionsByPlan  map[string][]string
	eventsByExecution map[string][]map[string]any
	tags              map[string]map[string]string
	route53Checks     map[string]map[string]any
}

func newARCRegionSwitchStore() *arcRegionSwitchStore {
	now := time.Now().UTC()
	planARN := arcRegionSwitchPlanARN("stackyard-plan")
	executionID := "exec-000001"
	executionARN := arcRegionSwitchExecutionARN(executionID)

	plan := map[string]any{
		"planArn":       planARN,
		"planName":      "stackyard-plan",
		"description":   "seed plan",
		"state":         "ACTIVE",
		"createdAt":     now.Format(time.RFC3339),
		"lastUpdatedAt": now.Format(time.RFC3339),
		"owner":         "123456789012",
		"region":        "us-east-1",
		"workflows":     []any{},
	}

	execution := map[string]any{
		"planExecutionArn": executionARN,
		"planExecutionId":  executionID,
		"planArn":          planARN,
		"state":            "SUCCEEDED",
		"startedAt":        now.Add(-10 * time.Minute).Format(time.RFC3339),
		"lastUpdatedAt":    now.Format(time.RFC3339),
		"steps": []any{
			map[string]any{
				"stepName": "routing-control-check",
				"state":    "SUCCEEDED",
			},
		},
	}

	events := []map[string]any{
		{
			"eventId":      "evt-000001",
			"eventType":    "EXECUTION_STARTED",
			"message":      "Plan execution started",
			"occurredAt":   now.Add(-10 * time.Minute).Format(time.RFC3339),
			"planArn":      planARN,
			"executionArn": executionARN,
		},
		{
			"eventId":      "evt-000002",
			"eventType":    "EXECUTION_SUCCEEDED",
			"message":      "Plan execution completed",
			"occurredAt":   now.Add(-9 * time.Minute).Format(time.RFC3339),
			"planArn":      planARN,
			"executionArn": executionARN,
		},
	}

	return &arcRegionSwitchStore{
		nextPlanID:        2,
		nextExecutionID:   2,
		plans:             map[string]map[string]any{planARN: plan},
		planExecutions:    map[string]map[string]any{executionARN: execution},
		executionsByPlan:  map[string][]string{planARN: {executionARN}},
		eventsByExecution: map[string][]map[string]any{executionARN: events},
		tags: map[string]map[string]string{
			planARN: {"seed": "true"},
		},
		route53Checks: map[string]map[string]any{
			"arn:aws:route53:::healthcheck/hc-000001": {
				"healthCheckArn": "arn:aws:route53:::healthcheck/hc-000001",
				"healthCheckId":  "hc-000001",
				"status":         "HEALTHY",
				"region":         "us-east-1",
			},
			"arn:aws:route53:::healthcheck/hc-000002": {
				"healthCheckArn": "arn:aws:route53:::healthcheck/hc-000002",
				"healthCheckId":  "hc-000002",
				"status":         "HEALTHY",
				"region":         "us-west-2",
			},
		},
	}
}

func (s *arcRegionSwitchStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	defaultPlanARN := s.firstPlanARNLocked()
	if defaultPlanARN == "" {
		defaultPlanARN = arcRegionSwitchPlanARN("stackyard-plan")
		s.plans[defaultPlanARN] = map[string]any{
			"planArn":       defaultPlanARN,
			"planName":      "stackyard-plan",
			"description":   "auto-created seed plan",
			"state":         "ACTIVE",
			"createdAt":     now.Format(time.RFC3339),
			"lastUpdatedAt": now.Format(time.RFC3339),
			"owner":         "123456789012",
			"region":        "us-east-1",
			"workflows":     []any{},
		}
	}

	switch action {
	case "CreatePlan":
		planName := arcRegionSwitchString(payload, "planName", fmt.Sprintf("stackyard-plan-%06d", s.nextPlanID))
		planARN := arcRegionSwitchString(payload, "planArn", arcRegionSwitchPlanARN(planName))
		plan := s.ensurePlanLocked(planARN, now)
		plan["planName"] = planName
		plan["description"] = arcRegionSwitchString(payload, "description", arcRegionSwitchString(plan, "description", ""))
		plan["state"] = arcRegionSwitchString(payload, "state", arcRegionSwitchString(plan, "state", "ACTIVE"))
		plan["region"] = arcRegionSwitchString(payload, "region", arcRegionSwitchString(plan, "region", "us-east-1"))
		plan["owner"] = "123456789012"
		plan["lastUpdatedAt"] = now.Format(time.RFC3339)
		s.nextPlanID++
		return map[string]any{"plan": arcRegionSwitchCloneMap(plan)}

	case "DeletePlan":
		planARN := arcRegionSwitchString(payload, "planArn", defaultPlanARN)
		plan := s.ensurePlanLocked(planARN, now)
		plan["state"] = "DELETED"
		plan["lastUpdatedAt"] = now.Format(time.RFC3339)
		delete(s.plans, planARN)
		delete(s.executionsByPlan, planARN)
		return map[string]any{"plan": arcRegionSwitchCloneMap(plan)}

	case "GetPlan", "GetPlanInRegion":
		planARN := arcRegionSwitchString(payload, "planArn", defaultPlanARN)
		plan := s.ensurePlanLocked(planARN, now)
		if action == "GetPlanInRegion" {
			region := arcRegionSwitchString(payload, "region", arcRegionSwitchString(plan, "region", "us-east-1"))
			plan["region"] = region
		}
		return map[string]any{"plan": arcRegionSwitchCloneMap(plan)}

	case "ListPlans":
		return map[string]any{"plans": s.listPlansLocked(), "nextToken": ""}

	case "ListPlansInRegion":
		region := arcRegionSwitchString(payload, "region", "")
		return map[string]any{"plans": s.listPlansByRegionLocked(region), "nextToken": ""}

	case "StartPlanExecution":
		planARN := arcRegionSwitchString(payload, "planArn", defaultPlanARN)
		execution := s.newExecutionLocked(planARN, now)
		execution["state"] = "IN_PROGRESS"
		s.appendEventLocked(execution, "EXECUTION_STARTED", "Plan execution started", now)
		return map[string]any{"planExecution": arcRegionSwitchCloneMap(execution)}

	case "CancelPlanExecution":
		execution := s.resolveExecutionLocked(payload, defaultPlanARN, now)
		execution["state"] = "CANCELED"
		execution["lastUpdatedAt"] = now.Format(time.RFC3339)
		s.appendEventLocked(execution, "EXECUTION_CANCELED", "Plan execution canceled", now)
		return map[string]any{"planExecution": arcRegionSwitchCloneMap(execution)}

	case "GetPlanExecution":
		execution := s.resolveExecutionLocked(payload, defaultPlanARN, now)
		return map[string]any{"planExecution": arcRegionSwitchCloneMap(execution)}

	case "UpdatePlanExecution":
		execution := s.resolveExecutionLocked(payload, defaultPlanARN, now)
		newState := arcRegionSwitchString(payload, "state", arcRegionSwitchString(execution, "state", "IN_PROGRESS"))
		execution["state"] = newState
		execution["lastUpdatedAt"] = now.Format(time.RFC3339)
		s.appendEventLocked(execution, "EXECUTION_UPDATED", "Plan execution updated", now)
		return map[string]any{"planExecution": arcRegionSwitchCloneMap(execution)}

	case "ApprovePlanExecutionStep", "UpdatePlanExecutionStep":
		execution := s.resolveExecutionLocked(payload, defaultPlanARN, now)
		stepName := arcRegionSwitchString(payload, "stepName", "step-1")
		stepState := "APPROVED"
		if action == "UpdatePlanExecutionStep" {
			stepState = arcRegionSwitchString(payload, "state", "IN_PROGRESS")
		}
		steps := arcRegionSwitchSlice(execution["steps"])
		updated := false
		for _, item := range steps {
			step, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if arcRegionSwitchString(step, "stepName", "") == stepName {
				step["state"] = stepState
				updated = true
			}
		}
		if !updated {
			steps = append(steps, map[string]any{"stepName": stepName, "state": stepState})
		}
		execution["steps"] = steps
		execution["lastUpdatedAt"] = now.Format(time.RFC3339)
		eventType := "EXECUTION_STEP_UPDATED"
		if action == "ApprovePlanExecutionStep" {
			eventType = "EXECUTION_STEP_APPROVED"
		}
		s.appendEventLocked(execution, eventType, "Plan execution step updated", now)
		return map[string]any{"planExecution": arcRegionSwitchCloneMap(execution)}

	case "ListPlanExecutions":
		planARN := arcRegionSwitchString(payload, "planArn", defaultPlanARN)
		ids := s.executionsByPlan[planARN]
		out := make([]any, 0, len(ids))
		for _, executionARN := range ids {
			execution := s.planExecutions[executionARN]
			if execution != nil {
				out = append(out, arcRegionSwitchCloneMap(execution))
			}
		}
		return map[string]any{"planExecutions": out, "nextToken": ""}

	case "ListPlanExecutionEvents":
		execution := s.resolveExecutionLocked(payload, defaultPlanARN, now)
		executionARN := arcRegionSwitchString(execution, "planExecutionArn", "")
		events := s.eventsByExecution[executionARN]
		out := make([]any, 0, len(events))
		for _, event := range events {
			out = append(out, arcRegionSwitchCloneMap(event))
		}
		return map[string]any{"events": out, "nextToken": ""}

	case "GetPlanEvaluationStatus":
		planARN := arcRegionSwitchString(payload, "planArn", defaultPlanARN)
		plan := s.ensurePlanLocked(planARN, now)
		return map[string]any{
			"planArn":          planARN,
			"evaluationStatus": "SUCCEEDED",
			"lastEvaluatedAt":  now.Format(time.RFC3339),
			"warnings":         []any{},
			"plan":             arcRegionSwitchCloneMap(plan),
		}

	case "ListRoute53HealthChecks":
		return map[string]any{"route53HealthChecks": s.listRoute53ChecksLocked(""), "nextToken": ""}

	case "ListRoute53HealthChecksInRegion":
		region := arcRegionSwitchString(payload, "region", "")
		return map[string]any{"route53HealthChecks": s.listRoute53ChecksLocked(region), "nextToken": ""}

	case "TagResource":
		resourceARN := arcRegionSwitchString(payload, "resourceArn", defaultPlanARN)
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		for key, value := range arcRegionSwitchTags(payload["tags"]) {
			s.tags[resourceARN][key] = value
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN := arcRegionSwitchString(payload, "resourceArn", defaultPlanARN)
		keys := arcRegionSwitchStringList(payload["tagKeys"])
		if len(keys) == 0 {
			keys = arcRegionSwitchStringList(payload["TagKeys"])
		}
		for _, key := range keys {
			delete(s.tags[resourceARN], key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := arcRegionSwitchString(payload, "resourceArn", defaultPlanARN)
		return map[string]any{"tags": arcRegionSwitchCloneStringMap(s.tags[resourceARN])}

	case "UpdatePlan":
		planARN := arcRegionSwitchString(payload, "planArn", defaultPlanARN)
		plan := s.ensurePlanLocked(planARN, now)
		if value := arcRegionSwitchString(payload, "planName", ""); value != "" {
			plan["planName"] = value
		}
		if value := arcRegionSwitchString(payload, "description", ""); value != "" {
			plan["description"] = value
		}
		if value := arcRegionSwitchString(payload, "state", ""); value != "" {
			plan["state"] = value
		}
		plan["lastUpdatedAt"] = now.Format(time.RFC3339)
		return map[string]any{"plan": arcRegionSwitchCloneMap(plan)}
	}

	return map[string]any{}
}

func (s *arcRegionSwitchStore) firstPlanARNLocked() string {
	if len(s.plans) == 0 {
		return ""
	}
	keys := make([]string, 0, len(s.plans))
	for key := range s.plans {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *arcRegionSwitchStore) ensurePlanLocked(planARN string, now time.Time) map[string]any {
	planARN = strings.TrimSpace(planARN)
	if planARN == "" {
		planARN = arcRegionSwitchPlanARN(fmt.Sprintf("stackyard-plan-%06d", s.nextPlanID))
		s.nextPlanID++
	}
	if plan, ok := s.plans[planARN]; ok {
		return plan
	}
	name := planARN[strings.LastIndex(planARN, "/")+1:]
	plan := map[string]any{
		"planArn":       planARN,
		"planName":      name,
		"description":   "auto-created plan",
		"state":         "ACTIVE",
		"createdAt":     now.Format(time.RFC3339),
		"lastUpdatedAt": now.Format(time.RFC3339),
		"owner":         "123456789012",
		"region":        "us-east-1",
		"workflows":     []any{},
	}
	s.plans[planARN] = plan
	return plan
}

func (s *arcRegionSwitchStore) newExecutionLocked(planARN string, now time.Time) map[string]any {
	plan := s.ensurePlanLocked(planARN, now)
	id := fmt.Sprintf("exec-%06d", s.nextExecutionID)
	s.nextExecutionID++
	arn := arcRegionSwitchExecutionARN(id)
	execution := map[string]any{
		"planExecutionArn": arn,
		"planExecutionId":  id,
		"planArn":          arcRegionSwitchString(plan, "planArn", planARN),
		"state":            "IN_PROGRESS",
		"startedAt":        now.Format(time.RFC3339),
		"lastUpdatedAt":    now.Format(time.RFC3339),
		"steps":            []any{},
	}
	s.planExecutions[arn] = execution
	s.executionsByPlan[arcRegionSwitchString(plan, "planArn", planARN)] = append(s.executionsByPlan[arcRegionSwitchString(plan, "planArn", planARN)], arn)
	if s.eventsByExecution[arn] == nil {
		s.eventsByExecution[arn] = []map[string]any{}
	}
	return execution
}

func (s *arcRegionSwitchStore) resolveExecutionLocked(payload map[string]any, defaultPlanARN string, now time.Time) map[string]any {
	executionARN := arcRegionSwitchString(payload, "planExecutionArn", "")
	if executionARN != "" {
		if execution, ok := s.planExecutions[executionARN]; ok {
			return execution
		}
	}

	planARN := arcRegionSwitchString(payload, "planArn", defaultPlanARN)
	if ids := s.executionsByPlan[planARN]; len(ids) > 0 {
		for i := len(ids) - 1; i >= 0; i-- {
			if execution := s.planExecutions[ids[i]]; execution != nil {
				return execution
			}
		}
	}
	return s.newExecutionLocked(planARN, now)
}

func (s *arcRegionSwitchStore) appendEventLocked(execution map[string]any, eventType, message string, now time.Time) {
	executionARN := arcRegionSwitchString(execution, "planExecutionArn", "")
	if executionARN == "" {
		return
	}
	event := map[string]any{
		"eventId":      fmt.Sprintf("evt-%d", now.UnixNano()),
		"eventType":    eventType,
		"message":      message,
		"occurredAt":   now.Format(time.RFC3339),
		"planArn":      arcRegionSwitchString(execution, "planArn", ""),
		"executionArn": executionARN,
	}
	s.eventsByExecution[executionARN] = append(s.eventsByExecution[executionARN], event)
}

func (s *arcRegionSwitchStore) listPlansLocked() []any {
	keys := make([]string, 0, len(s.plans))
	for key := range s.plans {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, arcRegionSwitchCloneMap(s.plans[key]))
	}
	return out
}

func (s *arcRegionSwitchStore) listPlansByRegionLocked(region string) []any {
	region = strings.TrimSpace(region)
	out := []any{}
	for _, planAny := range s.listPlansLocked() {
		plan, ok := planAny.(map[string]any)
		if !ok {
			continue
		}
		if region == "" || arcRegionSwitchString(plan, "region", "") == region {
			out = append(out, plan)
		}
	}
	return out
}

func (s *arcRegionSwitchStore) listRoute53ChecksLocked(region string) []any {
	region = strings.TrimSpace(region)
	keys := make([]string, 0, len(s.route53Checks))
	for key := range s.route53Checks {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		check := s.route53Checks[key]
		if region != "" && arcRegionSwitchString(check, "region", "") != region {
			continue
		}
		out = append(out, arcRegionSwitchCloneMap(check))
	}
	return out
}

func arcRegionSwitchPlanARN(planName string) string {
	return fmt.Sprintf("arn:aws:arc-region-switch:us-east-1:123456789012:plan/%s", strings.TrimSpace(planName))
}

func arcRegionSwitchExecutionARN(executionID string) string {
	return fmt.Sprintf("arn:aws:arc-region-switch:us-east-1:123456789012:plan-execution/%s", strings.TrimSpace(executionID))
}

func arcRegionSwitchString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return def
	}
	str, ok := raw.(string)
	if !ok {
		return def
	}
	str = strings.TrimSpace(str)
	if str == "" {
		return def
	}
	return str
}

func arcRegionSwitchTags(raw any) map[string]string {
	out := map[string]string{}
	switch typed := raw.(type) {
	case map[string]string:
		for key, value := range typed {
			k := strings.TrimSpace(key)
			if k != "" {
				out[k] = value
			}
		}
	case map[string]any:
		for key, value := range typed {
			k := strings.TrimSpace(key)
			if k != "" {
				out[k] = fmt.Sprintf("%v", value)
			}
		}
	}
	return out
}

func arcRegionSwitchStringList(raw any) []string {
	if raw == nil {
		return nil
	}
	if list, ok := raw.([]string); ok {
		out := make([]string, 0, len(list))
		for _, item := range list {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		str, ok := item.(string)
		if !ok {
			continue
		}
		str = strings.TrimSpace(str)
		if str != "" {
			out = append(out, str)
		}
	}
	return out
}

func arcRegionSwitchSlice(raw any) []any {
	list, ok := raw.([]any)
	if !ok {
		return []any{}
	}
	out := make([]any, 0, len(list))
	out = append(out, list...)
	return out
}

func arcRegionSwitchCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = arcRegionSwitchCloneMap(typed)
		case []any:
			out[key] = arcRegionSwitchCloneSlice(typed)
		case map[string]string:
			out[key] = arcRegionSwitchCloneStringMap(typed)
		default:
			out[key] = typed
		}
	}
	return out
}

func arcRegionSwitchCloneSlice(in []any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		switch typed := item.(type) {
		case map[string]any:
			out = append(out, arcRegionSwitchCloneMap(typed))
		case []any:
			out = append(out, arcRegionSwitchCloneSlice(typed))
		case map[string]string:
			out = append(out, arcRegionSwitchCloneStringMap(typed))
		default:
			out = append(out, typed)
		}
	}
	return out
}

func arcRegionSwitchCloneStringMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range in {
		out[key] = value
	}
	return out
}
