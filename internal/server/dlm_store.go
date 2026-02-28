package server

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

type dlmStore struct {
	mu sync.Mutex

	nextPolicyID int64
	policies     map[string]map[string]any
	tags         map[string]map[string]string
}

func newDLMStore() *dlmStore {
	s := &dlmStore{
		nextPolicyID: 2,
		policies:     map[string]map[string]any{},
		tags:         map[string]map[string]string{},
	}
	now := time.Now().UTC()
	seed := s.ensurePolicyLocked("policy-00000001", now)
	s.tags[dlmString(seed, "PolicyArn", dlmPolicyARN("policy-00000001"))] = map[string]string{"seed": "true"}
	return s
}

func (s *dlmStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	ctx := dlmMergeMaps(payload, pathParams, query)
	policyID := dlmStringAny(ctx, []string{"policyId", "PolicyId"}, s.defaultPolicyIDLocked())
	policy := s.ensurePolicyLocked(policyID, now)
	resourceARN := dlmStringAny(
		ctx,
		[]string{"resourceArn", "ResourceArn"},
		dlmString(policy, "PolicyArn", dlmPolicyARN(policyID)),
	)

	switch action {
	case "CreateLifecyclePolicy":
		policyID = fmt.Sprintf("policy-%08d", s.nextPolicyID)
		s.nextPolicyID++
		policy = s.ensurePolicyLocked(policyID, now)
		policy["Description"] = dlmStringAny(payload, []string{"Description"}, fmt.Sprintf("stackyard-%s", policyID))
		policy["State"] = dlmStringAny(payload, []string{"State"}, "ENABLED")
		policy["ExecutionRoleArn"] = dlmStringAny(
			payload,
			[]string{"ExecutionRoleArn"},
			"arn:aws:iam::123456789012:role/service-role/AWSDataLifecycleManagerDefaultRole",
		)
		policy["DateCreated"] = now.Format(time.RFC3339)
		policy["DateModified"] = now.Format(time.RFC3339)
		policy["PolicyDetails"] = dlmPolicyDetails(payload["PolicyDetails"])
		if tags := dlmTagsFromValue(payload["Tags"]); len(tags) > 0 {
			s.ensureTagsLocked(dlmString(policy, "PolicyArn", dlmPolicyARN(policyID)))
			for key, value := range tags {
				s.tags[dlmString(policy, "PolicyArn", dlmPolicyARN(policyID))][key] = value
			}
		}
		return map[string]any{"PolicyId": policyID}

	case "DeleteLifecyclePolicy":
		delete(s.policies, policyID)
		delete(s.tags, dlmPolicyARN(policyID))
		return map[string]any{}

	case "GetLifecyclePolicies":
		stateFilter := strings.ToUpper(strings.TrimSpace(dlmStringAny(ctx, []string{"state", "State"}, "")))
		policyFilter := dlmSplitCSV(dlmStringAny(ctx, []string{"policyIds", "PolicyIds"}, ""))
		policySet := map[string]struct{}{}
		for _, id := range policyFilter {
			policySet[id] = struct{}{}
		}

		ids := make([]string, 0, len(s.policies))
		for id := range s.policies {
			ids = append(ids, id)
		}
		sort.Strings(ids)

		items := make([]any, 0, len(ids))
		for _, id := range ids {
			entry := s.policies[id]
			if len(policySet) > 0 {
				if _, ok := policySet[id]; !ok {
					continue
				}
			}
			if stateFilter != "" && strings.ToUpper(dlmString(entry, "State", "ENABLED")) != stateFilter {
				continue
			}
			arn := dlmString(entry, "PolicyArn", dlmPolicyARN(id))
			items = append(items, map[string]any{
				"PolicyId":  id,
				"PolicyArn": arn,
				"Description": dlmString(
					entry,
					"Description",
					"",
				),
				"State":      dlmString(entry, "State", "ENABLED"),
				"PolicyType": dlmString(entry, "PolicyType", "EBS_SNAPSHOT_MANAGEMENT"),
				"Tags":       dlmCloneStringMap(s.ensureTagsLocked(arn)),
			})
		}
		return map[string]any{"Policies": items}

	case "GetLifecyclePolicy":
		arn := dlmString(policy, "PolicyArn", dlmPolicyARN(policyID))
		out := dlmCloneMap(policy)
		out["Tags"] = dlmCloneStringMap(s.ensureTagsLocked(arn))
		return map[string]any{"Policy": out}

	case "ListTagsForResource":
		return map[string]any{"Tags": dlmCloneStringMap(s.ensureTagsLocked(resourceARN))}

	case "TagResource":
		existing := s.ensureTagsLocked(resourceARN)
		for key, value := range dlmTagsFromValue(payload["Tags"]) {
			existing[key] = value
		}
		for key, value := range dlmTagsFromValue(payload["tags"]) {
			existing[key] = value
		}
		return map[string]any{}

	case "UntagResource":
		existing := s.ensureTagsLocked(resourceARN)
		for _, key := range dlmTagKeys(ctx, query) {
			delete(existing, key)
		}
		return map[string]any{}

	case "UpdateLifecyclePolicy":
		if description := dlmStringAny(payload, []string{"Description"}, ""); description != "" {
			policy["Description"] = description
		}
		if state := dlmStringAny(payload, []string{"State"}, ""); state != "" {
			policy["State"] = state
		}
		if role := dlmStringAny(payload, []string{"ExecutionRoleArn"}, ""); role != "" {
			policy["ExecutionRoleArn"] = role
		}
		if details, ok := payload["PolicyDetails"]; ok {
			policy["PolicyDetails"] = dlmPolicyDetails(details)
		}
		policy["DateModified"] = now.Format(time.RFC3339)
		return map[string]any{}
	}

	return map[string]any{}
}

