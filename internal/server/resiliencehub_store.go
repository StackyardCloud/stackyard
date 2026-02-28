package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type resilienceHubStore struct {
	mu sync.Mutex

	app                    map[string]any
	policy                 map[string]any
	recommendationTemplate map[string]any
	assessment             map[string]any
	metricsExport          map[string]any
	groupingTask           map[string]any
	tags                   map[string]map[string]string
}

func newResilienceHubStore() *resilienceHubStore {
	now := time.Now().UTC().Format(time.RFC3339)
	appARN := "arn:aws:resiliencehub:us-east-1:123456789012:app/stackyard-app"
	policyARN := "arn:aws:resiliencehub:us-east-1:123456789012:resiliency-policy/stackyard-policy"
	templateARN := "arn:aws:resiliencehub:us-east-1:123456789012:recommendation-template/stackyard-template"
	assessmentARN := "arn:aws:resiliencehub:us-east-1:123456789012:app-assessment/stackyard-assessment"
	metricsExportARN := "arn:aws:resiliencehub:us-east-1:123456789012:metrics-export/stackyard-export"
	groupingTaskARN := "arn:aws:resiliencehub:us-east-1:123456789012:resource-grouping-recommendation-task/stackyard-task"

	return &resilienceHubStore{
		app: map[string]any{
			"appArn":           appARN,
			"name":             "stackyard-app",
			"description":      "Stackyard seeded Resilience Hub app",
			"complianceStatus": "PolicyBreached",
			"policyArn":        policyARN,
			"creationTime":     now,
		},
		policy: map[string]any{
			"policyArn":              policyARN,
			"policyName":             "stackyard-policy",
			"tier":                   "MissionCritical",
			"creationTime":           now,
			"dataLocationConstraint": "AnyLocation",
		},
		recommendationTemplate: map[string]any{
			"recommendationTemplateArn": templateARN,
			"name":                      "stackyard-template",
			"format":                    "CfnYaml",
			"status":                    "Active",
			"creationTime":              now,
		},
		assessment: map[string]any{
			"assessmentArn":    assessmentARN,
			"appArn":           appARN,
			"invoker":          "User",
			"assessmentStatus": "Success",
			"startTime":        now,
		},
		metricsExport: map[string]any{
			"metricsExportId":  "stackyard-export",
			"metricsExportArn": metricsExportARN,
			"status":           "Success",
			"creationTime":     now,
		},
		groupingTask: map[string]any{
			"taskArn":      groupingTaskARN,
			"appArn":       appARN,
			"status":       "Success",
			"creationTime": now,
		},
		tags: map[string]map[string]string{
			appARN:           {"stackyard": "true"},
			policyARN:        {"stackyard": "true"},
			templateARN:      {"stackyard": "true"},
			assessmentARN:    {"stackyard": "true"},
			metricsExportARN: {"stackyard": "true"},
			groupingTaskARN:  {"stackyard": "true"},
		},
	}
}

