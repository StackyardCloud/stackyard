package server

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Mail Manager state. Every resource kind shares one record type and one
// generic create/get/list/update/delete path; kind-specific rules live in
// mailmanager_resources.go and mailmanager_archive.go.

const (
	mmgrAccount      = "123456789012"
	mmgrRegion       = "us-east-1"
	mmgrARNBase      = "arn:aws:ses:" + mmgrRegion + ":" + mmgrAccount + ":"
	mmgrMaxResources = 2000  // every kind together, jobs, searches and exports included
	mmgrMaxMembers   = 50000 // address list members across all lists
	mmgrMaxTags      = 50
	mmgrDefaultPage  = 50
)

// mmgrKind describes one resource type: id prefix, ARN type and model members.
type mmgrKind struct {
	id, prefix, arnType, idKey, nameKey, arnKey, createdKey, updatedKey, listKey, summary, getOp string
}

var mmgrKinds = func() map[string]*mmgrKind {
	out := map[string]*mmgrKind{}
	for _, k := range []mmgrKind{
		{"addonSubscription", "as-", "addon-subscription", "AddonSubscriptionId", "", "AddonSubscriptionArn", "CreatedTimestamp", "", "AddonSubscriptions", "AddonSubscription", "GetAddonSubscription"},
		{"addonInstance", "ai-", "addon-instance", "AddonInstanceId", "", "AddonInstanceArn", "CreatedTimestamp", "", "AddonInstances", "AddonInstance", "GetAddonInstance"},
		{"addressList", "al-", "mailmanager-address-list", "AddressListId", "AddressListName", "AddressListArn", "CreatedTimestamp", "LastUpdatedTimestamp", "AddressLists", "AddressList", "GetAddressList"},
		{"archive", "a-", "mailmanager-archive", "ArchiveId", "ArchiveName", "ArchiveArn", "CreatedTimestamp", "LastUpdatedTimestamp", "Archives", "Archive", "GetArchive"},
		{"ingressPoint", "inp-", "mailmanager-ingress-point", "IngressPointId", "IngressPointName", "IngressPointArn", "CreatedTimestamp", "LastUpdatedTimestamp", "IngressPoints", "IngressPoint", "GetIngressPoint"},
		{"relay", "rl-", "mailmanager-smtp-relay", "RelayId", "RelayName", "RelayArn", "CreatedTimestamp", "LastModifiedTimestamp", "Relays", "Relay", "GetRelay"},
		{"ruleSet", "rs-", "mailmanager-rule-set", "RuleSetId", "RuleSetName", "RuleSetArn", "CreatedDate", "LastModificationDate", "RuleSets", "RuleSet", "GetRuleSet"},
		{"trafficPolicy", "tp-", "mailmanager-traffic-policy", "TrafficPolicyId", "TrafficPolicyName", "TrafficPolicyArn", "CreatedTimestamp", "LastUpdatedTimestamp", "TrafficPolicies", "TrafficPolicy", "GetTrafficPolicy"},
		{"importJob", "", "", "JobId", "Name", "", "CreatedTimestamp", "", "ImportJobs", "ImportJob", "GetAddressListImportJob"},
		{"export", "", "", "ExportId", "", "", "", "", "Exports", "ExportSummary", "GetArchiveExport"},
		{"search", "", "", "SearchId", "", "", "", "", "Searches", "SearchSummary", "GetArchiveSearch"},
	} {
		k := k
		out[k.id] = &k
	}
	return out
}()

type mmgrRes struct {
	kind             *mmgrKind
	id, arn, name    string
	token            string // ClientToken the resource was created with
	parent           string // subscription, address list or archive id
	seq              int64
	fields           map[string]any // conformed model members owned by the store
	tags             map[string]string
	status           string
	created, updated time.Time
	// address list
	members map[string]time.Time
	// ingress point
	pwVersion   int
	prevPwUntil time.Time
	// import job, search, export
	upload            []byte
	uploadKey         string
	imported, failed  int
	started, finished time.Time
	errMsg            string
	rows              []string // search: archived message ids
}

type mmgrStore struct {
	mu          sync.Mutex
	model       *awsmodels.Model
	res         map[string]*mmgrRes // by id
	seq         int64
	memberCount int
	msgs        map[string]*mmgrMessage
	msgOrder    []*mmgrMessage // oldest first, for eviction
	msgBytes    int
	uploadBytes int
}

func newMailManagerStore() *mmgrStore {
	return &mmgrStore{model: awsmodels.MailManagerModel(), res: map[string]*mmgrRes{}, msgs: map[string]*mmgrMessage{}}
}

func mmgrNewID(prefix string) string {
	return prefix + strings.ReplaceAll(uuid.NewString(), "-", "")[:20]
}

func mmgrTags(in map[string]any) (map[string]string, *mmgrErr) {
	out := map[string]string{}
	for _, item := range mmgrList(in, "Tags") {
		t, _ := item.(map[string]any)
		key := mmgrStr(t, "Key")
		if strings.HasPrefix(strings.ToLower(key), "aws:") {
			return nil, mmgrInvalid("Tag keys can't start with aws:.")
		}
		out[key] = mmgrStr(t, "Value")
	}
	if len(out) > mmgrMaxTags {
		return nil, mmgrError("ServiceQuotaExceededException", "A resource can have at most %d tags.", mmgrMaxTags)
	}
	return out, nil
}

