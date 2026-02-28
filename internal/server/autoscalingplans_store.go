package server

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type autoScalingPlansStore struct {
	mu    sync.Mutex
	plans map[string]*autoScalingPlanRecord
}

type autoScalingPlanRecord struct {
	Name                string
	Version             int64
	ApplicationSource   map[string]any
	ScalingInstructions []map[string]any
	StatusCode          string
	StatusMessage       string
	StatusStartTime     time.Time
	CreationTime        time.Time
	UpdatedTime         time.Time
}

func newAutoScalingPlansStore() *autoScalingPlansStore {
	now := time.Now().UTC()
	seed := autoScalingPlansDefaultPlanRecord("stackyard-scaling-plan", 1, now)
	return &autoScalingPlansStore{
		plans: map[string]*autoScalingPlanRecord{
			seed.Name: seed,
		},
	}
}

func (s *autoScalingPlansStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateScalingPlan":
		return s.createScalingPlan(payload)
	case "DeleteScalingPlan":
		return s.deleteScalingPlan(payload)
	case "DescribeScalingPlanResources":
		return s.describeScalingPlanResources(payload)
	case "DescribeScalingPlans":
		return s.describeScalingPlans(payload)
	case "GetScalingPlanResourceForecastData":
		return s.getScalingPlanResourceForecastData(payload)
	case "UpdateScalingPlan":
		return s.updateScalingPlan(payload)
	default:
		return map[string]any{}
	}
}

func (s *autoScalingPlansStore) createScalingPlan(payload map[string]any) map[string]any {
	now := time.Now().UTC()
	name := autoScalingPlansPayloadString(payload, "ScalingPlanName", fmt.Sprintf("stackyard-scaling-plan-%06d", now.UnixNano()%1000000))
	plan, exists := s.plans[name]
	if !exists {
		plan = autoScalingPlansDefaultPlanRecord(name, 1, now)
		s.plans[name] = plan
	} else {
		plan.Version++
		plan.UpdatedTime = now
	}
	plan.ApplicationSource = autoScalingPlansPayloadMap(
		payload,
		"ApplicationSource",
		autoScalingPlansCloneMap(plan.ApplicationSource),
	)
	plan.ScalingInstructions = autoScalingPlansPayloadInstructionList(payload, "ScalingInstructions", plan.Name)
	plan.StatusCode = "Active"
	plan.StatusMessage = "ok"
	plan.StatusStartTime = now

	return map[string]any{
		"ScalingPlanVersion": plan.Version,
	}
}

func (s *autoScalingPlansStore) deleteScalingPlan(payload map[string]any) map[string]any {
	name := autoScalingPlansPayloadString(payload, "ScalingPlanName", "stackyard-scaling-plan")
	delete(s.plans, name)
	return map[string]any{}
}

func (s *autoScalingPlansStore) describeScalingPlanResources(payload map[string]any) map[string]any {
	name := autoScalingPlansPayloadString(payload, "ScalingPlanName", "stackyard-scaling-plan")
	plan := s.ensurePlanLocked(name)
	resources := make([]any, 0, len(plan.ScalingInstructions))
	for idx, instruction := range plan.ScalingInstructions {
		serviceNamespace := autoScalingPlansMapString(instruction, "ServiceNamespace", "autoscaling")
		resourceID := autoScalingPlansMapString(instruction, "ResourceId", fmt.Sprintf("autoScalingGroup/%s-asg", plan.Name))
		scalableDimension := autoScalingPlansMapString(
			instruction,
			"ScalableDimension",
			"autoscaling:autoScalingGroup:DesiredCapacity",
		)
		scalingPolicies := []any{
			map[string]any{
				"PolicyName": fmt.Sprintf("stackyard-policy-%02d", idx+1),
				"PolicyType": "TargetTrackingScaling",
				"TargetTrackingConfiguration": map[string]any{
					"TargetValue": 50.0,
				},
			},
		}
		resources = append(resources, map[string]any{
			"ScalingPlanName":    plan.Name,
			"ScalingPlanVersion": plan.Version,
			"ServiceNamespace":   serviceNamespace,
			"ResourceId":         resourceID,
			"ScalableDimension":  scalableDimension,
			"ScalingPolicies":    scalingPolicies,
			"ScalingStatusCode":  "Active",
			"ScalingStatusMessage": fmt.Sprintf(
				"resource %d is active",
				idx+1,
			),
		})
	}

	return map[string]any{
		"ScalingPlanResources": resources,
		"NextToken":            "",
	}
}

