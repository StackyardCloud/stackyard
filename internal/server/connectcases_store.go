package server

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Connect Cases state: domains with their fields (system fields are seeded),
// field options, layouts, templates, case rules, cases and tags. One mutex;
// contact lookups go to the chat store before this lock is taken.

const (
	cncMaxDomains        = 20
	cncMaxFields         = 300
	cncMaxOptions        = 500
	cncMaxLayouts        = 100
	cncMaxTemplates      = 100
	cncMaxRules          = 100
	cncMaxCasesPerDomain = 2000
	cncMaxCases          = 10000
	cncMaxCaseBytes      = 512 << 10
	cncMaxStoreBytes     = 16 << 20
	cncMaxRelated        = 200
	cncMaxAudit          = 100
	cncMaxMerges         = 200
	cncMaxTags           = 50
	cncMaxConfigBytes    = 16 << 20
)

type cncStore struct {
	mu      sync.Mutex
	chats   *cnpStore
	domains map[string]*cncDomain
	tags    map[string]map[string]string // resource ARN -> tags; a key exists per live resource
	seq     int64
	cases   int // store-wide
	bytes   int // store-wide case bytes
	config  int // store-wide bytes of fields, options, layouts, templates and rules
}

// cncRes is a field, layout, template or case rule.
type cncRes struct {
	id, arn           string
	seq               int64
	attrs             map[string]any
	options           []map[string]any // fields only: {name, value, active}
	created, modified time.Time
	size              int
}

func cncResSize(attrs map[string]any, options []map[string]any) int {
	a, _ := json.Marshal(attrs)
	o, _ := json.Marshal(options)
	return len(a) + len(o) + 256
}

// resizeLocked accounts a resource's new size against the configuration budget.
func (st *cncStore) resizeLocked(r *cncRes, size int) *cncErr {
	if st.config-r.size+size > cncMaxConfigBytes {
		return cncQuota("domain configuration storage is full")
	}
	st.config += size - r.size
	r.size = size
	return nil
}

func (st *cncStore) dropResLocked(m map[string]*cncRes, r *cncRes) {
	st.config -= r.size
	delete(m, r.id)
	delete(st.tags, r.arn)
}

type cncDomain struct {
	id, arn, name string
	seq           int64
	created       time.Time
	fields        map[string]*cncRes
	layouts       map[string]*cncRes
	templates     map[string]*cncRes
	rules         map[string]*cncRes
	cases         map[string]*cncCase
	refs          map[string]string // reference number -> case id
	clientTokens  map[string]string // CreateCase client token -> case id
	nextRef       int
	merges        []*cncMerge
	eventConfig   map[string]any
}

type cncMerge struct{ primary, secondary string }

func newConnectCasesStore(chats *cnpStore) *cncStore {
	return &cncStore{chats: chats, domains: map[string]*cncDomain{}, tags: map[string]map[string]string{}}
}

func (st *cncStore) nextSeq() int64 { st.seq++; return st.seq }

// cncSystemFields are created with every domain; their ids are their names.
var cncSystemFields = []struct{ id, typ string }{
	{"title", "Text"}, {"status", "SingleSelect"}, {"customer_id", "Text"}, {"reference_number", "Text"},
	{"summary", "Text"}, {"created_datetime", "DateTime"}, {"last_updated_datetime", "DateTime"},
	{"last_closed_datetime", "DateTime"},
}

// cncReadOnlyFields are maintained by the service.
var cncReadOnlyFields = map[string]bool{"reference_number": true, "created_datetime": true, "last_updated_datetime": true, "last_closed_datetime": true}

func (st *cncStore) domainLocked(id string) (*cncDomain, *cncErr) {
	d := st.domains[id]
	if d == nil {
		return nil, cncNotFound("domain %s not found", id)
	}
	return d, nil
}

func (st *cncStore) setTagsLocked(arn string, tags map[string]any) {
	t := map[string]string{}
	for k, v := range tags {
		if s, ok := v.(string); ok {
			t[k] = s
		}
	}
	st.tags[arn] = t
}

func (st *cncStore) tagsOut(arn string) map[string]any {
	out := map[string]any{}
	for k, v := range st.tags[arn] {
		out[k] = v
	}
	return out
}

// ---- domains ----

