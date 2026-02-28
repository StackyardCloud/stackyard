package server

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type wafv2Store struct {
	mu sync.Mutex

	webACLs               map[string]map[string]any
	ipSets                map[string]map[string]any
	regexPatternSets      map[string]map[string]any
	ruleGroups            map[string]map[string]any
	apiKeys               map[string]map[string]any
	loggingConfigurations map[string]map[string]any
	resourceTags          map[string]map[string]string
	permissionPolicy      string
}

func newWAFV2Store() *wafv2Store {
	now := time.Now().UTC().Format(time.RFC3339)
	defaultWebACL := map[string]any{
		"Name":        "stackyard-web-acl",
		"Id":          "wacl-00000001",
		"ARN":         "arn:aws:wafv2:us-east-1:123456789012:regional/webacl/stackyard-web-acl/wacl-00000001",
		"Description": "Stackyard seeded WebACL",
		"Scope":       "REGIONAL",
		"Capacity":    int64(1),
		"LockToken":   "token-00000001",
		"VisibilityConfig": map[string]any{
			"SampledRequestsEnabled":   true,
			"CloudWatchMetricsEnabled": true,
			"MetricName":               "stackyardWebACL",
		},
		"Rules":                    []any{},
		"DefaultAction":            map[string]any{"Allow": map[string]any{}},
		"ManagedByFirewallManager": false,
		"LabelNamespace":           "awswaf:123456789012:webacl:stackyard-web-acl:",
		"CreationTime":             now,
	}

	return &wafv2Store{
		webACLs: map[string]map[string]any{
			"wacl-00000001": defaultWebACL,
		},
		ipSets:                map[string]map[string]any{},
		regexPatternSets:      map[string]map[string]any{},
		ruleGroups:            map[string]map[string]any{},
		apiKeys:               map[string]map[string]any{},
		loggingConfigurations: map[string]map[string]any{},
		resourceTags: map[string]map[string]string{
			"arn:aws:wafv2:us-east-1:123456789012:regional/webacl/stackyard-web-acl/wacl-00000001": {"env": "coverage"},
		},
		permissionPolicy: "{}",
	}
}