func (s *resilienceHubStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.syncPayloadWithQuery(payload, query)

	switch action {
	case "CreateApp", "UpdateApp":
		s.mergeKnownFields(s.app, payload)
		return map[string]any{"app": rhCloneMap(s.app)}
	case "DescribeApp":
		return map[string]any{"app": rhCloneMap(s.app)}
	case "DeleteApp":
		return map[string]any{}

	case "CreateResiliencyPolicy", "UpdateResiliencyPolicy":
		s.mergeKnownFields(s.policy, payload)
		return map[string]any{"policy": rhCloneMap(s.policy)}
	case "DescribeResiliencyPolicy":
		return map[string]any{"policy": rhCloneMap(s.policy)}
	case "DeleteResiliencyPolicy":
		return map[string]any{}

	case "CreateRecommendationTemplate":
		s.mergeKnownFields(s.recommendationTemplate, payload)
		return map[string]any{"recommendationTemplate": rhCloneMap(s.recommendationTemplate)}
	case "DeleteRecommendationTemplate":
		return map[string]any{}

	case "DescribeAppAssessment":
		return map[string]any{"assessment": rhCloneMap(s.assessment)}
	case "DeleteAppAssessment":
		return map[string]any{}
	case "DescribeMetricsExport":
		return map[string]any{"metricsExport": rhCloneMap(s.metricsExport)}
	case "DescribeResourceGroupingRecommendationTask":
		return map[string]any{"resourceGroupingRecommendationTask": rhCloneMap(s.groupingTask)}
	case "DescribeAppVersion":
		return map[string]any{"appVersion": map[string]any{"appArn": rhString(s.app["appArn"], ""), "version": "v1", "status": "Active"}}
	case "DescribeAppVersionAppComponent":
		return map[string]any{"appComponent": map[string]any{"name": "stackyard-component", "type": "AWS::Lambda::Function"}}
	case "DescribeAppVersionResource":
		return map[string]any{"appVersionResource": map[string]any{"logicalResourceId": map[string]any{"identifier": "stackyard-resource"}}}
	case "DescribeAppVersionResourcesResolutionStatus":
		return map[string]any{"resolutionStatus": map[string]any{"status": "Resolved"}}
	case "DescribeAppVersionTemplate":
		return map[string]any{"appTemplateBody": "AWSTemplateFormatVersion: '2010-09-09'"}
	case "DescribeDraftAppVersionResourcesImportStatus":
		return map[string]any{"status": "Success"}

	case "ListApps":
		return map[string]any{"appSummaries": []any{rhCloneMap(s.app)}, "nextToken": ""}
	case "ListResiliencyPolicies":
		return map[string]any{"resiliencyPolicies": []any{rhCloneMap(s.policy)}, "nextToken": ""}
	case "ListRecommendationTemplates":
		return map[string]any{"recommendationTemplates": []any{rhCloneMap(s.recommendationTemplate)}, "nextToken": ""}
	case "ListSuggestedResiliencyPolicies":
		return map[string]any{"resiliencyPolicies": []any{rhCloneMap(s.policy)}, "nextToken": ""}
	case "ListAppAssessments":
		return map[string]any{"assessmentSummaries": []any{rhCloneMap(s.assessment)}, "nextToken": ""}
	case "ListMetrics":
		return map[string]any{"metrics": []any{map[string]any{"name": "availability", "value": 99.9}}, "nextToken": ""}
	case "ListAppVersionResources":
		return map[string]any{"appVersionResources": []any{map[string]any{"logicalResourceId": map[string]any{"identifier": "stackyard-resource"}}}, "nextToken": ""}
	case "ListAppVersionResourceMappings":
		return map[string]any{"resourceMappings": []any{map[string]any{"mappingType": "CfnStack"}}, "nextToken": ""}
	case "ListAppVersionAppComponents":
		return map[string]any{"appComponents": []any{map[string]any{"name": "stackyard-component"}}, "nextToken": ""}
	case "ListAppVersions":
		return map[string]any{"appVersions": []any{map[string]any{"version": "v1", "status": "Active"}}, "nextToken": ""}
	case "ListAppInputSources":
		return map[string]any{"appInputSources": []any{map[string]any{"sourceName": "stackyard-source", "importType": "CloudFormation"}}, "nextToken": ""}
	case "ListAlarmRecommendations":
		return map[string]any{"alarmRecommendations": []any{}, "nextToken": ""}
	case "ListSopRecommendations":
		return map[string]any{"sopRecommendations": []any{}, "nextToken": ""}
	case "ListTestRecommendations":
		return map[string]any{"testRecommendations": []any{}, "nextToken": ""}
	case "ListResourceGroupingRecommendations":
		return map[string]any{"groupingRecommendations": []any{}, "nextToken": ""}
	case "ListAppAssessmentComplianceDrifts":
		return map[string]any{"complianceDrifts": []any{}, "nextToken": ""}
	case "ListAppAssessmentResourceDrifts":
		return map[string]any{"resourceDrifts": []any{}, "nextToken": ""}
	case "ListAppComponentCompliances":
		return map[string]any{"componentCompliances": []any{}, "nextToken": ""}
	case "ListAppComponentRecommendations":
		return map[string]any{"componentRecommendations": []any{}, "nextToken": ""}
	case "ListUnsupportedAppVersionResources":
		return map[string]any{"unsupportedResources": []any{}, "nextToken": ""}

	case "TagResource":
		resourceARN := rhResourceARN(pathParams, payload, rhString(s.app["appArn"], ""))
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		rhMergeTags(s.tags[resourceARN], payload["tags"])
		return map[string]any{}
	case "UntagResource":
		resourceARN := rhResourceARN(pathParams, payload, rhString(s.app["appArn"], ""))
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		tagKeys := rhTagKeys(payload)
		for _, key := range tagKeys {
			delete(s.tags[resourceARN], key)
		}
		return map[string]any{}
	case "ListTagsForResource":
		resourceARN := rhResourceARN(pathParams, payload, rhString(s.app["appArn"], ""))
		if s.tags[resourceARN] == nil {
			s.tags[resourceARN] = map[string]string{}
		}
		return map[string]any{"tags": rhCloneTags(s.tags[resourceARN])}
	}

	if strings.HasPrefix(action, "List") {
		return map[string]any{"items": []any{}, "nextToken": ""}
	}
	if strings.HasPrefix(action, "Describe") {
		return map[string]any{"status": "Success"}
	}
	if strings.HasPrefix(action, "Create") {
		return map[string]any{"status": "Created", "arn": rhString(s.app["appArn"], "")}
	}
	if strings.HasPrefix(action, "Update") {
		return map[string]any{"status": "Updated", "arn": rhString(s.app["appArn"], "")}
	}
	if strings.HasPrefix(action, "Delete") || strings.HasPrefix(action, "Remove") {
		return map[string]any{}
	}
	if strings.HasPrefix(action, "Start") || strings.HasPrefix(action, "Accept") || strings.HasPrefix(action, "Reject") || strings.HasPrefix(action, "Put") || strings.HasPrefix(action, "Publish") || strings.HasPrefix(action, "Resolve") || strings.HasPrefix(action, "Import") || strings.HasPrefix(action, "Batch") || strings.HasPrefix(action, "Add") {
		return map[string]any{"status": "Success"}
	}

	return map[string]any{}
}