func (st *cncStore) createDomain(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	name := cncStr(in, "name")
	for _, d := range st.domains {
		if d.name == name {
			return nil, cncConflict("a domain named %s already exists", name)
		}
	}
	if len(st.domains) >= cncMaxDomains {
		return nil, cncQuota("at most %d domains", cncMaxDomains)
	}
	now := time.Now().UTC()
	id := uuid.NewString()
	d := &cncDomain{id: id, arn: fmt.Sprintf("arn:aws:cases:%s:%s:domain/%s", cncRegion, cncAccount, id), name: name, seq: st.nextSeq(), created: now,
		fields: map[string]*cncRes{}, layouts: map[string]*cncRes{}, templates: map[string]*cncRes{}, rules: map[string]*cncRes{},
		cases: map[string]*cncCase{}, refs: map[string]string{}, clientTokens: map[string]string{}, nextRef: 10000000}
	for _, f := range cncSystemFields {
		res := &cncRes{id: f.id, arn: d.arn + "/field/" + f.id, seq: st.nextSeq(), created: now, modified: now,
			attrs: map[string]any{"name": f.id, "type": f.typ, "namespace": "System"}}
		if f.id == "status" {
			res.options = []map[string]any{{"name": "Open", "value": "open", "active": true}, {"name": "Closed", "value": "closed", "active": true}}
		}
		res.size = cncResSize(res.attrs, res.options)
		st.config += res.size
		d.fields[f.id] = res
		st.tags[res.arn] = map[string]string{}
	}
	st.domains[id] = d
	st.tags[d.arn] = map[string]string{}
	return map[string]any{"domainId": id, "domainArn": d.arn, "domainStatus": "Active"}, nil
}

func (st *cncStore) getDomain(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"domainId": d.id, "domainArn": d.arn, "name": d.name, "createdTime": cncTime(d.created),
		"domainStatus": "Active", "tags": st.tagsOut(d.arn)}, nil
}

func (st *cncStore) listDomains(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	ds := make([]*cncDomain, 0, len(st.domains))
	for _, d := range st.domains {
		ds = append(ds, d)
	}
	sort.Slice(ds, func(i, j int) bool { return ds[i].seq < ds[j].seq })
	start, end, next, err := cncPage(in, len(ds), 10, 10)
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, d := range ds[start:end] {
		items = append(items, map[string]any{"domainId": d.id, "domainArn": d.arn, "name": d.name})
	}
	return cncWithNext(map[string]any{"domains": items}, next), nil
}

func cncWithNext(out map[string]any, next string) map[string]any {
	if next != "" {
		out["nextToken"] = next
	}
	return out
}

func (st *cncStore) deleteDomain(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	for _, c := range d.cases {
		st.dropCaseLocked(d, c)
	}
	for _, m := range []map[string]*cncRes{d.fields, d.layouts, d.templates, d.rules} {
		for _, r := range m {
			st.config -= r.size
		}
	}
	prefix := d.arn + "/"
	for arn := range st.tags {
		if strings.HasPrefix(arn, prefix) {
			delete(st.tags, arn)
		}
	}
	delete(st.tags, d.arn)
	delete(st.domains, d.id)
	return nil, nil
}

// ---- tags ----

func (st *cncStore) tagResource(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	arn := cncStr(in, "arn")
	tags, ok := st.tags[arn]
	if !ok {
		return nil, cncNotFound("resource %s not found", arn)
	}
	add := cncMap(in, "tags")
	merged := make(map[string]string, len(tags)+len(add))
	for k, v := range tags {
		merged[k] = v
	}
	for k, v := range add {
		if strings.HasPrefix(k, "aws:") || len(k) > 128 {
			return nil, cncValidation("tag key %q is not allowed", k)
		}
		s, _ := v.(string)
		if len(s) > 256 {
			return nil, cncValidation("tag values are at most 256 characters")
		}
		merged[k] = s
	}
	if len(merged) > cncMaxTags {
		return nil, cncValidation("a resource holds at most %d tags", cncMaxTags)
	}
	st.tags[arn] = merged
	return nil, nil
}

func (st *cncStore) untagResource(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	arn := cncStr(in, "arn")
	tags, ok := st.tags[arn]
	if !ok {
		return nil, cncNotFound("resource %s not found", arn)
	}
	keys := cncList(in, "tagKeys")
	if len(keys) == 0 {
		return nil, cncValidation("tagKeys is required")
	}
	kept := make(map[string]string, len(tags))
	for k, v := range tags {
		kept[k] = v
	}
	for _, k := range keys {
		delete(kept, k.(string))
	}
	st.tags[arn] = kept
	return nil, nil
}

