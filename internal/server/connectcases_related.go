package server

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Related items: contacts (chat contacts Connect started), comments, files,
// SLAs, linked cases and custom field sets.

type cncRelated struct {
	id, arn, typ         string
	seq                  int64
	content              map[string]any // output-shaped content union
	assoc                time.Time
	createdBy, updatedBy map[string]any
	size                 int
}

var cncRelatedTypes = map[string]string{"contact": "Contact", "comment": "Comment", "file": "File", "sla": "Sla", "connectCase": "ConnectCase", "custom": "Custom"}

// cncContactRef is a contact ARN resolved against the chat store.
type cncContactRef struct {
	arn     string
	view    cnpContactView
	invalid *cncErr
}

// resolveContact looks a contact ARN up before the cases lock is taken.
func (st *cncStore) resolveContact(content map[string]any) *cncContactRef {
	arn := cncStr(cncMap(content, "contact"), "contactArn")
	if arn == "" {
		return nil
	}
	ref := &cncContactRef{arn: arn}
	rest, ok := strings.CutPrefix(arn, "arn:aws:connect:")
	parts := strings.Split(rest, "/")
	if !ok || len(parts) != 4 || !strings.HasSuffix(parts[0], ":instance") || parts[2] != "contact" {
		ref.invalid = cncValidation("contactArn must be an Amazon Connect contact ARN")
		return ref
	}
	view, found := st.chats.contact(parts[1], parts[3], false)
	if !found {
		ref.invalid = cncNotFound("contact %s not found", arn)
	}
	ref.view = view
	return ref
}

// buildRelatedLocked turns a content union into stored, output-shaped content.
func (st *cncStore) buildRelatedLocked(d *cncDomain, c *cncCase, content map[string]any, contact *cncContactRef) (string, map[string]any, *cncErr) {
	kind, body, err := cncUnion(content, "content")
	if err != nil {
		return "", nil, err
	}
	m := body.(map[string]any)
	switch kind {
	case "contact":
		if contact.invalid != nil {
			return "", nil, contact.invalid
		}
		return "Contact", map[string]any{"contact": map[string]any{"contactArn": contact.arn, "channel": "CHAT", "connectedToSystemTime": cncTime(contact.view.started)}}, nil
	case "file":
		if !strings.HasPrefix(cncStr(m, "fileArn"), "arn:aws:connect:") || !strings.Contains(cncStr(m, "fileArn"), "/file/") {
			return "", nil, cncValidation("fileArn must be an Amazon Connect file ARN")
		}
	case "connectCase":
		id := cncStr(m, "caseId")
		if d.cases[id] == nil || id == c.id {
			return "", nil, cncValidation("connectCase.caseId must be another case of the domain")
		}
	case "custom":
		for _, f := range cncList(m, "fields") {
			fv := f.(map[string]any)
			if err := cncCheckValue(d, cncStr(fv, "id"), cncMap(fv, "value")); err != nil {
				return "", nil, err
			}
		}
	case "sla":
		cfg := cncMap(m, "slaInputConfiguration")
		if cfg == nil {
			return "", nil, cncValidation("sla must set slaInputConfiguration")
		}
		targets := cncList(cfg, "targetFieldValues")
		fid := cncStr(cfg, "fieldId")
		if (fid == "") != (len(targets) == 0) {
			return "", nil, cncValidation("an SLA sets fieldId and targetFieldValues together")
		}
		for _, t := range targets {
			if err := cncCheckValue(d, fid, t.(map[string]any)); err != nil {
				return "", nil, err
			}
		}
		minutes, _ := cfg["targetSlaMinutes"].(float64)
		out := map[string]any{"name": cfg["name"], "type": cfg["type"], "targetTime": cncTime(time.Now().Add(time.Duration(minutes) * time.Minute))}
		if fid != "" {
			out["fieldId"], out["targetFieldValues"] = fid, cncCopy(targets)
		}
		return "Sla", map[string]any{"sla": map[string]any{"slaConfiguration": out}}, nil
	}
	return cncRelatedTypes[kind], map[string]any{kind: cncCopy(m)}, nil
}

