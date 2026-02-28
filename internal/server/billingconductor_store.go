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
	billingConductorDefaultRegion    = "us-east-1"
	billingConductorDefaultAccountID = "123456789012"
)

type billingConductorStore struct {
	mu sync.Mutex

	nextBillingGroupID   int64
	nextPricingPlanID    int64
	nextPricingRuleID    int64
	nextCustomLineItemID int64

	billingGroups   map[string]map[string]any
	pricingPlans    map[string]map[string]any
	pricingRules    map[string]map[string]any
	customLineItems map[string]map[string]any

	billingGroupAccounts   map[string][]string
	pricingPlanRules       map[string][]string
	pricingRulePlans       map[string][]string
	customLineItemResource map[string][]string

	tags map[string]map[string]string
}

func newBillingConductorStore() *billingConductorStore {
	now := time.Now().UTC().Format(time.RFC3339)
	s := &billingConductorStore{
		nextBillingGroupID:     2,
		nextPricingPlanID:      2,
		nextPricingRuleID:      2,
		nextCustomLineItemID:   2,
		billingGroups:          map[string]map[string]any{},
		pricingPlans:           map[string]map[string]any{},
		pricingRules:           map[string]map[string]any{},
		customLineItems:        map[string]map[string]any{},
		billingGroupAccounts:   map[string][]string{},
		pricingPlanRules:       map[string][]string{},
		pricingRulePlans:       map[string][]string{},
		customLineItemResource: map[string][]string{},
		tags:                   map[string]map[string]string{},
	}
	s.ensureBillingGroupLocked("bg-000001", now)
	s.ensurePricingPlanLocked("pp-000001", now)
	s.ensurePricingRuleLocked("pr-000001", now)
	s.ensureCustomLineItemLocked("cli-000001", now)
	return s
}

