package server

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Amazon CloudSearch Domain: the per-domain document and search service
// (REST JSON, signed cloudsearch). AWS serves it at the domain's own
// doc-/search- host; Stackyard also accepts the domain name as a path prefix,
// so an SDK endpoint of http://localhost:4566/<domain> works:
//   POST [/<domain>]/2013-01-01/documents/batch
//   GET|POST [/<domain>]/2013-01-01/search
//   GET|POST [/<domain>]/2013-01-01/suggest
// Documents are checked against, and searched with, the domain's deployed
// (indexed) configuration.

const (
	csdMaxSearchBody   = 1 << 20
	csdMaxBatchBytes   = 5 << 20 // AWS's batch limit
	csdMaxDocBytes     = 1 << 20
	csdMaxDomainBytes  = 64 << 20
	csdMaxStoreBytes   = 256 << 20
	csdMaxDomainDocs   = 100000
	csdMaxBatchOps     = 10000
	csdMaxFieldValues  = 1000
	csdDocOverhead     = 64 // bookkeeping bytes counted per stored id and value
	csdMaxSearchSize   = 10000
	csdDefaultSize     = 10
	csdMaxSuggestSize  = 100
	csdSearchErrorCode = "SearchException"
	csdDocErrorCode    = "DocumentServiceException"
)

var (
	csdPathRe  = regexp.MustCompile(`^(?:/([a-z][a-z0-9-]{2,27}))?/2013-01-01/(search|suggest|documents/batch)$`)
	csdHostRe  = regexp.MustCompile(`^(?:doc|search)-([a-z][a-z0-9-]{2,27})-[a-z0-9]{1,64}\.`)
	csdDocIDRe = regexp.MustCompile(`^[A-Za-z0-9_\-=#;:/?@&.]{1,128}$`)
)

func (s *Server) handleCloudSearchDomainRESTRouter(w http.ResponseWriter, r *http.Request) bool {
	m := csdPathRe.FindStringSubmatch(r.URL.Path)
	if m == nil {
		return false
	}
	host := strings.ToLower(r.Host)
	if hint := sigV4ServiceHint(r); hint != "cloudsearch" && (hint != "" || !strings.Contains(host, ".cloudsearch.")) {
		return false
	}
	op, errCode := "Search", csdSearchErrorCode
	switch m[2] {
	case "suggest":
		op = "Suggest"
	case "documents/batch":
		op, errCode = "UploadDocuments", csdDocErrorCode
	}
	limit := int64(csdMaxSearchBody)
	if op == "UploadDocuments" {
		limit = csdMaxBatchBytes
	}
	if r.ContentLength < 0 || r.ContentLength > limit {
		respondCSDError(w, http.StatusRequestEntityTooLarge, errCode, "request body must declare a Content-Length of at most "+strconv.FormatInt(limit, 10)+" bytes")
		return true
	}
	if (op == "UploadDocuments" && r.Method != http.MethodPost) || (r.Method != http.MethodGet && r.Method != http.MethodPost) {
		respondCSDError(w, http.StatusMethodNotAllowed, errCode, "method not allowed")
		return true
	}
	if ok, status, code, msg, _ := s.validateSigV4WithService(r, "cloudsearch"); !ok {
		respondCSDError(w, status, code, msg)
		return true
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, limit))
	if err != nil {
		respondCSDError(w, http.StatusBadRequest, errCode, "unable to read request body")
		return true
	}
	domain := m[1]
	if domain == "" {
		if hm := csdHostRe.FindStringSubmatch(host); hm != nil {
			domain = hm[1]
		}
	}
	var out map[string]any
	var cerr *csErr
	if op == "UploadDocuments" {
		out, cerr = s.cloudSearch.uploadDocuments(domain, r.Header.Get("Content-Type"), raw)
	} else {
		params := r.URL.Query()
		if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/x-www-form-urlencoded") {
			form, perr := url.ParseQuery(string(raw))
			if perr != nil {
				respondCSDError(w, http.StatusBadRequest, errCode, "invalid form body")
				return true
			}
			for k, v := range form {
				params[k] = v
			}
		}
		if op == "Search" {
			out, cerr = s.cloudSearch.search(domain, params)
		} else {
			out, cerr = s.cloudSearch.suggest(domain, params)
		}
	}
	if cerr != nil {
		respondCSDError(w, cerr.status, errCode, cerr.msg)
		return true
	}
	// out is built fresh under the store lock; nothing in it is shared.
	body, _ := json.Marshal(out)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	return true
}

