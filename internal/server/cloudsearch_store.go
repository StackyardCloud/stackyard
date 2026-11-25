package server

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// CloudSearch domains and their configuration. Index fields, expressions,
// analysis schemes and suggesters are defined as RequiresIndexDocuments and
// become Active (deployed) on IndexDocuments; deletes are pending until then.
// The CloudSearch Domain data plane (cloudsearchdomain*.go) searches the
// deployed configuration and keeps its documents on the same domain records.

const (
	csMaxDomains        = 100
	csMaxIndexFields    = 200
	csMaxExpressions    = 50
	csMaxSchemes        = 50
	csMaxSuggesters     = 10
	csMaxPolicyBytes    = 100 << 10
	csMaxReplicas       = 5
	csMaxPartitions     = 10
	csDefaultInstance   = "search.small"
	csLargestInstance   = "search.2xlarge"
	csStateActive       = "Active"
	csStateRequiresIdx  = "RequiresIndexDocuments"
	csDefaultTLSPolicy  = "Policy-Min-TLS-1-0-2019-07"
	csDomainIDAlphabet  = "abcdefghijklmnopqrstuvwxyz0123456789"
	csDomainIDLength    = 26
	csDefaultSchemeTmpl = "_%s_default_"
)

// csOption is one versioned configuration item. options is never mutated in
// place: a redefinition stores a new value, so responses may share it.
type csOption struct {
	options          any
	deployed         any // nil until the first IndexDocuments
	created, updated time.Time
	version          int
	state            string
	pendingDeletion  bool
}

func (o *csOption) status(deployed bool) map[string]any {
	opts, state, pending := o.options, o.state, o.pendingDeletion
	if deployed {
		opts, state, pending = o.deployed, csStateActive, false
	}
	return map[string]any{
		"Options": opts,
		"Status": map[string]any{
			"CreationDate": o.created, "UpdateDate": o.updated, "UpdateVersion": o.version,
			"State": state, "PendingDeletion": pending,
		},
	}
}

// csImmediate is an option applied at once (availability, endpoint, scaling,
// access policies): it is Active as soon as it is updated.
func csImmediate(v any, now time.Time) *csOption {
	return &csOption{options: v, deployed: v, created: now, updated: now, state: csStateActive}
}

func (o *csOption) setImmediate(v any, now time.Time) {
	o.options, o.deployed, o.updated, o.state = v, v, now, csStateActive
	o.version++
}

type csDoc struct {
	fields map[string][]string
	size   int
}

type csDomain struct {
	name, id      string
	created       time.Time
	requiresIndex bool
	fields        map[string]*csOption
	expressions   map[string]*csOption
	schemes       map[string]*csOption
	suggesters    map[string]*csOption
	availability  *csOption // bool
	endpoint      *csOption // DomainEndpointOptions
	scaling       *csOption // ScalingParameters
	policies      *csOption // policy document string
	docs          map[string]*csDoc
	docBytes      int
}

func (d *csDomain) arn() string {
	return fmt.Sprintf("arn:aws:cloudsearch:%s:%s:domain/%s", csRegion, csAccountID, d.name)
}

func (d *csDomain) host(kind string) string {
	return fmt.Sprintf("%s-%s-%s.%s.cloudsearch.amazonaws.com", kind, d.name, d.id, csRegion)
}

func (d *csDomain) status() map[string]any {
	scaling, _ := d.scaling.options.(map[string]any)
	instance := csStr(scaling, "DesiredInstanceType")
	if instance == "" {
		instance = csDefaultInstance
	}
	partitions, replicas := csIntOr(scaling, "DesiredPartitionCount", 1), csIntOr(scaling, "DesiredReplicationCount", 1)
	if multiAZ, _ := d.availability.options.(bool); multiAZ {
		replicas *= 2
	}
	return map[string]any{
		"DomainId": d.id, "DomainName": d.name, "ARN": d.arn(), "Created": true, "Deleted": false,
		"DocService":             map[string]any{"Endpoint": d.host("doc")},
		"SearchService":          map[string]any{"Endpoint": d.host("search")},
		"RequiresIndexDocuments": d.requiresIndex, "Processing": false,
		"SearchInstanceType": instance, "SearchPartitionCount": partitions, "SearchInstanceCount": replicas,
		"Limits": map[string]any{"MaximumReplicationCount": csMaxReplicas, "MaximumPartitionCount": csMaxPartitions},
	}
}

