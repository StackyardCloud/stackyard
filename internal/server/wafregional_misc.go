package server

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ponytail: associated resources are checked by ARN pattern only; the ELBv2
// and API Gateway stores don't key resources by these ARNs.
var (
	wrALBArnRe   = regexp.MustCompile(`^arn:aws[a-z-]*:elasticloadbalancing:[a-z0-9-]+:\d{12}:loadbalancer/app/[^/]+/[^/]+$`)
	wrStageArnRe = regexp.MustCompile(`^arn:aws[a-z-]*:apigateway:[a-z0-9-]+::/restapis/[^/]+/stages/[^/]+$`)
	wrWAFArnRe   = regexp.MustCompile(`^arn:aws:waf-regional:[a-z0-9-]+:\d{12}:([a-z]+)/(.+)$`)
)

func wrResourceType(arn string) string {
	switch {
	case wrALBArnRe.MatchString(arn):
		return "APPLICATION_LOAD_BALANCER"
	case wrStageArnRe.MatchString(arn):
		return "API_GATEWAY"
	}
	return ""
}

func wrBadResource(arn string) *wrErr {
	return wrParam("RESOURCE_ARN", arn, "ILLEGAL_ARGUMENT", "%s is not an Application Load Balancer or API Gateway stage ARN.", arn)
}

// entity resolves a WAF Regional ARN of one of the given types.
func (st *wafRegionalStore) entity(arn string, types ...string) (*wrRes, *wrErr) {
	m := wrWAFArnRe.FindStringSubmatch(arn)
	if m == nil || !strings.Contains(" "+strings.Join(types, " ")+" ", " "+m[1]+" ") {
		return nil, wrParam("RESOURCE_ARN", arn, "ILLEGAL_ARGUMENT", "%s is not a supported WAF Regional resource ARN.", arn)
	}
	k := wrKindByARNType[m[1]]
	r := st.res[k.name][m[2]]
	if r == nil {
		return nil, wrMissing(k.name, m[2])
	}
	return r, nil
}

// ---- web ACL associations ----

func (st *wafRegionalStore) associateWebACL(in map[string]any) (map[string]any, *wrErr) {
	arn := wrStr(in["ResourceArn"])
	if wrResourceType(arn) == "" {
		return nil, wrBadResource(arn)
	}
	acl, err := st.find(wrKindByName["WebACL"], wrStr(in["WebACLId"]))
	if err != nil {
		return nil, err
	}
	if _, ok := st.assoc[arn]; !ok && len(st.assoc) >= wrMaxAssocs {
		return nil, wrError("WAFLimitsExceededException", "At most %d resources can be associated.", wrMaxAssocs)
	}
	st.assoc[arn] = acl.id
	return nil, nil
}

func (st *wafRegionalStore) disassociateWebACL(in map[string]any) (map[string]any, *wrErr) {
	arn := wrStr(in["ResourceArn"])
	if wrResourceType(arn) == "" {
		return nil, wrBadResource(arn)
	}
	delete(st.assoc, arn)
	return nil, nil
}

func (st *wafRegionalStore) getWebACLForResource(in map[string]any) (map[string]any, *wrErr) {
	arn := wrStr(in["ResourceArn"])
	if wrResourceType(arn) == "" {
		return nil, wrBadResource(arn)
	}
	acl := st.res["WebACL"][st.assoc[arn]]
	if acl == nil {
		return map[string]any{}, nil
	}
	return map[string]any{"WebACLSummary": map[string]any{"WebACLId": acl.id, "Name": acl.name}}, nil
}

