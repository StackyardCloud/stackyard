package server

import (
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Cases, their field values and audit history. Every case's stored bytes
// (field values, related items, audit events) are counted against a per-case
// and a store-wide budget; the audit history drops its oldest events to stay
// within the case budget.

type cncCase struct {
	id, arn, templateID, ref string
	clientToken              string
	seq                      int64
	fields                   map[string]map[string]any // field id -> value union; replaced, never mutated
	fieldBytes               int
	related                  map[string]*cncRelated
	relatedBytes             int
	audit                    []cncAudit
	auditBytes               int
	created, modified        time.Time
}

type cncAudit struct {
	ev   map[string]any
	size int
}

func (c *cncCase) size() int { return c.fieldBytes + c.relatedBytes + c.auditBytes }

func cncValueSize(id string, v map[string]any) int {
	n := len(id) + 32
	for _, k := range []string{"stringValue", "userArnValue"} {
		if s, ok := v[k].(string); ok {
			n += len(s)
		}
	}
	return n
}

func cncFieldsSize(fields map[string]map[string]any) int {
	n := 0
	for id, v := range fields {
		n += cncValueSize(id, v)
	}
	return n
}

// cncCheckValue checks a value against its field's type.
func cncCheckValue(d *cncDomain, id string, v map[string]any) *cncErr {
	f := d.fields[id]
	if f == nil {
		return cncValidation("field %s doesn't exist", id)
	}
	kind, val, err := cncUnion(v, "value of field "+id)
	if err != nil || kind == "emptyValue" {
		return err
	}
	typ := cncFieldType(f)
	want := map[string]string{"Text": "stringValue", "Url": "stringValue", "SingleSelect": "stringValue",
		"DateTime": "stringValue", "Number": "doubleValue", "Boolean": "booleanValue", "User": "userArnValue"}[typ]
	if kind != want && !(typ == "DateTime" && kind == "doubleValue") {
		return cncValidation("field %s is %s; it takes %s", id, typ, want)
	}
	s, _ := val.(string)
	switch typ {
	case "Url":
		if u, err := url.Parse(s); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return cncValidation("field %s takes an http(s) URL", id)
		}
	case "DateTime":
		if kind == "stringValue" {
			if _, err := time.Parse(time.RFC3339Nano, s); err != nil {
				return cncValidation("field %s takes an ISO 8601 date-time", id)
			}
		}
	case "SingleSelect":
		for _, o := range f.options {
			if o["value"] == s && o["active"] == true {
				return nil
			}
		}
		return cncValidation("%q isn't an active option of field %s", s, id)
	case "User":
		if !strings.HasPrefix(s, "arn:aws:connect:") || !strings.Contains(s, "/agent/") {
			return cncValidation("field %s takes an Amazon Connect user ARN", id)
		}
	}
	return nil
}

func cncEmpty(v map[string]any) bool {
	if v == nil || v["emptyValue"] != nil {
		return true
	}
	s, isStr := v["stringValue"].(string)
	return isStr && s == ""
}

// cncConditionHolds evaluates an equalTo/notEqualTo rule condition; compound
// conditions never match.
func cncConditionHolds(values map[string]map[string]any, c map[string]any) (bool, bool) {
	for kind, raw := range c {
		if kind != "equalTo" && kind != "notEqualTo" {
			return false, false
		}
		m := raw.(map[string]any)
		cur := values[cncStr(cncMap(m, "operandOne"), "fieldId")]
		var equal bool
		for k, want := range cncMap(m, "operandTwo") {
			if k == "emptyValue" {
				equal = cncEmpty(cur)
			} else {
				equal = cur != nil && cur[k] == want
			}
		}
		result, _ := m["result"].(bool)
		return equal == (kind == "equalTo"), result
	}
	return false, false
}