func csIntOr(m map[string]any, key string, def int) int {
	if f, ok := m[key].(float64); ok && f > 0 {
		return int(f)
	}
	return def
}

type csStore struct {
	mu      sync.RWMutex
	domains map[string]*csDomain
}

func newCloudSearchStore() *csStore { return &csStore{domains: map[string]*csDomain{}} }

func csNewDomainID() string {
	b := make([]byte, csDomainIDLength)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = csDomainIDAlphabet[int(b[i])%len(csDomainIDAlphabet)]
	}
	return string(b)
}

// domain returns the named domain; the caller holds mu.
func (st *csStore) domain(in map[string]any) (*csDomain, *csErr) {
	name := csStr(in, "DomainName")
	d := st.domains[name]
	if d == nil {
		return nil, csNotFound("Domain not found: %s", name)
	}
	return d, nil
}

// ---- domains ----

func (st *csStore) createDomain(in map[string]any) (map[string]any, *csErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	name := csStr(in, "DomainName")
	if st.domains[name] != nil {
		return nil, &csErr{409, "ResourceAlreadyExists", "Domain already exists: " + name}
	}
	if len(st.domains) >= csMaxDomains {
		return nil, csLimit("You can create at most %d domains.", csMaxDomains)
	}
	now := time.Now().UTC()
	d := &csDomain{
		name: name, id: csNewDomainID(), created: now,
		fields: map[string]*csOption{}, expressions: map[string]*csOption{},
		schemes: map[string]*csOption{}, suggesters: map[string]*csOption{},
		availability: csImmediate(false, now),
		endpoint:     csImmediate(map[string]any{"EnforceHTTPS": false, "TLSSecurityPolicy": csDefaultTLSPolicy}, now),
		scaling:      csImmediate(map[string]any{}, now),
		policies:     csImmediate("", now),
		docs:         map[string]*csDoc{},
	}
	st.domains[name] = d
	return map[string]any{"DomainStatus": d.status()}, nil
}

func (st *csStore) deleteDomain(in map[string]any) (map[string]any, *csErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d := st.domains[csStr(in, "DomainName")]
	if d == nil {
		return map[string]any{}, nil // deleting a missing domain is a no-op, as in AWS
	}
	delete(st.domains, d.name)
	status := d.status()
	status["Deleted"] = true
	return map[string]any{"DomainStatus": status}, nil
}