func (s *wafv2Store) Handle(action string, payload map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	scope := wafv2PayloadString(payload, "Scope", "REGIONAL")
	lockToken := wafv2PayloadString(payload, "LockToken", wafv2NextToken())
	resourceARN := wafv2PayloadString(
		payload,
		"ResourceARN",
		"arn:aws:wafv2:us-east-1:123456789012:regional/webacl/stackyard-web-acl/wacl-00000001",
	)
	if strings.TrimSpace(resourceARN) == "" {
		resourceARN = wafv2PayloadString(
			payload,
			"ResourceArn",
			"arn:aws:wafv2:us-east-1:123456789012:regional/webacl/stackyard-web-acl/wacl-00000001",
		)
	}

	switch action {
	case "CreateWebACL":
		name := wafv2PayloadString(payload, "Name", "stackyard-web-acl")
		id := fmt.Sprintf("wacl-%08d", len(s.webACLs)+1)
		arn := fmt.Sprintf("arn:aws:wafv2:us-east-1:123456789012:%s/webacl/%s/%s", wafv2ScopeNamespace(scope), name, id)
		nextToken := wafv2NextToken()
		webACL := map[string]any{
			"Name":        name,
			"Id":          id,
			"ARN":         arn,
			"Description": wafv2PayloadString(payload, "Description", ""),
			"Scope":       scope,
			"Capacity":    int64(1),
			"LockToken":   nextToken,
			"Rules":       wafv2PayloadAnySlice(payload, "Rules"),
			"VisibilityConfig": map[string]any{
				"SampledRequestsEnabled":   true,
				"CloudWatchMetricsEnabled": true,
				"MetricName":               name,
			},
		}
		s.webACLs[id] = webACL
		return map[string]any{
			"Summary": map[string]any{"Name": name, "Id": id, "ARN": arn, "LockToken": nextToken},
		}
	case "GetWebACL":
		webACL := s.ensureWebACLLocked(wafv2PayloadString(payload, "Id", "wacl-00000001"))
		return map[string]any{
			"WebACL":    wafv2CloneMap(webACL),
			"LockToken": wafv2PayloadString(webACL, "LockToken", wafv2NextToken()),
		}
	case "ListWebACLs":
		return map[string]any{"WebACLs": s.listWebACLSummariesLocked(), "NextMarker": ""}
	case "UpdateWebACL":
		webACL := s.ensureWebACLLocked(wafv2PayloadString(payload, "Id", "wacl-00000001"))
		webACL["LockToken"] = wafv2NextToken()
		return map[string]any{"NextLockToken": webACL["LockToken"]}
	case "DeleteWebACL":
		delete(s.webACLs, wafv2PayloadString(payload, "Id", "wacl-00000001"))
		return map[string]any{}
	case "AssociateWebACL", "DisassociateWebACL":
		return map[string]any{}
	case "GetWebACLForResource":
		webACL := s.ensureWebACLLocked("wacl-00000001")
		return map[string]any{"WebACL": wafv2CloneMap(webACL)}
	case "ListResourcesForWebACL":
		return map[string]any{
			"ResourceArns": []any{
				"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/stackyard/0000000000000001",
			},
		}

	case "CreateIPSet":
		name := wafv2PayloadString(payload, "Name", "stackyard-ipset")
		id := fmt.Sprintf("ipset-%08d", len(s.ipSets)+1)
		arn := fmt.Sprintf("arn:aws:wafv2:us-east-1:123456789012:%s/ipset/%s/%s", wafv2ScopeNamespace(scope), name, id)
		nextToken := wafv2NextToken()
		ipSet := map[string]any{
			"Name":             name,
			"Id":               id,
			"ARN":              arn,
			"IPAddressVersion": wafv2PayloadString(payload, "IPAddressVersion", "IPV4"),
			"Addresses":        wafv2PayloadAnySlice(payload, "Addresses"),
			"Description":      wafv2PayloadString(payload, "Description", ""),
			"LockToken":        nextToken,
		}
		s.ipSets[id] = ipSet
		return map[string]any{"Summary": map[string]any{"Name": name, "Id": id, "ARN": arn, "LockToken": nextToken}}
	case "GetIPSet":
		ipSet := s.ensureIPSetLocked(wafv2PayloadString(payload, "Id", "ipset-00000001"))
		return map[string]any{"IPSet": wafv2CloneMap(ipSet), "LockToken": ipSet["LockToken"]}
	case "ListIPSets":
		return map[string]any{"IPSets": s.listSummariesLocked(s.ipSets), "NextMarker": ""}
	case "UpdateIPSet":
		ipSet := s.ensureIPSetLocked(wafv2PayloadString(payload, "Id", "ipset-00000001"))
		ipSet["Addresses"] = wafv2PayloadAnySlice(payload, "Addresses")
		ipSet["LockToken"] = wafv2NextToken()
		return map[string]any{"NextLockToken": ipSet["LockToken"]}
	case "DeleteIPSet":
		delete(s.ipSets, wafv2PayloadString(payload, "Id", "ipset-00000001"))
		return map[string]any{}

	case "CreateRegexPatternSet":
		name := wafv2PayloadString(payload, "Name", "stackyard-regex")
		id := fmt.Sprintf("regex-%08d", len(s.regexPatternSets)+1)
		arn := fmt.Sprintf("arn:aws:wafv2:us-east-1:123456789012:%s/regexpatternset/%s/%s", wafv2ScopeNamespace(scope), name, id)
		nextToken := wafv2NextToken()
		set := map[string]any{
			"Name":                  name,
			"Id":                    id,
			"ARN":                   arn,
			"Description":           wafv2PayloadString(payload, "Description", ""),
			"RegularExpressionList": wafv2PayloadAnySlice(payload, "RegularExpressionList"),
			"LockToken":             nextToken,
		}
		s.regexPatternSets[id] = set
		return map[string]any{"Summary": map[string]any{"Name": name, "Id": id, "ARN": arn, "LockToken": nextToken}}
	case "GetRegexPatternSet":
		set := s.ensureRegexPatternSetLocked(wafv2PayloadString(payload, "Id", "regex-00000001"))
		return map[string]any{"RegexPatternSet": wafv2CloneMap(set), "LockToken": set["LockToken"]}
	case "ListRegexPatternSets":
		return map[string]any{"RegexPatternSets": s.listSummariesLocked(s.regexPatternSets), "NextMarker": ""}
	case "UpdateRegexPatternSet":
		set := s.ensureRegexPatternSetLocked(wafv2PayloadString(payload, "Id", "regex-00000001"))
		set["RegularExpressionList"] = wafv2PayloadAnySlice(payload, "RegularExpressionList")
		set["LockToken"] = wafv2NextToken()
		return map[string]any{"NextLockToken": set["LockToken"]}
	case "DeleteRegexPatternSet":
		delete(s.regexPatternSets, wafv2PayloadString(payload, "Id", "regex-00000001"))
		return map[string]any{}

	case "CreateRuleGroup":
		name := wafv2PayloadString(payload, "Name", "stackyard-rulegroup")
		id := fmt.Sprintf("rg-%08d", len(s.ruleGroups)+1)
		arn := fmt.Sprintf("arn:aws:wafv2:us-east-1:123456789012:%s/rulegroup/%s/%s", wafv2ScopeNamespace(scope), name, id)
		nextToken := wafv2NextToken()
		group := map[string]any{
			"Name":             name,
			"Id":               id,
			"ARN":              arn,
			"Description":      wafv2PayloadString(payload, "Description", ""),
			"Capacity":         int64(1),
			"Rules":            wafv2PayloadAnySlice(payload, "Rules"),
			"VisibilityConfig": wafv2CloneMapAny(payload["VisibilityConfig"]),
			"LockToken":        nextToken,
		}
		s.ruleGroups[id] = group
		return map[string]any{"Summary": map[string]any{"Name": name, "Id": id, "ARN": arn, "LockToken": nextToken}}
	case "GetRuleGroup":
		group := s.ensureRuleGroupLocked(wafv2PayloadString(payload, "Id", "rg-00000001"))
		return map[string]any{"RuleGroup": wafv2CloneMap(group), "LockToken": group["LockToken"]}
	case "ListRuleGroups":
		return map[string]any{"RuleGroups": s.listSummariesLocked(s.ruleGroups), "NextMarker": ""}
	case "UpdateRuleGroup":
		group := s.ensureRuleGroupLocked(wafv2PayloadString(payload, "Id", "rg-00000001"))
		group["Rules"] = wafv2PayloadAnySlice(payload, "Rules")
		group["LockToken"] = wafv2NextToken()
		return map[string]any{"NextLockToken": group["LockToken"]}
	case "DeleteRuleGroup":
		delete(s.ruleGroups, wafv2PayloadString(payload, "Id", "rg-00000001"))
		return map[string]any{}

	case "CheckCapacity":
		return map[string]any{"Capacity": int64(1)}
	case "GetRateBasedStatementManagedKeys":
		return map[string]any{
			"ManagedKeysIPV4": map[string]any{"Addresses": []any{}},
			"ManagedKeysIPV6": map[string]any{"Addresses": []any{}},
		}

	case "DescribeManagedRuleGroup":
		return map[string]any{
			"Capacity":       int64(1),
			"Rules":          []any{},
			"LabelNamespace": "awswaf:managed:aws:core-rule-set:",
		}
	case "DescribeAllManagedProducts":
		return map[string]any{"ManagedProducts": []any{}}
	case "DescribeManagedProductsByVendor":
		return map[string]any{"ManagedProducts": []any{}}
	case "ListAvailableManagedRuleGroups":
		return map[string]any{"ManagedRuleGroups": []any{}}
	case "ListAvailableManagedRuleGroupVersions":
		return map[string]any{"Versions": []any{}, "CurrentDefaultVersion": "1.0"}
	case "ListManagedRuleSets":
		return map[string]any{"ManagedRuleSets": []any{}, "NextMarker": ""}
	case "GetManagedRuleSet":
		return map[string]any{"ManagedRuleSet": map[string]any{"Name": "stackyard-managed-set", "Id": "mrs-00000001", "Description": "Stackyard managed rule set"}}
	case "PutManagedRuleSetVersions", "UpdateManagedRuleSetVersionExpiryDate":
		return map[string]any{}
	case "DeleteFirewallManagerRuleGroups":
		return map[string]any{}

	case "CreateAPIKey":
		token := wafv2NextToken()
		id := fmt.Sprintf("api-key-%08d", len(s.apiKeys)+1)
		apiKey := map[string]any{
			"APIKey":            "stackyard-api-key",
			"TokenDomains":      []any{wafv2PayloadString(payload, "TokenDomain", "localhost")},
			"CreationTimestamp": now,
			"Version":           int64(1),
		}
		s.apiKeys[id] = apiKey
		return map[string]any{"APIKey": apiKey["APIKey"], "LockToken": token}
	case "ListAPIKeys":
		summaries := make([]any, 0, len(s.apiKeys))
		ids := make([]string, 0, len(s.apiKeys))
		for id := range s.apiKeys {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			v := s.apiKeys[id]
			summaries = append(summaries, map[string]any{
				"APIKey":            wafv2PayloadString(v, "APIKey", "stackyard-api-key"),
				"CreationTimestamp": v["CreationTimestamp"],
				"Version":           v["Version"],
			})
		}
		return map[string]any{"APIKeySummaries": summaries, "NextMarker": ""}
	case "GetDecryptedAPIKey":
		return map[string]any{"APIKey": "stackyard-api-key"}
	case "DeleteAPIKey":
		for id := range s.apiKeys {
			delete(s.apiKeys, id)
			break
		}
		return map[string]any{}

	case "GenerateMobileSdkReleaseUrl":
		return map[string]any{"Url": "https://example.com/waf/mobile-sdk-release.zip"}
	case "GetMobileSdkRelease":
		return map[string]any{"MobileSdkRelease": map[string]any{"ReleaseVersion": "1.0.0", "Timestamp": now}}
	case "ListMobileSdkReleases":
		return map[string]any{"ReleaseSummaries": []any{}, "NextMarker": ""}

	case "PutLoggingConfiguration":
		cfg := map[string]any{
			"ResourceArn": resourceARN,
			"LogDestinationConfigs": []any{
				"arn:aws:logs:us-east-1:123456789012:log-group:/stackyard/waf",
			},
		}
		s.loggingConfigurations[resourceARN] = cfg
		return map[string]any{"LoggingConfiguration": cfg}
	case "GetLoggingConfiguration":
		cfg, ok := s.loggingConfigurations[resourceARN]
		if !ok {
			cfg = map[string]any{
				"ResourceArn":           resourceARN,
				"LogDestinationConfigs": []any{},
			}
		}
		return map[string]any{"LoggingConfiguration": wafv2CloneMap(cfg)}
	case "ListLoggingConfigurations":
		out := make([]any, 0, len(s.loggingConfigurations))
		keys := make([]string, 0, len(s.loggingConfigurations))
		for k := range s.loggingConfigurations {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			out = append(out, wafv2CloneMap(s.loggingConfigurations[k]))
		}
		return map[string]any{"LoggingConfigurations": out, "NextMarker": ""}
	case "DeleteLoggingConfiguration":
		delete(s.loggingConfigurations, resourceARN)
		return map[string]any{}

	case "GetSampledRequests":
		return map[string]any{
			"SampledRequests": []any{},
			"PopulationSize":  int64(0),
			"TimeWindow": map[string]any{
				"StartTime": now,
				"EndTime":   now,
			},
		}
	case "GetTopPathStatisticsByTraffic":
		return map[string]any{
			"PathStatistics": []any{},
			"TimeWindow": map[string]any{
				"StartTime": now,
				"EndTime":   now,
			},
		}

	case "PutPermissionPolicy":
		s.permissionPolicy = wafv2PayloadString(payload, "Policy", s.permissionPolicy)
		return map[string]any{}
	case "GetPermissionPolicy":
		return map[string]any{"Policy": s.permissionPolicy}
	case "DeletePermissionPolicy":
		s.permissionPolicy = "{}"
		return map[string]any{}

	case "TagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for k, v := range wafv2PayloadTags(payload, "Tags") {
			tags[k] = v
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
		return map[string]any{"TagInfoForResource": map[string]any{"ResourceARN": resourceARN, "TagList": out}}
	case "UntagResource":
		tags := s.ensureTagsLocked(resourceARN)
		for _, k := range wafv2PayloadStringSlice(payload, "TagKeys") {
			delete(tags, k)
		}
		return map[string]any{}
	}

	_ = lockToken
	return map[string]any{}
}

