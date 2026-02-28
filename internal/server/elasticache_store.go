package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type elastiCacheStore struct {
	mu sync.Mutex

	nextID int64
	tags   map[string]map[string]string
}

func newElastiCacheStore() *elastiCacheStore {
	return &elastiCacheStore{
		nextID: 2,
		tags: map[string]map[string]string{
			"arn:aws:elasticache:us-east-1:123456789012:cluster:stackyard-cache": {
				"seed":    "true",
				"service": "elasticache",
			},
		},
	}
}

func (s *elastiCacheStore) Handle(action string, form url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	resource := strings.TrimSpace(form.Get("ResourceName"))
	if resource == "" {
		resource = "arn:aws:elasticache:us-east-1:123456789012:cluster:stackyard-cache"
	}

	switch action {
	case "AddTagsToResource":
		tags := s.ensureTagsLocked(resource)
		for key, value := range elastiCacheParseTags(form) {
			tags[key] = value
		}
		return map[string]any{
			"TagList": elastiCacheTagsList(tags),
		}

	case "RemoveTagsFromResource":
		tags := s.ensureTagsLocked(resource)
		for _, key := range elastiCacheParseTagKeys(form) {
			delete(tags, key)
		}
		return map[string]any{
			"TagList": elastiCacheTagsList(tags),
		}

	case "ListTagsForResource":
		return map[string]any{
			"TagList": elastiCacheTagsList(s.ensureTagsLocked(resource)),
		}
	}

	resourceID := fmt.Sprintf("elasticache-%06d", s.nextID)
	s.nextID++
	arn := fmt.Sprintf("arn:aws:elasticache:us-east-1:123456789012:cluster:%s", resourceID)

	switch {
	case strings.HasPrefix(action, "Create"):
		return map[string]any{
			"ARN":        arn,
			"Status":     "creating",
			"CreateTime": now,
			"Id":         resourceID,
		}

	case strings.HasPrefix(action, "Delete"):
		return map[string]any{
			"ARN":    arn,
			"Status": "deleting",
			"Id":     resourceID,
		}

	case strings.HasPrefix(action, "Describe"), strings.HasPrefix(action, "List"):
		return map[string]any{
			"Items": []any{
				map[string]any{
					"ARN":        arn,
					"Status":     "available",
					"CreateTime": now,
					"Id":         resourceID,
				},
			},
			"Marker": "",
		}

	case strings.HasPrefix(action, "Modify"),
		strings.HasPrefix(action, "Increase"),
		strings.HasPrefix(action, "Decrease"),
		strings.HasPrefix(action, "Rebalance"),
		strings.HasPrefix(action, "Reboot"),
		strings.HasPrefix(action, "Authorize"),
		strings.HasPrefix(action, "Revoke"),
		strings.HasPrefix(action, "Start"),
		strings.HasPrefix(action, "Batch"),
		strings.HasPrefix(action, "Failover"),
		strings.HasPrefix(action, "Complete"),
		strings.HasPrefix(action, "Test"),
		strings.HasPrefix(action, "Copy"),
		strings.HasPrefix(action, "Export"),
		strings.HasPrefix(action, "Purchase"):
		return map[string]any{
			"Action":    action,
			"Status":    "available",
			"RequestId": resourceID,
		}
	}

	return map[string]any{
		"Action": action,
		"Status": "available",
	}
}

func (s *elastiCacheStore) ensureTagsLocked(resource string) map[string]string {
	resource = strings.TrimSpace(resource)
	if resource == "" {
		resource = "arn:aws:elasticache:us-east-1:123456789012:cluster:stackyard-cache"
	}
	if tags, ok := s.tags[resource]; ok {
		return tags
	}
	s.tags[resource] = map[string]string{"service": "elasticache"}
	return s.tags[resource]
}

func elastiCacheParseTags(form url.Values) map[string]string {
	out := map[string]string{}
	for key, values := range form {
		if !strings.HasPrefix(key, "Tags.member.") || !strings.HasSuffix(key, ".Key") || len(values) == 0 {
			continue
		}
		idx := strings.TrimSuffix(strings.TrimPrefix(key, "Tags.member."), ".Key")
		tagKey := strings.TrimSpace(values[0])
		if tagKey == "" {
			continue
		}
		tagValue := strings.TrimSpace(form.Get("Tags.member." + idx + ".Value"))
		out[tagKey] = tagValue
	}
	return out
}

func elastiCacheParseTagKeys(form url.Values) []string {
	keys := []string{}
	for key, values := range form {
		if !strings.HasPrefix(key, "TagKeys.member.") || len(values) == 0 {
			continue
		}
		tagKey := strings.TrimSpace(values[0])
		if tagKey != "" {
			keys = append(keys, tagKey)
		}
	}
	sort.Strings(keys)
	return keys
}

func elastiCacheTagsList(tags map[string]string) []any {
	if tags == nil {
		return []any{}
	}
	keys := make([]string, 0, len(tags))
	for key := range tags {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"Key": key, "Value": tags[key]})
	}
	return out
}
