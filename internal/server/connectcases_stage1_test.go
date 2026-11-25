package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestConnectCasesStage1CaseLifecycle(t *testing.T) {
	_, ts := newConnectTestServer(t)
	d, field, tmpl := cncFixture(t, ts)
	base := "/domains/" + d

	// Templates enforce their required fields; values must match field types.
	cncFail(t, ts, http.MethodPost, base+"/cases", fmt.Sprintf(`{"templateId":%q,"fields":[]}`, tmpl), http.StatusBadRequest)
	cncFail(t, ts, http.MethodPost, base+"/cases", fmt.Sprintf(`{"templateId":%q,"fields":[{"id":"title","value":{"stringValue":"x"}},{"id":%q,"value":{"stringValue":"high"}}]}`, tmpl, field), http.StatusBadRequest)
	cncFail(t, ts, http.MethodPost, base+"/cases", fmt.Sprintf(`{"templateId":%q,"fields":[{"id":"title","value":{"stringValue":"x"}},{"id":"status","value":{"stringValue":"nope"}}]}`, tmpl), http.StatusBadRequest)
	cncFail(t, ts, http.MethodPost, base+"/cases", `{"templateId":"missing","fields":[]}`, http.StatusNotFound)

	// Idempotent create with a client token.
	body := fmt.Sprintf(`{"templateId":%q,"clientToken":"tok-1","fields":[{"id":"title","value":{"stringValue":"Printer jammed"}},{"id":%q,"value":{"doubleValue":3}}],"tags":{"env":"test"}}`, tmpl, field)
	c1 := cncOK(t, ts, http.MethodPost, base+"/cases", body)["caseId"].(string)
	if again := cncOK(t, ts, http.MethodPost, base+"/cases", body)["caseId"]; again != c1 {
		t.Fatalf("client token should return the same case: %v vs %v", again, c1)
	}
	c2 := cncCreateCase(t, ts, d, tmpl, "Laptop screen cracked", 1)

	got := cncOK(t, ts, http.MethodPost, base+"/cases/"+c1, fmt.Sprintf(`{"fields":[{"id":"title"},{"id":"status"},{"id":"reference_number"},{"id":%q}]}`, field))
	raw, _ := json.Marshal(got)
	if !strings.Contains(string(raw), "Printer jammed") || !strings.Contains(string(raw), `"open"`) || !strings.Contains(string(raw), `"team":"support"`) || !strings.Contains(string(raw), `"env":"test"`) {
		t.Fatalf("GetCase: %s", raw)
	}

	// Updates: read-only system fields are refused; closing stamps last_closed_datetime.
	cncFail(t, ts, http.MethodPut, base+"/cases/"+c1, `{"fields":[{"id":"reference_number","value":{"stringValue":"1"}}]}`, http.StatusBadRequest)
	cncOK(t, ts, http.MethodPut, base+"/cases/"+c1, `{"fields":[{"id":"status","value":{"stringValue":"closed"}}],"performedBy":{"customEntity":"agent-7"}}`)
	got = cncOK(t, ts, http.MethodPost, base+"/cases/"+c1, `{"fields":[{"id":"last_closed_datetime"}]}`)
	if fields := got["fields"].([]any); len(fields) != 1 {
		t.Fatalf("closing should set last_closed_datetime: %v", got)
	}
	audit := cncOK(t, ts, http.MethodPost, base+"/cases/"+c1+"/audit-history", `{}`)["auditEvents"].([]any)
	first := audit[0].(map[string]any)
	raw, _ = json.Marshal(first)
	if first["type"] != "Case.Updated" || !strings.Contains(string(raw), `"oldValue":{"stringValue":"open"}`) || !strings.Contains(string(raw), "agent-7") {
		t.Fatalf("audit: %s", raw)
	}

	// Search: term, field filters, tags, sorting and totals.
	search := func(q string) []string {
		out := cncOK(t, ts, http.MethodPost, base+"/cases-search", q)
		ids := []string{}
		for _, c := range out["cases"].([]any) {
			ids = append(ids, c.(map[string]any)["caseId"].(string))
		}
		return ids
	}
	if ids := search(`{"searchTerm":"PRINTER"}`); len(ids) != 1 || ids[0] != c1 {
		t.Fatalf("term search: %v", ids)
	}
	if ids := search(fmt.Sprintf(`{"filter":{"field":{"greaterThanOrEqualTo":{"id":%q,"value":{"doubleValue":2}}}}}`, field)); len(ids) != 1 || ids[0] != c1 {
		t.Fatalf("number filter: %v", ids)
	}
	if ids := search(`{"filter":{"not":{"field":{"equalTo":{"id":"status","value":{"stringValue":"closed"}}}}}}`); len(ids) != 1 || ids[0] != c2 {
		t.Fatalf("not filter: %v", ids)
	}
	if ids := search(`{"filter":{"tag":{"equalTo":{"key":"env","value":"test"}}}}`); len(ids) != 1 || ids[0] != c1 {
		t.Fatalf("tag filter: %v", ids)
	}
	if ids := search(`{"sorts":[{"fieldId":"title","sortOrder":"Asc"}]}`); len(ids) != 2 || ids[0] != c2 {
		t.Fatalf("sort: %v", ids)
	}
	// Filter complexity is capped before the store is scanned.
	nodes := make([]string, 25)
	for i := range nodes {
		nodes[i] = `{"field":{"contains":{"id":"title","value":{"stringValue":"a"}}}}`
	}
	cncFail(t, ts, http.MethodPost, base+"/cases-search", `{"filter":{"orAll":[{"orAll":[`+strings.Join(nodes[:9], ",")+`]},{"orAll":[`+strings.Join(nodes[9:18], ",")+`]},{"orAll":[`+strings.Join(nodes[18:], ",")+`]}]}}`, http.StatusBadRequest)
	deep := `{"field":{"equalTo":{"id":"title","value":{"stringValue":"a"}}}}`
	for i := 0; i < 7; i++ {
		deep = `{"not":` + deep + `}`
	}
	cncFail(t, ts, http.MethodPost, base+"/cases-search", `{"filter":`+deep+`}`, http.StatusBadRequest)
	cncFail(t, ts, http.MethodPost, base+"/cases-search", `{"filter":{"field":{"equalTo":{"id":"title","value":{"stringValue":"a","doubleValue":1}}}}}`, http.StatusBadRequest)

	// Batch update by reference number and by case id.
	ref := cncOK(t, ts, http.MethodPost, base+"/cases/"+c2, `{"fields":[{"id":"reference_number"}]}`)["fields"].([]any)[0].(map[string]any)["value"].(map[string]any)["stringValue"].(string)
	out := cncOK(t, ts, http.MethodPut, "/cases:batchUpdateCase", fmt.Sprintf(`{"domainId":%q,"cases":[{"caseIdentifier":{"referenceId":%q}},{"caseIdentifier":{"caseId":"nope"}}],"fields":[{"fieldIdentifier":{"name":"priority"},"value":{"doubleValue":9}}],"title":"Laptop replaced"}`, d, ref))
	if len(out["successfulCases"].([]any)) != 1 || len(out["failedCases"].([]any)) != 1 {
		t.Fatalf("batch update: %v", out)
	}
	if ids := search(`{"searchTerm":"replaced"}`); len(ids) != 1 || ids[0] != c2 {
		t.Fatalf("batch update didn't apply: %v", ids)
	}

	cncOK(t, ts, http.MethodDelete, base+"/cases/"+c2, "")
	cncFail(t, ts, http.MethodPost, base+"/cases/"+c2, `{"fields":[{"id":"title"}]}`, http.StatusNotFound)
}

