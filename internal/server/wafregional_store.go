package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
)

const (
	wrRegion       = "us-east-1"
	wrAccount      = "123456789012"
	wrMaxTokens    = 10000 // change tokens kept; the oldest are forgotten first
	wrMaxEntities  = 100   // WAF Classic's per-type quota for conditions and rules
	wrMaxWebACLs   = 50
	wrMaxAssocs    = 1000
	wrMaxTags      = 50
	wrDefaultLimit = 100
	wrMaxTargetLen = 50 // ByteMatchTuple.TargetString, decoded
)

// wrKind describes one WAF Classic entity type. They all share the same
// Create/Get/Update/Delete/List shape and differ in the list of items an
// Update inserts into or deletes from.
type wrKind struct {
	name     string // operation suffix, e.g. "IPSet"
	objKey   string // response key holding the entity (Rule for RateBasedRule)
	idKey    string
	itemsKey string // entity member holding the items ("" = not shown on the entity)
	updKey   string // Update member holding one item
	listOp   string
	listKey  string
	predType string // Predicate.Type that references this condition
	arnType  string
	maxItems int
	metric   bool
	taggable bool
}

var wrKinds = []*wrKind{
	{name: "ByteMatchSet", idKey: "ByteMatchSetId", itemsKey: "ByteMatchTuples", updKey: "ByteMatchTuple", listKey: "ByteMatchSets", predType: "ByteMatch", arnType: "bytematchset", maxItems: 10},
	{name: "GeoMatchSet", idKey: "GeoMatchSetId", itemsKey: "GeoMatchConstraints", updKey: "GeoMatchConstraint", listKey: "GeoMatchSets", predType: "GeoMatch", arnType: "geomatchset", maxItems: 50},
	{name: "IPSet", idKey: "IPSetId", itemsKey: "IPSetDescriptors", updKey: "IPSetDescriptor", listKey: "IPSets", predType: "IPMatch", arnType: "ipset", maxItems: 10000},
	{name: "RegexMatchSet", idKey: "RegexMatchSetId", itemsKey: "RegexMatchTuples", updKey: "RegexMatchTuple", listKey: "RegexMatchSets", predType: "RegexMatch", arnType: "regexmatchset", maxItems: 10},
	{name: "RegexPatternSet", idKey: "RegexPatternSetId", itemsKey: "RegexPatternStrings", updKey: "RegexPatternString", listKey: "RegexPatternSets", arnType: "regexpatternset", maxItems: 10},
	{name: "SizeConstraintSet", idKey: "SizeConstraintSetId", itemsKey: "SizeConstraints", updKey: "SizeConstraint", listKey: "SizeConstraintSets", predType: "SizeConstraint", arnType: "sizeconstraintset", maxItems: 10},
	{name: "SqlInjectionMatchSet", idKey: "SqlInjectionMatchSetId", itemsKey: "SqlInjectionMatchTuples", updKey: "SqlInjectionMatchTuple", listKey: "SqlInjectionMatchSets", predType: "SqlInjectionMatch", arnType: "sqlinjectionset", maxItems: 10},
	{name: "XssMatchSet", idKey: "XssMatchSetId", itemsKey: "XssMatchTuples", updKey: "XssMatchTuple", listKey: "XssMatchSets", predType: "XssMatch", arnType: "xssmatchset", maxItems: 10},
	{name: "Rule", idKey: "RuleId", itemsKey: "Predicates", updKey: "Predicate", listKey: "Rules", arnType: "rule", maxItems: 10, metric: true, taggable: true},
	{name: "RateBasedRule", objKey: "Rule", idKey: "RuleId", itemsKey: "MatchPredicates", updKey: "Predicate", listOp: "ListRateBasedRules", listKey: "Rules", arnType: "ratebasedrule", maxItems: 10, metric: true, taggable: true},
	{name: "RuleGroup", idKey: "RuleGroupId", updKey: "ActivatedRule", listKey: "RuleGroups", arnType: "rulegroup", maxItems: 10, metric: true, taggable: true},
	{name: "WebACL", idKey: "WebACLId", itemsKey: "Rules", updKey: "ActivatedRule", listKey: "WebACLs", arnType: "webacl", maxItems: 10, metric: true, taggable: true},
}

