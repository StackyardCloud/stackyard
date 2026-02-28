package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type cloudWatchInvestigationsStore struct {
	mu       sync.Mutex
	nextID   int64
	groups   map[string]map[string]any
	policies map[string]string
	tags     map[string]map[string]string
}

func newCloudWatchInvestigationsStore() *cloudWatchInvestigationsStore {
	now := time.Now().UTC()
	identifier := "stackyard-investigation-group"
	arn := cloudWatchInvestigationsGroupARN(identifier)
	return &cloudWatchInvestigationsStore{
		nextID: 2,
		groups: map[string]map[string]any{
			identifier: {
				"identifier":         identifier,
				"name":               identifier,
				"arn":                arn,
				"description":        "seed investigation group",
				"createdTime":        now,
				"lastModifiedTime":   now,
				"retentionInDays":    14,
				"state":              "ACTIVE",
				"crossAccountConfig": map[string]any{"enabled": false},
			},
		},
		policies: map[string]string{
			identifier: `{"Version":"2012-10-17","Statement":[]}`,
		},
		tags: map[string]map[string]string{
			arn: {
				"seed":    "true",
				"service": "cloudwatchinvestigations",
			},
		},
	}
}

func (s *cloudWatchInvestigationsStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()

	switch action {
	case "CreateInvestigationGroup":
		identifier := cloudWatchInvestigationsResolveIdentifier(payload, pathParams, "")
		if identifier == "" {
			identifier = fmt.Sprintf("stackyard-investigation-group-%06d", s.next())
		}
		name := cloudWatchInvestigationsDefaultString(payload, "name", identifier)
		group := s.ensureGroupLocked(identifier)
		group["identifier"] = identifier
		group["name"] = name
		group["arn"] = cloudWatchInvestigationsGroupARN(identifier)
		group["description"] = cloudWatchInvestigationsDefaultString(payload, "description", "stackyard investigation group")
		group["lastModifiedTime"] = now
		group["createdTime"] = now
		group["state"] = "ACTIVE"
		if _, ok := group["retentionInDays"]; !ok {
			group["retentionInDays"] = 14
		}
		return map[string]any{"investigationGroup": cloudWatchInvestigationsCloneMap(group)}

	case "GetInvestigationGroup":
		identifier := cloudWatchInvestigationsResolveIdentifier(payload, pathParams, "stackyard-investigation-group")
		group := s.ensureGroupLocked(identifier)
		group["lastModifiedTime"] = now
		return map[string]any{"investigationGroup": cloudWatchInvestigationsCloneMap(group)}

	case "UpdateInvestigationGroup":
		identifier := cloudWatchInvestigationsResolveIdentifier(payload, pathParams, "stackyard-investigation-group")
		group := s.ensureGroupLocked(identifier)
		if v := cloudWatchInvestigationsPayloadValue(payload, "name"); v != nil {
			group["name"] = strings.TrimSpace(cloudWatchInvestigationsToString(v))
		}
		if v := cloudWatchInvestigationsPayloadValue(payload, "description"); v != nil {
			group["description"] = strings.TrimSpace(cloudWatchInvestigationsToString(v))
		}
		if v := cloudWatchInvestigationsPayloadValue(payload, "retentionInDays"); v != nil {
			group["retentionInDays"] = v
		}
		group["lastModifiedTime"] = now
		return map[string]any{"investigationGroup": cloudWatchInvestigationsCloneMap(group)}

	case "DeleteInvestigationGroup":
		identifier := cloudWatchInvestigationsResolveIdentifier(payload, pathParams, "stackyard-investigation-group")
		arn := cloudWatchInvestigationsGroupARN(identifier)
		delete(s.groups, identifier)
		delete(s.policies, identifier)
		delete(s.tags, arn)
		return map[string]any{}

	case "ListInvestigationGroups":
		identifiers := make([]string, 0, len(s.groups))
		for identifier := range s.groups {
			identifiers = append(identifiers, identifier)
		}
		sort.Strings(identifiers)
		groups := make([]any, 0, len(identifiers))
		for _, identifier := range identifiers {
			groups = append(groups, cloudWatchInvestigationsCloneMap(s.groups[identifier]))
		}
		return map[string]any{"investigationGroups": groups, "nextToken": ""}

	case "PutInvestigationGroupPolicy":
		identifier := cloudWatchInvestigationsResolveIdentifier(payload, pathParams, "stackyard-investigation-group")
		policy := cloudWatchInvestigationsDefaultString(payload, "policy", `{"Version":"2012-10-17","Statement":[]}`)
		s.policies[identifier] = policy
		return map[string]any{"policy": policy}

	case "GetInvestigationGroupPolicy":
		identifier := cloudWatchInvestigationsResolveIdentifier(payload, pathParams, "stackyard-investigation-group")
		policy := strings.TrimSpace(s.policies[identifier])
		if policy == "" {
			policy = `{"Version":"2012-10-17","Statement":[]}`
			s.policies[identifier] = policy
		}
		return map[string]any{"policy": policy}

	case "DeleteInvestigationGroupPolicy":
		identifier := cloudWatchInvestigationsResolveIdentifier(payload, pathParams, "stackyard-investigation-group")
		delete(s.policies, identifier)
		return map[string]any{}

	case "TagResource":
		arn := cloudWatchInvestigationsResolveResourceARN(payload, pathParams, query)
		if arn == "" {
			arn = cloudWatchInvestigationsGroupARN("stackyard-investigation-group")
		}
		tags := s.ensureTagsLocked(arn)
		for key, value := range cloudWatchInvestigationsStringMap(cloudWatchInvestigationsPayloadValue(payload, "tags")) {
			if strings.TrimSpace(key) == "" {
				continue
			}
			tags[strings.TrimSpace(key)] = value
		}
		return map[string]any{}

	case "UntagResource":
		arn := cloudWatchInvestigationsResolveResourceARN(payload, pathParams, query)
		if arn == "" {
			arn = cloudWatchInvestigationsGroupARN("stackyard-investigation-group")
		}
		tags := s.ensureTagsLocked(arn)
		for _, key := range cloudWatchInvestigationsStringSlice(cloudWatchInvestigationsPayloadValue(payload, "tagKeys")) {
			delete(tags, key)
		}
		for _, key := range query["tagKeys"] {
			key = strings.TrimSpace(key)
			if key != "" {
				delete(tags, key)
			}
		}
		return map[string]any{}

	case "ListTagsForResource":
		arn := cloudWatchInvestigationsResolveResourceARN(payload, pathParams, query)
		if arn == "" {
			arn = cloudWatchInvestigationsGroupARN("stackyard-investigation-group")
		}
		return map[string]any{"tags": cloudWatchInvestigationsCloneStringMap(s.tags[arn])}
	}

	return map[string]any{}
}

