package server

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Folders, documents, versions and comments.

func wdocDocHandlers() map[string]wdocHandler {
	return map[string]wdocHandler{
		"CreateFolder": (*wdocStore).createFolder, "GetFolder": (*wdocStore).getFolder, "UpdateFolder": (*wdocStore).updateFolder,
		"DeleteFolder": (*wdocStore).deleteFolder, "DeleteFolderContents": (*wdocStore).deleteFolderContents,
		"DescribeFolderContents": (*wdocStore).describeFolderContents, "DescribeRootFolders": (*wdocStore).describeRootFolders,
		"GetFolderPath": (*wdocStore).getFolderPath, "GetDocumentPath": (*wdocStore).getDocumentPath,
		"InitiateDocumentVersionUpload": (*wdocStore).initiateUpload, "AbortDocumentVersionUpload": (*wdocStore).abortUpload,
		"UpdateDocumentVersion": (*wdocStore).updateVersion, "GetDocumentVersion": (*wdocStore).getVersion,
		"DescribeDocumentVersions": (*wdocStore).describeVersions, "DeleteDocumentVersion": (*wdocStore).deleteVersion,
		"RestoreDocumentVersions": (*wdocStore).restoreVersions,
		"GetDocument":             (*wdocStore).getDocument, "UpdateDocument": (*wdocStore).updateDocument, "DeleteDocument": (*wdocStore).deleteDocument,
		"CreateComment": (*wdocStore).createComment, "DeleteComment": (*wdocStore).deleteComment, "DescribeComments": (*wdocStore).describeComments,
	}
}

func (st *wdocStore) resource(id, kind string) (*wdocRes, *wdocErr) {
	r := st.res[id]
	if r == nil || kind != "" && r.kind != kind {
		if kind == "" {
			kind = "RESOURCE"
		}
		return nil, wdocMissing(strings.ToLower(kind), id)
	}
	return r, nil
}

func (st *wdocStore) isRoot(r *wdocRes) bool {
	u := st.users[r.creator]
	return u != nil && (u.root == r.id || u.recycle == r.id)
}

// depth counts r's ancestors; nesting is capped at wdocMaxDepth.
func (st *wdocStore) depth(r *wdocRes) int {
	n := 0
	for r != nil && r.parent != "" && n <= wdocMaxDepth {
		r = st.res[r.parent]
		n++
	}
	return n
}

func (st *wdocStore) folderView(r *wdocRes) map[string]any {
	out := map[string]any{
		"Id": r.id, "CreatorId": r.creator, "CreatedTimestamp": wdocEpoch(r.created), "ModifiedTimestamp": wdocEpoch(r.modified),
		"ResourceState": r.state, "Labels": fcAnyList(r.labels), "Size": 0, "LatestVersionSize": 0,
	}
	if r.name != "" {
		out["Name"] = r.name
	}
	if r.parent != "" {
		out["ParentFolderId"] = r.parent
	}
	return out
}

// latest is the newest active version, else the newest pending one.
func (r *wdocRes) latest() *wdocVersion {
	var pending *wdocVersion
	for i := len(r.versions) - 1; i >= 0; i-- {
		v := r.versions[i]
		if v.deleted {
			continue
		}
		if v.status == "ACTIVE" {
			return v
		}
		if pending == nil {
			pending = v
		}
	}
	return pending
}

func (st *wdocStore) versionView(r *wdocRes, v *wdocVersion, source bool, base string) map[string]any {
	out := map[string]any{
		"Id": v.id, "Name": v.name, "ContentType": v.contentType, "Size": int64(len(v.data)), "Status": v.status,
		"CreatedTimestamp": wdocEpoch(v.created), "ModifiedTimestamp": wdocEpoch(v.modified),
		"ContentCreatedTimestamp": wdocEpoch(v.cCreated), "ContentModifiedTimestamp": wdocEpoch(v.cModified), "CreatorId": v.creator,
	}
	if v.signature != "" {
		out["Signature"] = v.signature
	}
	if source && v.status == "ACTIVE" {
		if v.downloadKey == "" {
			v.downloadKey = wdocNewID()
		}
		out["Source"] = map[string]any{"ORIGINAL": base + wdocFilePath + url.PathEscape(r.id) + "/" + url.PathEscape(v.id) + "?key=" + v.downloadKey}
	}
	return out
}

