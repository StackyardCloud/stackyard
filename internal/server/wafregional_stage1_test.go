package server

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func wrCreate(t *testing.T, ts *httptest.Server, op, key, idKey, extra string) string {
	t.Helper()
	out := wrOK(t, ts, op, fmt.Sprintf(`{"Name":"n","ChangeToken":%q%s}`, wrToken(t, ts), extra))
	return out[key].(map[string]any)[idKey].(string)
}

func TestWAFRegionalStage1ChangeTokens(t *testing.T) {
	_, ts := newWAFRegionalTestServer(t)
	tok := wrToken(t, ts)
	if s := wrOK(t, ts, "GetChangeTokenStatus", fmt.Sprintf(`{"ChangeToken":%q}`, tok))["ChangeTokenStatus"]; s != "PROVISIONED" {
		t.Fatalf("new token: %v", s)
	}
	out := wrOK(t, ts, "CreateIPSet", fmt.Sprintf(`{"Name":"a","ChangeToken":%q}`, tok))
	if out["ChangeToken"] != tok {
		t.Fatalf("create echoes the token: %v", out)
	}
	// A used token can't be reused.
	if code := wrErrCode(t, ts, "CreateIPSet", fmt.Sprintf(`{"Name":"b","ChangeToken":%q}`, tok)); code != "WAFStaleDataException" {
		t.Fatalf("reused token: %s", code)
	}
	if code := wrErrCode(t, ts, "CreateIPSet", `{"Name":"b","ChangeToken":"made-up"}`); code != "WAFStaleDataException" {
		t.Fatalf("unknown token: %s", code)
	}
	for _, want := range []string{"PENDING", "INSYNC", "INSYNC"} {
		if s := wrOK(t, ts, "GetChangeTokenStatus", fmt.Sprintf(`{"ChangeToken":%q}`, tok))["ChangeTokenStatus"]; s != want {
			t.Fatalf("used token: %v, want %s", s, want)
		}
	}
	if code := wrErrCode(t, ts, "GetChangeTokenStatus", `{"ChangeToken":"made-up"}`); code != "WAFNonexistentItemException" {
		t.Fatalf("status of unknown token: %s", code)
	}
	// A failed change leaves the token usable.
	tok2 := wrToken(t, ts)
	if code := wrErrCode(t, ts, "DeleteIPSet", fmt.Sprintf(`{"IPSetId":"nope","ChangeToken":%q}`, tok2)); code != "WAFNonexistentItemException" {
		t.Fatalf("delete missing: %s", code)
	}
	wrOK(t, ts, "CreateIPSet", fmt.Sprintf(`{"Name":"c","ChangeToken":%q}`, tok2))
}

