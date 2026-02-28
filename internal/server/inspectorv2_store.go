package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
)

type inspectorV2Store struct {
	mu sync.Mutex

	nextID int64

	filters map[string]map[string]any
	tags    map[string]map[string]string
}

func newInspectorV2Store() *inspectorV2Store {
	return &inspectorV2Store{
		nextID:  1,
		filters: map[string]map[string]any{},
		tags:    map[string]map[string]string{},
	}
}

func (s *inspectorV2Store) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++

	switch action {
	case "CreateFilter":
		name := inspectorV2PayloadString(payload, "name", fmt.Sprintf("stackyard-filter-%06d", s.nextID))
		filterArn := fmt.Sprintf("arn:aws:inspector2:us-east-1:123456789012:filter/%s", strings.ReplaceAll(name, " ", "-"))
		entry := map[string]any{
			"arn":            filterArn,
			"name":           name,
			"action":         inspectorV2PayloadString(payload, "action", "NONE"),
			"filterCriteria": map[string]any{},
		}
		s.filters[filterArn] = entry
		return map[string]any{"arn": filterArn}

	case "ListFilters":
		items := make([]string, 0, len(s.filters))
		for arn := range s.filters {
			items = append(items, arn)
		}
		sort.Strings(items)
		out := make([]any, 0, len(items))
		for _, arn := range items {
			out = append(out, cloneAnyMap(s.filters[arn]))
		}
		return map[string]any{"filters": out, "nextToken": ""}

	case "DeleteFilter":
		arn := inspectorV2PayloadString(payload, "arn", "")
		if arn != "" {
			delete(s.filters, arn)
		}
		return map[string]any{}

	case "TagResource":
		resourceArn := strings.TrimSpace(pathParams["resourceArn"])
		if resourceArn == "" {
			resourceArn = inspectorV2PayloadString(payload, "resourceArn", "arn:aws:inspector2:us-east-1:123456789012:target/stackyard")
		}
		if resourceArn == "" {
			resourceArn = "arn:aws:inspector2:us-east-1:123456789012:target/stackyard"
		}
		tags := s.ensureTagsLocked(resourceArn)
		if raw, ok := payload["tags"].(map[string]any); ok {
			for k, v := range raw {
				key := strings.TrimSpace(k)
				if key == "" {
					continue
				}
				tags[key] = strings.TrimSpace(fmt.Sprintf("%v", v))
			}
		}
		if raw, ok := payload["tags"].(map[string]string); ok {
			for k, v := range raw {
				key := strings.TrimSpace(k)
				if key == "" {
					continue
				}
				tags[key] = strings.TrimSpace(v)
			}
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceArn := strings.TrimSpace(pathParams["resourceArn"])
		if resourceArn == "" {
			resourceArn = inspectorV2PayloadString(payload, "resourceArn", "arn:aws:inspector2:us-east-1:123456789012:target/stackyard")
		}
		tags := s.ensureTagsLocked(resourceArn)
		out := map[string]any{}
		for k, v := range tags {
			out[k] = v
		}
		return map[string]any{"tags": out}

	case "UntagResource":
		resourceArn := strings.TrimSpace(pathParams["resourceArn"])
		if resourceArn == "" {
			resourceArn = inspectorV2PayloadString(payload, "resourceArn", "arn:aws:inspector2:us-east-1:123456789012:target/stackyard")
		}
		tags := s.ensureTagsLocked(resourceArn)
		for _, k := range query["tagKeys"] {
			for _, part := range strings.Split(k, ",") {
				if key := strings.TrimSpace(part); key != "" {
					delete(tags, key)
				}
			}
		}
		return map[string]any{}

	case "ListFindings":
		return map[string]any{"findings": []any{}, "nextToken": ""}

	case "ListCoverage":
		return map[string]any{"coveredResources": []any{}, "nextToken": ""}

	case "ListMembers":
		return map[string]any{"members": []any{}, "nextToken": ""}

	case "BatchGetAccountStatus":
		return map[string]any{
			"accounts": []any{
				map[string]any{
					"accountId": "123456789012",
					"state":     map[string]any{"status": "ENABLED"},
				},
			},
			"failedAccounts": []any{},
		}

	default:
		return map[string]any{}
	}
}

func (s *inspectorV2Store) ensureTagsLocked(resourceArn string) map[string]string {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		resourceArn = "arn:aws:inspector2:us-east-1:123456789012:target/stackyard"
	}
	if existing, ok := s.tags[resourceArn]; ok {
		return existing
	}
	t := map[string]string{}
	s.tags[resourceArn] = t
	return t
}

func inspectorV2PayloadString(payload map[string]any, key, fallback string) string {
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		if s, ok := v.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return s
			}
		}
	}
	return fallback
}