var wrKindByName, wrKindByPred, wrKindByARNType = func() (map[string]*wrKind, map[string]*wrKind, map[string]*wrKind) {
	byName, byPred, byARN := map[string]*wrKind{}, map[string]*wrKind{}, map[string]*wrKind{}
	for _, k := range wrKinds {
		if k.objKey == "" {
			k.objKey = k.name
		}
		if k.listOp == "" {
			k.listOp = "List" + k.listKey
		}
		byName[k.name] = k
		byARN[k.arnType] = k
		if k.predType != "" {
			byPred[k.predType] = k
		}
	}
	return byName, byPred, byARN
}()

// wrRuleTargets maps ActivatedRule.Type to the kind its RuleId names.
var wrRuleTargets = map[string]string{"REGULAR": "Rule", "RATE_BASED": "RateBasedRule", "GROUP": "RuleGroup"}

type wrRes struct {
	kind             *wrKind
	id, name, metric string
	seq              int64
	items            []any
	extra            map[string]any // DefaultAction, RateKey, RateLimit
}

func (r *wrRes) arn() string {
	return fmt.Sprintf("arn:aws:waf-regional:%s:%s:%s/%s", wrRegion, wrAccount, r.kind.arnType, r.id)
}

func (r *wrRes) render() map[string]any {
	out := map[string]any{r.kind.idKey: r.id, "Name": r.name}
	if r.kind.metric {
		out["MetricName"] = r.metric
	}
	if r.kind.itemsKey != "" {
		out[r.kind.itemsKey] = append([]any{}, r.items...)
	}
	for k, v := range r.extra {
		out[k] = v
	}
	if r.kind.name == "WebACL" {
		out["WebACLArn"] = r.arn()
	}
	return out
}

type wafRegionalStore struct {
	mu         sync.Mutex
	seq        int64
	tokens     map[string]string // change token -> PROVISIONED / PENDING / INSYNC
	tokenOrder []string
	res        map[string]map[string]*wrRes // kind name -> id -> entity
	assoc      map[string]string            // resource ARN -> WebACL id
	logging    map[string]map[string]any    // WebACL ARN -> LoggingConfiguration
	policies   map[string]string            // RuleGroup ARN -> policy
	tags       map[string]map[string]string // ARN -> tags
}

func newWAFRegionalStore() *wafRegionalStore {
	st := &wafRegionalStore{
		tokens: map[string]string{}, res: map[string]map[string]*wrRes{}, assoc: map[string]string{},
		logging: map[string]map[string]any{}, policies: map[string]string{}, tags: map[string]map[string]string{},
	}
	for _, k := range wrKinds {
		st.res[k.name] = map[string]*wrRes{}
	}
	return st
}

var wafRegionalHandlers = map[string]wrHandler{
	"GetChangeToken":                (*wafRegionalStore).getChangeToken,
	"GetChangeTokenStatus":          (*wafRegionalStore).getChangeTokenStatus,
	"AssociateWebACL":               (*wafRegionalStore).associateWebACL,
	"DisassociateWebACL":            (*wafRegionalStore).disassociateWebACL,
	"GetWebACLForResource":          (*wafRegionalStore).getWebACLForResource,
	"ListResourcesForWebACL":        (*wafRegionalStore).listResourcesForWebACL,
	"PutLoggingConfiguration":       (*wafRegionalStore).putLoggingConfiguration,
	"GetLoggingConfiguration":       (*wafRegionalStore).getLoggingConfiguration,
	"DeleteLoggingConfiguration":    (*wafRegionalStore).deleteLoggingConfiguration,
	"ListLoggingConfigurations":     (*wafRegionalStore).listLoggingConfigurations,
	"PutPermissionPolicy":           (*wafRegionalStore).putPermissionPolicy,
	"GetPermissionPolicy":           (*wafRegionalStore).getPermissionPolicy,
	"DeletePermissionPolicy":        (*wafRegionalStore).deletePermissionPolicy,
	"TagResource":                   (*wafRegionalStore).tagResource,
	"UntagResource":                 (*wafRegionalStore).untagResource,
	"ListTagsForResource":           (*wafRegionalStore).listTagsForResource,
	"GetSampledRequests":            (*wafRegionalStore).getSampledRequests,
	"GetRateBasedRuleManagedKeys":   (*wafRegionalStore).getRateBasedRuleManagedKeys,
	"ListSubscribedRuleGroups":      (*wafRegionalStore).listSubscribedRuleGroups,
	"ListActivatedRulesInRuleGroup": (*wafRegionalStore).listActivatedRulesInRuleGroup,
}