func respondCSDError(w http.ResponseWriter, status int, code, msg string) {
	payload := map[string]string{"__type": code, "message": msg}
	if code == csdDocErrorCode {
		payload["status"] = "error"
	}
	body, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Amzn-ErrorType", code)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func csdBad(format string, args ...any) *csErr { return csValidation(format, args...) }

// csdDomain resolves the target domain; the caller holds mu.
func (st *csStore) csdDomain(name string) (*csDomain, *csErr) {
	if name == "" {
		return nil, &csErr{http.StatusNotFound, "", "no CloudSearch domain in the request: use the domain's doc-/search- endpoint or a /<domain> path prefix"}
	}
	d := st.domains[name]
	if d == nil {
		return nil, &csErr{http.StatusNotFound, "", "domain not found: " + name}
	}
	return d, nil
}

// ---- deployed index configuration ----

type csdField struct {
	name, typ string
	opts      map[string]any
}

func (f *csdField) array() bool { return strings.HasSuffix(f.typ, "-array") }
func (f *csdField) base() string {
	return strings.TrimSuffix(f.typ, "-array")
}

// enabled reads a boolean option, defaulting to true as AWS does for the
// return, search, facet and sort switches of non-text fields.
func (f *csdField) enabled(key string) bool {
	v, ok := f.opts[key].(bool)
	return !ok || v
}

type csdSchema struct {
	exact   map[string]*csdField
	dynamic []*csdField
}

func (d *csDomain) schema() *csdSchema {
	sc := &csdSchema{exact: map[string]*csdField{}}
	for name, o := range d.fields {
		field, _ := o.deployed.(map[string]any)
		if field == nil {
			continue
		}
		typ := csStr(field, "IndexFieldType")
		f := &csdField{name: name, typ: typ, opts: csMap(field, csTypeOptions[typ])}
		if strings.Contains(name, "*") {
			sc.dynamic = append(sc.dynamic, f)
		} else {
			sc.exact[name] = f
		}
	}
	return sc
}

func (sc *csdSchema) field(name string) *csdField {
	if f := sc.exact[name]; f != nil {
		return f
	}
	for _, f := range sc.dynamic {
		p := f.name
		if (strings.HasSuffix(p, "*") && strings.HasPrefix(name, strings.TrimSuffix(p, "*"))) ||
			(strings.HasPrefix(p, "*") && strings.HasSuffix(name, strings.TrimPrefix(p, "*"))) {
			return &csdField{name: name, typ: f.typ, opts: f.opts}
		}
	}
	return nil
}

// ---- document upload ----

type csdOp struct {
	add    bool
	id     string
	fields map[string][]string
}

func (st *csStore) uploadDocuments(domain, contentType string, raw []byte) (map[string]any, *csErr) {
	var ops []csdOp
	var err *csErr
	switch strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0])) {
	case "application/json":
		ops, err = csdParseJSONBatch(raw)
	case "application/xml":
		ops, err = csdParseXMLBatch(raw)
	default:
		return nil, csdBad("Content-Type must be application/json or application/xml")
	}
	if err != nil {
		return nil, err
	}
	if len(ops) == 0 {
		return nil, csdBad("The batch contains no documents.")
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	d, err := st.csdDomain(domain)
	if err != nil {
		return nil, err
	}
	sc := d.schema()
	docs := make([]*csDoc, len(ops))
	for i, op := range ops {
		if !op.add {
			continue
		}
		doc, verr := csdNewDoc(sc, op)
		if verr != nil {
			return nil, verr
		}
		docs[i] = doc
	}
	// Apply on a projection first so a batch past the limits changes nothing.
	count, bytesUsed, storeBytes := len(d.docs), d.docBytes, st.totalDocBytes()
	sizes := map[string]int{}
	for i, op := range ops {
		prev, seen := sizes[op.id]
		if !seen {
			if old := d.docs[op.id]; old != nil {
				prev = old.size
			} else {
				prev = -1
			}
		}
		if prev >= 0 {
			count--
			bytesUsed -= prev
			storeBytes -= prev
		}
		sizes[op.id] = -1
		if op.add {
			count++
			bytesUsed += docs[i].size
			storeBytes += docs[i].size
			sizes[op.id] = docs[i].size
		}
	}
	if count > csdMaxDomainDocs || bytesUsed > csdMaxDomainBytes || storeBytes > csdMaxStoreBytes {
		return nil, &csErr{http.StatusRequestEntityTooLarge, "", "the batch would exceed the domain's document storage limit"}
	}
	adds, deletes := 0, 0
	for i, op := range ops {
		if old := d.docs[op.id]; old != nil {
			d.docBytes -= old.size
			delete(d.docs, op.id)
		}
		if op.add {
			d.docs[op.id] = docs[i]
			d.docBytes += docs[i].size
			adds++
		} else {
			deletes++
		}
	}
	return map[string]any{"status": "success", "adds": adds, "deletes": deletes}, nil
}

func (st *csStore) totalDocBytes() int {
	total := 0
	for _, d := range st.domains {
		total += d.docBytes
	}
	return total
}