func (st *wafRegionalStore) listResourcesForWebACL(in map[string]any) (map[string]any, *wrErr) {
	acl, err := st.find(wrKindByName["WebACL"], wrStr(in["WebACLId"]))
	if err != nil {
		return nil, err
	}
	want := wrStr(in["ResourceType"])
	if want == "" {
		want = "APPLICATION_LOAD_BALANCER"
	}
	arns := []any{}
	for arn, id := range st.assoc {
		if id == acl.id && wrResourceType(arn) == want {
			arns = append(arns, arn)
		}
	}
	sort.Slice(arns, func(i, j int) bool { return arns[i].(string) < arns[j].(string) })
	return map[string]any{"ResourceArns": arns}, nil
}

// ---- logging ----

func (st *wafRegionalStore) putLoggingConfiguration(in map[string]any) (map[string]any, *wrErr) {
	cfg, _ := in["LoggingConfiguration"].(map[string]any)
	acl, err := st.entity(wrStr(cfg["ResourceArn"]), "webacl")
	if err != nil {
		return nil, err
	}
	st.logging[acl.arn()] = cfg
	return map[string]any{"LoggingConfiguration": cfg}, nil
}

func (st *wafRegionalStore) getLoggingConfiguration(in map[string]any) (map[string]any, *wrErr) {
	cfg := st.logging[wrStr(in["ResourceArn"])]
	if cfg == nil {
		return nil, wrMissing("logging configuration for", wrStr(in["ResourceArn"]))
	}
	return map[string]any{"LoggingConfiguration": cfg}, nil
}

func (st *wafRegionalStore) deleteLoggingConfiguration(in map[string]any) (map[string]any, *wrErr) {
	arn := wrStr(in["ResourceArn"])
	if st.logging[arn] == nil {
		return nil, wrMissing("logging configuration for", arn)
	}
	delete(st.logging, arn)
	return nil, nil
}

func (st *wafRegionalStore) listLoggingConfigurations(in map[string]any) (map[string]any, *wrErr) {
	arns := make([]string, 0, len(st.logging))
	for arn := range st.logging {
		arns = append(arns, arn)
	}
	sort.Strings(arns)
	var cfgs []any
	for _, arn := range arns {
		cfgs = append(cfgs, st.logging[arn])
	}
	page, next, err := wrPage(cfgs, in)
	if err != nil {
		return nil, err
	}
	return wrPaged(map[string]any{"LoggingConfigurations": page}, next), nil
}

// ---- rule group permission policies ----

var wrPolicyActions = map[string]bool{
	"waf-regional:UpdateWebACL": true, "waf-regional:GetRuleGroup": true, "waf:UpdateWebACL": true, "waf:GetRuleGroup": true,
}

func (st *wafRegionalStore) putPermissionPolicy(in map[string]any) (map[string]any, *wrErr) {
	group, err := st.entity(wrStr(in["ResourceArn"]), "rulegroup")
	if err != nil {
		return nil, err
	}
	policy := wrStr(in["Policy"])
	if err := wrCheckPolicy(policy); err != nil {
		return nil, err
	}
	st.policies[group.arn()] = policy
	return nil, nil
}

// wrCheckPolicy applies WAF's rule-group policy rules: Allow statements only,
// with a Principal, and actions limited to UpdateWebACL and GetRuleGroup.
func wrCheckPolicy(policy string) *wrErr {
	bad := func(why string) *wrErr {
		return wrError("WAFInvalidPermissionPolicyException", "The policy is invalid: %s.", why)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(policy), &doc); err != nil {
		return bad("it isn't a JSON object")
	}
	stmts, _ := doc["Statement"].([]any)
	if s, ok := doc["Statement"].(map[string]any); ok {
		stmts = []any{s}
	}
	if len(stmts) == 0 {
		return bad("it has no Statement")
	}
	for _, raw := range stmts {
		s, _ := raw.(map[string]any)
		if wrStr(s["Effect"]) != "Allow" || s["Principal"] == nil {
			return bad("every statement must Allow a Principal")
		}
		for _, k := range []string{"NotAction", "NotPrincipal", "NotResource", "Condition"} {
			if s[k] != nil {
				return bad(k + " isn't supported")
			}
		}
		actions, _ := s["Action"].([]any)
		if a, ok := s["Action"].(string); ok {
			actions = []any{a}
		}
		if len(actions) == 0 {
			return bad("every statement needs an Action")
		}
		for _, a := range actions {
			if !wrPolicyActions[wrStr(a)] {
				return bad(fmt.Sprintf("action %v isn't allowed", a))
			}
		}
	}
	return nil
}