func init() {
	for _, k := range wrKinds {
		k := k
		wafRegionalHandlers["Create"+k.name] = func(st *wafRegionalStore, in map[string]any) (map[string]any, *wrErr) { return st.create(k, in) }
		wafRegionalHandlers["Get"+k.name] = func(st *wafRegionalStore, in map[string]any) (map[string]any, *wrErr) { return st.get(k, in) }
		wafRegionalHandlers["Update"+k.name] = func(st *wafRegionalStore, in map[string]any) (map[string]any, *wrErr) { return st.update(k, in) }
		wafRegionalHandlers["Delete"+k.name] = func(st *wafRegionalStore, in map[string]any) (map[string]any, *wrErr) { return st.remove(k, in) }
		wafRegionalHandlers[k.listOp] = func(st *wafRegionalStore, in map[string]any) (map[string]any, *wrErr) { return st.list(k, in) }
	}
}

func wrStr(v any) string { s, _ := v.(string); return s }

// ---- change tokens ----

func (st *wafRegionalStore) getChangeToken(map[string]any) (map[string]any, *wrErr) {
	if len(st.tokenOrder) >= wrMaxTokens {
		delete(st.tokens, st.tokenOrder[0])
		st.tokenOrder = st.tokenOrder[1:]
	}
	tok := uuid.NewString()
	st.tokens[tok] = "PROVISIONED"
	st.tokenOrder = append(st.tokenOrder, tok)
	return map[string]any{"ChangeToken": tok}, nil
}

// getChangeTokenStatus reports a used token as PENDING once, then INSYNC.
func (st *wafRegionalStore) getChangeTokenStatus(in map[string]any) (map[string]any, *wrErr) {
	tok := wrStr(in["ChangeToken"])
	status, ok := st.tokens[tok]
	if !ok {
		return nil, wrMissing("change token", tok)
	}
	if status == "PENDING" {
		st.tokens[tok] = "INSYNC"
	}
	return map[string]any{"ChangeTokenStatus": status}, nil
}

// checkToken returns the request's change token if it is issued and unused.
// The token is only consumed once the change succeeds.
func (st *wafRegionalStore) checkToken(in map[string]any) (string, *wrErr) {
	tok := wrStr(in["ChangeToken"])
	if st.tokens[tok] != "PROVISIONED" {
		return "", wrError("WAFStaleDataException", "The input token is no longer current; get a new one with GetChangeToken.")
	}
	return tok, nil
}

func (st *wafRegionalStore) consume(tok string) { st.tokens[tok] = "PENDING" }

// ---- entities ----

var wrMetricRe = regexp.MustCompile(`^[A-Za-z0-9]{1,128}$`)

func (st *wafRegionalStore) find(k *wrKind, id string) (*wrRes, *wrErr) {
	if r := st.res[k.name][id]; r != nil {
		return r, nil
	}
	return nil, wrMissing(k.name, id)
}

func (st *wafRegionalStore) create(k *wrKind, in map[string]any) (map[string]any, *wrErr) {
	tok, err := st.checkToken(in)
	if err != nil {
		return nil, err
	}
	quota := wrMaxEntities
	if k.name == "WebACL" {
		quota = wrMaxWebACLs
	}
	if len(st.res[k.name]) >= quota {
		return nil, wrError("WAFLimitsExceededException", "An account can have at most %d %s objects.", quota, k.name)
	}
	r := &wrRes{kind: k, id: uuid.NewString(), name: wrStr(in["Name"]), extra: map[string]any{}}
	if k.metric {
		r.metric = wrStr(in["MetricName"])
		if !wrMetricRe.MatchString(r.metric) || strings.EqualFold(r.metric, "All") {
			return nil, wrError("WAFDisallowedNameException", "MetricName must be 1-128 letters or digits and can't be \"All\".")
		}
	}
	switch k.name {
	case "WebACL":
		r.extra["DefaultAction"] = in["DefaultAction"]
	case "RateBasedRule":
		r.extra["RateKey"], r.extra["RateLimit"] = in["RateKey"], in["RateLimit"]
	}
	var tags map[string]string
	if k.taggable {
		if tags, err = wrApplyTags(nil, in["Tags"]); err != nil {
			return nil, err
		}
	}
	st.seq++
	r.seq = st.seq
	st.res[k.name][r.id] = r
	if len(tags) > 0 {
		st.tags[r.arn()] = tags
	}
	st.consume(tok)
	return map[string]any{k.objKey: r.render(), "ChangeToken": tok}, nil
}

