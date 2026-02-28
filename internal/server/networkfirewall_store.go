package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type networkFirewallStore struct {
	mu sync.Mutex

	nextID int

	firewallPolicies map[string]map[string]any
	ruleGroups       map[string]map[string]any
	tlsConfigs       map[string]map[string]any
	tags             map[string]map[string]string
	resourcePolicy   map[string]string
}

func newNetworkFirewallStore() *networkFirewallStore {
	return &networkFirewallStore{
		nextID:           1,
		firewallPolicies: map[string]map[string]any{},
		ruleGroups:       map[string]map[string]any{},
		tlsConfigs:       map[string]map[string]any{},
		tags:             map[string]map[string]string{},
		resourcePolicy:   map[string]string{},
	}
}

func (s *networkFirewallStore) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, known := networkFirewallOperationByName[action]; !known {
		return map[string]any{}
	}

	s.nextID++
	updateToken := fmt.Sprintf("ut-%06d", s.nextID)

	switch action {
	case "CreateFirewallPolicy":
		name := networkFirewallPayloadString(payload, "FirewallPolicyName", "stackyard-policy")
		arn := networkFirewallPolicyARN(name)
		resp := networkFirewallPolicyResponse(arn, name)
		s.firewallPolicies[arn] = map[string]any{"name": name, "arn": arn}
		return map[string]any{"UpdateToken": updateToken, "FirewallPolicyResponse": resp}
	case "DescribeFirewallPolicy", "UpdateFirewallPolicy":
		arn := networkFirewallPayloadString(payload, "FirewallPolicyArn", networkFirewallPolicyARN("stackyard-policy"))
		name := networkFirewallNameFromARN(arn, "stackyard-policy")
		resp := networkFirewallPolicyResponse(arn, name)
		s.firewallPolicies[arn] = map[string]any{"name": name, "arn": arn}
		return map[string]any{"UpdateToken": updateToken, "FirewallPolicyResponse": resp}
	case "DeleteFirewallPolicy":
		arn := networkFirewallPayloadString(payload, "FirewallPolicyArn", networkFirewallPolicyARN("stackyard-policy"))
		name := networkFirewallNameFromARN(arn, "stackyard-policy")
		delete(s.firewallPolicies, arn)
		return map[string]any{"FirewallPolicyResponse": networkFirewallPolicyResponse(arn, name)}

	case "CreateRuleGroup":
		name := networkFirewallPayloadString(payload, "RuleGroupName", "stackyard-rule-group")
		arn := networkFirewallRuleGroupARN(name)
		resp := networkFirewallRuleGroupResponse(arn, name)
		s.ruleGroups[arn] = map[string]any{"name": name, "arn": arn}
		return map[string]any{"UpdateToken": updateToken, "RuleGroupResponse": resp}
	case "DescribeRuleGroup", "UpdateRuleGroup":
		arn := networkFirewallPayloadString(payload, "RuleGroupArn", networkFirewallRuleGroupARN("stackyard-rule-group"))
		name := networkFirewallNameFromARN(arn, "stackyard-rule-group")
		resp := networkFirewallRuleGroupResponse(arn, name)
		s.ruleGroups[arn] = map[string]any{"name": name, "arn": arn}
		return map[string]any{"UpdateToken": updateToken, "RuleGroupResponse": resp}
	case "DescribeRuleGroupMetadata":
		arn := networkFirewallPayloadString(payload, "RuleGroupArn", networkFirewallRuleGroupARN("stackyard-rule-group"))
		name := networkFirewallNameFromARN(arn, "stackyard-rule-group")
		return map[string]any{"RuleGroupArn": arn, "RuleGroupName": name}
	case "DeleteRuleGroup":
		arn := networkFirewallPayloadString(payload, "RuleGroupArn", networkFirewallRuleGroupARN("stackyard-rule-group"))
		name := networkFirewallNameFromARN(arn, "stackyard-rule-group")
		delete(s.ruleGroups, arn)
		return map[string]any{"RuleGroupResponse": networkFirewallRuleGroupResponse(arn, name)}

	case "CreateTLSInspectionConfiguration":
		name := networkFirewallPayloadString(payload, "TLSInspectionConfigurationName", "stackyard-tls")
		arn := networkFirewallTLSARN(name)
		resp := networkFirewallTLSResponse(arn, name)
		s.tlsConfigs[arn] = map[string]any{"name": name, "arn": arn}
		return map[string]any{"UpdateToken": updateToken, "TLSInspectionConfigurationResponse": resp}
	case "DescribeTLSInspectionConfiguration", "UpdateTLSInspectionConfiguration":
		arn := networkFirewallPayloadString(payload, "TLSInspectionConfigurationArn", networkFirewallTLSARN("stackyard-tls"))
		name := networkFirewallNameFromARN(arn, "stackyard-tls")
		resp := networkFirewallTLSResponse(arn, name)
		s.tlsConfigs[arn] = map[string]any{"name": name, "arn": arn}
		return map[string]any{"UpdateToken": updateToken, "TLSInspectionConfigurationResponse": resp}
	case "DeleteTLSInspectionConfiguration":
		arn := networkFirewallPayloadString(payload, "TLSInspectionConfigurationArn", networkFirewallTLSARN("stackyard-tls"))
		name := networkFirewallNameFromARN(arn, "stackyard-tls")
		delete(s.tlsConfigs, arn)
		return map[string]any{"TLSInspectionConfigurationResponse": networkFirewallTLSResponse(arn, name)}

	case "ListFirewallPolicies":
		items := make([]any, 0, len(s.firewallPolicies))
		keys := make([]string, 0, len(s.firewallPolicies))
		for arn := range s.firewallPolicies {
			keys = append(keys, arn)
		}
		sort.Strings(keys)
		for _, arn := range keys {
			name := networkFirewallNameFromARN(arn, "stackyard-policy")
			items = append(items, map[string]any{"Name": name, "Arn": arn})
		}
		return map[string]any{"FirewallPolicies": items, "NextToken": ""}
	case "ListRuleGroups":
		items := make([]any, 0, len(s.ruleGroups))
		keys := make([]string, 0, len(s.ruleGroups))
		for arn := range s.ruleGroups {
			keys = append(keys, arn)
		}
		sort.Strings(keys)
		for _, arn := range keys {
			name := networkFirewallNameFromARN(arn, "stackyard-rule-group")
			items = append(items, map[string]any{"Name": name, "Arn": arn})
		}
		return map[string]any{"RuleGroups": items, "NextToken": ""}
	case "ListTLSInspectionConfigurations":
		items := make([]any, 0, len(s.tlsConfigs))
		keys := make([]string, 0, len(s.tlsConfigs))
		for arn := range s.tlsConfigs {
			keys = append(keys, arn)
		}
		sort.Strings(keys)
		for _, arn := range keys {
			name := networkFirewallNameFromARN(arn, "stackyard-tls")
			items = append(items, map[string]any{"Name": name, "Arn": arn})
		}
		return map[string]any{"TLSInspectionConfigurations": items, "NextToken": ""}
	case "ListFirewalls":
		return map[string]any{"Firewalls": []any{}, "NextToken": ""}

	case "TagResource":
		arn := networkFirewallPayloadString(payload, "ResourceArn", networkFirewallFirewallARN("stackyard-firewall"))
		tags := s.ensureTagsLocked(arn)
		for _, entry := range networkFirewallPayloadTags(payload) {
			tags[entry["Key"]] = entry["Value"]
		}
		return map[string]any{}
	case "ListTagsForResource":
		arn := networkFirewallPayloadString(payload, "ResourceArn", networkFirewallFirewallARN("stackyard-firewall"))
		tags := s.ensureTagsLocked(arn)
		keys := make([]string, 0, len(tags))
		for k := range tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]any, 0, len(keys))
		for _, k := range keys {
			out = append(out, map[string]any{"Key": k, "Value": tags[k]})
		}
		return map[string]any{"Tags": out, "NextToken": ""}
	case "UntagResource":
		arn := networkFirewallPayloadString(payload, "ResourceArn", networkFirewallFirewallARN("stackyard-firewall"))
		tags := s.ensureTagsLocked(arn)
		for _, key := range networkFirewallPayloadTagKeys(payload) {
			delete(tags, key)
		}
		return map[string]any{}

	case "PutResourcePolicy":
		arn := networkFirewallPayloadString(payload, "ResourceArn", networkFirewallFirewallARN("stackyard-firewall"))
		s.resourcePolicy[arn] = networkFirewallPayloadString(payload, "Policy", "{}")
		return map[string]any{}
	case "DescribeResourcePolicy":
		arn := networkFirewallPayloadString(payload, "ResourceArn", networkFirewallFirewallARN("stackyard-firewall"))
		policy := strings.TrimSpace(s.resourcePolicy[arn])
		if policy == "" {
			policy = "{}"
		}
		return map[string]any{"Policy": policy}
	case "DeleteResourcePolicy":
		arn := networkFirewallPayloadString(payload, "ResourceArn", networkFirewallFirewallARN("stackyard-firewall"))
		delete(s.resourcePolicy, arn)
		return map[string]any{}

	default:
		return map[string]any{}
	}
}

