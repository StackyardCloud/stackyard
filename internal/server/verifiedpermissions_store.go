package server

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	vpAccountID          = "123456789012"
	vpRegion             = "us-east-1"
	vpMaxPolicyStores    = 1000
	vpMaxPoliciesPer     = 10000
	vpMaxTemplatesPer    = 40 // AWS quota
	vpMaxIdentitySources = 1  // AWS quota: one identity source per policy store
	vpMaxAliases         = 1000
	vpMaxTags            = 200
	vpDefaultPage        = 50
	vpAliasPrefix        = "policy-store-alias/"
)

type vpErr struct {
	status int
	code   string
	msg    string
	extra  map[string]any
}

func vpValidation(format string, args ...any) *vpErr {
	return &vpErr{status: http.StatusBadRequest, code: "ValidationException", msg: fmt.Sprintf(format, args...)}
}

func vpNotFound(resourceType, id string) *vpErr {
	return &vpErr{status: http.StatusNotFound, code: "ResourceNotFoundException",
		msg:   fmt.Sprintf("%s %s does not exist.", strings.ToLower(strings.ReplaceAll(resourceType, "_", " ")), id),
		extra: map[string]any{"resourceId": id, "resourceType": resourceType}}
}

func vpConflict(resourceType, id, format string, args ...any) *vpErr {
	return &vpErr{status: http.StatusConflict, code: "ConflictException", msg: fmt.Sprintf(format, args...),
		extra: map[string]any{"resources": []any{map[string]any{"resourceId": id, "resourceType": resourceType}}}}
}

func vpQuota(resourceType, format string, args ...any) *vpErr {
	return &vpErr{status: http.StatusPaymentRequired, code: "ServiceQuotaExceededException", msg: fmt.Sprintf(format, args...),
		extra: map[string]any{"resourceType": resourceType}}
}

type vpPolicyStore struct {
	id, arn, description   string
	mode, deletion         string
	encryption             map[string]any // EncryptionState
	created, updated       time.Time
	tags                   map[string]string
	schema                 *vpSchema
	schemaCreated, schemaU time.Time
	policies               map[string]*vpPolicy
	templates              map[string]*vpTemplate
	identitySources        map[string]*vpIdentitySource
}

type vpPolicy struct {
	id, name, description string
	static                bool
	statement             string // static policies
	parsed                *cedarPolicy
	templateID            string // template-linked policies
	principal, resource   *cedarEntity
	created, updated      time.Time
}

type vpTemplate struct {
	id, name, description, statement string
	parsed                           *cedarPolicy
	slots                            map[string]bool
	created, updated                 time.Time
}

type vpIdentitySource struct {
	id, principalType string
	config            map[string]any // the Configuration union as given
	created, updated  time.Time
}

type vpAlias struct {
	name, storeID string
	created       time.Time
}

type vpStore struct {
	mu      sync.Mutex
	stores  map[string]*vpPolicyStore
	aliases map[string]*vpAlias
	tokens  map[string]string // op + clientToken -> resource id
}

func newVerifiedPermissionsStore() *vpStore {
	return &vpStore{stores: map[string]*vpPolicyStore{}, aliases: map[string]*vpAlias{}, tokens: map[string]string{}}
}

const vpIDChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func vpNewID() string {
	b := make([]byte, 22)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(vpIDChars))))
		b[i] = vpIDChars[n.Int64()]
	}
	return string(b)
}