func TestConnectCasesStage1RelatedItemsAndMerge(t *testing.T) {
	srv, ts := newConnectTestServer(t)
	d, _, tmpl := cncFixture(t, ts)
	base := "/domains/" + d
	c1 := cncCreateCase(t, ts, d, tmpl, "Primary", 1)
	c2 := cncCreateCase(t, ts, d, tmpl, "Secondary", 1)

	// Contacts must be real chat contacts started through Connect.
	contactID, _, _, cerr := srv.connectParticipant.startChat(cnpChatStart{instanceID: "instance-000001", flowID: "f", displayName: "Ann"})
	if cerr != nil {
		t.Fatal(cerr.msg)
	}
	contactARN := cnpContactARN("instance-000001", contactID)
	cncFail(t, ts, http.MethodPost, base+"/cases/"+c1+"/related-items/", `{"type":"Contact","content":{"contact":{"contactArn":"arn:aws:connect:us-east-1:123456789012:instance/instance-000001/contact/nope"}}}`, http.StatusNotFound)
	cncFail(t, ts, http.MethodPost, base+"/cases/"+c1+"/related-items/", `{"type":"Comment","content":{"contact":{"contactArn":"x"}}}`, http.StatusBadRequest)
	cncOK(t, ts, http.MethodPost, base+"/cases/"+c1+"/related-items/", fmt.Sprintf(`{"type":"Contact","content":{"contact":{"contactArn":%q}}}`, contactARN))
	comment := cncOK(t, ts, http.MethodPost, base+"/cases/"+c2+"/related-items/", `{"type":"Comment","content":{"comment":{"body":"from secondary","contentType":"Text/Plain"}}}`)["relatedItemId"].(string)
	sla := cncOK(t, ts, http.MethodPost, base+"/cases/"+c1+"/related-items/", `{"type":"Sla","content":{"sla":{"slaInputConfiguration":{"name":"close fast","type":"CaseField","fieldId":"status","targetFieldValues":[{"stringValue":"closed"}],"targetSlaMinutes":60}}}}`)["relatedItemId"].(string)

	cases := cncOK(t, ts, http.MethodPost, base+"/list-cases-for-contact", fmt.Sprintf(`{"contactArn":%q}`, contactARN))["cases"].([]any)
	if len(cases) != 1 || cases[0].(map[string]any)["caseId"] != c1 {
		t.Fatalf("ListCasesForContact: %v", cases)
	}
	slaStatus := func() string {
		items := cncOK(t, ts, http.MethodPost, base+"/cases/"+c1+"/related-items-search", `{"filters":[{"sla":{}}]}`)["relatedItems"].([]any)
		return items[0].(map[string]any)["content"].(map[string]any)["sla"].(map[string]any)["slaConfiguration"].(map[string]any)["status"].(string)
	}
	if s := slaStatus(); s != "Active" {
		t.Fatalf("SLA should start Active: %s", s)
	}
	cncOK(t, ts, http.MethodPut, base+"/cases/"+c1, `{"fields":[{"id":"status","value":{"stringValue":"closed"}}]}`)
	if s := slaStatus(); s != "Met" {
		t.Fatalf("SLA should be Met once the case closes: %s", s)
	}
	upd := cncOK(t, ts, http.MethodPut, base+"/cases/"+c1+"/related-items/"+sla, `{"content":{"sla":{"slaUpdateConfiguration":{"targetSlaMinutes":5}}}}`)
	if upd["type"] != "Sla" {
		t.Fatalf("UpdateRelatedItem: %v", upd)
	}
	cncFail(t, ts, http.MethodPut, base+"/cases/"+c1+"/related-items/"+sla, `{"content":{"comment":{"body":"x","contentType":"Text/Plain"}}}`, http.StatusBadRequest)

	// Merging moves the secondary's comment and copies its title.
	cncOK(t, ts, http.MethodPost, base+"/cases/"+c1+"/merge", fmt.Sprintf(`{"secondaryCaseIds":[%q],"fieldsConfiguration":[{"sourceCaseId":%q,"fieldIds":[{"id":"title"}]}]}`, c2, c2))
	cncFail(t, ts, http.MethodPost, base+"/cases/"+c2, `{"fields":[{"id":"title"}]}`, http.StatusNotFound)
	status := cncOK(t, ts, http.MethodPost, base+"/cases/"+c2+"/merge-status", "")
	if status["status"] != "COMPLETED" || status["primaryCaseId"] != c1 {
		t.Fatalf("merge status: %v", status)
	}
	all := cncOK(t, ts, http.MethodPost, base+"/related-items-search", `{"filters":[{"comment":{}}]}`)["relatedItems"].([]any)
	if len(all) != 1 || all[0].(map[string]any)["caseId"] != c1 || all[0].(map[string]any)["relatedItemId"] != comment {
		t.Fatalf("merged comment: %v", all)
	}
	title := cncOK(t, ts, http.MethodPost, base+"/cases/"+c1, `{"fields":[{"id":"title"}]}`)
	if raw, _ := json.Marshal(title); !strings.Contains(string(raw), "Secondary") {
		t.Fatalf("merged title: %s", raw)
	}
	events := cncOK(t, ts, http.MethodPost, base+"/cases/"+c1+"/audit-history", `{"maxResults":1}`)
	if ev := events["auditEvents"].([]any)[0].(map[string]any); ev["type"] != "Case.Merged" || events["nextToken"] == nil {
		t.Fatalf("merge audit: %v", events)
	}
}

