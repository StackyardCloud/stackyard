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
	wdocOrgID        = "d-90670a1b2c"
	wdocAdminName    = "stackyard-admin"
	wdocMaxUsers     = 500
	wdocMaxResources = 5000 // folders and documents
	wdocMaxVersions  = 100  // per document
	wdocMaxComments  = 1000 // per version
	wdocMaxPerms     = 100  // shares per resource
	wdocMaxLabels    = 20
	wdocMaxMetadata  = 8
	wdocMaxSubs      = 256
	wdocMaxActivity  = 1000 // most recent activities kept
	wdocMaxDepth     = 32   // folder nesting
	wdocMaxUpload    = 8 << 20
	wdocMaxBytes     = 64 << 20 // document bytes plus comment, label and metadata text
	wdocDefaultQuota = 1 << 40
	wdocDefaultPage  = 100
)

type wdocUser struct {
	id, username, email, given, surname, status, typ, tz, locale string
	root, recycle                                                string
	created, modified                                            time.Time
	storageType                                                  string
	allocated, used                                              int64
}

type wdocComment struct {
	id, parent, thread, text, contributor, visibility, recipient string
	created                                                      time.Time
}

type wdocVersion struct {
	id, name, contentType, creator, signature string
	status                                    string // INITIALIZED or ACTIVE
	deleted                                   bool
	created, modified, cCreated, cModified    time.Time
	declared                                  int64 // DocumentSizeInBytes, or -1
	data                                      []byte
	uploaded                                  bool
	uploadKey, downloadKey                    string
	meta                                      map[string]string
	comments                                  []*wdocComment
}

type wdocPerm struct{ typ, role string }

type wdocRes struct {
	id, kind, name, parent, creator, state string
	created, modified                      time.Time
	labels                                 []string
	meta                                   map[string]string
	children                               map[string]string // folders: name -> child id
	versions                               []*wdocVersion    // documents, oldest first
	perms                                  map[string]wdocPerm
	bytes                                  int // charged against wdocMaxBytes
}

type wdocActivity struct {
	typ, resource, user string
	at                  time.Time
	view                map[string]any
}

type wdocStore struct {
	mu         sync.Mutex
	users      map[string]*wdocUser
	usernames  map[string]string // lower-cased username -> id
	admin      string
	res        map[string]*wdocRes
	subs       map[string]map[string]any
	subOrder   []string
	sharedWith map[string]map[string]bool // principal -> resources shared with it
	acts       []wdocActivity
	bytes      int
}

var wdocGroups = []map[string]any{
	{"Id": "S-1-5-21-stackyard-513", "Name": "Domain Users"},
	{"Id": "S-1-5-21-stackyard-512", "Name": "Domain Admins"},
	{"Id": "S-1-5-21-stackyard-1104", "Name": "AWS Delegated Administrators"},
}

func newWorkDocsStore() *wdocStore {
	st := &wdocStore{
		users: map[string]*wdocUser{}, usernames: map[string]string{}, res: map[string]*wdocRes{},
		subs: map[string]map[string]any{}, sharedWith: map[string]map[string]bool{},
	}
	admin, _ := st.addUser(map[string]any{"Username": wdocAdminName, "GivenName": "Stackyard", "Surname": "Administrator", "EmailAddress": "admin@example.com"})
	admin.typ = "ADMIN"
	st.admin = admin.id
	return st
}

func wdocNewID() string { return strings.ReplaceAll(uuid.NewString(), "-", "") }

func wdocEpoch(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return float64(t.UnixMilli()) / 1000
}

type wdocReq struct {
	in    map[string]any
	actor *wdocUser
	base  string
}

type wdocHandler func(st *wdocStore, q wdocReq) (map[string]any, *wdocErr)

var wdocHandlers map[string]wdocHandler

