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
	workspacesThinClientDefaultRegion    = "us-east-1"
	workspacesThinClientDefaultAccountID = "123456789012"
)

type workspacesThinClientStore struct {
	mu sync.Mutex

	nextID int64

	environments map[string]map[string]any
	devices      map[string]map[string]any
	softwareSets map[string]map[string]any
	tags         map[string]map[string]string
}

func newWorkSpacesThinClientStore() *workspacesThinClientStore {
	now := time.Now().UTC().Format(time.RFC3339)
	seedID := "abcdefghijklmnopqrstuvwx"

	s := &workspacesThinClientStore{
		nextID:       1,
		environments: map[string]map[string]any{},
		devices:      map[string]map[string]any{},
		softwareSets: map[string]map[string]any{},
		tags:         map[string]map[string]string{},
	}

	software := s.ensureSoftwareSetLocked(seedID, now)
	env := s.ensureEnvironmentLocked(seedID, now)
	env["desiredSoftwareSetId"] = software["id"]
	s.ensureDeviceLocked(seedID, now)
	resourceARN := workspacesThinClientResourceARN("environment", seedID)
	s.ensureTagsLocked(resourceARN)["stackyard"] = "true"

	return s
}

func (s *workspacesThinClientStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	id := workspacesThinClientLookupString(payload, pathParams, query, "id", "Id")
	if id == "" {
		id = "abcdefghijklmnopqrstuvwx"
	}
	resourceArn := workspacesThinClientLookupString(payload, pathParams, query, "resourceArn", "resourceARN", "ResourceArn", "ResourceARN")
	if resourceArn == "" {
		resourceArn = workspacesThinClientResourceARN("environment", id)
	}

	environment := s.ensureEnvironmentLocked(id, now)
	device := s.ensureDeviceLocked(id, now)
	softwareSet := s.ensureSoftwareSetLocked(id, now)
	s.ensureTagsLocked(resourceArn)

	s.applyGenericUpdatesLocked(payload, now)

	switch action {
	case "CreateEnvironment":
		createdID := workspacesThinClientLookupString(payload, pathParams, query, "id", "Id")
		if createdID == "" {
			createdID = s.nextIdentifierLocked()
		}
		item := s.ensureEnvironmentLocked(createdID, now)
		if v := workspacesThinClientLookupString(payload, pathParams, query, "name", "Name"); v != "" {
			item["name"] = v
		}
		if v := workspacesThinClientLookupString(payload, pathParams, query, "desktopArn", "DesktopArn"); v != "" {
			item["desktopArn"] = v
		}
		if v := workspacesThinClientLookupString(payload, pathParams, query, "desktopEndpoint", "DesktopEndpoint"); v != "" {
			item["desktopEndpoint"] = v
		}
		if v := workspacesThinClientLookupString(payload, pathParams, query, "desiredSoftwareSetId", "DesiredSoftwareSetId"); v != "" {
			item["desiredSoftwareSetId"] = v
		}
		item["updatedAt"] = now
		s.mergeTagsLocked(workspacesThinClientExtractTags(payload), workspacesThinClientResourceARN("environment", createdID))
		return workspacesThinClientCloneMap(item)

	case "DeleteDevice":
		delete(s.devices, id)
		delete(s.tags, workspacesThinClientResourceARN("device", id))
		return map[string]any{}
	case "DeleteEnvironment":
		delete(s.environments, id)
		delete(s.tags, workspacesThinClientResourceARN("environment", id))
		return map[string]any{}
	case "DeregisterDevice":
		device["registered"] = false
		device["deviceStatus"] = "DEREGISTERED"
		device["updatedAt"] = now
		return map[string]any{}

	case "GetDevice":
		return workspacesThinClientCloneMap(device)
	case "GetEnvironment":
		return workspacesThinClientCloneMap(environment)
	case "GetSoftwareSet":
		return workspacesThinClientCloneMap(softwareSet)

	case "ListDevices":
		return map[string]any{"devices": s.listResourcesLocked(s.devices), "nextToken": ""}
	case "ListEnvironments":
		return map[string]any{"environments": s.listResourcesLocked(s.environments), "nextToken": ""}
	case "ListSoftwareSets":
		return map[string]any{"softwareSets": s.listResourcesLocked(s.softwareSets), "nextToken": ""}
	case "ListTagsForResource":
		return map[string]any{"tags": workspacesThinClientCloneStringMap(s.ensureTagsLocked(resourceArn))}

	case "TagResource":
		tags := s.ensureTagsLocked(resourceArn)
		for k, v := range workspacesThinClientExtractTags(payload) {
			tags[k] = v
		}
		return map[string]any{}
	case "UntagResource":
		tags := s.ensureTagsLocked(resourceArn)
		for _, key := range workspacesThinClientExtractTagKeys(payload, query) {
			delete(tags, key)
		}
		return map[string]any{}

	case "UpdateDevice":
		if v := workspacesThinClientLookupString(payload, pathParams, query, "name", "Name"); v != "" {
			device["name"] = v
		}
		if v := workspacesThinClientLookupString(payload, pathParams, query, "environmentId", "EnvironmentId"); v != "" {
			device["environmentId"] = v
		}
		device["updatedAt"] = now
		return workspacesThinClientCloneMap(device)
	case "UpdateEnvironment":
		if v := workspacesThinClientLookupString(payload, pathParams, query, "name", "Name"); v != "" {
			environment["name"] = v
		}
		if v := workspacesThinClientLookupString(payload, pathParams, query, "desiredSoftwareSetId", "DesiredSoftwareSetId"); v != "" {
			environment["desiredSoftwareSetId"] = v
		}
		environment["updatedAt"] = now
		return workspacesThinClientCloneMap(environment)
	case "UpdateSoftwareSet":
		if v := workspacesThinClientLookupString(payload, pathParams, query, "name", "Name"); v != "" {
			softwareSet["name"] = v
		}
		softwareSet["status"] = "AVAILABLE"
		softwareSet["updatedAt"] = now
		return workspacesThinClientCloneMap(softwareSet)
	}

	return map[string]any{}
}

