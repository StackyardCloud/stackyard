package server

import (
	"strconv"
	"testing"
)

func TestRoute53ResolverStage2DomainLists(t *testing.T) {
	srv, ts := newRoute53ResolverTestServer(t)
	dl := r53rOK(t, ts, "CreateFirewallDomainList", `{"CreatorRequestId":"d1","Name":"blocked"}`)["FirewallDomainList"].(map[string]any)
	id := dl["Id"].(string)
	again := r53rOK(t, ts, "CreateFirewallDomainList", `{"CreatorRequestId":"d1","Name":"blocked"}`)["FirewallDomainList"].(map[string]any)
	if again["Id"] != id {
		t.Fatalf("CreatorRequestId should make create idempotent: %v", again)
	}
	up := r53rOK(t, ts, "UpdateFirewallDomains", `{"FirewallDomainListId":"`+id+`","Operation":"ADD","Domains":["bad.example","*.evil.example","BAD.example"]}`)
	if up["Status"] != "UPDATING" {
		t.Fatalf("update: %v", up)
	}
	got := r53rOK(t, ts, "GetFirewallDomainList", `{"FirewallDomainListId":"`+id+`"}`)["FirewallDomainList"].(map[string]any)
	if got["DomainCount"] != float64(2) || got["Status"] != "COMPLETE" {
		t.Fatalf("after add: %v", got)
	}
	r53rOK(t, ts, "UpdateFirewallDomains", `{"FirewallDomainListId":"`+id+`","Operation":"REMOVE","Domains":["bad.example"]}`)
	domains := r53rOK(t, ts, "ListFirewallDomains", `{"FirewallDomainListId":"`+id+`"}`)["Domains"].([]any)
	if len(domains) != 1 || domains[0] != "*.evil.example" {
		t.Fatalf("after remove: %v", domains)
	}
	for _, bad := range []string{`a..b`, `foo.*.com`, `sp ace.com`} {
		if code := r53rFail(t, ts, "UpdateFirewallDomains", `{"FirewallDomainListId":"`+id+`","Operation":"ADD","Domains":["`+bad+`"]}`); code != "ValidationException" {
			t.Fatalf("%s: %s", bad, code)
		}
	}

	if _, err := srv.s3.CreateBucket("lists"); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.s3.PutObject("lists", "good.txt", "text/plain", []byte("# comment\none.example\ntwo.example\n\n"), nil, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.s3.PutObject("lists", "bad.txt", "text/plain", []byte("ok.example\nnot valid!\n"), nil, "", ""); err != nil {
		t.Fatal(err)
	}
	imp := r53rOK(t, ts, "ImportFirewallDomains", `{"FirewallDomainListId":"`+id+`","Operation":"REPLACE","DomainFileUrl":"s3://lists/good.txt"}`)
	if imp["Status"] != "IMPORTING" {
		t.Fatalf("import: %v", imp)
	}
	if got := r53rOK(t, ts, "GetFirewallDomainList", `{"FirewallDomainListId":"`+id+`"}`)["FirewallDomainList"].(map[string]any); got["DomainCount"] != float64(2) || got["Status"] != "COMPLETE" {
		t.Fatalf("after import: %v", got)
	}
	r53rOK(t, ts, "ImportFirewallDomains", `{"FirewallDomainListId":"`+id+`","Operation":"REPLACE","DomainFileUrl":"s3://lists/bad.txt"}`)
	if got := r53rOK(t, ts, "GetFirewallDomainList", `{"FirewallDomainListId":"`+id+`"}`)["FirewallDomainList"].(map[string]any); got["Status"] != "COMPLETE_IMPORT_FAILED" || got["DomainCount"] != float64(2) {
		t.Fatalf("failed import should keep the old domains: %v", got)
	}
	if code := r53rFail(t, ts, "ImportFirewallDomains", `{"FirewallDomainListId":"`+id+`","Operation":"REPLACE","DomainFileUrl":"s3://lists/missing.txt"}`); code != "ResourceNotFoundException" {
		t.Fatalf("missing file: %s", code)
	}

	lists := r53rOK(t, ts, "ListFirewallDomainLists", `{}`)["FirewallDomainLists"].([]any)
	if len(lists) != 4 {
		t.Fatalf("lists should include the three managed lists: %v", lists)
	}
	if code := r53rFail(t, ts, "DeleteFirewallDomainList", `{"FirewallDomainListId":"rslvr-fdl-aws-malware"}`); code != "AccessDeniedException" {
		t.Fatalf("managed delete: %s", code)
	}
	r53rOK(t, ts, "DeleteFirewallDomainList", `{"FirewallDomainListId":"`+id+`"}`)
	if srv.route53resolver.domains != 0 {
		t.Fatalf("domain budget not released: %d", srv.route53resolver.domains)
	}
}

func TestRoute53ResolverStage2RulesAndAssociations(t *testing.T) {
	_, ts := newRoute53ResolverTestServer(t)
	dl := r53rOK(t, ts, "CreateFirewallDomainList", `{"CreatorRequestId":"d","Name":"block"}`)["FirewallDomainList"].(map[string]any)["Id"].(string)
	dl2 := r53rOK(t, ts, "CreateFirewallDomainList", `{"CreatorRequestId":"d2","Name":"allow"}`)["FirewallDomainList"].(map[string]any)["Id"].(string)
	g := r53rOK(t, ts, "CreateFirewallRuleGroup", `{"CreatorRequestId":"g","Name":"main"}`)["FirewallRuleGroup"].(map[string]any)
	gid := g["Id"].(string)

	rule := func(req, extra string) string {
		return `{"CreatorRequestId":"` + req + `","FirewallRuleGroupId":"` + gid + `","Name":"r"` + extra + `}`
	}
	r := r53rOK(t, ts, "CreateFirewallRule", rule("c1", `,"FirewallDomainListId":"`+dl+`","Priority":10,"Action":"BLOCK","BlockResponse":"NXDOMAIN"`))["FirewallRule"].(map[string]any)
	if r["FirewallDomainRedirectionAction"] != "INSPECT_REDIRECTION_DOMAIN" {
		t.Fatalf("rule: %v", r)
	}
	for name, extra := range map[string]string{
		"block without response": `,"FirewallDomainListId":"` + dl2 + `","Priority":20,"Action":"BLOCK"`,
		"allow with response":    `,"FirewallDomainListId":"` + dl2 + `","Priority":20,"Action":"ALLOW","BlockResponse":"NODATA"`,
		"override incomplete":    `,"FirewallDomainListId":"` + dl2 + `","Priority":20,"Action":"BLOCK","BlockResponse":"OVERRIDE"`,
		"duplicate priority":     `,"FirewallDomainListId":"` + dl2 + `","Priority":10,"Action":"ALLOW"`,
		"no target":              `,"Priority":20,"Action":"ALLOW"`,
		"threat no confidence":   `,"DnsThreatProtection":"DGA","Priority":20,"Action":"ALERT"`,
	} {
		if code := r53rFail(t, ts, "CreateFirewallRule", rule("x-"+name, extra)); code != "ValidationException" {
			t.Fatalf("%s: %s", name, code)
		}
	}
	threat := r53rOK(t, ts, "CreateFirewallRule", rule("c2", `,"DnsThreatProtection":"DGA","ConfidenceThreshold":"HIGH","Priority":5,"Action":"ALERT"`))["FirewallRule"].(map[string]any)
	if threat["FirewallThreatProtectionId"] == nil {
		t.Fatalf("threat rule: %v", threat)
	}

	batch := r53rOK(t, ts, "BatchCreateFirewallRule", `{"CreateFirewallRuleEntries":[`+
		rule("b1", `,"FirewallDomainListId":"`+dl2+`","Priority":30,"Action":"ALLOW"`)+`,`+
		rule("b2", `,"FirewallDomainListId":"`+dl2+`","Priority":40,"Action":"BLOCK"`)+`]}`)
	if len(batch["CreatedFirewallRules"].([]any)) != 1 || len(batch["CreateErrors"].([]any)) != 1 {
		t.Fatalf("batch: %v", batch)
	}

	listed := r53rOK(t, ts, "ListFirewallRules", `{"FirewallRuleGroupId":"`+gid+`"}`)["FirewallRules"].([]any)
	if len(listed) != 3 || listed[0].(map[string]any)["Priority"] != float64(5) {
		t.Fatalf("rules sorted by priority: %v", listed)
	}
	if code := r53rFail(t, ts, "UpdateFirewallRule", `{"FirewallRuleGroupId":"`+gid+`","FirewallDomainListId":"`+dl+`","Priority":30}`); code != "ConflictException" {
		t.Fatalf("update to a taken priority: %s", code)
	}
	upd := r53rOK(t, ts, "UpdateFirewallRule", `{"FirewallRuleGroupId":"`+gid+`","FirewallDomainListId":"`+dl+`","Action":"ALERT"}`)["FirewallRule"].(map[string]any)
	if upd["Action"] != "ALERT" || upd["BlockResponse"] != nil {
		t.Fatalf("switching to ALERT should drop the block response: %v", upd)
	}
	if code := r53rFail(t, ts, "DeleteFirewallDomainList", `{"FirewallDomainListId":"`+dl+`"}`); code != "ConflictException" {
		t.Fatalf("delete list in use: %s", code)
	}
	if n := r53rOK(t, ts, "GetFirewallRuleGroup", `{"FirewallRuleGroupId":"`+gid+`"}`)["FirewallRuleGroup"].(map[string]any)["RuleCount"]; n != float64(3) {
		t.Fatalf("rule count: %v", n)
	}

	assoc := func(req, group string, prio int, extra string) string {
		return `{"CreatorRequestId":"` + req + `","FirewallRuleGroupId":"` + group + `","VpcId":"vpc-00000001","Priority":` + strconv.Itoa(prio) + `,"Name":"a"` + extra + `}`
	}
	if code := r53rFail(t, ts, "AssociateFirewallRuleGroup", assoc("a0", gid, 50, "")); code != "ValidationException" {
		t.Fatalf("priority out of range: %s", code)
	}
	a := r53rOK(t, ts, "AssociateFirewallRuleGroup", assoc("a1", gid, 101, `,"MutationProtection":"ENABLED"`))["FirewallRuleGroupAssociation"].(map[string]any)
	aid := a["Id"].(string)
	g2 := r53rOK(t, ts, "CreateFirewallRuleGroup", `{"CreatorRequestId":"g2","Name":"second"}`)["FirewallRuleGroup"].(map[string]any)["Id"].(string)
	if code := r53rFail(t, ts, "AssociateFirewallRuleGroup", assoc("a2", g2, 101, "")); code != "ConflictException" {
		t.Fatalf("priority taken in VPC: %s", code)
	}
	if code := r53rFail(t, ts, "AssociateFirewallRuleGroup", assoc("a3", gid, 200, "")); code != "ConflictException" {
		t.Fatalf("group already in VPC: %s", code)
	}
	if code := r53rFail(t, ts, "DisassociateFirewallRuleGroup", `{"FirewallRuleGroupAssociationId":"`+aid+`"}`); code != "ValidationException" {
		t.Fatalf("mutation protection: %s", code)
	}
	if code := r53rFail(t, ts, "DeleteFirewallRuleGroup", `{"FirewallRuleGroupId":"`+gid+`"}`); code != "ConflictException" {
		t.Fatalf("delete associated group: %s", code)
	}
	r53rOK(t, ts, "UpdateFirewallRuleGroupAssociation", `{"FirewallRuleGroupAssociationId":"`+aid+`","MutationProtection":"DISABLED"}`)
	if st := r53rOK(t, ts, "GetFirewallRuleGroupAssociation", `{"FirewallRuleGroupAssociationId":"`+aid+`"}`)["FirewallRuleGroupAssociation"].(map[string]any)["Status"]; st != "COMPLETE" {
		t.Fatalf("association status: %v", st)
	}
	r53rOK(t, ts, "DisassociateFirewallRuleGroup", `{"FirewallRuleGroupAssociationId":"`+aid+`"}`)
	r53rOK(t, ts, "DeleteFirewallRuleGroup", `{"FirewallRuleGroupId":"`+gid+`"}`)
	r53rOK(t, ts, "DeleteFirewallDomainList", `{"FirewallDomainListId":"`+dl+`"}`)

	fc := r53rOK(t, ts, "UpdateFirewallConfig", `{"ResourceId":"vpc-00000001","FirewallFailOpen":"ENABLED"}`)["FirewallConfig"].(map[string]any)
	if fc["FirewallFailOpen"] != "ENABLED" {
		t.Fatalf("firewall config: %v", fc)
	}
	if code := r53rFail(t, ts, "GetFirewallConfig", `{"ResourceId":"vpc-nope"}`); code != "ResourceNotFoundException" {
		t.Fatalf("missing vpc: %s", code)
	}
	if n := len(r53rOK(t, ts, "ListFirewallRuleTypes", `{"RuleType":"DnsThreatProtection"}`)["FirewallRuleTypes"].([]any)); n != 3 {
		t.Fatalf("rule types: %d", n)
	}
	if code := r53rFail(t, ts, "PutFirewallRuleGroupPolicy", `{"Arn":"arn:aws:route53resolver:us-east-1:123456789012:firewall-rule-group/nope","FirewallRuleGroupPolicy":"{}"}`); code != "ResourceNotFoundException" {
		t.Fatalf("policy on missing group: %s", code)
	}
}