func (st *wdocStore) docView(r *wdocRes) map[string]any {
	out := map[string]any{
		"Id": r.id, "CreatorId": r.creator, "ParentFolderId": r.parent, "CreatedTimestamp": wdocEpoch(r.created),
		"ModifiedTimestamp": wdocEpoch(r.modified), "ResourceState": r.state, "Labels": fcAnyList(r.labels),
	}
	if v := r.latest(); v != nil {
		out["LatestVersionMetadata"] = st.versionView(r, v, false, "")
	}
	return out
}

func wdocMetaMap(m map[string]string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// ---- folders ----

// parentFolder returns an active folder that can take a child named name.
func (st *wdocStore) parentFolder(id, name, except string) (*wdocRes, *wdocErr) {
	p, err := st.resource(id, "FOLDER")
	if err != nil {
		return nil, err
	}
	if p.state != "ACTIVE" {
		return nil, wdocProhibited("folder %s is %s", p.id, p.state)
	}
	if other, taken := p.children[strings.ToLower(name)]; taken && other != except {
		return nil, wdocError(http.StatusConflict, "EntityAlreadyExistsException", "A resource named %s already exists in folder %s.", name, p.id)
	}
	if st.depth(p) >= wdocMaxDepth {
		return nil, wdocLimit("folders can be nested at most %d levels deep", wdocMaxDepth)
	}
	return p, nil
}

func (st *wdocStore) addResource(kind, name string, parent *wdocRes, actor *wdocUser) (*wdocRes, *wdocErr) {
	if len(st.res) >= wdocMaxResources {
		return nil, wdocLimit("Stackyard keeps at most %d WorkDocs folders and documents", wdocMaxResources)
	}
	now := time.Now().UTC()
	r := &wdocRes{id: wdocNewID(), kind: kind, name: name, parent: parent.id, creator: actor.id, state: "ACTIVE", created: now, modified: now, perms: map[string]wdocPerm{}}
	if kind == "FOLDER" {
		r.children = map[string]string{}
	}
	if err := st.charge(r, len(name)); err != nil {
		return nil, err
	}
	st.res[r.id] = r
	parent.children[strings.ToLower(name)] = r.id
	return r, nil
}

func (st *wdocStore) createFolder(q wdocReq) (map[string]any, *wdocErr) {
	name := fcStr(q.in, "Name")
	if strings.TrimSpace(name) == "" {
		return nil, wdocInvalid("Name is required")
	}
	parent, err := st.parentFolder(fcStr(q.in, "ParentFolderId"), name, "")
	if err != nil {
		return nil, err
	}
	f, err := st.addResource("FOLDER", name, parent, q.actor)
	if err != nil {
		return nil, err
	}
	st.record("FOLDER_CREATED", q.actor, f, "", nil)
	return map[string]any{"Metadata": st.folderView(f)}, nil
}

func (st *wdocStore) getFolder(q wdocReq) (map[string]any, *wdocErr) {
	f, err := st.resource(fcStr(q.in, "FolderId"), "FOLDER")
	if err != nil {
		return nil, err
	}
	out := map[string]any{"Metadata": st.folderView(f)}
	if fcBool(q.in, "IncludeCustomMetadata") {
		out["CustomMetadata"] = wdocMetaMap(f.meta)
	}
	return out, nil
}

// move renames, reparents and recycles or restores r per UpdateFolder and
// UpdateDocument, recording the matching activities.
func (st *wdocStore) move(q wdocReq, r *wdocRes) *wdocErr {
	if st.isRoot(r) {
		return wdocProhibited("a root folder can't be changed")
	}
	name, parentID := r.name, r.parent
	if v, ok := q.in["Name"].(string); ok && strings.TrimSpace(v) != "" {
		name = v
	}
	if v, ok := q.in["ParentFolderId"].(string); ok && v != "" {
		parentID = v
	}
	state := fcStr(q.in, "ResourceState")
	if state != "" && state != "ACTIVE" && state != "RECYCLED" {
		return wdocInvalid("ResourceState can only be set to ACTIVE or RECYCLED")
	}
	parent, err := st.parentFolder(parentID, name, r.id)
	if err != nil {
		return err
	}
	for a := parent; a != nil; a = st.res[a.parent] { // depth-bounded by wdocMaxDepth
		if a.id == r.id {
			return wdocError(http.StatusConflict, "ConflictingOperationException", "a folder can't be moved into itself")
		}
		if a.parent == "" {
			break
		}
	}
	if err := st.charge(r, len(name)-len(r.name)); err != nil {
		return err
	}
	kind := r.kind
	old := st.res[r.parent]
	if name != r.name {
		st.record(kind+"_RENAMED", q.actor, r, "", map[string]any{"OriginalParent": map[string]any{"Type": kind, "OriginalName": r.name, "Id": r.id}})
	}
	if old != nil {
		delete(old.children, strings.ToLower(r.name))
	}
	if parent.id != r.parent {
		st.record(kind+"_MOVED", q.actor, r, "", map[string]any{"OriginalParent": map[string]any{"Type": "FOLDER", "Id": r.parent}})
	}
	r.name, r.parent = name, parent.id
	parent.children[strings.ToLower(name)] = r.id
	if state != "" && state != r.state {
		r.state = state
		if state == "RECYCLED" {
			st.record(kind+"_RECYCLED", q.actor, r, "", nil)
		} else {
			st.record(kind+"_RESTORED", q.actor, r, "", nil)
		}
	}
	r.modified = time.Now().UTC()
	return nil
}

func (st *wdocStore) updateFolder(q wdocReq) (map[string]any, *wdocErr) {
	f, err := st.resource(fcStr(q.in, "FolderId"), "FOLDER")
	if err != nil {
		return nil, err
	}
	return nil, st.move(q, f)
}

// removeTree deletes r and everything under it, refunding its storage. Each
// resource is removed once, so the work is paid for by the creates.
func (st *wdocStore) removeTree(r *wdocRes) {
	if r == nil {
		return
	}
	stack := []*wdocRes{r}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, id := range cur.children {
			if c := st.res[id]; c != nil {
				stack = append(stack, c)
			}
		}
		if u := st.users[cur.creator]; u != nil {
			for _, v := range cur.versions {
				if v.status == "ACTIVE" {
					u.used -= int64(len(v.data))
				}
			}
		}
		for p := range cur.perms {
			delete(st.sharedWith[p], cur.id)
		}
		st.bytes -= cur.bytes
		delete(st.res, cur.id)
	}
	if p := st.res[r.parent]; p != nil {
		delete(p.children, strings.ToLower(r.name))
	}
}

