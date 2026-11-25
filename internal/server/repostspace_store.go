package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	rpsRegion        = "us-east-1"
	rpsAccount       = "123456789012"
	rpsMaxSpaces     = 25
	rpsMaxChannels   = 100   // per space
	rpsMaxAccessors  = 5000  // accessors holding a role, per space and per channel
	rpsMaxGrants     = 50000 // accessors holding roles, store-wide
	rpsMaxIDLen      = 256   // accessor and admin IDs
	rpsMaxAdmins     = 100   // per space
	rpsMaxTags       = 50    // per space
	rpsDefaultPage   = 100
	rpsBasicStorage  = 10 << 30   // BASIC tier storage limit (bytes)
	rpsStdStorage    = 1000 << 30 // STANDARD tier storage limit (bytes)
	rpsDomainSuffix  = ".private.repost.aws"
	rpsIdentityStore = "d-9067stackyard"
)

type rpsChannel struct {
	id, name, desc string
	created        time.Time
	readyAt        time.Time
	roles          map[string]map[string]bool // accessor -> channel roles
}

type rpsSpace struct {
	id, name, subdomain, desc, tier string
	roleArn, kmsKey, clientID       string
	random                          string
	domains                         map[string]any // conformed SupportedEmailDomainsParameters
	created, readyAt                time.Time
	deleteAt                        time.Time // zero unless DELETING
	admins                          map[string]bool
	roles                           map[string]map[string]bool // accessor -> space roles
	channels                        map[string]*rpsChannel
	channelNames                    map[string]string // name -> channel id
	tags                            map[string]string
}

type rpsStore struct {
	mu         sync.Mutex
	delay      time.Duration // provisioning and deletion time
	spaces     map[string]*rpsSpace
	subdomains map[string]string // subdomain -> space id
	grants     int               // accessor entries across every role map
}

func newRepostSpaceStore() *rpsStore {
	return &rpsStore{delay: 2 * time.Second, spaces: map[string]*rpsSpace{}, subdomains: map[string]string{}}
}

func rpsNewID() string { return strings.ReplaceAll(uuid.NewString(), "-", "")[:24] }

func rpsSpaceArn(id string) string {
	return "arn:aws:repostspace:" + rpsRegion + ":" + rpsAccount + ":space/" + id
}

type rpsHandler func(st *rpsStore, in map[string]any) (map[string]any, *rpsErr)

var rpsHandlers map[string]rpsHandler

func init() {
	rpsHandlers = map[string]rpsHandler{
		"CreateSpace": (*rpsStore).createSpace, "GetSpace": (*rpsStore).getSpace, "ListSpaces": (*rpsStore).listSpaces,
		"UpdateSpace": (*rpsStore).updateSpace, "DeleteSpace": (*rpsStore).deleteSpace,
		"RegisterAdmin": (*rpsStore).registerAdmin, "DeregisterAdmin": (*rpsStore).deregisterAdmin,
		"BatchAddRole": (*rpsStore).batchAddRole, "BatchRemoveRole": (*rpsStore).batchRemoveRole,
		"SendInvites":   (*rpsStore).sendInvites,
		"CreateChannel": (*rpsStore).createChannel, "GetChannel": (*rpsStore).getChannel,
		"ListChannels": (*rpsStore).listChannels, "UpdateChannel": (*rpsStore).updateChannel,
		"BatchAddChannelRoleToAccessors":      (*rpsStore).batchAddChannelRole,
		"BatchRemoveChannelRoleFromAccessors": (*rpsStore).batchRemoveChannelRole,
		"TagResource":                         (*rpsStore).tagResource, "UntagResource": (*rpsStore).untagResource,
		"ListTagsForResource": (*rpsStore).listTags,
	}
}

