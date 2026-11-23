package server

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	svcatAccount = "123456789012"
	svcatRegion  = "us-east-1"

	svcatMaxPortfolios     = 1000
	svcatMaxProducts       = 2000
	svcatMaxArtifacts      = 100 // per product
	svcatMaxConstraints    = 5000
	svcatMaxTagOptions     = 1000
	svcatMaxActions        = 1000
	svcatMaxProvisioned    = 2000
	svcatMaxRecords        = 10000 // oldest settled records are evicted past this
	svcatMaxPlans          = 1000
	svcatMaxTokens         = 20000
	svcatMaxTags           = 50
	svcatMaxPrincipals     = 100 // per portfolio
	svcatMaxShares         = 1000
	svcatMaxBudgets        = 100 // per resource
	svcatMaxTemplate       = 1 << 20
	svcatMaxStackInstances = 2000 // listed per provisioned product
	svcatMaxTemplateDefs   = 200  // parameters / outputs / resources read from one template
)

type svcatError struct{ code, msg string }

func svcatErr(code, format string, args ...any) *svcatError {
	return &svcatError{code: code, msg: fmt.Sprintf(format, args...)}
}

func svcatNotFound(format string, args ...any) *svcatError {
	return svcatErr("ResourceNotFoundException", format, args...)
}

func svcatInvalid(format string, args ...any) *svcatError {
	return svcatErr("InvalidParametersException", format, args...)
}

// svcatTemplate is what Stackyard reads from a provisioning artifact.
type svcatTemplate struct {
	params    []svcatParam
	outputs   []svcatOutputDef
	resources []svcatResource
}

type svcatParam struct {
	key, typ, desc, def string
	hasDefault, noEcho  bool
	allowed             []any
}

type svcatOutputDef struct{ key, desc, value, ref string }

type svcatResource struct{ logical, typ string }

type svcatPortfolio struct {
	id, arn, name, desc, provider string
	created                       time.Time
	tags                          map[string]string
	products                      map[string]bool
	principals                    []svcatPrincipal
	shares                        map[string]*svcatShare // by type|value
	tagOptions, budgets           map[string]bool
}

type svcatPrincipal struct{ arn, typ string }

type svcatShare struct {
	typ, value                                 string
	shareTagOptions, sharePrincipals, accepted bool
}

type svcatProduct struct {
	id, viewID, arn, name, owner, desc, distributor string
	supportDesc, supportEmail, supportURL, typ      string
	created                                         time.Time
	tags                                            map[string]string
	artifacts                                       []*svcatArtifact
	tagOptions, budgets                             map[string]bool
	source                                          map[string]any
}

type svcatArtifact struct {
	id, name, desc, typ, guidance string
	active                        bool
	created                       time.Time
	info                          map[string]string
	tmpl                          *svcatTemplate
}

type svcatConstraint struct{ id, typ, desc, params, portfolio, product string }

type svcatTagOption struct {
	id, key, value string
	active         bool
}

type svcatAction struct {
	id, name, desc, defType string
	def                     map[string]string
	assoc                   map[[2]string]bool // product, artifact
}

type svcatProvisioned struct {
	id, name, arn, status, statusMsg, productID, artifactID, pathID, token string
	lastRecord, lastProvRecord, lastOKRecord, physicalID, stackName        string
	created                                                                time.Time
	params                                                                 map[string]string
	tags                                                                   []any
	props                                                                  map[string]string
	outputs                                                                []any // RecordOutput maps
	engine                                                                 bool  // Terraform / external: waits for Notify*
}

type svcatRecord struct {
	id, ppID, ppName, typ, status, productID, artifactID, pathID string
	created, updated                                             time.Time
	errors, tags, outputs                                        []any
	pending                                                      bool   // settles on the next request
	engine                                                       bool   // settles only when the engine notifies
	settle                                                       func() // runs under the store lock when it settles
	notifyToken                                                  string
}

type svcatPlan struct {
	id, name, typ, status, productID, artifactID, pathID, ppName, ppID string
	created, updated                                                   time.Time
	params, tags, notifications                                        []any
}

type svcatCopy struct{ token, status, target, detail string }

type svcatShareToken struct{ token, portfolio, node, status string }

type serviceCatalogStore struct {
	mu          sync.Mutex
	portfolios  map[string]*svcatPortfolio
	products    map[string]*svcatProduct
	views       map[string]string // product view id -> product id
	constraints map[string]*svcatConstraint
	tagOptions  map[string]*svcatTagOption
	actions     map[string]*svcatAction
	provisioned map[string]*svcatProvisioned
	records     map[string]*svcatRecord
	recordOrder []string
	plans       map[string]*svcatPlan
	copies      map[string]*svcatCopy
	shareTokens map[string]*svcatShareToken
	tokens      map[string]string // op|token -> resource id
	accepted    map[string]string // portfolio id -> share type it was accepted as
	orgAccess   string
	post        []func() // run after the store lock is released
}