func (s *dlmStore) ensurePolicyLocked(policyID string, now time.Time) map[string]any {
	id := strings.TrimSpace(policyID)
	if id == "" {
		id = s.defaultPolicyIDLocked()
	}
	if id == "" {
		id = "policy-00000001"
	}
	if existing := s.policies[id]; existing != nil {
		return existing
	}

	policy := map[string]any{
		"PolicyId":          id,
		"PolicyArn":         dlmPolicyARN(id),
		"PolicyType":        "EBS_SNAPSHOT_MANAGEMENT",
		"Description":       "Stackyard lifecycle policy " + id,
		"State":             "ENABLED",
		"ExecutionRoleArn":  "arn:aws:iam::123456789012:role/service-role/AWSDataLifecycleManagerDefaultRole",
		"DateCreated":       now.Format(time.RFC3339),
		"DateModified":      now.Format(time.RFC3339),
		"PolicyDetails":     dlmPolicyDetails(nil),
		"DefaultPolicy":     false,
		"DefaultPolicyType": "",
	}
	s.policies[id] = policy
	return policy
}

func (s *dlmStore) defaultPolicyIDLocked() string {
	if len(s.policies) == 0 {
		return ""
	}
	keys := make([]string, 0, len(s.policies))
	for key := range s.policies {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func (s *dlmStore) ensureTagsLocked(resourceARN string) map[string]string {
	arn := strings.TrimSpace(resourceARN)
	if arn == "" {
		arn = dlmPolicyARN(s.defaultPolicyIDLocked())
	}
	if s.tags[arn] == nil {
		s.tags[arn] = map[string]string{}
	}
	return s.tags[arn]
}

func dlmMergeMaps(payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
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
		copied := make([]string, len(values))
		copy(copied, values)
		out[key] = copied
	}
	return out
}

func dlmPolicyDetails(value any) map[string]any {
	if src, ok := value.(map[string]any); ok && len(src) > 0 {
		return dlmCloneMap(src)
	}
	return map[string]any{
		"PolicyType":    "EBS_SNAPSHOT_MANAGEMENT",
		"ResourceTypes": []any{"VOLUME"},
		"TargetTags": []any{
			map[string]any{"Key": "Name", "Value": "stackyard"},
		},
		"Schedules": []any{
			map[string]any{
				"Name": "DailySnapshots",
				"CreateRule": map[string]any{
					"Interval":     24,
					"IntervalUnit": "HOURS",
				},
				"RetainRule": map[string]any{
					"Count": 7,
				},
			},
		},
	}
}

func dlmTagsFromValue(value any) map[string]string {
	out := map[string]string{}
	switch v := value.(type) {
	case map[string]string:
		for key, raw := range v {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			out[k] = strings.TrimSpace(raw)
		}
	case map[string]any:
		for key, raw := range v {
			k := strings.TrimSpace(key)
			if k == "" {
				continue
			}
			if raw == nil {
				out[k] = ""
				continue
			}
			out[k] = strings.TrimSpace(fmt.Sprint(raw))
		}
	case []any:
		for _, item := range v {
			tag, ok := item.(map[string]any)
			if !ok {
				continue
			}
			key := dlmStringAny(tag, []string{"Key", "key"}, "")
			if key == "" {
				continue
			}
			out[key] = dlmStringAny(tag, []string{"Value", "value"}, "")
		}
	}
	return out
}

func dlmTagKeys(payload map[string]any, query url.Values) []string {
	keys := dlmStringSlice(payload["tagKeys"])
	if len(keys) > 0 {
		return keys
	}
	keys = dlmStringSlice(payload["TagKeys"])
	if len(keys) > 0 {
		return keys
	}
	keys = dlmSplitCSV(dlmStringAny(payload, []string{"tagKeys", "TagKeys"}, ""))
	if len(keys) > 0 {
		return keys
	}
	for _, queryKey := range []string{"tagKeys", "TagKeys"} {
		if values, ok := query[queryKey]; ok && len(values) > 0 {
			collected := make([]string, 0, len(values))
			for _, value := range values {
				collected = append(collected, dlmSplitCSV(value)...)
			}
			if len(collected) > 0 {
				return collected
			}
		}
	}
	return nil
}

func dlmSplitCSV(raw string) []string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token != "" {
			out = append(out, token)
		}
	}
	return out
}

func dlmStringAny(values map[string]any, keys []string, def string) string {
	for _, key := range keys {
		if raw, ok := values[key]; ok {
			switch v := raw.(type) {
			case string:
				if trimmed := strings.TrimSpace(v); trimmed != "" {
					return trimmed
				}
			case []string:
				if len(v) > 0 {
					if trimmed := strings.TrimSpace(v[0]); trimmed != "" {
						return trimmed
					}
				}
			case []any:
				if len(v) > 0 {
					if trimmed := strings.TrimSpace(fmt.Sprint(v[0])); trimmed != "" {
						return trimmed
					}
				}
			default:
				if raw != nil {
					if trimmed := strings.TrimSpace(fmt.Sprint(raw)); trimmed != "" {
						return trimmed
					}
				}
			}
		}
	}
	return def
}

func dlmString(values map[string]any, key, def string) string {
	if raw, ok := values[key]; ok && raw != nil {
		if text := strings.TrimSpace(fmt.Sprint(raw)); text != "" {
			return text
		}
	}
	return def
}

func dlmStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
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
		return dlmSplitCSV(v)
	default:
		return nil
	}
}

func dlmCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func dlmCloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func dlmPolicyARN(policyID string) string {
	id := strings.TrimSpace(policyID)
	if id == "" {
		id = "policy-00000001"
	}
	return fmt.Sprintf("arn:aws:dlm:us-east-1:123456789012:policy/%s", id)
}
