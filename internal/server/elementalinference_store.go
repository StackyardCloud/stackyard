package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	elementalInferenceDefaultRegion    = "us-east-1"
	elementalInferenceDefaultAccountID = "123456789012"
)

type elementalInferenceStore struct {
	mu sync.Mutex

	nextID int64

	feeds            map[string]map[string]any
	tags             map[string]map[string]string
	createFeedTokens map[string]string
}

func newElementalInferenceStore() *elementalInferenceStore {
	now := time.Now().UTC().Format(time.RFC3339)
	s := &elementalInferenceStore{
		nextID: 2,
		feeds: map[string]map[string]any{
			"feed-000001": {
				"id":                "feed-000001",
				"arn":               elementalInferenceFeedARN("feed-000001"),
				"name":              "stackyard-seeded-feed",
				"status":            "ACTIVE",
				"associationStatus": "DISASSOCIATED",
				"createdAt":         now,
				"updatedAt":         now,
			},
		},
		tags: map[string]map[string]string{
			elementalInferenceFeedARN("feed-000001"): {"stackyard": "true"},
		},
		createFeedTokens: map[string]string{},
	}
	return s
}

func (s *elementalInferenceStore) Handle(
	action string,
	payload map[string]any,
	pathParams map[string]string,
	query url.Values,
) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	switch action {
	case "CreateFeed":
		clientToken := elementalInferenceLookupString(nil, payload, query, "clientToken", "ClientToken")
		if clientToken != "" {
			if existingID := strings.TrimSpace(s.createFeedTokens[clientToken]); existingID != "" {
				feed := s.ensureFeedLocked(existingID, now)
				return map[string]any{"feed": elementalInferenceCloneMap(feed)}
			}
		}

		feedID := elementalInferenceLookupString(nil, payload, query, "id", "feedId", "feedID")
		if feedID == "" {
			feedID = fmt.Sprintf("feed-%06d", s.nextID)
			s.nextID++
		}
		feed := s.ensureFeedLocked(feedID, now)
		if name := elementalInferenceLookupString(nil, payload, query, "name", "feedName", "Name"); name != "" {
			feed["name"] = name
		}
		if status := elementalInferenceLookupString(nil, payload, query, "status", "Status"); status != "" {
			feed["status"] = status
		}
		if assoc := elementalInferenceLookupString(nil, payload, query, "flowArn", "FlowArn", "resourceArn", "ResourceArn"); assoc != "" {
			feed["associatedResourceArn"] = assoc
			feed["associationStatus"] = "ASSOCIATED"
		}
		feed["updatedAt"] = now
		if clientToken != "" {
			s.createFeedTokens[clientToken] = feedID
		}
		return map[string]any{"feed": elementalInferenceCloneMap(feed)}

	case "GetFeed":
		feedID := elementalInferenceLookupString(pathParams, payload, query, "id", "feedId", "feedID")
		feed := s.ensureFeedLocked(feedID, now)
		return map[string]any{"feed": elementalInferenceCloneMap(feed)}

	case "ListFeeds":
		ids := make([]string, 0, len(s.feeds))
		for id := range s.feeds {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		items := make([]any, 0, len(ids))
		for _, id := range ids {
			feed := s.feeds[id]
			items = append(items, map[string]any{
				"id":                    feed["id"],
				"arn":                   feed["arn"],
				"name":                  feed["name"],
				"status":                feed["status"],
				"associationStatus":     feed["associationStatus"],
				"associatedResourceArn": feed["associatedResourceArn"],
			})
		}
		return map[string]any{"feeds": items, "nextToken": ""}

	case "UpdateFeed":
		feedID := elementalInferenceLookupString(pathParams, payload, query, "id", "feedId", "feedID")
		feed := s.ensureFeedLocked(feedID, now)
		if name := elementalInferenceLookupString(nil, payload, query, "name", "feedName", "Name"); name != "" {
			feed["name"] = name
		}
		if status := elementalInferenceLookupString(nil, payload, query, "status", "Status"); status != "" {
			feed["status"] = status
		}
		feed["updatedAt"] = now
		return map[string]any{"feed": elementalInferenceCloneMap(feed)}

	case "DeleteFeed":
		feedID := elementalInferenceLookupString(pathParams, payload, query, "id", "feedId", "feedID")
		feed := s.ensureFeedLocked(feedID, now)
		arn := elementalInferenceLookupString(nil, feed, nil, "arn")
		delete(s.feeds, feedID)
		if arn != "" {
			delete(s.tags, arn)
		}
		return map[string]any{}

	case "AssociateFeed":
		feedID := elementalInferenceLookupString(pathParams, payload, query, "id", "feedId", "feedID")
		feed := s.ensureFeedLocked(feedID, now)
		assoc := elementalInferenceLookupString(
			nil,
			payload,
			query,
			"flowArn",
			"FlowArn",
			"resourceArn",
			"ResourceArn",
			"associatedResourceArn",
			"AssociatedResourceArn",
		)
		if assoc == "" {
			assoc = "arn:aws:mediaconnect:us-east-1:123456789012:flow/flow-000001"
		}
		feed["associatedResourceArn"] = assoc
		feed["associationStatus"] = "ASSOCIATED"
		feed["updatedAt"] = now
		return map[string]any{"feed": elementalInferenceCloneMap(feed)}

	case "DisassociateFeed":
		feedID := elementalInferenceLookupString(pathParams, payload, query, "id", "feedId", "feedID")
		feed := s.ensureFeedLocked(feedID, now)
		delete(feed, "associatedResourceArn")
		feed["associationStatus"] = "DISASSOCIATED"
		feed["updatedAt"] = now
		return map[string]any{"feed": elementalInferenceCloneMap(feed)}

	case "TagResource":
		resourceARN := elementalInferenceLookupString(pathParams, payload, query, "resourceArn", "ResourceArn")
		if resourceARN == "" {
			resourceARN = elementalInferenceFeedARN("feed-000001")
		}
		tags := s.ensureTagsLocked(resourceARN)
		for key, value := range elementalInferenceExtractTags(payload) {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			tags[key] = value
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN := elementalInferenceLookupString(pathParams, payload, query, "resourceArn", "ResourceArn")
		if resourceARN == "" {
			resourceARN = elementalInferenceFeedARN("feed-000001")
		}
		return map[string]any{"tags": elementalInferenceCloneStringMap(s.ensureTagsLocked(resourceARN))}

	case "UntagResource":
		resourceARN := elementalInferenceLookupString(pathParams, payload, query, "resourceArn", "ResourceArn")
		if resourceARN == "" {
			resourceARN = elementalInferenceFeedARN("feed-000001")
		}
		tags := s.ensureTagsLocked(resourceARN)
		for _, key := range elementalInferenceExtractTagKeys(payload, query) {
			delete(tags, key)
		}
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *elementalInferenceStore) ensureFeedLocked(id, now string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "feed-000001"
	}
	if existing, ok := s.feeds[id]; ok {
		return existing
	}

	feed := map[string]any{
		"id":                id,
		"arn":               elementalInferenceFeedARN(id),
		"name":              "stackyard-" + id,
		"status":            "ACTIVE",
		"associationStatus": "DISASSOCIATED",
		"createdAt":         now,
		"updatedAt":         now,
	}
	s.feeds[id] = feed
	s.ensureTagsLocked(elementalInferenceFeedARN(id))["stackyard"] = "true"
	return feed
}

func (s *elementalInferenceStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = elementalInferenceFeedARN("feed-000001")
	}
	if existing, ok := s.tags[resourceARN]; ok {
		return existing
	}
	fresh := map[string]string{}
	s.tags[resourceARN] = fresh
	return fresh
}

