package server

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Route 53 Resolver DNS Firewall: domain lists, rule groups, rules, VPC
// associations, per-VPC fail-open configuration and rule types.

const (
	r53rMaxDomainsPerList = 100_000
	r53rMaxDomainsTotal   = 500_000
	r53rMaxDomainLists    = 1000
	r53rMaxRuleGroups     = 1000
	r53rMaxRulesPerGroup  = 100
	r53rMaxBatchEntries   = 100
	r53rMaxGroupsPerVPC   = 5
	r53rMaxImportBytes    = 10 << 20
	r53rMinFWPriority     = 101
	r53rMaxFWPriority     = 9899
	r53rManagedOwner      = "Route 53 Resolver DNS Firewall"
)

type r53rDomainList struct {
	r53rRec
	domains []string
	set     map[string]struct{}
	managed bool
}

func (d *r53rDomainList) out() map[string]any {
	m := d.view()
	m["DomainCount"] = len(d.domains)
	return m
}

func (st *r53rStore) seedManagedDomainListsLocked() {
	for _, m := range []struct{ id, name string }{
		{"rslvr-fdl-aws-malware", "AWSManagedDomainsMalwareDomainList"},
		{"rslvr-fdl-aws-botnet", "AWSManagedDomainsBotnetCommandandControl"},
		{"rslvr-fdl-aws-aggregate", "AWSManagedDomainsAggregateThreatList"},
	} {
		st.domainLists[m.id] = &r53rDomainList{managed: true, set: map[string]struct{}{}, r53rRec: r53rRec{rec: map[string]any{
			"Id": m.id, "Arn": "arn:aws:route53resolver:" + r53rRegion + "::firewall-domain-list/" + m.id, "Name": m.name,
			"Status": "COMPLETE", "ManagedOwnerName": r53rManagedOwner, "ManagedListType": "THREAT", "CreatorRequestId": m.id,
		}}}
	}
}

// r53rFirewallDomain checks a DNS Firewall domain: labels of letters, digits,
// hyphens and underscores, with an optional leading "*" wildcard label.
var r53rFWLabelRE = regexp.MustCompile(`^[a-z0-9_-]{1,63}$`)

func r53rFirewallDomain(raw string) (string, bool) {
	d := strings.ToLower(strings.TrimSpace(raw))
	if d == "" || len(d) > 255 {
		return "", false
	}
	if d == "*" {
		return d, true
	}
	labels := strings.Split(strings.TrimSuffix(d, "."), ".")
	for i, l := range labels {
		if l == "*" && i == 0 {
			continue
		}
		if !r53rFWLabelRE.MatchString(l) {
			return "", false
		}
	}
	return d, true
}

func (st *r53rStore) domainListLocked(id string) (*r53rDomainList, *r53rErr) {
	if d := st.domainLists[id]; d != nil {
		return d, nil
	}
	return nil, r53rE("ResourceNotFoundException", "firewall domain list %s doesn't exist", id)
}

// setDomainsLocked replaces d's domains, enforcing the per-list and
// store-wide caps that keep memory bounded.
func (st *r53rStore) setDomainsLocked(d *r53rDomainList, domains []string) *r53rErr {
	if len(domains) > r53rMaxDomainsPerList {
		return r53rE("LimitExceededException", "a domain list can hold at most %d domains", r53rMaxDomainsPerList)
	}
	if st.domains-len(d.domains)+len(domains) > r53rMaxDomainsTotal {
		return r53rE("LimitExceededException", "the account can hold at most %d firewall domains", r53rMaxDomainsTotal)
	}
	set := make(map[string]struct{}, len(domains))
	for _, x := range domains {
		set[x] = struct{}{}
	}
	st.domains += len(domains) - len(d.domains)
	d.domains, d.set = domains, set
	d.rec["ModificationTime"] = r53rNow()
	return nil
}

func r53rDomainUpdateResult(d *r53rDomainList) map[string]any {
	return map[string]any{"Id": d.rec["Id"], "Name": d.rec["Name"], "Status": d.rec["Status"], "StatusMessage": d.rec["StatusMessage"]}
}

func (st *r53rStore) ruleGroupLocked(id string) (*r53rRec, *r53rErr) {
	if g := st.ruleGroups[id]; g != nil {
		return g, nil
	}
	return nil, r53rE("ResourceNotFoundException", "firewall rule group %s doesn't exist", id)
}

