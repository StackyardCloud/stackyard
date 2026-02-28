package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type resourceGroupsTaggingAPIStore struct {
	mu            sync.Mutex
	resources     map[string]map[string]string
	reportStarted bool
	reportTime    time.Time
}

func newResourceGroupsTaggingAPIStore() *resourceGroupsTaggingAPIStore {
	return &resourceGroupsTaggingAPIStore{
		resources: map[string]map[string]string{
			"arn:aws:s3:::stackyard-example-bucket": {
				"Environment": "dev",
				"Owner":       "stackyard",
			},
			"arn:aws:ec2:us-east-1:123456789012:instance/i-00000000000000001": {
				"Environment": "test",
				"Service":     "payments",
			},
		},
	}
}

func (s *resourceGroupsTaggingAPIStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureSeedLocked()

	switch action {
	case "DescribeReportCreation":
		if !s.reportStarted {
			return map[string]any{"Status": "NOT_STARTED"}
		}
		return map[string]any{
			"Status":       "SUCCEEDED",
			"S3Location":   "s3://stackyard-tagging-reports/aws-tagging-report.csv",
			"StartDate":    s.reportTime.UTC().Format(time.RFC3339),
			"ErrorMessage": "",
		}
	case "GetComplianceSummary":
		return map[string]any{
			"SummaryList": []any{
				map[string]any{
					"TargetId":              "123456789012",
					"TargetIdType":          "ACCOUNT",
					"Region":                "us-east-1",
					"ResourceType":          "s3:bucket",
					"NonCompliantResources": 0,
					"LastUpdated":           time.Now().UTC().Format(time.RFC3339),
				},
			},
			"PaginationToken": "",
		}
	case "GetResources":
		items := make([]any, 0, len(s.resources))
		arns := sortedResourceARNs(s.resources)
		for _, arn := range arns {
			tags := s.resources[arn]
			tagList := make([]any, 0, len(tags))
			for _, key := range sortedTagKeys(tags) {
				tagList = append(tagList, map[string]any{"Key": key, "Value": tags[key]})
			}
			items = append(items, map[string]any{
				"ResourceARN": arn,
				"Tags":        tagList,
				"ComplianceDetails": map[string]any{
					"NoncompliantKeys":           []any{},
					"KeysWithNoncompliantValues": []any{},
					"ComplianceStatus":           true,
				},
			})
		}
		return map[string]any{"ResourceTagMappingList": items, "PaginationToken": ""}
	case "GetTagKeys":
		keys := collectDistinctKeys(s.resources)
		return map[string]any{"TagKeys": keys, "PaginationToken": ""}
	case "GetTagValues":
		key := resourceGroupsTaggingPayloadString(payload, "Key", "")
		values := collectValuesForKey(s.resources, key)
		return map[string]any{"TagValues": values, "PaginationToken": ""}
	case "StartReportCreation":
		s.reportStarted = true
		s.reportTime = time.Now().UTC()
		return map[string]any{}
	case "TagResources":
		arns := resourceGroupsTaggingPayloadStrings(payload, "ResourceARNList")
		tags := resourceGroupsTaggingPayloadTagMap(payload, "Tags")
		for _, arn := range arns {
			if s.resources[arn] == nil {
				s.resources[arn] = map[string]string{}
			}
			for key, value := range tags {
				s.resources[arn][key] = value
			}
		}
		return map[string]any{"FailedResourcesMap": map[string]any{}}
	case "UntagResources":
		arns := resourceGroupsTaggingPayloadStrings(payload, "ResourceARNList")
		tagKeys := resourceGroupsTaggingPayloadStrings(payload, "TagKeys")
		for _, arn := range arns {
			if s.resources[arn] == nil {
				continue
			}
			for _, key := range tagKeys {
				delete(s.resources[arn], key)
			}
		}
		return map[string]any{"FailedResourcesMap": map[string]any{}}
	}

	return map[string]any{}
}

func (s *resourceGroupsTaggingAPIStore) ensureSeedLocked() {
	if len(s.resources) != 0 {
		return
	}
	s.resources["arn:aws:s3:::stackyard-example-bucket"] = map[string]string{"Environment": "dev", "Owner": "stackyard"}
}

func resourceGroupsTaggingPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	for k, v := range payload {
		if strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			if s := strings.TrimSpace(fmt.Sprintf("%v", v)); s != "" && s != "%!v(<nil>)" {
				return s
			}
		}
	}
	return fallback
}

func resourceGroupsTaggingPayloadStrings(payload map[string]any, key string) []string {
	if payload == nil {
		return []string{}
	}
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		list, ok := v.([]any)
		if !ok {
			return []string{}
		}
		out := make([]string, 0, len(list))
		for _, item := range list {
			if s := strings.TrimSpace(fmt.Sprintf("%v", item)); s != "" && s != "%!v(<nil>)" {
				out = append(out, s)
			}
		}
		return out
	}
	return []string{}
}

func resourceGroupsTaggingPayloadTagMap(payload map[string]any, key string) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), strings.TrimSpace(key)) {
			continue
		}
		m, ok := v.(map[string]any)
		if !ok {
			return out
		}
		for tagKey, tagValue := range m {
			trimmedKey := strings.TrimSpace(tagKey)
			if trimmedKey == "" {
				continue
			}
			out[trimmedKey] = strings.TrimSpace(fmt.Sprintf("%v", tagValue))
		}
		return out
	}
	return out
}

func sortedResourceARNs(resources map[string]map[string]string) []string {
	keys := make([]string, 0, len(resources))
	for arn := range resources {
		keys = append(keys, arn)
	}
	sort.Strings(keys)
	return keys
}

func sortedTagKeys(tags map[string]string) []string {
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func collectDistinctKeys(resources map[string]map[string]string) []string {
	set := map[string]struct{}{}
	for _, tags := range resources {
		for key := range tags {
			set[key] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func collectValuesForKey(resources map[string]map[string]string, key string) []string {
	if strings.TrimSpace(key) == "" {
		return []string{}
	}
	set := map[string]struct{}{}
	for _, tags := range resources {
		if value, ok := tags[key]; ok {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