func vpTime(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

func vpStr(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

func vpMap(m map[string]any, key string) map[string]any {
	v, _ := m[key].(map[string]any)
	return v
}

func vpList(m map[string]any, key string) []any {
	v, _ := m[key].([]any)
	return v
}

func vpEntityOf(m map[string]any) *cedarEntity {
	if m == nil {
		return nil
	}
	return &cedarEntity{Type: vpStr(m, "entityType"), ID: vpStr(m, "entityId")}
}

func vpEntityOut(e *cedarEntity) map[string]any {
	if e == nil {
		return nil
	}
	return map[string]any{"entityType": e.Type, "entityId": e.ID}
}

type vpHandler func(st *vpStore, in map[string]any, x vpExtra) (map[string]any, *vpErr)

// vpExtra carries facts the router looked up in other stores before taking
// this store's lock.
type vpExtra struct {
	userPoolExists bool
	now            time.Time
}

var vpHandlers map[string]vpHandler

func init() {
	vpHandlers = map[string]vpHandler{
		"CreatePolicyStore": vpCreatePolicyStore, "GetPolicyStore": vpGetPolicyStore, "UpdatePolicyStore": vpUpdatePolicyStore,
		"DeletePolicyStore": vpDeletePolicyStore, "ListPolicyStores": vpListPolicyStores,
		"PutSchema": vpPutSchema, "GetSchema": vpGetSchema,
		"CreatePolicy": vpCreatePolicy, "GetPolicy": vpGetPolicy, "UpdatePolicy": vpUpdatePolicy, "DeletePolicy": vpDeletePolicy,
		"ListPolicies": vpListPolicies, "BatchGetPolicy": vpBatchGetPolicy,
		"CreatePolicyTemplate": vpCreateTemplate, "GetPolicyTemplate": vpGetTemplate, "UpdatePolicyTemplate": vpUpdateTemplate,
		"DeletePolicyTemplate": vpDeleteTemplate, "ListPolicyTemplates": vpListTemplates,
		"CreateIdentitySource": vpCreateIdentitySource, "GetIdentitySource": vpGetIdentitySource, "UpdateIdentitySource": vpUpdateIdentitySource,
		"DeleteIdentitySource": vpDeleteIdentitySource, "ListIdentitySources": vpListIdentitySources,
		"CreatePolicyStoreAlias": vpCreateAlias, "GetPolicyStoreAlias": vpGetAlias, "DeletePolicyStoreAlias": vpDeleteAlias,
		"ListPolicyStoreAliases": vpListAliases,
		"TagResource":            vpTagResource, "UntagResource": vpUntagResource, "ListTagsForResource": vpListTags,
		"IsAuthorized": vpIsAuthorized, "BatchIsAuthorized": vpBatchIsAuthorized,
		"IsAuthorizedWithToken": vpIsAuthorizedWithToken, "BatchIsAuthorizedWithToken": vpBatchIsAuthorizedWithToken,
	}
}

// handle runs op and conforms its output to the model while holding the lock,
// so nothing read after unlock aliases stored state.
func (st *vpStore) handle(op string, in map[string]any, x vpExtra) (any, *vpErr) {
	h := vpHandlers[op]
	if h == nil {
		return nil, vpValidation("operation %s is not supported", op)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := h(st, in, x)
	if err != nil {
		return nil, err
	}
	model := awsmodels.VerifiedPermissionsModel()
	return model.Conform(model.Operations[op].Output, out), nil
}

// idempotent returns the id an earlier call with the same client token made.
func (st *vpStore) idempotent(op string, in map[string]any) (key, id string) {
	tok := vpStr(in, "clientToken")
	if tok == "" {
		return "", ""
	}
	key = op + "\x00" + vpStr(in, "policyStoreId") + "\x00" + tok
	return key, st.tokens[key]
}

// policyStore resolves a policy store id or alias ("policy-store-alias/name"
// or an alias ARN) to the store.
func (st *vpStore) policyStore(ref string) (*vpPolicyStore, *vpErr) {
	id := ref
	if name, ok := strings.CutPrefix(ref, "arn:aws:verifiedpermissions::"+vpAccountID+":"+vpAliasPrefix); ok {
		ref = vpAliasPrefix + name
	}
	if name, ok := strings.CutPrefix(ref, vpAliasPrefix); ok {
		a := st.aliases[name]
		if a == nil {
			return nil, vpNotFound("POLICY_STORE_ALIAS", name)
		}
		id = a.storeID
	}
	ps := st.stores[id]
	if ps == nil {
		return nil, vpNotFound("POLICY_STORE", ref)
	}
	return ps, nil
}

func vpPage(n int, in map[string]any) (start, end int, next string, err *vpErr) {
	max := vpDefaultPage
	if v, ok := in["maxResults"].(float64); ok {
		if v < 1 || v > vpDefaultPage {
			return 0, 0, "", vpValidation("maxResults must be between 1 and %d", vpDefaultPage)
		}
		max = int(v)
	}
	if tok := vpStr(in, "nextToken"); tok != "" {
		off, perr := strconv.ParseInt(tok, 10, 32)
		if perr != nil || off < 0 || int(off) > n {
			return 0, 0, "", vpValidation("nextToken is invalid")
		}
		start = int(off)
	}
	end = min(n, start+max)
	if end < n {
		next = strconv.Itoa(end)
	}
	return start, end, next, nil
}

// ---- policy stores ----

func (ps *vpPolicyStore) item() map[string]any {
	out := map[string]any{"policyStoreId": ps.id, "arn": ps.arn, "createdDate": vpTime(ps.created), "lastUpdatedDate": vpTime(ps.updated)}
	if ps.description != "" {
		out["description"] = ps.description
	}
	return out
}

func vpCreatePolicyStore(st *vpStore, in map[string]any, x vpExtra) (map[string]any, *vpErr) {
	key, prev := st.idempotent("CreatePolicyStore", in)
	if ps := st.stores[prev]; ps != nil {
		return ps.item(), nil
	}
	if len(st.stores) >= vpMaxPolicyStores {
		return nil, vpQuota("POLICY_STORE", "an account can have at most %d policy stores", vpMaxPolicyStores)
	}
	id := vpNewID()
	ps := &vpPolicyStore{
		id: id, arn: "arn:aws:verifiedpermissions::" + vpAccountID + ":policy-store/" + id,
		description: vpStr(in, "description"), mode: vpStr(vpMap(in, "validationSettings"), "mode"),
		deletion: "DISABLED", created: x.now, updated: x.now, tags: map[string]string{},
		encryption: map[string]any{"default": map[string]any{}},
		policies:   map[string]*vpPolicy{}, templates: map[string]*vpTemplate{}, identitySources: map[string]*vpIdentitySource{},
	}
	if d := vpStr(in, "deletionProtection"); d != "" {
		ps.deletion = d
	}
	if kms := vpMap(vpMap(in, "encryptionSettings"), "kmsEncryptionSettings"); kms != nil {
		ctx := vpMap(kms, "encryptionContext")
		if ctx == nil {
			ctx = map[string]any{}
		}
		ps.encryption = map[string]any{"kmsEncryptionState": map[string]any{"key": vpStr(kms, "key"), "encryptionContext": ctx}}
	}
	for k, v := range vpMap(in, "tags") {
		ps.tags[k], _ = v.(string)
	}
	st.stores[id] = ps
	if key != "" {
		st.tokens[key] = id
	}
	return ps.item(), nil
}

func vpGetPolicyStore(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, err
	}
	out := ps.item()
	out["validationSettings"] = map[string]any{"mode": ps.mode}
	out["deletionProtection"] = ps.deletion
	out["encryptionState"] = ps.encryption
	out["cedarVersion"] = "CEDAR_4"
	if b, _ := in["tags"].(bool); b {
		out["tags"] = vpTagsOut(ps.tags)
	}
	return out, nil
}

func vpTagsOut(tags map[string]string) map[string]any {
	out := make(map[string]any, len(tags))
	for k, v := range tags {
		out[k] = v
	}
	return out
}

func vpUpdatePolicyStore(st *vpStore, in map[string]any, x vpExtra) (map[string]any, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, err
	}
	mode := vpStr(vpMap(in, "validationSettings"), "mode")
	if mode == "STRICT" && ps.mode != "STRICT" {
		if err := ps.validateAll(ps.schema); err != nil {
			return nil, err
		}
	}
	ps.mode = mode
	if d := vpStr(in, "deletionProtection"); d != "" {
		ps.deletion = d
	}
	if _, ok := in["description"]; ok {
		ps.description = vpStr(in, "description")
	}
	ps.updated = x.now
	return ps.item(), nil
}

// validateAll checks every policy and template against schema (STRICT mode).
func (ps *vpPolicyStore) validateAll(schema *vpSchema) *vpErr {
	if schema == nil {
		if len(ps.policies) > 0 || len(ps.templates) > 0 {
			return vpValidation("STRICT validation requires a schema; put a schema before enabling it")
		}
		return nil
	}
	for _, t := range ps.templates {
		if err := schema.validatePolicy(t.parsed); err != nil {
			return vpValidation("policy template %s fails validation: %v", t.id, err)
		}
	}
	for _, p := range ps.policies {
		if eff := ps.effective(p); eff != nil {
			if err := schema.validatePolicy(eff); err != nil {
				return vpValidation("policy %s fails validation: %v", p.id, err)
			}
		}
	}
	return nil
}

func vpDeletePolicyStore(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	id := vpStr(in, "policyStoreId")
	ps := st.stores[id]
	if ps == nil {
		return map[string]any{}, nil // DeletePolicyStore is idempotent
	}
	if ps.deletion == "ENABLED" {
		return nil, &vpErr{status: http.StatusBadRequest, code: "InvalidStateException", msg: "policy store " + id + " has deletion protection enabled"}
	}
	delete(st.stores, id)
	// ponytail: aliases pointing at a deleted store are removed with it rather
	// than lingering in PendingDeletion.
	for name, a := range st.aliases {
		if a.storeID == id {
			delete(st.aliases, name)
		}
	}
	return map[string]any{}, nil
}

func vpListPolicyStores(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	all := make([]*vpPolicyStore, 0, len(st.stores))
	for _, ps := range st.stores {
		all = append(all, ps)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].created.Before(all[j].created) || (all[i].created.Equal(all[j].created) && all[i].id < all[j].id)
	})
	start, end, next, err := vpPage(len(all), in)
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, ps := range all[start:end] {
		items = append(items, ps.item())
	}
	return vpWithNext(map[string]any{"policyStores": items}, next), nil
}

