package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type finspaceManagementStore struct {
	mu sync.Mutex

	nextID int64
	tags   map[string]map[string]string
}

func newFinSpaceManagementStore() *finspaceManagementStore {
	return &finspaceManagementStore{
		nextID: 2,
		tags: map[string]map[string]string{
			"arn:aws:finspace:us-east-1:123456789012:environment/env-000001": {
				"seed":    "true",
				"service": "finspacemanagement",
			},
		},
	}
}

func (s *finspaceManagementStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	ctx := finspaceManagementMergeMaps(payload, pathParams, query)
	environmentID := finspaceManagementString(ctx, "environmentId", "env-000001")
	databaseName := finspaceManagementString(ctx, "databaseName", "db-000001")
	clusterName := finspaceManagementString(ctx, "clusterName", "cluster-000001")
	dataviewName := finspaceManagementString(ctx, "dataviewName", "dataview-000001")
	scalingGroupName := finspaceManagementString(ctx, "scalingGroupName", "scaling-group-000001")
	userName := finspaceManagementString(ctx, "userName", "user-000001")
	volumeName := finspaceManagementString(ctx, "volumeName", "volume-000001")
	resourceArn := finspaceManagementString(ctx, "resourceArn", fmt.Sprintf("arn:aws:finspace:us-east-1:123456789012:environment/%s", environmentID))

	switch action {
	case "TagResource":
		existing := s.ensureTagsLocked(resourceArn)
		for key, value := range finspaceManagementMapString(payload["tags"]) {
			existing[key] = value
		}
		for key, value := range finspaceManagementMapString(payload["Tags"]) {
			existing[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		existing := s.ensureTagsLocked(resourceArn)
		for _, key := range finspaceManagementTagKeys(ctx, query) {
			delete(existing, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		return map[string]any{"tags": finspaceManagementCloneStringMap(s.ensureTagsLocked(resourceArn))}

	case "GetKxConnectionString":
		return map[string]any{
			"signedConnectionString": fmt.Sprintf("wss://%s.stackyard.finspace.local/q", environmentID),
			"userName":               userName,
			"validUntil":             now.Add(15 * time.Minute).Format(time.RFC3339),
		}
	}

	entity := map[string]any{
		"action":           action,
		"id":               fmt.Sprintf("fsm-%06d", s.nextID),
		"environmentId":    environmentID,
		"databaseName":     databaseName,
		"clusterName":      clusterName,
		"dataviewName":     dataviewName,
		"scalingGroupName": scalingGroupName,
		"userName":         userName,
		"volumeName":       volumeName,
		"status":           "ACTIVE",
		"createdAt":        now.Format(time.RFC3339),
		"updatedAt":        now.Format(time.RFC3339),
	}

	if strings.HasPrefix(action, "List") {
		s.nextID++
		listKey := finspaceManagementListKey(action)
		return map[string]any{
			listKey:     []any{entity},
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

	return map[string]any{}
}

func (s *finspaceManagementStore) ensureTagsLocked(resourceArn string) map[string]string {
	arn := strings.TrimSpace(resourceArn)
	if arn == "" {
		arn = "arn:aws:finspace:us-east-1:123456789012:environment/env-000001"
	}
	if existing := s.tags[arn]; existing != nil {
		return existing
	}
	s.tags[arn] = map[string]string{"service": "finspacemanagement"}
	return s.tags[arn]
}

func finspaceManagementListKey(action string) string {
	switch action {
	case "ListEnvironments", "ListKxEnvironments":
		return "environments"
	case "ListKxClusters":
		return "clusters"
	case "ListKxClusterNodes":
		return "nodes"
	case "ListKxDatabases":
		return "databases"
	case "ListKxDataviews":
		return "dataviews"
	case "ListKxChangesets":
		return "changesets"
	case "ListKxScalingGroups":
		return "scalingGroups"
	case "ListKxUsers":
		return "users"
	case "ListKxVolumes":
		return "volumes"
	default:
		return "items"
	}
}

func finspaceManagementMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
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

func finspaceManagementString(values map[string]any, key, def string) string {
	if values == nil {
		return def
	}
	for candidate, raw := range values {
		if !strings.EqualFold(strings.TrimSpace(candidate), strings.TrimSpace(key)) {
			continue
		}
		if raw == nil {
			continue
		}
		text := strings.TrimSpace(fmt.Sprint(raw))
		if text != "" {
			return text
		}
	}
	return def
}

func finspaceManagementMapString(value any) map[string]string {
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

func finspaceManagementTagKeys(payload map[string]any, query url.Values) []string {
	if keys := finspaceManagementStringSlice(payload["tagKeys"]); len(keys) > 0 {
		return keys
	}
	if keys := finspaceManagementStringSlice(payload["TagKeys"]); len(keys) > 0 {
		return keys
	}
	for _, queryKey := range []string{"tagKeys", "TagKeys"} {
		if values := query[queryKey]; len(values) > 0 {
			return finspaceManagementSplitCSV(strings.Join(values, ","))
		}
	}
	return nil
}

func finspaceManagementStringSlice(value any) []string {
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
		return finspaceManagementSplitCSV(v)
	default:
		return nil
	}
}

func finspaceManagementSplitCSV(raw string) []string {
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

func finspaceManagementCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
