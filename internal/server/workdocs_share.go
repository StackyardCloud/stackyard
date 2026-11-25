package server

import (
	"net/http"
	"sort"
	"strings"
)

// Sharing (resource permissions), labels, custom metadata, GetResources and
// SearchResources.

func wdocShareHandlers() map[string]wdocHandler {
	return map[string]wdocHandler{
		"AddResourcePermissions": (*wdocStore).addPermissions, "DescribeResourcePermissions": (*wdocStore).describePermissions,
		"RemoveResourcePermission": (*wdocStore).removePermission, "RemoveAllResourcePermissions": (*wdocStore).removeAllPermissions,
		"CreateLabels": (*wdocStore).createLabels, "DeleteLabels": (*wdocStore).deleteLabels,
		"CreateCustomMetadata": (*wdocStore).createMetadata, "DeleteCustomMetadata": (*wdocStore).deleteMetadata,
		"GetResources": (*wdocStore).getResources, "SearchResources": (*wdocStore).searchResources,
	}
}

func wdocShareFailure(id, role, msg string) map[string]any {
	return map[string]any{"PrincipalId": id, "Role": role, "Status": "FAILURE", "StatusMessage": msg}
}

func (st *wdocStore) addPermissions(q wdocReq) (map[string]any, *wdocErr) {
	r, err := st.resource(fcStr(q.in, "ResourceId"), "")
	if err != nil {
		return nil, err
	}
	results := []any{}
	shared := false
	for _, p := range fcObjList(q.in["Principals"]) {
		id, typ, role := fcStr(p, "Id"), fcStr(p, "Type"), fcStr(p, "Role")
		known := true
		switch typ {
		case "USER":
			known = st.users[id] != nil
		case "GROUP":
			known = wdocGroup(id)
		case "ORGANIZATION":
			known = id == wdocOrgID
		case "INVITE":
			known = strings.Contains(id, "@")
		}
		_, had := r.perms[id]
		switch {
		case !known:
			results = append(results, wdocShareFailure(id, role, "The principal "+id+" does not exist."))
		case role == "OWNER" || id == r.creator:
			results = append(results, wdocShareFailure(id, role, "The owner's role can't be changed by sharing."))
		case !had && len(r.perms) >= wdocMaxPerms:
			results = append(results, wdocShareFailure(id, role, "The resource is shared with too many principals."))
		default:
			r.perms[id] = wdocPerm{typ: typ, role: role}
			if st.sharedWith[id] == nil {
				st.sharedWith[id] = map[string]bool{}
			}
			st.sharedWith[id][r.id] = true
			res := map[string]any{"PrincipalId": id, "Role": role, "Status": "SUCCESS", "ShareId": r.id + "-" + id}
			if typ == "INVITE" {
				res["InviteePrincipalId"] = "invite-" + id
			}
			results = append(results, res)
			shared = true
		}
	}
	if shared {
		st.record(r.kind+"_SHARED", q.actor, r, "", nil)
	}
	return map[string]any{"ShareResults": results}, nil
}

