package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
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
)

const (
	aossAccountID = "123456789012"
	aossRegion    = "us-east-1"
	// ponytail: caps sit well above AWS quotas so repeated coverage runs
	// against one server don't hit them.
	aossMaxCollections = 1000
	aossMaxPerKind     = 1000
	aossMaxIndexes     = 1000
	aossMaxTags        = 50
	aossDefaultPage    = 20
)

type aossErr struct {
	status    int
	code, msg string
}

func aossError(code, format string, args ...any) *aossErr {
	return &aossErr{status: http.StatusBadRequest, code: code, msg: fmt.Sprintf(format, args...)}
}

func aossValidation(format string, args ...any) *aossErr {
	return aossError("ValidationException", format, args...)
}

func aossNotFound(format string, args ...any) *aossErr {
	return aossError("ResourceNotFoundException", format, args...)
}

func aossConflict(format string, args ...any) *aossErr {
	return aossError("ConflictException", format, args...)
}

func aossQuota(format string, args ...any) *aossErr {
	return aossError("ServiceQuotaExceededException", format, args...)
}

type aossCollection struct {
	id, name, ctype, description, kmsKeyARN, standby, deletionProtection, group, status string
	vector                                                                              map[string]any
	created, modified                                                                   int64
	indexes                                                                             map[string]any // index name -> schema document
}

type aossGroup struct {
	id, name, standby, description, generation string
	limits                                     map[string]any
	created, modified                          int64
}

type aossIngestion struct {
	stream                               bool
	id, collectionID, name, index        string
	failureIndex, roleARN, status, state string
	source                               map[string]any
	revision                             int64
	created, updated                     float64
}

type openSearchServerlessStore struct {
	mu  sync.Mutex
	ec2 *ec2.Service

	collections map[string]*aossCollection // by id
	tokens      map[string]string          // CreateCollection clientToken -> id
	groups      map[string]*aossGroup      // by id
	ingestion   map[string]*aossIngestion  // by id
	policies    map[string]*aossPolicy     // kind + "/" + type + "/" + name
	configs     map[string]*aossConfig     // by id
	endpoints   map[string]*aossEndpoint   // by id
	tags        map[string]map[string]string
	limits      map[string]any // account capacityLimits
}

func newOpenSearchServerlessStore(ec2Svc *ec2.Service) *openSearchServerlessStore {
	return &openSearchServerlessStore{
		ec2:         ec2Svc,
		collections: map[string]*aossCollection{},
		tokens:      map[string]string{},
		groups:      map[string]*aossGroup{},
		ingestion:   map[string]*aossIngestion{},
		policies:    map[string]*aossPolicy{},
		configs:     map[string]*aossConfig{},
		endpoints:   map[string]*aossEndpoint{},
		tags:        map[string]map[string]string{},
		limits:      map[string]any{"maxIndexingCapacityInOCU": float64(10), "maxSearchCapacityInOCU": float64(10)},
	}
}

type aossHandler func(st *openSearchServerlessStore, in map[string]any, base string) (map[string]any, *aossErr)

var aossHandlers map[string]aossHandler