func (st *r53rStore) groupView(g *r53rRec) map[string]any {
	m := g.view()
	n := 0
	for _, r := range st.fwRules {
		if r.rec["FirewallRuleGroupId"] == m["Id"] {
			n++
		}
	}
	m["RuleCount"] = n
	return m
}

// r53rFWRuleKey names a rule: its group, what it matches and its Qtype.
func r53rFWRuleKey(group, target, qtype string) string { return group + "|" + target + "|" + qtype }

func r53rFWTarget(in map[string]any) string {
	if t := r53rStr(in, "FirewallDomainListId"); t != "" {
		return t
	}
	return r53rStr(in, "FirewallThreatProtectionId")
}

var r53rQtypeRE = regexp.MustCompile(`^([A-Z0-9]{1,10}|TYPE[0-9]{1,5})$`)

// checkFWAction validates the action-specific members of a merged rule.
func r53rCheckFWAction(rec map[string]any) *r53rErr {
	action := r53rStrS(rec["Action"])
	block := r53rStrS(rec["BlockResponse"])
	override := r53rHas(rec, "BlockOverrideDomain") || r53rHas(rec, "BlockOverrideDnsType") || r53rHas(rec, "BlockOverrideTtl")
	switch {
	case action == "BLOCK" && block == "":
		return r53rE("ValidationException", "BLOCK rules need a BlockResponse")
	case action != "BLOCK" && block != "":
		return r53rE("ValidationException", "only BLOCK rules take a BlockResponse")
	case block == "OVERRIDE" && !(r53rHas(rec, "BlockOverrideDomain") && r53rHas(rec, "BlockOverrideDnsType") && r53rHas(rec, "BlockOverrideTtl")):
		return r53rE("ValidationException", "OVERRIDE responses need BlockOverrideDomain, BlockOverrideDnsType and BlockOverrideTtl")
	case block != "OVERRIDE" && override:
		return r53rE("ValidationException", "BlockOverride settings only apply to OVERRIDE responses")
	}
	if q := r53rStrS(rec["Qtype"]); q != "" && !r53rQtypeRE.MatchString(q) {
		return r53rE("ValidationException", "%q isn't a DNS query type", q)
	}
	if r53rHas(rec, "DnsThreatProtection") && !r53rHas(rec, "ConfidenceThreshold") {
		return r53rE("ValidationException", "DnsThreatProtection rules need a ConfidenceThreshold")
	}
	return nil
}

func (st *r53rStore) checkPriorityFreeLocked(group string, priority any, except string) *r53rErr {
	for k, r := range st.fwRules {
		if k != except && r.rec["FirewallRuleGroupId"] == group && fmt.Sprint(r.rec["Priority"]) == fmt.Sprint(priority) {
			return r53rE("ValidationException", "rule group %s already has a rule with priority %v", group, priority)
		}
	}
	return nil
}

var r53rFWRuleMembers = []string{
	"Name", "Priority", "Action", "BlockResponse", "BlockOverrideDomain", "BlockOverrideDnsType", "BlockOverrideTtl",
	"FirewallDomainRedirectionAction", "Qtype", "DnsThreatProtection", "ConfidenceThreshold", "FirewallRuleType",
}