func (s *billingConductorStore) Handle(action string, payload map[string]any, pathParams map[string]string, query url.Values) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)

	billingGroupARN := billingConductorLookupString(pathParams, payload, query, []string{"billingGroupArn", "BillingGroupArn"}, billingConductorBillingGroupARN("bg-000001"))
	pricingPlanARN := billingConductorLookupString(pathParams, payload, query, []string{"pricingPlanArn", "PricingPlanArn"}, billingConductorPricingPlanARN("pp-000001"))
	pricingRuleARN := billingConductorLookupString(pathParams, payload, query, []string{"pricingRuleArn", "PricingRuleArn"}, billingConductorPricingRuleARN("pr-000001"))
	customLineItemARN := billingConductorLookupString(pathParams, payload, query, []string{"customLineItemArn", "CustomLineItemArn"}, billingConductorCustomLineItemARN("cli-000001"))
	resourceARN := billingConductorLookupString(pathParams, payload, query, []string{"resourceArn", "ResourceArn"}, billingConductorBillingGroupARN("bg-000001"))

	bg := s.ensureBillingGroupLocked(billingConductorIDFromARNOrID(billingGroupARN, "bg"), now)
	pp := s.ensurePricingPlanLocked(billingConductorIDFromARNOrID(pricingPlanARN, "pp"), now)
	pr := s.ensurePricingRuleLocked(billingConductorIDFromARNOrID(pricingRuleARN, "pr"), now)
	cli := s.ensureCustomLineItemLocked(billingConductorIDFromARNOrID(customLineItemARN, "cli"), now)

	s.applyTagMutationsLocked(action, payload, pathParams, query, resourceARN)

	s.applyAssociationMutationsLocked(action, payload, bg, pp, pr, cli)

	s.applyUpdateHintsLocked(payload, now, bg, pp, pr, cli)

	s.ensureTagMapLocked(billingConductorAnyString(bg["Arn"]))
	s.ensureTagMapLocked(billingConductorAnyString(pp["Arn"]))
	s.ensureTagMapLocked(billingConductorAnyString(pr["Arn"]))
	s.ensureTagMapLocked(billingConductorAnyString(cli["Arn"]))

	switch action {
	case "CreateBillingGroup":
		created := s.ensureBillingGroupLocked(billingConductorPayloadID(payload, "billingGroupName", "bg", &s.nextBillingGroupID), now)
		return map[string]any{"Arn": created["Arn"], "BillingGroupArn": created["Arn"]}
	case "CreatePricingPlan":
		created := s.ensurePricingPlanLocked(billingConductorPayloadID(payload, "name", "pp", &s.nextPricingPlanID), now)
		return map[string]any{"Arn": created["Arn"], "PricingPlanArn": created["Arn"]}
	case "CreatePricingRule":
		created := s.ensurePricingRuleLocked(billingConductorPayloadID(payload, "name", "pr", &s.nextPricingRuleID), now)
		return map[string]any{"Arn": created["Arn"], "PricingRuleArn": created["Arn"]}
	case "CreateCustomLineItem":
		created := s.ensureCustomLineItemLocked(billingConductorPayloadID(payload, "name", "cli", &s.nextCustomLineItemID), now)
		return map[string]any{"Arn": created["Arn"], "CustomLineItemArn": created["Arn"]}

	case "UpdateBillingGroup":
		return map[string]any{"Arn": bg["Arn"]}
	case "UpdatePricingPlan":
		return map[string]any{"Arn": pp["Arn"]}
	case "UpdatePricingRule":
		return map[string]any{"Arn": pr["Arn"]}
	case "UpdateCustomLineItem":
		return map[string]any{"Arn": cli["Arn"]}

	case "DeleteBillingGroup":
		delete(s.billingGroups, billingConductorAnyString(bg["BillingGroupId"]))
		delete(s.billingGroupAccounts, billingConductorAnyString(bg["Arn"]))
		delete(s.tags, billingConductorAnyString(bg["Arn"]))
		return map[string]any{}
	case "DeletePricingPlan":
		delete(s.pricingPlans, billingConductorAnyString(pp["PricingPlanId"]))
		delete(s.pricingPlanRules, billingConductorAnyString(pp["Arn"]))
		delete(s.tags, billingConductorAnyString(pp["Arn"]))
		return map[string]any{}
	case "DeletePricingRule":
		delete(s.pricingRules, billingConductorAnyString(pr["PricingRuleId"]))
		delete(s.pricingRulePlans, billingConductorAnyString(pr["Arn"]))
		delete(s.tags, billingConductorAnyString(pr["Arn"]))
		return map[string]any{}
	case "DeleteCustomLineItem":
		delete(s.customLineItems, billingConductorAnyString(cli["CustomLineItemId"]))
		delete(s.customLineItemResource, billingConductorAnyString(cli["Arn"]))
		delete(s.tags, billingConductorAnyString(cli["Arn"]))
		return map[string]any{}

	case "ListBillingGroups":
		return map[string]any{"BillingGroups": billingConductorSortedSummaries(s.billingGroups), "NextToken": ""}
	case "ListPricingPlans":
		return map[string]any{"PricingPlans": billingConductorSortedSummaries(s.pricingPlans), "NextToken": ""}
	case "ListPricingRules":
		return map[string]any{"PricingRules": billingConductorSortedSummaries(s.pricingRules), "NextToken": ""}
	case "ListCustomLineItems":
		return map[string]any{"CustomLineItems": billingConductorSortedSummaries(s.customLineItems), "NextToken": ""}
	case "ListCustomLineItemVersions":
		versions := make([]any, 0, len(s.customLineItems))
		for _, item := range billingConductorSortedSummaries(s.customLineItems) {
			m, _ := item.(map[string]any)
			versions = append(versions, map[string]any{
				"Arn":                m["Arn"],
				"Name":               m["Name"],
				"StartBillingPeriod": "2026-01",
				"EndBillingPeriod":   "2026-12",
			})
		}
		return map[string]any{"CustomLineItemVersions": versions, "NextToken": ""}
	case "ListAccountAssociations":
		accounts := make([]any, 0)
		for _, accountID := range s.billingGroupAccounts[billingConductorAnyString(bg["Arn"])] {
			accounts = append(accounts, map[string]any{"AccountId": accountID, "BillingGroupArn": bg["Arn"]})
		}
		return map[string]any{"LinkedAccounts": accounts, "NextToken": ""}
	case "ListPricingRulesAssociatedToPricingPlan":
		rules := make([]any, 0)
		for _, arn := range s.pricingPlanRules[billingConductorAnyString(pp["Arn"])] {
			rules = append(rules, map[string]any{"Arn": arn})
		}
		return map[string]any{"PricingRules": rules, "NextToken": ""}
	case "ListPricingPlansAssociatedWithPricingRule":
		plans := make([]any, 0)
		for _, arn := range s.pricingRulePlans[billingConductorAnyString(pr["Arn"])] {
			plans = append(plans, map[string]any{"Arn": arn})
		}
		return map[string]any{"PricingPlans": plans, "NextToken": ""}
	case "ListResourcesAssociatedToCustomLineItem":
		resources := make([]any, 0)
		for _, arn := range s.customLineItemResource[billingConductorAnyString(cli["Arn"])] {
			resources = append(resources, map[string]any{"Arn": arn})
		}
		return map[string]any{"AssociatedResources": resources, "NextToken": ""}
	case "GetBillingGroupCostReport", "ListBillingGroupCostReports":
		report := map[string]any{
			"Arn":             bg["Arn"],
			"BillingGroupArn": bg["Arn"],
			"BillingPeriod":   "2026-01",
			"Currency":        "USD",
			"EstimatedAmount": "123.45",
		}
		if action == "GetBillingGroupCostReport" {
			return map[string]any{"BillingGroupCostReportResults": []any{report}, "NextToken": ""}
		}
		return map[string]any{"BillingGroupCostReports": []any{report}, "NextToken": ""}
	case "BatchAssociateResourcesToCustomLineItem":
		return map[string]any{"FailedAssociatedResources": []any{}}
	case "BatchDisassociateResourcesFromCustomLineItem":
		return map[string]any{"FailedDisassociatedResources": []any{}}
	case "AssociateAccounts", "DisassociateAccounts", "AssociatePricingRules", "DisassociatePricingRules":
		return map[string]any{"Arn": bg["Arn"], "PricingPlanArn": pp["Arn"]}
	case "TagResource", "UntagResource":
		return map[string]any{}
	case "ListTagsForResource":
		tags := map[string]string{}
		for k, v := range s.ensureTagMapLocked(resourceARN) {
			tags[k] = v
		}
		return map[string]any{"Tags": tags}
	default:
		return map[string]any{"Operation": action, "Status": "OK"}
	}
}