func (s *wafv2Store) ensureWebACLLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "wacl-00000001"
	}
	if existing, ok := s.webACLs[id]; ok {
		return existing
	}
	created := map[string]any{
		"Name":      "stackyard-web-acl",
		"Id":        id,
		"ARN":       fmt.Sprintf("arn:aws:wafv2:us-east-1:123456789012:regional/webacl/stackyard-web-acl/%s", id),
		"Scope":     "REGIONAL",
		"Capacity":  int64(1),
		"LockToken": wafv2NextToken(),
		"Rules":     []any{},
	}
	s.webACLs[id] = created
	return created
}

func (s *wafv2Store) ensureIPSetLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "ipset-00000001"
	}
	if existing, ok := s.ipSets[id]; ok {
		return existing
	}
	created := map[string]any{
		"Name":             "stackyard-ipset",
		"Id":               id,
		"ARN":              fmt.Sprintf("arn:aws:wafv2:us-east-1:123456789012:regional/ipset/stackyard-ipset/%s", id),
		"IPAddressVersion": "IPV4",
		"Addresses":        []any{},
		"LockToken":        wafv2NextToken(),
	}
	s.ipSets[id] = created
	return created
}

func (s *wafv2Store) ensureRegexPatternSetLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "regex-00000001"
	}
	if existing, ok := s.regexPatternSets[id]; ok {
		return existing
	}
	created := map[string]any{
		"Name":                  "stackyard-regex",
		"Id":                    id,
		"ARN":                   fmt.Sprintf("arn:aws:wafv2:us-east-1:123456789012:regional/regexpatternset/stackyard-regex/%s", id),
		"RegularExpressionList": []any{},
		"LockToken":             wafv2NextToken(),
	}
	s.regexPatternSets[id] = created
	return created
}