func (st *r53rStore) createFWRuleLocked(in map[string]any) (map[string]any, *r53rErr) {
	group := r53rStr(in, "FirewallRuleGroupId")
	if _, err := st.ruleGroupLocked(group); err != nil {
		return nil, err
	}
	if err := r53rCheckName(in, "Name", "ValidationException"); err != nil {
		return nil, err
	}
	for _, r := range st.fwRules {
		if r.rec["FirewallRuleGroupId"] == group && r.rec["CreatorRequestId"] == r53rStr(in, "CreatorRequestId") {
			return r.rec, nil // the request id makes retries safe
		}
	}
	kinds := 0
	for _, k := range []string{"FirewallDomainListId", "DnsThreatProtection", "FirewallRuleType"} {
		if r53rHas(in, k) {
			kinds++
		}
	}
	if kinds != 1 {
		return nil, r53rE("ValidationException", "a rule matches exactly one of FirewallDomainListId, DnsThreatProtection or FirewallRuleType")
	}
	n := 0
	for _, r := range st.fwRules {
		if r.rec["FirewallRuleGroupId"] == group {
			n++
		}
	}
	if n >= r53rMaxRulesPerGroup {
		return nil, r53rE("LimitExceededException", "a rule group can hold at most %d rules", r53rMaxRulesPerGroup)
	}
	now := r53rNow()
	rec := map[string]any{
		"FirewallRuleGroupId": group, "CreatorRequestId": r53rStr(in, "CreatorRequestId"),
		"Status": "COMPLETE", "CreationTime": now, "ModificationTime": now,
	}
	r53rCopy(rec, in, r53rFWRuleMembers...)
	if id := r53rStr(in, "FirewallDomainListId"); id != "" {
		if _, err := st.domainListLocked(id); err != nil {
			return nil, err
		}
		rec["FirewallDomainListId"] = id
		if !r53rHas(rec, "FirewallDomainRedirectionAction") {
			rec["FirewallDomainRedirectionAction"] = "INSPECT_REDIRECTION_DOMAIN"
		}
	} else {
		rec["FirewallThreatProtectionId"] = r53rID("rslvr-ftp-")
	}
	if err := r53rCheckFWAction(rec); err != nil {
		return nil, err
	}
	if err := st.checkPriorityFreeLocked(group, rec["Priority"], ""); err != nil {
		return nil, err
	}
	key := r53rFWRuleKey(group, r53rFWTarget(rec), r53rStrS(rec["Qtype"]))
	if st.fwRules[key] != nil {
		return nil, r53rE("ValidationException", "rule group %s already has a rule for that domain list and query type", group)
	}
	st.fwRules[key] = &r53rRec{rec: rec}
	return rec, nil
}

func (st *r53rStore) fwRuleLocked(in map[string]any) (string, *r53rRec, *r53rErr) {
	group := r53rStr(in, "FirewallRuleGroupId")
	if _, err := st.ruleGroupLocked(group); err != nil {
		return "", nil, err
	}
	key := r53rFWRuleKey(group, r53rFWTarget(in), r53rStr(in, "Qtype"))
	if r := st.fwRules[key]; r != nil {
		return key, r, nil
	}
	return "", nil, r53rE("ResourceNotFoundException", "rule group %s has no such rule", group)
}

func (st *r53rStore) updateFWRuleLocked(in map[string]any) (map[string]any, *r53rErr) {
	key, r, err := st.fwRuleLocked(in)
	if err != nil {
		return nil, err
	}
	if err := r53rCheckName(in, "Name", "ValidationException"); err != nil {
		return nil, err
	}
	next := make(map[string]any, len(r.rec))
	for k, v := range r.rec {
		next[k] = v
	}
	for _, k := range r53rFWRuleMembers {
		if k != "Qtype" && r53rHas(in, k) {
			next[k] = in[k]
		}
	}
	if next["Action"] != "BLOCK" {
		for _, k := range []string{"BlockResponse", "BlockOverrideDomain", "BlockOverrideDnsType", "BlockOverrideTtl"} {
			if !r53rHas(in, k) {
				delete(next, k)
			}
		}
	}
	if err := r53rCheckFWAction(next); err != nil {
		return nil, err
	}
	if r53rHas(in, "Priority") {
		if err := st.checkPriorityFreeLocked(r53rStrS(next["FirewallRuleGroupId"]), next["Priority"], key); err != nil {
			return nil, r53rE("ConflictException", "%s", err.msg)
		}
	}
	next["ModificationTime"] = r53rNow()
	r.rec = next
	return next, nil
}

func (st *r53rStore) deleteFWRuleLocked(in map[string]any) (map[string]any, *r53rErr) {
	key, r, err := st.fwRuleLocked(in)
	if err != nil {
		return nil, err
	}
	delete(st.fwRules, key)
	return r.rec, nil
}

// r53rBatch runs fn on each entry, collecting results and per-entry errors.
func r53rBatch(entries []any, fn func(map[string]any) (map[string]any, *r53rErr), okKey, errKey, entryKey string) (map[string]any, *r53rErr) {
	if len(entries) > r53rMaxBatchEntries {
		return nil, r53rE("ValidationException", "a batch can hold at most %d entries", r53rMaxBatchEntries)
	}
	done, failed := []any{}, []any{}
	for _, raw := range entries {
		entry, _ := raw.(map[string]any)
		rec, err := fn(entry)
		if err != nil {
			failed = append(failed, map[string]any{entryKey: entry, "Code": err.code, "Message": err.msg})
			continue
		}
		done = append(done, rec)
	}
	return map[string]any{okKey: done, errKey: failed}, nil
}