func (s *workspacesThinClientStore) applyGenericUpdatesLocked(payload map[string]any, now string) {
	for _, value := range []struct {
		keys  []string
		apply func(map[string]any, string)
	}{
		{keys: []string{"name", "Name"}, apply: func(item map[string]any, v string) { item["name"] = v }},
		{keys: []string{"desktopArn", "DesktopArn"}, apply: func(item map[string]any, v string) { item["desktopArn"] = v }},
		{keys: []string{"desktopEndpoint", "DesktopEndpoint"}, apply: func(item map[string]any, v string) { item["desktopEndpoint"] = v }},
		{keys: []string{"desiredSoftwareSetId", "DesiredSoftwareSetId"}, apply: func(item map[string]any, v string) { item["desiredSoftwareSetId"] = v }},
	} {
		if v := workspacesThinClientLookupPayloadString(payload, value.keys...); v != "" {
			for _, item := range s.environments {
				value.apply(item, v)
				item["updatedAt"] = now
			}
			for _, item := range s.devices {
				value.apply(item, v)
				item["updatedAt"] = now
			}
			for _, item := range s.softwareSets {
				value.apply(item, v)
				item["updatedAt"] = now
			}
		}
	}
}

func (s *workspacesThinClientStore) ensureEnvironmentLocked(id, now string) map[string]any {
	id = workspacesThinClientNormalizeID(id)
	if existing := s.environments[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"id":                        id,
		"name":                      "stackyard-environment-" + id,
		"environmentArn":            workspacesThinClientResourceARN("environment", id),
		"desktopArn":                "arn:aws:workspaces:us-east-1:123456789012:workspace/ws-000001",
		"desktopEndpoint":           "https://stackyard-workspaces.example.com",
		"desiredSoftwareSetId":      id,
		"deviceCreationTags":        map[string]any{"managed-by": "stackyard"},
		"softwareSetUpdateMode":     "USE_DESIRED",
		"softwareSetUpdateSchedule": "USE_MAINTENANCE_WINDOW",
		"status":                    "ACTIVE",
		"createdAt":                 now,
		"updatedAt":                 now,
	}
	s.environments[id] = item
	return item
}