func (st *wdocStore) deleteFolder(q wdocReq) (map[string]any, *wdocErr) {
	f, err := st.resource(fcStr(q.in, "FolderId"), "FOLDER")
	if err != nil {
		return nil, err
	}
	if st.isRoot(f) {
		return nil, wdocProhibited("a root folder can't be deleted")
	}
	st.record("FOLDER_DELETED", q.actor, f, "", nil)
	st.removeTree(f)
	return nil, nil
}

func (st *wdocStore) deleteFolderContents(q wdocReq) (map[string]any, *wdocErr) {
	f, err := st.resource(fcStr(q.in, "FolderId"), "FOLDER")
	if err != nil {
		return nil, err
	}
	for _, id := range f.children {
		st.removeTree(st.res[id])
	}
	f.children = map[string]string{}
	f.modified = time.Now().UTC()
	return nil, nil
}

func (st *wdocStore) describeFolderContents(q wdocReq) (map[string]any, *wdocErr) {
	f, err := st.resource(fcStr(q.in, "FolderId"), "FOLDER")
	if err != nil {
		return nil, err
	}
	typ := fcStr(q.in, "Type")
	initialized := strings.Contains(fcStr(q.in, "Include"), "INITIALIZED")
	var list []*wdocRes
	for _, id := range f.children {
		c := st.res[id]
		switch {
		case c == nil || c.state != "ACTIVE",
			typ == "FOLDER" && c.kind != "FOLDER", typ == "DOCUMENT" && c.kind != "DOCUMENT",
			c.kind == "DOCUMENT" && !initialized && (c.latest() == nil || c.latest().status != "ACTIVE"):
			continue
		}
		list = append(list, c)
	}
	byName := fcStr(q.in, "Sort") == "NAME"
	desc := fcStr(q.in, "Order") == "DESCENDING"
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		less := a.created.Before(b.created) || a.created.Equal(b.created) && a.id < b.id
		if byName && !strings.EqualFold(a.name, b.name) {
			less = strings.ToLower(a.name) < strings.ToLower(b.name)
		}
		return less != desc
	})
	start, end, next, perr := wdocPage(len(list), q.in, "Marker", "Limit")
	if perr != nil {
		return nil, perr
	}
	folders, docs := []any{}, []any{}
	for _, c := range list[start:end] {
		if c.kind == "FOLDER" {
			folders = append(folders, st.folderView(c))
		} else {
			docs = append(docs, st.docView(c))
		}
	}
	return wdocWithMarker(map[string]any{"Folders": folders, "Documents": docs}, next), nil
}