func (st *r53rStore) fwAssocLocked(id string) (*r53rRec, *r53rErr) {
	if a := st.fwAssocs[id]; a != nil {
		return a, nil
	}
	return nil, r53rE("ResourceNotFoundException", "firewall rule group association %s doesn't exist", id)
}

func r53rCheckFWPriority(p any) *r53rErr {
	n, _ := p.(float64)
	if n < r53rMinFWPriority || n > r53rMaxFWPriority {
		return r53rE("ValidationException", "association priority must be between %d and %d", r53rMinFWPriority, r53rMaxFWPriority)
	}
	return nil
}

// r53rRuleTypes is the ListFirewallRuleTypes catalog (ponytail: a fixed
// sample of the built-in categories; partner products aren't listed).
var r53rRuleTypes = []map[string]any{
	{"RuleType": "DnsThreatProtection", "Value": "DGA", "DisplayName": "Domain generation algorithms", "Description": "Domains generated by malware to reach command-and-control servers."},
	{"RuleType": "DnsThreatProtection", "Value": "DNS_TUNNELING", "DisplayName": "DNS tunneling", "Description": "Data exfiltration or command channels carried over DNS."},
	{"RuleType": "DnsThreatProtection", "Value": "DICTIONARY_DGA", "DisplayName": "Dictionary DGA", "Description": "Generated domains built from dictionary words."},
	{"RuleType": "FirewallAdvancedThreatCategory", "Value": "PHISHING", "DisplayName": "Phishing", "Description": "Domains hosting credential-phishing content."},
	{"RuleType": "FirewallAdvancedThreatCategory", "Value": "MALWARE", "DisplayName": "Malware", "Description": "Domains distributing malware."},
	{"RuleType": "FirewallAdvancedContentCategory", "Value": "GAMBLING", "DisplayName": "Gambling", "Description": "Online gambling sites."},
	{"RuleType": "FirewallAdvancedContentCategory", "Value": "VIOLENCE_AND_HATE_SPEECH", "DisplayName": "Violence and hate speech", "Description": "Sites promoting violence or hate speech."},
}