func init() {
	wdocHandlers = map[string]wdocHandler{
		"CreateUser": (*wdocStore).createUser, "UpdateUser": (*wdocStore).updateUser, "DeleteUser": (*wdocStore).deleteUser,
		"ActivateUser": (*wdocStore).activateUser, "DeactivateUser": (*wdocStore).deactivateUser,
		"DescribeUsers": (*wdocStore).describeUsers, "GetCurrentUser": (*wdocStore).currentUser,
		"DescribeGroups": (*wdocStore).describeGroups, "DescribeActivities": (*wdocStore).describeActivities,
		"CreateNotificationSubscription":    (*wdocStore).createSub,
		"DeleteNotificationSubscription":    (*wdocStore).deleteSub,
		"DescribeNotificationSubscriptions": (*wdocStore).describeSubs,
	}
	for _, m := range []map[string]wdocHandler{wdocDocHandlers(), wdocShareHandlers()} {
		for name, h := range m {
			wdocHandlers[name] = h
		}
	}
}

// run resolves the acting user, executes op under the store lock and
// marshals its conformed output there.
func (st *wdocStore) run(op string, in map[string]any, token, base string) ([]byte, *wdocErr) {
	h := wdocHandlers[op]
	if h == nil {
		return nil, wdocInvalid("unknown operation %s", op)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	actor := st.users[st.admin]
	if token != "" {
		actor = st.users[strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))]
		if actor == nil || actor.status != "ACTIVE" {
			return nil, wdocError(http.StatusForbidden, "UnauthorizedOperationException", "The authentication token is not valid.")
		}
	} else if op == "GetCurrentUser" || op == "DescribeRootFolders" {
		return nil, wdocInvalid("%s requires an AuthenticationToken", op)
	}
	if org := fcStr(in, "OrganizationId"); org != "" && org != wdocOrgID {
		return nil, wdocMissing("organization", org)
	}
	out, err := h(st, wdocReq{in: in, actor: actor, base: base})
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	model := awsmodels.WorkDocsModel()
	body, _ := json.Marshal(model.Conform(model.Operations[op].Output, out))
	return body, nil
}

// charge accounts n more stored bytes to r (n may be negative).
func (st *wdocStore) charge(r *wdocRes, n int) *wdocErr {
	if n > 0 && st.bytes+n > wdocMaxBytes {
		return wdocError(http.StatusConflict, "StorageLimitExceededException", "Stackyard stores at most %d MiB of WorkDocs data", wdocMaxBytes>>20)
	}
	st.bytes += n
	if r != nil {
		r.bytes += n
	}
	return nil
}

// wdocPage pages n items by an offset marker and a limit (1-999 by the model).
func wdocPage(n int, in map[string]any, markerKey, limitKey string) (int, int, string, *wdocErr) {
	start := 0
	if tok := fcStr(in, markerKey); tok != "" {
		v, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || v < 0 || int(v) > n {
			return 0, 0, "", wdocInvalid("%s is not valid", markerKey)
		}
		start = int(v)
	}
	size := wdocDefaultPage
	if v, ok := fcNum(in, limitKey); ok && v >= 1 && v <= 999 {
		size = int(v)
	}
	end := min(start+size, n)
	next := ""
	if end < n {
		next = strconv.Itoa(end)
	}
	return start, end, next, nil
}

func wdocWithMarker(out map[string]any, next string) map[string]any {
	if next != "" {
		out["Marker"] = next
	}
	return out
}

// ---- activities ----

func (st *wdocStore) record(typ string, actor *wdocUser, r *wdocRes, version string, extra map[string]any) {
	now := time.Now().UTC()
	view := map[string]any{
		"Type": typ, "TimeStamp": wdocEpoch(now), "IsIndirectActivity": false, "OrganizationId": wdocOrgID,
		"Initiator": actor.meta(),
	}
	if r != nil {
		rm := map[string]any{"Type": r.kind, "Name": r.name, "Id": r.id, "ParentId": r.parent}
		if owner := st.users[r.creator]; owner != nil {
			rm["Owner"] = owner.meta()
		}
		if version != "" {
			rm["VersionId"] = version
		}
		view["ResourceMetadata"] = rm
	}
	for k, v := range extra {
		view[k] = v
	}
	if len(st.acts) >= wdocMaxActivity {
		st.acts = append(st.acts[:0:0], st.acts[1:]...)
	}
	resID := ""
	if r != nil {
		resID = r.id
	}
	st.acts = append(st.acts, wdocActivity{typ: typ, resource: resID, user: actor.id, at: now, view: view})
}

