package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
	"github.com/stackyard/stackyard/internal/services/ec2"
	"github.com/stackyard/stackyard/internal/services/s3"
)

// Amazon Route 53 Resolver: awsJson 1.1, X-Amz-Target Route53Resolver.<Op>,
// SigV4 service "route53resolver". Requests are checked against the embedded
// botocore model before the store sees them; handlers copy only the members
// they model into stored records, and every response is conformed to the
// model's output shape under the store lock (so it never aliases stored maps).

const (
	r53rTargetPrefix = "Route53Resolver."
	r53rMaxBody      = 1 << 20
	r53rAccountID    = "123456789012"
	r53rRegion       = "us-east-1"
	r53rMaxTags      = 200
	r53rDefaultPage  = 100
)

type r53rErr struct {
	code string
	msg  string
}

func r53rE(code, format string, args ...any) *r53rErr {
	return &r53rErr{code: code, msg: fmt.Sprintf(format, args...)}
}

type r53rHandler func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr)

// r53rHandlers is filled by the area files' init functions.
var r53rHandlers = map[string]r53rHandler{}

type r53rStore struct {
	mu  sync.Mutex
	ec2 *ec2.Service
	s3  *s3.Service

	endpoints  map[string]*r53rEndpoint
	rules      map[string]*r53rRule
	ruleAssocs map[string]*r53rRec
	qlcs       map[string]*r53rRec
	qlcAssocs  map[string]*r53rRec
	outposts   map[string]*r53rRec

	resolverConfigs map[string]*r53rRec // by VPC id
	dnssecConfigs   map[string]*r53rRec
	firewallConfigs map[string]*r53rRec

	domainLists map[string]*r53rDomainList
	ruleGroups  map[string]*r53rRec
	fwRules     map[string]*r53rRec // by r53rFWRuleKey
	fwAssocs    map[string]*r53rRec

	tags     map[string]map[string]string // by resource ARN
	policies map[string]string            // by resource ARN
	domains  int                          // domains held across all domain lists
}

// r53rRec is one stored resource: its output members plus a status to report
// after the next read (the transient state is shown once, then settles).
type r53rRec struct {
	rec    map[string]any
	settle string
}

func newRoute53ResolverStore(ec2Svc *ec2.Service, s3Svc *s3.Service) *r53rStore {
	st := &r53rStore{
		ec2: ec2Svc, s3: s3Svc,
		endpoints: map[string]*r53rEndpoint{}, rules: map[string]*r53rRule{},
		ruleAssocs: map[string]*r53rRec{}, qlcs: map[string]*r53rRec{}, qlcAssocs: map[string]*r53rRec{},
		outposts: map[string]*r53rRec{}, resolverConfigs: map[string]*r53rRec{}, dnssecConfigs: map[string]*r53rRec{},
		firewallConfigs: map[string]*r53rRec{}, domainLists: map[string]*r53rDomainList{}, ruleGroups: map[string]*r53rRec{},
		fwRules: map[string]*r53rRec{}, fwAssocs: map[string]*r53rRec{},
		tags: map[string]map[string]string{}, policies: map[string]string{},
	}
	st.seedLocked()
	return st
}

// handle runs action and conforms its result to outShape while holding the
// lock, so responses are fresh maps built from (never aliasing) stored state.
func (st *r53rStore) handle(action, outShape string, in map[string]any) (any, *r53rErr) {
	h := r53rHandlers[action]
	if h == nil {
		return nil, r53rE("InternalServiceErrorException", "operation %s has no handler", action)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	out, e := h(st, in)
	if e != nil {
		return nil, e
	}
	if outShape == "" {
		return map[string]any{}, nil
	}
	return awsmodels.Route53ResolverModel().Conform(outShape, out), nil
}

func (s *Server) handleRoute53ResolverJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost || !strings.HasPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), r53rTargetPrefix) {
		return false
	}
	action := strings.TrimPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), r53rTargetPrefix)
	if _, ok := route53resolverOperationByName[action]; !ok {
		respondRoute53ResolverError(w, http.StatusBadRequest, "UnknownOperationException", "unknown operation "+action)
		return true
	}
	// A declared length bounds what net/http reads, so require one; chunked
	// bodies of unknown length are refused (r.Body is never reassigned).
	if r.ContentLength < 0 || r.ContentLength > r53rMaxBody {
		respondRoute53ResolverError(w, http.StatusBadRequest, "InvalidRequestException", "request body must declare a Content-Length of at most 1 MiB")
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "route53resolver"); !ok {
		respondRoute53ResolverError(w, status, code, msg)
		return true
	}
	body, err := readBodyBytes(r)
	if err != nil {
		respondRoute53ResolverError(w, http.StatusBadRequest, "SerializationException", "unable to read request body")
		return true
	}
	in := map[string]any{}
	if strings.TrimSpace(string(body)) != "" {
		if err := json.Unmarshal(body, &in); err != nil {
			respondRoute53ResolverError(w, http.StatusBadRequest, "SerializationException", "request body is not a JSON object")
			return true
		}
	}
	model := awsmodels.Route53ResolverModel()
	op := model.Operations[action]
	for _, p := range model.Validate(op.Input, in) {
		if strings.Contains(p, "unknown member") {
			continue // handlers only read modelled members
		}
		respondRoute53ResolverError(w, http.StatusBadRequest, r53rValidationCode(action), p)
		return true
	}
	out, e := s.route53resolver.handle(action, op.Output, in)
	if e != nil {
		respondRoute53ResolverError(w, http.StatusBadRequest, e.code, e.msg)
		return true
	}
	raw, err := json.Marshal(out)
	if err != nil {
		respondRoute53ResolverError(w, http.StatusInternalServerError, "InternalServiceErrorException", "unable to encode response")
		return true
	}
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
	return true
}