func (s *workspacesThinClientStore) ensureDeviceLocked(id, now string) map[string]any {
	id = workspacesThinClientNormalizeID(id)
	if existing := s.devices[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"id":            id,
		"name":          "stackyard-device-" + id,
		"deviceArn":     workspacesThinClientResourceARN("device", id),
		"environmentId": id,
		"model":         "WORKSPACES_THIN_CLIENT",
		"serialNumber":  strings.ToUpper(id[:12]),
		"deviceStatus":  "REGISTERED",
		"registered":    true,
		"createdAt":     now,
		"updatedAt":     now,
	}
	s.devices[id] = item
	return item
}

func (s *workspacesThinClientStore) ensureSoftwareSetLocked(id, now string) map[string]any {
	id = workspacesThinClientNormalizeID(id)
	if existing := s.softwareSets[id]; existing != nil {
		return existing
	}
	item := map[string]any{
		"id":             id,
		"name":           "stackyard-software-set-" + id,
		"softwareSetArn": workspacesThinClientResourceARN("softwareset", id),
		"version":        "1.0.0",
		"status":         "AVAILABLE",
		"releasedAt":     now,
		"createdAt":      now,
		"updatedAt":      now,
		"software": []any{
			map[string]any{"name": "workspaces-client", "version": "1.0.0"},
		},
	}
	s.softwareSets[id] = item
	return item
}

func (s *workspacesThinClientStore) ensureTagsLocked(resourceArn string) map[string]string {
	resourceArn = strings.TrimSpace(resourceArn)
	if resourceArn == "" {
		resourceArn = workspacesThinClientResourceARN("environment", "abcdefghijklmnopqrstuvwx")
	}
	if existing := s.tags[resourceArn]; existing != nil {
		return existing
	}
	created := map[string]string{}
	s.tags[resourceArn] = created
	return created
}

func (s *workspacesThinClientStore) mergeTagsLocked(tags map[string]string, resourceArn string) {
	if len(tags) == 0 {
		return
	}
	target := s.ensureTagsLocked(resourceArn)
	for k, v := range tags {
		target[k] = v
	}
}

func (s *workspacesThinClientStore) listResourcesLocked(resources map[string]map[string]any) []any {
	ids := make([]string, 0, len(resources))
	for id := range resources {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		out = append(out, workspacesThinClientCloneMap(resources[id]))
	}
	return out
}

func (s *workspacesThinClientStore) nextIdentifierLocked() string {
	id := s.nextID
	s.nextID++
	base := fmt.Sprintf("%024d", id)
	if len(base) > 24 {
		return base[len(base)-24:]
	}
	return base
}

func workspacesThinClientResourceARN(resourceType, id string) string {
	resourceType = strings.TrimSpace(resourceType)
	if resourceType == "" {
		resourceType = "environment"
	}
	id = workspacesThinClientNormalizeID(id)
	return fmt.Sprintf("arn:aws:thinclient:%s:%s:%s/%s", workspacesThinClientDefaultRegion, workspacesThinClientDefaultAccountID, resourceType, id)
}

func workspacesThinClientNormalizeID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "abcdefghijklmnopqrstuvwx"
	}
	if len(id) >= 24 {
		return id[:24]
	}
	builder := strings.Builder{}
	builder.WriteString(id)
	for builder.Len() < 24 {
		builder.WriteByte('0')
	}
	return builder.String()
}