func (st *wdocStore) describeActivities(q wdocReq) (map[string]any, *wdocErr) {
	in := q.in
	var types []string
	if raw := fcStr(in, "ActivityTypes"); raw != "" {
		types = strings.Split(raw, ",")
	}
	start, hasStart := mtkTime(in["StartTime"])
	end, hasEnd := mtkTime(in["EndTime"])
	resID, userID := fcStr(in, "ResourceId"), fcStr(in, "UserId")
	var items []any
	for i := len(st.acts) - 1; i >= 0; i-- { // newest first
		a := st.acts[i]
		switch {
		case len(types) > 0 && !fcContains(types, a.typ),
			hasStart && a.at.Before(start), hasEnd && a.at.After(end),
			resID != "" && a.resource != resID, userID != "" && a.user != userID:
			continue
		}
		items = append(items, a.view)
	}
	from, to, next, err := wdocPage(len(items), in, "Marker", "Limit")
	if err != nil {
		return nil, err
	}
	return wdocWithMarker(map[string]any{"UserActivities": append([]any{}, items[from:to]...)}, next), nil
}

// ---- users ----

func (u *wdocUser) meta() map[string]any {
	return map[string]any{"Id": u.id, "Username": u.username, "GivenName": u.given, "Surname": u.surname, "EmailAddress": u.email}
}

func (u *wdocUser) view() map[string]any {
	out := u.meta()
	for k, v := range map[string]any{
		"OrganizationId": wdocOrgID, "RootFolderId": u.root, "RecycleBinFolderId": u.recycle, "Status": u.status, "Type": u.typ,
		"CreatedTimestamp": wdocEpoch(u.created), "ModifiedTimestamp": wdocEpoch(u.modified), "TimeZoneId": u.tz, "Locale": u.locale,
		"Storage": map[string]any{"StorageUtilizedInBytes": u.used, "StorageRule": map[string]any{"StorageAllocatedInBytes": u.allocated, "StorageType": u.storageType}},
	} {
		out[k] = v
	}
	return out
}

// addUser creates a user with a root folder and a recycle bin. The password
// is validated by the model and never stored.
func (st *wdocStore) addUser(in map[string]any) (*wdocUser, *wdocErr) {
	name := fcStr(in, "Username")
	if _, taken := st.usernames[strings.ToLower(name)]; taken {
		return nil, wdocError(http.StatusConflict, "EntityAlreadyExistsException", "The user %s already exists.", name)
	}
	if len(st.users) >= wdocMaxUsers || len(st.res)+2 > wdocMaxResources {
		return nil, wdocLimit("Stackyard keeps at most %d WorkDocs users", wdocMaxUsers)
	}
	now := time.Now().UTC()
	u := &wdocUser{
		id: "S-1-1-" + wdocNewID(), username: name, email: fcStr(in, "EmailAddress"), given: fcStr(in, "GivenName"),
		surname: fcStr(in, "Surname"), status: "ACTIVE", typ: "USER", tz: fcStr(in, "TimeZoneId"), locale: "default",
		created: now, modified: now, storageType: "QUOTA", allocated: wdocDefaultQuota,
	}
	if u.tz == "" {
		u.tz = "GMT"
	}
	if err := u.applyStorage(fcObj(in, "StorageRule")); err != nil {
		return nil, err
	}
	for _, kind := range []string{"root", "recycle"} {
		f := &wdocRes{id: wdocNewID(), kind: "FOLDER", creator: u.id, state: "ACTIVE", created: now, modified: now, children: map[string]string{}, perms: map[string]wdocPerm{}}
		st.res[f.id] = f
		if kind == "root" {
			u.root = f.id
		} else {
			u.recycle = f.id
		}
	}
	st.users[u.id] = u
	st.usernames[strings.ToLower(name)] = u.id
	return u, nil
}