func (st *wafRegionalStore) get(k *wrKind, in map[string]any) (map[string]any, *wrErr) {
	r, err := st.find(k, wrStr(in[k.idKey]))
	if err != nil {
		return nil, err
	}
	return map[string]any{k.objKey: r.render()}, nil
}

// update applies every INSERT/DELETE to a copy of the items and stores the
// result only if all of them succeed.
func (st *wafRegionalStore) update(k *wrKind, in map[string]any) (map[string]any, *wrErr) {
	tok, err := st.checkToken(in)
	if err != nil {
		return nil, err
	}
	r, err := st.find(k, wrStr(in[k.idKey]))
	if err != nil {
		return nil, err
	}
	updates, _ := in["Updates"].([]any)
	if len(updates) == 0 && k.name != "WebACL" {
		return nil, wrError("WAFInvalidOperationException", "Updates must contain at least one change.")
	}
	items := append([]any(nil), r.items...)
	for _, u := range updates {
		upd, _ := u.(map[string]any)
		item := upd[k.updKey]
		if item == nil {
			return nil, wrError("WAFInvalidParameterException", "Each update needs a %s.", k.updKey)
		}
		if err := st.checkItem(k, item); err != nil {
			return nil, err
		}
		idx := wrIndex(k, items, item)
		switch wrStr(upd["Action"]) {
		case "INSERT":
			if idx >= 0 {
				return nil, wrError("WAFInvalidOperationException", "The %s already contains that %s.", k.name, k.updKey)
			}
			if len(items) >= k.maxItems {
				return nil, wrError("WAFLimitsExceededException", "A %s can hold at most %d items.", k.name, k.maxItems)
			}
			items = append(items, item)
		case "DELETE":
			if idx < 0 {
				return nil, wrError("WAFNonexistentItemException", "The %s doesn't contain that %s.", k.name, k.updKey)
			}
			items = append(items[:idx:idx], items[idx+1:]...)
		}
	}
	if k.updKey == "ActivatedRule" {
		seen := map[float64]bool{}
		for _, it := range items {
			p, _ := it.(map[string]any)["Priority"].(float64)
			if seen[p] {
				return nil, wrError("WAFInvalidParameterException", "Rule priorities in a %s must be unique.", k.name)
			}
			seen[p] = true
		}
	}
	r.items = items
	switch k.name {
	case "WebACL":
		if in["DefaultAction"] != nil {
			r.extra["DefaultAction"] = in["DefaultAction"]
		}
	case "RateBasedRule":
		r.extra["RateLimit"] = in["RateLimit"]
	}
	st.consume(tok)
	return map[string]any{"ChangeToken": tok}, nil
}

func (st *wafRegionalStore) remove(k *wrKind, in map[string]any) (map[string]any, *wrErr) {
	tok, err := st.checkToken(in)
	if err != nil {
		return nil, err
	}
	r, err := st.find(k, wrStr(in[k.idKey]))
	if err != nil {
		return nil, err
	}
	if len(r.items) > 0 {
		return nil, wrError("WAFNonEmptyEntityException", "Remove the items from %s %s before deleting it.", k.name, r.id)
	}
	if by := st.referencedBy(r); by != "" {
		return nil, wrError("WAFReferencedItemException", "%s %s is still used by %s.", k.name, r.id, by)
	}
	delete(st.res[k.name], r.id)
	delete(st.tags, r.arn())
	delete(st.logging, r.arn())
	delete(st.policies, r.arn())
	st.consume(tok)
	return map[string]any{"ChangeToken": tok}, nil
}

func (st *wafRegionalStore) sorted(k *wrKind) []*wrRes {
	out := make([]*wrRes, 0, len(st.res[k.name]))
	for _, r := range st.res[k.name] {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].seq < out[j].seq })
	return out
}

