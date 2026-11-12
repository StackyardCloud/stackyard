package server

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type recycleBinStore struct {
	mu       sync.Mutex
	nextID   int64
	rules    map[string]map[string]any
	arnToID  map[string]string
	tagIndex map[string]map[string]string
}

func newRecycleBinStore() *recycleBinStore {
	s := &recycleBinStore{
		nextID:   2,
		rules:    map[string]map[string]any{},
		arnToID:  map[string]string{},
		tagIndex: map[string]map[string]string{},
	}

	seedID := "rbin-00000001"
	seedARN := recycleBinRuleARN(seedID)
	seedRule := map[string]any{
		"Identifier":   seedID,
		"Description":  "Stackyard seeded retention rule",
		"ResourceType": "EBS_SNAPSHOT",
		"RetentionPeriod": map[string]any{
			"RetentionPeriodValue": float64(30),
			"RetentionPeriodUnit":  "DAYS",
		},
		"ResourceTags": []any{
			map[string]any{
				"ResourceTagKey":   "seed",
				"ResourceTagValue": "true",
			},
		},
		"ExcludeResourceTags": []any{},
		"Status":              "available",
		"LockConfiguration": map[string]any{
			"UnlockDelay": map[string]any{
				"UnlockDelayValue": float64(7),
				"UnlockDelayUnit":  "DAYS",
			},
		},
		"LockState": "unlocked",
		"RuleArn":   seedARN,
	}
	s.rules[seedID] = recycleBinCloneMap(seedRule)
	s.arnToID[seedARN] = seedID
	s.tagIndex[seedARN] = map[string]string{"seed": "true"}
	return s
}

func (s *recycleBinStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	identifier := recycleBinFirstNonEmpty(
		recycleBinPathParam(pathParams, "identifier"),
		recycleBinPayloadString(payload, "Identifier", "identifier"),
	)
	resourceARN := recycleBinFirstNonEmpty(
		recycleBinPathParam(pathParams, "resourceArn"),
		recycleBinPayloadString(payload, "ResourceArn", "resourceArn"),
	)

	switch action {
	case "CreateRule":
		identifier = s.nextRuleIDLocked()
		rule := s.newRuleFromPayloadLocked(identifier, payload)
		arn := recycleBinPayloadString(rule, "RuleArn")
		s.rules[identifier] = recycleBinCloneMap(rule)
		s.arnToID[arn] = identifier
		if existingTags := recycleBinTagMapFromAny(payload["Tags"]); len(existingTags) > 0 {
			s.tagIndex[arn] = existingTags
		} else if _, exists := s.tagIndex[arn]; !exists {
			s.tagIndex[arn] = map[string]string{}
		}
		return recycleBinRuleResponse(rule)

	case "DeleteRule":
		rule := s.ensureRuleLocked(identifier)
		arn := recycleBinPayloadString(rule, "RuleArn")
		delete(s.rules, recycleBinPayloadString(rule, "Identifier"))
		delete(s.arnToID, arn)
		delete(s.tagIndex, arn)
		return map[string]any{}

	case "GetRule":
		return recycleBinRuleResponse(s.ensureRuleLocked(identifier))

	case "ListRules":
		items := make([]any, 0, len(s.rules))
		for _, id := range recycleBinSortedRuleIDs(s.rules) {
			rule := s.rules[id]
			items = append(items, map[string]any{
				"Identifier":  recycleBinPayloadString(rule, "Identifier"),
				"Description": recycleBinPayloadString(rule, "Description"),
				"ResourceType": recycleBinFirstNonEmpty(
					recycleBinPayloadString(rule, "ResourceType"),
					"EBS_SNAPSHOT",
				),
				"RetentionPeriod": recycleBinCloneAny(rule["RetentionPeriod"]),
				"Status":          recycleBinPayloadString(rule, "Status"),
				"LockState":       recycleBinPayloadString(rule, "LockState"),
				"LockEndTime":     recycleBinCloneAny(rule["LockEndTime"]),
				"RuleArn":         recycleBinPayloadString(rule, "RuleArn"),
			})
		}
		return map[string]any{"Rules": items, "NextToken": ""}

	case "LockRule":
		rule := s.ensureRuleLocked(identifier)
		if lockCfg, ok := payload["LockConfiguration"]; ok && lockCfg != nil {
			rule["LockConfiguration"] = recycleBinCloneAny(lockCfg)
		}
		rule["LockState"] = "locked"
		rule["LockEndTime"] = time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
		return recycleBinRuleResponse(rule)

	case "UnlockRule":
		rule := s.ensureRuleLocked(identifier)
		rule["LockState"] = "unlocked"
		rule["LockEndTime"] = time.Now().UTC().Add(10 * time.Minute).Format(time.RFC3339)
		return recycleBinRuleResponse(rule)

	case "UpdateRule":
		rule := s.ensureRuleLocked(identifier)
		for _, key := range []string{"RetentionPeriod", "Description", "ResourceType", "ResourceTags", "ExcludeResourceTags"} {
			if value, ok := payload[key]; ok {
				rule[key] = recycleBinCloneAny(value)
			}
		}
		return map[string]any{
			"Identifier":      recycleBinPayloadString(rule, "Identifier"),
			"RetentionPeriod": recycleBinCloneAny(rule["RetentionPeriod"]),
			"Description":     recycleBinPayloadString(rule, "Description"),
			"ResourceType":    recycleBinPayloadString(rule, "ResourceType"),
			"ResourceTags":    recycleBinCloneAny(rule["ResourceTags"]),
			"Status":          recycleBinPayloadString(rule, "Status"),
			"LockState":       recycleBinPayloadString(rule, "LockState"),
			"LockEndTime":     recycleBinCloneAny(rule["LockEndTime"]),
			"RuleArn":         recycleBinPayloadString(rule, "RuleArn"),
		}

	case "TagResource":
		resourceARN = s.resolveARNLocked(resourceARN, identifier)
		tagMap := s.ensureTagMapLocked(resourceARN)
		for k, v := range recycleBinTagMapFromAny(payload["Tags"]) {
			tagMap[k] = v
		}
		return map[string]any{}

	case "UntagResource":
		resourceARN = s.resolveARNLocked(resourceARN, identifier)
		tagMap := s.ensureTagMapLocked(resourceARN)
		for _, key := range recycleBinTagKeys(payload, query) {
			delete(tagMap, key)
		}
		return map[string]any{}

	case "ListTagsForResource":
		resourceARN = s.resolveARNLocked(resourceARN, identifier)
		tagMap := s.ensureTagMapLocked(resourceARN)
		return map[string]any{"Tags": recycleBinTagsToList(tagMap)}
	}

	return map[string]any{}
}

