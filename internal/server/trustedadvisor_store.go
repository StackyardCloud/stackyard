package server

import (
	"sort"
	"strings"
	"sync"
)

type trustedAdvisorStore struct {
	mu sync.Mutex

	checks                      map[string]map[string]any
	recommendations             map[string]map[string]any
	recommendationResources     map[string][]map[string]any
	organizationRecommendations map[string]map[string]any
	organizationResources       map[string][]map[string]any
	organizationAccounts        map[string][]map[string]any
}

func newTrustedAdvisorStore() *trustedAdvisorStore {
	checkID := "check-ec2-ri-optimization"
	recommendationID := "rec-ec2-ri-001"
	orgRecommendationID := "org-rec-ec2-ri-001"

	s := &trustedAdvisorStore{
		checks: map[string]map[string]any{
			checkID: {
				"arn":         "arn:aws:trustedadvisor:::check/" + checkID,
				"description": "Checks for underutilized EC2 reserved instances",
				"id":          checkID,
				"metadata":    []any{"Region", "Resource", "Savings"},
				"name":        "EC2 Reserved Instance Optimization",
				"pillars":     []any{"cost_optimizing"},
				"source":      "aws",
			},
		},
		recommendations: map[string]map[string]any{
			recommendationID: {
				"arn":            "arn:aws:trustedadvisor:::recommendation/" + recommendationID,
				"checkArn":       "arn:aws:trustedadvisor:::check/" + checkID,
				"id":             recommendationID,
				"lifecycleStage": "resolved",
				"name":           "Reduce underutilized EC2 Reserved Instances",
				"pillars":        []any{"cost_optimizing"},
				"resourcesAggregates": map[string]any{
					"errorCount":   0,
					"okCount":      1,
					"warningCount": 0,
				},
				"pillarSpecificAggregates": map[string]any{
					"costOptimizing": map[string]any{
						"estimatedMonthlySavings":        42.12,
						"estimatedPercentMonthlySavings": 7.6,
					},
				},
				"source": "aws",
				"status": "ok",
			},
		},
		recommendationResources: map[string][]map[string]any{
			recommendationID: {
				{
					"arn":        "arn:aws:ec2:us-east-1:123456789012:instance/i-00000000000000001",
					"id":         "i-00000000000000001",
					"isExcluded": false,
					"metadata":   []any{"us-east-1", "m6i.large", "42.12"},
					"regionCode": "us-east-1",
					"status":     "ok",
				},
			},
		},
		organizationRecommendations: map[string]map[string]any{
			orgRecommendationID: {
				"arn":            "arn:aws:trustedadvisor:::organization-recommendation/" + orgRecommendationID,
				"id":             orgRecommendationID,
				"lifecycleStage": "in_progress",
				"name":           "Organization RI Optimization",
				"pillars":        []any{"cost_optimizing"},
				"resourcesAggregates": map[string]any{
					"errorCount":   0,
					"okCount":      2,
					"warningCount": 0,
				},
				"source": "aws",
				"status": "ok",
			},
		},
		organizationResources: map[string][]map[string]any{
			orgRecommendationID: {
				{
					"arn":        "arn:aws:ec2:us-east-1:111111111111:instance/i-00000000000000011",
					"id":         "i-00000000000000011",
					"isExcluded": false,
					"metadata":   []any{"us-east-1", "m6i.large", "20.00"},
					"regionCode": "us-east-1",
					"status":     "ok",
				},
				{
					"arn":        "arn:aws:ec2:us-east-1:222222222222:instance/i-00000000000000022",
					"id":         "i-00000000000000022",
					"isExcluded": false,
					"metadata":   []any{"us-east-1", "m6i.xlarge", "22.12"},
					"regionCode": "us-east-1",
					"status":     "ok",
				},
			},
		},
		organizationAccounts: map[string][]map[string]any{
			orgRecommendationID: {
				{
					"accountId":             "111111111111",
					"affectedResourceCount": int64(1),
					"lifecycleSummary": map[string]any{
						"inProgressCount": int64(0),
						"resolvedCount":   int64(1),
						"dismissedCount":  int64(0),
					},
				},
				{
					"accountId":             "222222222222",
					"affectedResourceCount": int64(1),
					"lifecycleSummary": map[string]any{
						"inProgressCount": int64(1),
						"resolvedCount":   int64(0),
						"dismissedCount":  int64(0),
					},
				},
			},
		},
	}
	return s
}