func vpWithNext(out map[string]any, next string) map[string]any {
	if next != "" {
		out["nextToken"] = next
	}
	return out
}

// ---- schema ----

func vpPutSchema(st *vpStore, in map[string]any, x vpExtra) (map[string]any, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, err
	}
	raw := vpStr(vpMap(in, "definition"), "cedarJson")
	if raw == "" {
		return nil, vpValidation("definition.cedarJson is required")
	}
	schema, perr := vpParseSchema(raw)
	if perr != nil {
		return nil, vpValidation("%v", perr)
	}
	if ps.mode == "STRICT" {
		if err := ps.validateAll(schema); err != nil {
			return nil, err
		}
	}
	if ps.schema == nil {
		ps.schemaCreated = x.now
	}
	ps.schema, ps.schemaU = schema, x.now
	return map[string]any{"policyStoreId": ps.id, "namespaces": vpStrings(schema.namespaces),
		"createdDate": vpTime(ps.schemaCreated), "lastUpdatedDate": vpTime(ps.schemaU)}, nil
}

func vpStrings(list []string) []any {
	out := make([]any, 0, len(list))
	for _, s := range list {
		out = append(out, s)
	}
	return out
}

func vpGetSchema(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, err
	}
	if ps.schema == nil {
		return nil, vpNotFound("SCHEMA", ps.id)
	}
	return map[string]any{"policyStoreId": ps.id, "schema": ps.schema.raw, "namespaces": vpStrings(ps.schema.namespaces),
		"createdDate": vpTime(ps.schemaCreated), "lastUpdatedDate": vpTime(ps.schemaU)}, nil
}