func (st *wdocStore) describePermissions(q wdocReq) (map[string]any, *wdocErr) {
	r, err := st.resource(fcStr(q.in, "ResourceId"), "")
	if err != nil {
		return nil, err
	}
	only := fcStr(q.in, "PrincipalId")
	var items []any
	if only == "" || only == r.creator {
		items = append(items, map[string]any{"Id": r.creator, "Type": "USER", "Roles": []any{map[string]any{"Role": "OWNER", "Type": "DIRECT"}}})
	}
	ids := make([]string, 0, len(r.perms))
	for id := range r.perms {
		if only == "" || only == id {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		p := r.perms[id]
		items = append(items, map[string]any{"Id": id, "Type": p.typ, "Roles": []any{map[string]any{"Role": p.role, "Type": "DIRECT"}}})
	}
	start, end, next, perr := wdocPage(len(items), q.in, "Marker", "Limit")
	if perr != nil {
		return nil, perr
	}
	return wdocWithMarker(map[string]any{"Principals": append([]any{}, items[start:end]...)}, next), nil
}

func (st *wdocStore) unshare(r *wdocRes, id string) {
	delete(r.perms, id)
	delete(st.sharedWith[id], r.id)
	if len(st.sharedWith[id]) == 0 {
		delete(st.sharedWith, id)
	}
}

func (st *wdocStore) removePermission(q wdocReq) (map[string]any, *wdocErr) {
	r, err := st.resource(fcStr(q.in, "ResourceId"), "")
	if err != nil {
		return nil, err
	}
	id := fcStr(q.in, "PrincipalId")
	p, ok := r.perms[id]
	if !ok || fcStr(q.in, "PrincipalType") != "" && fcStr(q.in, "PrincipalType") != p.typ {
		return nil, wdocMissing("permission for principal", id)
	}
	st.unshare(r, id)
	st.record(r.kind+"_UNSHARED", q.actor, r, "", nil)
	return nil, nil
}

func (st *wdocStore) removeAllPermissions(q wdocReq) (map[string]any, *wdocErr) {
	r, err := st.resource(fcStr(q.in, "ResourceId"), "")
	if err != nil {
		return nil, err
	}
	for id := range r.perms {
		st.unshare(r, id)
	}
	st.record(r.kind+"_UNSHARED", q.actor, r, "", nil)
	return nil, nil
}

// ---- labels ----

func (st *wdocStore) createLabels(q wdocReq) (map[string]any, *wdocErr) {
	r, err := st.resource(fcStr(q.in, "ResourceId"), "")
	if err != nil {
		return nil, err
	}
	labels := append([]string{}, r.labels...)
	added := 0
	for _, l := range fcStrs(q.in, "Labels") {
		if !fcContains(labels, l) {
			labels = append(labels, l)
			added += len(l)
		}
	}
	if len(labels) > wdocMaxLabels {
		return nil, wdocError(http.StatusTooManyRequests, "TooManyLabelsException", "A resource can have at most %d labels.", wdocMaxLabels)
	}
	if err := st.charge(r, added); err != nil {
		return nil, err
	}
	r.labels = labels
	return nil, nil
}

func (st *wdocStore) deleteLabels(q wdocReq) (map[string]any, *wdocErr) {
	r, err := st.resource(fcStr(q.in, "ResourceId"), "")
	if err != nil {
		return nil, err
	}
	drop := fcStrs(q.in, "Labels")
	all := fcBool(q.in, "DeleteAll")
	if !all && len(drop) == 0 {
		return nil, wdocInvalid("set Labels or DeleteAll")
	}
	kept := []string{}
	freed := 0
	for _, l := range r.labels {
		if all || fcContains(drop, l) {
			freed += len(l)
			continue
		}
		kept = append(kept, l)
	}
	_ = st.charge(r, -freed)
	r.labels = kept
	return nil, nil
}

// ---- custom metadata (on a resource, or a document version) ----

func (st *wdocStore) metaTarget(in map[string]any) (*wdocRes, *map[string]string, *wdocErr) {
	r, err := st.resource(fcStr(in, "ResourceId"), "")
	if err != nil {
		return nil, nil, err
	}
	if vid := fcStr(in, "VersionId"); vid != "" {
		if r.kind != "DOCUMENT" {
			return nil, nil, wdocInvalid("VersionId only applies to documents")
		}
		v, _, err := r.version(vid)
		if err != nil {
			return nil, nil, err
		}
		return r, &v.meta, nil
	}
	return r, &r.meta, nil
}

func wdocMetaSize(m map[string]string) int {
	n := 0
	for k, v := range m {
		n += len(k) + len(v)
	}
	return n
}

func (st *wdocStore) createMetadata(q wdocReq) (map[string]any, *wdocErr) {
	r, target, err := st.metaTarget(q.in)
	if err != nil {
		return nil, err
	}
	next := make(map[string]string, len(*target)+1)
	for k, v := range *target {
		next[k] = v
	}
	for k, v := range fcObj(q.in, "CustomMetadata") {
		next[k], _ = v.(string)
	}
	if len(next) > wdocMaxMetadata {
		return nil, wdocError(http.StatusTooManyRequests, "CustomMetadataLimitExceededException", "A resource can have at most %d custom metadata keys.", wdocMaxMetadata)
	}
	if err := st.charge(r, wdocMetaSize(next)-wdocMetaSize(*target)); err != nil {
		return nil, err
	}
	*target = next
	return nil, nil
}

func (st *wdocStore) deleteMetadata(q wdocReq) (map[string]any, *wdocErr) {
	r, target, err := st.metaTarget(q.in)
	if err != nil {
		return nil, err
	}
	keys := fcStrs(q.in, "Keys")
	all := fcBool(q.in, "DeleteAll")
	if !all && len(keys) == 0 {
		return nil, wdocInvalid("set Keys or DeleteAll")
	}
	next := map[string]string{}
	for k, v := range *target {
		if !all && !fcContains(keys, k) {
			next[k] = v
		}
	}
	_ = st.charge(r, wdocMetaSize(next)-wdocMetaSize(*target))
	*target = next
	return nil, nil
}

// ---- listing and search ----

func (st *wdocStore) pageResources(list []*wdocRes, in map[string]any) (map[string]any, *wdocErr) {
	sort.Slice(list, func(i, j int) bool {
		return list[i].created.Before(list[j].created) || list[i].created.Equal(list[j].created) && list[i].id < list[j].id
	})
	start, end, next, err := wdocPage(len(list), in, "Marker", "Limit")
	if err != nil {
		return nil, err
	}
	folders, docs := []any{}, []any{}
	for _, r := range list[start:end] {
		if r.kind == "FOLDER" {
			folders = append(folders, st.folderView(r))
		} else {
			docs = append(docs, st.docView(r))
		}
	}
	return wdocWithMarker(map[string]any{"Folders": folders, "Documents": docs}, next), nil
}

// getResources lists what is shared with a user, or the user's root folder
// contents when no collection type is given.
func (st *wdocStore) getResources(q wdocReq) (map[string]any, *wdocErr) {
	u := q.actor
	if id := fcStr(q.in, "UserId"); id != "" {
		var err *wdocErr
		if u, err = st.user(id); err != nil {
			return nil, err
		}
	}
	var list []*wdocRes
	if fcStr(q.in, "CollectionType") == "SHARED_WITH_ME" {
		for id := range st.sharedWith[u.id] {
			if r := st.res[id]; r != nil && r.state == "ACTIVE" {
				list = append(list, r)
			}
		}
	} else if root := st.res[u.root]; root != nil {
		for _, id := range root.children {
			if r := st.res[id]; r != nil && r.state == "ACTIVE" {
				list = append(list, r)
			}
		}
	}
	return st.pageResources(list, q.in)
}

var wdocCategories = map[string][]string{
	"IMAGE": {"image/"}, "PDF": {"application/pdf"}, "AUDIO": {"audio/"}, "VIDEO": {"video/"},
	"DOCUMENT":     {"text/plain", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml", "application/rtf"},
	"SPREADSHEET":  {"text/csv", "application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml"},
	"PRESENTATION": {"application/vnd.ms-powerpoint", "application/vnd.openxmlformats-officedocument.presentationml"},
	"SOURCE_CODE":  {"text/x-", "application/json", "application/xml", "text/html", "application/javascript"},
}

func wdocCategory(contentType string) string {
	for cat, prefixes := range wdocCategories {
		for _, p := range prefixes {
			if strings.HasPrefix(contentType, p) {
				return cat
			}
		}
	}
	return "OTHER"
}

// searchMatch applies SearchResources' query and filters to one resource.
func (st *wdocStore) searchMatch(r *wdocRes, query string, filters map[string]any, actor *wdocUser) bool {
	if r.state != "ACTIVE" || r.name == "" {
		return false
	}
	if query != "" && !strings.Contains(strings.ToLower(r.name), query) {
		return false
	}
	if types := fcStrs(filters, "ResourceTypes"); len(types) > 0 && !fcContains(types, r.kind) {
		return false
	}
	for _, l := range fcStrs(filters, "Labels") {
		if !fcContains(r.labels, l) {
			return false
		}
	}
	size := int64(0)
	if v := r.latest(); v != nil {
		size = int64(len(v.data))
	}
	if cats := fcStrs(filters, "ContentCategories"); len(cats) > 0 {
		v := r.latest()
		if r.kind != "DOCUMENT" || v == nil || !fcContains(cats, wdocCategory(v.contentType)) {
			return false
		}
	}
	if rng := fcObj(filters, "SizeRange"); rng != nil {
		if lo, ok := fcNum(rng, "StartValue"); ok && float64(size) < lo {
			return false
		}
		if hi, ok := fcNum(rng, "EndValue"); ok && float64(size) > hi {
			return false
		}
	}
	for key, t := range map[string]float64{"CreatedRange": float64(r.created.UnixMilli()) / 1000, "ModifiedRange": float64(r.modified.UnixMilli()) / 1000} {
		if rng := fcObj(filters, key); rng != nil {
			if lo, ok := mtkTime(rng["StartValue"]); ok && t < float64(lo.UnixMilli())/1000 {
				return false
			}
			if hi, ok := mtkTime(rng["EndValue"]); ok && t > float64(hi.UnixMilli())/1000 {
				return false
			}
		}
	}
	if colls := fcStrs(filters, "SearchCollectionTypes"); len(colls) > 0 {
		owned := r.creator == actor.id
		shared := st.sharedWith[actor.id][r.id]
		if !(fcContains(colls, "OWNED") && owned || fcContains(colls, "SHARED_WITH_ME") && shared) {
			return false
		}
	}
	if ancestors := fcStrs(filters, "AncestorIds"); len(ancestors) > 0 {
		found := false
		for a, n := st.res[r.parent], 0; a != nil && n <= wdocMaxDepth && !found; a, n = st.res[a.parent], n+1 {
			found = fcContains(ancestors, a.id)
		}
		if !found {
			return false
		}
	}
	for _, p := range fcObjList(filters["Principals"]) {
		perm, ok := r.perms[fcStr(p, "Id")]
		role := perm.role
		if fcStr(p, "Id") == r.creator {
			ok, role = true, "OWNER"
		}
		if !ok || len(fcStrs(p, "Roles")) > 0 && !fcContains(fcStrs(p, "Roles"), role) {
			return false
		}
	}
	return true
}

// searchResources scans the (bounded) resource set by name and filters.
func (st *wdocStore) searchResources(q wdocReq) (map[string]any, *wdocErr) {
	query := strings.ToLower(fcStr(q.in, "QueryText"))
	filters := fcObj(q.in, "Filters")
	var list []*wdocRes
	for _, r := range st.res {
		if st.searchMatch(r, query, filters, q.actor) {
			list = append(list, r)
		}
	}
	field, desc := "RELEVANCE", false
	if sorts := fcObjList(q.in["OrderBy"]); len(sorts) > 0 {
		field, desc = fcStr(sorts[0], "Field"), fcStr(sorts[0], "Order") == "DESC"
	}
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		less := a.id < b.id
		switch field {
		case "NAME":
			less = strings.ToLower(a.name) < strings.ToLower(b.name) || strings.EqualFold(a.name, b.name) && a.id < b.id
		case "CREATED_TIMESTAMP":
			less = a.created.Before(b.created) || a.created.Equal(b.created) && a.id < b.id
		case "MODIFIED_TIMESTAMP":
			less = a.modified.Before(b.modified) || a.modified.Equal(b.modified) && a.id < b.id
		case "SIZE":
			less = a.bytes < b.bytes || a.bytes == b.bytes && a.id < b.id
		default: // relevance: exact name matches first, then by name
			ea, eb := strings.ToLower(a.name) == query, strings.ToLower(b.name) == query
			less = ea && !eb || ea == eb && (strings.ToLower(a.name) < strings.ToLower(b.name) || strings.EqualFold(a.name, b.name) && a.id < b.id)
		}
		return less != desc
	})
	limit := 100
	if v, ok := fcNum(q.in, "Limit"); ok && v >= 1 {
		limit = int(min(v, 100))
	}
	start, end, next, err := wdocPage(len(list), map[string]any{"Marker": q.in["Marker"], "Limit": float64(limit)}, "Marker", "Limit")
	if err != nil {
		return nil, err
	}
	web := fcContains(fcStrs(q.in, "AdditionalResponseFields"), "WEBURL")
	items := make([]any, 0, end-start)
	for _, r := range list[start:end] {
		item := map[string]any{"ResourceType": r.kind}
		if r.kind == "FOLDER" {
			item["FolderMetadata"] = st.folderView(r)
		} else {
			item["DocumentMetadata"] = st.docView(r)
		}
		if web {
			item["WebUrl"] = "https://" + wdocOrgID + ".awsapps.com/workdocs/index.html#/" + strings.ToLower(r.kind) + "/" + r.id
		}
		items = append(items, item)
	}
	return wdocWithMarker(map[string]any{"Items": items}, next), nil
}
