package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// Amazon Fraud Detector: awsJson1_1, X-Amz-Target AWSHawksNestServiceFacade.*,
// signed frauddetector. Records are kept in the shape the API returns them
// (camelCase members); bookkeeping lives under "_" keys, which conforming
// to the output shape drops.

const (
	frdTargetPrefix = "AWSHawksNestServiceFacade."
	frdRegion       = "us-east-1"
	frdAccount      = "123456789012"
	frdMaxPerKind   = 1000   // records of one kind
	frdMaxEvents    = 10000  // stored events
	frdMaxPreds     = 1000   // remembered predictions; the oldest are forgotten
	frdMaxElements  = 100000 // elements in one list
	frdMaxAllElems  = 200000 // elements across all lists (at most 320 bytes each)
	frdMaxTags      = 200
	frdMaxBytes     = 64 << 20 // stored records and events, store-wide
)

type frdStore struct {
	mu sync.Mutex
	// named holds variables, entity types, labels, outcomes, event types,
	// external models, lists, detectors, models, batch jobs, by kind then key.
	named    map[string]map[string]map[string]any
	versions map[string]map[string]map[string]any // detector id -> version id -> detector version
	nextVer  map[string]int
	rules    map[string]map[string]any // detectorId/ruleId/version -> rule detail
	mvs      map[string]map[string]any // modelType/modelId/version -> model version
	elements map[string][]string       // list name -> elements
	events   map[string]map[string]any // eventType/eventId -> event
	preds    []map[string]any
	predSeq  int
	deletes  map[string]string // event type -> deletion status
	tags     map[string]map[string]string
	kms      string
	sizes    map[string]int // charged bytes by record key
	bytes    int
}

func newFraudDetectorStore() *frdStore {
	return &frdStore{
		named:    map[string]map[string]map[string]any{},
		versions: map[string]map[string]map[string]any{},
		nextVer:  map[string]int{},
		rules:    map[string]map[string]any{},
		mvs:      map[string]map[string]any{},
		elements: map[string][]string{},
		events:   map[string]map[string]any{},
		deletes:  map[string]string{},
		tags:     map[string]map[string]string{},
		kms:      "DEFAULT",
		sizes:    map[string]int{},
	}
}

func frdValidation(format string, args ...any) *fcErr {
	return fcError("ValidationException", format, args...)
}

func frdNotFound(format string, args ...any) *fcErr {
	return fcError("ResourceNotFoundException", format, args...)
}

func frdConflict(format string, args ...any) *fcErr {
	return fcError("ConflictException", format, args...)
}

func frdNow() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000Z") }

func frdArn(path string) string {
	return "arn:aws:frauddetector:" + frdRegion + ":" + frdAccount + ":" + path
}

func (s *Server) handleFraudDetectorJSONRouter(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	op, ok := strings.CutPrefix(strings.TrimSpace(r.Header.Get("X-Amz-Target")), frdTargetPrefix)
	if !ok {
		return false
	}
	model := awsmodels.FraudDetectorModel()
	in, ferr := fcDecode(s, w, r, "frauddetector", model, op)
	if ferr != nil {
		respondFraudDetectorError(w, ferr)
		return true
	}
	if in == nil {
		return true
	}
	out, ferr := s.fraudDetector.run(op, in)
	if ferr != nil {
		respondFraudDetectorError(w, ferr)
		return true
	}
	frdWriteJSON(w, http.StatusOK, out, "")
	return true
}

func respondFraudDetectorError(w http.ResponseWriter, e *fcErr) {
	code := e.code
	if code == "InvalidInputException" || code == "InvalidNextTokenException" {
		code = "ValidationException"
	}
	status := http.StatusBadRequest
	switch code {
	case "ResourceNotFoundException":
		status = http.StatusNotFound
	case "ConflictException":
		status = http.StatusConflict
	}
	frdWriteJSON(w, status, map[string]any{"__type": code, "message": e.msg}, code)
}