func TestWAFRegionalStage1ConditionsRulesAndWebACLs(t *testing.T) {
	_, ts := newWAFRegionalTestServer(t)
	ipset := wrCreate(t, ts, "CreateIPSet", "IPSet", "IPSetId", "")
	ins := func(op, idKey, id, updKey, item string) string {
		return fmt.Sprintf(`{%q:%q,"ChangeToken":%q,"Updates":[{"Action":"INSERT",%q:%s}]}`, idKey, id, wrToken(t, ts), updKey, item)
	}
	wrOK(t, ts, "UpdateIPSet", ins("UpdateIPSet", "IPSetId", ipset, "IPSetDescriptor", `{"Type":"IPV4","Value":"192.0.2.0/24"}`))
	if code := wrErrCode(t, ts, "UpdateIPSet", ins("UpdateIPSet", "IPSetId", ipset, "IPSetDescriptor", `{"Type":"IPV4","Value":"192.0.2.0/24"}`)); code != "WAFInvalidOperationException" {
		t.Fatalf("duplicate insert: %s", code)
	}
	for _, bad := range []string{`{"Type":"IPV4","Value":"10.0.0.0/12"}`, `{"Type":"IPV6","Value":"192.0.2.0/24"}`, `{"Type":"IPV4","Value":"nope"}`} {
		if code := wrErrCode(t, ts, "UpdateIPSet", ins("UpdateIPSet", "IPSetId", ipset, "IPSetDescriptor", bad)); code != "WAFInvalidParameterException" {
			t.Fatalf("bad range %s: %s", bad, code)
		}
	}
	got := wrOK(t, ts, "GetIPSet", fmt.Sprintf(`{"IPSetId":%q}`, ipset))["IPSet"].(map[string]any)
	if d := got["IPSetDescriptors"].([]any); len(d) != 1 {
		t.Fatalf("descriptors: %v", got)
	}

	rule := wrCreate(t, ts, "CreateRule", "Rule", "RuleId", `,"MetricName":"blockips"`)
	if code := wrErrCode(t, ts, "UpdateRule", ins("UpdateRule", "RuleId", rule, "Predicate", `{"Negated":false,"Type":"IPMatch","DataId":"missing"}`)); code != "WAFNonexistentItemException" {
		t.Fatalf("predicate on a missing set: %s", code)
	}
	wrOK(t, ts, "UpdateRule", ins("UpdateRule", "RuleId", rule, "Predicate", fmt.Sprintf(`{"Negated":false,"Type":"IPMatch","DataId":%q}`, ipset)))

	acl := wrCreate(t, ts, "CreateWebACL", "WebACL", "WebACLId", `,"MetricName":"acl","DefaultAction":{"Type":"ALLOW"}`)
	if code := wrErrCode(t, ts, "UpdateWebACL", ins("UpdateWebACL", "WebACLId", acl, "ActivatedRule", fmt.Sprintf(`{"Priority":1,"RuleId":%q,"OverrideAction":{"Type":"NONE"}}`, rule))); code != "WAFInvalidParameterException" {
		t.Fatalf("REGULAR rule with an OverrideAction: %s", code)
	}
	wrOK(t, ts, "UpdateWebACL", ins("UpdateWebACL", "WebACLId", acl, "ActivatedRule", fmt.Sprintf(`{"Priority":1,"RuleId":%q,"Action":{"Type":"BLOCK"}}`, rule)))

	// Referenced and non-empty entities can't be deleted.
	if code := wrErrCode(t, ts, "DeleteRule", fmt.Sprintf(`{"RuleId":%q,"ChangeToken":%q}`, rule, wrToken(t, ts))); code != "WAFNonEmptyEntityException" {
		t.Fatalf("delete non-empty rule: %s", code)
	}
	if code := wrErrCode(t, ts, "DeleteIPSet", fmt.Sprintf(`{"IPSetId":%q,"ChangeToken":%q}`, ipset, wrToken(t, ts))); code != "WAFNonEmptyEntityException" {
		t.Fatalf("delete non-empty ipset: %s", code)
	}
	empty := wrCreate(t, ts, "CreateIPSet", "IPSet", "IPSetId", "")
	wrOK(t, ts, "UpdateRule", ins("UpdateRule", "RuleId", rule, "Predicate", fmt.Sprintf(`{"Negated":true,"Type":"IPMatch","DataId":%q}`, empty)))
	if code := wrErrCode(t, ts, "DeleteIPSet", fmt.Sprintf(`{"IPSetId":%q,"ChangeToken":%q}`, empty, wrToken(t, ts))); code != "WAFReferencedItemException" {
		t.Fatalf("delete referenced ipset: %s", code)
	}

	// Association, then the web ACL is in use.
	alb := "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/web/50dc6c495c0c9188"
	if code := wrErrCode(t, ts, "AssociateWebACL", fmt.Sprintf(`{"WebACLId":%q,"ResourceArn":"arn:aws:s3:::bucket"}`, acl)); code != "WAFInvalidParameterException" {
		t.Fatalf("associate an S3 bucket: %s", code)
	}
	wrOK(t, ts, "AssociateWebACL", fmt.Sprintf(`{"WebACLId":%q,"ResourceArn":%q}`, acl, alb))
	if s := wrOK(t, ts, "GetWebACLForResource", fmt.Sprintf(`{"ResourceArn":%q}`, alb))["WebACLSummary"].(map[string]any); s["WebACLId"] != acl {
		t.Fatalf("web ACL for resource: %v", s)
	}
	if arns := wrOK(t, ts, "ListResourcesForWebACL", fmt.Sprintf(`{"WebACLId":%q}`, acl))["ResourceArns"].([]any); len(arns) != 1 {
		t.Fatalf("resources for web ACL: %v", arns)
	}

	// Tags on the web ACL, policies on rule groups only.
	arn := wrOK(t, ts, "GetWebACL", fmt.Sprintf(`{"WebACLId":%q}`, acl))["WebACL"].(map[string]any)["WebACLArn"].(string)
	wrOK(t, ts, "TagResource", fmt.Sprintf(`{"ResourceARN":%q,"Tags":[{"Key":"env","Value":"dev"}]}`, arn))
	if code := wrErrCode(t, ts, "TagResource", fmt.Sprintf(`{"ResourceARN":%q,"Tags":[{"Key":"aws:x","Value":"v"}]}`, arn)); code != "WAFInvalidParameterException" {
		t.Fatalf("aws: tag key: %s", code)
	}
	if tags := wrOK(t, ts, "ListTagsForResource", fmt.Sprintf(`{"ResourceARN":%q}`, arn))["TagInfoForResource"].(map[string]any)["TagList"].([]any); len(tags) != 1 {
		t.Fatalf("tags: %v", tags)
	}
	group := wrCreate(t, ts, "CreateRuleGroup", "RuleGroup", "RuleGroupId", `,"MetricName":"group"`)
	groupArn := "arn:aws:waf-regional:us-east-1:123456789012:rulegroup/" + group
	if code := wrErrCode(t, ts, "PutPermissionPolicy", fmt.Sprintf(`{"ResourceArn":%q,"Policy":%q}`, groupArn, `{"Statement":[{"Effect":"Allow","Principal":{"AWS":"111122223333"},"Action":"waf-regional:DeleteRuleGroup"}]}`)); code != "WAFInvalidPermissionPolicyException" {
		t.Fatalf("disallowed policy action: %s", code)
	}
	wrOK(t, ts, "PutPermissionPolicy", fmt.Sprintf(`{"ResourceArn":%q,"Policy":%q}`, groupArn, `{"Statement":[{"Effect":"Allow","Principal":{"AWS":"111122223333"},"Action":["waf-regional:UpdateWebACL","waf-regional:GetRuleGroup"]}]}`))

	// Logging needs an existing web ACL.
	wrOK(t, ts, "PutLoggingConfiguration", fmt.Sprintf(`{"LoggingConfiguration":{"ResourceArn":%q,"LogDestinationConfigs":["arn:aws:firehose:us-east-1:123456789012:deliverystream/aws-waf-logs-x"]}}`, arn))
	if n := len(wrOK(t, ts, "ListLoggingConfigurations", `{}`)["LoggingConfigurations"].([]any)); n != 1 {
		t.Fatalf("logging configurations: %d", n)
	}
	wrOK(t, ts, "DisassociateWebACL", fmt.Sprintf(`{"ResourceArn":%q}`, alb))
	if _, ok := wrOK(t, ts, "GetWebACLForResource", fmt.Sprintf(`{"ResourceArn":%q}`, alb))["WebACLSummary"]; ok {
		t.Fatal("web ACL still associated")
	}
}