func (s *cloudWatchInvestigationsStore) next() int64 {
	id := s.nextID
	s.nextID++
	return id
}

func (s *cloudWatchInvestigationsStore) ensureGroupLocked(identifier string) map[string]any {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		identifier = "stackyard-investigation-group"
	}
	group := s.groups[identifier]
	if group != nil {
		return group
	}
	now := time.Now().UTC()
	group = map[string]any{
		"identifier":       identifier,
		"name":             identifier,
		"arn":              cloudWatchInvestigationsGroupARN(identifier),
		"description":      "stackyard investigation group",
		"createdTime":      now,
		"lastModifiedTime": now,
		"retentionInDays":  14,
		"state":            "ACTIVE",
	}
	s.groups[identifier] = group
	return group
}

func (s *cloudWatchInvestigationsStore) ensureTagsLocked(arn string) map[string]string {
	tags := s.tags[arn]
	if tags == nil {
		tags = map[string]string{}
		s.tags[arn] = tags
	}
	return tags
}

func cloudWatchInvestigationsResolveIdentifier(payload map[string]any, pathParams map[string]string, fallback string) string {
	for _, key := range []string{"identifier", "Identifier", "investigationGroupIdentifier", "investigationGroupId", "id"} {
		if v := cloudWatchInvestigationsPathParam(pathParams, key, ""); v != "" {
			return v
		}
	}
	for _, key := range []string{"identifier", "investigationGroupIdentifier", "name", "id"} {
		if v := cloudWatchInvestigationsDefaultString(payload, key, ""); v != "" {
			return v
		}
	}
	return fallback
}

func cloudWatchInvestigationsResolveResourceARN(payload map[string]any, pathParams map[string]string, query url.Values) string {
	if v := cloudWatchInvestigationsDefaultString(payload, "resourceArn", ""); v != "" {
		return v
	}
	if v := cloudWatchInvestigationsPathParam(pathParams, "resourceArn", ""); v != "" {
		return v
	}
	if v := strings.TrimSpace(query.Get("resourceArn")); v != "" {
		return v
	}
	return ""
}

func cloudWatchInvestigationsGroupARN(identifier string) string {
	return fmt.Sprintf("arn:aws:aiops:us-east-1:123456789012:investigation-group/%s", strings.TrimSpace(identifier))
}

func cloudWatchInvestigationsPayloadValue(payload map[string]any, key string) any {
	if payload == nil {
		return nil
	}
	if v, ok := payload[key]; ok {
		return v
	}
	for k, v := range payload {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return nil
}

func cloudWatchInvestigationsDefaultString(payload map[string]any, key, fallback string) string {
	text := strings.TrimSpace(cloudWatchInvestigationsToString(cloudWatchInvestigationsPayloadValue(payload, key)))
	if text == "" {
		return fallback
	}
	return text
}

func cloudWatchInvestigationsPathParam(pathParams map[string]string, key, fallback string) string {
	if pathParams == nil {
		return fallback
	}
	if value, ok := pathParams[key]; ok {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	for k, value := range pathParams {
		if strings.EqualFold(k, key) {
			value = strings.TrimSpace(value)
			if value != "" {
				return value
			}
		}
	}
	return fallback
}

func cloudWatchInvestigationsToString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func cloudWatchInvestigationsStringMap(value any) map[string]string {
	out := map[string]string{}
	switch v := value.(type) {
	case map[string]any:
		for key, val := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(cloudWatchInvestigationsToString(val))
		}
	case map[string]string:
		for key, val := range v {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			out[key] = strings.TrimSpace(val)
		}
	}
	return out
}

func cloudWatchInvestigationsStringSlice(value any) []string {
	out := []string{}
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			text := strings.TrimSpace(cloudWatchInvestigationsToString(item))
			if text != "" {
				out = append(out, text)
			}
		}
	case []string:
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
	}
	return out
}

func cloudWatchInvestigationsCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch typed := v.(type) {
		case map[string]any:
			out[k] = cloudWatchInvestigationsCloneMap(typed)
		case map[string]string:
			out[k] = cloudWatchInvestigationsCloneStringMap(typed)
		default:
			out[k] = typed
		}
	}
	return out
}

func cloudWatchInvestigationsCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(map[string]string, len(in))
	for _, k := range keys {
		out[k] = in[k]
	}
	return out
}