func (st *cncStore) listTags(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	arn := cncStr(in, "arn")
	if _, ok := st.tags[arn]; !ok {
		return nil, cncNotFound("resource %s not found", arn)
	}
	return map[string]any{"tags": st.tagsOut(arn)}, nil
}

// ---- shared resource helpers ----

func cncSorted(m map[string]*cncRes) []*cncRes {
	out := make([]*cncRes, 0, len(m))
	for _, r := range m {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].seq < out[j].seq })
	return out
}

func cncNameTaken(m map[string]*cncRes, name, except string) bool {
	for id, r := range m {
		if id != except && r.attrs["name"] == name {
			return true
		}
	}
	return false
}

// newResLocked stores a resource of kind (field, layout, template, case-rule).
func (st *cncStore) newResLocked(d *cncDomain, m map[string]*cncRes, kind string, attrs map[string]any) (*cncRes, *cncErr) {
	now := time.Now().UTC()
	id := uuid.NewString()
	r := &cncRes{id: id, arn: d.arn + "/" + kind + "/" + id, seq: st.nextSeq(), attrs: attrs, created: now, modified: now}
	if err := st.resizeLocked(r, cncResSize(attrs, nil)); err != nil {
		return nil, err
	}
	m[id] = r
	st.tags[r.arn] = map[string]string{}
	return r, nil
}

// updateResLocked replaces the listed attributes present in in (copy-on-write).
func (st *cncStore) updateResLocked(r *cncRes, in map[string]any, keys ...string) *cncErr {
	attrs := cncCopyMap(r.attrs)
	for _, k := range keys {
		if v, ok := in[k]; ok {
			attrs[k] = cncCopy(v)
		}
	}
	if err := st.resizeLocked(r, cncResSize(attrs, r.options)); err != nil {
		return err
	}
	r.attrs = attrs
	r.modified = time.Now().UTC()
	return nil
}

func (r *cncRes) out(st *cncStore, idKey, arnKey string, keys ...string) map[string]any {
	out := map[string]any{idKey: r.id, arnKey: r.arn, "createdTime": cncTime(r.created), "lastModifiedTime": cncTime(r.modified),
		"deleted": false, "tags": st.tagsOut(r.arn)}
	for _, k := range keys {
		if v, ok := r.attrs[k]; ok {
			out[k] = cncCopy(v)
		}
	}
	return out
}

func (st *cncStore) resLocked(in map[string]any, pick func(*cncDomain) map[string]*cncRes, idKey, kind string) (*cncDomain, *cncRes, *cncErr) {
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, nil, err
	}
	r := pick(d)[cncStr(in, idKey)]
	if r == nil {
		return nil, nil, cncNotFound("%s %s not found", kind, cncStr(in, idKey))
	}
	return d, r, nil
}

func cncFields(d *cncDomain) map[string]*cncRes    { return d.fields }
func cncLayouts(d *cncDomain) map[string]*cncRes   { return d.layouts }
func cncTemplates(d *cncDomain) map[string]*cncRes { return d.templates }
func cncRules(d *cncDomain) map[string]*cncRes     { return d.rules }

// ---- fields ----

func cncFieldType(f *cncRes) string { s, _ := f.attrs["type"].(string); return s }

func (st *cncStore) createField(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	if cncNameTaken(d.fields, cncStr(in, "name"), "") {
		return nil, cncConflict("a field named %s already exists", cncStr(in, "name"))
	}
	if len(d.fields) >= cncMaxFields {
		return nil, cncQuota("a domain holds at most %d fields", cncMaxFields)
	}
	if in["attributes"] != nil && in["type"] != "Text" {
		return nil, cncValidation("attributes apply to Text fields only")
	}
	attrs := map[string]any{"name": in["name"], "type": in["type"], "namespace": "Custom"}
	for _, k := range []string{"description", "attributes"} {
		if v, ok := in[k]; ok {
			attrs[k] = cncCopy(v)
		}
	}
	f, err := st.newResLocked(d, d.fields, "field", attrs)
	if err != nil {
		return nil, err
	}
	return map[string]any{"fieldId": f.id, "fieldArn": f.arn}, nil
}