func (s *wafv2Store) ensureRuleGroupLocked(id string) map[string]any {
	id = strings.TrimSpace(id)
	if id == "" {
		id = "rg-00000001"
	}
	if existing, ok := s.ruleGroups[id]; ok {
		return existing
	}
	created := map[string]any{
		"Name":      "stackyard-rulegroup",
		"Id":        id,
		"ARN":       fmt.Sprintf("arn:aws:wafv2:us-east-1:123456789012:regional/rulegroup/stackyard-rulegroup/%s", id),
		"Rules":     []any{},
		"Capacity":  int64(1),
		"LockToken": wafv2NextToken(),
	}
	s.ruleGroups[id] = created
	return created
}

func (s *wafv2Store) listWebACLSummariesLocked() []any {
	out := make([]any, 0, len(s.webACLs))
	ids := make([]string, 0, len(s.webACLs))
	for id := range s.webACLs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		item := s.webACLs[id]
		out = append(out, map[string]any{
			"Name":      wafv2PayloadString(item, "Name", "stackyard-web-acl"),
			"Id":        wafv2PayloadString(item, "Id", id),
			"ARN":       wafv2PayloadString(item, "ARN", ""),
			"LockToken": wafv2PayloadString(item, "LockToken", ""),
		})
	}
	return out
}