func (s *autoScalingPlansStore) describeScalingPlans(payload map[string]any) map[string]any {
	if len(s.plans) == 0 {
		_ = s.ensurePlanLocked("stackyard-scaling-plan")
	}

	filterNames := autoScalingPlansPayloadStringSlice(payload, "ScalingPlanNames")
	filterSet := map[string]struct{}{}
	for _, name := range filterNames {
		filterSet[name] = struct{}{}
	}
	versionFilter := autoScalingPlansPayloadInt(payload, "ScalingPlanVersion", 0)

	names := make([]string, 0, len(s.plans))
	for name := range s.plans {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]any, 0, len(names))
	for _, name := range names {
		plan := s.plans[name]
		if len(filterSet) > 0 {
			if _, ok := filterSet[name]; !ok {
				continue
			}
		}
		if versionFilter > 0 && plan.Version != versionFilter {
			continue
		}
		items = append(items, autoScalingPlansScalingPlanPayload(plan))
	}

	return map[string]any{
		"ScalingPlans": items,
		"NextToken":    "",
	}
}

func (s *autoScalingPlansStore) getScalingPlanResourceForecastData(payload map[string]any) map[string]any {
	name := autoScalingPlansPayloadString(payload, "ScalingPlanName", "stackyard-scaling-plan")
	plan := s.ensurePlanLocked(name)
	start := autoScalingPlansPayloadTime(payload, "StartTime", time.Now().UTC().Add(-3*time.Hour))
	end := autoScalingPlansPayloadTime(payload, "EndTime", start.Add(6*time.Hour))
	if !end.After(start) {
		end = start.Add(1 * time.Hour)
	}

	const points = 6
	step := end.Sub(start) / (points - 1)
	datapoints := make([]any, 0, points)
	base := float64(autoScalingPlansPayloadInt(payload, "ScalingPlanVersion", plan.Version))
	if base < 1 {
		base = 1
	}
	for i := 0; i < points; i++ {
		datapoints = append(datapoints, map[string]any{
			"Timestamp": start.Add(step * time.Duration(i)),
			"Value":     base*10.0 + float64(i),
		})
	}

	return map[string]any{
		"Datapoints": datapoints,
	}
}

func (s *autoScalingPlansStore) updateScalingPlan(payload map[string]any) map[string]any {
	now := time.Now().UTC()
	name := autoScalingPlansPayloadString(payload, "ScalingPlanName", "stackyard-scaling-plan")
	plan := s.ensurePlanLocked(name)
	plan.Version++
	plan.UpdatedTime = now
	plan.StatusStartTime = now
	plan.StatusCode = "Active"
	plan.StatusMessage = "updated"

	if _, ok := autoScalingPlansPayloadValue(payload, "ApplicationSource"); ok {
		plan.ApplicationSource = autoScalingPlansPayloadMap(
			payload,
			"ApplicationSource",
			autoScalingPlansCloneMap(plan.ApplicationSource),
		)
	}
	if _, ok := autoScalingPlansPayloadValue(payload, "ScalingInstructions"); ok {
		plan.ScalingInstructions = autoScalingPlansPayloadInstructionList(payload, "ScalingInstructions", plan.Name)
	}

	return map[string]any{}
}

func (s *autoScalingPlansStore) ensurePlanLocked(name string) *autoScalingPlanRecord {
	if strings.TrimSpace(name) == "" {
		name = "stackyard-scaling-plan"
	}
	if plan, ok := s.plans[name]; ok {
		return plan
	}
	now := time.Now().UTC()
	plan := autoScalingPlansDefaultPlanRecord(name, 1, now)
	s.plans[name] = plan
	return plan
}

func autoScalingPlansDefaultPlanRecord(name string, version int64, now time.Time) *autoScalingPlanRecord {
	return &autoScalingPlanRecord{
		Name:                name,
		Version:             version,
		ApplicationSource:   autoScalingPlansDefaultApplicationSource(),
		ScalingInstructions: autoScalingPlansDefaultScalingInstructions(name),
		StatusCode:          "Active",
		StatusMessage:       "ok",
		StatusStartTime:     now,
		CreationTime:        now,
		UpdatedTime:         now,
	}
}

func autoScalingPlansDefaultApplicationSource() map[string]any {
	return map[string]any{
		"CloudFormationStackARN": "arn:aws:cloudformation:us-east-1:123456789012:stack/stackyard/1",
		"TagFilters": []any{
			map[string]any{
				"Key":    "stackyard",
				"Values": []any{"true"},
			},
		},
	}
}

func autoScalingPlansDefaultScalingInstructions(name string) []map[string]any {
	return []map[string]any{
		{
			"ServiceNamespace":  "autoscaling",
			"ResourceId":        fmt.Sprintf("autoScalingGroup/%s-asg", strings.ReplaceAll(name, " ", "-")),
			"ScalableDimension": "autoscaling:autoScalingGroup:DesiredCapacity",
			"MinCapacity":       int64(1),
			"MaxCapacity":       int64(4),
			"TargetTrackingConfigurations": []any{
				map[string]any{
					"PredefinedScalingMetricSpecification": map[string]any{
						"PredefinedScalingMetricType": "ASGAverageCPUUtilization",
					},
					"TargetValue": 50.0,
				},
			},
			"PredefinedLoadMetricSpecification": map[string]any{
				"PredefinedLoadMetricType": "ASGTotalCPUUtilization",
			},
			"PredictiveScalingMode": "ForecastAndScale",
		},
	}
}