func (st *csStore) sortedDomains(names []string) []*csDomain {
	var out []*csDomain
	if len(names) == 0 {
		for _, d := range st.domains {
			out = append(out, d)
		}
	} else {
		seen := map[string]bool{}
		for _, n := range names {
			if d := st.domains[n]; d != nil && !seen[n] {
				seen[n] = true
				out = append(out, d)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func (st *csStore) describeDomains(in map[string]any) (map[string]any, *csErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	list := []any{}
	for _, d := range st.sortedDomains(csStrList(in, "DomainNames")) {
		list = append(list, d.status())
	}
	return map[string]any{"DomainStatusList": list}, nil
}

func (st *csStore) listDomainNames(map[string]any) (map[string]any, *csErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	names := map[string]any{}
	for name := range st.domains {
		names[name] = csVersion
	}
	return map[string]any{"DomainNames": names}, nil
}

// indexDocuments deploys every pending definition and drops pending deletes.
func (st *csStore) indexDocuments(in map[string]any) (map[string]any, *csErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domain(in)
	if err != nil {
		return nil, err
	}
	for _, set := range []map[string]*csOption{d.fields, d.expressions, d.schemes, d.suggesters} {
		for name, o := range set {
			if o.pendingDeletion {
				delete(set, name)
				continue
			}
			o.deployed, o.state = o.options, csStateActive
		}
	}
	d.requiresIndex = false
	return map[string]any{"FieldNames": csSortedNames(d.fields)}, nil
}

func (st *csStore) buildSuggesters(in map[string]any) (map[string]any, *csErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domain(in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"FieldNames": csSortedNames(d.suggesters)}, nil
}

func csSortedNames(set map[string]*csOption) []any {
	names := make([]string, 0, len(set))
	for n := range set {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]any, len(names))
	for i, n := range names {
		out[i] = n
	}
	return out
}

// ---- versioned definitions ----

// define stores options under name, as a new version when it already exists.
func (d *csDomain) define(set map[string]*csOption, name string, options map[string]any, max int, kind string) (*csOption, *csErr) {
	now := time.Now().UTC()
	o := set[name]
	if o == nil {
		if len(set) >= max {
			return nil, csLimit("A domain can have at most %d %s.", max, kind)
		}
		o = &csOption{created: now}
		set[name] = o
	}
	o.options, o.updated, o.state, o.pendingDeletion = options, now, csStateRequiresIdx, false
	o.version++
	d.requiresIndex = true
	return o, nil
}

func (d *csDomain) remove(set map[string]*csOption, name, kind string) (*csOption, *csErr) {
	o := set[name]
	if o == nil || o.pendingDeletion {
		return nil, csNotFound("%s not found: %s", kind, name)
	}
	if o.deployed == nil { // never deployed: nothing to undo at the next index
		delete(set, name)
	}
	o.pendingDeletion, o.state, o.updated = true, csStateRequiresIdx, time.Now().UTC()
	d.requiresIndex = true
	return o, nil
}

func csDescribe(set map[string]*csOption, names []string, deployed bool) []any {
	keys := names
	if len(keys) == 0 {
		for n := range set {
			keys = append(keys, n)
		}
	}
	sort.Strings(keys)
	out := []any{}
	for i, n := range keys {
		if i > 0 && keys[i-1] == n {
			continue
		}
		o := set[n]
		if o == nil || (deployed && o.deployed == nil) {
			continue
		}
		out = append(out, o.status(deployed))
	}
	return out
}

func (st *csStore) withDomain(in map[string]any, fn func(d *csDomain) (map[string]any, *csErr)) (map[string]any, *csErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domain(in)
	if err != nil {
		return nil, err
	}
	return fn(d)
}

var csTypeOptions = map[string]string{
	"int": "IntOptions", "double": "DoubleOptions", "literal": "LiteralOptions", "text": "TextOptions",
	"date": "DateOptions", "latlon": "LatLonOptions", "int-array": "IntArrayOptions",
	"double-array": "DoubleArrayOptions", "literal-array": "LiteralArrayOptions",
	"text-array": "TextArrayOptions", "date-array": "DateArrayOptions",
}

var csLanguages = []string{"ar", "bg", "ca", "cs", "da", "de", "el", "en", "es", "eu", "fa", "fi", "fr", "ga", "gl", "he", "hi", "hu", "hy", "id", "it", "ja", "ko", "lv", "mul", "nl", "no", "pt", "ro", "ru", "sv", "th", "tr", "zh-Hans", "zh-Hant"}

// schemeExists reports whether name is a built-in _<lang>_default_ scheme or
// one defined on the domain.
func (d *csDomain) schemeExists(name string) bool {
	for _, lang := range csLanguages {
		if name == fmt.Sprintf(csDefaultSchemeTmpl, lang) {
			return true
		}
	}
	o := d.schemes[name]
	return o != nil && !o.pendingDeletion
}

func (st *csStore) defineIndexField(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		field := csMap(in, "IndexField")
		typ := csStr(field, "IndexFieldType")
		want := csTypeOptions[typ]
		for _, member := range csTypeOptions {
			if member != want && field[member] != nil {
				return nil, csInvalidType("%s can't be set on a field of type %s.", member, typ)
			}
		}
		if opts := csMap(field, want); opts != nil {
			if scheme := csStr(opts, "AnalysisScheme"); scheme != "" && !d.schemeExists(scheme) {
				return nil, csInvalidType("Analysis scheme %s is not defined.", scheme)
			}
		}
		o, err := d.define(d.fields, csStr(field, "IndexFieldName"), field, csMaxIndexFields, "index fields")
		if err != nil {
			return nil, err
		}
		return map[string]any{"IndexField": o.status(false)}, nil
	})
}

func (st *csStore) deleteIndexField(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		o, err := d.remove(d.fields, csStr(in, "IndexFieldName"), "Index field")
		if err != nil {
			return nil, err
		}
		return map[string]any{"IndexField": o.status(false)}, nil
	})
}