func (s *billingConductorStore) applyAssociationMutationsLocked(action string, payload map[string]any, bg, pp, pr, cli map[string]any) {
	switch action {
	case "AssociateAccounts", "DisassociateAccounts":
		accountIDs := billingConductorStringSlice(payload, "AccountIds", "accountIds")
		if len(accountIDs) == 0 {
			accountIDs = []string{"111122223333"}
		}
		key := billingConductorAnyString(bg["Arn"])
		if action == "AssociateAccounts" {
			s.billingGroupAccounts[key] = billingConductorUnionStrings(s.billingGroupAccounts[key], accountIDs)
		} else {
			s.billingGroupAccounts[key] = billingConductorWithoutStrings(s.billingGroupAccounts[key], accountIDs)
		}
	case "AssociatePricingRules", "DisassociatePricingRules":
		ruleARNs := billingConductorStringSlice(payload, "PricingRuleArns", "pricingRuleArns")
		if len(ruleARNs) == 0 {
			ruleARNs = []string{billingConductorAnyString(pr["Arn"])}
		}
		planKey := billingConductorAnyString(pp["Arn"])
		for _, ruleARN := range ruleARNs {
			ruleKey := ruleARN
			if action == "AssociatePricingRules" {
				s.pricingPlanRules[planKey] = billingConductorUnionStrings(s.pricingPlanRules[planKey], []string{ruleARN})
				s.pricingRulePlans[ruleKey] = billingConductorUnionStrings(s.pricingRulePlans[ruleKey], []string{planKey})
			} else {
				s.pricingPlanRules[planKey] = billingConductorWithoutStrings(s.pricingPlanRules[planKey], []string{ruleARN})
				s.pricingRulePlans[ruleKey] = billingConductorWithoutStrings(s.pricingRulePlans[ruleKey], []string{planKey})
			}
		}
	case "BatchAssociateResourcesToCustomLineItem", "BatchDisassociateResourcesFromCustomLineItem":
		resources := billingConductorStringSlice(payload, "ResourceArns", "resourceArns")
		if len(resources) == 0 {
			resources = []string{"arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0"}
		}
		cliKey := billingConductorAnyString(cli["Arn"])
		if action == "BatchAssociateResourcesToCustomLineItem" {
			s.customLineItemResource[cliKey] = billingConductorUnionStrings(s.customLineItemResource[cliKey], resources)
		} else {
			s.customLineItemResource[cliKey] = billingConductorWithoutStrings(s.customLineItemResource[cliKey], resources)
		}
	}
}