func TestConnectCasesStage1ConfigurationResources(t *testing.T) {
	_, ts := newConnectTestServer(t)
	d, field, tmpl := cncFixture(t, ts)
	base := "/domains/" + d

	cncFail(t, ts, http.MethodPost, "/domains", `{"name":"support"}`, http.StatusConflict)
	cncFail(t, ts, http.MethodPost, base+"/fields", `{"name":"priority","type":"Text"}`, http.StatusConflict)
	cncFail(t, ts, http.MethodDelete, base+"/fields/title", "", http.StatusBadRequest)
	cncFail(t, ts, http.MethodDelete, base+"/fields/"+field, "", http.StatusConflict) // in the layout

	// Field options apply to SingleSelect fields and are listed back.
	sel := cncOK(t, ts, http.MethodPost, base+"/fields", `{"name":"tier","type":"SingleSelect"}`)["fieldId"].(string)
	cncFail(t, ts, http.MethodPut, base+"/fields/"+field+"/options", `{"options":[{"name":"A","value":"a","active":true}]}`, http.StatusBadRequest)
	cncOK(t, ts, http.MethodPut, base+"/fields/"+sel+"/options", `{"options":[{"name":"Gold","value":"gold","active":true},{"name":"Silver","value":"silver","active":false}]}`)
	opts := cncOK(t, ts, http.MethodPost, base+"/fields/"+sel+"/options-list?values=gold", "")["options"].([]any)
	if len(opts) != 1 || opts[0].(map[string]any)["name"] != "Gold" {
		t.Fatalf("options: %v", opts)
	}
	fields := cncOK(t, ts, http.MethodPost, base+"/fields-batch", fmt.Sprintf(`{"fields":[{"id":%q},{"id":"nope"}]}`, sel))
	if len(fields["fields"].([]any)) != 1 || len(fields["errors"].([]any)) != 1 {
		t.Fatalf("BatchGetField: %v", fields)
	}

	// Case rules: a Required rule on a template makes its field required.
	rule := cncOK(t, ts, http.MethodPost, base+"/case-rules", `{"name":"tier required","rule":{"required":{"defaultValue":true,"conditions":[{"equalTo":{"operandOne":{"fieldId":"status"},"operandTwo":{"stringValue":"closed"},"result":false}}]}}}`)["caseRuleId"].(string)
	cncFail(t, ts, http.MethodPost, base+"/case-rules", `{"name":"bad","rule":{"required":{"defaultValue":true,"conditions":[]},"hidden":{"defaultValue":true,"conditions":[]}}}`, http.StatusBadRequest)
	cncOK(t, ts, http.MethodPut, base+"/templates/"+tmpl, fmt.Sprintf(`{"rules":[{"caseRuleId":%q,"fieldId":%q}]}`, rule, sel))
	cncFail(t, ts, http.MethodPost, base+"/cases", fmt.Sprintf(`{"templateId":%q,"fields":[{"id":"title","value":{"stringValue":"x"}}]}`, tmpl), http.StatusBadRequest)
	cncOK(t, ts, http.MethodPost, base+"/cases", fmt.Sprintf(`{"templateId":%q,"fields":[{"id":"title","value":{"stringValue":"x"}},{"id":"status","value":{"stringValue":"closed"}}]}`, tmpl))
	cncFail(t, ts, http.MethodDelete, base+"/case-rules/"+rule, "", http.StatusConflict)
	rules := cncOK(t, ts, http.MethodPost, base+"/rules-list/", "")["caseRules"].([]any)
	if len(rules) != 1 || rules[0].(map[string]any)["ruleType"] != "Required" {
		t.Fatalf("ListCaseRules: %v", rules)
	}

	// Templates by status; inactive templates take no new cases.
	cncOK(t, ts, http.MethodPut, base+"/templates/"+tmpl, `{"status":"Inactive"}`)
	if ts := cncOK(t, ts, http.MethodPost, base+"/templates-list?status=Active", "")["templates"].([]any); len(ts) != 0 {
		t.Fatalf("status filter: %v", ts)
	}
	cncFail(t, ts, http.MethodPost, base+"/cases", fmt.Sprintf(`{"templateId":%q,"fields":[]}`, tmpl), http.StatusBadRequest)

	// Event configuration and tags round-trip.
	cncOK(t, ts, http.MethodPut, base+"/case-event-configuration", `{"eventBridge":{"enabled":true,"includedData":{"caseData":{"fields":[{"id":"title"}]}}}}`)
	if eb := cncOK(t, ts, http.MethodPost, base+"/case-event-configuration", "")["eventBridge"].(map[string]any); eb["enabled"] != true {
		t.Fatalf("event config: %v", eb)
	}
	arn := strings.ReplaceAll(fmt.Sprintf("arn:aws:cases:us-east-1:123456789012:domain/%s", d), "/", "%2F")
	cncOK(t, ts, http.MethodPost, "/tags/"+arn, `{"tags":{"a":"1","b":"2"}}`)
	cncOK(t, ts, http.MethodDelete, "/tags/"+arn+"?tagKeys=a", "")
	if tags := cncOK(t, ts, http.MethodGet, "/tags/"+arn, "")["tags"].(map[string]any); len(tags) != 1 || tags["b"] != "2" {
		t.Fatalf("tags: %v", tags)
	}
	cncFail(t, ts, http.MethodPost, "/tags/"+arn, `{"tags":{"aws:x":"1"}}`, http.StatusBadRequest)

	cncOK(t, ts, http.MethodDelete, base, "")
	cncFail(t, ts, http.MethodPost, base, "", http.StatusNotFound)
	cncFail(t, ts, http.MethodGet, "/tags/"+arn, "", http.StatusNotFound)
}