func (s *trustedAdvisorStore) Handle(action string, payload map[string]any, pathParams map[string]string, _ map[string][]string) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "BatchUpdateRecommendationResourceExclusion":
		exclusions := trustedAdvisorPayloadSlice(payload, "resourceExclusions")
		errorsOut := make([]any, 0)
		for _, item := range exclusions {
			entry, ok := item.(map[string]any)
			if !ok {
				continue
			}
			resourceArn := trustedAdvisorPayloadString(entry, "arn", "")
			resourceExcluded := trustedAdvisorPayloadBool(entry, "isExcluded", false)
			updated := false
			for recommendationID, resources := range s.recommendationResources {
				for i := range resources {
					resource := resources[i]
					if trustedAdvisorPayloadString(resource, "arn", "") == resourceArn {
						resource["isExcluded"] = resourceExcluded
						s.recommendationResources[recommendationID][i] = trustedAdvisorCloneMap(resource)
						updated = true
					}
				}
			}
			if !updated && strings.TrimSpace(resourceArn) != "" {
				errorsOut = append(errorsOut, map[string]any{
					"arn":          resourceArn,
					"errorCode":    "ResourceNotFoundException",
					"errorMessage": "resource not found",
				})
			}
		}
		return map[string]any{"batchUpdateRecommendationResourceExclusionErrors": errorsOut}

	case "GetOrganizationRecommendation":
		id := trustedAdvisorPathParam(pathParams, "organizationRecommendationIdentifier", s.firstOrganizationRecommendationIDLocked())
		org := s.ensureOrganizationRecommendationLocked(id)
		return map[string]any{"organizationRecommendation": trustedAdvisorCloneMap(org)}

	case "GetRecommendation":
		id := trustedAdvisorPathParam(pathParams, "recommendationIdentifier", s.firstRecommendationIDLocked())
		rec := s.ensureRecommendationLocked(id)
		return map[string]any{"recommendation": trustedAdvisorCloneMap(rec)}

	case "ListChecks":
		checks := make([]any, 0, len(s.checks))
		for _, c := range trustedAdvisorSortedMapValues(s.checks) {
			checks = append(checks, trustedAdvisorCloneMap(c))
		}
		return map[string]any{"checkSummaries": checks, "nextToken": ""}

	case "ListOrganizationRecommendationAccounts":
		id := trustedAdvisorPathParam(pathParams, "organizationRecommendationIdentifier", s.firstOrganizationRecommendationIDLocked())
		s.ensureOrganizationRecommendationLocked(id)
		accounts := trustedAdvisorCloneMapSlice(s.organizationAccounts[id])
		return map[string]any{"accountRecommendationLifecycleSummaries": accounts, "nextToken": ""}

	case "ListOrganizationRecommendationResources":
		id := trustedAdvisorPathParam(pathParams, "organizationRecommendationIdentifier", s.firstOrganizationRecommendationIDLocked())
		s.ensureOrganizationRecommendationLocked(id)
		resources := trustedAdvisorCloneMapSlice(s.organizationResources[id])
		return map[string]any{"organizationRecommendationResourceSummaries": resources, "nextToken": ""}

	case "ListOrganizationRecommendations":
		out := make([]any, 0, len(s.organizationRecommendations))
		for _, org := range trustedAdvisorSortedMapValues(s.organizationRecommendations) {
			out = append(out, trustedAdvisorCloneMap(org))
		}
		return map[string]any{"organizationRecommendationSummaries": out, "nextToken": ""}

	case "ListRecommendationResources":
		id := trustedAdvisorPathParam(pathParams, "recommendationIdentifier", s.firstRecommendationIDLocked())
		s.ensureRecommendationLocked(id)
		resources := trustedAdvisorCloneMapSlice(s.recommendationResources[id])
		return map[string]any{"recommendationResourceSummaries": resources, "nextToken": ""}

	case "ListRecommendations":
		out := make([]any, 0, len(s.recommendations))
		for _, rec := range trustedAdvisorSortedMapValues(s.recommendations) {
			out = append(out, trustedAdvisorCloneMap(rec))
		}
		return map[string]any{"recommendationSummaries": out, "nextToken": ""}

	case "UpdateOrganizationRecommendationLifecycle":
		id := trustedAdvisorPathParam(pathParams, "organizationRecommendationIdentifier", s.firstOrganizationRecommendationIDLocked())
		org := s.ensureOrganizationRecommendationLocked(id)
		stage := trustedAdvisorNormalizeLifecycle(trustedAdvisorPayloadString(payload, "lifecycleStage", trustedAdvisorPayloadString(org, "lifecycleStage", "in_progress")))
		org["lifecycleStage"] = stage
		return map[string]any{"organizationRecommendation": trustedAdvisorCloneMap(org)}

	case "UpdateRecommendationLifecycle":
		id := trustedAdvisorPathParam(pathParams, "recommendationIdentifier", s.firstRecommendationIDLocked())
		rec := s.ensureRecommendationLocked(id)
		stage := trustedAdvisorNormalizeLifecycle(trustedAdvisorPayloadString(payload, "lifecycleStage", trustedAdvisorPayloadString(rec, "lifecycleStage", "in_progress")))
		rec["lifecycleStage"] = stage
		return map[string]any{"recommendation": trustedAdvisorCloneMap(rec)}
	}

	return map[string]any{}
}