func (s *billingConductorStore) applyUpdateHintsLocked(payload map[string]any, now string, resources ...map[string]any) {
	for _, resource := range resources {
		if resource == nil {
			continue
		}
		if name := billingConductorLookupString(nil, payload, nil, []string{"name", "Name"}, ""); name != "" {
			resource["Name"] = name
		}
		resource["LastModifiedTime"] = now
	}
}

func (s *billingConductorStore) applyTagMutationsLocked(action string, payload map[string]any, pathParams map[string]string, query url.Values, defaultARN string) {
	resourceARN := billingConductorLookupString(pathParams, payload, query, []string{"resourceArn", "ResourceArn"}, defaultARN)
	if resourceARN == "" {
		resourceARN = defaultARN
	}

	switch action {
	case "TagResource":
		tagMap := s.ensureTagMapLocked(resourceARN)
		for k, v := range billingConductorExtractTags(payload) {
			tagMap[k] = v
		}
	case "UntagResource":
		tagMap := s.ensureTagMapLocked(resourceARN)
		for _, key := range billingConductorStringSlice(payload, "TagKeys", "tagKeys") {
			delete(tagMap, key)
		}
		for _, key := range query["tagKeys"] {
			delete(tagMap, strings.TrimSpace(key))
		}
	case "ListTagsForResource":
		s.ensureTagMapLocked(resourceARN)
	}
}

func (s *billingConductorStore) ensureBillingGroupLocked(id, now string) map[string]any {
	if id == "" {
		id = "bg-000001"
	}
	if existing := s.billingGroups[id]; existing != nil {
		return existing
	}
	arn := billingConductorBillingGroupARN(id)
	item := map[string]any{
		"BillingGroupId":   id,
		"Arn":              arn,
		"Name":             id,
		"PrimaryAccountId": "123456789012",
		"CreationTime":     now,
		"LastModifiedTime": now,
	}
	s.billingGroups[id] = item
	return item
}

func (s *billingConductorStore) ensurePricingPlanLocked(id, now string) map[string]any {
	if id == "" {
		id = "pp-000001"
	}
	if existing := s.pricingPlans[id]; existing != nil {
		return existing
	}
	arn := billingConductorPricingPlanARN(id)
	item := map[string]any{
		"PricingPlanId":    id,
		"Arn":              arn,
		"Name":             id,
		"CreationTime":     now,
		"LastModifiedTime": now,
	}
	s.pricingPlans[id] = item
	return item
}

func (s *billingConductorStore) ensurePricingRuleLocked(id, now string) map[string]any {
	if id == "" {
		id = "pr-000001"
	}
	if existing := s.pricingRules[id]; existing != nil {
		return existing
	}
	arn := billingConductorPricingRuleARN(id)
	item := map[string]any{
		"PricingRuleId":    id,
		"Arn":              arn,
		"Name":             id,
		"Type":             "MARKUP",
		"CreationTime":     now,
		"LastModifiedTime": now,
	}
	s.pricingRules[id] = item
	return item
}

func (s *billingConductorStore) ensureCustomLineItemLocked(id, now string) map[string]any {
	if id == "" {
		id = "cli-000001"
	}
	if existing := s.customLineItems[id]; existing != nil {
		return existing
	}
	arn := billingConductorCustomLineItemARN(id)
	item := map[string]any{
		"CustomLineItemId": id,
		"Arn":              arn,
		"Name":             id,
		"CreationTime":     now,
		"LastModifiedTime": now,
	}
	s.customLineItems[id] = item
	return item
}

func (s *billingConductorStore) ensureTagMapLocked(resourceARN string) map[string]string {
	resourceARN = strings.TrimSpace(resourceARN)
	if resourceARN == "" {
		resourceARN = billingConductorBillingGroupARN("bg-000001")
	}
	if existing := s.tags[resourceARN]; existing != nil {
		return existing
	}
	m := map[string]string{}
	s.tags[resourceARN] = m
	return m
}