func elementalInferenceFeedARN(feedID string) string {
	feedID = strings.TrimSpace(feedID)
	if feedID == "" {
		feedID = "feed-000001"
	}
	return fmt.Sprintf(
		"arn:aws:elemental-inference:%s:%s:feed/%s",
		elementalInferenceDefaultRegion,
		elementalInferenceDefaultAccountID,
		feedID,
	)
}

func elementalInferenceLookupString(pathParams map[string]string, payload map[string]any, query url.Values, keys ...string) string {
	for _, key := range keys {
		candidates := []string{key, lowerFirst(key), upperFirst(key)}
		for _, candidate := range candidates {
			if pathParams != nil {
				if value := strings.TrimSpace(pathParams[candidate]); value != "" {
					return value
				}
			}
			if payload != nil {
				if raw, ok := payload[candidate]; ok {
					if value, ok := raw.(string); ok && strings.TrimSpace(value) != "" {
						return strings.TrimSpace(value)
					}
				}
			}
			if query != nil {
				if value := strings.TrimSpace(query.Get(candidate)); value != "" {
					return value
				}
			}
		}
	}
	return ""
}

func elementalInferenceExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}

	if raw, ok := payload["tags"]; ok {
		switch typed := raw.(type) {
		case map[string]any:
			for key, value := range typed {
				if s, ok := value.(string); ok {
					out[strings.TrimSpace(key)] = strings.TrimSpace(s)
				}
			}
		case map[string]string:
			for key, value := range typed {
				out[strings.TrimSpace(key)] = strings.TrimSpace(value)
			}
		}
	}
	if raw, ok := payload["Tags"]; ok {
		if pairs, ok := raw.([]any); ok {
			for _, pair := range pairs {
				item, ok := pair.(map[string]any)
				if !ok {
					continue
				}
				key, _ := item["Key"].(string)
				value, _ := item["Value"].(string)
				key = strings.TrimSpace(key)
				if key == "" {
					continue
				}
				out[key] = strings.TrimSpace(value)
			}
		}
	}

	return out
}

func elementalInferenceExtractTagKeys(payload map[string]any, query url.Values) []string {
	unique := map[string]struct{}{}
	appendKey := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		unique[value] = struct{}{}
	}

	if query != nil {
		for _, key := range query["tagKeys"] {
			for _, split := range strings.Split(key, ",") {
				appendKey(split)
			}
		}
		for _, key := range query["tagKey"] {
			for _, split := range strings.Split(key, ",") {
				appendKey(split)
			}
		}
	}

	for _, field := range []string{"tagKeys", "TagKeys"} {
		raw, ok := payload[field]
		if !ok || raw == nil {
			continue
		}
		switch typed := raw.(type) {
		case []any:
			for _, item := range typed {
				if s, ok := item.(string); ok {
					appendKey(s)
				}
			}
		case []string:
			for _, item := range typed {
				appendKey(item)
			}
		case string:
			for _, split := range strings.Split(typed, ",") {
				appendKey(split)
			}
		}
	}

	if len(unique) == 0 {
		return nil
	}
	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func elementalInferenceCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = elementalInferenceCloneMap(typed)
		case []any:
			copied := make([]any, len(typed))
			copy(copied, typed)
			out[key] = copied
		default:
			out[key] = value
		}
	}
	return out
}

func elementalInferenceCloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