// add registers a new resource; callers hold the lock.
func (st *mmgrStore) add(kind string, in map[string]any, fields map[string]any, now time.Time) (*mmgrRes, *mmgrErr) {
	if len(st.res) >= mmgrMaxResources {
		return nil, mmgrError("ServiceQuotaExceededException", "Stackyard keeps at most %d Mail Manager resources.", mmgrMaxResources)
	}
	tags, err := mmgrTags(in)
	if err != nil {
		return nil, err
	}
	k := mmgrKinds[kind]
	id := mmgrNewID(k.prefix)
	if k.prefix == "" {
		id = uuid.NewString()
	}
	st.seq++
	r := &mmgrRes{kind: k, id: id, seq: st.seq, fields: fields, tags: tags, token: mmgrStr(in, "ClientToken"), created: now, updated: now}
	if k.arnType != "" {
		r.arn = mmgrARNBase + k.arnType + "/" + id
	}
	if k.nameKey != "" {
		r.name = mmgrStr(in, k.nameKey)
	}
	st.res[id] = r
	return r, nil
}

// byToken returns the resource of kind created with token (idempotent retries).
func (st *mmgrStore) byToken(kind, token string) *mmgrRes {
	if token == "" {
		return nil
	}
	for _, r := range st.res {
		if r.kind.id == kind && r.token == token {
			return r
		}
	}
	return nil
}

func (st *mmgrStore) nameTaken(kind, name, except string) *mmgrErr {
	for _, r := range st.res {
		if r.kind.id == kind && r.name == name && r.id != except {
			return mmgrError("ConflictException", "A resource named %s already exists.", name)
		}
	}
	return nil
}

// get returns the resource of kind with id (or ARN), or ResourceNotFoundException.
func (st *mmgrStore) get(kind, ref string) (*mmgrRes, *mmgrErr) {
	if r := st.resolve(kind, ref); r != nil {
		return r, nil
	}
	return nil, mmgrError("ResourceNotFoundException", "Resource %s was not found.", ref)
}

func (st *mmgrStore) resolve(kind, ref string) *mmgrRes {
	if r := st.res[ref]; r != nil && r.kind.id == kind {
		return r
	}
	if strings.HasPrefix(ref, "arn:") {
		if r := st.res[ref[strings.LastIndex(ref, "/")+1:]]; r != nil && r.kind.id == kind && r.arn == ref {
			return r
		}
	}
	return nil
}

// view is everything a Get or List can report about r; callers conform it.
func (st *mmgrStore) view(r *mmgrRes, now time.Time) map[string]any {
	st.settle(r, now)
	out := map[string]any{}
	for k, v := range r.fields {
		out[k] = v
	}
	k := r.kind
	out[k.idKey] = r.id
	if k.arnKey != "" {
		out[k.arnKey] = r.arn
	}
	if k.nameKey != "" {
		out[k.nameKey] = r.name
	}
	if k.createdKey != "" {
		out[k.createdKey] = mmgrEpoch(r.created)
	}
	if k.updatedKey != "" {
		out[k.updatedKey] = mmgrEpoch(r.updated)
	}
	st.viewKind(r, out)
	return out
}

func (st *mmgrStore) getOne(kind, ref string) (map[string]any, *mmgrErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	r, err := st.get(kind, ref)
	if err != nil {
		return nil, err
	}
	return st.view(r, time.Now().UTC()), nil
}

// list pages the resources of kind (optionally under parent), oldest first.
func (st *mmgrStore) list(kind, parent string, in map[string]any) (map[string]any, *mmgrErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	k := mmgrKinds[kind]
	var matches []*mmgrRes
	for _, r := range st.res {
		if r.kind == k && (parent == "" || r.parent == parent) {
			matches = append(matches, r)
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].seq < matches[j].seq })
	start, end, next, ok := mmgrPage(len(matches), mmgrStr(in, "NextToken"), mmgrInt(in, "PageSize", 0), mmgrDefaultPage)
	if !ok {
		return nil, mmgrInvalid("NextToken is not valid.")
	}
	now := time.Now().UTC()
	items := make([]any, 0, end-start)
	for _, r := range matches[start:end] {
		items = append(items, st.model.Conform(k.summary, st.view(r, now)))
	}
	out := map[string]any{k.listKey: items}
	if next != "" {
		out["NextToken"] = next
	}
	return out, nil
}

// referencedBy names a rule set, traffic policy or ingress point that points
// at r, or "".
func (st *mmgrStore) referencedBy(r *mmgrRes) string {
	for _, other := range st.res {
		switch other.kind.id {
		case "ingressPoint":
			if mmgrStr(other.fields, "RuleSetId") == r.id || mmgrStr(other.fields, "TrafficPolicyId") == r.id {
				return other.id
			}
		case "ruleSet", "trafficPolicy":
			for _, ref := range mmgrRefs(other.fields) {
				if _, id, _ := strings.Cut(ref, ":"); id == r.id || id == r.arn {
					return other.id
				}
			}
		}
	}
	return ""
}

// mmgrRefs collects the archive, relay and address list references in rules
// and policy statements, as "kind:id-or-arn".
func mmgrRefs(v any) []string {
	var out []string
	var walk func(any)
	walk = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			for k, item := range t {
				switch k {
				case "TargetArchive":
					out = append(out, "archive:"+mmgrStr(t, k))
				case "AddressLists":
					for _, a := range mmgrStrings(item) {
						out = append(out, "addressList:"+a)
					}
				case "Relay":
					if s, ok := item.(string); ok {
						out = append(out, "relay:"+s)
					}
				}
				walk(item)
			}
		case []any:
			for _, item := range t {
				walk(item)
			}
		}
	}
	walk(v)
	return out
}

// checkRefs requires every archive, relay and address list the rules or
// statements name to exist.
func (st *mmgrStore) checkRefs(v any) *mmgrErr {
	var missing []string
	for _, ref := range mmgrRefs(v) {
		kind, id, _ := strings.Cut(ref, ":")
		if st.resolve(kind, id) == nil {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		return mmgrInvalid("Referenced resources don't exist: %s.", strings.Join(missing, ", "))
	}
	return nil
}