func (st *cncStore) updateField(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, f, err := st.resLocked(in, cncFields, "fieldId", "field")
	if err != nil {
		return nil, err
	}
	if f.attrs["namespace"] == "System" {
		return nil, cncValidation("system field %s can't be updated", f.id)
	}
	if name, ok := in["name"].(string); ok && cncNameTaken(d.fields, name, f.id) {
		return nil, cncConflict("a field named %s already exists", name)
	}
	if in["attributes"] != nil && cncFieldType(f) != "Text" {
		return nil, cncValidation("attributes apply to Text fields only")
	}
	return nil, st.updateResLocked(f, in, "name", "description", "attributes")
}

// fieldInUse names what still references a field.
func (d *cncDomain) fieldInUse(id string) string {
	for _, t := range d.templates {
		for _, rf := range cncListOf(t.attrs["requiredFields"]) {
			if m, _ := rf.(map[string]any); m["fieldId"] == id {
				return "template " + t.id
			}
		}
	}
	for _, l := range d.layouts {
		for _, fid := range cncLayoutFieldIDs(cncMap(l.attrs, "content")) {
			if fid == id {
				return "layout " + l.id
			}
		}
	}
	return ""
}

func cncListOf(v any) []any { l, _ := v.([]any); return l }

func (st *cncStore) deleteField(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, f, err := st.resLocked(in, cncFields, "fieldId", "field")
	if err != nil {
		return nil, err
	}
	if f.attrs["namespace"] == "System" {
		return nil, cncValidation("system field %s can't be deleted", f.id)
	}
	if user := d.fieldInUse(f.id); user != "" {
		return nil, cncConflict("field %s is used by %s", f.id, user)
	}
	st.dropResLocked(d.fields, f)
	return nil, nil
}

func cncFieldOut(st *cncStore, f *cncRes) map[string]any {
	return f.out(st, "fieldId", "fieldArn", "name", "type", "namespace", "description", "attributes")
}

func (st *cncStore) batchGetField(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	fields, errs := []any{}, []any{}
	for _, raw := range cncList(in, "fields") {
		id := cncStr(raw.(map[string]any), "id")
		if f := d.fields[id]; f != nil {
			fields = append(fields, cncFieldOut(st, f))
		} else {
			errs = append(errs, map[string]any{"id": id, "errorCode": "ResourceNotFoundException", "message": "field not found"})
		}
	}
	return map[string]any{"fields": fields, "errors": errs}, nil
}

func (st *cncStore) listFields(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	all := cncSorted(d.fields)
	start, end, next, err := cncPage(in, len(all), 100, 100)
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, f := range all[start:end] {
		o := cncFieldOut(st, f)
		items = append(items, map[string]any{"fieldId": o["fieldId"], "fieldArn": o["fieldArn"], "name": o["name"],
			"type": o["type"], "namespace": o["namespace"], "attributes": o["attributes"]})
	}
	return cncWithNext(map[string]any{"fields": items}, next), nil
}

func (st *cncStore) batchPutFieldOptions(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	_, f, err := st.resLocked(in, cncFields, "fieldId", "field")
	if err != nil {
		return nil, err
	}
	if cncFieldType(f) != "SingleSelect" {
		return nil, cncValidation("options apply to SingleSelect fields only")
	}
	opts := make([]map[string]any, len(f.options))
	copy(opts, f.options)
	errs := []any{}
	for _, raw := range cncList(in, "options") {
		o := cncCopyMap(raw.(map[string]any))
		replaced := false
		for i, prev := range opts {
			if prev["value"] == o["value"] {
				opts[i], replaced = o, true
				break
			}
		}
		if replaced {
			continue
		}
		if len(opts) >= cncMaxOptions {
			errs = append(errs, map[string]any{"value": o["value"], "errorCode": "ServiceQuotaExceededException", "message": fmt.Sprintf("a field holds at most %d options", cncMaxOptions)})
			continue
		}
		opts = append(opts, o)
	}
	if err := st.resizeLocked(f, cncResSize(f.attrs, opts)); err != nil {
		return nil, err
	}
	f.options = opts
	f.modified = time.Now().UTC()
	return map[string]any{"errors": errs}, nil
}