func init() {
	aossHandlers = map[string]aossHandler{
		"CreateCollection":                 (*openSearchServerlessStore).createCollection,
		"BatchGetCollection":               (*openSearchServerlessStore).batchGetCollection,
		"ListCollections":                  (*openSearchServerlessStore).listCollections,
		"UpdateCollection":                 (*openSearchServerlessStore).updateCollection,
		"DeleteCollection":                 (*openSearchServerlessStore).deleteCollection,
		"CreateCollectionGroup":            (*openSearchServerlessStore).createGroup,
		"BatchGetCollectionGroup":          (*openSearchServerlessStore).batchGetGroup,
		"ListCollectionGroups":             (*openSearchServerlessStore).listGroups,
		"UpdateCollectionGroup":            (*openSearchServerlessStore).updateGroup,
		"DeleteCollectionGroup":            (*openSearchServerlessStore).deleteGroup,
		"CreateIndex":                      (*openSearchServerlessStore).createIndex,
		"GetIndex":                         (*openSearchServerlessStore).getIndex,
		"UpdateIndex":                      (*openSearchServerlessStore).updateIndex,
		"DeleteIndex":                      (*openSearchServerlessStore).deleteIndex,
		"CreateIngestionJob":               (*openSearchServerlessStore).createIngestionJob,
		"GetIngestionJob":                  (*openSearchServerlessStore).getIngestionJob,
		"ListIngestionJobs":                (*openSearchServerlessStore).listIngestionJobs,
		"CancelIngestionJob":               (*openSearchServerlessStore).cancelIngestionJob,
		"CreateIngestionStream":            (*openSearchServerlessStore).createIngestionStream,
		"GetIngestionStream":               (*openSearchServerlessStore).getIngestionStream,
		"ListIngestionStreams":             (*openSearchServerlessStore).listIngestionStreams,
		"DeleteIngestionStream":            (*openSearchServerlessStore).deleteIngestionStream,
		"GetAccountSettings":               (*openSearchServerlessStore).getAccountSettings,
		"UpdateAccountSettings":            (*openSearchServerlessStore).updateAccountSettings,
		"GetPoliciesStats":                 (*openSearchServerlessStore).getPoliciesStats,
		"TagResource":                      (*openSearchServerlessStore).tagResource,
		"UntagResource":                    (*openSearchServerlessStore).untagResource,
		"ListTagsForResource":              (*openSearchServerlessStore).listTags,
		"CreateSecurityPolicy":             aossCreatePolicy("security"),
		"GetSecurityPolicy":                aossGetPolicy("security"),
		"UpdateSecurityPolicy":             aossUpdatePolicy("security"),
		"DeleteSecurityPolicy":             aossDeletePolicy("security"),
		"ListSecurityPolicies":             aossListPolicies("security"),
		"CreateAccessPolicy":               aossCreatePolicy("access"),
		"GetAccessPolicy":                  aossGetPolicy("access"),
		"UpdateAccessPolicy":               aossUpdatePolicy("access"),
		"DeleteAccessPolicy":               aossDeletePolicy("access"),
		"ListAccessPolicies":               aossListPolicies("access"),
		"CreateLifecyclePolicy":            aossCreatePolicy("lifecycle"),
		"UpdateLifecyclePolicy":            aossUpdatePolicy("lifecycle"),
		"DeleteLifecyclePolicy":            aossDeletePolicy("lifecycle"),
		"ListLifecyclePolicies":            aossListPolicies("lifecycle"),
		"BatchGetLifecyclePolicy":          (*openSearchServerlessStore).batchGetLifecyclePolicy,
		"BatchGetEffectiveLifecyclePolicy": (*openSearchServerlessStore).batchGetEffectiveLifecyclePolicy,
		"CreateSecurityConfig":             (*openSearchServerlessStore).createConfig,
		"GetSecurityConfig":                (*openSearchServerlessStore).getConfig,
		"UpdateSecurityConfig":             (*openSearchServerlessStore).updateConfig,
		"DeleteSecurityConfig":             (*openSearchServerlessStore).deleteConfig,
		"ListSecurityConfigs":              (*openSearchServerlessStore).listConfigs,
		"CreateVpcEndpoint":                (*openSearchServerlessStore).createEndpoint,
		"BatchGetVpcEndpoint":              (*openSearchServerlessStore).batchGetEndpoint,
		"ListVpcEndpoints":                 (*openSearchServerlessStore).listEndpoints,
		"UpdateVpcEndpoint":                (*openSearchServerlessStore).updateEndpoint,
		"DeleteVpcEndpoint":                (*openSearchServerlessStore).deleteEndpoint,
	}
}