func (st *csStore) describeIndexFields(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		deployed, _ := csBool(in, "Deployed")
		return map[string]any{"IndexFields": csDescribe(d.fields, csStrList(in, "FieldNames"), deployed)}, nil
	})
}

func (st *csStore) defineExpression(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		expr := csMap(in, "Expression")
		if err := csCheckExpression(csStr(expr, "ExpressionValue")); err != nil {
			return nil, err
		}
		o, err := d.define(d.expressions, csStr(expr, "ExpressionName"), expr, csMaxExpressions, "expressions")
		if err != nil {
			return nil, err
		}
		return map[string]any{"Expression": o.status(false)}, nil
	})
}

// csCheckExpression rejects unbalanced parentheses, the error AWS reports
// most often for a malformed expression.
func csCheckExpression(v string) *csErr {
	depth := 0
	for _, c := range v {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		}
		if depth < 0 {
			break
		}
	}
	if depth != 0 {
		return csValidation("Expression %q has unbalanced parentheses.", v)
	}
	return nil
}

func (st *csStore) deleteExpression(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		o, err := d.remove(d.expressions, csStr(in, "ExpressionName"), "Expression")
		if err != nil {
			return nil, err
		}
		return map[string]any{"Expression": o.status(false)}, nil
	})
}

func (st *csStore) describeExpressions(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		deployed, _ := csBool(in, "Deployed")
		return map[string]any{"Expressions": csDescribe(d.expressions, csStrList(in, "ExpressionNames"), deployed)}, nil
	})
}

func (st *csStore) defineAnalysisScheme(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		scheme := csMap(in, "AnalysisScheme")
		opts := csMap(scheme, "AnalysisOptions")
		for _, key := range []string{"Synonyms", "Stopwords", "StemmingDictionary", "JapaneseTokenizationDictionary"} {
			if v := csStr(opts, key); v != "" && !json.Valid([]byte(v)) {
				return nil, csValidation("AnalysisOptions.%s must be a JSON document.", key)
			}
		}
		if csStr(opts, "JapaneseTokenizationDictionary") != "" && csStr(scheme, "AnalysisSchemeLanguage") != "ja" {
			return nil, csValidation("JapaneseTokenizationDictionary is only valid for Japanese (ja).")
		}
		o, err := d.define(d.schemes, csStr(scheme, "AnalysisSchemeName"), scheme, csMaxSchemes, "analysis schemes")
		if err != nil {
			return nil, err
		}
		return map[string]any{"AnalysisScheme": o.status(false)}, nil
	})
}

func (st *csStore) deleteAnalysisScheme(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		name := csStr(in, "AnalysisSchemeName")
		for field, o := range d.fields {
			if !o.pendingDeletion && csFieldScheme(o.options) == name {
				return nil, csInvalidType("Analysis scheme %s is used by index field %s.", name, field)
			}
		}
		o, err := d.remove(d.schemes, name, "Analysis scheme")
		if err != nil {
			return nil, err
		}
		return map[string]any{"AnalysisScheme": o.status(false)}, nil
	})
}

func csFieldScheme(options any) string {
	field, _ := options.(map[string]any)
	return csStr(csMap(field, csTypeOptions[csStr(field, "IndexFieldType")]), "AnalysisScheme")
}

func (st *csStore) describeAnalysisSchemes(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		deployed, _ := csBool(in, "Deployed")
		return map[string]any{"AnalysisSchemes": csDescribe(d.schemes, csStrList(in, "AnalysisSchemeNames"), deployed)}, nil
	})
}

func (st *csStore) defineSuggester(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		sug := csMap(in, "Suggester")
		source := csStr(csMap(sug, "DocumentSuggesterOptions"), "SourceField")
		f := d.fields[source]
		typ := ""
		if f != nil && !f.pendingDeletion {
			typ = csStr(f.options.(map[string]any), "IndexFieldType")
		}
		if typ != "text" && typ != "literal" {
			return nil, csInvalidType("Suggester source field %s must be an existing text or literal field.", source)
		}
		o, err := d.define(d.suggesters, csStr(sug, "SuggesterName"), sug, csMaxSuggesters, "suggesters")
		if err != nil {
			return nil, err
		}
		return map[string]any{"Suggester": o.status(false)}, nil
	})
}