func (st *cncStore) listFieldOptions(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	_, f, err := st.resLocked(in, cncFields, "fieldId", "field")
	if err != nil {
		return nil, err
	}
	opts := f.options
	if values := cncList(in, "values"); len(values) > 0 {
		opts = nil
		for _, o := range f.options {
			for _, v := range values {
				if o["value"] == v {
					opts = append(opts, o)
				}
			}
		}
	}
	start, end, next, err := cncPage(in, len(opts), 100, 100)
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, o := range opts[start:end] {
		items = append(items, cncCopy(o))
	}
	return cncWithNext(map[string]any{"options": items}, next), nil
}

// ---- layouts ----

// cncLayoutFieldIDs lists the field ids a layout's sections reference.
func cncLayoutFieldIDs(content map[string]any) []string {
	ids := []string{}
	basic := cncMap(content, "basic")
	for _, panel := range []string{"topPanel", "moreInfo"} {
		for _, sec := range cncList(cncMap(basic, panel), "sections") {
			for _, item := range cncList(cncMap(sec.(map[string]any), "fieldGroup"), "fields") {
				ids = append(ids, cncStr(item.(map[string]any), "id"))
			}
		}
	}
	return ids
}

// cncLayoutFields checks a layout's sections reference existing fields.
func cncLayoutFields(d *cncDomain, content map[string]any) *cncErr {
	if cncMap(content, "basic") == nil {
		return cncValidation("content must set basic")
	}
	for _, id := range cncLayoutFieldIDs(content) {
		if d.fields[id] == nil {
			return cncValidation("layout field %s doesn't exist", id)
		}
	}
	return nil
}

func (st *cncStore) createLayout(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	if cncNameTaken(d.layouts, cncStr(in, "name"), "") {
		return nil, cncConflict("a layout named %s already exists", cncStr(in, "name"))
	}
	if len(d.layouts) >= cncMaxLayouts {
		return nil, cncQuota("a domain holds at most %d layouts", cncMaxLayouts)
	}
	if err := cncLayoutFields(d, cncMap(in, "content")); err != nil {
		return nil, err
	}
	l, err := st.newResLocked(d, d.layouts, "layout", map[string]any{"name": in["name"], "content": cncCopy(in["content"])})
	if err != nil {
		return nil, err
	}
	return map[string]any{"layoutId": l.id, "layoutArn": l.arn}, nil
}

func (st *cncStore) getLayout(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	_, l, err := st.resLocked(in, cncLayouts, "layoutId", "layout")
	if err != nil {
		return nil, err
	}
	return l.out(st, "layoutId", "layoutArn", "name", "content"), nil
}

func (st *cncStore) updateLayout(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, l, err := st.resLocked(in, cncLayouts, "layoutId", "layout")
	if err != nil {
		return nil, err
	}
	if name, ok := in["name"].(string); ok && cncNameTaken(d.layouts, name, l.id) {
		return nil, cncConflict("a layout named %s already exists", name)
	}
	if c := cncMap(in, "content"); c != nil {
		if err := cncLayoutFields(d, c); err != nil {
			return nil, err
		}
	}
	return nil, st.updateResLocked(l, in, "name", "content")
}

func (st *cncStore) deleteLayout(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, l, err := st.resLocked(in, cncLayouts, "layoutId", "layout")
	if err != nil {
		return nil, err
	}
	for _, t := range d.templates {
		if cncMap(t.attrs, "layoutConfiguration")["defaultLayout"] == l.id {
			return nil, cncConflict("layout %s is the default layout of template %s", l.id, t.id)
		}
	}
	st.dropResLocked(d.layouts, l)
	return nil, nil
}

func (st *cncStore) listLayouts(in map[string]any) (map[string]any, *cncErr) {
	return st.listRes(in, cncLayouts, "layouts", func(r *cncRes) map[string]any {
		return map[string]any{"layoutId": r.id, "layoutArn": r.arn, "name": r.attrs["name"]}
	})
}

func (st *cncStore) listRes(in map[string]any, pick func(*cncDomain) map[string]*cncRes, key string, shape func(*cncRes) map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	all := cncSorted(pick(d))
	if statuses := cncList(in, "status"); key == "templates" && len(statuses) > 0 {
		kept := all[:0:0]
		for _, r := range all {
			for _, s := range statuses {
				if r.attrs["status"] == s {
					kept = append(kept, r)
				}
			}
		}
		all = kept
	}
	start, end, next, err := cncPage(in, len(all), 100, 100)
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, r := range all[start:end] {
		items = append(items, shape(r))
	}
	return cncWithNext(map[string]any{key: items}, next), nil
}

