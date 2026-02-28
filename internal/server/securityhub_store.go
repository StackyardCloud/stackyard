package server

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type securityHubStore struct {
	mu sync.Mutex

	enabled bool
	tags    map[string]map[string]string
}

func newSecurityHubStore() *securityHubStore {
	return &securityHubStore{
		enabled: true,
		tags:    map[string]map[string]string{},
	}
}

func (s *securityHubStore) Handle(action string, payload map[string]any, query map[string][]string) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	resourceArn := securityHubPayloadString(payload, "ResourceArn", "arn:aws:securityhub:us-east-1:123456789012:hub/default")

	switch action {
	case "EnableSecurityHub":
		s.enabled = true
		return map[string]any{"HubArn": "arn:aws:securityhub:us-east-1:123456789012:hub/default"}
	case "DisableSecurityHub":
		s.enabled = false
		return map[string]any{}
	case "DescribeHub":
		return map[string]any{
			"HubArn":             "arn:aws:securityhub:us-east-1:123456789012:hub/default",
			"SubscribedAt":       now,
			"AutoEnableControls": true,
		}
	case "ListEnabledProductsForImport":
		return map[string]any{"ProductSubscriptions": []any{}, "NextToken": ""}
	case "ListOrganizationAdminAccounts":
		return map[string]any{
			"AdminAccounts": []any{
				map[string]any{"AccountId": "123456789012", "Status": "ENABLED"},
			},
			"NextToken": "",
		}
	case "GetEnabledStandards":
		return map[string]any{"StandardsSubscriptions": []any{}, "NextToken": ""}
	case "GetFindings", "GetFindingsV2":
		return map[string]any{"Findings": []any{}, "NextToken": ""}
	case "GetInsights":
		return map[string]any{"Insights": []any{}, "NextToken": ""}
	case "GetMembers":
		return map[string]any{"Members": []any{}, "UnprocessedAccounts": []any{}}
	case "ListTagsForResource":
		tags := s.ensureTagsLocked(resourceArn)
		out := map[string]any{}
		for k, v := range tags {
			out[k] = v
		}
		return map[string]any{"Tags": out}
	case "TagResource":
		tags := s.ensureTagsLocked(resourceArn)
		if raw, ok := payload["Tags"].(map[string]any); ok {
			for k, v := range raw {
				key := strings.TrimSpace(k)
				if key == "" {
					continue
				}
				tags[key] = strings.TrimSpace(anyString(v))
			}
		}
		if raw, ok := payload["Tags"].(map[string]string); ok {
			for k, v := range raw {
				key := strings.TrimSpace(k)
				if key == "" {
					continue
				}
				tags[key] = strings.TrimSpace(v)
			}
		}
		return map[string]any{}
	case "UntagResource":
		tags := s.ensureTagsLocked(resourceArn)
		for _, key := range securityHubStringSlicePayload(payload, "TagKeys") {
			delete(tags, key)
		}
		for _, key := range strings.Split(securityHubStringFromQuery(query, "tagKeys"), ",") {
			key = strings.TrimSpace(key)
			if key != "" {
				delete(tags, key)
			}
		}
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func (s *securityHubStore) ensureTagsLocked(resourceArn string) map[string]string {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		resourceArn = "arn:aws:securityhub:us-east-1:123456789012:hub/default"
	}
	if existing, ok := s.tags[resourceArn]; ok {
		return existing
	}
	created := map[string]string{}
	s.tags[resourceArn] = created
	return created
}

func securityHubPayloadString(payload map[string]any, key, fallback string) string {
	if payload == nil {
		return fallback
	}
	if v, ok := payload[key]; ok {
		if s := strings.TrimSpace(anyString(v)); s != "" {
			return s
		}
	}
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		if s := strings.TrimSpace(anyString(v)); s != "" {
			return s
		}
	}
	return fallback
}

func securityHubStringSlicePayload(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		switch tv := v.(type) {
		case []any:
			out := make([]string, 0, len(tv))
			for _, item := range tv {
				if s := strings.TrimSpace(anyString(item)); s != "" {
					out = append(out, s)
				}
			}
			return out
		case []string:
			out := make([]string, 0, len(tv))
			for _, item := range tv {
				if s := strings.TrimSpace(item); s != "" {
					out = append(out, s)
				}
			}
			return out
		}
	}
	return nil
}

func anyString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", t)
	}
}
