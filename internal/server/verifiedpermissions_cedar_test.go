package server

import (
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// A small photo-sharing world used by the evaluator tests.
func cedarTestRequest(principal, action, resource string, ctx cedarRecord) *cedarRequest {
	e := func(t, id string) cedarEntity { return cedarEntity{Type: t, ID: id} }
	ents := map[cedarEntity]*cedarEntityData{
		e("User", "alice"): {Attrs: cedarRecord{"department": "eng", "level": int64(5), "tags": cedarSet{"a", "b"}, "manager": e("User", "bob")},
			Parents: []cedarEntity{e("Group", "admins")}},
		e("User", "bob"):     {Attrs: cedarRecord{"department": "sales", "level": int64(2)}, Parents: []cedarEntity{e("Group", "viewers")}},
		e("Group", "admins"): {Parents: []cedarEntity{e("Group", "viewers")}},
		// A cycle: viewers -> everyone -> viewers. in must still terminate.
		e("Group", "viewers"):  {Parents: []cedarEntity{e("Group", "everyone")}},
		e("Group", "everyone"): {Parents: []cedarEntity{e("Group", "viewers")}},
		e("Photo", "p1"):       {Attrs: cedarRecord{"owner": e("User", "alice"), "private": true, "name": "beach-2024.jpg"}, Parents: []cedarEntity{e("Album", "trips")}},
		e("Action", "view"):    {Parents: []cedarEntity{e("Action", "read")}},
	}
	if ctx == nil {
		ctx = cedarRecord{}
	}
	split := func(s string) cedarEntity {
		i := strings.LastIndex(s, ":")
		return cedarEntity{Type: s[:i], ID: s[i+1:]}
	}
	return &cedarRequest{Principal: split(principal), Action: split(action), Resource: split(resource), Context: ctx, Entities: ents}
}

func mustCedar(t *testing.T, src string) *cedarPolicy {
	t.Helper()
	p, _, err := cedarParsePolicy(src, false)
	if err != nil {
		t.Fatalf("parse %q: %v", src, err)
	}
	return p
}

func TestCedarConditions(t *testing.T) {
	cases := []struct {
		name, policy string
		want         bool
	}{
		{"unconstrained permit", `permit(principal, action, resource);`, true},
		{"principal ==", `permit(principal == User::"alice", action, resource);`, true},
		{"principal == other", `permit(principal == User::"bob", action, resource);`, false},
		{"principal in group chain", `permit(principal in Group::"viewers", action, resource);`, true},
		{"principal in cycle-reachable group", `permit(principal in Group::"everyone", action, resource);`, true},
		{"principal in unrelated group", `permit(principal in Group::"nobody", action, resource);`, false},
		{"principal is", `permit(principal is User, action, resource);`, true},
		{"principal is in", `permit(principal is User in Group::"admins", action, resource);`, true},
		{"principal is wrong type", `permit(principal is Group, action, resource);`, false},
		{"action ==", `permit(principal, action == Action::"view", resource);`, true},
		{"action in group", `permit(principal, action in Action::"read", resource);`, true},
		{"action in list", `permit(principal, action in [Action::"edit", Action::"view"], resource);`, true},
		{"action in list miss", `permit(principal, action in [Action::"edit"], resource);`, false},
		{"resource in album", `permit(principal, action, resource in Album::"trips");`, true},
		{"attribute equality", `permit(principal, action, resource) when { principal.department == "eng" };`, true},
		{"entity attribute", `permit(principal, action, resource) when { resource.owner == principal };`, true},
		{"nested entity attribute", `permit(principal, action, resource) when { principal.manager.department == "sales" };`, true},
		{"comparison", `permit(principal, action, resource) when { principal.level >= 5 && principal.level < 10 };`, true},
		{"arithmetic", `permit(principal, action, resource) when { principal.level * 2 + 1 == 11 };`, true},
		{"negative literal", `permit(principal, action, resource) when { -3 < 0 };`, true},
		{"or short-circuit", `permit(principal, action, resource) when { true || principal.missing };`, true},
		{"and short-circuit", `permit(principal, action, resource) when { false && principal.missing };`, false},
		{"not", `permit(principal, action, resource) when { !resource.private };`, false},
		{"unless", `permit(principal, action, resource) unless { resource.private };`, false},
		{"has present", `permit(principal, action, resource) when { principal has department };`, true},
		{"has missing", `permit(principal, action, resource) when { principal has salary };`, false},
		{"has on record", `permit(principal, action, resource) when { context has mfa && context.mfa };`, true},
		{"like wildcard", `permit(principal, action, resource) when { resource.name like "beach-*.jpg" };`, true},
		{"like miss", `permit(principal, action, resource) when { resource.name like "*.png" };`, false},
		{"like escaped star", `permit(principal, action, resource) when { "a*b" like "a\*b" };`, true},
		{"like escaped star miss", `permit(principal, action, resource) when { "axb" like "a\*b" };`, false},
		{"set contains", `permit(principal, action, resource) when { principal.tags.contains("a") };`, true},
		{"set containsAll", `permit(principal, action, resource) when { principal.tags.containsAll(["a", "b"]) };`, true},
		{"set containsAll miss", `permit(principal, action, resource) when { principal.tags.containsAll(["a", "z"]) };`, false},
		{"set containsAny", `permit(principal, action, resource) when { principal.tags.containsAny(["z", "b"]) };`, true},
		{"set isEmpty", `permit(principal, action, resource) when { [].isEmpty() };`, true},
		{"set equality ignores order", `permit(principal, action, resource) when { [1, 2, 2] == [2, 1] };`, true},
		{"record literal", `permit(principal, action, resource) when { {a: 1, "b": [true]}.b.contains(true) };`, true},
		{"in set of entities", `permit(principal, action, resource) when { principal in [Group::"x", Group::"admins"] };`, true},
		{"is expression", `permit(principal, action, resource) when { resource is Photo in Album::"trips" };`, true},
		{"if then else", `permit(principal, action, resource) when { if principal.level > 3 then context.ok else false };`, true},
		{"cross-type equality is false", `permit(principal, action, resource) when { 1 == "1" };`, false},
		{"indexing", `permit(principal, action, resource) when { principal["department"] == "eng" };`, true},
		{"annotation", `@id("p") permit(principal, action, resource);`, true},
		{"comment", "permit(principal, action, resource) // trailing comment\n;", true},
	}
	ctx := cedarRecord{"mfa": true, "ok": true}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := cedarSatisfied(mustCedar(t, c.policy), cedarTestRequest("User:alice", "Action:view", "Photo:p1", ctx))
			if err != nil {
				t.Fatalf("eval: %v", err)
			}
			if got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestCedarEvaluationErrors(t *testing.T) {
	for _, policy := range []string{
		`permit(principal, action, resource) when { principal.salary > 1 };`, // missing attribute
		`permit(principal, action, resource) when { principal.level + "x" == 1 };`,
		`permit(principal, action, resource) when { 9223372036854775807 + 1 == 0 };`, // overflow
		`permit(principal, action, resource) when { principal.department };`,         // not a boolean
		`permit(principal, action, resource) when { User::"ghost".name == "x" };`,    // unknown entity
		`permit(principal, action, resource) when { "x".contains(1) };`,
	} {
		if _, err := cedarSatisfied(mustCedar(t, policy), cedarTestRequest("User:alice", "Action:view", "Photo:p1", nil)); err == nil {
			t.Errorf("%s: expected an evaluation error", policy)
		}
	}
}

func TestCedarDecision(t *testing.T) {
	req := cedarTestRequest("User:alice", "Action:view", "Photo:p1", nil)
	set := func(policies ...string) map[string]*cedarPolicy {
		out := map[string]*cedarPolicy{}
		for i, p := range policies {
			out[string(rune('a'+i))] = mustCedar(t, p)
		}
		return out
	}
	// Default deny.
	if allow, det, _ := cedarAuthorize(set(), req); allow || len(det) != 0 {
		t.Fatalf("empty set: %v %v", allow, det)
	}
	// Permit allows; determining lists the satisfied permits only.
	allow, det, _ := cedarAuthorize(set(`permit(principal, action, resource);`, `permit(principal == User::"bob", action, resource);`), req)
	if !allow || len(det) != 1 || det[0] != "a" {
		t.Fatalf("permit: %v %v", allow, det)
	}
	// Forbid overrides permit and is the determining policy.
	allow, det, _ = cedarAuthorize(set(`permit(principal, action, resource);`, `forbid(principal, action, resource) when { resource.private };`), req)
	if allow || len(det) != 1 || det[0] != "b" {
		t.Fatalf("forbid: %v %v", allow, det)
	}
	// An erroring policy is skipped and reported; the rest still decide.
	allow, det, errs := cedarAuthorize(set(`permit(principal, action, resource);`, `forbid(principal, action, resource) when { principal.salary > 1 };`), req)
	if !allow || len(det) != 1 || len(errs) != 1 || !strings.Contains(errs[0], "salary") {
		t.Fatalf("erroring forbid: %v %v %v", allow, det, errs)
	}
}

func TestCedarParseErrors(t *testing.T) {
	for _, c := range []struct{ src, want string }{
		{`permit(principal, action);`, "line 1"},
		{`allow(principal, action, resource);`, "permit or forbid"},
		{"permit(\n  principal ==,\n  action, resource);", "line 2"},
		{`permit(principal, action, resource) when { principal.x == "unterminated };`, "unterminated"},
		{`permit(principal, action, resource); permit(principal, action, resource);`, "exactly one policy"},
		{`permit(principal == ?principal, action, resource);`, "static policy"},
		{`permit(principal, action, resource) when { ip("1.2.3.4") };`, "unsupported function"},
		{`permit(principal, action, resource) when { "a\*" == "a" };`, "like pattern"},
		{`permit(principal, action, resource) when { 99999999999999999999 > 1 };`, "out of range"},
		{`permit(principal, action, resource) when { principal.tags.frobnicate(1) };`, "unsupported method"},
		{"permit(principal, action, resource) when { " + strings.Repeat("(", 200) + "true" + strings.Repeat(")", 200) + " };", "nesting"},
	} {
		_, _, err := cedarParsePolicy(c.src, false)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: got %v, want an error mentioning %q", c.src, err, c.want)
		}
	}
	if _, _, err := cedarParsePolicy(strings.Repeat(" ", vpMaxPolicyBytes+1), false); err == nil {
		t.Error("oversized policy was accepted")
	}
	// Templates may use slots.
	p, slots, err := cedarParsePolicy(`permit(principal in ?principal, action, resource == ?resource);`, true)
	if err != nil || !slots["principal"] || !slots["resource"] || !p.Principal.Slot {
		t.Fatalf("template: %v %v", err, slots)
	}
}

func TestCedarLikeMatching(t *testing.T) {
	pat := func(s string) []cedarPatternPart {
		parts, err := cedarPattern(s)
		if err != nil {
			t.Fatal(err)
		}
		return parts
	}
	for _, c := range []struct {
		pattern, s string
		want       bool
	}{
		{"", "", true}, {"", "x", false}, {"*", "", true}, {"*", "anything", true},
		{"abc", "abc", true}, {"abc", "abcd", false}, {"a*", "a", true}, {"*a", "ba", true},
		{"a*b*c", "aXbYc", true}, {"a*b*c", "acb", false}, {"ab*ab", "ab", false}, {"ab*ab", "abab", true},
		{`a\*b`, "a*b", true}, {`\\*`, `\anything`, true},
	} {
		if got := cedarMatch(pat(c.pattern), c.s); got != c.want {
			t.Errorf("%q like %q: got %v, want %v", c.s, c.pattern, got, c.want)
		}
	}
}

// One request's evaluation is bounded. Running out of budget denies the whole
// request: here the expensive forbid would be false, so skipping it would
// wrongly let the permit allow.
func TestCedarEvaluationBudgetDenies(t *testing.T) {
	big := make(cedarSet, 300000)
	for i := range big {
		big[i] = int64(i)
	}
	req := cedarTestRequest("User:alice", "Action:view", "Photo:p1", cedarRecord{"s": big})
	policies := map[string]*cedarPolicy{
		"permit": mustCedar(t, `permit(principal, action, resource);`),
		"heavy":  mustCedar(t, `forbid(principal, action, resource) when { !(context.s.containsAll(context.s) && context.s.containsAll(context.s)) };`),
	}
	allow, det, errs := cedarAuthorize(policies, req)
	if allow || len(det) != 0 || len(errs) != 1 || !strings.Contains(errs[0], "evaluation steps") {
		t.Fatalf("budget: allow=%v determining=%v errors=%v", allow, det, errs)
	}
}

// A value's key is built against the remaining budget: a key far larger than
// the budget allows is abandoned part-way rather than encoded in full first.
func TestCedarKeyStopsAtBudget(t *testing.T) {
	big := make(cedarSet, 50000)
	for i := range big {
		big[i] = strings.Repeat("x", 100) + strconv.Itoa(i)
	}
	budget := 10 // 640 bytes of key
	ev := &cedarEval{budget: &budget}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, err := ev.key(big)
	runtime.ReadMemStats(&after)
	if err == nil {
		t.Fatal("a ~5 MB key fit in a 640-byte budget")
	}
	if grew := after.TotalAlloc - before.TotalAlloc; grew > 1<<20 {
		t.Fatalf("building an over-budget key allocated %d bytes before giving up", grew)
	}
	small := 1000
	ev = &cedarEval{budget: &small}
	if k, err := ev.key(cedarSet{int64(2), int64(1), int64(2)}); err != nil || k != cedarKey(cedarSet{int64(1), int64(2)}) {
		t.Fatalf("small key: %q %v", k, err)
	}
}
