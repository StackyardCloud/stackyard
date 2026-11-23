package server

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	nmAccount        = "123456789012"
	nmMaxRecords     = 20000 // every stored record, across all kinds
	nmMaxTags        = 50
	nmMaxTokens      = 10000
	nmMaxVersion     = 50       // policy versions kept per core network
	nmMaxPolicyBytes = 64 << 20 // policy document bytes kept across all core networks
)

// nmRec is one stored resource or association. data holds output-shaped
// members (never State, Tags or CreatedAt, which render adds); ext holds
// type-specific members rendered beside the base shape (attachments).
type nmRec struct {
	kind     string
	id       string
	arn      string
	parent   string // global network or core network the record belongs to
	data     map[string]any
	ext      map[string]any
	stateKey string
	state    string
	settle   string // state the next read moves to (no background timers)
	tags     map[string]string
	created  time.Time
	updated  time.Time
	seq      int
}

type nmPolicy struct {
	version     int
	alias       string // LIVE, LATEST or ""
	description string
	document    string
	changeSet   string
	created     time.Time
	executedAt  time.Time
	parsed      *nmPolicyDoc
	prevLive    *nmPolicyDoc // LIVE document when this version was executed
}

type nmContextChange struct {
	version int
	change  string
	tgwArn  string
}

type nmStore struct {
	mu          sync.Mutex
	recs        map[string]*nmRec // resources by id
	byArn       map[string]*nmRec
	assocs      map[string]*nmRec // associations by kind|key
	policies    map[string][]*nmPolicy
	prefixLists map[string]map[string]string // core network → prefix list ARN → alias
	resPolicies map[string]string
	metadata    map[string]map[string]any // network resource ARN → metadata
	ctxVersions map[string]int
	ctxChanges  map[string][]nmContextChange
	tokens      map[string]string // op|client token → id
	orgAccess   string
	seq         int
}

func newNetworkManagerStore() *nmStore {
	return &nmStore{
		recs: map[string]*nmRec{}, byArn: map[string]*nmRec{}, assocs: map[string]*nmRec{},
		policies: map[string][]*nmPolicy{}, prefixLists: map[string]map[string]string{},
		resPolicies: map[string]string{}, metadata: map[string]map[string]any{},
		ctxVersions: map[string]int{}, ctxChanges: map[string][]nmContextChange{},
		tokens: map[string]string{}, orgAccess: "DISABLED",
	}
}

func nmNow() time.Time { return time.Now().UTC() }