// checkRequired enforces a template's required fields and Required rules.
func (d *cncDomain) checkRequired(t *cncRes, values map[string]map[string]any) *cncErr {
	for _, raw := range cncListOf(t.attrs["requiredFields"]) {
		if id := cncStr(raw.(map[string]any), "fieldId"); cncEmpty(values[id]) {
			return cncValidation("template %s requires field %s", t.id, id)
		}
	}
	for _, raw := range cncListOf(t.attrs["rules"]) {
		ref := raw.(map[string]any)
		r := d.rules[cncStr(ref, "caseRuleId")]
		req := cncMap(cncMap(r.attrsOrNil(), "rule"), "required")
		field := cncStr(ref, "fieldId")
		if req == nil || field == "" {
			continue
		}
		required, _ := req["defaultValue"].(bool)
		for _, c := range cncList(req, "conditions") {
			if holds, result := cncConditionHolds(values, c.(map[string]any)); holds {
				required = result
				break
			}
		}
		if required && cncEmpty(values[field]) {
			return cncValidation("case rule %s requires field %s", r.id, field)
		}
	}
	return nil
}

func (r *cncRes) attrsOrNil() map[string]any {
	if r == nil {
		return nil
	}
	return r.attrs
}

func (st *cncStore) caseLocked(d *cncDomain, id string) (*cncCase, *cncErr) {
	c := d.cases[id]
	if c == nil {
		return nil, cncNotFound("case %s not found", id)
	}
	return c, nil
}

// performer shapes an audit event's performedBy from the request's user.
func cncPerformer(in map[string]any) map[string]any {
	p := map[string]any{"iamPrincipalArn": "arn:aws:iam::" + cncAccount + ":root"}
	if u := cncMap(in, "performedBy"); len(u) > 0 {
		p["user"] = cncCopy(u)
	}
	return p
}

// addAuditLocked appends an event and trims the oldest ones to the case budget.
func (st *cncStore) addAuditLocked(c *cncCase, typ, relatedType string, fields []any, in map[string]any, extra map[string]any) {
	ev := map[string]any{"eventId": uuid.NewString(), "type": typ, "performedTime": cncTime(time.Now()), "fields": fields, "performedBy": cncPerformer(in)}
	if relatedType != "" {
		ev["relatedItemType"] = relatedType
	}
	for k, v := range extra {
		ev[k] = v
	}
	size := 128
	for _, f := range fields {
		m := f.(map[string]any)
		size += cncValueSize(cncStr(m, "eventFieldId"), cncMap(m, "newValue")) + cncValueSize("", cncMap(m, "oldValue"))
	}
	c.audit = append(c.audit, cncAudit{ev, size})
	c.auditBytes += size
	st.bytes += size
	for len(c.audit) > 0 && (len(c.audit) > cncMaxAudit || c.size() > cncMaxCaseBytes || st.bytes > cncMaxStoreBytes) {
		c.auditBytes -= c.audit[0].size
		st.bytes -= c.audit[0].size
		c.audit = c.audit[1:]
	}
}

// setFieldsLocked applies new values (with audit field entries) within budget.
func (st *cncStore) setFieldsLocked(d *cncDomain, c *cncCase, updates []any) ([]any, *cncErr) {
	next := make(map[string]map[string]any, len(c.fields)+len(updates))
	for k, v := range c.fields {
		next[k] = v
	}
	auditFields := []any{}
	now := time.Now().UTC()
	for _, raw := range updates {
		m := raw.(map[string]any)
		id, v := cncStr(m, "id"), cncCopyMap(cncMap(m, "value"))
		if cncReadOnlyFields[id] {
			return nil, cncValidation("field %s is maintained by Amazon Connect Cases", id)
		}
		if err := cncCheckValue(d, id, v); err != nil {
			return nil, err
		}
		entry := map[string]any{"eventFieldId": id, "newValue": cncCopy(v)}
		if old := next[id]; old != nil {
			entry["oldValue"] = cncCopy(old)
		}
		auditFields = append(auditFields, entry)
		next[id] = v
		if id == "status" && v["stringValue"] == "closed" {
			next["last_closed_datetime"] = map[string]any{"stringValue": cncTime(now)}
		}
	}
	next["last_updated_datetime"] = map[string]any{"stringValue": cncTime(now)}
	if t := d.templates[c.templateID]; t != nil {
		if err := d.checkRequired(t, next); err != nil {
			return nil, err
		}
	}
	size := cncFieldsSize(next)
	if size+c.relatedBytes > cncMaxCaseBytes {
		return nil, cncValidation("a case holds at most %d bytes of fields and related items", cncMaxCaseBytes)
	}
	if st.bytes-c.fieldBytes+size > cncMaxStoreBytes {
		return nil, cncValidation("case storage is full")
	}
	st.bytes += size - c.fieldBytes
	c.fields, c.fieldBytes, c.modified = next, size, now
	return auditFields, nil
}