func (s *networkFirewallStore) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = networkFirewallFirewallARN("stackyard-firewall")
	}
	if existing, ok := s.tags[resourceARN]; ok {
		return existing
	}
	created := map[string]string{}
	s.tags[resourceARN] = created
	return created
}

func networkFirewallPayloadString(payload map[string]any, key, fallback string) string {
	for k, v := range payload {
		if !strings.EqualFold(k, key) {
			continue
		}
		if s, ok := v.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				return s
			}
		}
	}
	return fallback
}

func networkFirewallPayloadTags(payload map[string]any) []map[string]string {
	raw, ok := payload["Tags"]
	if !ok {
		raw = payload["tags"]
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]string, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		k := strings.TrimSpace(fmt.Sprintf("%v", m["Key"]))
		v := strings.TrimSpace(fmt.Sprintf("%v", m["Value"]))
		if k == "" {
			continue
		}
		out = append(out, map[string]string{"Key": k, "Value": v})
	}
	return out
}

func networkFirewallPayloadTagKeys(payload map[string]any) []string {
	raw, ok := payload["TagKeys"]
	if !ok {
		raw = payload["tagKeys"]
	}
	list, ok := raw.([]any)
	if !ok {
		if strList, ok2 := raw.([]string); ok2 {
			out := make([]string, 0, len(strList))
			for _, item := range strList {
				item = strings.TrimSpace(item)
				if item != "" {
					out = append(out, item)
				}
			}
			return out
		}
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		key := strings.TrimSpace(fmt.Sprintf("%v", item))
		if key != "" {
			out = append(out, key)
		}
	}
	return out
}