// contentOut computes live SLA status: Met once the case's field holds the
// target value, Overdue past the target time, Active otherwise.
func (ri *cncRelated) contentOut(c *cncCase) map[string]any {
	out := cncCopyMap(ri.content)
	if cfg := cncMap(cncMap(out, "sla"), "slaConfiguration"); cfg != nil {
		target, _ := time.Parse(cncTimeForm, cncStr(cfg, "targetTime"))
		status := "Active"
		if targets := cncList(cfg, "targetFieldValues"); len(targets) == 1 && cncSameValue(c.fields[cncStr(cfg, "fieldId")], targets[0].(map[string]any)) {
			status = "Met"
			cfg["completionTime"] = cncTime(c.modified)
		} else if time.Now().After(target) {
			status = "Overdue"
		}
		cfg["status"] = status
	}
	return out
}

func (ri *cncRelated) out(st *cncStore, c *cncCase) map[string]any {
	out := map[string]any{"relatedItemId": ri.id, "type": ri.typ, "associationTime": cncTime(ri.assoc), "content": ri.contentOut(c), "tags": st.tagsOut(ri.arn)}
	if ri.createdBy != nil {
		out["performedBy"] = cncCopy(ri.createdBy)
	}
	return out
}

// addRelatedLocked stores a related item on c within the case budgets.
func (st *cncStore) addRelatedLocked(d *cncDomain, c *cncCase, wantType string, in map[string]any, contact *cncContactRef) (*cncRelated, *cncErr) {
	typ, content, err := st.buildRelatedLocked(d, c, cncMap(in, "content"), contact)
	if err != nil {
		return nil, err
	}
	if wantType != "" && wantType != typ {
		return nil, cncValidation("type %s doesn't match %s content", wantType, typ)
	}
	if len(c.related) >= cncMaxRelated {
		return nil, cncQuota("a case holds at most %d related items", cncMaxRelated)
	}
	raw, _ := json.Marshal(content)
	size := len(raw) + 128
	if c.fieldBytes+c.relatedBytes+size > cncMaxCaseBytes || st.bytes+size > cncMaxStoreBytes {
		return nil, cncQuota("case storage is full")
	}
	id := uuid.NewString()
	ri := &cncRelated{id: id, arn: c.arn + "/related-item/" + id, typ: typ, seq: st.nextSeq(), content: content, assoc: time.Now().UTC(), size: size}
	if u := cncMap(in, "performedBy"); len(u) > 0 {
		ri.createdBy = cncCopyMap(u)
	}
	c.related[id] = ri
	c.relatedBytes += size
	st.bytes += size
	st.tags[ri.arn] = map[string]string{}
	st.addAuditLocked(c, "RelatedItem.Created", typ, []any{}, in, nil)
	return ri, nil
}

func (st *cncStore) createRelatedItem(in map[string]any) (map[string]any, *cncErr) {
	contact := st.resolveContact(cncMap(in, "content"))
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
	ri, err := st.addRelatedLocked(d, c, cncStr(in, "type"), in, contact)
	if err != nil {
		return nil, err
	}
	return map[string]any{"relatedItemId": ri.id, "relatedItemArn": ri.arn}, nil
}

func (st *cncStore) batchCreateRelatedItem(in map[string]any) (map[string]any, *cncErr) {
	contact := st.resolveContact(cncMap(in, "content"))
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	ok, failed := []any{}, []any{}
	for _, raw := range cncList(in, "cases") {
		c, ident, err := d.caseByIdentifier(cncMap(raw.(map[string]any), "caseIdentifier"))
		if err == nil {
			var ri *cncRelated
			if ri, err = st.addRelatedLocked(d, c, "", in, contact); err == nil {
				ok = append(ok, map[string]any{"caseIdentifier": ident, "relatedItemId": ri.id, "relatedItemArn": ri.arn})
				continue
			}
		}
		failed = append(failed, cncFailed(ident, err))
	}
	return map[string]any{"successfulItems": ok, "failedItems": failed}, nil
}