func (s *resilienceHubStore) mergeKnownFields(target map[string]any, source map[string]any) {
	for key, value := range source {
		if strings.TrimSpace(key) == "" || value == nil {
			continue
		}
		target[key] = value
	}
}

func (s *resilienceHubStore) syncPayloadWithQuery(payload map[string]any, query url.Values) {
	if payload == nil {
		return
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		if _, exists := payload[key]; exists {
			continue
		}
		payload[key] = values[len(values)-1]
	}
}

func rhResourceARN(pathParams map[string]string, payload map[string]any, fallback string) string {
	if value := strings.TrimSpace(pathParams["resourceArn"]); value != "" {
		return value
	}
	if value := strings.TrimSpace(rhStringAny(payload, "resourceArn", "")); value != "" {
		return value
	}
	if value := strings.TrimSpace(rhStringAny(payload, "ResourceArn", "")); value != "" {
		return value
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "arn:aws:resiliencehub:us-east-1:123456789012:app/stackyard-app"
}

func rhMergeTags(dst map[string]string, raw any) {
	if dst == nil {
		return
	}
	if m, ok := raw.(map[string]any); ok {
		for key, value := range m {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			dst[key] = strings.TrimSpace(fmt.Sprintf("%v", value))
		}
		return
	}
	if m, ok := raw.(map[string]string); ok {
		for key, value := range m {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			dst[key] = strings.TrimSpace(value)
		}
	}
}

func rhTagKeys(payload map[string]any) []string {
	out := []string{}
	seen := map[string]struct{}{}
	appendKey := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}

	if raw, ok := payload["tagKeys"]; ok {
		switch values := raw.(type) {
		case []any:
			for _, value := range values {
				appendKey(fmt.Sprintf("%v", value))
			}
		case []string:
			for _, value := range values {
				appendKey(value)
			}
		case string:
			appendKey(values)
		}
	}

	if raw, ok := payload["TagKeys"]; ok {
		switch values := raw.(type) {
		case []any:
			for _, value := range values {
				appendKey(fmt.Sprintf("%v", value))
			}
		case []string:
			for _, value := range values {
				appendKey(value)
			}
		case string:
			appendKey(values)
		}
	}

	sort.Strings(out)
	return out
}

func rhCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func rhCloneTags(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func rhStringAny(values map[string]any, key, fallback string) string {
	if values == nil {
		return fallback
	}
	if v, ok := values[key]; ok {
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s != "" {
			return s
		}
	}
	return fallback
}

func rhString(value any, fallback string) string {
	if value == nil {
		return fallback
	}
	s := strings.TrimSpace(fmt.Sprintf("%v", value))
	if s == "" {
		return fallback
	}
	return s
}