// ---- policies ----

// effective returns the policy as evaluated: a static policy as written, a
// template-linked one with its template's slots filled in.
func (ps *vpPolicyStore) effective(p *vpPolicy) *cedarPolicy {
	if p.static {
		return p.parsed
	}
	t := ps.templates[p.templateID]
	if t == nil {
		return nil
	}
	linked := *t.parsed
	if linked.Principal.Slot {
		linked.Principal.Entity, linked.Principal.Slot = p.principal, false
	}
	if linked.Resource.Slot {
		linked.Resource.Entity, linked.Resource.Slot = p.resource, false
	}
	return &linked
}

func vpScopeEntity(sc cedarScope) *cedarEntity {
	if (sc.Op == "==" || sc.Op == "in" || sc.Op == "is") && sc.Entity != nil {
		return sc.Entity
	}
	return nil
}

// summary fills the principal, resource, actions and effect a policy reports.
func (ps *vpPolicyStore) summary(p *vpPolicy, out map[string]any) map[string]any {
	out["policyStoreId"], out["policyId"] = ps.id, p.id
	out["createdDate"], out["lastUpdatedDate"] = vpTime(p.created), vpTime(p.updated)
	if p.static {
		out["policyType"] = "STATIC"
	} else {
		out["policyType"] = "TEMPLATE_LINKED"
	}
	if p.name != "" {
		out["name"] = p.name
	}
	eff := ps.effective(p)
	if eff == nil {
		return out
	}
	if e := vpScopeEntity(eff.Principal); e != nil {
		out["principal"] = vpEntityOut(e)
	}
	if e := vpScopeEntity(eff.Resource); e != nil {
		out["resource"] = vpEntityOut(e)
	}
	actions := []any{}
	for _, a := range eff.Action.Entities {
		actions = append(actions, map[string]any{"actionType": a.Type, "actionId": a.ID})
	}
	if len(actions) > 0 {
		out["actions"] = actions
	}
	out["effect"] = map[string]string{"permit": "Permit", "forbid": "Forbid"}[eff.Effect]
	return out
}

