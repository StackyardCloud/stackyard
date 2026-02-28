package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type dmsStore struct {
	mu     sync.Mutex
	nextID int64
	tags   map[string]map[string]string
}

func newDMSStore() *dmsStore {
	defaultARN := dmsDefaultReplicationInstanceARN()
	return &dmsStore{
		nextID: 1,
		tags: map[string]map[string]string{
			defaultARN: {"seed": "true"},
		},
	}
}

func (s *dmsStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch action {
	case "CreateReplicationInstance":
		identifier := dmsPayloadString(payload, "ReplicationInstanceIdentifier", fmt.Sprintf("stackyard-repl-%06d", s.nextID))
		arn := dmsReplicationInstanceARN(identifier)
		s.nextID++
		return map[string]any{
			"ReplicationInstance": map[string]any{
				"ReplicationInstanceIdentifier": identifier,
				"ReplicationInstanceArn":        arn,
				"ReplicationInstanceStatus":     "available",
			},
		}
	case "CreateEndpoint":
		identifier := dmsPayloadString(payload, "EndpointIdentifier", fmt.Sprintf("stackyard-endpoint-%06d", s.nextID))
		arn := fmt.Sprintf("arn:aws:dms:us-east-1:123456789012:endpoint:%s", identifier)
		s.nextID++
		return map[string]any{
			"Endpoint": map[string]any{
				"EndpointIdentifier": identifier,
				"EndpointArn":        arn,
				"Status":             "active",
			},
		}
	case "CreateReplicationTask":
		identifier := dmsPayloadString(payload, "ReplicationTaskIdentifier", fmt.Sprintf("stackyard-task-%06d", s.nextID))
		arn := fmt.Sprintf("arn:aws:dms:us-east-1:123456789012:task:%s", identifier)
		s.nextID++
		return map[string]any{
			"ReplicationTask": map[string]any{
				"ReplicationTaskIdentifier": identifier,
				"ReplicationTaskArn":        arn,
				"Status":                    "ready",
			},
		}
	case "DescribeReplicationInstances":
		return map[string]any{
			"ReplicationInstances": []any{
				map[string]any{
					"ReplicationInstanceIdentifier": "stackyard-replication-instance",
					"ReplicationInstanceArn":        dmsDefaultReplicationInstanceARN(),
					"ReplicationInstanceStatus":     "available",
				},
			},
		}
	case "DescribeReplicationTasks":
		return map[string]any{"ReplicationTasks": []any{}}
	case "DescribeEndpoints":
		return map[string]any{"Endpoints": []any{}}
	case "DescribeReplicationInstanceTaskLogs":
		return map[string]any{"ReplicationInstanceTaskLogs": []any{}}
	case "ListTagsForResource":
		arn := dmsPayloadString(payload, "ResourceArn", dmsDefaultReplicationInstanceARN())
		return map[string]any{"TagList": dmsTagsToList(s.tags[arn])}
	case "AddTagsToResource":
		arn := dmsPayloadString(payload, "ResourceArn", dmsDefaultReplicationInstanceARN())
		if _, ok := s.tags[arn]; !ok {
			s.tags[arn] = map[string]string{}
		}
		for k, v := range dmsTagsFromAny(payload["Tags"]) {
			s.tags[arn][k] = v
		}
		return map[string]any{}
	case "RemoveTagsFromResource":
		arn := dmsPayloadString(payload, "ResourceArn", dmsDefaultReplicationInstanceARN())
		keys := dmsPayloadStringSlice(payload, "TagKeys")
		for _, key := range keys {
			delete(s.tags[arn], key)
		}
		return map[string]any{}
	default:
		return map[string]any{}
	}
}

func dmsReplicationInstanceARN(identifier string) string {
	id := strings.TrimSpace(identifier)
	if id == "" {
		id = "stackyard-replication-instance"
	}
	return fmt.Sprintf("arn:aws:dms:us-east-1:123456789012:rep:%s", id)
}

func dmsDefaultReplicationInstanceARN() string {
	return dmsReplicationInstanceARN("stackyard-replication-instance")
}

func dmsPayloadString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	}
	return def
}

func dmsPayloadStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	var raw any
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			raw = v
			break
		}
	}
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok || strings.TrimSpace(s) == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func dmsTagsFromAny(raw any) map[string]string {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := map[string]string{}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := strings.TrimSpace(dmsPayloadString(m, "Key", ""))
		if key == "" {
			continue
		}
		out[key] = dmsPayloadString(m, "Value", "")
	}
	return out
}

func dmsTagsToList(tags map[string]string) []any {
	if len(tags) == 0 {
		return []any{}
	}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{"Key": k, "Value": tags[k]})
	}
	return out
}