func (s *recycleBinStore) nextRuleIDLocked() string {
	id := s.nextID
	s.nextID++
	return fmt.Sprintf("rbin-%08d", id)
}

func (s *recycleBinStore) ensureRuleLocked(identifier string) map[string]any {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		identifier = "rbin-00000001"
	}
	if existing, ok := s.rules[identifier]; ok {
		return existing
	}
	rule := s.newRuleFromPayloadLocked(identifier, map[string]any{})
	arn := recycleBinPayloadString(rule, "RuleArn")
	s.rules[identifier] = rule
	s.arnToID[arn] = identifier
	if _, ok := s.tagIndex[arn]; !ok {
		s.tagIndex[arn] = map[string]string{}
	}
	return rule
}

func (s *recycleBinStore) newRuleFromPayloadLocked(identifier string, payload map[string]any) map[string]any {
	rule := map[string]any{
		"Identifier":   identifier,
		"Description":  recycleBinFirstNonEmpty(recycleBinPayloadString(payload, "Description", "description"), "Stackyard recycle bin rule"),
		"ResourceType": recycleBinFirstNonEmpty(recycleBinPayloadString(payload, "ResourceType", "resourceType"), "EBS_SNAPSHOT"),
		"RetentionPeriod": recycleBinFirstNonNil(
			recycleBinCloneAny(payload["RetentionPeriod"]),
			map[string]any{
				"RetentionPeriodValue": float64(30),
				"RetentionPeriodUnit":  "DAYS",
			},
		),
		"ResourceTags": recycleBinFirstNonNil(
			recycleBinCloneAny(payload["ResourceTags"]),
			[]any{
				map[string]any{
					"ResourceTagKey":   "stackyard",
					"ResourceTagValue": "true",
				},
			},
		),
		"ExcludeResourceTags": recycleBinFirstNonNil(
			recycleBinCloneAny(payload["ExcludeResourceTags"]),
			[]any{},
		),
		"Status": "available",
		"LockConfiguration": recycleBinFirstNonNil(
			recycleBinCloneAny(payload["LockConfiguration"]),
			map[string]any{
				"UnlockDelay": map[string]any{
					"UnlockDelayValue": float64(7),
					"UnlockDelayUnit":  "DAYS",
				},
			},
		),
		"LockState": "unlocked",
		"RuleArn":   recycleBinRuleARN(identifier),
	}
	return rule
}