func (st *wdocStore) describeRootFolders(q wdocReq) (map[string]any, *wdocErr) {
	root := st.res[q.actor.root]
	items := []any{}
	if root != nil {
		items = append(items, st.folderView(root))
	}
	return map[string]any{"Folders": items}, nil
}

// path lists r's ancestors from the root down to r (at most limit levels).
func (st *wdocStore) path(r *wdocRes, in map[string]any) map[string]any {
	limit := wdocDefaultPage
	if v, ok := fcNum(in, "Limit"); ok && v >= 1 && v <= 999 {
		limit = int(v)
	}
	names := strings.Contains(fcStr(in, "Fields"), "NAME")
	var comps []any
	for cur := r; cur != nil && len(comps) < limit && len(comps) <= wdocMaxDepth; cur = st.res[cur.parent] {
		c := map[string]any{"Id": cur.id}
		if names && cur.name != "" {
			c["Name"] = cur.name
		}
		comps = append([]any{c}, comps...)
		if cur.parent == "" {
			break
		}
	}
	return map[string]any{"Path": map[string]any{"Components": comps}}
}

func (st *wdocStore) getFolderPath(q wdocReq) (map[string]any, *wdocErr) {
	f, err := st.resource(fcStr(q.in, "FolderId"), "FOLDER")
	if err != nil {
		return nil, err
	}
	return st.path(f, q.in), nil
}

func (st *wdocStore) getDocumentPath(q wdocReq) (map[string]any, *wdocErr) {
	d, err := st.resource(fcStr(q.in, "DocumentId"), "DOCUMENT")
	if err != nil {
		return nil, err
	}
	return st.path(d, q.in), nil
}

// ---- documents and versions ----

func (st *wdocStore) initiateUpload(q wdocReq) (map[string]any, *wdocErr) {
	in := q.in
	declared := int64(-1)
	if n, ok := fcNum(in, "DocumentSizeInBytes"); ok {
		if n < 0 || n > wdocMaxUpload {
			return nil, wdocError(http.StatusRequestEntityTooLarge, "RequestedEntityTooLargeException", "Stackyard takes documents of at most %d MiB", wdocMaxUpload>>20)
		}
		declared = int64(n)
	}
	var d *wdocRes
	if id := fcStr(in, "Id"); id != "" {
		var err *wdocErr
		if d, err = st.resource(id, "DOCUMENT"); err != nil {
			return nil, err
		}
		if len(d.versions) >= wdocMaxVersions {
			return nil, wdocLimit("a document can have at most %d versions", wdocMaxVersions)
		}
		for _, v := range d.versions {
			if v.status == "INITIALIZED" {
				return nil, wdocError(http.StatusConflict, "ConflictingOperationException", "document %s already has an upload in progress (version %s)", d.id, v.id)
			}
		}
	} else {
		name := fcStr(in, "Name")
		if strings.TrimSpace(name) == "" {
			return nil, wdocInvalid("Name is required for a new document")
		}
		parent, err := st.parentFolder(fcStr(in, "ParentFolderId"), name, "")
		if err != nil {
			return nil, err
		}
		if d, err = st.addResource("DOCUMENT", name, parent, q.actor); err != nil {
			return nil, err
		}
	}
	now := time.Now().UTC()
	ct := fcStr(in, "ContentType")
	if ct == "" {
		ct = "application/octet-stream"
	}
	v := &wdocVersion{
		id: fmt.Sprintf("%d-%s", now.UnixMilli(), wdocNewID()[:16]), name: d.name, contentType: ct, creator: q.actor.id,
		status: "INITIALIZED", created: now, modified: now, declared: declared, uploadKey: wdocNewID(),
	}
	if v.cCreated, _ = mtkTime(in["ContentCreatedTimestamp"]); v.cCreated.IsZero() {
		v.cCreated = now
	}
	if v.cModified, _ = mtkTime(in["ContentModifiedTimestamp"]); v.cModified.IsZero() {
		v.cModified = now
	}
	if err := st.charge(d, len(ct)); err != nil {
		return nil, err
	}
	d.versions = append(d.versions, v)
	return map[string]any{
		"Metadata": st.docView(d),
		"UploadMetadata": map[string]any{
			"UploadUrl":     q.base + wdocUploadPath + url.PathEscape(d.id) + "/" + url.PathEscape(v.id) + "?key=" + v.uploadKey,
			"SignedHeaders": map[string]any{"Content-Type": ct},
		},
	}, nil
}