func nmTime(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

func nmHex(n int) string {
	b := make([]byte, (n+1)/2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

func nmArn(resource string) string { return "arn:aws:networkmanager::" + nmAccount + ":" + resource }

// nmArnParts splits arn:partition:service:region:account:type/id.
func nmArnParts(arn string) (service, region, account, resourceType, resourceID string, ok bool) {
	parts := strings.SplitN(arn, ":", 6)
	if len(parts) != 6 || parts[0] != "arn" {
		return "", "", "", "", "", false
	}
	resourceType, resourceID, _ = strings.Cut(parts[5], "/")
	return parts[2], parts[3], parts[4], resourceType, resourceID, resourceType != "" && resourceID != ""
}

func (st *nmStore) count() int { return len(st.recs) + len(st.assocs) }

// newRec stores a resource; callers have checked the record cap.
func (st *nmStore) newRec(kind, id, arn, parent string, data map[string]any, state, settle string, tags []any) *nmRec {
	now := nmNow()
	st.seq++
	rec := &nmRec{kind: kind, id: id, arn: arn, parent: parent, data: data, ext: map[string]any{},
		stateKey: "State", state: state, settle: settle, tags: map[string]string{}, created: now, updated: now, seq: st.seq}
	nmSetTags(rec, tags)
	st.recs[id] = rec
	if arn != "" {
		st.byArn[arn] = rec
	}
	return rec
}

func (st *nmStore) deleteRec(rec *nmRec) {
	delete(st.recs, rec.id)
	delete(st.byArn, rec.arn)
	delete(st.metadata, rec.arn)
}

func (st *nmStore) newAssoc(kind, key, parent string, data map[string]any, stateKey, state, settle string) *nmRec {
	now := nmNow()
	st.seq++
	rec := &nmRec{kind: kind, id: key, parent: parent, data: data, ext: map[string]any{}, stateKey: stateKey,
		state: state, settle: settle, tags: map[string]string{}, created: now, updated: now, seq: st.seq}
	st.assocs[kind+"|"+key] = rec
	return rec
}

// get returns a resource of kind, settling a pending state transition.
func (st *nmStore) get(kind, resourceType, id string) (*nmRec, *nmErr) {
	rec := st.recs[id]
	if rec == nil || rec.kind != kind {
		return nil, nmNotFound(resourceType, id)
	}
	st.read(rec)
	return rec, nil
}

func (st *nmStore) read(rec *nmRec) {
	if rec.settle != "" {
		rec.state, rec.settle = rec.settle, ""
	}
}

// render returns the output form of a record: its data plus state, tags and
// timestamps. Conform (run by the dispatcher under the lock) copies it.
func (st *nmStore) render(rec *nmRec) map[string]any {
	out := make(map[string]any, len(rec.data)+3)
	for k, v := range rec.data {
		out[k] = v
	}
	switch rec.stateKey {
	case "":
	case "RegistrationState":
		out["State"] = map[string]any{"Code": rec.state}
	default:
		out[rec.stateKey] = rec.state
	}
	if rec.kind != "assoc" {
		out["Tags"] = nmTagList(rec.tags)
	}
	out["CreatedAt"] = nmTime(rec.created)
	return out
}

// list returns records of kind (settled), oldest first, that match keep.
func (st *nmStore) list(kind string, keep func(*nmRec) bool) []*nmRec {
	var out []*nmRec
	for _, rec := range st.recs {
		if rec.kind == kind && (keep == nil || keep(rec)) {
			out = append(out, rec)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].seq < out[j].seq })
	for _, rec := range out {
		st.read(rec)
	}
	return out
}

func (st *nmStore) listAssocs(kind string, keep func(*nmRec) bool) []*nmRec {
	var out []*nmRec
	for _, rec := range st.assocs {
		if rec.kind == kind && (keep == nil || keep(rec)) {
			out = append(out, rec)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].seq < out[j].seq })
	for _, rec := range out {
		st.read(rec)
	}
	return out
}

func (st *nmStore) renderAll(recs []*nmRec) []any {
	out := make([]any, 0, len(recs))
	for _, rec := range recs {
		out = append(out, st.render(rec))
	}
	return out
}

// token remembers a client token for an operation; it reports the id an
// earlier identical request created.
func (st *nmStore) token(op, tok string) (string, bool) {
	if tok == "" {
		return "", false
	}
	id, ok := st.tokens[op+"|"+tok]
	if ok && st.recs[id] == nil {
		delete(st.tokens, op+"|"+tok)
		return "", false
	}
	return id, ok
}

func (st *nmStore) rememberToken(op, tok, id string) {
	if tok == "" {
		return
	}
	if len(st.tokens) >= nmMaxTokens {
		for k := range st.tokens { // ponytail: evict an arbitrary token at the cap
			delete(st.tokens, k)
			break
		}
	}
	st.tokens[op+"|"+tok] = id
}

func nmSetTags(rec *nmRec, tags []any) {
	for _, t := range tags {
		tag, _ := t.(map[string]any)
		key, _ := tag["Key"].(string)
		value, _ := tag["Value"].(string)
		rec.tags[key] = value
	}
}

func nmCheckTags(existing map[string]string, tags []any) *nmErr {
	merged := map[string]bool{}
	for k := range existing {
		merged[k] = true
	}
	for _, t := range tags {
		tag, _ := t.(map[string]any)
		key, _ := tag["Key"].(string)
		if key == "" || len(key) > 128 {
			return nmValidation("Tag keys must be 1 to 128 characters.")
		}
		if strings.HasPrefix(strings.ToLower(key), "aws:") {
			return nmValidation("Tag keys can't start with aws:.")
		}
		if value, _ := tag["Value"].(string); len(value) > 256 {
			return nmValidation("Tag values must be at most 256 characters.")
		}
		merged[key] = true
	}
	if len(merged) > nmMaxTags {
		return nmValidation("A resource can have at most %d tags.", nmMaxTags)
	}
	return nil
}

func nmTagList(tags map[string]string) []any {
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{"Key": k, "Value": tags[k]})
	}
	return out
}

func nmTags(in map[string]any) []any { return nmList(in, "Tags") }

// nmCopyFields copies the named members of in into a new map.
func nmCopyFields(in map[string]any, keys ...string) map[string]any {
	out := map[string]any{}
	for _, k := range keys {
		if v, ok := in[k]; ok {
			out[k] = v
		}
	}
	return out
}

// nmConformOutput shapes an operation's output to its model (dropping
// anything the model doesn't define, and copying maps and lists).
func nmConformOutput(op string, out map[string]any) map[string]any {
	m := awsmodels.NetworkManagerModel()
	shape := m.Operations[op].Output
	if shape == "" || out == nil {
		return map[string]any{}
	}
	conformed, _ := m.Conform(shape, out).(map[string]any)
	if conformed == nil {
		return map[string]any{}
	}
	return conformed
}

func nmCheckCap(st *nmStore, resourceType string) *nmErr {
	if st.count() >= nmMaxRecords {
		return nmQuota(resourceType, "Stackyard holds at most %d Network Manager resources.", nmMaxRecords)
	}
	return nil
}
