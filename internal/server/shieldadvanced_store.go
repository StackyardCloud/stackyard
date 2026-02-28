package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type shieldAdvancedStore struct {
	mu sync.Mutex

	subscriptionActive bool
	proactiveEnabled   bool
	drtRoleArn         string
	drtLogBucket       string
	emergencyContacts  []map[string]any
	protections        map[string]map[string]any
	protectionGroups   map[string]map[string]any
	attacks            map[string]map[string]any
	resourceTags       map[string]map[string]string
}

func newShieldAdvancedStore() *shieldAdvancedStore {
	now := time.Now().UTC().Format(time.RFC3339)
	protectionID := "protection-00000001"
	protectionARN := shieldAdvancedProtectionARN(protectionID)
	resourceARN := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/stackyard/0000000000000001"
	groupID := "stackyard-protection-group"
	attackID := "attack-00000001"

	return &shieldAdvancedStore{
		subscriptionActive: true,
		proactiveEnabled:   true,
		drtRoleArn:         "arn:aws:iam::123456789012:role/stackyard-shield-drt",
		drtLogBucket:       "stackyard-shield-drt-logs",
		emergencyContacts: []map[string]any{
			{"EmailAddress": "security@example.com", "PhoneNumber": "+12065550100", "ContactNotes": "Primary"},
		},
		protections: map[string]map[string]any{
			protectionID: {
				"Id":            protectionID,
				"Name":          "stackyard-protection",
				"ResourceArn":   resourceARN,
				"ProtectionArn": protectionARN,
			},
		},
		protectionGroups: map[string]map[string]any{
			groupID: {
				"ProtectionGroupId": groupID,
				"Pattern":           "ARBITRARY",
				"Aggregation":       "SUM",
				"Members":           []any{resourceARN},
			},
		},
		attacks: map[string]map[string]any{
			attackID: {
				"AttackId":         attackID,
				"ResourceArn":      resourceARN,
				"StartTime":        now,
				"EndTime":          now,
				"AttackCounters":   []any{},
				"SubResources":     []any{},
				"Mitigations":      []any{},
				"AttackProperties": []any{},
			},
		},
		resourceTags: map[string]map[string]string{
			protectionARN: {"env": "coverage"},
		},
	}
}