func (r *wdocRes) version(id string) (*wdocVersion, int, *wdocErr) {
	for i, v := range r.versions {
		if v.id == id {
			return v, i, nil
		}
	}
	return nil, -1, wdocMissing("document version", id)
}

func (st *wdocStore) docVersion(in map[string]any) (*wdocRes, *wdocVersion, int, *wdocErr) {
	d, err := st.resource(fcStr(in, "DocumentId"), "DOCUMENT")
	if err != nil {
		return nil, nil, -1, err
	}
	v, i, err := d.version(fcStr(in, "VersionId"))
	if err != nil {
		return nil, nil, -1, err
	}
	return d, v, i, nil
}

// upload stores the bytes PUT to a version's pre-signed URL.
func (st *wdocStore) upload(docID, verID, key, contentType string, data []byte) (int, string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d := st.res[docID]
	if d == nil {
		return http.StatusForbidden, "the upload URL is not valid"
	}
	v, _, err := d.version(verID)
	if err != nil || !wdocKeyOK(key, v.uploadKey) {
		return http.StatusForbidden, "the upload URL is not valid"
	}
	switch {
	case v.status != "INITIALIZED":
		return http.StatusConflict, "this version has already been activated"
	case contentType != v.contentType:
		return http.StatusForbidden, "Content-Type must match the signed header " + v.contentType
	case v.declared >= 0 && int64(len(data)) != v.declared:
		return http.StatusBadRequest, fmt.Sprintf("the upload must be the %d bytes declared in DocumentSizeInBytes", v.declared)
	}
	if err := st.charge(d, len(data)-len(v.data)); err != nil {
		return http.StatusRequestEntityTooLarge, err.msg
	}
	v.data = append([]byte(nil), data...)
	v.uploaded = true
	return http.StatusOK, ""
}

// download returns an active version's bytes for its download key.
func (st *wdocStore) download(docID, verID, key string) ([]byte, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d := st.res[docID]
	if d == nil {
		return nil, false
	}
	v, _, err := d.version(verID)
	if err != nil || v.deleted || v.status != "ACTIVE" || !wdocKeyOK(key, v.downloadKey) {
		return nil, false
	}
	if u := st.users[d.creator]; u != nil {
		st.record("DOCUMENT_VERSION_DOWNLOADED", u, d, v.id, nil)
	}
	return v.data, true
}

func (st *wdocStore) updateVersion(q wdocReq) (map[string]any, *wdocErr) {
	d, v, _, err := st.docVersion(q.in)
	if err != nil {
		return nil, err
	}
	if fcStr(q.in, "VersionStatus") != "ACTIVE" {
		return nil, wdocInvalid("VersionStatus must be ACTIVE")
	}
	if v.status == "ACTIVE" {
		return nil, nil
	}
	if !v.uploaded {
		return nil, wdocError(http.StatusConflict, "DraftUploadOutOfSyncException", "Nothing has been uploaded to version %s yet.", v.id)
	}
	if u := st.users[d.creator]; u != nil {
		if u.storageType == "QUOTA" && u.used+int64(len(v.data)) > u.allocated {
			return nil, wdocError(http.StatusConflict, "StorageLimitExceededException", "user %s has no storage left", u.username)
		}
		u.used += int64(len(v.data))
	}
	sum := md5.Sum(v.data)
	v.status, v.signature, v.uploadKey = "ACTIVE", hex.EncodeToString(sum[:]), ""
	v.modified = time.Now().UTC()
	d.modified = v.modified
	st.record("DOCUMENT_VERSION_UPLOADED", q.actor, d, v.id, nil)
	return nil, nil
}