func respondRoute53ResolverError(w http.ResponseWriter, status int, code, msg string) {
	raw, _ := json.Marshal(map[string]string{"__type": code, "Message": msg})
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

// r53rValidationCode is the model-validation error an operation reports:
// DNS Firewall and Outposts operations use ValidationException, the older
// resolver operations InvalidParameterException.
func r53rValidationCode(action string) string {
	if strings.Contains(action, "Firewall") || strings.Contains(action, "Outpost") {
		return "ValidationException"
	}
	return "InvalidParameterException"
}

// --- small input helpers ---------------------------------------------------

func r53rStr(in map[string]any, key string) string {
	v, _ := in[key].(string)
	return v
}

func r53rHas(in map[string]any, key string) bool {
	v, ok := in[key]
	return ok && v != nil
}

func r53rInt(in map[string]any, key string) (int, bool) {
	f, ok := in[key].(float64)
	if !ok {
		return 0, false
	}
	return int(f), true
}

func r53rList(in map[string]any, key string) []any {
	v, _ := in[key].([]any)
	return v
}

func r53rMap(in map[string]any, key string) map[string]any {
	v, _ := in[key].(map[string]any)
	return v
}

func r53rStrings(in map[string]any, key string) []string {
	var out []string
	for _, v := range r53rList(in, key) {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func r53rNow() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000Z") }

func r53rID(prefix string) string {
	b := make([]byte, 9)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)[:17]
}

func r53rARN(kind, id string) string {
	return "arn:aws:route53resolver:" + r53rRegion + ":" + r53rAccountID + ":" + kind + "/" + id
}

// Name members: letters, digits, space, hyphen, underscore and apostrophe,
// not all digits (the model's pattern uses a lookahead RE2 can't compile).
var r53rNameRE = regexp.MustCompile(`^[a-zA-Z0-9\-_' ]+$`)
var r53rDigitsRE = regexp.MustCompile(`^[0-9]+$`)

func r53rCheckName(in map[string]any, key, code string) *r53rErr {
	if !r53rHas(in, key) {
		return nil
	}
	name := r53rStr(in, key)
	if !r53rNameRE.MatchString(name) || r53rDigitsRE.MatchString(name) {
		return r53rE(code, "%s %q must contain only letters, digits, spaces, hyphens, underscores and apostrophes, and can't be all digits", key, name)
	}
	return nil
}

// copyMembers copies the listed members that are present in src into dst.
func r53rCopy(dst, src map[string]any, keys ...string) {
	for _, k := range keys {
		if r53rHas(src, k) {
			dst[k] = src[k]
		}
	}
}

// --- reads, paging, filters --------------------------------------------------

// view returns rec's output members, settling a transient status first.
func (r *r53rRec) view() map[string]any {
	if r.settle != "" {
		r.rec["Status"] = r.settle
		r.settle = ""
	}
	return r.rec
}

func r53rSorted[T any](m map[string]T, key func(T) string) []T {
	out := make([]T, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return key(out[i]) < key(out[j]) })
	return out
}

// r53rPage slices items by the request's MaxResults and an offset NextToken.
func r53rPage(items []any, in map[string]any, badToken string) ([]any, string, *r53rErr) {
	start := 0
	if tok := r53rStr(in, "NextToken"); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, "", r53rE(badToken, "the NextToken %q is not valid", tok)
		}
		start = int(n)
	}
	size := r53rDefaultPage
	if n, ok := r53rInt(in, "MaxResults"); ok && n > 0 {
		size = n
	}
	end := min(start+size, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[start:end], next, nil
}

// r53rFilter keeps items whose fields match every filter. fields maps each
// filter name the operation accepts to the item member it compares.
func r53rFilter(items []map[string]any, filters []any, fields map[string]string, code string) ([]map[string]any, *r53rErr) {
	out := items
	for _, f := range filters {
		fm, _ := f.(map[string]any)
		name := r53rStr(fm, "Name")
		member, ok := fields[name]
		if !ok {
			return nil, r53rE(code, "the filter name %q is not supported", name)
		}
		values := r53rStrings(fm, "Values")
		var kept []map[string]any
		for _, item := range out {
			if r53rMatches(item[member], values) {
				kept = append(kept, item)
			}
		}
		out = kept
	}
	return out, nil
}