func (s *shieldAdvancedStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	protectionID := shieldAdvancedPayloadString(payload, "ProtectionId", "protection-00000001")
	protectionGroupID := shieldAdvancedPayloadString(payload, "ProtectionGroupId", "stackyard-protection-group")
	attackID := shieldAdvancedPayloadString(payload, "AttackId", "attack-00000001")
	resourceARN := shieldAdvancedPayloadString(
		payload,
		"ResourceArn",
		"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/stackyard/0000000000000001",
	)
	protectionARN := shieldAdvancedProtectionARN(protectionID)

	switch action {
	case "CreateSubscription":
		s.subscriptionActive = true
		return map[string]any{}
	case "DeleteSubscription":
		s.subscriptionActive = false
		return map[string]any{}
	case "GetSubscriptionState":
		if s.subscriptionActive {
			return map[string]any{"SubscriptionState": "ACTIVE"}
		}
		return map[string]any{"SubscriptionState": "INACTIVE"}
	case "DescribeSubscription":
		return map[string]any{
			"Subscription": map[string]any{
				"StartTime":               now,
				"TimeCommitmentInSeconds": int64(31536000),
				"AutoRenew":               "ENABLED",
				"Limits": []any{
					map[string]any{"Type": "MAX_PROTECTIONS", "Max": int64(1000)},
				},
				"ProactiveEngagementStatus": map[bool]string{true: "ENABLED", false: "DISABLED"}[s.proactiveEnabled],
			},
		}
	case "UpdateSubscription":
		return map[string]any{}

	case "CreateProtection":
		name := shieldAdvancedPayloadString(payload, "Name", "stackyard-protection")
		protectionID = fmt.Sprintf("protection-%08d", len(s.protections)+1)
		protectionARN = shieldAdvancedProtectionARN(protectionID)
		s.protections[protectionID] = map[string]any{
			"Id":            protectionID,
			"Name":          name,
			"ResourceArn":   resourceARN,
			"ProtectionArn": protectionARN,
		}
		return map[string]any{"ProtectionId": protectionID}
	case "DeleteProtection":
		delete(s.protections, protectionID)
		return map[string]any{}
	case "DescribeProtection":
		return map[string]any{"Protection": shieldAdvancedCloneMap(s.ensureProtectionLocked(protectionID))}
	case "ListProtections":
		return map[string]any{"Protections": s.listProtectionsLocked(), "NextToken": ""}

	case "CreateProtectionGroup":
		s.protectionGroups[protectionGroupID] = map[string]any{
			"ProtectionGroupId": protectionGroupID,
			"Pattern":           shieldAdvancedPayloadString(payload, "Pattern", "ARBITRARY"),
			"Aggregation":       shieldAdvancedPayloadString(payload, "Aggregation", "SUM"),
			"Members":           shieldAdvancedPayloadAnySlice(payload, "Members"),
		}
		return map[string]any{}
	case "UpdateProtectionGroup":
		group := s.ensureProtectionGroupLocked(protectionGroupID)
		for k, v := range payload {
			group[k] = v
		}
		return map[string]any{}
	case "DeleteProtectionGroup":
		delete(s.protectionGroups, protectionGroupID)
		return map[string]any{}
	case "DescribeProtectionGroup":
		return map[string]any{"ProtectionGroup": shieldAdvancedCloneMap(s.ensureProtectionGroupLocked(protectionGroupID))}
	case "ListProtectionGroups":
		return map[string]any{"ProtectionGroups": s.listProtectionGroupsLocked(), "NextToken": ""}
	case "ListResourcesInProtectionGroup":
		group := s.ensureProtectionGroupLocked(protectionGroupID)
		members := shieldAdvancedSliceOfStrings(group["Members"])
		return map[string]any{"ResourceArns": members, "NextToken": ""}

	case "DescribeAttack":
		return map[string]any{"Attack": shieldAdvancedCloneMap(s.ensureAttackLocked(attackID))}
	case "DescribeAttackStatistics":
		return map[string]any{
			"TimeRange": map[string]any{"FromInclusive": now, "ToExclusive": now},
			"DataItems": []any{},
		}
	case "ListAttacks":
		return map[string]any{"AttackSummaries": s.listAttackSummariesLocked(), "NextToken": ""}

	case "AssociateDRTRole":
		s.drtRoleArn = shieldAdvancedPayloadString(payload, "RoleArn", s.drtRoleArn)
		return map[string]any{}
	case "DisassociateDRTRole":
		s.drtRoleArn = ""
		return map[string]any{}
	case "AssociateDRTLogBucket":
		s.drtLogBucket = shieldAdvancedPayloadString(payload, "LogBucket", "stackyard-shield-drt-logs")
		return map[string]any{}
	case "DisassociateDRTLogBucket":
		s.drtLogBucket = ""
		return map[string]any{}
	case "DescribeDRTAccess":
		buckets := []any{}
		if strings.TrimSpace(s.drtLogBucket) != "" {
			buckets = append(buckets, s.drtLogBucket)
		}
		return map[string]any{
			"RoleArn":       s.drtRoleArn,
			"LogBucketList": buckets,
		}

	case "AssociateHealthCheck", "DisassociateHealthCheck":
		return map[string]any{}

	case "EnableProactiveEngagement":
		s.proactiveEnabled = true
		return map[string]any{}
	case "DisableProactiveEngagement":
		s.proactiveEnabled = false
		return map[string]any{}
	case "AssociateProactiveEngagementDetails":
		s.emergencyContacts = shieldAdvancedPayloadContactList(payload, "EmergencyContactList")
		if len(s.emergencyContacts) == 0 {
			s.emergencyContacts = []map[string]any{
				{"EmailAddress": "security@example.com", "PhoneNumber": "+12065550100", "ContactNotes": "Primary"},
			}
		}
		return map[string]any{}
	case "UpdateEmergencyContactSettings":
		s.emergencyContacts = shieldAdvancedPayloadContactList(payload, "EmergencyContactList")
		return map[string]any{}
	case "DescribeEmergencyContactSettings":
		return map[string]any{"EmergencyContactList": shieldAdvancedCloneListOfMaps(s.emergencyContacts)}

	case "EnableApplicationLayerAutomaticResponse",
		"DisableApplicationLayerAutomaticResponse",
		"UpdateApplicationLayerAutomaticResponse":
		return map[string]any{}

	case "TagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for k, v := range shieldAdvancedPayloadTags(payload, "Tags") {
			tags[k] = v
		}
		return map[string]any{}
	case "UntagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for _, k := range shieldAdvancedPayloadStringSlice(payload, "TagKeys") {
			delete(tags, k)
		}
		return map[string]any{}
	case "ListTagsForResource":
		tags := s.ensureTagsLocked(resourceARN)
		out := make([]any, 0, len(tags))
		keys := make([]string, 0, len(tags))
		for k := range tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			out = append(out, map[string]any{"Key": k, "Value": tags[k]})
		}
		return map[string]any{"Tags": out}
	}

	return map[string]any{}
}