func workspacesThinClientLookupString(payload map[string]any, pathParams map[string]string, query url.Values, keys ...string) string {
	for _, key := range keys {
		if value := workspacesThinClientLookupMapString(pathParams, key); value != "" {
			return value
		}
		if value := strings.TrimSpace(query.Get(key)); value != "" {
			return value
		}
		if value := workspacesThinClientLookupPayloadString(payload, key); value != "" {
			return value
		}
	}
	return ""
}

func workspacesThinClientLookupMapString(values map[string]string, key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if value := strings.TrimSpace(values[key]); value != "" {
		return value
	}
	for k, v := range values {
		if strings.EqualFold(strings.TrimSpace(k), key) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func workspacesThinClientLookupPayloadString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if value, ok := payload[key]; ok {
			if out := workspacesThinClientAnyToString(value); out != "" {
				return out
			}
		}
		for k, value := range payload {
			if strings.EqualFold(strings.TrimSpace(k), key) {
				if out := workspacesThinClientAnyToString(value); out != "" {
					return out
				}
			}
		}
	}
	return ""
}

func workspacesThinClientAnyToString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case []string:
		if len(v) > 0 {
			return strings.TrimSpace(v[0])
		}
	case []any:
		if len(v) > 0 {
			return workspacesThinClientAnyToString(v[0])
		}
	}
	return ""
}

func workspacesThinClientExtractTags(payload map[string]any) map[string]string {
	tags := map[string]string{}

	for _, key := range []string{"tags", "Tags"} {
		raw, ok := payload[key]
		if !ok {
			continue
		}
		switch v := raw.(type) {
		case map[string]any:
			for tagKey, tagValue := range v {
				tagKey = strings.TrimSpace(tagKey)
				if tagKey == "" {
					continue
				}
				tags[tagKey] = workspacesThinClientAnyToString(tagValue)
			}
		case map[string]string:
			for tagKey, tagValue := range v {
				tagKey = strings.TrimSpace(tagKey)
				if tagKey == "" {
					continue
				}
				tags[tagKey] = strings.TrimSpace(tagValue)
			}
		case []any:
			for _, entry := range v {
				entryMap, ok := entry.(map[string]any)
				if !ok {
					continue
				}
				tagKey := workspacesThinClientLookupPayloadString(entryMap, "key", "Key")
				if tagKey == "" {
					continue
				}
				tags[tagKey] = workspacesThinClientLookupPayloadString(entryMap, "value", "Value")
			}
		}
	}

	if len(tags) == 0 {
		tags["env"] = "coverage"
	}
	return tags
}

func workspacesThinClientExtractTagKeys(payload map[string]any, query url.Values) []string {
	seen := map[string]struct{}{}
	out := []string{}
	appendKey := func(value string) {
		value = strings.TrimSpace(strings.Trim(value, "[]\"'"))
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	for _, key := range []string{"tagKeys", "TagKeys"} {
		for _, raw := range query[key] {
			for _, part := range strings.Split(raw, ",") {
				appendKey(part)
			}
		}
	}
	for _, key := range []string{"tagKeys", "TagKeys"} {
		if raw, ok := payload[key]; ok {
			switch v := raw.(type) {
			case []any:
				for _, item := range v {
					appendKey(workspacesThinClientAnyToString(item))
				}
			case []string:
				for _, item := range v {
					appendKey(item)
				}
			default:
				appendKey(workspacesThinClientAnyToString(v))
			}
		}
	}
	if len(out) == 0 {
		out = append(out, "env")
	}
	return out
}

func workspacesThinClientCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch typed := v.(type) {
		case map[string]any:
			out[k] = workspacesThinClientCloneMap(typed)
		case map[string]string:
			out[k] = workspacesThinClientCloneStringMap(typed)
		case []any:
			copied := make([]any, len(typed))
			copy(copied, typed)
			out[k] = copied
		default:
			out[k] = v
		}
	}
	return out
}

func workspacesThinClientCloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