// csdNewDoc checks an add against the deployed fields and normalizes values.
func csdNewDoc(sc *csdSchema, op csdOp) (*csDoc, *csErr) {
	doc := &csDoc{fields: map[string][]string{}, size: csdDocOverhead + len(op.id)}
	values := 0
	for name, vals := range op.fields {
		f := sc.field(name)
		if f == nil {
			return nil, csdBad("Document %s: field %q is not defined in the domain's index configuration.", op.id, name)
		}
		if len(vals) > 1 && !f.array() {
			return nil, csdBad("Document %s: field %q of type %s takes a single value.", op.id, name, f.typ)
		}
		values += len(vals)
		if values > csdMaxFieldValues {
			return nil, csdBad("Document %s has more than %d field values.", op.id, csdMaxFieldValues)
		}
		out := make([]string, 0, len(vals))
		for _, v := range vals {
			norm, ok := csdNormalize(f.base(), v)
			if !ok {
				return nil, csdBad("Document %s: %q is not a valid %s value for field %q.", op.id, v, f.base(), name)
			}
			out = append(out, norm)
			doc.size += csdDocOverhead + len(name) + len(norm)
		}
		doc.fields[name] = out
	}
	if doc.size > csdMaxDocBytes {
		return nil, csdBad("Document %s is larger than 1 MB.", op.id)
	}
	return doc, nil
}

func csdNormalize(typ, v string) (string, bool) {
	switch typ {
	case "int":
		n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return strconv.FormatInt(n, 10), err == nil
	case "double":
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return "", false
		}
		return strconv.FormatFloat(f, 'f', -1, 64), true
	case "date":
		t, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(v))
		return t.UTC().Format("2006-01-02T15:04:05Z"), err == nil
	case "latlon":
		parts := strings.Split(v, ",")
		if len(parts) != 2 {
			return "", false
		}
		lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		lon, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err1 != nil || err2 != nil || math.Abs(lat) > 90 || math.Abs(lon) > 180 {
			return "", false
		}
		return strconv.FormatFloat(lat, 'f', -1, 64) + "," + strconv.FormatFloat(lon, 'f', -1, 64), true
	}
	return v, true
}

func csdCheckOp(i int, typ, id string, nfields int) (bool, *csErr) {
	if !csdDocIDRe.MatchString(id) {
		return false, csdBad("Operation %d: id %q must be 1-128 letters, digits or _ - = # ; : / ? @ & .", i+1, id)
	}
	switch typ {
	case "add":
		return true, nil
	case "delete":
		if nfields > 0 {
			return false, csdBad("Operation %d: a delete takes no fields.", i+1)
		}
		return false, nil
	}
	return false, csdBad("Operation %d: type must be add or delete.", i+1)
}

func csdParseJSONBatch(raw []byte) ([]csdOp, *csErr) {
	var batch []struct {
		Type   string                     `json:"type"`
		ID     string                     `json:"id"`
		Fields map[string]json.RawMessage `json:"fields"`
	}
	if err := json.Unmarshal(raw, &batch); err != nil {
		return nil, csdBad("The batch is not a JSON array of add and delete operations.")
	}
	if len(batch) > csdMaxBatchOps {
		return nil, csdBad("A batch can hold at most %d operations.", csdMaxBatchOps)
	}
	ops := make([]csdOp, 0, len(batch))
	for i, b := range batch {
		add, err := csdCheckOp(i, b.Type, b.ID, len(b.Fields))
		if err != nil {
			return nil, err
		}
		op := csdOp{add: add, id: b.ID, fields: map[string][]string{}}
		for name, v := range b.Fields {
			vals, ok := csdJSONValues(v)
			if !ok {
				return nil, csdBad("Document %s: field %q must be a string, a number or an array of them.", b.ID, name)
			}
			op.fields[name] = vals
		}
		ops = append(ops, op)
	}
	return ops, nil
}

func csdJSONValues(raw json.RawMessage) ([]string, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) > 0 && raw[0] == '[' {
		var items []json.RawMessage
		if json.Unmarshal(raw, &items) != nil {
			return nil, false
		}
		out := make([]string, 0, len(items))
		for _, item := range items {
			v, ok := csdJSONScalar(item)
			if !ok {
				return nil, false
			}
			out = append(out, v)
		}
		return out, true
	}
	v, ok := csdJSONScalar(raw)
	return []string{v}, ok
}

func csdJSONScalar(raw json.RawMessage) (string, bool) {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s, true
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String(), true
	}
	return "", false
}

func csdParseXMLBatch(raw []byte) ([]csdOp, *csErr) {
	var batch struct {
		XMLName xml.Name `xml:"batch"`
		Ops     []struct {
			XMLName xml.Name
			ID      string `xml:"id,attr"`
			Fields  []struct {
				Name  string `xml:"name,attr"`
				Value string `xml:",chardata"`
			} `xml:"field"`
		} `xml:",any"`
	}
	if err := xml.Unmarshal(raw, &batch); err != nil {
		return nil, csdBad("The batch is not a valid <batch> XML document.")
	}
	if len(batch.Ops) > csdMaxBatchOps {
		return nil, csdBad("A batch can hold at most %d operations.", csdMaxBatchOps)
	}
	ops := make([]csdOp, 0, len(batch.Ops))
	for i, b := range batch.Ops {
		add, err := csdCheckOp(i, b.XMLName.Local, b.ID, len(b.Fields))
		if err != nil {
			return nil, err
		}
		op := csdOp{add: add, id: b.ID, fields: map[string][]string{}}
		for _, f := range b.Fields {
			op.fields[f.Name] = append(op.fields[f.Name], f.Value)
		}
		ops = append(ops, op)
	}
	return ops, nil
}