func networkFirewallPolicyResponse(arn, name string) map[string]any {
	return map[string]any{
		"FirewallPolicyName": name,
		"FirewallPolicyArn":  arn,
		"FirewallPolicyId":   networkFirewallIDFromName(name),
	}
}

func networkFirewallRuleGroupResponse(arn, name string) map[string]any {
	return map[string]any{
		"RuleGroupArn":  arn,
		"RuleGroupName": name,
		"RuleGroupId":   networkFirewallIDFromName(name),
	}
}

func networkFirewallTLSResponse(arn, name string) map[string]any {
	return map[string]any{
		"TLSInspectionConfigurationArn":  arn,
		"TLSInspectionConfigurationName": name,
		"TLSInspectionConfigurationId":   networkFirewallIDFromName(name),
	}
}

func networkFirewallIDFromName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "stackyard-id"
	}
	return strings.ReplaceAll(name, " ", "-")
}

func networkFirewallNameFromARN(arn, fallback string) string {
	arn = strings.TrimSpace(arn)
	if arn == "" {
		return fallback
	}
	parts := strings.Split(arn, "/")
	if len(parts) == 0 {
		return fallback
	}
	name := strings.TrimSpace(parts[len(parts)-1])
	if name == "" {
		return fallback
	}
	return name
}

func networkFirewallPolicyARN(name string) string {
	return fmt.Sprintf("arn:aws:network-firewall:us-east-1:123456789012:firewall-policy/%s", strings.TrimSpace(name))
}

func networkFirewallRuleGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:network-firewall:us-east-1:123456789012:rulegroup/%s", strings.TrimSpace(name))
}

func networkFirewallTLSARN(name string) string {
	return fmt.Sprintf("arn:aws:network-firewall:us-east-1:123456789012:tls-configuration/%s", strings.TrimSpace(name))
}

func networkFirewallFirewallARN(name string) string {
	return fmt.Sprintf("arn:aws:network-firewall:us-east-1:123456789012:firewall/%s", strings.TrimSpace(name))
}