func (st *wdocStore) abortUpload(q wdocReq) (map[string]any, *wdocErr) {
	d, v, i, err := st.docVersion(q.in)
	if err != nil {
		return nil, err
	}
	if v.status != "INITIALIZED" {
		return nil, wdocProhibited("version %s has already been uploaded", v.id)
	}
	_ = st.charge(d, -len(v.data)-len(v.contentType))
	d.versions = append(d.versions[:i:i], d.versions[i+1:]...)
	if len(d.versions) == 0 {
		st.removeTree(d)
	}
	return nil, nil
}

func (st *wdocStore) getVersion(q wdocReq) (map[string]any, *wdocErr) {
	d, v, _, err := st.docVersion(q.in)
	if err != nil {
		return nil, err
	}
	if v.deleted {
		return nil, wdocMissing("document version", v.id)
	}
	out := map[string]any{"Metadata": st.versionView(d, v, strings.Contains(fcStr(q.in, "Fields"), "SOURCE"), q.base)}
	if fcBool(q.in, "IncludeCustomMetadata") {
		out["CustomMetadata"] = wdocMetaMap(v.meta)
	}
	return out, nil
}

func (st *wdocStore) describeVersions(q wdocReq) (map[string]any, *wdocErr) {
	d, err := st.resource(fcStr(q.in, "DocumentId"), "DOCUMENT")
	if err != nil {
		return nil, err
	}
	initialized := strings.Contains(fcStr(q.in, "Include"), "INITIALIZED")
	source := strings.Contains(fcStr(q.in, "Fields"), "SOURCE")
	var list []*wdocVersion
	for i := len(d.versions) - 1; i >= 0; i-- { // newest first
		v := d.versions[i]
		if !v.deleted && (v.status == "ACTIVE" || initialized) {
			list = append(list, v)
		}
	}
	start, end, next, perr := wdocPage(len(list), q.in, "Marker", "Limit")
	if perr != nil {
		return nil, perr
	}
	items := make([]any, 0, end-start)
	for _, v := range list[start:end] {
		items = append(items, st.versionView(d, v, source, q.base))
	}
	return wdocWithMarker(map[string]any{"DocumentVersions": items}, next), nil
}

func (st *wdocStore) deleteVersion(q wdocReq) (map[string]any, *wdocErr) {
	d, v, i, err := st.docVersion(q.in)
	if err != nil {
		return nil, err
	}
	if v.status != "ACTIVE" || v.deleted {
		return nil, wdocProhibited("version %s is not an active version", v.id)
	}
	doomed := map[int]bool{i: true}
	if fcBool(q.in, "DeletePriorVersions") {
		for j := 0; j < i; j++ {
			doomed[j] = true
		}
	}
	left := 0
	for j, other := range d.versions {
		if !doomed[j] && !other.deleted && other.status == "ACTIVE" {
			left++
		}
	}
	if left == 0 {
		return nil, wdocProhibited("a document must keep at least one version; delete the document instead")
	}
	for j := range doomed {
		if !d.versions[j].deleted && d.versions[j].status == "ACTIVE" {
			d.versions[j].deleted = true
			st.record("DOCUMENT_VERSION_DELETED", q.actor, d, d.versions[j].id, nil)
		}
	}
	d.modified = time.Now().UTC()
	return nil, nil
}

// restoreVersions brings back every deleted version of the document.
func (st *wdocStore) restoreVersions(q wdocReq) (map[string]any, *wdocErr) {
	d, err := st.resource(fcStr(q.in, "DocumentId"), "DOCUMENT")
	if err != nil {
		return nil, err
	}
	restored := false
	for _, v := range d.versions {
		if v.deleted {
			v.deleted, restored = false, true
		}
	}
	if !restored {
		return nil, wdocProhibited("document %s has no deleted versions to restore", d.id)
	}
	d.modified = time.Now().UTC()
	st.record("DOCUMENT_REVERTED", q.actor, d, "", nil)
	return nil, nil
}

func (st *wdocStore) getDocument(q wdocReq) (map[string]any, *wdocErr) {
	d, err := st.resource(fcStr(q.in, "DocumentId"), "DOCUMENT")
	if err != nil {
		return nil, err
	}
	out := map[string]any{"Metadata": st.docView(d)}
	if fcBool(q.in, "IncludeCustomMetadata") {
		out["CustomMetadata"] = wdocMetaMap(d.meta)
	}
	return out, nil
}