func (st *wafRegionalStore) getPermissionPolicy(in map[string]any) (map[string]any, *wrErr) {
	policy, ok := st.policies[wrStr(in["ResourceArn"])]
	if !ok {
		return nil, wrMissing("permission policy for", wrStr(in["ResourceArn"]))
	}
	return map[string]any{"Policy": policy}, nil
}

func (st *wafRegionalStore) deletePermissionPolicy(in map[string]any) (map[string]any, *wrErr) {
	arn := wrStr(in["ResourceArn"])
	if _, ok := st.policies[arn]; !ok {
		return nil, wrMissing("permission policy for", arn)
	}
	delete(st.policies, arn)
	return nil, nil
}

// ---- tags (web ACLs, rules, rate-based rules and rule groups) ----

var wrTaggable = []string{"webacl", "rule", "ratebasedrule", "rulegroup"}

func (st *wafRegionalStore) tagResource(in map[string]any) (map[string]any, *wrErr) {
	r, err := st.entity(wrStr(in["ResourceARN"]), wrTaggable...)
	if err != nil {
		return nil, err
	}
	tags, err := wrApplyTags(st.tags[r.arn()], in["Tags"])
	if err != nil {
		return nil, err
	}
	st.tags[r.arn()] = tags
	return nil, nil
}

func (st *wafRegionalStore) untagResource(in map[string]any) (map[string]any, *wrErr) {
	r, err := st.entity(wrStr(in["ResourceARN"]), wrTaggable...)
	if err != nil {
		return nil, err
	}
	keys, _ := in["TagKeys"].([]any)
	for _, k := range keys {
		delete(st.tags[r.arn()], wrStr(k))
	}
	return nil, nil
}

func (st *wafRegionalStore) listTagsForResource(in map[string]any) (map[string]any, *wrErr) {
	r, err := st.entity(wrStr(in["ResourceARN"]), wrTaggable...)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(st.tags[r.arn()]))
	for k := range st.tags[r.arn()] {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var tags []any
	for _, k := range keys {
		tags = append(tags, map[string]any{"Key": k, "Value": st.tags[r.arn()][k]})
	}
	page, next, err := wrPage(tags, in)
	if err != nil {
		return nil, err
	}
	return wrPaged(map[string]any{"TagInfoForResource": map[string]any{"ResourceARN": r.arn(), "TagList": page}}, next), nil
}

// ---- traffic views (no traffic flows through Stackyard's WAF) ----

func (st *wafRegionalStore) getSampledRequests(in map[string]any) (map[string]any, *wrErr) {
	id := wrStr(in["WebAclId"])
	owner := st.res["WebACL"][id]
	if owner == nil {
		owner = st.res["RuleGroup"][id]
	}
	if owner == nil {
		return nil, wrMissing("WebACL or RuleGroup", id)
	}
	if rule := wrStr(in["RuleId"]); rule != "Default_Action" && wrIndex(owner.kind, owner.items, map[string]any{"RuleId": rule}) < 0 {
		return nil, wrMissing("rule in "+owner.kind.name, rule)
	}
	return map[string]any{"SampledRequests": []any{}, "PopulationSize": 0, "TimeWindow": in["TimeWindow"]}, nil
}

func (st *wafRegionalStore) getRateBasedRuleManagedKeys(in map[string]any) (map[string]any, *wrErr) {
	if _, err := st.find(wrKindByName["RateBasedRule"], wrStr(in["RuleId"])); err != nil {
		return nil, err
	}
	page, next, err := wrPage(nil, in)
	if err != nil {
		return nil, err
	}
	return wrPaged(map[string]any{"ManagedKeys": page}, next), nil
}