func (p *vpPolicy) definitionDetail() map[string]any {
	if p.static {
		d := map[string]any{"statement": p.statement}
		if p.description != "" {
			d["description"] = p.description
		}
		return map[string]any{"static": d}
	}
	d := map[string]any{"policyTemplateId": p.templateID}
	if p.principal != nil {
		d["principal"] = vpEntityOut(p.principal)
	}
	if p.resource != nil {
		d["resource"] = vpEntityOut(p.resource)
	}
	return map[string]any{"templateLinked": d}
}

func (p *vpPolicy) definitionItem() map[string]any {
	if p.static {
		d := map[string]any{}
		if p.description != "" {
			d["description"] = p.description
		}
		return map[string]any{"static": d}
	}
	return p.definitionDetail()
}

func vpParse(statement string, template bool, ps *vpPolicyStore) (*cedarPolicy, map[string]bool, *vpErr) {
	parsed, slots, err := cedarParsePolicy(statement, template)
	if err != nil {
		return nil, nil, vpValidation("invalid policy statement: %v", err)
	}
	if ps.mode == "STRICT" {
		if ps.schema == nil {
			return nil, nil, vpValidation("STRICT validation requires a schema in policy store %s", ps.id)
		}
		if err := ps.schema.validatePolicy(parsed); err != nil {
			return nil, nil, vpValidation("policy fails schema validation: %v", err)
		}
	}
	return parsed, slots, nil
}

func vpCreatePolicy(st *vpStore, in map[string]any, x vpExtra) (map[string]any, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, err
	}
	key, prev := st.idempotent("CreatePolicy", in)
	if p := ps.policies[prev]; p != nil {
		return ps.summary(p, map[string]any{}), nil
	}
	if len(ps.policies) >= vpMaxPoliciesPer {
		return nil, vpQuota("POLICY", "a policy store can have at most %d policies", vpMaxPoliciesPer)
	}
	def := vpMap(in, "definition")
	p := &vpPolicy{id: vpNewID(), name: vpStr(in, "name"), created: x.now, updated: x.now}
	if static := vpMap(def, "static"); static != nil {
		parsed, _, perr := vpParse(vpStr(static, "statement"), false, ps)
		if perr != nil {
			return nil, perr
		}
		p.static, p.statement, p.description, p.parsed = true, vpStr(static, "statement"), vpStr(static, "description"), parsed
	} else if link := vpMap(def, "templateLinked"); link != nil {
		t := ps.templates[vpStr(link, "policyTemplateId")]
		if t == nil {
			return nil, vpNotFound("POLICY_TEMPLATE", vpStr(link, "policyTemplateId"))
		}
		p.templateID, p.principal, p.resource = t.id, vpEntityOf(vpMap(link, "principal")), vpEntityOf(vpMap(link, "resource"))
		if err := vpCheckSlots(t, p); err != nil {
			return nil, err
		}
		if ps.mode == "STRICT" && ps.schema != nil {
			if err := ps.schema.validatePolicy(ps.effective(p)); err != nil {
				return nil, vpValidation("linked policy fails schema validation: %v", err)
			}
		}
	} else {
		return nil, vpValidation("definition must contain static or templateLinked")
	}
	if p.name != "" {
		for _, other := range ps.policies {
			if other.name == p.name {
				return nil, vpConflict("POLICY", other.id, "a policy named %s already exists", p.name)
			}
		}
	}
	ps.policies[p.id] = p
	if key != "" {
		st.tokens[key] = p.id
	}
	return ps.summary(p, map[string]any{}), nil
}

// vpCheckSlots requires a link to fill exactly the slots its template has.
func vpCheckSlots(t *vpTemplate, p *vpPolicy) *vpErr {
	for slot, given := range map[string]bool{"principal": p.principal != nil, "resource": p.resource != nil} {
		if t.slots[slot] && !given {
			return vpValidation("policy template %s has a ?%s placeholder; templateLinked.%s is required", t.id, slot, slot)
		}
		if !t.slots[slot] && given {
			return vpValidation("policy template %s has no ?%s placeholder; remove templateLinked.%s", t.id, slot, slot)
		}
	}
	return nil
}