func init() {
	h := r53rHandlers

	h["CreateFirewallDomainList"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		if err := r53rCheckName(in, "Name", "ValidationException"); err != nil {
			return nil, err
		}
		for _, d := range st.domainLists {
			if !d.managed && d.rec["CreatorRequestId"] == r53rStr(in, "CreatorRequestId") {
				return map[string]any{"FirewallDomainList": d.out()}, nil
			}
		}
		if len(st.domainLists) >= r53rMaxDomainLists {
			return nil, r53rE("LimitExceededException", "an account can have at most %d domain lists", r53rMaxDomainLists)
		}
		id := r53rID("rslvr-fdl-")
		now := r53rNow()
		rec := map[string]any{
			"Id": id, "Arn": r53rARN("firewall-domain-list", id), "Name": r53rStr(in, "Name"),
			"CreatorRequestId": r53rStr(in, "CreatorRequestId"), "Status": "COMPLETE", "StatusMessage": "Created Firewall Domain List",
			"CreationTime": now, "ModificationTime": now,
		}
		if err := st.tagsLocked(rec["Arn"].(string), r53rList(in, "Tags")); err != nil {
			return nil, err
		}
		d := &r53rDomainList{r53rRec: r53rRec{rec: rec}, set: map[string]struct{}{}}
		st.domainLists[id] = d
		return map[string]any{"FirewallDomainList": d.out()}, nil
	}
	h["GetFirewallDomainList"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		d, err := st.domainListLocked(r53rStr(in, "FirewallDomainListId"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"FirewallDomainList": d.out()}, nil
	}
	h["DeleteFirewallDomainList"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		d, err := st.domainListLocked(r53rStr(in, "FirewallDomainListId"))
		if err != nil {
			return nil, err
		}
		if d.managed {
			return nil, r53rE("AccessDeniedException", "managed domain lists can't be deleted")
		}
		for _, r := range st.fwRules {
			if r.rec["FirewallDomainListId"] == d.rec["Id"] {
				return nil, r53rE("ConflictException", "domain list %v is used by a rule in group %v", d.rec["Id"], r.rec["FirewallRuleGroupId"])
			}
		}
		out := map[string]any{}
		for k, v := range d.out() {
			out[k] = v
		}
		out["Status"] = "DELETING"
		st.domains -= len(d.domains)
		delete(st.domainLists, d.rec["Id"].(string))
		st.forgetLocked(d.rec["Arn"].(string))
		return map[string]any{"FirewallDomainList": out}, nil
	}
	h["ListFirewallDomainLists"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		var items []any
		for _, d := range r53rSorted(st.domainLists, func(d *r53rDomainList) string { return d.rec["Id"].(string) }) {
			items = append(items, d.out())
		}
		page, next, err := r53rPage(items, in, "ValidationException")
		return map[string]any{"FirewallDomainLists": page, "NextToken": r53rOmit(next)}, err
	}
	h["ListFirewallDomains"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		d, err := st.domainListLocked(r53rStr(in, "FirewallDomainListId"))
		if err != nil {
			return nil, err
		}
		if d.managed {
			return nil, r53rE("AccessDeniedException", "the domains of managed list %v can't be listed", d.rec["Name"])
		}
		page, next, err := r53rPage(r53rStringsAny(d.domains), in, "ValidationException")
		return map[string]any{"Domains": page, "NextToken": r53rOmit(next)}, err
	}
	h["UpdateFirewallDomains"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		d, err := st.domainListLocked(r53rStr(in, "FirewallDomainListId"))
		if err != nil {
			return nil, err
		}
		if d.managed {
			return nil, r53rE("AccessDeniedException", "managed domain lists can't be changed")
		}
		var given []string
		for _, raw := range r53rStrings(in, "Domains") {
			x, ok := r53rFirewallDomain(raw)
			if !ok {
				return nil, r53rE("ValidationException", "%q isn't a valid domain name", raw)
			}
			given = append(given, x)
		}
		var next []string
		switch r53rStr(in, "Operation") {
		case "ADD":
			next = append([]string(nil), d.domains...)
			seen := map[string]bool{}
			for _, x := range given {
				if _, ok := d.set[x]; !ok && !seen[x] {
					next = append(next, x)
					seen[x] = true
				}
			}
		case "REMOVE":
			drop := map[string]bool{}
			for _, x := range given {
				drop[x] = true
			}
			for _, x := range d.domains {
				if !drop[x] {
					next = append(next, x)
				}
			}
		case "REPLACE":
			seen := map[string]bool{}
			for _, x := range given {
				if !seen[x] {
					next = append(next, x)
					seen[x] = true
				}
			}
		}
		if err := st.setDomainsLocked(d, next); err != nil {
			return nil, err
		}
		d.rec["Status"], d.settle, d.rec["StatusMessage"] = "UPDATING", "COMPLETE", "Updated Firewall Domain List"
		return r53rDomainUpdateResult(d), nil
	}
	h["ImportFirewallDomains"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		d, err := st.domainListLocked(r53rStr(in, "FirewallDomainListId"))
		if err != nil {
			return nil, err
		}
		if d.managed {
			return nil, r53rE("AccessDeniedException", "managed domain lists can't be changed")
		}
		url := r53rStr(in, "DomainFileUrl")
		bucket, key, ok := strings.Cut(strings.TrimPrefix(url, "s3://"), "/")
		if !strings.HasPrefix(url, "s3://") || !ok || bucket == "" || key == "" {
			return nil, r53rE("ValidationException", "DomainFileUrl must be an s3://bucket/key URL")
		}
		obj, gerr := st.s3.GetObject(bucket, key)
		if gerr != nil {
			return nil, r53rE("ResourceNotFoundException", "can't read %s: %v", url, gerr)
		}
		if len(obj.Body) > r53rMaxImportBytes {
			return nil, r53rE("ValidationException", "the domain file can be at most %d bytes", r53rMaxImportBytes)
		}
		var next []string
		seen := map[string]bool{}
		bad := ""
		for i, line := range strings.Split(string(obj.Body), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			x, ok := r53rFirewallDomain(line)
			if !ok {
				bad = fmt.Sprintf("line %d: %q isn't a valid domain name", i+1, line)
				break
			}
			if !seen[x] {
				next, seen[x] = append(next, x), true
			}
		}
		d.rec["Status"] = "IMPORTING"
		if bad != "" {
			d.settle, d.rec["StatusMessage"] = "COMPLETE_IMPORT_FAILED", "Import failed: "+bad
			return r53rDomainUpdateResult(d), nil
		}
		if err := st.setDomainsLocked(d, next); err != nil {
			d.rec["Status"] = "COMPLETE"
			return nil, err
		}
		d.settle, d.rec["StatusMessage"] = "COMPLETE", fmt.Sprintf("Imported %d domains", len(next))
		return r53rDomainUpdateResult(d), nil
	}

	h["CreateFirewallRuleGroup"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		if err := r53rCheckName(in, "Name", "ValidationException"); err != nil {
			return nil, err
		}
		for _, g := range st.ruleGroups {
			if g.rec["CreatorRequestId"] == r53rStr(in, "CreatorRequestId") {
				return map[string]any{"FirewallRuleGroup": st.groupView(g)}, nil
			}
		}
		if len(st.ruleGroups) >= r53rMaxRuleGroups {
			return nil, r53rE("LimitExceededException", "an account can have at most %d rule groups", r53rMaxRuleGroups)
		}
		id := r53rID("rslvr-frg-")
		now := r53rNow()
		rec := map[string]any{
			"Id": id, "Arn": r53rARN("firewall-rule-group", id), "Name": r53rStr(in, "Name"), "OwnerId": r53rAccountID,
			"CreatorRequestId": r53rStr(in, "CreatorRequestId"), "ShareStatus": "NOT_SHARED", "Status": "COMPLETE",
			"StatusMessage": "Created Firewall Rule Group", "CreationTime": now, "ModificationTime": now,
		}
		if err := st.tagsLocked(rec["Arn"].(string), r53rList(in, "Tags")); err != nil {
			return nil, err
		}
		g := &r53rRec{rec: rec}
		st.ruleGroups[id] = g
		return map[string]any{"FirewallRuleGroup": st.groupView(g)}, nil
	}
	h["GetFirewallRuleGroup"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		g, err := st.ruleGroupLocked(r53rStr(in, "FirewallRuleGroupId"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"FirewallRuleGroup": st.groupView(g)}, nil
	}
	h["DeleteFirewallRuleGroup"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		g, err := st.ruleGroupLocked(r53rStr(in, "FirewallRuleGroupId"))
		if err != nil {
			return nil, err
		}
		for _, a := range st.fwAssocs {
			if a.rec["FirewallRuleGroupId"] == g.rec["Id"] {
				return nil, r53rE("ConflictException", "rule group %v is still associated with VPC %v", g.rec["Id"], a.rec["VpcId"])
			}
		}
		out := map[string]any{}
		for k, v := range st.groupView(g) {
			out[k] = v
		}
		out["Status"] = "DELETING"
		for k, r := range st.fwRules {
			if r.rec["FirewallRuleGroupId"] == g.rec["Id"] {
				delete(st.fwRules, k)
			}
		}
		delete(st.ruleGroups, g.rec["Id"].(string))
		st.forgetLocked(g.rec["Arn"].(string))
		return map[string]any{"FirewallRuleGroup": out}, nil
	}
	h["ListFirewallRuleGroups"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		var items []any
		for _, g := range r53rSorted(st.ruleGroups, func(g *r53rRec) string { return g.rec["Id"].(string) }) {
			items = append(items, st.groupView(g))
		}
		page, next, err := r53rPage(items, in, "ValidationException")
		return map[string]any{"FirewallRuleGroups": page, "NextToken": r53rOmit(next)}, err
	}

	h["CreateFirewallRule"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		rec, err := st.createFWRuleLocked(in)
		return map[string]any{"FirewallRule": rec}, err
	}
	h["UpdateFirewallRule"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		rec, err := st.updateFWRuleLocked(in)
		return map[string]any{"FirewallRule": rec}, err
	}
	h["DeleteFirewallRule"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		rec, err := st.deleteFWRuleLocked(in)
		return map[string]any{"FirewallRule": rec}, err
	}
	h["BatchCreateFirewallRule"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		return r53rBatch(r53rList(in, "CreateFirewallRuleEntries"), st.createFWRuleLocked, "CreatedFirewallRules", "CreateErrors", "FirewallRule")
	}
	h["BatchUpdateFirewallRule"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		return r53rBatch(r53rList(in, "UpdateFirewallRuleEntries"), st.updateFWRuleLocked, "UpdatedFirewallRules", "UpdateErrors", "FirewallRule")
	}
	h["BatchDeleteFirewallRule"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		return r53rBatch(r53rList(in, "DeleteFirewallRuleEntries"), st.deleteFWRuleLocked, "DeletedFirewallRules", "DeleteErrors", "FirewallRule")
	}
	h["ListFirewallRules"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		group := r53rStr(in, "FirewallRuleGroupId")
		if _, err := st.ruleGroupLocked(group); err != nil {
			return nil, err
		}
		var rules []map[string]any
		for _, r := range st.fwRules {
			if r.rec["FirewallRuleGroupId"] != group {
				continue
			}
			if p, ok := r53rInt(in, "Priority"); ok && fmt.Sprint(r.rec["Priority"]) != fmt.Sprint(float64(p)) {
				continue
			}
			if a := r53rStr(in, "Action"); a != "" && r.rec["Action"] != a {
				continue
			}
			rules = append(rules, r.rec)
		}
		sort.Slice(rules, func(i, j int) bool {
			pi, _ := rules[i]["Priority"].(float64)
			pj, _ := rules[j]["Priority"].(float64)
			return pi < pj
		})
		page, next, err := r53rPage(r53rAny(rules), in, "ValidationException")
		return map[string]any{"FirewallRules": page, "NextToken": r53rOmit(next)}, err
	}

	h["AssociateFirewallRuleGroup"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		g, err := st.ruleGroupLocked(r53rStr(in, "FirewallRuleGroupId"))
		if err != nil {
			return nil, err
		}
		if err := r53rCheckName(in, "Name", "ValidationException"); err != nil {
			return nil, err
		}
		vpc := r53rStr(in, "VpcId")
		if !st.vpcExists(vpc) {
			return nil, r53rE("ResourceNotFoundException", "VPC %s doesn't exist", vpc)
		}
		for _, a := range st.fwAssocs {
			if a.rec["CreatorRequestId"] == r53rStr(in, "CreatorRequestId") {
				return map[string]any{"FirewallRuleGroupAssociation": a.view()}, nil
			}
		}
		if err := r53rCheckFWPriority(in["Priority"]); err != nil {
			return nil, err
		}
		n := 0
		for _, a := range st.fwAssocs {
			if a.rec["VpcId"] != vpc {
				continue
			}
			n++
			if a.rec["FirewallRuleGroupId"] == g.rec["Id"] {
				return nil, r53rE("ConflictException", "rule group %v is already associated with VPC %s", g.rec["Id"], vpc)
			}
			if fmt.Sprint(a.rec["Priority"]) == fmt.Sprint(in["Priority"]) {
				return nil, r53rE("ConflictException", "VPC %s already has a rule group at priority %v", vpc, in["Priority"])
			}
		}
		if n >= r53rMaxGroupsPerVPC {
			return nil, r53rE("LimitExceededException", "a VPC can have at most %d rule groups", r53rMaxGroupsPerVPC)
		}
		if len(st.fwAssocs) >= r53rMaxAssociations {
			return nil, r53rE("LimitExceededException", "an account can have at most %d rule group associations", r53rMaxAssociations)
		}
		id := r53rID("rslvr-frgassoc-")
		now := r53rNow()
		rec := map[string]any{
			"Id": id, "Arn": r53rARN("firewall-rule-group-association", id), "FirewallRuleGroupId": g.rec["Id"], "VpcId": vpc,
			"Name": r53rStr(in, "Name"), "Priority": in["Priority"], "MutationProtection": "DISABLED", "Status": "UPDATING",
			"StatusMessage": "Creating Firewall Rule Group Association", "CreatorRequestId": r53rStr(in, "CreatorRequestId"),
			"CreationTime": now, "ModificationTime": now,
		}
		r53rCopy(rec, in, "MutationProtection")
		if err := st.tagsLocked(rec["Arn"].(string), r53rList(in, "Tags")); err != nil {
			return nil, err
		}
		st.fwAssocs[id] = &r53rRec{rec: rec, settle: "COMPLETE"}
		return map[string]any{"FirewallRuleGroupAssociation": rec}, nil
	}
	h["GetFirewallRuleGroupAssociation"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		a, err := st.fwAssocLocked(r53rStr(in, "FirewallRuleGroupAssociationId"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"FirewallRuleGroupAssociation": a.view()}, nil
	}
	h["UpdateFirewallRuleGroupAssociation"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		a, err := st.fwAssocLocked(r53rStr(in, "FirewallRuleGroupAssociationId"))
		if err != nil {
			return nil, err
		}
		if err := r53rCheckName(in, "Name", "ValidationException"); err != nil {
			return nil, err
		}
		if r53rHas(in, "Priority") {
			if err := r53rCheckFWPriority(in["Priority"]); err != nil {
				return nil, err
			}
			for id, other := range st.fwAssocs {
				if id != a.rec["Id"] && other.rec["VpcId"] == a.rec["VpcId"] && fmt.Sprint(other.rec["Priority"]) == fmt.Sprint(in["Priority"]) {
					return nil, r53rE("ConflictException", "VPC %v already has a rule group at priority %v", a.rec["VpcId"], in["Priority"])
				}
			}
		}
		r53rCopy(a.rec, in, "Priority", "MutationProtection", "Name")
		a.rec["Status"], a.settle, a.rec["ModificationTime"] = "UPDATING", "COMPLETE", r53rNow()
		return map[string]any{"FirewallRuleGroupAssociation": a.rec}, nil
	}
	h["DisassociateFirewallRuleGroup"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		a, err := st.fwAssocLocked(r53rStr(in, "FirewallRuleGroupAssociationId"))
		if err != nil {
			return nil, err
		}
		if a.rec["MutationProtection"] == "ENABLED" {
			return nil, r53rE("ValidationException", "association %v has mutation protection enabled; disable it first", a.rec["Id"])
		}
		out := map[string]any{}
		for k, v := range a.view() {
			out[k] = v
		}
		out["Status"] = "DELETING"
		delete(st.fwAssocs, a.rec["Id"].(string))
		st.forgetLocked(a.rec["Arn"].(string))
		return map[string]any{"FirewallRuleGroupAssociation": out}, nil
	}
	h["ListFirewallRuleGroupAssociations"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		var items []any
		for _, a := range r53rViews(st.fwAssocs) {
			if v := r53rStr(in, "FirewallRuleGroupId"); v != "" && a["FirewallRuleGroupId"] != v {
				continue
			}
			if v := r53rStr(in, "VpcId"); v != "" && a["VpcId"] != v {
				continue
			}
			if v := r53rStr(in, "Status"); v != "" && a["Status"] != v {
				continue
			}
			if p, ok := r53rInt(in, "Priority"); ok && fmt.Sprint(a["Priority"]) != fmt.Sprint(float64(p)) {
				continue
			}
			items = append(items, a)
		}
		page, next, err := r53rPage(items, in, "ValidationException")
		return map[string]any{"FirewallRuleGroupAssociations": page, "NextToken": r53rOmit(next)}, err
	}

	h["GetFirewallConfig"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		rec, err := r53rFirewallConfig.get(st, r53rStr(in, "ResourceId"))
		return map[string]any{"FirewallConfig": rec}, err
	}
	h["UpdateFirewallConfig"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		vpc := r53rStr(in, "ResourceId")
		if !st.vpcExists(vpc) {
			return nil, r53rE("ResourceNotFoundException", "VPC %s doesn't exist", vpc)
		}
		r := r53rFirewallConfig.getLocked(st, vpc)
		r.rec["FirewallFailOpen"] = r53rStr(in, "FirewallFailOpen")
		return map[string]any{"FirewallConfig": r.rec}, nil
	}
	h["ListFirewallConfigs"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		var items []any
		for _, vpc := range st.ec2.DescribeVpcs(nil) {
			items = append(items, r53rFirewallConfig.getLocked(st, vpc.ID).rec)
		}
		page, next, err := r53rPage(items, in, "ValidationException")
		return map[string]any{"FirewallConfigs": page, "NextToken": r53rOmit(next)}, err
	}
	h["ListFirewallRuleTypes"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		want := r53rStr(in, "RuleType")
		var items []any
		known := false
		for _, t := range r53rRuleTypes {
			if want == "" || t["RuleType"] == want {
				known = true
				items = append(items, t)
			}
		}
		if !known && want != "" {
			return nil, r53rE("ValidationException", "unknown rule type %q", want)
		}
		page, next, err := r53rPage(items, in, "ValidationException")
		return map[string]any{"FirewallRuleTypes": page, "NextToken": r53rOmit(next)}, err
	}

	r53rPolicyHandlers(h, "PutFirewallRuleGroupPolicy", "GetFirewallRuleGroupPolicy", "FirewallRuleGroupPolicy",
		func(st *r53rStore, arn string) bool {
			for _, g := range st.ruleGroups {
				if g.rec["Arn"] == arn {
					return true
				}
			}
			return false
		}, "ResourceNotFoundException", "ValidationException")
}