func (st *cncStore) relatedLocked(in map[string]any) (*cncCase, *cncRelated, *cncErr) {
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, nil, err
	}
	c, err := st.caseLocked(d, cncStr(in, "caseId"))
	if err != nil {
		return nil, nil, err
	}
	ri := c.related[cncStr(in, "relatedItemId")]
	if ri == nil {
		return nil, nil, cncNotFound("related item %s not found", cncStr(in, "relatedItemId"))
	}
	return c, ri, nil
}

func (st *cncStore) deleteRelatedItem(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, ri, err := st.relatedLocked(in)
	if err != nil {
		return nil, err
	}
	delete(c.related, ri.id)
	c.relatedBytes -= ri.size
	st.bytes -= ri.size
	delete(st.tags, ri.arn)
	st.addAuditLocked(c, "RelatedItem.Deleted", ri.typ, []any{}, in, nil)
	return nil, nil
}

func (st *cncStore) updateRelatedItem(in map[string]any) (map[string]any, *cncErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	c, ri, err := st.relatedLocked(in)
	if err != nil {
		return nil, err
	}
	d := st.domains[cncStr(in, "domainId")]
	kind, body, err := cncUnion(cncMap(in, "content"), "content")
	if err != nil {
		return nil, err
	}
	if cncRelatedTypes[kind] != ri.typ {
		return nil, cncValidation("a %s related item can't take %s content", ri.typ, kind)
	}
	m := body.(map[string]any)
	var content map[string]any
	switch kind {
	case "sla":
		upd := cncMap(m, "slaUpdateConfiguration")
		if upd == nil {
			return nil, cncValidation("sla must set slaUpdateConfiguration")
		}
		minutes, _ := upd["targetSlaMinutes"].(float64)
		content = cncCopyMap(ri.content)
		cfg := cncMap(cncMap(content, "sla"), "slaConfiguration")
		cfg["targetTime"] = cncTime(ri.assoc.Add(time.Duration(minutes) * time.Minute))
	case "custom":
		for _, f := range cncList(m, "fields") {
			fv := f.(map[string]any)
			if err := cncCheckValue(d, cncStr(fv, "id"), cncMap(fv, "value")); err != nil {
				return nil, err
			}
		}
		content = map[string]any{kind: cncCopy(m)}
	case "comment":
		content = map[string]any{kind: cncCopy(m)}
	default:
		// RelatedItemUpdateContent only has comment, custom and sla; contact,
		// file and connectCase content is validated at creation and can't change.
		return nil, cncValidation("content can only update a comment, custom or sla related item")
	}
	raw, _ := json.Marshal(content)
	size := len(raw) + 128
	if c.fieldBytes+c.relatedBytes-ri.size+size > cncMaxCaseBytes || st.bytes-ri.size+size > cncMaxStoreBytes {
		return nil, cncQuota("case storage is full")
	}
	c.relatedBytes += size - ri.size
	st.bytes += size - ri.size
	updated := *ri
	updated.content, updated.size = content, size
	if u := cncMap(in, "performedBy"); len(u) > 0 {
		updated.updatedBy = cncCopyMap(u)
	}
	c.related[ri.id] = &updated
	st.addAuditLocked(c, "RelatedItem.Updated", ri.typ, []any{}, in, nil)
	out := map[string]any{"relatedItemId": updated.id, "relatedItemArn": updated.arn, "type": updated.typ,
		"content": updated.contentOut(c), "associationTime": cncTime(updated.assoc), "tags": st.tagsOut(updated.arn)}
	if updated.createdBy != nil {
		out["createdBy"] = cncCopy(updated.createdBy)
	}
	if updated.updatedBy != nil {
		out["lastUpdatedUser"] = cncCopy(updated.updatedBy)
	}
	return out, nil
}

// ---- related item search ----

// cncRelatedFilters checks a filter list (OR'd) within the node budget.
func cncRelatedFilters(in map[string]any) ([]map[string]any, *cncErr) {
	budget := cncMaxFilterNodes
	out := []map[string]any{}
	for _, raw := range cncList(in, "filters") {
		f := raw.(map[string]any)
		kind, body, err := cncUnion(f, "filter")
		if err != nil {
			return nil, err
		}
		if kind == "custom" {
			if fields := cncMap(body.(map[string]any), "fields"); fields != nil {
				if err := cncCheckFieldsFilter(fields, 0, &budget); err != nil {
					return nil, err
				}
			}
		}
		out = append(out, f)
	}
	return out, nil
}