func vpGetPolicy(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, p, err := st.policy(in)
	if err != nil {
		return nil, err
	}
	return ps.summary(p, map[string]any{"definition": p.definitionDetail()}), nil
}

func (st *vpStore) policy(in map[string]any) (*vpPolicyStore, *vpPolicy, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, nil, err
	}
	p := ps.policies[vpStr(in, "policyId")]
	if p == nil {
		return nil, nil, vpNotFound("POLICY", vpStr(in, "policyId"))
	}
	return ps, p, nil
}

func vpUpdatePolicy(st *vpStore, in map[string]any, x vpExtra) (map[string]any, *vpErr) {
	ps, p, err := st.policy(in)
	if err != nil {
		return nil, err
	}
	if static := vpMap(vpMap(in, "definition"), "static"); static != nil {
		if !p.static {
			return nil, vpValidation("a template-linked policy can't be changed into a static policy")
		}
		parsed, _, perr := vpParse(vpStr(static, "statement"), false, ps)
		if perr != nil {
			return nil, perr
		}
		// AWS lets an update change the actions and conditions only.
		switch {
		case parsed.Effect != p.parsed.Effect:
			return nil, vpValidation("an update can't change a policy's effect")
		case cedarKeyScope(parsed.Principal) != cedarKeyScope(p.parsed.Principal):
			return nil, vpValidation("an update can't change a policy's principal")
		case cedarKeyScope(parsed.Resource) != cedarKeyScope(p.parsed.Resource):
			return nil, vpValidation("an update can't change a policy's resource")
		}
		p.statement, p.description, p.parsed = vpStr(static, "statement"), vpStr(static, "description"), parsed
	}
	if name, ok := in["name"].(string); ok {
		p.name = name
	}
	p.updated = x.now
	return ps.summary(p, map[string]any{}), nil
}

func cedarKeyScope(sc cedarScope) string {
	e := ""
	if sc.Entity != nil {
		e = sc.Entity.String()
	}
	return sc.Op + "|" + sc.IsType + "|" + e + "|" + strconv.FormatBool(sc.Slot)
}

func vpDeletePolicy(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, p, err := st.policy(in)
	if err != nil {
		return nil, err
	}
	delete(ps.policies, p.id)
	return map[string]any{}, nil
}

func (ps *vpPolicyStore) sortedPolicies() []*vpPolicy {
	out := make([]*vpPolicy, 0, len(ps.policies))
	for _, p := range ps.policies {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].created.Before(out[j].created) || (out[i].created.Equal(out[j].created) && out[i].id < out[j].id)
	})
	return out
}

func vpListPolicies(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, err
	}
	f := vpMap(in, "filter")
	var match []map[string]any
	for _, p := range ps.sortedPolicies() {
		item := ps.summary(p, map[string]any{"definition": p.definitionItem()})
		if !vpFilterRef(vpMap(f, "principal"), item["principal"]) || !vpFilterRef(vpMap(f, "resource"), item["resource"]) {
			continue
		}
		if t := vpStr(f, "policyType"); t != "" && item["policyType"] != t {
			continue
		}
		if t := vpStr(f, "policyTemplateId"); t != "" && p.templateID != t {
			continue
		}
		match = append(match, item)
	}
	start, end, next, perr := vpPage(len(match), in)
	if perr != nil {
		return nil, perr
	}
	items := []any{}
	for _, m := range match[start:end] {
		items = append(items, m)
	}
	return vpWithNext(map[string]any{"policies": items}, next), nil
}

// vpFilterRef applies an EntityReference filter: unspecified matches policies
// without that scope entity; identifier matches that entity.
func vpFilterRef(ref map[string]any, got any) bool {
	if ref == nil {
		return true
	}
	if u, _ := ref["unspecified"].(bool); u {
		return got == nil
	}
	want := vpEntityOf(vpMap(ref, "identifier"))
	g, _ := got.(map[string]any)
	return want != nil && g != nil && vpStr(g, "entityType") == want.Type && vpStr(g, "entityId") == want.ID
}