func autoScalingPlansScalingPlanPayload(plan *autoScalingPlanRecord) map[string]any {
	return map[string]any{
		"ScalingPlanName":    plan.Name,
		"ScalingPlanVersion": plan.Version,
		"ApplicationSource":  autoScalingPlansCloneMap(plan.ApplicationSource),
		"ScalingInstructions": func() []any {
			out := make([]any, 0, len(plan.ScalingInstructions))
			for _, instruction := range plan.ScalingInstructions {
				out = append(out, autoScalingPlansCloneMap(instruction))
			}
			return out
		}(),
		"StatusCode":      plan.StatusCode,
		"StatusMessage":   plan.StatusMessage,
		"StatusStartTime": plan.StatusStartTime,
		"CreationTime":    plan.CreationTime,
	}
}

func autoScalingPlansPayloadValue(payload map[string]any, key string) (any, bool) {
	if payload == nil {
		return nil, false
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return nil, false
}

func autoScalingPlansPayloadString(payload map[string]any, key, fallback string) string {
	if value, ok := autoScalingPlansPayloadValue(payload, key); ok {
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		default:
			text := strings.TrimSpace(fmt.Sprintf("%v", v))
			if text != "" {
				return text
			}
		}
	}
	return fallback
}

func autoScalingPlansPayloadInt(payload map[string]any, key string, fallback int64) int64 {
	value, ok := autoScalingPlansPayloadValue(payload, key)
	if !ok {
		return fallback
	}
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return n
		}
		if f, err := v.Float64(); err == nil {
			return int64(f)
		}
	case string:
		var n int64
		if _, err := fmt.Sscan(strings.TrimSpace(v), &n); err == nil {
			return n
		}
	}
	return fallback
}

func autoScalingPlansPayloadMap(payload map[string]any, key string, fallback map[string]any) map[string]any {
	value, ok := autoScalingPlansPayloadValue(payload, key)
	if !ok {
		return fallback
	}
	if typed, ok := value.(map[string]any); ok {
		return autoScalingPlansCloneMap(typed)
	}
	return fallback
}

func autoScalingPlansPayloadStringSlice(payload map[string]any, key string) []string {
	value, ok := autoScalingPlansPayloadValue(payload, key)
	if !ok {
		return nil
	}
	if list, ok := value.([]any); ok {
		out := make([]string, 0, len(list))
		for _, item := range list {
			text := strings.TrimSpace(fmt.Sprintf("%v", item))
			if text == "" {
				continue
			}
			out = append(out, text)
		}
		return out
	}
	return nil
}

func autoScalingPlansPayloadInstructionList(payload map[string]any, key, planName string) []map[string]any {
	value, ok := autoScalingPlansPayloadValue(payload, key)
	if !ok {
		return autoScalingPlansDefaultScalingInstructions(planName)
	}
	list, ok := value.([]any)
	if !ok {
		return autoScalingPlansDefaultScalingInstructions(planName)
	}
	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, autoScalingPlansCloneMap(m))
	}
	if len(out) == 0 {
		return autoScalingPlansDefaultScalingInstructions(planName)
	}
	return out
}

func autoScalingPlansPayloadTime(payload map[string]any, key string, fallback time.Time) time.Time {
	value, ok := autoScalingPlansPayloadValue(payload, key)
	if !ok {
		return fallback
	}
	switch v := value.(type) {
	case time.Time:
		return v.UTC()
	case string:
		text := strings.TrimSpace(v)
		if text == "" {
			return fallback
		}
		layouts := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
		}
		for _, layout := range layouts {
			if ts, err := time.Parse(layout, text); err == nil {
				return ts.UTC()
			}
		}
	case json.Number:
		if f, err := v.Float64(); err == nil {
			return time.Unix(int64(f), 0).UTC()
		}
	case float64:
		return time.Unix(int64(v), 0).UTC()
	case int64:
		return time.Unix(v, 0).UTC()
	case int:
		return time.Unix(int64(v), 0).UTC()
	}
	return fallback
}

func autoScalingPlansMapString(values map[string]any, key, fallback string) string {
	if values == nil {
		return fallback
	}
	for k, v := range values {
		if strings.EqualFold(k, key) {
			text := strings.TrimSpace(fmt.Sprintf("%v", v))
			if text != "" {
				return text
			}
		}
	}
	return fallback
}

func autoScalingPlansCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = autoScalingPlansCloneValue(v)
	}
	return out
}

func autoScalingPlansCloneValue(v any) any {
	switch typed := v.(type) {
	case map[string]any:
		return autoScalingPlansCloneMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = autoScalingPlansCloneValue(item)
		}
		return out
	default:
		return typed
	}
}