// Stored case bytes are bounded per case and store-wide, and the audit
// history trims its oldest events instead of growing without limit.
func TestConnectCasesStage1Limits(t *testing.T) {
	srv, ts := newConnectTestServer(t)
	d, _, tmpl := cncFixture(t, ts)
	base := "/domains/" + d
	c := cncCreateCase(t, ts, d, tmpl, "big", 1)
	long := strings.Repeat("x", 4000)
	for i := 0; i < cncMaxAudit+20; i++ {
		cncOK(t, ts, http.MethodPut, base+"/cases/"+c, fmt.Sprintf(`{"fields":[{"id":"summary","value":{"stringValue":"%d%s"}}]}`, i, long))
	}
	srv.connectCases.mu.Lock()
	kase := srv.connectCases.domains[d].cases[c]
	n, size, total := len(kase.audit), kase.size(), srv.connectCases.bytes
	srv.connectCases.mu.Unlock()
	if n > cncMaxAudit || size > cncMaxCaseBytes || total > cncMaxStoreBytes || total < size {
		t.Fatalf("audit %d events, case %d bytes, store %d bytes", n, size, total)
	}
	// Related items stop at the per-case cap.
	srv.connectCases.mu.Lock()
	for i := len(kase.related); i < cncMaxRelated; i++ {
		kase.related[fmt.Sprint(i)] = &cncRelated{id: fmt.Sprint(i), typ: "Comment"}
	}
	srv.connectCases.mu.Unlock()
	if code := cncFail(t, ts, http.MethodPost, base+"/cases/"+c+"/related-items/", `{"type":"Comment","content":{"comment":{"body":"x","contentType":"Text/Plain"}}}`, http.StatusPaymentRequired); code != "ServiceQuotaExceededException" {
		t.Fatalf("related cap: %s", code)
	}

	// Configuration (fields, options, layouts, templates, rules) has its own budget.
	srv.connectCases.mu.Lock()
	used := srv.connectCases.config
	srv.connectCases.config = cncMaxConfigBytes - 100
	srv.connectCases.mu.Unlock()
	if code := cncFail(t, ts, http.MethodPost, base+"/fields", `{"name":"overflow","type":"Text"}`, http.StatusPaymentRequired); code != "ServiceQuotaExceededException" {
		t.Fatalf("config budget: %s", code)
	}
	srv.connectCases.mu.Lock()
	srv.connectCases.config = used
	srv.connectCases.mu.Unlock()

	// Deleting a case drops its client token, so tokens don't pile up.
	tok := cncOK(t, ts, http.MethodPost, base+"/cases", fmt.Sprintf(`{"templateId":%q,"clientToken":"once","fields":[{"id":"title","value":{"stringValue":"t"}}]}`, tmpl))["caseId"].(string)
	cncOK(t, ts, http.MethodDelete, base+"/cases/"+tok, "")
	srv.connectCases.mu.Lock()
	_, kept := srv.connectCases.domains[d].clientTokens["once"]
	srv.connectCases.mu.Unlock()
	if kept {
		t.Fatalf("client token outlived its case")
	}
	cncOK(t, ts, http.MethodDelete, base, "")
	srv.connectCases.mu.Lock()
	left := srv.connectCases.config + srv.connectCases.bytes
	srv.connectCases.mu.Unlock()
	if left != 0 {
		t.Fatalf("deleting the only domain should release every byte, %d left", left)
	}
}