func vpBatchGetPolicy(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	results, errs := []any{}, []any{}
	for _, raw := range vpList(in, "requests") {
		req, _ := raw.(map[string]any)
		storeID, policyID := vpStr(req, "policyStoreId"), vpStr(req, "policyId")
		ps, err := st.policyStore(storeID)
		if err != nil {
			code := "POLICY_STORE_NOT_FOUND"
			if strings.HasPrefix(storeID, vpAliasPrefix) || strings.Contains(storeID, ":"+vpAliasPrefix) {
				code = "POLICY_STORE_ALIAS_NOT_FOUND"
			}
			errs = append(errs, map[string]any{"code": code, "policyStoreId": storeID, "policyId": policyID, "message": err.msg})
			continue
		}
		p := ps.policies[policyID]
		if p == nil {
			errs = append(errs, map[string]any{"code": "POLICY_NOT_FOUND", "policyStoreId": storeID, "policyId": policyID, "message": "policy " + policyID + " does not exist"})
			continue
		}
		out := ps.summary(p, map[string]any{"definition": p.definitionDetail()})
		results = append(results, out)
	}
	return map[string]any{"results": results, "errors": errs}, nil
}

// ---- policy templates ----

func (t *vpTemplate) item(ps *vpPolicyStore) map[string]any {
	out := map[string]any{"policyStoreId": ps.id, "policyTemplateId": t.id, "createdDate": vpTime(t.created), "lastUpdatedDate": vpTime(t.updated)}
	if t.description != "" {
		out["description"] = t.description
	}
	if t.name != "" {
		out["name"] = t.name
	}
	return out
}

func vpCreateTemplate(st *vpStore, in map[string]any, x vpExtra) (map[string]any, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, err
	}
	key, prev := st.idempotent("CreatePolicyTemplate", in)
	if t := ps.templates[prev]; t != nil {
		return t.item(ps), nil
	}
	if len(ps.templates) >= vpMaxTemplatesPer {
		return nil, vpQuota("POLICY_TEMPLATE", "a policy store can have at most %d policy templates", vpMaxTemplatesPer)
	}
	parsed, slots, perr := vpParse(vpStr(in, "statement"), true, ps)
	if perr != nil {
		return nil, perr
	}
	if len(slots) == 0 {
		return nil, vpValidation("a policy template needs at least one ?principal or ?resource placeholder")
	}
	t := &vpTemplate{id: vpNewID(), name: vpStr(in, "name"), description: vpStr(in, "description"), statement: vpStr(in, "statement"),
		parsed: parsed, slots: slots, created: x.now, updated: x.now}
	ps.templates[t.id] = t
	if key != "" {
		st.tokens[key] = t.id
	}
	return t.item(ps), nil
}

func (st *vpStore) template(in map[string]any) (*vpPolicyStore, *vpTemplate, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, nil, err
	}
	t := ps.templates[vpStr(in, "policyTemplateId")]
	if t == nil {
		return nil, nil, vpNotFound("POLICY_TEMPLATE", vpStr(in, "policyTemplateId"))
	}
	return ps, t, nil
}

func vpGetTemplate(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, t, err := st.template(in)
	if err != nil {
		return nil, err
	}
	out := t.item(ps)
	out["statement"] = t.statement
	return out, nil
}

func vpUpdateTemplate(st *vpStore, in map[string]any, x vpExtra) (map[string]any, *vpErr) {
	ps, t, err := st.template(in)
	if err != nil {
		return nil, err
	}
	parsed, slots, perr := vpParse(vpStr(in, "statement"), true, ps)
	if perr != nil {
		return nil, perr
	}
	for slot := range t.slots {
		if !slots[slot] {
			return nil, vpValidation("an update can't remove the ?%s placeholder that linked policies fill", slot)
		}
	}
	for slot := range slots {
		if !t.slots[slot] {
			return nil, vpValidation("an update can't add a ?%s placeholder", slot)
		}
	}
	t.statement, t.parsed, t.updated = vpStr(in, "statement"), parsed, x.now
	t.description = vpStr(in, "description")
	if name, ok := in["name"].(string); ok {
		t.name = name
	}
	return t.item(ps), nil
}

func vpDeleteTemplate(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, t, err := st.template(in)
	if err != nil {
		return nil, err
	}
	// Deleting a template deletes the policies linked to it, as in AWS.
	for id, p := range ps.policies {
		if p.templateID == t.id {
			delete(ps.policies, id)
		}
	}
	delete(ps.templates, t.id)
	return map[string]any{}, nil
}