func (st *cncStore) createCase(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	token := cncStr(in, "clientToken")
	if c := d.cases[d.clientTokens[token]]; token != "" && c != nil {
		return map[string]any{"caseId": c.id, "caseArn": c.arn}, nil
	}
	t := d.templates[cncStr(in, "templateId")]
	if t == nil {
		return nil, cncNotFound("template %s not found", cncStr(in, "templateId"))
	}
	if t.attrs["status"] != "Active" {
		return nil, cncValidation("template %s is Inactive", t.id)
	}
	tags := cncCopyMap(cncMap(in, "tags"))
	for _, raw := range cncListOf(t.attrs["tagPropagationConfigurations"]) {
		for k, v := range cncMap(raw.(map[string]any), "tagMap") {
			tags[k] = v
		}
	}
	if len(tags) > cncMaxTags {
		return nil, cncValidation("a case holds at most %d tags", cncMaxTags)
	}
	if len(d.cases) >= cncMaxCasesPerDomain || st.cases >= cncMaxCases {
		return nil, cncValidation("case storage is full (%d per domain, %d in all)", cncMaxCasesPerDomain, cncMaxCases)
	}
	now := time.Now().UTC()
	id := strings.ReplaceAll(uuid.NewString(), "-", "")
	d.nextRef++
	c := &cncCase{id: id, arn: d.arn + "/case/" + id, templateID: t.id, ref: strconv.Itoa(d.nextRef), seq: st.nextSeq(),
		fields: map[string]map[string]any{
			"reference_number": {"stringValue": strconv.Itoa(d.nextRef)},
			"created_datetime": {"stringValue": cncTime(now)},
			"status":           {"stringValue": "open"},
		}, related: map[string]*cncRelated{}, created: now}
	c.fieldBytes = cncFieldsSize(c.fields)
	st.bytes += c.fieldBytes
	auditFields, err := st.setFieldsLocked(d, c, cncList(in, "fields"))
	if err != nil {
		st.bytes -= c.fieldBytes
		d.nextRef--
		return nil, err
	}
	d.cases[id] = c
	d.refs[c.ref] = id
	if token != "" {
		d.clientTokens[token] = id
		c.clientToken = token
	}
	st.cases++
	st.setTagsLocked(c.arn, tags)
	st.addAuditLocked(c, "Case.Created", "", auditFields, in, nil)
	return map[string]any{"caseId": id, "caseArn": c.arn}, nil
}