// handle runs op under the store lock and shapes the result to outShape while
// still holding it, so the encoder never reads live store state.
func (st *openSearchServerlessStore) handle(op, outShape string, in map[string]any, base string) (any, *aossErr) {
	h := aossHandlers[op]
	if h == nil {
		return nil, &aossErr{status: http.StatusNotImplemented, code: "InternalServerException", msg: op + " is not implemented"}
	}
	// VPC endpoint subnets and security groups come from the EC2 store, which
	// has its own lock: look them up before taking ours.
	var ec2Snap *aossEC2Snapshot
	if op == "CreateVpcEndpoint" || op == "UpdateVpcEndpoint" {
		ec2Snap = st.snapshotEC2(in)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if ec2Snap != nil {
		in["__ec2"] = ec2Snap // internal only: callers' "__" keys never survive Conform
	}
	out, err := h(st, in, base)
	if err != nil {
		return nil, err
	}
	if outShape == "" || out == nil {
		return map[string]any{}, nil
	}
	return awsmodels.OpenSearchServerlessModel().Conform(outShape, out), nil
}

// --- helpers ---------------------------------------------------------------

func aossStr(in map[string]any, key string) string {
	s, _ := in[key].(string)
	return s
}

func aossMap(in map[string]any, key string) map[string]any {
	m, _ := in[key].(map[string]any)
	return m
}

func aossList(in map[string]any, key string) []any {
	l, _ := in[key].([]any)
	return l
}

func aossStrings(in map[string]any, key string) []string {
	var out []string
	for _, v := range aossList(in, key) {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func aossOr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func aossRandID(n int) string {
	b := make([]byte, (n+1)/2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

// aossVersion is an opaque policy / config version in the model's base64 form.
func aossVersion() string {
	b := make([]byte, 15)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b) // 20 characters, no padding
}

func aossNowMillis() int64 { return time.Now().UnixMilli() }

func aossNowSeconds() float64 { return float64(time.Now().UnixMilli()) / 1000 }

func aossARN(resource string) string {
	return "arn:aws:aoss:" + aossRegion + ":" + aossAccountID + ":" + resource
}

// aossPage slices items by an offset nextToken.
func aossPage[T any](items []T, in map[string]any) ([]T, string, *aossErr) {
	start := 0
	if tok := aossStr(in, "nextToken"); tok != "" {
		n, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || n < 0 || int(n) > len(items) {
			return nil, "", aossValidation("invalid nextToken")
		}
		start = int(n)
	}
	size := aossDefaultPage
	if v, ok := in["maxResults"].(float64); ok && v >= 1 && v <= 100 {
		size = int(v)
	}
	end := min(start+size, len(items))
	next := ""
	if end < len(items) {
		next = strconv.Itoa(end)
	}
	return items[start:end], next, nil
}

func aossTagList(m map[string]string) []any {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{"key": k, "value": m[k]})
	}
	return out
}

func (st *openSearchServerlessStore) setTagsLocked(arn string, tags []any) *aossErr {
	cur := st.tags[arn]
	next := map[string]string{}
	for k, v := range cur {
		next[k] = v
	}
	for _, t := range tags {
		m, _ := t.(map[string]any)
		key := aossStr(m, "key")
		if strings.HasPrefix(strings.ToLower(key), "aws:") {
			return aossValidation("tag keys can't start with aws:")
		}
		next[key] = aossStr(m, "value")
	}
	if len(next) > aossMaxTags {
		return aossQuota("a resource can have at most %d tags", aossMaxTags)
	}
	if len(next) > 0 {
		st.tags[arn] = next
	}
	return nil
}

// --- collections -------------------------------------------------------------

var aossIndexNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._+-]{0,254}$`)

func (c *aossCollection) arn() string { return aossARN("collection/" + c.id) }

func (c *aossCollection) detail(base string) map[string]any {
	out := map[string]any{
		"id": c.id, "name": c.name, "status": c.status, "type": c.ctype, "arn": c.arn(),
		"kmsKeyArn": c.kmsKeyARN, "standbyReplicas": c.standby, "deletionProtection": c.deletionProtection,
		"createdDate": c.created, "lastModifiedDate": c.modified,
		"collectionEndpoint": base + "/aoss/" + c.id,
		"dashboardEndpoint":  base + "/aoss/" + c.id + "/_dashboards",
	}
	if c.description != "" {
		out["description"] = c.description
	}
	if c.vector != nil {
		out["vectorOptions"] = c.vector
	}
	if c.group != "" {
		out["collectionGroupName"] = c.group
	}
	return out
}

// settle moves a collection out of a transitional state on read.
func (c *aossCollection) settle() {
	if c.status == "CREATING" || c.status == "UPDATING" {
		c.status = "ACTIVE"
	}
}

func (st *openSearchServerlessStore) collectionByNameLocked(name string) *aossCollection {
	for _, c := range st.collections {
		if c.name == name {
			return c
		}
	}
	return nil
}

func (st *openSearchServerlessStore) groupByNameLocked(name string) *aossGroup {
	for _, g := range st.groups {
		if g.name == name {
			return g
		}
	}
	return nil
}

func (st *openSearchServerlessStore) createCollection(in map[string]any, base string) (map[string]any, *aossErr) {
	if tok := aossStr(in, "clientToken"); tok != "" {
		if c := st.collections[st.tokens[tok]]; c != nil {
			return map[string]any{"createCollectionDetail": c.detail(base)}, nil
		}
	}
	name := aossStr(in, "name")
	if st.collectionByNameLocked(name) != nil {
		return nil, aossConflict("A collection named %s already exists", name)
	}
	if len(st.collections) >= aossMaxCollections {
		return nil, aossQuota("the account can have at most %d collections", aossMaxCollections)
	}
	ctype := aossOr(aossStr(in, "type"), "SEARCH")
	group := aossStr(in, "collectionGroupName")
	if group != "" && st.groupByNameLocked(group) == nil {
		return nil, aossValidation("Collection group %s does not exist", group)
	}
	// AWS needs a matching encryption policy unless the request carries its
	// own encryption configuration.
	kms := ""
	if cfg := aossMap(in, "encryptionConfig"); cfg != nil {
		kms = aossOr(aossStr(cfg, "kmsKeyArn"), "auto")
	} else {
		p := st.encryptionPolicyForLocked(name)
		if p == nil {
			return nil, aossValidation("No matching security policy of encryption type found for collection name: %s. Please create security policy of encryption type for this collection.", name)
		}
		kms = aossOr(aossStr(p.doc.(map[string]any), "KmsARN"), "auto")
	}
	now := aossNowMillis()
	c := &aossCollection{
		id: aossRandID(20), name: name, ctype: ctype, description: aossStr(in, "description"), kmsKeyARN: kms,
		standby: aossOr(aossStr(in, "standbyReplicas"), "ENABLED"), deletionProtection: aossOr(aossStr(in, "deletionProtection"), "DISABLED"),
		group: group, status: "CREATING", vector: aossMap(in, "vectorOptions"), created: now, modified: now, indexes: map[string]any{},
	}
	if err := st.setTagsLocked(c.arn(), aossList(in, "tags")); err != nil {
		return nil, err
	}
	st.collections[c.id] = c
	if tok := aossStr(in, "clientToken"); tok != "" {
		st.tokens[tok] = c.id
	}
	return map[string]any{"createCollectionDetail": c.detail(base)}, nil
}

func (st *openSearchServerlessStore) batchGetCollection(in map[string]any, base string) (map[string]any, *aossErr) {
	ids, names := aossStrings(in, "ids"), aossStrings(in, "names")
	if (len(ids) == 0) == (len(names) == 0) {
		return nil, aossValidation("Specify either ids or names, but not both")
	}
	details, errs := []any{}, []any{}
	for _, id := range ids {
		if c := st.collections[id]; c != nil {
			c.settle()
			details = append(details, c.detail(base))
		} else {
			errs = append(errs, map[string]any{"id": id, "errorCode": "NOT_FOUND", "errorMessage": "Collection not found"})
		}
	}
	for _, name := range names {
		if c := st.collectionByNameLocked(name); c != nil {
			c.settle()
			details = append(details, c.detail(base))
		} else {
			errs = append(errs, map[string]any{"name": name, "errorCode": "NOT_FOUND", "errorMessage": "Collection not found"})
		}
	}
	return map[string]any{"collectionDetails": details, "collectionErrorDetails": errs}, nil
}

func (st *openSearchServerlessStore) sortedCollectionsLocked() []*aossCollection {
	out := make([]*aossCollection, 0, len(st.collections))
	for _, c := range st.collections {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

func (st *openSearchServerlessStore) listCollections(in map[string]any, base string) (map[string]any, *aossErr) {
	f := aossMap(in, "collectionFilters")
	var matched []any
	for _, c := range st.sortedCollectionsLocked() {
		c.settle()
		if (aossStr(f, "name") != "" && c.name != aossStr(f, "name")) ||
			(aossStr(f, "status") != "" && c.status != aossStr(f, "status")) ||
			(aossStr(f, "collectionGroupName") != "" && c.group != aossStr(f, "collectionGroupName")) {
			continue
		}
		matched = append(matched, c.detail(base))
	}
	page, next, err := aossPage(matched, in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"collectionSummaries": aossNonNil(page), "nextToken": next}, nil
}

func aossNonNil(items []any) []any {
	if items == nil {
		return []any{}
	}
	return items
}

func (st *openSearchServerlessStore) collectionLocked(id string) (*aossCollection, *aossErr) {
	c := st.collections[id]
	if c == nil {
		return nil, aossNotFound("Collection %s does not exist", id)
	}
	return c, nil
}

func (st *openSearchServerlessStore) updateCollection(in map[string]any, base string) (map[string]any, *aossErr) {
	c := st.collections[aossStr(in, "id")]
	if c == nil {
		// botocore models no ResourceNotFoundException for UpdateCollection.
		return nil, aossValidation("Collection %s does not exist", aossStr(in, "id"))
	}
	if c.status != "ACTIVE" {
		return nil, aossConflict("Collection %s is %s; it must be ACTIVE to update", c.id, c.status)
	}
	if v, ok := in["description"].(string); ok {
		c.description = v
	}
	if v := aossMap(in, "vectorOptions"); v != nil {
		c.vector = v
	}
	if v := aossStr(in, "deletionProtection"); v != "" {
		c.deletionProtection = v
	}
	if cfg := aossMap(in, "encryptionConfig"); cfg != nil {
		c.kmsKeyARN = aossOr(aossStr(cfg, "kmsKeyArn"), "auto")
	}
	c.status, c.modified = "UPDATING", aossNowMillis()
	return map[string]any{"updateCollectionDetail": c.detail(base)}, nil
}

func (st *openSearchServerlessStore) deleteCollection(in map[string]any, _ string) (map[string]any, *aossErr) {
	c, err := st.collectionLocked(aossStr(in, "id"))
	if err != nil {
		return nil, err
	}
	if c.deletionProtection == "ENABLED" {
		return nil, aossValidation("Collection %s has deletion protection enabled", c.id)
	}
	if c.status == "CREATING" || c.status == "UPDATING" {
		return nil, aossConflict("Collection %s is %s and can't be deleted yet", c.id, c.status)
	}
	// ponytail: deletion completes at once; AWS reports DELETING for a while.
	delete(st.collections, c.id)
	delete(st.tags, c.arn())
	for tok, id := range st.tokens { // keep the idempotency map bounded
		if id == c.id {
			delete(st.tokens, tok)
		}
	}
	for id, j := range st.ingestion {
		if j.collectionID == c.id {
			delete(st.tags, j.arn())
			delete(st.ingestion, id)
		}
	}
	return map[string]any{"deleteCollectionDetail": map[string]any{
		"id": c.id, "name": c.name, "status": "DELETING", "deletionProtection": c.deletionProtection,
	}}, nil
}

// --- collection groups ----------------------------------------------------------

func (g *aossGroup) arn() string { return aossARN("collection-group/" + g.id) }

func (st *openSearchServerlessStore) groupDetailLocked(g *aossGroup) map[string]any {
	n := 0
	for _, c := range st.collections {
		if c.group == g.name {
			n++
		}
	}
	out := map[string]any{
		"id": g.id, "arn": g.arn(), "name": g.name, "standbyReplicas": g.standby, "createdDate": g.created,
		"lastModifiedDate": g.modified, "generation": g.generation, "numberOfCollections": n,
		"tags": aossTagList(st.tags[g.arn()]),
	}
	if g.description != "" {
		out["description"] = g.description
	}
	if g.limits != nil {
		out["capacityLimits"] = g.limits
	}
	return out
}

func aossCheckGroupLimits(limits map[string]any) *aossErr {
	for _, kind := range []string{"Indexing", "Search"} {
		lo, hasLo := limits["min"+kind+"CapacityInOCU"].(float64)
		hi, hasHi := limits["max"+kind+"CapacityInOCU"].(float64)
		if hasLo && hasHi && lo > hi {
			return aossValidation("min%sCapacityInOCU can't exceed max%sCapacityInOCU", kind, kind)
		}
	}
	return nil
}

func (st *openSearchServerlessStore) createGroup(in map[string]any, _ string) (map[string]any, *aossErr) {
	name := aossStr(in, "name")
	if st.groupByNameLocked(name) != nil {
		return nil, aossConflict("A collection group named %s already exists", name)
	}
	if len(st.groups) >= aossMaxPerKind {
		return nil, aossQuota("the account can have at most %d collection groups", aossMaxPerKind)
	}
	limits := aossMap(in, "capacityLimits")
	if err := aossCheckGroupLimits(limits); err != nil {
		return nil, err
	}
	now := aossNowMillis()
	g := &aossGroup{
		id: aossRandID(20), name: name, standby: aossStr(in, "standbyReplicas"), description: aossStr(in, "description"),
		generation: aossOr(aossStr(in, "generation"), "CLASSIC"), limits: limits, created: now, modified: now,
	}
	if err := st.setTagsLocked(g.arn(), aossList(in, "tags")); err != nil {
		return nil, err
	}
	st.groups[g.id] = g
	return map[string]any{"createCollectionGroupDetail": st.groupDetailLocked(g)}, nil
}

func (st *openSearchServerlessStore) batchGetGroup(in map[string]any, _ string) (map[string]any, *aossErr) {
	ids, names := aossStrings(in, "ids"), aossStrings(in, "names")
	if (len(ids) == 0) == (len(names) == 0) {
		return nil, aossValidation("Specify either ids or names, but not both")
	}
	details, errs := []any{}, []any{}
	for _, id := range ids {
		if g := st.groups[id]; g != nil {
			details = append(details, st.groupDetailLocked(g))
		} else {
			errs = append(errs, map[string]any{"id": id, "errorCode": "NOT_FOUND", "errorMessage": "Collection group not found"})
		}
	}
	for _, name := range names {
		if g := st.groupByNameLocked(name); g != nil {
			details = append(details, st.groupDetailLocked(g))
		} else {
			errs = append(errs, map[string]any{"name": name, "errorCode": "NOT_FOUND", "errorMessage": "Collection group not found"})
		}
	}
	return map[string]any{"collectionGroupDetails": details, "collectionGroupErrorDetails": errs}, nil
}

func (st *openSearchServerlessStore) listGroups(in map[string]any, _ string) (map[string]any, *aossErr) {
	groups := make([]*aossGroup, 0, len(st.groups))
	for _, g := range st.groups {
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].name < groups[j].name })
	page, next, err := aossPage(groups, in)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, g := range page {
		out = append(out, st.groupDetailLocked(g))
	}
	return map[string]any{"collectionGroupSummaries": out, "nextToken": next}, nil
}

func (st *openSearchServerlessStore) updateGroup(in map[string]any, _ string) (map[string]any, *aossErr) {
	g := st.groups[aossStr(in, "id")]
	if g == nil {
		return nil, aossValidation("Collection group %s does not exist", aossStr(in, "id"))
	}
	limits := g.limits
	if v := aossMap(in, "capacityLimits"); v != nil {
		limits = v
	}
	if err := aossCheckGroupLimits(limits); err != nil {
		return nil, err
	}
	if v, ok := in["description"].(string); ok {
		g.description = v
	}
	g.limits, g.modified = limits, aossNowMillis()
	return map[string]any{"updateCollectionGroupDetail": st.groupDetailLocked(g)}, nil
}

func (st *openSearchServerlessStore) deleteGroup(in map[string]any, _ string) (map[string]any, *aossErr) {
	g := st.groups[aossStr(in, "id")]
	if g == nil {
		return nil, aossNotFound("Collection group %s does not exist", aossStr(in, "id"))
	}
	for _, c := range st.collections {
		if c.group == g.name {
			return nil, aossConflict("Collection group %s still contains collection %s", g.name, c.name)
		}
	}
	delete(st.groups, g.id)
	delete(st.tags, g.arn())
	return map[string]any{}, nil
}

// --- indexes ----------------------------------------------------------------------

func (st *openSearchServerlessStore) indexTargetLocked(in map[string]any) (*aossCollection, string, *aossErr) {
	c, err := st.collectionLocked(aossStr(in, "id"))
	if err != nil {
		return nil, "", err
	}
	name := aossStr(in, "indexName")
	if !aossIndexNamePattern.MatchString(name) {
		return nil, "", aossValidation("Index names must be lowercase and can't begin with an underscore or hyphen")
	}
	return c, name, nil
}

func (st *openSearchServerlessStore) createIndex(in map[string]any, _ string) (map[string]any, *aossErr) {
	c, name, err := st.indexTargetLocked(in)
	if err != nil {
		return nil, err
	}
	if _, ok := c.indexes[name]; ok {
		return nil, aossConflict("Index %s already exists in collection %s", name, c.id)
	}
	if len(c.indexes) >= aossMaxIndexes {
		return nil, aossValidation("a collection can have at most %d indexes", aossMaxIndexes)
	}
	c.indexes[name] = aossSchema(in)
	return map[string]any{}, nil
}

// aossSchema returns the request's index schema, an empty document if none.
// Request values are freshly decoded and never shared, so storing them is safe.
func aossSchema(in map[string]any) any {
	if v, ok := in["indexSchema"]; ok && v != nil {
		return v
	}
	return map[string]any{}
}

func (st *openSearchServerlessStore) getIndex(in map[string]any, _ string) (map[string]any, *aossErr) {
	c, name, err := st.indexTargetLocked(in)
	if err != nil {
		return nil, err
	}
	schema, ok := c.indexes[name]
	if !ok {
		return nil, aossNotFound("Index %s does not exist in collection %s", name, c.id)
	}
	return map[string]any{"indexSchema": schema}, nil
}

func (st *openSearchServerlessStore) updateIndex(in map[string]any, _ string) (map[string]any, *aossErr) {
	c, name, err := st.indexTargetLocked(in)
	if err != nil {
		return nil, err
	}
	if _, ok := c.indexes[name]; !ok {
		return nil, aossNotFound("Index %s does not exist in collection %s", name, c.id)
	}
	c.indexes[name] = aossSchema(in)
	return map[string]any{}, nil
}

func (st *openSearchServerlessStore) deleteIndex(in map[string]any, _ string) (map[string]any, *aossErr) {
	c, name, err := st.indexTargetLocked(in)
	if err != nil {
		return nil, err
	}
	if _, ok := c.indexes[name]; !ok {
		return nil, aossNotFound("Index %s does not exist in collection %s", name, c.id)
	}
	delete(c.indexes, name)
	return map[string]any{}, nil
}

// --- ingestion jobs and streams ------------------------------------------------------

func (j *aossIngestion) arn() string {
	kind := "ingestion-job"
	if j.stream {
		kind = "ingestion-stream"
	}
	return aossARN("collection/" + j.collectionID + "/" + kind + "/" + j.id)
}

func (j *aossIngestion) detail() map[string]any {
	out := map[string]any{
		"arn": j.arn(), "collectionId": j.collectionID, "indexName": j.index, "source": j.source,
		"status": j.status, "state": j.state, "revision": j.revision, "createdAt": j.created, "updatedAt": j.updated,
		"statistics": map[string]any{"s3": map[string]any{"objectsDiscovered": 0, "objectsIngested": 0, "objectsFailed": 0}},
	}
	if j.stream {
		out["ingestionStreamId"], out["ingestionStreamName"], out["roleArn"] = j.id, j.name, j.roleARN
	} else {
		out["ingestionJobId"], out["ingestionJobName"] = j.id, j.name
	}
	if j.failureIndex != "" {
		out["failureIndexName"] = j.failureIndex
	}
	return out
}

// settle advances a job or stream one step per read.
func (j *aossIngestion) settle() {
	next := map[string][2]string{
		"CREATING":   {"ACTIVE", "SCANNING"},
		"CANCELLING": {"CANCELLED", "STOPPED"},
	}
	if !j.stream {
		next["ACTIVE"] = [2]string{"COMPLETED", "INGESTION_COMPLETE"}
	}
	if n, ok := next[j.status]; ok {
		j.status, j.state, j.updated = n[0], n[1], aossNowSeconds()
		j.revision++
	}
}

func (st *openSearchServerlessStore) createIngestion(in map[string]any, stream bool) (map[string]any, *aossErr) {
	c, err := st.collectionLocked(aossStr(in, "collectionId"))
	if err != nil {
		return nil, err
	}
	nameKey := "ingestionJobName"
	if stream {
		nameKey = "ingestionStreamName"
	}
	name := aossStr(in, nameKey)
	for _, j := range st.ingestion {
		if j.collectionID == c.id && j.stream == stream && j.name == name {
			return nil, aossConflict("%s %s already exists in collection %s", nameKey, name, c.id)
		}
	}
	if len(st.ingestion) >= aossMaxPerKind {
		return nil, aossQuota("the account can have at most %d ingestion jobs and streams", aossMaxPerKind)
	}
	now := aossNowSeconds()
	j := &aossIngestion{
		stream: stream, id: aossRandID(26), collectionID: c.id, name: name, index: aossStr(in, "indexName"),
		failureIndex: aossStr(in, "failureIndexName"), roleARN: aossStr(in, "roleArn"), status: "CREATING", state: "STARTING",
		source: aossMap(in, "source"), revision: 1, created: now, updated: now,
	}
	if err := st.setTagsLocked(j.arn(), aossList(in, "tags")); err != nil {
		return nil, err
	}
	st.ingestion[j.id] = j
	return j.detail(), nil
}

func (st *openSearchServerlessStore) ingestionLocked(in map[string]any, idKey string, stream bool) (*aossIngestion, *aossErr) {
	j := st.ingestion[aossStr(in, idKey)]
	if j == nil || j.stream != stream || j.collectionID != aossStr(in, "collectionId") {
		return nil, aossNotFound("%s %s does not exist in collection %s", idKey, aossStr(in, idKey), aossStr(in, "collectionId"))
	}
	return j, nil
}

func (st *openSearchServerlessStore) listIngestion(in map[string]any, stream bool) ([]any, string, *aossErr) {
	if _, err := st.collectionLocked(aossStr(in, "collectionId")); err != nil {
		return nil, "", err
	}
	var jobs []*aossIngestion
	for _, j := range st.ingestion {
		if j.collectionID == aossStr(in, "collectionId") && j.stream == stream {
			jobs = append(jobs, j)
		}
	}
	sort.Slice(jobs, func(a, b int) bool {
		return jobs[a].created < jobs[b].created || (jobs[a].created == jobs[b].created && jobs[a].id < jobs[b].id)
	})
	page, next, err := aossPage(jobs, in)
	if err != nil {
		return nil, "", err
	}
	out := []any{}
	for _, j := range page {
		j.settle()
		out = append(out, j.detail())
	}
	return out, next, nil
}

func (st *openSearchServerlessStore) createIngestionJob(in map[string]any, _ string) (map[string]any, *aossErr) {
	d, err := st.createIngestion(in, false)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ingestionJobDetail": d}, nil
}

func (st *openSearchServerlessStore) getIngestionJob(in map[string]any, _ string) (map[string]any, *aossErr) {
	j, err := st.ingestionLocked(in, "ingestionJobId", false)
	if err != nil {
		return nil, err
	}
	j.settle()
	return map[string]any{"ingestionJobDetail": j.detail()}, nil
}

func (st *openSearchServerlessStore) listIngestionJobs(in map[string]any, _ string) (map[string]any, *aossErr) {
	out, next, err := st.listIngestion(in, false)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ingestionJobSummaries": out, "nextToken": next}, nil
}

func (st *openSearchServerlessStore) cancelIngestionJob(in map[string]any, _ string) (map[string]any, *aossErr) {
	j, err := st.ingestionLocked(in, "ingestionJobId", false)
	if err != nil {
		return nil, err
	}
	if j.status != "CREATING" && j.status != "ACTIVE" {
		return nil, aossConflict("Ingestion job %s is %s and can't be cancelled", j.id, j.status)
	}
	j.status, j.state, j.updated = "CANCELLING", "STOPPED", aossNowSeconds()
	j.revision++
	return map[string]any{"ingestionJobId": j.id, "collectionId": j.collectionID, "ingestionJobName": j.name, "status": j.status}, nil
}

func (st *openSearchServerlessStore) createIngestionStream(in map[string]any, _ string) (map[string]any, *aossErr) {
	d, err := st.createIngestion(in, true)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ingestionStreamDetail": d}, nil
}

func (st *openSearchServerlessStore) getIngestionStream(in map[string]any, _ string) (map[string]any, *aossErr) {
	j, err := st.ingestionLocked(in, "ingestionStreamId", true)
	if err != nil {
		return nil, err
	}
	j.settle()
	return map[string]any{"ingestionStreamDetail": j.detail()}, nil
}

func (st *openSearchServerlessStore) listIngestionStreams(in map[string]any, _ string) (map[string]any, *aossErr) {
	out, next, err := st.listIngestion(in, true)
	if err != nil {
		return nil, err
	}
	return map[string]any{"ingestionStreamSummaries": out, "nextToken": next}, nil
}

func (st *openSearchServerlessStore) deleteIngestionStream(in map[string]any, _ string) (map[string]any, *aossErr) {
	j, err := st.ingestionLocked(in, "ingestionStreamId", true)
	if err != nil {
		return nil, err
	}
	if j.status == "CREATING" {
		return nil, aossConflict("Ingestion stream %s is still CREATING", j.id)
	}
	delete(st.ingestion, j.id)
	delete(st.tags, j.arn())
	return map[string]any{"ingestionStreamId": j.id, "collectionId": j.collectionID, "ingestionStreamName": j.name, "status": "DELETING"}, nil
}

// --- account settings, stats, tags ---------------------------------------------------

func (st *openSearchServerlessStore) getAccountSettings(map[string]any, string) (map[string]any, *aossErr) {
	return map[string]any{"accountSettingsDetail": map[string]any{"capacityLimits": st.limits}}, nil
}

func (st *openSearchServerlessStore) updateAccountSettings(in map[string]any, _ string) (map[string]any, *aossErr) {
	next := map[string]any{}
	for k, v := range st.limits {
		next[k] = v
	}
	for k, v := range aossMap(in, "capacityLimits") {
		next[k] = v
	}
	st.limits = next
	return map[string]any{"accountSettingsDetail": map[string]any{"capacityLimits": st.limits}}, nil
}

func (st *openSearchServerlessStore) getPoliciesStats(map[string]any, string) (map[string]any, *aossErr) {
	count := map[string]int{}
	for key := range st.policies {
		parts := strings.SplitN(key, "/", 3)
		count[parts[1]]++
	}
	saml := 0
	for _, c := range st.configs {
		if c.ctype == "saml" {
			saml++
		}
	}
	return map[string]any{
		"SecurityPolicyStats":  map[string]any{"EncryptionPolicyCount": count["encryption"], "NetworkPolicyCount": count["network"]},
		"AccessPolicyStats":    map[string]any{"DataPolicyCount": count["data"]},
		"LifecyclePolicyStats": map[string]any{"RetentionPolicyCount": count["retention"]},
		"SecurityConfigStats":  map[string]any{"SamlConfigCount": saml},
		"TotalPolicyCount":     len(st.policies) + len(st.configs),
	}, nil
}

// arnExistsLocked reports whether arn names a taggable resource here.
func (st *openSearchServerlessStore) arnExistsLocked(arn string) bool {
	for _, c := range st.collections {
		if c.arn() == arn {
			return true
		}
	}
	for _, g := range st.groups {
		if g.arn() == arn {
			return true
		}
	}
	for _, j := range st.ingestion {
		if j.arn() == arn {
			return true
		}
	}
	return false
}

func (st *openSearchServerlessStore) tagResource(in map[string]any, _ string) (map[string]any, *aossErr) {
	arn := aossStr(in, "resourceArn")
	if !st.arnExistsLocked(arn) {
		return nil, aossNotFound("Resource %s does not exist", arn)
	}
	if err := st.setTagsLocked(arn, aossList(in, "tags")); err != nil {
		return nil, err
	}
	return map[string]any{}, nil
}

func (st *openSearchServerlessStore) untagResource(in map[string]any, _ string) (map[string]any, *aossErr) {
	arn := aossStr(in, "resourceArn")
	if !st.arnExistsLocked(arn) {
		return nil, aossNotFound("Resource %s does not exist", arn)
	}
	next := map[string]string{}
	for k, v := range st.tags[arn] {
		next[k] = v
	}
	for _, k := range aossStrings(in, "tagKeys") {
		delete(next, k)
	}
	st.tags[arn] = next
	return map[string]any{}, nil
}

func (st *openSearchServerlessStore) listTags(in map[string]any, _ string) (map[string]any, *aossErr) {
	arn := aossStr(in, "resourceArn")
	if !st.arnExistsLocked(arn) {
		return nil, aossNotFound("Resource %s does not exist", arn)
	}
	return map[string]any{"tags": aossTagList(st.tags[arn])}, nil
}