func (s *shieldAdvancedStore) ensureProtectionLocked(protectionID string) map[string]any {
	protectionID = strings.TrimSpace(protectionID)
	if protectionID == "" {
		protectionID = "protection-00000001"
	}
	if existing, ok := s.protections[protectionID]; ok {
		return existing
	}
	protection := map[string]any{
		"Id":            protectionID,
		"Name":          "stackyard-protection",
		"ResourceArn":   "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/stackyard/0000000000000001",
		"ProtectionArn": shieldAdvancedProtectionARN(protectionID),
	}
	s.protections[protectionID] = protection
	return protection
}

func (s *shieldAdvancedStore) ensureProtectionGroupLocked(groupID string) map[string]any {
	groupID = strings.TrimSpace(groupID)
	if groupID == "" {
		groupID = "stackyard-protection-group"
	}
	if existing, ok := s.protectionGroups[groupID]; ok {
		return existing
	}
	group := map[string]any{
		"ProtectionGroupId": groupID,
		"Pattern":           "ARBITRARY",
		"Aggregation":       "SUM",
		"Members": []any{
			"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/stackyard/0000000000000001",
		},
	}
	s.protectionGroups[groupID] = group
	return group
}

func (s *shieldAdvancedStore) ensureAttackLocked(attackID string) map[string]any {
	attackID = strings.TrimSpace(attackID)
	if attackID == "" {
		attackID = "attack-00000001"
	}
	if existing, ok := s.attacks[attackID]; ok {
		return existing
	}
	now := time.Now().UTC().Format(time.RFC3339)
	attack := map[string]any{
		"AttackId":         attackID,
		"ResourceArn":      "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/stackyard/0000000000000001",
		"StartTime":        now,
		"EndTime":          now,
		"AttackCounters":   []any{},
		"SubResources":     []any{},
		"Mitigations":      []any{},
		"AttackProperties": []any{},
	}
	s.attacks[attackID] = attack
	return attack
}

func (s *shieldAdvancedStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = shieldAdvancedProtectionARN("protection-00000001")
	}
	if existing, ok := s.resourceTags[resourceARN]; ok {
		return existing
	}
	created := map[string]string{}
	s.resourceTags[resourceARN] = created
	return created
}

func (s *shieldAdvancedStore) listProtectionsLocked() []any {
	keys := make([]string, 0, len(s.protections))
	for k := range s.protections {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, shieldAdvancedCloneMap(s.protections[k]))
	}
	return out
}

