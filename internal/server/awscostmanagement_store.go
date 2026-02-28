package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	awsCostManagementDefaultRegion    = "us-east-1"
	awsCostManagementDefaultAccountID = "123456789012"
)

type awsCostManagementStore struct {
	mu sync.Mutex

	nextID int64

	resources map[string]map[string]any
	tags      map[string]map[string]string
}

func newAWSCostManagementStore() *awsCostManagementStore {
	now := time.Now().UTC().Format(time.RFC3339)
	s := &awsCostManagementStore{
		nextID:    2,
		resources: map[string]map[string]any{},
		tags:      map[string]map[string]string{},
	}
	s.ensureResourceLocked("AnomalyMonitor", "monitor-000001", now)
	s.ensureResourceLocked("CostCategoryDefinition", "costcat-000001", now)
	s.ensureResourceLocked("Budget", "budget-000001", now)
	return s
}

func (s *awsCostManagementStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	resourceARN := awscmFirstString(payload, []string{"resourceArn", "ResourceArn", "resourceARN", "ResourceARN"}, awscmDefaultResourceARN("resource", "res-000001"))
	if action == "TagResource" || strings.HasSuffix(action, "_TagResource") {
		tagMap := s.ensureTagMapLocked(resourceARN)
		for k, v := range awscmExtractTags(payload) {
			tagMap[k] = v
		}
		return map[string]any{}
	}
	if action == "UntagResource" || strings.HasSuffix(action, "_UntagResource") {
		tagMap := s.ensureTagMapLocked(resourceARN)
		for _, key := range awscmExtractTagKeys(payload) {
			delete(tagMap, key)
		}
		return map[string]any{}
	}
	if action == "ListTagsForResource" || strings.HasSuffix(action, "_ListTagsForResource") {
		tagMap := s.ensureTagMapLocked(resourceARN)
		out := map[string]string{}
		for k, v := range tagMap {
			out[k] = v
		}
		return map[string]any{"tags": out}
	}

	switch {
	case strings.Contains(action, "Create") || strings.Contains(action, "Put") || strings.Contains(action, "Start") || strings.Contains(action, "Associate") || strings.Contains(action, "BatchPut"):
		kind := awscmActionKind(action)
		id := awscmFirstString(payload, []string{
			"id", "Id", "resourceId", "ResourceId", "resourceID", "ResourceID",
			"monitorArn", "MonitorArn", "subscriptionArn", "SubscriptionArn",
			"CostCategoryArn", "costCategoryArn", "BillingViewArn", "billingViewArn",
			"InvoiceUnitArn", "invoiceUnitArn", "ExportArn", "exportArn",
		}, s.nextResourceIDLocked(kind))
		res := s.ensureResourceLocked(kind, id, now)
		res["status"] = "ACTIVE"
		res["updatedAt"] = now
		if name := awscmFirstString(payload, []string{"name", "Name", "displayName", "DisplayName"}, ""); name != "" {
			res["name"] = name
		}
		if tags := awscmExtractTags(payload); len(tags) > 0 {
			t := s.ensureTagMapLocked(awscmAnyToString(res["arn"]))
			for k, v := range tags {
				t[k] = v
			}
		}
		key := awscmLowerFirst(kind) + "Id"
		return map[string]any{
			key:           id,
			"resourceArn": res["arn"],
			"status":      res["status"],
			"operation":   action,
		}

	case strings.Contains(action, "Delete") || strings.Contains(action, "Disassociate") || strings.Contains(action, "BatchDelete") || strings.Contains(action, "Cancel"):
		id := awscmFirstString(payload, []string{"id", "Id", "resourceId", "ResourceId"}, "")
		if id != "" {
			for key, item := range s.resources {
				if awscmAnyToString(item["id"]) == id || strings.HasSuffix(key, "/"+id) {
					delete(s.tags, awscmAnyToString(item["arn"]))
					delete(s.resources, key)
				}
			}
		}
		return map[string]any{}

	case strings.Contains(action, "Update") || strings.Contains(action, "Modify") || strings.Contains(action, "Execute"):
		kind := awscmActionKind(action)
		id := awscmFirstString(payload, []string{"id", "Id", "resourceId", "ResourceId"}, s.nextResourceIDLocked(kind))
		res := s.ensureResourceLocked(kind, id, now)
		res["updatedAt"] = now
		return map[string]any{"operation": action, "status": "UPDATED", "resourceArn": res["arn"]}

	case strings.Contains(action, "List"):
		items := awscmSortedResources(s.resources)
		return map[string]any{
			"items":     items,
			"nextToken": "",
			"operation": action,
		}

	case strings.Contains(action, "Get") || strings.Contains(action, "Describe"):
		kind := awscmActionKind(action)
		id := awscmFirstString(payload, []string{"id", "Id", "resourceId", "ResourceId"}, "res-000001")
		res := s.ensureResourceLocked(kind, id, now)
		return map[string]any{
			"item":      awscmCloneMap(res),
			"operation": action,
		}
	}

	return map[string]any{"operation": action, "status": "OK"}
}