func (u *wdocUser) applyStorage(rule map[string]any) *wdocErr {
	if rule == nil {
		return nil
	}
	if t := fcStr(rule, "StorageType"); t != "" {
		u.storageType = t
	}
	if n, ok := fcNum(rule, "StorageAllocatedInBytes"); ok {
		if n < 0 || n > 1<<50 {
			return wdocInvalid("StorageAllocatedInBytes is out of range")
		}
		u.allocated = int64(n)
	}
	return nil
}

func (st *wdocStore) user(id string) (*wdocUser, *wdocErr) {
	u := st.users[id]
	if u == nil {
		return nil, wdocMissing("user", id)
	}
	return u, nil
}

func (st *wdocStore) createUser(q wdocReq) (map[string]any, *wdocErr) {
	u, err := st.addUser(q.in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"User": u.view()}, nil
}

func (st *wdocStore) updateUser(q wdocReq) (map[string]any, *wdocErr) {
	u, err := st.user(fcStr(q.in, "UserId"))
	if err != nil {
		return nil, err
	}
	next := *u
	for key, field := range map[string]*string{"GivenName": &next.given, "Surname": &next.surname, "Type": &next.typ, "TimeZoneId": &next.tz, "Locale": &next.locale} {
		if v, ok := q.in[key].(string); ok {
			*field = v
		}
	}
	switch fcStr(q.in, "GrantPoweruserPrivileges") {
	case "TRUE":
		next.typ = "POWERUSER"
	case "FALSE":
		if next.typ == "POWERUSER" {
			next.typ = "USER"
		}
	}
	if u.id == st.admin && next.typ != "ADMIN" {
		return nil, wdocProhibited("the organization's administrator must stay an ADMIN")
	}
	if err := next.applyStorage(fcObj(q.in, "StorageRule")); err != nil {
		return nil, err
	}
	next.modified = time.Now().UTC()
	*u = next
	return map[string]any{"User": u.view()}, nil
}

func (st *wdocStore) activateUser(q wdocReq) (map[string]any, *wdocErr) {
	u, err := st.user(fcStr(q.in, "UserId"))
	if err != nil {
		return nil, err
	}
	u.status, u.modified = "ACTIVE", time.Now().UTC()
	return map[string]any{"User": u.view()}, nil
}

func (st *wdocStore) deactivateUser(q wdocReq) (map[string]any, *wdocErr) {
	u, err := st.user(fcStr(q.in, "UserId"))
	if err != nil {
		return nil, err
	}
	if u.id == st.admin {
		return nil, wdocError(http.StatusConflict, "DeactivatingLastSystemUserException", "The last system user can't be deactivated.")
	}
	u.status, u.modified = "INACTIVE", time.Now().UTC()
	return nil, nil
}

// deleteUser removes the user and everything in their folders.
func (st *wdocStore) deleteUser(q wdocReq) (map[string]any, *wdocErr) {
	u, err := st.user(fcStr(q.in, "UserId"))
	if err != nil {
		return nil, err
	}
	if u.id == st.admin {
		return nil, wdocProhibited("the organization's administrator can't be deleted")
	}
	st.removeTree(st.res[u.root])
	st.removeTree(st.res[u.recycle])
	delete(st.sharedWith, u.id)
	delete(st.usernames, strings.ToLower(u.username))
	delete(st.users, u.id)
	return nil, nil
}

func (st *wdocStore) currentUser(q wdocReq) (map[string]any, *wdocErr) {
	return map[string]any{"User": q.actor.view()}, nil
}