// ---- templates ----

func (st *cncStore) checkTemplate(d *cncDomain, in map[string]any) *cncErr {
	if l := cncStr(cncMap(in, "layoutConfiguration"), "defaultLayout"); l != "" && d.layouts[l] == nil {
		return cncValidation("layout %s doesn't exist", l)
	}
	for _, raw := range cncList(in, "requiredFields") {
		if id := cncStr(raw.(map[string]any), "fieldId"); d.fields[id] == nil {
			return cncValidation("required field %s doesn't exist", id)
		}
	}
	for _, raw := range cncList(in, "rules") {
		m := raw.(map[string]any)
		if id := cncStr(m, "caseRuleId"); d.rules[id] == nil {
			return cncValidation("case rule %s doesn't exist", id)
		}
		if id := cncStr(m, "fieldId"); id != "" && d.fields[id] == nil {
			return cncValidation("rule field %s doesn't exist", id)
		}
	}
	return nil
}

var cncTemplateAttrs = []string{"name", "description", "layoutConfiguration", "requiredFields", "status", "rules", "tagPropagationConfigurations"}

func (st *cncStore) createTemplate(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	if cncNameTaken(d.templates, cncStr(in, "name"), "") {
		return nil, cncConflict("a template named %s already exists", cncStr(in, "name"))
	}
	if len(d.templates) >= cncMaxTemplates {
		return nil, cncQuota("a domain holds at most %d templates", cncMaxTemplates)
	}
	if err := st.checkTemplate(d, in); err != nil {
		return nil, err
	}
	attrs := map[string]any{"status": "Active"}
	for _, k := range cncTemplateAttrs {
		if v, ok := in[k]; ok {
			attrs[k] = cncCopy(v)
		}
	}
	t, err := st.newResLocked(d, d.templates, "template", attrs)
	if err != nil {
		return nil, err
	}
	return map[string]any{"templateId": t.id, "templateArn": t.arn}, nil
}

func (st *cncStore) getTemplate(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	_, t, err := st.resLocked(in, cncTemplates, "templateId", "template")
	if err != nil {
		return nil, err
	}
	return t.out(st, "templateId", "templateArn", cncTemplateAttrs...), nil
}

func (st *cncStore) updateTemplate(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, t, err := st.resLocked(in, cncTemplates, "templateId", "template")
	if err != nil {
		return nil, err
	}
	if name, ok := in["name"].(string); ok && cncNameTaken(d.templates, name, t.id) {
		return nil, cncConflict("a template named %s already exists", name)
	}
	if err := st.checkTemplate(d, in); err != nil {
		return nil, err
	}
	return nil, st.updateResLocked(t, in, cncTemplateAttrs...)
}

func (st *cncStore) deleteTemplate(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, t, err := st.resLocked(in, cncTemplates, "templateId", "template")
	if err != nil {
		return nil, err
	}
	st.dropResLocked(d.templates, t)
	return nil, nil
}

func (st *cncStore) listTemplates(in map[string]any) (map[string]any, *cncErr) {
	return st.listRes(in, cncTemplates, "templates", func(r *cncRes) map[string]any {
		return map[string]any{"templateId": r.id, "templateArn": r.arn, "name": r.attrs["name"], "status": r.attrs["status"],
			"tagPropagationConfigurations": cncCopy(r.attrs["tagPropagationConfigurations"])}
	})
}

// ---- case rules ----

func cncRuleType(r *cncRes) string {
	for k := range cncMap(r.attrs, "rule") {
		return map[string]string{"required": "Required", "hidden": "Hidden", "fieldOptions": "FieldOptions"}[k]
	}
	return ""
}

func (st *cncStore) checkRule(d *cncDomain, rule map[string]any) *cncErr {
	kind, body, err := cncUnion(rule, "rule")
	if err != nil {
		return err
	}
	m := body.(map[string]any)
	if kind == "fieldOptions" {
		for _, k := range []string{"parentFieldId", "childFieldId"} {
			id := cncStr(m, k)
			if f := d.fields[id]; id != "" && (f == nil || cncFieldType(f) != "SingleSelect") {
				return cncValidation("%s %s must be a SingleSelect field", k, id)
			}
		}
		return nil
	}
	for _, c := range cncList(m, "conditions") {
		if err := cncCheckCondition(d, c.(map[string]any), 0); err != nil {
			return err
		}
	}
	return nil
}