func newServiceCatalogStore() *serviceCatalogStore {
	return &serviceCatalogStore{
		portfolios: map[string]*svcatPortfolio{}, products: map[string]*svcatProduct{}, views: map[string]string{},
		constraints: map[string]*svcatConstraint{}, tagOptions: map[string]*svcatTagOption{}, actions: map[string]*svcatAction{},
		provisioned: map[string]*svcatProvisioned{}, records: map[string]*svcatRecord{}, plans: map[string]*svcatPlan{},
		copies: map[string]*svcatCopy{}, shareTokens: map[string]*svcatShareToken{}, tokens: map[string]string{},
		accepted: map[string]string{}, orgAccess: "DISABLED",
	}
}

// svcatPre is what a request needs from other stores, fetched before the
// Service Catalog lock is taken.
type svcatPre struct {
	template    []byte // provisioning artifact body from S3
	templateErr *svcatError
	stackName   string // CloudFormation stack named by a physical id
	stackFound  bool
}

type svcatHandler func(st *serviceCatalogStore, s *Server, in map[string]any, pre *svcatPre) (map[string]any, *svcatError)

var svcatHandlers map[string]svcatHandler

func (st *serviceCatalogStore) handle(s *Server, op, outShape string, in map[string]any) (any, *svcatError) {
	h := svcatHandlers[op]
	if h == nil {
		return nil, svcatInvalid("operation %s has no handler", op)
	}
	pre := s.svcatPrefetch(op, in)
	st.mu.Lock()
	st.settleLocked()
	out, err := h(st, s, in, pre)
	post := st.post
	st.post = nil
	var shaped any = map[string]any{}
	if err == nil && outShape != "" {
		shaped = awsmodels.ServiceCatalogModel().Conform(outShape, out)
	}
	st.mu.Unlock()
	for _, f := range post {
		f()
	}
	if err != nil {
		return nil, err
	}
	return shaped, nil
}

// svcatPrefetch reads S3 objects and CloudFormation stacks a request names.
func (s *Server) svcatPrefetch(op string, in map[string]any) *svcatPre {
	pre := &svcatPre{}
	var info map[string]any
	switch op {
	case "CreateProduct":
		info = svcatMap(svcatMap(in, "ProvisioningArtifactParameters"), "Info")
	case "CreateProvisioningArtifact":
		info = svcatMap(svcatMap(in, "Parameters"), "Info")
	case "ImportAsProvisionedProduct":
		pre.stackName, pre.stackFound = s.svcatFindStack(svcatStr(in, "PhysicalId"))
		return pre
	default:
		return pre
	}
	if u := svcatStr(info, "LoadTemplateFromURL"); u != "" {
		pre.template, pre.templateErr = s.svcatFetchTemplate(u)
	} else if id := svcatStr(info, "ImportFromPhysicalId"); id != "" {
		if pre.stackName, pre.stackFound = s.svcatFindStack(id); !pre.stackFound {
			pre.templateErr = svcatInvalid("ImportFromPhysicalId %s is not a CloudFormation stack", id)
		}
		pre.template = []byte(`{"Resources":{}}`)
	}
	return pre
}

// svcatFetchTemplate loads a template from Stackyard's S3. Virtual-hosted
// (bucket.s3[.region].amazonaws.com/key) and path-style (host/bucket/key)
// URLs are both accepted.
func (s *Server) svcatFetchTemplate(raw string) ([]byte, *svcatError) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil, svcatInvalid("LoadTemplateFromURL %q is not a URL", raw)
	}
	host, path := u.Hostname(), strings.TrimPrefix(u.Path, "/")
	var bucket, key string
	if i := strings.Index(host, ".s3"); i > 0 && strings.HasSuffix(host, ".amazonaws.com") {
		bucket, key = host[:i], path
	} else {
		bucket, key, _ = strings.Cut(path, "/")
	}
	if bucket == "" || key == "" {
		return nil, svcatInvalid("LoadTemplateFromURL %q doesn't name an S3 object", raw)
	}
	obj, err := s.s3.GetObject(bucket, key)
	if err != nil {
		return nil, svcatInvalid("LoadTemplateFromURL: s3://%s/%s could not be read", bucket, key)
	}
	if len(obj.Body) > svcatMaxTemplate {
		return nil, svcatInvalid("template exceeds %d bytes", svcatMaxTemplate)
	}
	return obj.Body, nil
}

// svcatFindStack looks a CloudFormation stack up by ARN or name.
func (s *Server) svcatFindStack(id string) (string, bool) {
	if id == "" {
		return "", false
	}
	s.cloudformation.mu.Lock()
	defer s.cloudformation.mu.Unlock()
	for name, arn := range s.cloudformation.stacks {
		if arn == id || name == id {
			return name, true
		}
	}
	return "", false
}

