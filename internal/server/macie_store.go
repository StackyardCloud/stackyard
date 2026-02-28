package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type macieStore struct {
	mu sync.Mutex

	nextID int64

	sessionStatus string
	members       map[string]map[string]any
	tags          map[string]map[string]string
}

func newMacieStore() *macieStore {
	s := &macieStore{
		nextID:        1,
		sessionStatus: "ENABLED",
		members:       map[string]map[string]any{},
		tags:          map[string]map[string]string{},
	}
	s.members["123456789012"] = map[string]any{
		"accountId": "123456789012",
		"status":    "ENABLED",
	}
	return s
}

func (s *macieStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	resourceArn := maciePathParam(pathParams, "resourceArn")
	if resourceArn == "" {
		resourceArn = maciePayloadString(payload, "resourceArn", "arn:aws:macie2:us-east-1:123456789012:classification-job/stackyard-job")
	}

	switch action {
	case "EnableMacie":
		s.sessionStatus = "ENABLED"
		return map[string]any{"status": s.sessionStatus}
	case "DisableMacie":
		s.sessionStatus = "PAUSED"
		return map[string]any{}
	case "GetMacieSession", "UpdateMacieSession":
		if action == "UpdateMacieSession" {
			if status := strings.TrimSpace(maciePayloadString(payload, "status", "")); status != "" {
				s.sessionStatus = status
			}
		}
		return map[string]any{
			"createdAt": now,
			"updatedAt": now,
			"status":    s.sessionStatus,
		}
	case "ListMembers":
		items := make([]string, 0, len(s.members))
		for accountID := range s.members {
			items = append(items, accountID)
		}
		sort.Strings(items)
		out := make([]any, 0, len(items))
		for _, accountID := range items {
			out = append(out, cloneAnyMap(s.members[accountID]))
		}
		return map[string]any{"members": out, "nextToken": ""}
	case "GetMember":
		id := maciePathParam(pathParams, "id")
		if id == "" {
			id = maciePayloadString(payload, "id", "123456789012")
		}
		if item, ok := s.members[id]; ok {
			return cloneAnyMap(item)
		}
		return map[string]any{"accountId": id, "status": "ENABLED"}
	case "CreateMember":
		id := maciePayloadString(payload, "account", "")
		if id == "" {
			id = maciePayloadString(payload, "accountId", "")
		}
		if id == "" {
			s.nextID++
			id = fmt.Sprintf("%012d", s.nextID)
		}
		s.members[id] = map[string]any{"accountId": id, "status": "INVITED"}
		return map[string]any{}
	case "DeleteMember":
		id := maciePathParam(pathParams, "id")
		if id != "" {
			delete(s.members, id)
		}
		return map[string]any{}
	case "ListFindings":
		return map[string]any{"findingIds": []any{}, "nextToken": ""}
	case "GetFindings":
		return map[string]any{"findings": []any{}}
	case "GetFindingStatistics":
		return map[string]any{"countsByGroup": []any{}}
	case "GetUsageTotals":
		return map[string]any{"timeRange": "MONTH_TO_DATE", "usageTotals": []any{}}
	case "GetUsageStatistics":
		return map[string]any{"records": []any{}, "timeRange": "LAST_30_DAYS"}
	case "TagResource":
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
		tags := s.ensureTagsLocked(resourceArn)
		out := map[string]any{}
		for k, v := range tags {
			out[k] = v
		}
		return map[string]any{"tags": out}
	case "UntagResource":
		tags := s.ensureTagsLocked(resourceArn)
		for _, k := range query["tagKeys"] {
			for _, part := range strings.Split(k, ",") {
				key := strings.TrimSpace(part)
				if key != "" {
					delete(tags, key)
				}
			}
		}
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func (s *macieStore) ensureTagsLocked(resourceArn string) map[string]string {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		resourceArn = "arn:aws:macie2:us-east-1:123456789012:classification-job/stackyard-job"
	}
	if existing, ok := s.tags[resourceArn]; ok {
		return existing
	}
	t := map[string]string{}
	s.tags[resourceArn] = t
	return t
}

func maciePayloadString(payload map[string]any, key, fallback string) string {
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

func maciePathParam(pathParams map[string]string, key string) string {
	if pathParams == nil {
		return ""
	}
	if v, ok := pathParams[key]; ok {
		return strings.TrimSpace(v)
	}
	for k, v := range pathParams {
		if strings.EqualFold(k, key) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