func (s *awsCostManagementStore) nextResourceIDLocked(kind string) string {
	id := fmt.Sprintf("%s-%06d", strings.ToLower(kind), s.nextID)
	s.nextID++
	return id
}

func (s *awsCostManagementStore) ensureResourceLocked(kind, id, now string) map[string]any {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		kind = "Resource"
	}
	id = strings.TrimSpace(id)
	if id == "" {
		id = s.nextResourceIDLocked(kind)
	}
	key := kind + "/" + id
	if existing := s.resources[key]; existing != nil {
		return existing
	}
	resource := map[string]any{
		"id":        id,
		"kind":      kind,
		"name":      id,
		"status":    "ACTIVE",
		"createdAt": now,
		"updatedAt": now,
		"arn":       awscmDefaultResourceARN(strings.ToLower(kind), id),
	}
	s.resources[key] = resource
	return resource
}

func (s *awsCostManagementStore) ensureTagMapLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = awscmDefaultResourceARN("resource", "res-000001")
	}
	if existing := s.tags[resourceARN]; existing != nil {
		return existing
	}
	m := map[string]string{}
	s.tags[resourceARN] = m
	return m
}

func awscmSortedResources(resources map[string]map[string]any) []map[string]any {
	keys := make([]string, 0, len(resources))
	for key := range resources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, awscmCloneMap(resources[key]))
	}
	return out
}

func awscmActionKind(action string) string {
	base := action
	if idx := strings.Index(base, "_"); idx >= 0 && idx+1 < len(base) {
		base = base[idx+1:]
	}
	prefixes := []string{"Create", "Get", "List", "Describe", "Update", "Delete", "Put", "Start", "Modify", "Associate", "Disassociate", "BatchPut", "BatchDelete", "BatchGet", "BatchCreate", "BatchUpdate", "Cancel", "Execute"}
	for _, p := range prefixes {
		if strings.HasPrefix(base, p) {
			kind := strings.TrimPrefix(base, p)
			if kind != "" {
				return kind
			}
		}
	}
	return "Resource"
}

func awscmDefaultResourceARN(kind, id string) string {
	kind = strings.Trim(strings.ToLower(kind), " /")
	if kind == "" {
		kind = "resource"
	}
	id = strings.TrimSpace(id)
	if id == "" {
		id = "res-000001"
	}
	return fmt.Sprintf("arn:aws:awscostmanagement:%s:%s:%s/%s", awsCostManagementDefaultRegion, awsCostManagementDefaultAccountID, kind, id)
}

func awscmExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	for _, key := range []string{"tags", "Tags"} {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		switch typed := raw.(type) {
		case map[string]any:
			for k, v := range typed {
				k = strings.TrimSpace(k)
				if k == "" {
					continue
				}
				out[k] = awscmAnyToString(v)
			}
		case map[string]string:
			for k, v := range typed {
				k = strings.TrimSpace(k)
				if k == "" {
					continue
				}
				out[k] = strings.TrimSpace(v)
			}
		case []any:
			for _, entry := range typed {
				m, ok := entry.(map[string]any)
				if !ok {
					continue
				}
				k := awscmFirstString(m, []string{"key", "Key"}, "")
				if k == "" {
					continue
				}
				out[k] = awscmFirstString(m, []string{"value", "Value"}, "")
			}
		}
	}
	return out
}

func awscmExtractTagKeys(payload map[string]any) []string {
	out := make([]string, 0)
	for _, key := range []string{"tagKeys", "TagKeys"} {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		switch typed := raw.(type) {
		case []any:
			for _, entry := range typed {
				value := strings.TrimSpace(awscmAnyToString(entry))
				if value != "" {
					out = append(out, value)
				}
			}
		case []string:
			for _, entry := range typed {
				value := strings.TrimSpace(entry)
				if value != "" {
					out = append(out, value)
				}
			}
		case string:
			value := strings.TrimSpace(typed)
			if value != "" {
				out = append(out, value)
			}
		}
	}
	return out
}

func awscmFirstString(m map[string]any, keys []string, def string) string {
	for _, key := range keys {
		raw, ok := m[key]
		if !ok || raw == nil {
			continue
		}
		value := strings.TrimSpace(awscmAnyToString(raw))
		if value != "" {
			return value
		}
	}
	return def
}

func awscmAnyToString(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case []byte:
		return string(typed)
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func awscmCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func awscmLowerFirst(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "resource"
	}
	if len(s) == 1 {
		return strings.ToLower(s)
	}
	return strings.ToLower(s[:1]) + s[1:]
}