func (st *wafRegionalStore) list(k *wrKind, in map[string]any) (map[string]any, *wrErr) {
	var summaries []any
	for _, r := range st.sorted(k) {
		summaries = append(summaries, map[string]any{k.idKey: r.id, "Name": r.name})
	}
	page, next, err := wrPage(summaries, in)
	if err != nil {
		return nil, err
	}
	return wrPaged(map[string]any{k.listKey: page}, next), nil
}

// referencedBy names an entity that still uses r, or "".
func (st *wafRegionalStore) referencedBy(r *wrRes) string {
	k := r.kind
	switch {
	case k.predType != "":
		for _, owner := range []string{"Rule", "RateBasedRule"} {
			for _, o := range st.res[owner] {
				for _, p := range o.items {
					m, _ := p.(map[string]any)
					if wrStr(m["Type"]) == k.predType && wrStr(m["DataId"]) == r.id {
						return owner + " " + o.id
					}
				}
			}
		}
	case k.name == "RegexPatternSet":
		for _, o := range st.res["RegexMatchSet"] {
			for _, t := range o.items {
				if m, _ := t.(map[string]any); wrStr(m["RegexPatternSetId"]) == r.id {
					return "RegexMatchSet " + o.id
				}
			}
		}
	case k.name == "WebACL":
		for arn, id := range st.assoc {
			if id == r.id {
				return arn
			}
		}
	default: // Rule, RateBasedRule, RuleGroup: activated in a WebACL or RuleGroup
		for _, owner := range []string{"WebACL", "RuleGroup"} {
			for _, o := range st.res[owner] {
				for _, a := range o.items {
					m, _ := a.(map[string]any)
					if wrStr(m["RuleId"]) == r.id && wrRuleTargets[wrRuleType(m)] == k.name {
						return owner + " " + o.id
					}
				}
			}
		}
	}
	return ""
}

func wrRuleType(activated map[string]any) string {
	if t := wrStr(activated["Type"]); t != "" {
		return t
	}
	return "REGULAR"
}

// wrIndex finds item in items: activated rules match by RuleId, everything
// else by value.
func wrIndex(k *wrKind, items []any, item any) int {
	key := wrItemKey(k, item)
	for i, it := range items {
		if wrItemKey(k, it) == key {
			return i
		}
	}
	return -1
}

func wrItemKey(k *wrKind, item any) string {
	if k.updKey == "ActivatedRule" {
		m, _ := item.(map[string]any)
		return wrStr(m["RuleId"])
	}
	b, _ := json.Marshal(item) // map keys marshal sorted, so equal values compare equal
	return string(b)
}

// checkItem validates one item an Update inserts or deletes.
func (st *wafRegionalStore) checkItem(k *wrKind, item any) *wrErr {
	m, _ := item.(map[string]any)
	switch k.updKey {
	case "IPSetDescriptor":
		return wrCheckCIDR(wrStr(m["Type"]), wrStr(m["Value"]))
	case "RegexPatternString":
		if _, err := regexp.Compile(wrStr(item)); err != nil {
			return wrError("WAFInvalidRegexPatternException", "The regular expression is invalid: %v", err)
		}
	case "RegexMatchTuple":
		if err := wrCheckField(m["FieldToMatch"]); err != nil {
			return err
		}
		if id := wrStr(m["RegexPatternSetId"]); st.res["RegexPatternSet"][id] == nil {
			return wrMissing("RegexPatternSet", id)
		}
	case "ByteMatchTuple":
		if err := wrCheckField(m["FieldToMatch"]); err != nil {
			return err
		}
		target, err := base64.StdEncoding.DecodeString(wrStr(m["TargetString"]))
		if err != nil || len(target) > wrMaxTargetLen {
			return wrParam("BYTE_MATCH_FIELD_TYPE", "TargetString", "ILLEGAL_ARGUMENT", "TargetString must be base64 of at most %d bytes.", wrMaxTargetLen)
		}
	case "SizeConstraint", "SqlInjectionMatchTuple", "XssMatchTuple":
		return wrCheckField(m["FieldToMatch"])
	case "Predicate":
		pk := wrKindByPred[wrStr(m["Type"])]
		if pk == nil || st.res[pk.name][wrStr(m["DataId"])] == nil {
			return wrMissing(wrStr(m["Type"])+" condition", wrStr(m["DataId"]))
		}
	case "ActivatedRule":
		return st.checkActivated(k, m)
	}
	return nil
}