func (s *Server) svcatPutStack(name, arn string) {
	s.cloudformation.mu.Lock()
	s.cloudformation.stacks[name] = arn
	s.cloudformation.mu.Unlock()
}

func (s *Server) svcatDeleteStack(name string) {
	s.cloudformation.mu.Lock()
	delete(s.cloudformation.stacks, name)
	s.cloudformation.mu.Unlock()
}

// svcatParseTemplate reads parameters, outputs and resources from a
// CloudFormation template. JSON templates are parsed; YAML is accepted when it
// looks like a template (a top-level Resources key) without being parsed.
// ponytail: no YAML library in go.mod, so YAML templates expose no parameters.
func svcatParseTemplate(artifactType string, body []byte, skipValidation bool) (*svcatTemplate, *svcatError) {
	t := &svcatTemplate{}
	if body == nil || artifactType != "CLOUD_FORMATION_TEMPLATE" {
		return t, nil // Terraform / external artifacts are opaque archives
	}
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		text := string(body)
		if skipValidation || strings.HasPrefix(text, "Resources:") || strings.Contains(text, "\nResources:") {
			return t, nil
		}
		return nil, svcatInvalid("the template is neither JSON nor a YAML CloudFormation template")
	}
	res, _ := doc["Resources"].(map[string]any)
	if len(res) == 0 && !skipValidation {
		if _, ok := doc["Resources"]; !ok {
			return nil, svcatInvalid("the template has no Resources section")
		}
	}
	for _, name := range svcatSortedKeys(res) {
		r, _ := res[name].(map[string]any)
		typ, _ := r["Type"].(string)
		t.resources = append(t.resources, svcatResource{logical: name, typ: typ})
	}
	params, _ := doc["Parameters"].(map[string]any)
	for _, name := range svcatSortedKeys(params) {
		p, _ := params[name].(map[string]any)
		sp := svcatParam{key: name}
		sp.typ, _ = p["Type"].(string)
		sp.desc, _ = p["Description"].(string)
		if d, ok := p["Default"]; ok {
			sp.def, sp.hasDefault = fmt.Sprint(d), true
		}
		switch v := p["NoEcho"].(type) {
		case bool:
			sp.noEcho = v
		case string:
			sp.noEcho = strings.EqualFold(v, "true")
		}
		if av, ok := p["AllowedValues"].([]any); ok {
			for _, a := range av {
				sp.allowed = append(sp.allowed, fmt.Sprint(a))
			}
		}
		t.params = append(t.params, sp)
	}
	outs, _ := doc["Outputs"].(map[string]any)
	for _, name := range svcatSortedKeys(outs) {
		o, _ := outs[name].(map[string]any)
		od := svcatOutputDef{key: name}
		od.desc, _ = o["Description"].(string)
		switch v := o["Value"].(type) {
		case string:
			od.value = v
		case map[string]any:
			od.ref, _ = v["Ref"].(string)
		}
		t.outputs = append(t.outputs, od)
	}
	if len(t.params) > svcatMaxTemplateDefs || len(t.outputs) > svcatMaxTemplateDefs || len(t.resources) > svcatMaxTemplateDefs {
		return nil, svcatInvalid("the template has more than %d parameters, outputs or resources", svcatMaxTemplateDefs)
	}
	return t, nil
}

func (t *svcatTemplate) paramShapes() []any {
	out := []any{}
	for _, p := range t.params {
		m := map[string]any{"ParameterKey": p.key, "ParameterType": p.typ, "IsNoEcho": p.noEcho, "Description": p.desc}
		if p.hasDefault {
			m["DefaultValue"] = p.def
		}
		if len(p.allowed) > 0 {
			m["ParameterConstraints"] = map[string]any{"AllowedValues": p.allowed}
		}
		out = append(out, m)
	}
	return out
}

// resolveParams checks provisioning parameters against the template and
// returns the effective values (defaults filled in).
func (t *svcatTemplate) resolveParams(given []any, previous map[string]string) (map[string]string, *svcatError) {
	vals := map[string]string{}
	known := map[string]svcatParam{}
	for _, p := range t.params {
		known[p.key] = p
	}
	for _, g := range given {
		m, _ := g.(map[string]any)
		key := svcatStr(m, "Key")
		p, ok := known[key]
		if !ok && len(t.params) > 0 {
			return nil, svcatInvalid("Parameter %s is not defined in the template", key)
		}
		if use, _ := m["UsePreviousValue"].(bool); use {
			if v, ok := previous[key]; ok {
				vals[key] = v
				continue
			}
		}
		v := svcatStr(m, "Value")
		if len(p.allowed) > 0 && !svcatContains(p.allowed, v) {
			return nil, svcatInvalid("Parameter %s must be one of %v", key, p.allowed)
		}
		vals[key] = v
	}
	for _, p := range t.params {
		if _, ok := vals[p.key]; ok {
			continue
		}
		if v, ok := previous[p.key]; ok {
			vals[p.key] = v
		} else if p.hasDefault {
			vals[p.key] = p.def
		} else {
			return nil, svcatInvalid("Parameter %s must have a value", p.key)
		}
	}
	return vals, nil
}