func (s *shieldAdvancedStore) listProtectionGroupsLocked() []any {
	keys := make([]string, 0, len(s.protectionGroups))
	for k := range s.protectionGroups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, shieldAdvancedCloneMap(s.protectionGroups[k]))
	}
	return out
}

func (s *shieldAdvancedStore) listAttackSummariesLocked() []any {
	keys := make([]string, 0, len(s.attacks))
	for k := range s.attacks {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		src := s.attacks[k]
		out = append(out, map[string]any{
			"AttackId":    src["AttackId"],
			"ResourceArn": src["ResourceArn"],
			"StartTime":   src["StartTime"],
			"EndTime":     src["EndTime"],
		})
	}
	return out
}

func shieldAdvancedProtectionARN(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "protection-00000001"
	}
	return fmt.Sprintf("arn:aws:shield::123456789012:protection/%s", id)
}

func shieldAdvancedPayloadString(payload map[string]any, key, fallback string) string {
	if payload != nil {
		for k, v := range payload {
			if strings.EqualFold(strings.TrimSpace(k), key) {
				s := strings.TrimSpace(fmt.Sprintf("%v", v))
				if s != "" {
					return s
				}
			}
		}
	}
	return fallback
}

func shieldAdvancedPayloadAnySlice(payload map[string]any, key string) []any {
	if payload == nil {
		return []any{}
	}
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), key) {
			continue
		}
		if out, ok := v.([]any); ok {
			return out
		}
	}
	return []any{}
}

func shieldAdvancedPayloadStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return nil
	}
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), key) {
			continue
		}
		switch vv := v.(type) {
		case []string:
			out := make([]string, 0, len(vv))
			for _, item := range vv {
				if s := strings.TrimSpace(item); s != "" {
					out = append(out, s)
				}
			}
			return out
		case []any:
			out := make([]string, 0, len(vv))
			for _, item := range vv {
				if s := strings.TrimSpace(fmt.Sprintf("%v", item)); s != "" {
					out = append(out, s)
				}
			}
			return out
		}
	}
	return nil
}

func shieldAdvancedPayloadTags(payload map[string]any, key string) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}
	var raw any
	for k, v := range payload {
		if strings.EqualFold(strings.TrimSpace(k), key) {
			raw = v
			break
		}
	}
	switch vv := raw.(type) {
	case map[string]any:
		for k, v := range vv {
			tagKey := strings.TrimSpace(k)
			if tagKey == "" {
				continue
			}
			out[tagKey] = strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	case []any:
		for _, item := range vv {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			tagKey := strings.TrimSpace(fmt.Sprintf("%v", m["Key"]))
			if tagKey == "" {
				tagKey = strings.TrimSpace(fmt.Sprintf("%v", m["key"]))
			}
			if tagKey == "" {
				continue
			}
			val := strings.TrimSpace(fmt.Sprintf("%v", m["Value"]))
			if val == "" {
				val = strings.TrimSpace(fmt.Sprintf("%v", m["value"]))
			}
			out[tagKey] = val
		}
	}
	return out
}

func shieldAdvancedPayloadContactList(payload map[string]any, key string) []map[string]any {
	if payload == nil {
		return nil
	}
	for k, v := range payload {
		if !strings.EqualFold(strings.TrimSpace(k), key) {
			continue
		}
		items, ok := v.([]any)
		if !ok {
			return nil
		}
		out := make([]map[string]any, 0, len(items))
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			out = append(out, shieldAdvancedCloneMap(m))
		}
		return out
	}
	return nil
}

func shieldAdvancedSliceOfStrings(v any) []string {
	switch vv := v.(type) {
	case []string:
		out := make([]string, 0, len(vv))
		for _, s := range vv {
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(vv))
		for _, item := range vv {
			s := strings.TrimSpace(fmt.Sprintf("%v", item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func shieldAdvancedCloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func shieldAdvancedCloneListOfMaps(in []map[string]any) []any {
	out := make([]any, 0, len(in))
	for _, item := range in {
		out = append(out, shieldAdvancedCloneMap(item))
	}
	return out
}