func frdWriteJSON(w http.ResponseWriter, status int, v any, errorType string) {
	body, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/x-amz-json-1.1")
	if errorType != "" {
		w.Header().Set("X-Amzn-ErrorType", errorType)
	}
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

type frdHandler func(st *frdStore, in map[string]any) (map[string]any, *fcErr)

// run executes op under the store lock and conforms its output there.
func (st *frdStore) run(op string, in map[string]any) (any, *fcErr) {
	h := frdHandlers[op]
	if h == nil {
		return nil, fcError("UnknownOperationException", "unknown operation %s", op)
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
	model := awsmodels.FraudDetectorModel()
	return model.Conform(model.Operations[op].Output, out), nil
}

func (st *frdStore) coll(kind string) map[string]map[string]any {
	c := st.named[kind]
	if c == nil {
		c = map[string]map[string]any{}
		st.named[kind] = c
	}
	return c
}

// charge accounts for a new record's size under key, refusing it past the
// store-wide byte bound.
func (st *frdStore) charge(key string, rec map[string]any) *fcErr {
	raw, _ := json.Marshal(rec)
	if st.bytes-st.sizes[key]+len(raw) > frdMaxBytes {
		return frdValidation("stored Fraud Detector data is limited to %d MiB", frdMaxBytes>>20)
	}
	st.bytes += len(raw) - st.sizes[key]
	st.sizes[key] = len(raw)
	return nil
}

func (st *frdStore) refund(key string) {
	st.bytes -= st.sizes[key]
	delete(st.sizes, key)
}

// add stores a new record with its ARN, times and tags.
func (st *frdStore) add(kind, key, arnPath string, rec map[string]any, tags any) *fcErr {
	c := st.coll(kind)
	if len(c) >= frdMaxPerKind {
		return frdValidation("at most %d %s resources", frdMaxPerKind, kind)
	}
	t, err := frdTagMap(nil, tags)
	if err != nil {
		return err
	}
	if err := st.charge(kind+"/"+key, rec); err != nil {
		return err
	}
	now := frdNow()
	rec["arn"] = frdArn(arnPath)
	fcDefault(rec, "createdTime", now)
	rec["lastUpdatedTime"] = now
	c[key] = rec
	st.tags[frdArn(arnPath)] = t
	return nil
}

func (st *frdStore) drop(kind, key string) {
	if rec := st.coll(kind)[key]; rec != nil {
		delete(st.tags, fcStr(rec, "arn"))
	}
	delete(st.coll(kind), key)
	st.refund(kind + "/" + key)
}

// frdCopy returns a shallow copy of rec, so updates never change a map a
// response may still be shaping.
func frdCopy(rec map[string]any) map[string]any {
	out := make(map[string]any, len(rec)+2)
	for k, v := range rec {
		out[k] = v
	}
	return out
}

func frdSortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// frdPage pages items; Fraud Detector reports bad tokens as ValidationException.
func frdPage(items []any, in map[string]any, tokenKey string, def int) ([]any, string, *fcErr) {
	page, next, err := fcPage(items, in, tokenKey, "maxResults", def)
	if err != nil {
		return nil, "", frdValidation("invalid nextToken")
	}
	return page, next, nil
}

// getNamed answers Get<Kind>s: one record when name is given, else a page.
func (st *frdStore) getNamed(kind, name string, in map[string]any, listKey string, def int) (map[string]any, *fcErr) {
	c := st.coll(kind)
	if name != "" {
		rec := c[name]
		if rec == nil {
			return nil, frdNotFound("%s %s not found", kind, name)
		}
		return map[string]any{listKey: []any{st.viewNamed(kind, rec)}}, nil
	}
	var items []any
	for _, k := range frdSortedKeys(c) {
		items = append(items, st.viewNamed(kind, c[k]))
	}
	page, next, err := frdPage(items, in, "nextToken", def)
	if err != nil {
		return nil, err
	}
	out := map[string]any{listKey: page}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

func (st *frdStore) viewNamed(kind string, rec map[string]any) map[string]any {
	switch kind {
	case "event-type":
		v := frdCopy(rec)
		v["ingestedEventStatistics"] = st.eventStats(fcStr(rec, "name"))
		return v
	case "batch-import", "batch-prediction":
		frdSettleJob(rec)
	}
	return rec
}

// putSimple is PutEntityType, PutLabel and PutOutcome: create or update.
func (st *frdStore) putSimple(kind string, in map[string]any) *fcErr {
	name := fcStr(in, "name")
	if old := st.coll(kind)[name]; old != nil {
		rec := frdCopy(old)
		if d, ok := in["description"]; ok {
			rec["description"] = d
		}
		rec["lastUpdatedTime"] = frdNow()
		st.coll(kind)[name] = rec
		return nil
	}
	return st.add(kind, name, kind+"/"+name, map[string]any{"name": name, "description": in["description"]}, in["tags"])
}

func frdTagMap(base map[string]string, raw any) (map[string]string, *fcErr) {
	out := make(map[string]string, len(base))
	for k, v := range base {
		out[k] = v
	}
	for _, m := range fcObjList(raw) {
		key := fcStr(m, "key")
		if strings.HasPrefix(strings.ToLower(key), "aws:") {
			return nil, frdValidation("tag keys can't start with aws:")
		}
		out[key] = fcStr(m, "value")
	}
	if len(out) > frdMaxTags {
		return nil, frdValidation("a resource can have at most %d tags", frdMaxTags)
	}
	return out, nil
}

func (st *frdStore) tagResource(in map[string]any) (map[string]any, *fcErr) {
	arn := fcStr(in, "resourceARN")
	old, ok := st.tags[arn]
	if !ok {
		return nil, frdNotFound("resource %s not found", arn)
	}
	t, err := frdTagMap(old, in["tags"])
	if err != nil {
		return nil, err
	}
	st.tags[arn] = t
	return nil, nil
}

func (st *frdStore) untagResource(in map[string]any) (map[string]any, *fcErr) {
	arn := fcStr(in, "resourceARN")
	old, ok := st.tags[arn]
	if !ok {
		return nil, frdNotFound("resource %s not found", arn)
	}
	t, _ := frdTagMap(old, nil)
	for _, k := range fcStrs(in, "tagKeys") {
		delete(t, k)
	}
	st.tags[arn] = t
	return nil, nil
}

func (st *frdStore) listTags(in map[string]any) (map[string]any, *fcErr) {
	arn := fcStr(in, "resourceARN")
	t, ok := st.tags[arn]
	if !ok {
		return nil, frdNotFound("resource %s not found", arn)
	}
	var items []any
	for _, k := range frdSortedKeys(t) {
		items = append(items, map[string]any{"key": k, "value": t[k]})
	}
	page, next, err := frdPage(items, in, "nextToken", 50)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"tags": page}
	if next != "" {
		out["nextToken"] = next
	}
	return out, nil
}

// frdHandlers maps every operation to its handler; see frauddetector_*.go.
var frdHandlers map[string]frdHandler

func init() {
	frdHandlers = map[string]frdHandler{
		"TagResource":         (*frdStore).tagResource,
		"UntagResource":       (*frdStore).untagResource,
		"ListTagsForResource": (*frdStore).listTags,
		"GetKMSEncryptionKey": func(st *frdStore, _ map[string]any) (map[string]any, *fcErr) {
			return map[string]any{"kmsKey": map[string]any{"kmsEncryptionKeyArn": st.kms}}, nil
		},
		"PutKMSEncryptionKey": func(st *frdStore, in map[string]any) (map[string]any, *fcErr) {
			st.kms = fcStr(in, "kmsEncryptionKeyArn")
			return nil, nil
		},
	}
	for name, h := range frdEntityHandlers() {
		frdHandlers[name] = h
	}
	for name, h := range frdDetectorHandlers() {
		frdHandlers[name] = h
	}
	for name, h := range frdModelHandlers() {
		frdHandlers[name] = h
	}
	for name, h := range frdEventHandlers() {
		frdHandlers[name] = h
	}
}