func (st *wdocStore) describeUsers(q wdocReq) (map[string]any, *wdocErr) {
	in := q.in
	var ids []string
	if raw := fcStr(in, "UserIds"); raw != "" {
		for _, id := range strings.Split(raw, ",") {
			ids = append(ids, strings.TrimSpace(id))
		}
	}
	query := strings.ToLower(fcStr(in, "Query"))
	activeOnly := fcStr(in, "Include") == "ACTIVE_PENDING"
	var list []*wdocUser
	for _, u := range st.users {
		switch {
		case len(ids) > 0 && !fcContains(ids, u.id),
			activeOnly && u.status == "INACTIVE",
			query != "" && !strings.HasPrefix(strings.ToLower(u.username), query) && !strings.HasPrefix(strings.ToLower(u.given), query) && !strings.HasPrefix(strings.ToLower(u.surname), query):
			continue
		}
		list = append(list, u)
	}
	key := func(u *wdocUser) string {
		switch fcStr(in, "Sort") {
		case "FULL_NAME":
			return strings.ToLower(u.given + " " + u.surname)
		case "STORAGE_LIMIT":
			return strconv.FormatInt(u.allocated+1<<51, 10)
		case "STORAGE_USED":
			return strconv.FormatInt(u.used+1<<51, 10)
		case "USER_STATUS":
			return u.status
		}
		return strings.ToLower(u.username)
	}
	desc := fcStr(in, "Order") == "DESCENDING"
	sort.Slice(list, func(i, j int) bool {
		a, b := key(list[i]), key(list[j])
		if a == b {
			return list[i].id < list[j].id
		}
		return (a < b) != desc
	})
	start, end, next, err := wdocPage(len(list), in, "Marker", "Limit")
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, u := range list[start:end] {
		items = append(items, u.view())
	}
	return wdocWithMarker(map[string]any{"Users": items, "TotalNumberOfUsers": len(list)}, next), nil
}

func (st *wdocStore) describeGroups(q wdocReq) (map[string]any, *wdocErr) {
	query := strings.ToLower(fcStr(q.in, "SearchQuery"))
	var items []any
	for _, g := range wdocGroups {
		if name, _ := g["Name"].(string); strings.Contains(strings.ToLower(name), query) {
			items = append(items, map[string]any{"Id": g["Id"], "Name": name})
		}
	}
	start, end, next, err := wdocPage(len(items), q.in, "Marker", "Limit")
	if err != nil {
		return nil, err
	}
	return wdocWithMarker(map[string]any{"Groups": append([]any{}, items[start:end]...)}, next), nil
}

func wdocGroup(id string) bool {
	for _, g := range wdocGroups {
		if g["Id"] == id {
			return true
		}
	}
	return false
}

// ---- notification subscriptions ----

func (st *wdocStore) createSub(q wdocReq) (map[string]any, *wdocErr) {
	endpoint, proto := fcStr(q.in, "Endpoint"), fcStr(q.in, "Protocol")
	switch {
	case proto == "HTTPS" && !strings.HasPrefix(endpoint, "https://"):
		return nil, wdocInvalid("an HTTPS subscription needs an https:// endpoint")
	case proto == "SQS" && !strings.HasPrefix(endpoint, "arn:aws:sqs:"):
		return nil, wdocInvalid("an SQS subscription needs a queue ARN")
	case len(st.subs) >= wdocMaxSubs:
		return nil, wdocError(http.StatusTooManyRequests, "TooManySubscriptionsException", "An organization can have at most %d subscriptions.", wdocMaxSubs)
	}
	sub := map[string]any{"SubscriptionId": wdocNewID(), "EndPoint": endpoint, "Protocol": proto}
	id, _ := sub["SubscriptionId"].(string)
	st.subs[id] = sub
	st.subOrder = append(st.subOrder, id)
	return map[string]any{"Subscription": sub}, nil
}

func (st *wdocStore) deleteSub(q wdocReq) (map[string]any, *wdocErr) {
	id := fcStr(q.in, "SubscriptionId")
	if st.subs[id] == nil {
		return nil, wdocMissing("subscription", id)
	}
	delete(st.subs, id)
	for i, s := range st.subOrder {
		if s == id {
			st.subOrder = append(st.subOrder[:i:i], st.subOrder[i+1:]...)
			break
		}
	}
	return nil, nil
}

func (st *wdocStore) describeSubs(q wdocReq) (map[string]any, *wdocErr) {
	start, end, next, err := wdocPage(len(st.subOrder), q.in, "Marker", "Limit")
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, id := range st.subOrder[start:end] {
		items = append(items, st.subs[id])
	}
	return wdocWithMarker(map[string]any{"Subscriptions": items}, next), nil
}