func (st *wafRegionalStore) checkActivated(k *wrKind, m map[string]any) *wrErr {
	typ := wrRuleType(m)
	if k.name == "RuleGroup" && typ != "REGULAR" {
		return wrParam("RULE_TYPE", typ, "INVALID_OPTION", "A rule group can only contain REGULAR rules.")
	}
	if target := wrRuleTargets[typ]; st.res[target][wrStr(m["RuleId"])] == nil {
		return wrMissing(target, wrStr(m["RuleId"]))
	}
	group := typ == "GROUP"
	switch {
	case group && (m["OverrideAction"] == nil || m["Action"] != nil):
		return wrParam("WAF_OVERRIDE_ACTION", "OverrideAction", "ILLEGAL_COMBINATION", "A GROUP rule takes an OverrideAction and no Action.")
	case !group && (m["Action"] == nil || m["OverrideAction"] != nil):
		return wrParam("WAF_ACTION", "Action", "ILLEGAL_COMBINATION", "A %s rule takes an Action and no OverrideAction.", typ)
	case !group && m["ExcludedRules"] != nil:
		return wrParam("RULE_TYPE", "ExcludedRules", "ILLEGAL_COMBINATION", "ExcludedRules only applies to GROUP rules.")
	}
	return nil
}

// wrCheckField requires Data for the field types that name a header or argument.
func wrCheckField(v any) *wrErr {
	f, _ := v.(map[string]any)
	if t := wrStr(f["Type"]); (t == "HEADER" || t == "SINGLE_QUERY_ARG") && wrStr(f["Data"]) == "" {
		return wrParam("BYTE_MATCH_FIELD_TYPE", "FieldToMatch.Data", "ILLEGAL_COMBINATION", "FieldToMatch of type %s needs Data.", t)
	}
	return nil
}

// wrCheckCIDR enforces WAF Classic's supported ranges: IPv4 /8 or /16-/32,
// IPv6 /24, /32, /48, /56, /64 or /128.
func wrCheckCIDR(typ, value string) *wrErr {
	ip, ipnet, err := net.ParseCIDR(value)
	bad := func() *wrErr {
		return wrParam("IPSET_TYPE", value, "ILLEGAL_ARGUMENT", "%s is not a supported %s range.", value, typ)
	}
	if err != nil || (typ == "IPV4") != (ip.To4() != nil) {
		return bad()
	}
	ones, _ := ipnet.Mask.Size()
	if typ == "IPV4" && ones != 8 && ones < 16 {
		return bad()
	}
	if typ == "IPV6" && !strings.Contains(" 24 32 48 56 64 128 ", " "+strconv.Itoa(ones)+" ") {
		return bad()
	}
	return nil
}

// ---- pagination and tags ----

// wrPage returns one page of items. NextMarker is the offset of the next item.
func wrPage(items []any, in map[string]any) ([]any, string, *wrErr) {
	limit := wrDefaultLimit
	if n, ok := in["Limit"].(float64); ok && n > 0 {
		limit = int(n)
	}
	start := 0
	if m := wrStr(in["NextMarker"]); m != "" {
		n, err := strconv.ParseInt(m, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, "", wrParam("NEXT_MARKER", m, "ILLEGAL_ARGUMENT", "NextMarker is invalid.")
		}
		start = int(n)
	}
	end := min(start+limit, len(items))
	page := append([]any{}, items[start:end]...)
	if end < len(items) {
		return page, strconv.Itoa(end), nil
	}
	return page, "", nil
}

func wrPaged(out map[string]any, next string) map[string]any {
	if next != "" {
		out["NextMarker"] = next
	}
	return out
}

// wrApplyTags merges a TagList into a copy of base.
func wrApplyTags(base map[string]string, list any) (map[string]string, *wrErr) {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	items, _ := list.([]any)
	for _, t := range items {
		m, _ := t.(map[string]any)
		key := wrStr(m["Key"])
		if strings.HasPrefix(strings.ToLower(key), "aws:") {
			return nil, wrParam("TAGS", key, "INVALID_TAG_KEY", "Tag keys can't start with aws:.")
		}
		out[key] = wrStr(m["Value"])
	}
	if len(out) > wrMaxTags {
		return nil, wrError("WAFLimitsExceededException", "A resource can have at most %d tags.", wrMaxTags)
	}
	return out, nil
}