func r53rMatches(v any, values []string) bool {
	if len(values) == 0 {
		return true
	}
	var have []string
	switch t := v.(type) {
	case []any:
		for _, x := range t {
			have = append(have, fmt.Sprint(x))
		}
	case nil:
	default:
		have = []string{fmt.Sprint(t)}
	}
	for _, h := range have {
		for _, want := range values {
			if strings.EqualFold(h, want) {
				return true
			}
		}
	}
	return false
}

func r53rAny(items []map[string]any) []any {
	out := make([]any, len(items))
	for i, v := range items {
		out[i] = v
	}
	return out
}

// --- tags and policies -------------------------------------------------------

func (st *r53rStore) tagsLocked(arn string, tags []any) *r53rErr {
	if len(tags) == 0 {
		return nil
	}
	cur := st.tags[arn]
	next := make(map[string]string, len(cur)+len(tags))
	for k, v := range cur {
		next[k] = v
	}
	for _, t := range tags {
		tm, _ := t.(map[string]any)
		key := r53rStr(tm, "Key")
		if strings.HasPrefix(strings.ToLower(key), "aws:") {
			return r53rE("InvalidTagException", "tag keys can't start with aws:")
		}
		next[key] = r53rStr(tm, "Value")
	}
	if len(next) > r53rMaxTags {
		return r53rE("LimitExceededException", "a resource can have at most %d tags", r53rMaxTags)
	}
	st.tags[arn] = next
	return nil
}

func (st *r53rStore) forgetLocked(arn string) {
	delete(st.tags, arn)
	delete(st.policies, arn)
}

// arnExistsLocked reports whether arn names a taggable resource in the store.
func (st *r53rStore) arnExistsLocked(arn string) bool {
	for _, e := range st.endpoints {
		if e.rec["Arn"] == arn {
			return true
		}
	}
	for _, r := range st.rules {
		if r.rec["Arn"] == arn && !r.autodefined {
			return true
		}
	}
	for _, m := range []map[string]*r53rRec{st.qlcs, st.outposts, st.ruleGroups, st.fwAssocs} {
		for _, r := range m {
			if r.rec["Arn"] == arn {
				return true
			}
		}
	}
	for _, d := range st.domainLists {
		if d.rec["Arn"] == arn && !d.managed {
			return true
		}
	}
	return false
}

// r53rPolicyJSON checks a resource policy is a JSON object with a Statement.
func r53rPolicyJSON(policy string) bool {
	var doc map[string]any
	if json.Unmarshal([]byte(policy), &doc) != nil {
		return false
	}
	_, ok := doc["Statement"]
	return ok
}

func init() {
	r53rHandlers["TagResource"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		arn := r53rStr(in, "ResourceArn")
		if !st.arnExistsLocked(arn) {
			return nil, r53rE("ResourceNotFoundException", "resource %s doesn't exist", arn)
		}
		if e := st.tagsLocked(arn, r53rList(in, "Tags")); e != nil {
			return nil, e
		}
		return map[string]any{}, nil
	}
	r53rHandlers["UntagResource"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		arn := r53rStr(in, "ResourceArn")
		if !st.arnExistsLocked(arn) {
			return nil, r53rE("ResourceNotFoundException", "resource %s doesn't exist", arn)
		}
		cur := st.tags[arn]
		next := make(map[string]string, len(cur))
		for k, v := range cur {
			next[k] = v
		}
		for _, k := range r53rStrings(in, "TagKeys") {
			delete(next, k)
		}
		st.tags[arn] = next
		return map[string]any{}, nil
	}
	r53rHandlers["ListTagsForResource"] = func(st *r53rStore, in map[string]any) (map[string]any, *r53rErr) {
		arn := r53rStr(in, "ResourceArn")
		if !st.arnExistsLocked(arn) {
			return nil, r53rE("ResourceNotFoundException", "resource %s doesn't exist", arn)
		}
		keys := make([]string, 0, len(st.tags[arn]))
		for k := range st.tags[arn] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		items := make([]any, len(keys))
		for i, k := range keys {
			items[i] = map[string]any{"Key": k, "Value": st.tags[arn][k]}
		}
		page, next, e := r53rPage(items, in, "InvalidNextTokenException")
		if e != nil {
			return nil, e
		}
		return map[string]any{"Tags": page, "NextToken": r53rOmit(next)}, nil
	}
}

// r53rOmit turns an empty string into nil so Conform drops the member.
func r53rOmit(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// HasResource reports whether a Resolver rule, query logging configuration or
// DNS Firewall rule group with this id exists (used by Route 53 Profiles).
func (st *r53rStore) HasResource(kind, id string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	switch kind {
	case "RESOLVER_RULE":
		return st.rules[id] != nil
	case "RESOLVER_QUERY_LOGGING_CONFIG":
		return st.qlcs[id] != nil
	case "FIREWALL_RULE_GROUP":
		return st.ruleGroups[id] != nil
	}
	return false
}