func (s *trustedAdvisorStore) ensureRecommendationLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.firstRecommendationIDLocked()
	}
	if rec, ok := s.recommendations[id]; ok {
		return rec
	}
	rec := map[string]any{
		"arn":                 "arn:aws:trustedadvisor:::recommendation/" + id,
		"checkArn":            "arn:aws:trustedadvisor:::check/check-stackyard",
		"id":                  id,
		"lifecycleStage":      "in_progress",
		"name":                "Stackyard recommendation",
		"pillars":             []any{"operational_excellence"},
		"resourcesAggregates": map[string]any{"errorCount": 0, "okCount": 1, "warningCount": 0},
		"source":              "aws",
		"status":              "ok",
	}
	s.recommendations[id] = rec
	if _, ok := s.recommendationResources[id]; !ok {
		s.recommendationResources[id] = []map[string]any{
			{
				"arn":        "arn:aws:ec2:us-east-1:123456789012:instance/i-stackyard",
				"id":         "i-stackyard",
				"isExcluded": false,
				"metadata":   []any{"us-east-1", "t3.medium", "0.00"},
				"regionCode": "us-east-1",
				"status":     "ok",
			},
		}
	}
	return rec
}

func (s *trustedAdvisorStore) ensureOrganizationRecommendationLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.firstOrganizationRecommendationIDLocked()
	}
	if org, ok := s.organizationRecommendations[id]; ok {
		return org
	}
	org := map[string]any{
		"arn":                 "arn:aws:trustedadvisor:::organization-recommendation/" + id,
		"id":                  id,
		"lifecycleStage":      "in_progress",
		"name":                "Stackyard organization recommendation",
		"pillars":             []any{"security"},
		"resourcesAggregates": map[string]any{"errorCount": 0, "okCount": 1, "warningCount": 0},
		"source":              "aws",
		"status":              "ok",
	}
	s.organizationRecommendations[id] = org
	if _, ok := s.organizationResources[id]; !ok {
		s.organizationResources[id] = []map[string]any{
			{
				"arn":        "arn:aws:ec2:us-east-1:123456789012:instance/i-org-stackyard",
				"id":         "i-org-stackyard",
				"isExcluded": false,
				"metadata":   []any{"us-east-1", "t3.large", "0.00"},
				"regionCode": "us-east-1",
				"status":     "ok",
			},
		}
	}
	if _, ok := s.organizationAccounts[id]; !ok {
		s.organizationAccounts[id] = []map[string]any{
			{
				"accountId":             "123456789012",
				"affectedResourceCount": int64(1),
				"lifecycleSummary": map[string]any{
					"inProgressCount": int64(1),
					"resolvedCount":   int64(0),
					"dismissedCount":  int64(0),
				},
			},
		}
	}
	return org
}

func (s *trustedAdvisorStore) firstRecommendationIDLocked() string {
	if len(s.recommendations) == 0 {
		return "rec-stackyard-000001"
	}
	keys := make([]string, 0, len(s.recommendations))
	for key := range s.recommendations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *trustedAdvisorStore) firstOrganizationRecommendationIDLocked() string {
	if len(s.organizationRecommendations) == 0 {
		return "org-rec-stackyard-000001"
	}
	keys := make([]string, 0, len(s.organizationRecommendations))
	for key := range s.organizationRecommendations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func trustedAdvisorNormalizeLifecycle(stage string) string {
	switch strings.ToLower(strings.TrimSpace(stage)) {
	case "dismissed", "in_progress", "resolved":
		return strings.ToLower(strings.TrimSpace(stage))
	default:
		return "in_progress"
	}
}

func trustedAdvisorPayloadString(payload map[string]any, key, def string) string {
	if payload == nil {
		return strings.TrimSpace(def)
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return strings.TrimSpace(def)
	}
	switch v := value.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return strings.TrimSpace(def)
		}
		return trimmed
	default:
		return strings.TrimSpace(def)
	}
}

func trustedAdvisorPathParam(params map[string]string, key, def string) string {
	if params == nil {
		return strings.TrimSpace(def)
	}
	value, ok := params[key]
	if !ok {
		return strings.TrimSpace(def)
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return strings.TrimSpace(def)
	}
	return value
}

func trustedAdvisorPayloadBool(payload map[string]any, key string, def bool) bool {
	if payload == nil {
		return def
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return def
	}
	b, ok := value.(bool)
	if !ok {
		return def
	}
	return b
}

func trustedAdvisorPayloadSlice(payload map[string]any, key string) []any {
	if payload == nil {
		return nil
	}
	value, ok := payload[key]
	if !ok || value == nil {
		return nil
	}
	out, ok := value.([]any)
	if !ok {
		return nil
	}
	return out
}

func trustedAdvisorSortedMapValues(input map[string]map[string]any) []map[string]any {
	if len(input) == 0 {
		return nil
	}
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, trustedAdvisorCloneMap(input[key]))
	}
	return out
}

func trustedAdvisorCloneMapSlice(in []map[string]any) []any {
	if len(in) == 0 {
		return []any{}
	}
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, trustedAdvisorCloneMap(item))
	}
	return out
}

func trustedAdvisorCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = trustedAdvisorCloneValue(value)
	}
	return out
}

func trustedAdvisorCloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return trustedAdvisorCloneMap(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = trustedAdvisorCloneValue(typed[i])
		}
		return out
	default:
		return typed
	}
}