func (st *csStore) deleteSuggester(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		o, err := d.remove(d.suggesters, csStr(in, "SuggesterName"), "Suggester")
		if err != nil {
			return nil, err
		}
		return map[string]any{"Suggester": o.status(false)}, nil
	})
}

func (st *csStore) describeSuggesters(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		deployed, _ := csBool(in, "Deployed")
		return map[string]any{"Suggesters": csDescribe(d.suggesters, csStrList(in, "SuggesterNames"), deployed)}, nil
	})
}

// ---- immediate options ----

func (st *csStore) describeAvailabilityOptions(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		return map[string]any{"AvailabilityOptions": d.availability.status(false)}, nil
	})
}

func (st *csStore) updateAvailabilityOptions(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		multiAZ, _ := csBool(in, "MultiAZ")
		d.availability.setImmediate(multiAZ, time.Now().UTC())
		return map[string]any{"AvailabilityOptions": d.availability.status(false)}, nil
	})
}

func (st *csStore) describeDomainEndpointOptions(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		return map[string]any{"DomainEndpointOptions": d.endpoint.status(false)}, nil
	})
}

func (st *csStore) updateDomainEndpointOptions(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		req := csMap(in, "DomainEndpointOptions")
		cur, _ := d.endpoint.options.(map[string]any)
		next := map[string]any{"EnforceHTTPS": cur["EnforceHTTPS"], "TLSSecurityPolicy": cur["TLSSecurityPolicy"]}
		for k, v := range req {
			next[k] = v
		}
		d.endpoint.setImmediate(next, time.Now().UTC())
		return map[string]any{"DomainEndpointOptions": d.endpoint.status(false)}, nil
	})
}

func (st *csStore) describeScalingParameters(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		return map[string]any{"ScalingParameters": d.scaling.status(false)}, nil
	})
}

func (st *csStore) updateScalingParameters(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		p := csMap(in, "ScalingParameters")
		partitions, replicas := csIntOr(p, "DesiredPartitionCount", 1), csIntOr(p, "DesiredReplicationCount", 1)
		if partitions > 1 && csStr(p, "DesiredInstanceType") != csLargestInstance {
			return nil, csValidation("DesiredPartitionCount can only be set with DesiredInstanceType %s.", csLargestInstance)
		}
		if partitions > csMaxPartitions || replicas > csMaxReplicas {
			return nil, csLimit("At most %d partitions and %d replicas are allowed.", csMaxPartitions, csMaxReplicas)
		}
		d.scaling.setImmediate(p, time.Now().UTC())
		return map[string]any{"ScalingParameters": d.scaling.status(false)}, nil
	})
}

func (st *csStore) describeServiceAccessPolicies(in map[string]any) (map[string]any, *csErr) {
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		return map[string]any{"AccessPolicies": d.policies.status(false)}, nil
	})
}

var csPolicyEffectRe = regexp.MustCompile(`^(Allow|Deny)$`)

func (st *csStore) updateServiceAccessPolicies(in map[string]any) (map[string]any, *csErr) {
	policy := strings.TrimSpace(csStr(in, "AccessPolicies"))
	if err := csCheckPolicy(policy); err != nil {
		return nil, err
	}
	return st.withDomain(in, func(d *csDomain) (map[string]any, *csErr) {
		d.policies.setImmediate(policy, time.Now().UTC())
		return map[string]any{"AccessPolicies": d.policies.status(false)}, nil
	})
}

// csCheckPolicy accepts an empty policy or a JSON policy whose statements
// each carry an Allow/Deny effect.
func csCheckPolicy(policy string) *csErr {
	if policy == "" {
		return nil
	}
	if len(policy) > csMaxPolicyBytes {
		return csLimit("Access policies can be at most %d bytes.", csMaxPolicyBytes)
	}
	var doc struct {
		Statement []struct {
			Effect string `json:"Effect"`
		} `json:"Statement"`
	}
	if err := json.Unmarshal([]byte(policy), &doc); err != nil {
		return csValidation("AccessPolicies is not a valid JSON policy document.")
	}
	for _, s := range doc.Statement {
		if !csPolicyEffectRe.MatchString(s.Effect) {
			return csValidation("Each policy statement needs an Effect of Allow or Deny.")
		}
	}
	return nil
}