func (st *cncStore) getCase(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	c, err := st.caseLocked(d, cncStr(in, "caseId"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"templateId": c.templateID, "fields": c.fieldsOut(cncList(in, "fields")), "tags": st.tagsOut(c.arn)}, nil
}

// fieldsOut returns the requested ({id}) field values that are set.
func (c *cncCase) fieldsOut(requested []any) []any {
	out := []any{}
	for _, raw := range requested {
		id := cncStr(raw.(map[string]any), "id")
		if v := c.fields[id]; v != nil {
			out = append(out, map[string]any{"id": id, "value": cncCopy(v)})
		}
	}
	return out
}

func (st *cncStore) updateCase(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	c, err := st.caseLocked(d, cncStr(in, "caseId"))
	if err != nil {
		return nil, err
	}
	auditFields, err := st.setFieldsLocked(d, c, cncList(in, "fields"))
	if err != nil {
		return nil, err
	}
	st.addAuditLocked(c, "Case.Updated", "", auditFields, in, nil)
	return nil, nil
}

// caseByIdentifier resolves a {caseId | referenceId} union.
func (d *cncDomain) caseByIdentifier(ident map[string]any) (*cncCase, map[string]any, *cncErr) {
	kind, val, err := cncUnion(ident, "caseIdentifier")
	if err != nil {
		return nil, map[string]any{}, err
	}
	s, _ := val.(string)
	echo := map[string]any{kind: s}
	if kind == "referenceId" {
		s = d.refs[s]
	}
	c := d.cases[s]
	if c == nil {
		return nil, echo, cncNotFound("case not found")
	}
	return c, map[string]any{"caseId": c.id, "referenceNumber": c.ref}, nil
}

func cncFailed(ident map[string]any, err *cncErr) map[string]any {
	return map[string]any{"caseIdentifier": ident, "errorCode": err.code, "errorMessage": err.msg}
}

func (st *cncStore) batchUpdateCase(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	updates := []any{}
	for _, raw := range cncList(in, "fields") {
		m := raw.(map[string]any)
		kind, val, err := cncUnion(cncMap(m, "fieldIdentifier"), "fieldIdentifier")
		if err != nil {
			return nil, err
		}
		id, _ := val.(string)
		if kind == "name" {
			id = ""
			for fid, f := range d.fields {
				if f.attrs["name"] == val {
					id = fid
				}
			}
			if id == "" {
				return nil, cncValidation("no field is named %v", val)
			}
		}
		updates = append(updates, map[string]any{"id": id, "value": m["value"]})
	}
	for key, field := range map[string]string{"title": "title", "status": "status", "customerId": "customer_id"} {
		if v, ok := in[key].(string); ok {
			updates = append(updates, map[string]any{"id": field, "value": map[string]any{"stringValue": v}})
		}
	}
	ok, failed := []any{}, []any{}
	for _, raw := range cncList(in, "cases") {
		c, ident, err := d.caseByIdentifier(cncMap(raw.(map[string]any), "caseIdentifier"))
		if err == nil {
			var auditFields []any
			if auditFields, err = st.setFieldsLocked(d, c, updates); err == nil {
				st.addAuditLocked(c, "Case.Updated", "", auditFields, in, nil)
				ok = append(ok, map[string]any{"caseIdentifier": ident})
				continue
			}
		}
		failed = append(failed, cncFailed(ident, err))
	}
	return map[string]any{"successfulCases": ok, "failedCases": failed}, nil
}

func (st *cncStore) dropCaseLocked(d *cncDomain, c *cncCase) {
	for _, ri := range c.related {
		delete(st.tags, ri.arn)
	}
	delete(st.tags, c.arn)
	delete(d.refs, c.ref)
	delete(d.clientTokens, c.clientToken)
	delete(d.cases, c.id)
	st.bytes -= c.size()
	st.cases--
}

func (st *cncStore) deleteCase(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	c, err := st.caseLocked(d, cncStr(in, "caseId"))
	if err != nil {
		return nil, err
	}
	st.dropCaseLocked(d, c)
	return nil, nil
}

func (st *cncStore) getCaseAuditEvents(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	c, err := st.caseLocked(d, cncStr(in, "caseId"))
	if err != nil {
		return nil, err
	}
	start, end, next, err := cncPage(in, len(c.audit), 25, 25)
	if err != nil {
		return nil, err
	}
	events := []any{}
	for i := start; i < end; i++ { // newest first
		events = append(events, cncCopy(c.audit[len(c.audit)-1-i].ev))
	}
	return cncWithNext(map[string]any{"auditEvents": events}, next), nil
}

// ---- merges ----

func (st *cncStore) startMergeCases(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	primary, err := st.caseLocked(d, cncStr(in, "caseId"))
	if err != nil {
		return nil, err
	}
	ids := cncList(in, "secondaryCaseIds")
	if len(ids) != 1 {
		return nil, cncValidation("secondaryCaseIds takes exactly one case")
	}
	secondary, err := st.caseLocked(d, ids[0].(string))
	if err != nil {
		return nil, err
	}
	if secondary == primary {
		return nil, cncValidation("a case can't be merged into itself")
	}
	// Fields chosen from the secondary case are copied onto the primary.
	updates := []any{}
	for _, raw := range cncList(in, "fieldsConfiguration") {
		m := raw.(map[string]any)
		src := cncStr(m, "sourceCaseId")
		if src != primary.id && src != secondary.id {
			return nil, cncValidation("sourceCaseId %s isn't one of the merged cases", src)
		}
		if src == primary.id {
			continue
		}
		for _, f := range cncList(m, "fieldIds") {
			id := cncStr(f.(map[string]any), "id")
			if v := secondary.fields[id]; v != nil && !cncReadOnlyFields[id] {
				updates = append(updates, map[string]any{"id": id, "value": v})
			}
		}
	}
	cfg := cncMap(in, "relatedItemsConfiguration")
	keep := func(typ string) bool {
		key := map[string]string{"Comment": "comment", "ConnectCase": "connectCase", "Contact": "contact", "Custom": "custom"}[typ]
		v, set := cfg[key].(bool)
		return key != "" && (!set || v)
	}
	moved, movedBytes := 0, 0
	for _, ri := range secondary.related {
		if keep(ri.typ) {
			moved++
			movedBytes += ri.size
		}
	}
	if len(primary.related)+moved > cncMaxRelated {
		return nil, cncQuota("a case holds at most %d related items", cncMaxRelated)
	}
	if primary.fieldBytes+primary.relatedBytes+movedBytes > cncMaxCaseBytes {
		return nil, cncQuota("a case holds at most %d bytes of fields and related items", cncMaxCaseBytes)
	}
	// Count the incoming items while the field updates are checked against
	// the per-case cap, so fields plus moved items can't exceed it together.
	primary.relatedBytes += movedBytes
	auditFields, err := st.setFieldsLocked(d, primary, updates)
	primary.relatedBytes -= movedBytes
	if err != nil {
		return nil, err
	}
	for _, ri := range secondary.related {
		if !keep(ri.typ) {
			continue
		}
		delete(secondary.related, ri.id)
		secondary.relatedBytes -= ri.size
		delete(st.tags, ri.arn)
		cp := *ri
		cp.arn = primary.arn + "/related-item/" + ri.id
		primary.related[ri.id] = &cp
		primary.relatedBytes += ri.size
		st.tags[cp.arn] = map[string]string{}
	}
	mergeID := uuid.NewString()
	st.addAuditLocked(primary, "Case.Merged", "", auditFields, in, map[string]any{
		"mergeMetadata": map[string]any{"mergeEventId": mergeID, "primaryCaseId": primary.id, "secondaryCaseId": secondary.id},
	})
	st.dropCaseLocked(d, secondary)
	d.merges = append(d.merges, &cncMerge{primary.id, secondary.id})
	if len(d.merges) > cncMaxMerges {
		d.merges = d.merges[1:]
	}
	return nil, nil
}

func (st *cncStore) getMergeCasesStatus(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	id := cncStr(in, "caseId")
	var primary string
	secondaries := []any{}
	for _, m := range d.merges {
		if m.primary == id || m.secondary == id {
			primary = m.primary
		}
	}
	for _, m := range d.merges {
		if primary != "" && m.primary == primary {
			secondaries = append(secondaries, m.secondary)
		}
	}
	if primary == "" {
		return nil, cncNotFound("case %s has no merge", id)
	}
	return map[string]any{"status": "COMPLETED", "primaryCaseId": primary, "secondaryCaseIds": secondaries}, nil
}

// ---- contact lookups ----

func (st *cncStore) listCasesForContact(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	arn := cncStr(in, "contactArn")
	matches := []*cncCase{}
	for _, c := range d.cases {
		for _, ri := range c.related {
			if ri.typ == "Contact" && cncStr(cncMap(ri.content, "contact"), "contactArn") == arn {
				matches = append(matches, c)
				break
			}
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].seq < matches[j].seq })
	start, end, next, err := cncPage(in, len(matches), 10, 10)
	if err != nil {
		return nil, err
	}
	items := []any{}
	for _, c := range matches[start:end] {
		items = append(items, map[string]any{"caseId": c.id, "templateId": c.templateID})
	}
	return cncWithNext(map[string]any{"cases": items}, next), nil
}
