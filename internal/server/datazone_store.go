package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type datazoneStore struct {
	mu sync.Mutex

	nextID int64
	tags   map[string]map[string]string
}

func newDataZoneStore() *datazoneStore {
	return &datazoneStore{
		nextID: 2,
		tags: map[string]map[string]string{
			"arn:aws:datazone:us-east-1:123456789012:domain/dzd-000001": {
				"seed":    "true",
				"service": "datazone",
			},
		},
	}
}

func (s *datazoneStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	ctx := datazoneMergeMaps(payload, pathParams, query)
	domainID := datazoneString(ctx, "domainIdentifier", "dzd-000001")
	resourceArn := datazoneString(ctx, "resourceArn", fmt.Sprintf("arn:aws:datazone:us-east-1:123456789012:domain/%s", domainID))

	switch action {
	case "TagResource":
		existing := s.ensureTagsLocked(resourceArn)
		for key, value := range datazoneMapString(payload["tags"]) {
			existing[key] = value
		}
		for key, value := range datazoneMapString(payload["Tags"]) {
			existing[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		existing := s.ensureTagsLocked(resourceArn)
		for _, key := range datazoneTagKeys(ctx, query) {
			delete(existing, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"tags": datazoneCloneStringMap(s.ensureTagsLocked(resourceArn))}
	}

	entity := datazoneEntityForAction(action, now, domainID, s.nextID)

	if strings.HasPrefix(action, "List") || strings.HasPrefix(action, "Search") {
		s.nextID++
		return map[string]any{
			"items":     []any{entity},
			"nextToken": "",
		}
	}

	if strings.HasPrefix(action, "Get") {
		return entity
	}

	if strings.HasPrefix(action, "Create") || strings.HasPrefix(action, "Update") {
		s.nextID++
		return entity
	}

	if strings.HasPrefix(action, "Delete") {
		return map[string]any{}
	}

	if strings.HasPrefix(action, "Accept") ||
		strings.HasPrefix(action, "Reject") ||
		strings.HasPrefix(action, "Cancel") ||
		strings.HasPrefix(action, "Revoke") ||
		strings.HasPrefix(action, "Associate") ||
		strings.HasPrefix(action, "Disassociate") ||
		strings.HasPrefix(action, "Add") ||
		strings.HasPrefix(action, "Remove") ||
		strings.HasPrefix(action, "Batch") ||
		strings.HasPrefix(action, "Put") ||
		strings.HasPrefix(action, "Post") ||
		strings.HasPrefix(action, "Start") {
		return map[string]any{
			"status":    "SUCCEEDED",
			"action":    action,
			"requestId": fmt.Sprintf("req-%06d", s.nextID),
		}
	}

	return map[string]any{}
}

func (s *datazoneStore) ensureTagsLocked(resourceArn string) map[string]string {
	arn := strings.TrimSpace(resourceArn)
	if arn == "" {
		arn = "arn:aws:datazone:us-east-1:123456789012:domain/dzd-000001"
	}
	if existing := s.tags[arn]; existing != nil {
		return existing
	}
	s.tags[arn] = map[string]string{"service": "datazone"}
	return s.tags[arn]
}

func datazoneEntityForAction(action string, now time.Time, domainID string, id int64) map[string]any {
	resourceID := fmt.Sprintf("dz-%06d", id)
	name := strings.TrimPrefix(strings.TrimPrefix(action, "Create"), "Update")
	if strings.TrimSpace(name) == "" {
		name = strings.TrimPrefix(strings.TrimPrefix(action, "Get"), "List")
	}
	entity := map[string]any{
		"action":           action,
		"id":               resourceID,
		"identifier":       resourceID,
		"name":             "stackyard-" + strings.ToLower(name),
		"description":      "Stackyard DataZone stub for " + action,
		"domainIdentifier": domainID,
		"arn":              fmt.Sprintf("arn:aws:datazone:us-east-1:123456789012:%s/%s", strings.ToLower(name), resourceID),
		"createdAt":        now.Format(time.RFC3339),
		"updatedAt":        now.Format(time.RFC3339),
		"status":           "ACTIVE",
	}
	return entity
}

func datazoneMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	out := map[string]any{}
	for key, value := range payload {
		out[key] = value
	}
	for key, value := range pathParams {
		out[key] = value
	}
	for key, values := range query {
		if len(values) == 0 {
			continue
		}
		if len(values) == 1 {
			out[key] = values[0]
			continue
		}
		dup := make([]string, len(values))
		copy(dup, values)
		out[key] = dup
	}
	return out
}

func datazoneString(values map[string]any, key, def string) string {
	if values == nil {
		return def
	}
	if raw, ok := values[key]; ok && raw != nil {
		text := strings.TrimSpace(fmt.Sprint(raw))
		if text != "" {
			return text
		}
	}
	return def
}

func datazoneMapString(value any) map[string]string {
	out := map[string]string{}
	switch v := value.(type) {
	case map[string]string:
		for key, raw := range v {
			k := strings.TrimSpace(key)
			if k != "" {
				out[k] = strings.TrimSpace(raw)
			}
		}
	case map[string]any:
		for key, raw := range v {
			k := strings.TrimSpace(key)
			if k != "" {
				out[k] = strings.TrimSpace(fmt.Sprint(raw))
			}
		}
	}
	return out
}

func datazoneTagKeys(payload map[string]any, query url.Values) []string {
	keys := datazoneStringSlice(payload["tagKeys"])
	if len(keys) > 0 {
		return keys
	}
	keys = datazoneStringSlice(payload["TagKeys"])
	if len(keys) > 0 {
		return keys
	}
	for _, queryKey := range []string{"tagKeys", "TagKeys"} {
		if values := query[queryKey]; len(values) > 0 {
			return datazoneSplitCSV(strings.Join(values, ","))
		}
	}
	return nil
}

func datazoneStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			token := strings.TrimSpace(item)
			if token != "" {
				out = append(out, token)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if item == nil {
				continue
			}
			token := strings.TrimSpace(fmt.Sprint(item))
			if token != "" {
				out = append(out, token)
			}
		}
		return out
	case string:
		return datazoneSplitCSV(v)
	default:
		return nil
	}
}

func datazoneSplitCSV(raw string) []string {
	parts := strings.Split(strings.TrimSpace(raw), ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token != "" {
			out = append(out, token)
		}
	}
	sort.Strings(out)
	return out
}

func datazoneCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