// outputs resolves template outputs: literal values, or Refs to parameters.
func (t *svcatTemplate) outputValues(params map[string]string, stackID string) []any {
	out := []any{}
	for _, o := range t.outputs {
		v := o.value
		if o.ref != "" {
			if pv, ok := params[o.ref]; ok {
				v = pv
			} else {
				v = o.ref // a Ref to a resource: its logical id stands in for the physical id
			}
		}
		out = append(out, map[string]any{"OutputKey": o.key, "OutputValue": v, "Description": o.desc})
	}
	if stackID != "" {
		out = append(out, map[string]any{"OutputKey": "CloudformationStackARN", "OutputValue": stackID, "Description": "The ARN of the launched CloudFormation stack"})
	}
	return out
}

// ---- tokens, ids, paging, helpers ----

// svcatIdempotent returns the resource an earlier call with this token made.
func (st *serviceCatalogStore) svcatIdempotent(op, token string) (string, bool) {
	if token == "" {
		return "", false
	}
	id, ok := st.tokens[op+"|"+token]
	return id, ok
}

func (st *serviceCatalogStore) rememberToken(op, token, id string) *svcatError {
	if token == "" {
		return nil
	}
	// Bounded: once full, an arbitrary old token is forgotten, so the call never
	// fails after its resource was created.
	if len(st.tokens) >= svcatMaxTokens {
		for k := range st.tokens {
			delete(st.tokens, k)
			break
		}
	}
	st.tokens[op+"|"+token] = id
	return nil
}

const svcatIDChars = "abcdefghijklmnopqrstuvwxyz0123456789"

func svcatNewID(prefix string) string {
	b := make([]byte, 13)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(svcatIDChars))))
		b[i] = svcatIDChars[n.Int64()]
	}
	return prefix + string(b)
}

func svcatPortfolioARN(id string) string {
	return "arn:aws:catalog:" + svcatRegion + ":" + svcatAccount + ":portfolio/" + id
}

func svcatProductARN(id string) string {
	return "arn:aws:catalog:" + svcatRegion + ":" + svcatAccount + ":product/" + id
}

// svcatPage slices items by an offset page token.
func svcatPage(items []any, in map[string]any, defaultSize int) ([]any, string, *svcatError) {
	start := 0
	if tok := svcatStr(in, "PageToken"); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, "", svcatInvalid("PageToken is not valid")
		}
		start = int(n)
	}
	size := defaultSize
	if v, ok := in["PageSize"].(float64); ok && v > 0 {
		size = int(v)
	}
	end := min(start+size, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[start:end], next, nil
}

func svcatStr(in map[string]any, key string) string { v, _ := in[key].(string); return v }

func svcatMap(in map[string]any, key string) map[string]any {
	v, _ := in[key].(map[string]any)
	return v
}

func svcatList(in map[string]any, key string) []any { v, _ := in[key].([]any); return v }

func svcatEpoch(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

func svcatSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func svcatContains(list []any, v string) bool {
	for _, x := range list {
		if fmt.Sprint(x) == v {
			return true
		}
	}
	return false
}

// svcatTags turns stored tags into Tag shapes, sorted by key.
func svcatTags(tags map[string]string) []any {
	out := []any{}
	for _, k := range svcatSortedKeys(tags) {
		out = append(out, map[string]any{"Key": k, "Value": tags[k]})
	}
	return out
}

// svcatApplyTags adds Tag shapes to tags, keeping the AWS limit and refusing
// the reserved aws: prefix.
func svcatApplyTags(tags map[string]string, add []any, remove []any) *svcatError {
	next := make(map[string]string, len(tags))
	for k, v := range tags {
		next[k] = v
	}
	for _, r := range remove {
		delete(next, fmt.Sprint(r))
	}
	for _, a := range add {
		m, _ := a.(map[string]any)
		k := svcatStr(m, "Key")
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return svcatInvalid("Tag keys can't start with aws:")
		}
		next[k] = svcatStr(m, "Value")
	}
	if len(next) > svcatMaxTags {
		return svcatInvalid("a resource can have at most %d tags", svcatMaxTags)
	}
	for k := range tags {
		delete(tags, k)
	}
	for k, v := range next {
		tags[k] = v
	}
	return nil
}

func svcatSet(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