func cncCheckCondition(d *cncDomain, c map[string]any, depth int) *cncErr {
	if depth > 5 {
		return cncValidation("conditions nest at most 5 deep")
	}
	kind, body, err := cncUnion(c, "condition")
	if err != nil {
		return err
	}
	m := body.(map[string]any)
	if kind == "andAll" || kind == "orAll" {
		for _, sub := range cncList(m, "conditions") {
			if err := cncCheckCondition(d, sub.(map[string]any), depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if id := cncStr(cncMap(m, "operandOne"), "fieldId"); d.fields[id] == nil {
		return cncValidation("condition field %s doesn't exist", id)
	}
	return nil
}

func (st *cncStore) createCaseRule(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	if len(d.rules) >= cncMaxRules {
		return nil, cncQuota("a domain holds at most %d case rules", cncMaxRules)
	}
	if err := st.checkRule(d, cncMap(in, "rule")); err != nil {
		return nil, err
	}
	attrs := map[string]any{"name": in["name"], "rule": cncCopy(in["rule"])}
	if v, ok := in["description"]; ok {
		attrs["description"] = v
	}
	r, err := st.newResLocked(d, d.rules, "case-rule", attrs)
	if err != nil {
		return nil, err
	}
	return map[string]any{"caseRuleId": r.id, "caseRuleArn": r.arn}, nil
}

func (st *cncStore) updateCaseRule(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, r, err := st.resLocked(in, cncRules, "caseRuleId", "case rule")
	if err != nil {
		return nil, err
	}
	if rule := cncMap(in, "rule"); rule != nil {
		if err := st.checkRule(d, rule); err != nil {
			return nil, err
		}
	}
	return nil, st.updateResLocked(r, in, "name", "description", "rule")
}

func (st *cncStore) deleteCaseRule(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, r, err := st.resLocked(in, cncRules, "caseRuleId", "case rule")
	if err != nil {
		return nil, err
	}
	for _, t := range d.templates {
		for _, raw := range cncListOf(t.attrs["rules"]) {
			if cncStr(raw.(map[string]any), "caseRuleId") == r.id {
				return nil, cncConflict("case rule %s is used by template %s", r.id, t.id)
			}
		}
	}
	st.dropResLocked(d.rules, r)
	return nil, nil
}

func (st *cncStore) batchGetCaseRule(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	rules, errs := []any{}, []any{}
	for _, raw := range cncList(in, "caseRules") {
		id := cncStr(raw.(map[string]any), "id")
		if r := d.rules[id]; r != nil {
			rules = append(rules, r.out(st, "caseRuleId", "caseRuleArn", "name", "description", "rule"))
		} else {
			errs = append(errs, map[string]any{"id": id, "errorCode": "ResourceNotFoundException", "message": "case rule not found"})
		}
	}
	return map[string]any{"caseRules": rules, "errors": errs, "unprocessedCaseRules": []any{}}, nil
}

func (st *cncStore) listCaseRules(in map[string]any) (map[string]any, *cncErr) {
	return st.listRes(in, cncRules, "caseRules", func(r *cncRes) map[string]any {
		return map[string]any{"caseRuleId": r.id, "caseRuleArn": r.arn, "name": r.attrs["name"], "ruleType": cncRuleType(r), "description": r.attrs["description"]}
	})
}

// ---- event configuration ----

func (st *cncStore) getCaseEventConfiguration(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	if d.eventConfig == nil {
		return map[string]any{"eventBridge": map[string]any{"enabled": false}}, nil
	}
	return map[string]any{"eventBridge": cncCopy(d.eventConfig)}, nil
}

func (st *cncStore) putCaseEventConfiguration(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	eb := cncMap(in, "eventBridge")
	for _, raw := range cncList(cncMap(cncMap(eb, "includedData"), "caseData"), "fields") {
		if id := cncStr(raw.(map[string]any), "id"); d.fields[id] == nil {
			return nil, cncValidation("field %s doesn't exist", id)
		}
	}
	d.eventConfig = cncCopyMap(eb)
	return nil, nil
}