// run executes op under the store lock and marshals its conformed output there.
func (st *rpsStore) run(op string, in map[string]any) ([]byte, *rpsErr) {
	h := rpsHandlers[op]
	if h == nil {
		return nil, rpsValidation("unknown operation %s", op)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	out, err := h(st, in)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	model := awsmodels.RepostSpaceModel()
	body, _ := json.Marshal(model.Conform(model.Operations[op].Output, out))
	return body, nil
}

// space returns a live space, settling its deletion; callers hold the lock.
func (st *rpsStore) space(id string) (*rpsSpace, *rpsErr) {
	sp := st.spaces[id]
	if sp != nil && !sp.deleteAt.IsZero() && !time.Now().Before(sp.deleteAt) {
		st.drop(sp)
		sp = nil
	}
	if sp == nil {
		return nil, rpsNotFound("space", id)
	}
	return sp, nil
}

// activeSpace is space, refusing spaces that are being deleted.
func (st *rpsStore) activeSpace(id string) (*rpsSpace, *rpsErr) {
	sp, err := st.space(id)
	if err == nil && !sp.deleteAt.IsZero() {
		return nil, rpsConflict("space", id, "Space "+id+" is being deleted.")
	}
	return sp, err
}

func (st *rpsStore) drop(sp *rpsSpace) {
	st.grants -= len(sp.roles)
	for _, ch := range sp.channels {
		st.grants -= len(ch.roles)
	}
	delete(st.subdomains, sp.subdomain)
	delete(st.spaces, sp.id)
}

func (sp *rpsSpace) status() string {
	now := time.Now()
	switch {
	case !sp.deleteAt.IsZero():
		return "DELETING"
	case now.Before(sp.readyAt):
		return "CREATING"
	}
	return "CREATED"
}

func rpsStorageLimit(tier string) int64 {
	if tier == "STANDARD" {
		return rpsStdStorage
	}
	return rpsBasicStorage
}

func (sp *rpsSpace) summary() map[string]any {
	status := sp.status()
	vanity := "APPROVED"
	if status == "CREATING" {
		vanity = "PENDING"
	}
	configured := "UNCONFIGURED"
	if len(sp.admins) > 0 {
		configured = "CONFIGURED"
	}
	users := map[string]bool{}
	for a := range sp.admins {
		users[a] = true
	}
	for a := range sp.roles {
		users[a] = true
	}
	out := map[string]any{
		"spaceId": sp.id, "arn": rpsSpaceArn(sp.id), "name": sp.name, "status": status,
		"configurationStatus": configured, "vanityDomainStatus": vanity,
		"vanityDomain": "https://" + sp.subdomain + rpsDomainSuffix, "randomDomain": "https://" + sp.random + rpsDomainSuffix,
		"tier": sp.tier, "storageLimit": rpsStorageLimit(sp.tier), "createDateTime": rpsTime(sp.created),
		"userCount": len(users), "contentSize": 0,
	}
	if sp.desc != "" {
		out["description"] = sp.desc
	}
	if sp.kmsKey != "" {
		out["userKMSKey"] = sp.kmsKey
	}
	if !sp.deleteAt.IsZero() {
		out["deleteDateTime"] = rpsTime(sp.deleteAt)
	}
	if sp.domains != nil {
		out["supportedEmailDomains"] = sp.domains
	}
	return out
}

func rpsTime(t time.Time) float64 { return float64(t.UnixMilli()) / 1000 }

func rpsRoleMap(roles map[string]map[string]bool) map[string]any {
	out := make(map[string]any, len(roles))
	for acc, set := range roles {
		list := make([]string, 0, len(set))
		for r := range set {
			list = append(list, r)
		}
		sort.Strings(list)
		out[acc] = fcAnyList(list)
	}
	return out
}

func rpsSortedKeys(m map[string]bool) []any {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return fcAnyList(keys)
}

func rpsCheckTags(tags map[string]any) *rpsErr {
	for k := range tags {
		if strings.HasPrefix(strings.ToLower(k), "aws:") {
			return rpsValidation("tag keys can't start with aws:")
		}
	}
	return nil
}

// ---- spaces ----

func (st *rpsStore) createSpace(in map[string]any) (map[string]any, *rpsErr) {
	sub := strings.ToLower(fcStr(in, "subdomain"))
	if !rpsSubdomain(sub) {
		return nil, rpsValidation("subdomain must be lowercase letters, digits and hyphens, starting and ending with a letter or digit")
	}
	if id, taken := st.subdomains[sub]; taken {
		if _, err := st.space(id); err == nil {
			return nil, rpsConflict("space", id, "Subdomain "+sub+" is already in use.")
		}
	}
	if len(st.spaces) >= rpsMaxSpaces {
		return nil, rpsQuota("space", rpsMaxSpaces)
	}
	tags := fcObj(in, "tags")
	if err := rpsCheckTags(tags); err != nil {
		return nil, err
	}
	if len(tags) > rpsMaxTags {
		return nil, rpsValidation("a space can have at most %d tags", rpsMaxTags)
	}
	now := time.Now().UTC()
	sp := &rpsSpace{
		id: rpsNewID(), name: fcStr(in, "name"), subdomain: sub, desc: fcStr(in, "description"), tier: fcStr(in, "tier"),
		roleArn: fcStr(in, "roleArn"), kmsKey: fcStr(in, "userKMSKey"), clientID: rpsNewID(), random: strings.ToLower(rpsNewID()),
		domains: rpsDomains(fcObj(in, "supportedEmailDomains")), created: now, readyAt: now.Add(st.delay),
		admins: map[string]bool{}, roles: map[string]map[string]bool{}, channels: map[string]*rpsChannel{},
		channelNames: map[string]string{}, tags: map[string]string{},
	}
	for k, v := range tags {
		sp.tags[k], _ = v.(string)
	}
	st.spaces[sp.id] = sp
	st.subdomains[sub] = sp.id
	return map[string]any{"spaceId": sp.id}, nil
}

func rpsSubdomain(s string) bool {
	if s == "" || s[0] == '-' || s[len(s)-1] == '-' {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}

// rpsDomains turns SupportedEmailDomainsParameters into the status shape.
func rpsDomains(p map[string]any) map[string]any {
	if p == nil {
		return nil
	}
	out := map[string]any{"enabled": fcStr(p, "enabled")}
	if out["enabled"] == "" {
		out["enabled"] = "DISABLED"
	}
	if list, ok := p["allowedDomains"].([]any); ok {
		out["allowedDomains"] = list
	}
	return out
}

func (st *rpsStore) getSpace(in map[string]any) (map[string]any, *rpsErr) {
	sp, err := st.space(fcStr(in, "spaceId"))
	if err != nil {
		return nil, err
	}
	out := sp.summary()
	out["clientId"] = sp.clientID
	out["identityStoreId"] = rpsIdentityStore
	out["applicationArn"] = "arn:aws:sso::" + rpsAccount + ":application/ssoins-stackyard/apl-" + sp.id
	if sp.roleArn != "" {
		out["customerRoleArn"] = sp.roleArn
	}
	out["userAdmins"] = rpsSortedKeys(sp.admins)
	out["groupAdmins"] = []any{}
	out["roles"] = rpsRoleMap(sp.roles)
	return out, nil
}

func (st *rpsStore) listSpaces(in map[string]any) (map[string]any, *rpsErr) {
	var live []*rpsSpace
	for id := range st.spaces {
		if sp, err := st.space(id); err == nil {
			live = append(live, sp)
		}
	}
	sort.Slice(live, func(i, j int) bool {
		return live[i].created.Before(live[j].created) || live[i].created.Equal(live[j].created) && live[i].id < live[j].id
	})
	start, end, next, err := rpsPage(len(live), in)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, sp := range live[start:end] {
		items = append(items, sp.summary())
	}
	out := map[string]any{"spaces": items}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

// rpsPage reads nextToken/maxResults (the model bounds maxResults to 1-100).
func rpsPage(n int, in map[string]any) (int, int, string, *rpsErr) {
	start := 0
	if tok := fcStr(in, "nextToken"); tok != "" {
		v, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || v < 0 || int(v) > n {
			return 0, 0, "", rpsValidation("nextToken is not valid")
		}
		start = int(v)
	}
	size := rpsDefaultPage
	if v, ok := fcNum(in, "maxResults"); ok && v >= 1 && v <= rpsDefaultPage {
		size = int(v)
	}
	end := min(start+size, n)
	next := ""
	if end < n {
		next = strconv.Itoa(end)
	}
	return start, end, next, nil
}

func (st *rpsStore) updateSpace(in map[string]any) (map[string]any, *rpsErr) {
	sp, err := st.activeSpace(fcStr(in, "spaceId"))
	if err != nil {
		return nil, err
	}
	if v, ok := in["description"].(string); ok {
		sp.desc = v
	}
	if v, ok := in["tier"].(string); ok {
		sp.tier = v
	}
	if v, ok := in["roleArn"].(string); ok {
		sp.roleArn = v
	}
	if p := fcObj(in, "supportedEmailDomains"); p != nil {
		sp.domains = rpsDomains(p)
	}
	return nil, nil
}

func (st *rpsStore) deleteSpace(in map[string]any) (map[string]any, *rpsErr) {
	sp, err := st.activeSpace(fcStr(in, "spaceId"))
	if err != nil {
		return nil, err
	}
	sp.deleteAt = time.Now().UTC().Add(st.delay)
	return nil, nil
}

func (st *rpsStore) registerAdmin(in map[string]any) (map[string]any, *rpsErr) {
	sp, err := st.activeSpace(fcStr(in, "spaceId"))
	if err != nil {
		return nil, err
	}
	admin := fcStr(in, "adminId")
	if len(admin) > rpsMaxIDLen {
		return nil, rpsValidation("adminId can have at most %d characters", rpsMaxIDLen)
	}
	if !sp.admins[admin] && len(sp.admins) >= rpsMaxAdmins {
		return nil, rpsQuota("admin", rpsMaxAdmins)
	}
	sp.admins[admin] = true
	return nil, nil
}

func (st *rpsStore) deregisterAdmin(in map[string]any) (map[string]any, *rpsErr) {
	sp, err := st.activeSpace(fcStr(in, "spaceId"))
	if err != nil {
		return nil, err
	}
	admin := fcStr(in, "adminId")
	if !sp.admins[admin] {
		return nil, rpsNotFound("admin", admin)
	}
	delete(sp.admins, admin)
	return nil, nil
}

// grant adds role to each accessor in roles, reporting accessors past
// rpsMaxAccessors (or rpsMaxGrants store-wide) as batch errors.
func (st *rpsStore) grant(roles map[string]map[string]bool, accessors []string, role string) (map[string]any, *rpsErr) {
	added, errs := []any{}, []any{}
	for _, acc := range accessors {
		if strings.TrimSpace(acc) == "" || len(acc) > rpsMaxIDLen {
			errs = append(errs, map[string]any{"accessorId": acc, "error": http.StatusBadRequest, "message": "accessor IDs must have 1-256 characters"})
			continue
		}
		set := roles[acc]
		if set == nil {
			if len(roles) >= rpsMaxAccessors || st.grants >= rpsMaxGrants {
				errs = append(errs, map[string]any{"accessorId": acc, "error": http.StatusPaymentRequired, "message": "too many accessors hold roles"})
				continue
			}
			set = map[string]bool{}
			roles[acc] = set
			st.grants++
		}
		set[role] = true
		added = append(added, acc)
	}
	return map[string]any{"addedAccessorIds": added, "errors": errs}, nil
}

func (st *rpsStore) revoke(roles map[string]map[string]bool, accessors []string, role string) map[string]any {
	removed := []any{}
	for _, acc := range accessors {
		if set := roles[acc]; set != nil {
			delete(set, role)
			if len(set) == 0 {
				delete(roles, acc)
				st.grants--
			}
		}
		removed = append(removed, acc)
	}
	return map[string]any{"removedAccessorIds": removed, "errors": []any{}}
}

func (st *rpsStore) batchAddRole(in map[string]any) (map[string]any, *rpsErr) {
	sp, err := st.activeSpace(fcStr(in, "spaceId"))
	if err != nil {
		return nil, err
	}
	return st.grant(sp.roles, fcStrs(in, "accessorIds"), fcStr(in, "role"))
}

func (st *rpsStore) batchRemoveRole(in map[string]any) (map[string]any, *rpsErr) {
	sp, err := st.activeSpace(fcStr(in, "spaceId"))
	if err != nil {
		return nil, err
	}
	return st.revoke(sp.roles, fcStrs(in, "accessorIds"), fcStr(in, "role")), nil
}

// sendInvites checks the space can take members; Stackyard sends no email.
func (st *rpsStore) sendInvites(in map[string]any) (map[string]any, *rpsErr) {
	sp, err := st.activeSpace(fcStr(in, "spaceId"))
	if err != nil {
		return nil, err
	}
	if sp.status() != "CREATED" {
		return nil, rpsConflict("space", sp.id, "Space "+sp.id+" is still being created.")
	}
	for _, acc := range fcStrs(in, "accessorIds") {
		if strings.TrimSpace(acc) == "" {
			return nil, rpsValidation("accessor IDs can't be empty")
		}
	}
	return nil, nil
}

// ---- channels ----

func (ch *rpsChannel) status() string {
	if time.Now().Before(ch.readyAt) {
		return "CREATING"
	}
	return "CREATED"
}

func (ch *rpsChannel) view(spaceID string) map[string]any {
	users := len(ch.roles)
	out := map[string]any{
		"spaceId": spaceID, "channelId": ch.id, "channelName": ch.name, "createDateTime": rpsTime(ch.created),
		"channelStatus": ch.status(), "userCount": users, "groupCount": 0,
	}
	if ch.desc != "" {
		out["channelDescription"] = ch.desc
	}
	return out
}

func (st *rpsStore) channel(in map[string]any) (*rpsSpace, *rpsChannel, *rpsErr) {
	sp, err := st.activeSpace(fcStr(in, "spaceId"))
	if err != nil {
		return nil, nil, err
	}
	id := fcStr(in, "channelId")
	ch := sp.channels[id]
	if ch == nil {
		return nil, nil, rpsNotFound("channel", id)
	}
	return sp, ch, nil
}

func (st *rpsStore) createChannel(in map[string]any) (map[string]any, *rpsErr) {
	sp, err := st.activeSpace(fcStr(in, "spaceId"))
	if err != nil {
		return nil, err
	}
	name := fcStr(in, "channelName")
	if id, taken := sp.channelNames[name]; taken {
		return nil, rpsConflict("channel", id, "A channel named "+name+" already exists.")
	}
	if len(sp.channels) >= rpsMaxChannels {
		return nil, rpsQuota("channel", rpsMaxChannels)
	}
	now := time.Now().UTC()
	ch := &rpsChannel{id: rpsNewID(), name: name, desc: fcStr(in, "channelDescription"), created: now, readyAt: now.Add(st.delay), roles: map[string]map[string]bool{}}
	sp.channels[ch.id] = ch
	sp.channelNames[name] = ch.id
	return map[string]any{"channelId": ch.id}, nil
}

func (st *rpsStore) getChannel(in map[string]any) (map[string]any, *rpsErr) {
	sp, ch, err := st.channel(in)
	if err != nil {
		return nil, err
	}
	out := ch.view(sp.id)
	out["channelRoles"] = rpsRoleMap(ch.roles)
	return out, nil
}

func (st *rpsStore) listChannels(in map[string]any) (map[string]any, *rpsErr) {
	sp, err := st.space(fcStr(in, "spaceId"))
	if err != nil {
		return nil, err
	}
	list := make([]*rpsChannel, 0, len(sp.channels))
	for _, ch := range sp.channels {
		list = append(list, ch)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].created.Before(list[j].created) || list[i].created.Equal(list[j].created) && list[i].id < list[j].id
	})
	start, end, next, perr := rpsPage(len(list), in)
	if perr != nil {
		return nil, perr
	}
	items := make([]any, 0, end-start)
	for _, ch := range list[start:end] {
		items = append(items, ch.view(sp.id))
	}
	out := map[string]any{"channels": items}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

func (st *rpsStore) updateChannel(in map[string]any) (map[string]any, *rpsErr) {
	sp, ch, err := st.channel(in)
	if err != nil {
		return nil, err
	}
	name := fcStr(in, "channelName")
	if id, taken := sp.channelNames[name]; taken && id != ch.id {
		return nil, rpsConflict("channel", id, "A channel named "+name+" already exists.")
	}
	delete(sp.channelNames, ch.name)
	ch.name = name
	sp.channelNames[name] = ch.id
	if v, ok := in["channelDescription"].(string); ok {
		ch.desc = v
	}
	return nil, nil
}

func (st *rpsStore) batchAddChannelRole(in map[string]any) (map[string]any, *rpsErr) {
	_, ch, err := st.channel(in)
	if err != nil {
		return nil, err
	}
	return st.grant(ch.roles, fcStrs(in, "accessorIds"), fcStr(in, "channelRole"))
}

func (st *rpsStore) batchRemoveChannelRole(in map[string]any) (map[string]any, *rpsErr) {
	_, ch, err := st.channel(in)
	if err != nil {
		return nil, err
	}
	return st.revoke(ch.roles, fcStrs(in, "accessorIds"), fcStr(in, "channelRole")), nil
}

// ---- tags ----

func (st *rpsStore) tagged(arn string) (*rpsSpace, *rpsErr) {
	id, ok := strings.CutPrefix(arn, rpsSpaceArn(""))
	if !ok {
		return nil, rpsNotFound("resource", arn)
	}
	sp, err := st.space(id)
	if err != nil {
		return nil, rpsNotFound("resource", arn)
	}
	return sp, nil
}

func (st *rpsStore) tagResource(in map[string]any) (map[string]any, *rpsErr) {
	sp, err := st.tagged(fcStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	tags := fcObj(in, "tags")
	if err := rpsCheckTags(tags); err != nil {
		return nil, err
	}
	added := 0
	for k := range tags {
		if _, ok := sp.tags[k]; !ok {
			added++
		}
	}
	if len(sp.tags)+added > rpsMaxTags {
		return nil, rpsValidation("a space can have at most %d tags", rpsMaxTags)
	}
	for k, v := range tags {
		sp.tags[k], _ = v.(string)
	}
	return nil, nil
}

func (st *rpsStore) untagResource(in map[string]any) (map[string]any, *rpsErr) {
	sp, err := st.tagged(fcStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	for _, k := range fcStrs(in, "tagKeys") {
		delete(sp.tags, k)
	}
	return nil, nil
}

func (st *rpsStore) listTags(in map[string]any) (map[string]any, *rpsErr) {
	sp, err := st.tagged(fcStr(in, "resourceArn"))
	if err != nil {
		return nil, err
	}
	tags := make(map[string]any, len(sp.tags))
	for k, v := range sp.tags {
		tags[k] = v
	}
	return map[string]any{"tags": tags}, nil
}