// ListSubscribedRuleGroups lists AWS Marketplace rule groups, which Stackyard
// doesn't sell.
func (st *wafRegionalStore) listSubscribedRuleGroups(in map[string]any) (map[string]any, *wrErr) {
	page, next, err := wrPage(nil, in)
	if err != nil {
		return nil, err
	}
	return wrPaged(map[string]any{"RuleGroups": page}, next), nil
}

func (st *wafRegionalStore) listActivatedRulesInRuleGroup(in map[string]any) (map[string]any, *wrErr) {
	var rules []any
	if id := wrStr(in["RuleGroupId"]); id != "" {
		group, err := st.find(wrKindByName["RuleGroup"], id)
		if err != nil {
			return nil, err
		}
		rules = group.items
	}
	page, next, err := wrPage(rules, in)
	if err != nil {
		return nil, err
	}
	return wrPaged(map[string]any{"ActivatedRules": page}, next), nil
}

// ---- migration to WAFv2 ----

// wrMigrationStack writes a CloudFormation template for an AWS::WAFv2::WebACL
// to the S3 store. The WAF store is read under its lock; S3 is written after.
// ponytail: rules aren't translated; IgnoreUnsupportedType drops them.
func (s *Server) wrMigrationStack(in map[string]any) (any, *wrErr) {
	st := s.wafRegional
	st.mu.Lock()
	acl := st.res["WebACL"][wrStr(in["WebACLId"])]
	var name, metric, action string
	rules := 0
	if acl != nil {
		name, metric, rules = acl.name, acl.metric, len(acl.items)
		da, _ := acl.extra["DefaultAction"].(map[string]any)
		action = wrStr(da["Type"])
	}
	st.mu.Unlock()
	if acl == nil {
		return nil, wrMissing("WebACL", wrStr(in["WebACLId"]))
	}
	migrationErr := func(typ, reason string) *wrErr {
		e := wrError("WAFEntityMigrationException", "%s", reason)
		e.extra = map[string]any{"MigrationErrorType": typ, "MigrationErrorReason": reason}
		return e
	}
	if ignore, _ := in["IgnoreUnsupportedType"].(bool); rules > 0 && !ignore {
		return nil, migrationErr("ENTITY_NOT_SUPPORTED", "Stackyard doesn't convert web ACL rules; set IgnoreUnsupportedType to migrate without them.")
	}
	bucket := wrStr(in["S3BucketName"])
	if _, err := s.s3.GetBucket(bucket); err != nil {
		return nil, migrationErr("S3_BUCKET_NOT_FOUND", "The S3 bucket "+bucket+" doesn't exist.")
	}
	defaultAction := map[string]any{"Allow": map[string]any{}}
	if action == "BLOCK" {
		defaultAction = map[string]any{"Block": map[string]any{}}
	}
	template, _ := json.Marshal(map[string]any{
		"AWSTemplateFormatVersion": "2010-09-09",
		"Resources": map[string]any{"WebACL": map[string]any{"Type": "AWS::WAFv2::WebACL", "Properties": map[string]any{
			"Name": name, "Scope": "REGIONAL", "DefaultAction": defaultAction, "Rules": []any{},
			"VisibilityConfig": map[string]any{"MetricName": metric, "CloudWatchMetricsEnabled": true, "SampledRequestsEnabled": true},
		}}},
	})
	key := "AWSWAF-Migration/" + wrStr(in["WebACLId"]) + ".json"
	if _, err := s.s3.PutObject(bucket, key, "application/json", template, nil, "", ""); err != nil {
		return nil, migrationErr("S3_INTERNAL_ERROR", "Writing the template failed.")
	}
	return map[string]any{"S3ObjectUrl": fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucket, key)}, nil
}