func TestWAFRegionalStage1Pagination(t *testing.T) {
	_, ts := newWAFRegionalTestServer(t)
	for i := 0; i < 3; i++ {
		wrCreate(t, ts, "CreateXssMatchSet", "XssMatchSet", "XssMatchSetId", "")
	}
	page := wrOK(t, ts, "ListXssMatchSets", `{"Limit":2}`)
	if len(page["XssMatchSets"].([]any)) != 2 || page["NextMarker"] == nil {
		t.Fatalf("first page: %v", page)
	}
	rest := wrOK(t, ts, "ListXssMatchSets", fmt.Sprintf(`{"Limit":2,"NextMarker":%q}`, page["NextMarker"]))
	if len(rest["XssMatchSets"].([]any)) != 1 || rest["NextMarker"] != nil {
		t.Fatalf("second page: %v", rest)
	}
	if code := wrErrCode(t, ts, "ListXssMatchSets", `{"NextMarker":"x"}`); code != "WAFInvalidParameterException" {
		t.Fatalf("bad marker: %s", code)
	}
}

func TestWAFRegionalStage1MigrationStack(t *testing.T) {
	srv, ts := newWAFRegionalTestServer(t)
	acl := wrCreate(t, ts, "CreateWebACL", "WebACL", "WebACLId", `,"MetricName":"acl","DefaultAction":{"Type":"BLOCK"}`)
	if code := wrErrCode(t, ts, "CreateWebACLMigrationStack", fmt.Sprintf(`{"WebACLId":%q,"S3BucketName":"aws-waf-migration-none","IgnoreUnsupportedType":true}`, acl)); code != "WAFEntityMigrationException" {
		t.Fatalf("missing bucket: %s", code)
	}
	if _, err := srv.s3.CreateBucket("aws-waf-migration-test"); err != nil {
		t.Fatal(err)
	}
	out := wrOK(t, ts, "CreateWebACLMigrationStack", fmt.Sprintf(`{"WebACLId":%q,"S3BucketName":"aws-waf-migration-test","IgnoreUnsupportedType":true}`, acl))
	if url, _ := out["S3ObjectUrl"].(string); !strings.Contains(url, acl) {
		t.Fatalf("object url: %v", out)
	}
	obj, err := srv.s3.GetObject("aws-waf-migration-test", "AWSWAF-Migration/"+acl+".json")
	if err != nil || !strings.Contains(string(obj.Body), `"Block"`) {
		t.Fatalf("template: %v %s", err, obj.Body)
	}
}