func (s *wafv2Store) listSummariesLocked(in map[string]map[string]any) []any {
	out := make([]any, 0, len(in))
	ids := make([]string, 0, len(in))
	for id := range in {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		item := in[id]
		out = append(out, map[string]any{
			"Name":      wafv2PayloadString(item, "Name", ""),
			"Id":        wafv2PayloadString(item, "Id", id),
			"ARN":       wafv2PayloadString(item, "ARN", ""),
			"LockToken": wafv2PayloadString(item, "LockToken", ""),
		})
	}
	return out
}

func (s *wafv2Store) ensureTagsLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = "arn:aws:wafv2:us-east-1:123456789012:regional/webacl/stackyard-web-acl/wacl-00000001"
	}
	if tags, ok := s.resourceTags[resourceARN]; ok {
		return tags
	}
	tags := map[string]string{}
	s.resourceTags[resourceARN] = tags
	return tags
}

func wafv2PayloadString(payload map[string]any, key, def string) string {
	if payload == nil {
		return def
	}
	v, ok := payload[key]
	if !ok || v == nil {
		return def
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" {
		return def
	}
	return s
}

func wafv2PayloadAnySlice(payload map[string]any, key string) []any {
	if payload == nil {
		return []any{}
	}
	v, ok := payload[key]
	if !ok || v == nil {
		return []any{}
	}
	items, ok := v.([]any)
	if !ok {
		return []any{}
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, wafv2CloneAny(item))
	}
	return out
}

func wafv2PayloadTags(payload map[string]any, key string) map[string]string {
	out := map[string]string{}
	if payload == nil {
		return out
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return out
	}
	items, ok := raw.([]any)
	if !ok {
		return out
	}
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		k := wafv2PayloadString(m, "Key", "")
		if k == "" {
			continue
		}
		out[k] = wafv2PayloadString(m, "Value", "")
	}
	return out
}

func wafv2PayloadStringSlice(payload map[string]any, key string) []string {
	if payload == nil {
		return []string{}
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return []string{}
	}
	items, ok := raw.([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		val := strings.TrimSpace(fmt.Sprint(item))
		if val != "" {
			out = append(out, val)
		}
	}
	return out
}

func wafv2ScopeNamespace(scope string) string {
	if strings.EqualFold(strings.TrimSpace(scope), "CLOUDFRONT") {
		return "global"
	}
	return "regional"
}

func wafv2NextToken() string {
	return fmt.Sprintf("token-%d", time.Now().UnixNano())
}

func wafv2CloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = wafv2CloneAny(v)
	}
	return out
}

func wafv2CloneMapAny(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok || m == nil {
		return map[string]any{}
	}
	return wafv2CloneMap(m)
}

func wafv2CloneAny(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return wafv2CloneMap(t)
	case []any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			out = append(out, wafv2CloneAny(item))
		}
		return out
	default:
		return t
	}
}