func (st *wdocStore) updateDocument(q wdocReq) (map[string]any, *wdocErr) {
	d, err := st.resource(fcStr(q.in, "DocumentId"), "DOCUMENT")
	if err != nil {
		return nil, err
	}
	return nil, st.move(q, d)
}

func (st *wdocStore) deleteDocument(q wdocReq) (map[string]any, *wdocErr) {
	d, err := st.resource(fcStr(q.in, "DocumentId"), "DOCUMENT")
	if err != nil {
		return nil, err
	}
	st.removeTree(d)
	return nil, nil
}

// ---- comments ----

func (st *wdocStore) commentView(c *wdocComment) map[string]any {
	out := map[string]any{
		"CommentId": c.id, "ThreadId": c.thread, "Text": c.text, "CreatedTimestamp": wdocEpoch(c.created),
		"Status": "PUBLISHED", "Visibility": c.visibility,
	}
	if c.parent != "" {
		out["ParentId"] = c.parent
	}
	if c.recipient != "" {
		out["RecipientId"] = c.recipient
	}
	if u := st.users[c.contributor]; u != nil {
		out["Contributor"] = u.view()
	}
	return out
}

func (st *wdocStore) createComment(q wdocReq) (map[string]any, *wdocErr) {
	d, v, _, err := st.docVersion(q.in)
	if err != nil {
		return nil, err
	}
	if v.status != "ACTIVE" || v.deleted {
		return nil, wdocError(http.StatusConflict, "InvalidCommentOperationException", "comments can only be added to active versions")
	}
	if len(v.comments) >= wdocMaxComments {
		return nil, wdocLimit("a version can have at most %d comments", wdocMaxComments)
	}
	c := &wdocComment{id: wdocNewID(), text: fcStr(q.in, "Text"), contributor: q.actor.id, visibility: fcStr(q.in, "Visibility"), created: time.Now().UTC()}
	if c.visibility == "" {
		c.visibility = "PUBLIC"
	}
	c.thread = c.id
	if pid := fcStr(q.in, "ParentId"); pid != "" {
		var parent *wdocComment
		for _, other := range v.comments {
			if other.id == pid {
				parent = other
			}
		}
		if parent == nil {
			return nil, wdocMissing("comment", pid)
		}
		c.parent, c.thread = parent.id, parent.thread
		if c.visibility == "PRIVATE" {
			c.recipient = parent.contributor
		}
	} else if tid := fcStr(q.in, "ThreadId"); tid != "" {
		return nil, wdocInvalid("ThreadId needs a ParentId in that thread")
	}
	if err := st.charge(d, len(c.text)); err != nil {
		return nil, err
	}
	v.comments = append(v.comments, c)
	st.record("DOCUMENT_COMMENT_ADDED", q.actor, d, v.id, map[string]any{"CommentMetadata": map[string]any{
		"CommentId": c.id, "CreatedTimestamp": wdocEpoch(c.created), "CommentStatus": "PUBLISHED", "ContributorId": c.contributor,
	}})
	return map[string]any{"Comment": st.commentView(c)}, nil
}

func (st *wdocStore) deleteComment(q wdocReq) (map[string]any, *wdocErr) {
	d, v, _, err := st.docVersion(q.in)
	if err != nil {
		return nil, err
	}
	id := fcStr(q.in, "CommentId")
	for i, c := range v.comments {
		if c.id == id {
			_ = st.charge(d, -len(c.text))
			v.comments = append(v.comments[:i:i], v.comments[i+1:]...)
			st.record("DOCUMENT_COMMENT_DELETED", q.actor, d, v.id, nil)
			return nil, nil
		}
	}
	return nil, wdocMissing("comment", id)
}

func (st *wdocStore) describeComments(q wdocReq) (map[string]any, *wdocErr) {
	_, v, _, err := st.docVersion(q.in)
	if err != nil {
		return nil, err
	}
	start, end, next, perr := wdocPage(len(v.comments), q.in, "Marker", "Limit")
	if perr != nil {
		return nil, perr
	}
	items := make([]any, 0, end-start)
	for _, c := range v.comments[start:end] {
		items = append(items, st.commentView(c))
	}
	return wdocWithMarker(map[string]any{"Comments": items}, next), nil
}
