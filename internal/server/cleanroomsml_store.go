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
	cleanRoomsMLDefaultRegion    = "us-east-1"
	cleanRoomsMLDefaultAccountID = "123456789012"
)

type cleanRoomsMLStore struct {
	mu sync.Mutex

	nextID int64
	tags   map[string]map[string]string
}

func newCleanRoomsMLStore() *cleanRoomsMLStore {
	seedARN := fmt.Sprintf(
		"arn:aws:cleanrooms-ml:%s:%s:configured-audience-model/cam-000001",
		cleanRoomsMLDefaultRegion,
		cleanRoomsMLDefaultAccountID,
	)
	return &cleanRoomsMLStore{
		nextID: 2,
		tags: map[string]map[string]string{
			seedARN: {
				"seed":    "true",
				"service": "cleanrooms-ml",
			},
		},
	}
}

func (s *cleanRoomsMLStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	ctx := cleanRoomsMLMergeMaps(payload, pathParams, query)
	resourceArn := cleanRoomsMLString(ctx, "resourceArn", "")

	switch action {
	case "TagResource":
		existing := s.ensureTagsLocked(resourceArn)
		for key, value := range cleanRoomsMLMapString(payload["tags"]) {
			existing[key] = value
		}
		for key, value := range cleanRoomsMLMapString(payload["Tags"]) {
			existing[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		existing := s.ensureTagsLocked(resourceArn)
		for _, key := range cleanRoomsMLTagKeys(ctx, query) {
			delete(existing, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"tags": cleanRoomsMLCloneStringMap(s.ensureTagsLocked(resourceArn))}
	}

	entity := cleanRoomsMLEntityForAction(action, ctx, now, s.nextID)

	if strings.HasPrefix(action, "List") {
		s.nextID++
		return map[string]any{
			"items":     []any{entity},
			"nextToken": "",
		}
	}

	if strings.HasPrefix(action, "Get") {
		return entity
	}

	if strings.HasPrefix(action, "Create") ||
		strings.HasPrefix(action, "Update") ||
		strings.HasPrefix(action, "Put") ||
		strings.HasPrefix(action, "Start") ||
		strings.HasPrefix(action, "Cancel") ||
		strings.HasPrefix(action, "Delete") {
		s.nextID++
		return entity
	}

	return map[string]any{}
}

func (s *cleanRoomsMLStore) ensureTagsLocked(resourceArn string) map[string]string {
	arn := strings.TrimSpace(resourceArn)
	if arn == "" {
		arn = fmt.Sprintf(
			"arn:aws:cleanrooms-ml:%s:%s:configured-audience-model/cam-000001",
			cleanRoomsMLDefaultRegion,
			cleanRoomsMLDefaultAccountID,
		)
	}
	if existing := s.tags[arn]; existing != nil {
		return existing
	}
	s.tags[arn] = map[string]string{"service": "cleanrooms-ml"}
	return s.tags[arn]
}

func cleanRoomsMLEntityForAction(action string, values map[string]any, now time.Time, id int64) map[string]any {
	identifier := fmt.Sprintf("crm-%06d", id)
	name := cleanRoomsMLDefaultString(cleanRoomsMLString(values, "name", ""), "stackyard-cleanrooms-ml")
	membershipIdentifier := cleanRoomsMLDefaultString(cleanRoomsMLString(values, "membershipIdentifier", ""), "mem-000001")
	collaborationIdentifier := cleanRoomsMLDefaultString(cleanRoomsMLString(values, "collaborationIdentifier", ""), "col-000001")

	entity := map[string]any{
		"action":                  action,
		"id":                      identifier,
		"identifier":              identifier,
		"name":                    name,
		"description":             "Stackyard Clean Rooms ML stub for " + action,
		"membershipIdentifier":    membershipIdentifier,
		"collaborationIdentifier": collaborationIdentifier,
		"arn": fmt.Sprintf(
			"arn:aws:cleanrooms-ml:%s:%s:resource/%s",
			cleanRoomsMLDefaultRegion,
			cleanRoomsMLDefaultAccountID,
			identifier,
		),
		"status":    "ACTIVE",
		"createdAt": now.Format(time.RFC3339),
		"updatedAt": now.Format(time.RFC3339),
	}

	if v := strings.TrimSpace(cleanRoomsMLString(values, "audienceModelArn", "")); v != "" {
		entity["audienceModelArn"] = v
	}
	if v := strings.TrimSpace(cleanRoomsMLString(values, "configuredAudienceModelArn", "")); v != "" {
		entity["configuredAudienceModelArn"] = v
	}
	if v := strings.TrimSpace(cleanRoomsMLString(values, "configuredModelAlgorithmArn", "")); v != "" {
		entity["configuredModelAlgorithmArn"] = v
	}
	if v := strings.TrimSpace(cleanRoomsMLString(values, "configuredModelAlgorithmAssociationArn", "")); v != "" {
		entity["configuredModelAlgorithmAssociationArn"] = v
	}
	if v := strings.TrimSpace(cleanRoomsMLString(values, "trainedModelArn", "")); v != "" {
		entity["trainedModelArn"] = v
	}
	if v := strings.TrimSpace(cleanRoomsMLString(values, "trainedModelInferenceJobArn", "")); v != "" {
		entity["trainedModelInferenceJobArn"] = v
	}
	if v := strings.TrimSpace(cleanRoomsMLString(values, "trainingDatasetArn", "")); v != "" {
		entity["trainingDatasetArn"] = v
	}
	if v := strings.TrimSpace(cleanRoomsMLString(values, "mlInputChannelArn", "")); v != "" {
		entity["mlInputChannelArn"] = v
	}
	if v := strings.TrimSpace(cleanRoomsMLString(values, "audienceGenerationJobArn", "")); v != "" {
		entity["audienceGenerationJobArn"] = v
	}

	return entity
}

func cleanRoomsMLMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
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

func cleanRoomsMLString(values map[string]any, key, def string) string {
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

func cleanRoomsMLMapString(value any) map[string]string {
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

func cleanRoomsMLTagKeys(payload map[string]any, query url.Values) []string {
	keys := cleanRoomsMLStringSlice(payload["tagKeys"])
	if len(keys) > 0 {
		return keys
	}
	keys = cleanRoomsMLStringSlice(payload["TagKeys"])
	if len(keys) > 0 {
		return keys
	}
	for _, queryKey := range []string{"tagKeys", "TagKeys"} {
		if values := query[queryKey]; len(values) > 0 {
			return cleanRoomsMLSplitCSV(strings.Join(values, ","))
		}
	}
	return nil
}

func cleanRoomsMLStringSlice(value any) []string {
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
		return cleanRoomsMLSplitCSV(v)
	default:
		return nil
	}
}

func cleanRoomsMLSplitCSV(raw string) []string {
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

func cleanRoomsMLCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cleanRoomsMLDefaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
