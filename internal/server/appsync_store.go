package server

import (
	"net/url"
	"strings"
	"sync"
)

type appSyncStore struct {
	mu   sync.Mutex
	tags map[string]map[string]string
}

func newAppSyncStore() *appSyncStore {
	const seedARN = "arn:aws:appsync:us-east-1:123456789012:apis/api-00000001"
	return &appSyncStore{
		tags: map[string]map[string]string{
			seedARN: {"seed": "true"},
		},
	}
}

func (s *appSyncStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	resourceARN := strings.TrimSpace(pathParams["resourceArn"])
	if resourceARN == "" {
		resourceARN = appSyncPayloadString(payload, "resourceArn", "ResourceArn")
	}

	switch action {
	case "TagResource":
		if resourceARN == "" {
			return map[string]any{}
		}
		tags := s.ensureTagsLocked(resourceARN)
		for key, value := range appSyncTagsFromAny(payload["tags"]) {
			tags[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		if resourceARN == "" {
			return map[string]any{}
		}
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range appSyncTagKeys(payload, query) {
			delete(tags, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		if resourceARN == "" {
			return map[string]any{"tags": map[string]string{}}
		}
		return map[string]any{"tags": appSyncCloneTags(s.ensureTagsLocked(resourceARN))}
	}

	return map[string]any{}
}

func (s *appSyncStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = "arn:aws:appsync:us-east-1:123456789012:apis/api-00000001"
	}
	tags, ok := s.tags[resourceARN]
	if !ok {
		tags = map[string]string{}
		s.tags[resourceARN] = tags
	}
	return tags
}

func appSyncPayloadString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := payload[key]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func appSyncTagKeys(payload map[string]any, query url.Values) []string {
	out := make([]string, 0)
	seen := map[string]struct{}{}

	if values, ok := payload["tagKeys"].([]any); ok {
		for _, v := range values {
			s, ok := v.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if _, exists := seen[s]; exists {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}

	for _, key := range query["tagKeys"] {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}

	return out
}

func appSyncTagsFromAny(value any) map[string]string {
	out := map[string]string{}
	switch v := value.(type) {
	case map[string]any:
		for key, raw := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			text, ok := raw.(string)
			if !ok {
				continue
			}
			out[key] = text
		}
	case map[string]string:
		for key, text := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = text
		}
	}
	return out
}

func appSyncCloneTags(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