func vpListTemplates(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, err := st.policyStore(vpStr(in, "policyStoreId"))
	if err != nil {
		return nil, err
	}
	all := make([]*vpTemplate, 0, len(ps.templates))
	for _, t := range ps.templates {
		all = append(all, t)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].created.Before(all[j].created) || (all[i].created.Equal(all[j].created) && all[i].id < all[j].id)
	})
	start, end, next, perr := vpPage(len(all), in)
	if perr != nil {
		return nil, perr
	}
	items := []any{}
	for _, t := range all[start:end] {
		items = append(items, t.item(ps))
	}
	return vpWithNext(map[string]any{"policyTemplates": items}, next), nil
}

// ---- tags ----

func (st *vpStore) taggable(arn string) (*vpPolicyStore, *vpErr) {
	id, ok := strings.CutPrefix(arn, "arn:aws:verifiedpermissions::"+vpAccountID+":policy-store/")
	if !ok || st.stores[id] == nil {
		return nil, vpNotFound("POLICY_STORE", arn)
	}
	return st.stores[id], nil
}

func vpTagResource(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, err := st.taggable(vpStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	tags := vpMap(in, "tags")
	added := 0
	for k := range tags {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return nil, vpValidation("tag keys can't start with aws:")
		}
		if _, ok := ps.tags[k]; !ok {
			added++
		}
	}
	if len(ps.tags)+added > vpMaxTags {
		return nil, &vpErr{status: http.StatusBadRequest, code: "TooManyTagsException", msg: fmt.Sprintf("a resource can have at most %d tags", vpMaxTags)}
	}
	for k, v := range tags {
		ps.tags[k], _ = v.(string)
	}
	return map[string]any{}, nil
}

func vpUntagResource(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, err := st.taggable(vpStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	for _, k := range vpList(in, "tagKeys") {
		key, _ := k.(string)
		delete(ps.tags, key)
	}
	return map[string]any{}, nil
}

func vpListTags(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	ps, err := st.taggable(vpStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"tags": vpTagsOut(ps.tags)}, nil
}

// ---- aliases ----

func (a *vpAlias) item() map[string]any {
	return map[string]any{"aliasName": a.name, "policyStoreId": a.storeID, "createdAt": vpTime(a.created), "state": "Active",
		"aliasArn": "arn:aws:verifiedpermissions::" + vpAccountID + ":" + vpAliasPrefix + a.name}
}

func vpCreateAlias(st *vpStore, in map[string]any, x vpExtra) (map[string]any, *vpErr) {
	name, storeID := vpStr(in, "aliasName"), vpStr(in, "policyStoreId")
	if strings.HasPrefix(storeID, vpAliasPrefix) {
		return nil, vpValidation("an alias must point at a policy store id, not another alias")
	}
	if st.stores[storeID] == nil {
		return nil, vpNotFound("POLICY_STORE", storeID)
	}
	if a := st.aliases[name]; a != nil {
		return nil, vpConflict("POLICY_STORE_ALIAS", name, "alias %s already exists", name)
	}
	if len(st.aliases) >= vpMaxAliases {
		return nil, vpQuota("POLICY_STORE_ALIAS", "an account can have at most %d policy store aliases", vpMaxAliases)
	}
	a := &vpAlias{name: name, storeID: storeID, created: x.now}
	st.aliases[name] = a
	return a.item(), nil
}

func vpGetAlias(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	a := st.aliases[vpStr(in, "aliasName")]
	if a == nil {
		return nil, vpNotFound("POLICY_STORE_ALIAS", vpStr(in, "aliasName"))
	}
	return a.item(), nil
}

func vpDeleteAlias(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	// ponytail: the alias is removed at once instead of passing through
	// PendingDeletion; deleting a missing alias succeeds, as the API allows.
	delete(st.aliases, vpStr(in, "aliasName"))
	return map[string]any{}, nil
}

func vpListAliases(st *vpStore, in map[string]any, _ vpExtra) (map[string]any, *vpErr) {
	filter := vpStr(vpMap(in, "filter"), "policyStoreId")
	var all []*vpAlias
	for _, a := range st.aliases {
		if filter == "" || a.storeID == filter {
			all = append(all, a)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].name < all[j].name })
	start, end, next, err := vpPage(len(all), in)
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, a := range all[start:end] {
		items = append(items, a.item())
	}
	return vpWithNext(map[string]any{"policyStoreAliases": items}, next), nil
}