func recycleBinRuleResponse(rule map[string]any) map[string]any {
	cloned := recycleBinCloneMap(rule)
	delete(cloned, "ExcludeResourceTags")
	return cloned
}

func (s *recycleBinStore) resolveARNLocked(resourceARN, identifier string) string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN != "" {
		if _, ok := s.arnToID[resourceARN]; !ok {
			id := recycleBinFirstNonEmpty(identifier, "rbin-00000001")
			s.arnToID[resourceARN] = id
			if _, ok := s.rules[id]; !ok {
				s.rules[id] = s.newRuleFromPayloadLocked(id, map[string]any{})
			}
		}
		return resourceARN
	}
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		identifier = "rbin-00000001"
	}
	rule := s.ensureRuleLocked(identifier)
	return recycleBinPayloadString(rule, "RuleArn")
}

func (s *recycleBinStore) ensureTagMapLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = recycleBinRuleARN("rbin-00000001")
	}
	tags, ok := s.tagIndex[resourceARN]
	if !ok {
		tags = map[string]string{}
		s.tagIndex[resourceARN] = tags
	}
	return tags
}

func recycleBinSortedRuleIDs(rules map[string]map[string]any) []string {
	out := make([]string, 0, len(rules))
	for id := range rules {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func recycleBinRuleARN(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		identifier = "rbin-00000001"
	}
	return "arn:aws:rbin:us-east-1:123456789012:rule/" + identifier
}

func recycleBinPathParam(pathParams map[string]string, key string) string {
	if pathParams == nil {
		return ""
	}
	return strings.TrimSpace(pathParams[key])
}

func recycleBinPayloadString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := payload[key]; ok {
			switch tv := v.(type) {
			case string:
				if strings.TrimSpace(tv) != "" {
					return strings.TrimSpace(tv)
				}
			}
		}
	}
	return ""
}

func recycleBinTagMapFromAny(value any) map[string]string {
	out := map[string]string{}
	items, ok := value.([]any)
	if !ok {
		return out
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := strings.TrimSpace(recycleBinPayloadString(m, "Key", "key"))
		val := strings.TrimSpace(recycleBinPayloadString(m, "Value", "value"))
		if key == "" {
			continue
		}
		out[key] = val
	}
	return out
}

func recycleBinTagsToList(tags map[string]string) []any {
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

func recycleBinTagKeys(payload map[string]any, query url.Values) []string {
	out := make([]string, 0)
	seen := map[string]struct{}{}

	if values, ok := payload["TagKeys"].([]any); ok {
		for _, value := range values {
			text, ok := value.(string)
			if !ok {
				continue
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			if _, exists := seen[text]; exists {
				continue
			}
			seen[text] = struct{}{}
			out = append(out, text)
		}
	}

	for _, value := range query["tagKeys"] {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}

	return out
}

func recycleBinFirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func recycleBinFirstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func recycleBinCloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = recycleBinCloneAny(v)
	}
	return out
}

func recycleBinCloneAny(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return recycleBinCloneMap(v)
	case map[string]string:
		out := make(map[string]string, len(v))
		for k, val := range v {
			out[k] = val
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i := range v {
			out[i] = recycleBinCloneAny(v[i])
		}
		return out
	case []string:
		out := make([]string, len(v))
		copy(out, v)
		return out
	case float64, bool, nil:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		return v
	default:
		if f, ok := v.(fmt.Stringer); ok {
			return f.String()
		}
	}
	return value
}

func recycleBinIntFromAny(value any, fallback int) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return parsed
		}
	}
	return fallback
}