func billingConductorSortedSummaries(src map[string]map[string]any) []any {
	keys := make([]string, 0, len(src))
	for key := range src {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, key := range keys {
		clone := map[string]any{}
		for k, v := range src[key] {
			clone[k] = v
		}
		out = append(out, clone)
	}
	return out
}

func billingConductorLookupString(pathParams map[string]string, payload map[string]any, query url.Values, keys []string, def string) string {
	for _, key := range keys {
		if pathParams != nil {
			if value := strings.TrimSpace(pathParams[key]); value != "" {
				return value
			}
		}
		if payload != nil {
			if raw, ok := payload[key]; ok {
				if value := strings.TrimSpace(billingConductorAnyString(raw)); value != "" {
					return value
				}
			}
		}
		if query != nil {
			if value := strings.TrimSpace(query.Get(key)); value != "" {
				return value
			}
		}
	}
	return def
}

func billingConductorPayloadID(payload map[string]any, preferredKey, prefix string, next *int64) string {
	if payload != nil {
		if raw, ok := payload[preferredKey]; ok {
			if value := strings.TrimSpace(billingConductorAnyString(raw)); value != "" {
				return value
			}
		}
	}
	id := fmt.Sprintf("%s-%06d", prefix, *next)
	*next = *next + 1
	return id
}

func billingConductorIDFromARNOrID(value, prefix string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Sprintf("%s-000001", prefix)
	}
	if strings.HasPrefix(value, "arn:") {
		parts := strings.Split(strings.Trim(value, "/"), "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return value
}

func billingConductorBillingGroupARN(id string) string {
	return fmt.Sprintf("arn:aws:billingconductor:%s:%s:billinggroup/%s", billingConductorDefaultRegion, billingConductorDefaultAccountID, id)
}

func billingConductorPricingPlanARN(id string) string {
	return fmt.Sprintf("arn:aws:billingconductor:%s:%s:pricingplan/%s", billingConductorDefaultRegion, billingConductorDefaultAccountID, id)
}

func billingConductorPricingRuleARN(id string) string {
	return fmt.Sprintf("arn:aws:billingconductor:%s:%s:pricingrule/%s", billingConductorDefaultRegion, billingConductorDefaultAccountID, id)
}

func billingConductorCustomLineItemARN(id string) string {
	return fmt.Sprintf("arn:aws:billingconductor:%s:%s:customlineitem/%s", billingConductorDefaultRegion, billingConductorDefaultAccountID, id)
}

func billingConductorStringSlice(payload map[string]any, keys ...string) []string {
	for _, key := range keys {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		switch typed := raw.(type) {
		case []string:
			out := make([]string, 0, len(typed))
			for _, value := range typed {
				value = strings.TrimSpace(value)
				if value != "" {
					out = append(out, value)
				}
			}
			if len(out) > 0 {
				return out
			}
		case []any:
			out := make([]string, 0, len(typed))
			for _, value := range typed {
				v := strings.TrimSpace(billingConductorAnyString(value))
				if v != "" {
					out = append(out, v)
				}
			}
			if len(out) > 0 {
				return out
			}
		case string:
			value := strings.TrimSpace(typed)
			if value != "" {
				return []string{value}
			}
		}
	}
	return nil
}

func billingConductorExtractTags(payload map[string]any) map[string]string {
	out := map[string]string{}
	for _, key := range []string{"Tags", "tags"} {
		raw, ok := payload[key]
		if !ok || raw == nil {
			continue
		}
		switch typed := raw.(type) {
		case map[string]string:
			for k, v := range typed {
				k = strings.TrimSpace(k)
				if k != "" {
					out[k] = strings.TrimSpace(v)
				}
			}
		case map[string]any:
			for k, v := range typed {
				k = strings.TrimSpace(k)
				if k != "" {
					out[k] = strings.TrimSpace(billingConductorAnyString(v))
				}
			}
		}
	}
	return out
}

func billingConductorUnionStrings(base, adds []string) []string {
	set := map[string]struct{}{}
	for _, value := range base {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	for _, value := range adds {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func billingConductorWithoutStrings(base, removes []string) []string {
	removeSet := map[string]struct{}{}
	for _, value := range removes {
		value = strings.TrimSpace(value)
		if value != "" {
			removeSet[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(base))
	for _, value := range base {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, remove := removeSet[value]; !remove {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func billingConductorAnyString(v any) string {
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