func (ri *cncRelated) matches(c *cncCase, filters []map[string]any) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		for kind, raw := range f {
			if cncRelatedTypes[kind] != ri.typ {
				continue
			}
			m, _ := raw.(map[string]any)
			content := cncMap(ri.content, kind)
			switch kind {
			case "contact":
				if arn := cncStr(m, "contactArn"); arn != "" && arn != cncStr(content, "contactArn") {
					continue
				}
				if ch := cncList(m, "channel"); len(ch) > 0 && !cncContains(ch, content["channel"]) {
					continue
				}
			case "file":
				if arn := cncStr(m, "fileArn"); arn != "" && arn != cncStr(content, "fileArn") {
					continue
				}
			case "connectCase":
				if id := cncStr(m, "caseId"); id != "" && id != cncStr(content, "caseId") {
					continue
				}
			case "sla":
				cfg := cncMap(cncMap(ri.contentOut(c), "sla"), "slaConfiguration")
				if name := cncStr(m, "name"); name != "" && name != cncStr(cfg, "name") {
					continue
				}
				if status := cncStr(m, "status"); status != "" && status != cncStr(cfg, "status") {
					continue
				}
			case "custom":
				if ff := cncMap(m, "fields"); ff != nil {
					values := map[string]map[string]any{}
					for _, fv := range cncList(content, "fields") {
						values[cncStr(fv.(map[string]any), "id")] = cncMap(fv.(map[string]any), "value")
					}
					if !cncEvalFieldsFilter(ff, values) {
						continue
					}
				}
			}
			return true
		}
	}
	return false
}

func cncContains(list []any, v any) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func (st *cncStore) searchRelatedItems(in map[string]any) (map[string]any, *cncErr) {
	filters, err := cncRelatedFilters(in)
	if err != nil {
		return nil, err
	}
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
	items := []*cncRelated{}
	for _, ri := range c.related {
		if ri.matches(c, filters) {
			items = append(items, ri)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].seq > items[j].seq })
	start, end, next, err := cncPage(in, len(items), 25, 25)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, ri := range items[start:end] {
		out = append(out, ri.out(st, c))
	}
	return cncWithNext(map[string]any{"relatedItems": out}, next), nil
}

func (st *cncStore) searchAllRelatedItems(in map[string]any) (map[string]any, *cncErr) {
	filters, err := cncRelatedFilters(in)
	if err != nil {
		return nil, err
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.domainLocked(cncStr(in, "domainId"))
	if err != nil {
		return nil, err
	}
	type hit struct {
		c  *cncCase
		ri *cncRelated
	}
	hits := []hit{}
	for _, c := range d.cases {
		for _, ri := range c.related {
			if ri.matches(c, filters) {
				hits = append(hits, hit{c, ri})
			}
		}
	}
	sorts := cncList(in, "sorts")
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].ri.seq > hits[j].ri.seq })
	for k := len(sorts) - 1; k >= 0; k-- {
		s := sorts[k].(map[string]any)
		desc := s["sortOrder"] == "Desc"
		byCase := s["sortProperty"] == "CaseId"
		sort.SliceStable(hits, func(i, j int) bool {
			a, b := hits[i], hits[j]
			if byCase {
				if a.c.id == b.c.id {
					return false
				}
				return (a.c.id < b.c.id) != desc
			}
			if a.ri.seq == b.ri.seq {
				return false
			}
			return (a.ri.seq < b.ri.seq) != desc
		})
	}
	start, end, next, err := cncPage(in, len(hits), 25, 25)
	if err != nil {
		return nil, err
	}
	out := []any{}
	for _, h := range hits[start:end] {
		o := h.ri.out(st, h.c)
		o["caseId"] = h.c.id
		out = append(out, o)
	}
	return cncWithNext(map[string]any{"relatedItems": out}, next), nil
}
