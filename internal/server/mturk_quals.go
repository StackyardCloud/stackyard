package server

import (
	"sort"
	"strings"
	"time"
)

// Qualification types, worker qualifications and requests, worker blocks and
// the simulated workers' records.

// System qualification types every requester can use. Masters is never held
// in the sandbox; the others come from the worker's record.
const (
	mtkQualPercentApproved = "000000000000000000L0"
	mtkQualHITsApproved    = "00000000000000000040"
	mtkQualLocale          = "00000000000000000071"
	mtkQualAdult           = "00000000000000000060"
	mtkQualMasters         = "2ARFPLSP75KLA8M8DH1HTEQVJT3SY6"
)

var mtkSystemQuals = map[string]string{
	mtkQualPercentApproved: "Worker_PercentAssignmentsApproved",
	mtkQualHITsApproved:    "Worker_NumberHITsApproved",
	mtkQualLocale:          "Worker_Locale",
	mtkQualAdult:           "Worker_Adult",
	mtkQualMasters:         "Masters",
}

type mtkQualType struct {
	id, name, desc, keywords, status, test, answerKey string
	created                                           time.Time
	seq                                               int
	retryDelay, testDuration                          int64
	hasRetry                                          bool
	autoGranted                                       bool
	autoValue                                         int64
	size                                              int // bytes charged to mtkStore.bytes
}

type mtkQual struct {
	value   int64
	status  string
	granted time.Time
}

type mtkQualRequest struct {
	id, typeID, worker, answer string
	submitted                  time.Time
	seq                        int
	size                       int // bytes charged to mtkStore.bytes
}

type mtkWorker struct {
	id, country, subdivision      string
	submitted, approved, rejected int
}

type mtkQualState struct {
	seq        int
	types      map[string]*mtkQualType
	names      map[string]string              // lower-cased name -> type id
	held       map[string]map[string]*mtkQual // type id -> worker -> qualification
	heldCount  int                            // qualifications across all types
	requests   map[string]*mtkQualRequest     // request id -> request
	pending    map[string]string              // type/worker -> open request id
	rejected   map[string]time.Time           // type/worker -> last rejection
	blocks     map[string]string              // worker -> reason
	workers    map[string]*mtkWorker          // simulated workers
	hitsByQual map[string]map[string]bool     // qualification type -> HITs requiring it
}

func newMTurkQualState() mtkQualState {
	return mtkQualState{
		types: map[string]*mtkQualType{}, names: map[string]string{}, held: map[string]map[string]*mtkQual{},
		requests: map[string]*mtkQualRequest{}, pending: map[string]string{}, rejected: map[string]time.Time{},
		blocks: map[string]string{}, workers: map[string]*mtkWorker{}, hitsByQual: map[string]map[string]bool{},
	}
}

func (qs *mtkQualState) worker(id string) *mtkWorker { return qs.workers[id] }

func (qs *mtkQualState) ensureWorker(id, country, subdivision string) (*mtkWorker, *mtkErr) {
	if w := qs.workers[id]; w != nil {
		return w, nil
	}
	if len(qs.workers) >= mtkMaxWorkers {
		return nil, mtkError("AWS.MechanicalTurk.LimitExceeded", "Stackyard simulates at most %d workers", mtkMaxWorkers)
	}
	if country == "" {
		country = "US"
	}
	if len(country) != 2 || len(subdivision) > 2 {
		return nil, mtkInvalid("Country and Subdivision are two-letter codes")
	}
	w := &mtkWorker{id: id, country: strings.ToUpper(country), subdivision: strings.ToUpper(subdivision)}
	qs.workers[id] = w
	return w, nil
}

// indexHIT moves hit from the qualification types in from to those in to.
func (qs *mtkQualState) indexHIT(hit string, from, to []any) {
	for _, q := range fcObjList(from) {
		delete(qs.hitsByQual[fcStr(q, "QualificationTypeId")], hit)
	}
	for _, q := range fcObjList(to) {
		id := fcStr(q, "QualificationTypeId")
		if qs.hitsByQual[id] == nil {
			qs.hitsByQual[id] = map[string]bool{}
		}
		qs.hitsByQual[id][hit] = true
	}
}

func (qs *mtkQualState) qualType(id string) (*mtkQualType, *mtkErr) {
	if name, ok := mtkSystemQuals[id]; ok {
		return &mtkQualType{id: id, name: name, desc: "System qualification: " + name, status: "Active"}, nil
	}
	t := qs.types[id]
	if t == nil {
		return nil, mtkError("AWS.MechanicalTurk.QualificationTypeDoesNotExist", "Qualification type %s does not exist.", id)
	}
	return t, nil
}

func (qs *mtkQualState) checkRequirement(q map[string]any) *mtkErr {
	id := fcStr(q, "QualificationTypeId")
	if _, err := qs.qualType(id); err != nil {
		return err
	}
	cmp := fcStr(q, "Comparator")
	ints, locales := fcNums(q, "IntegerValues"), fcObjList(q["LocaleValues"])
	switch cmp {
	case "Exists", "DoesNotExist":
		if len(ints)+len(locales) > 0 {
			return mtkInvalid("%s takes no values", cmp)
		}
	case "In", "NotIn":
		if len(ints)+len(locales) == 0 || len(ints)+len(locales) > 30 {
			return mtkInvalid("%s takes 1-30 values", cmp)
		}
	default:
		if len(ints)+len(locales) != 1 {
			return mtkInvalid("%s takes exactly one value", cmp)
		}
	}
	return nil
}

func fcNums(m map[string]any, key string) []float64 {
	items, _ := m[key].([]any)
	out := make([]float64, 0, len(items))
	for _, item := range items {
		if n, ok := item.(float64); ok {
			out = append(out, n)
		}
	}
	return out
}

// value reports the worker's value for a qualification type.
func (qs *mtkQualState) value(w *mtkWorker, typeID string) (int64, bool) {
	switch typeID {
	case mtkQualPercentApproved:
		if w.approved+w.rejected == 0 {
			return 100, true
		}
		return int64(w.approved * 100 / (w.approved + w.rejected)), true
	case mtkQualHITsApproved:
		return int64(w.approved), true
	case mtkQualAdult:
		return 1, true
	case mtkQualMasters:
		return 0, false
	}
	q := qs.held[typeID][w.id]
	if q == nil || q.status != "Granted" {
		return 0, false
	}
	return q.value, true
}

// meets evaluates one qualification requirement against a worker.
func (qs *mtkQualState) meets(w *mtkWorker, req map[string]any) bool {
	id, cmp := fcStr(req, "QualificationTypeId"), fcStr(req, "Comparator")
	if id == mtkQualLocale {
		match := false
		for _, l := range fcObjList(req["LocaleValues"]) {
			if strings.EqualFold(fcStr(l, "Country"), w.country) && (fcStr(l, "Subdivision") == "" || strings.EqualFold(fcStr(l, "Subdivision"), w.subdivision)) {
				match = true
			}
		}
		switch cmp {
		case "EqualTo", "In":
			return match
		case "NotEqualTo", "NotIn":
			return !match
		}
		return cmp == "Exists"
	}
	v, held := qs.value(w, id)
	switch cmp {
	case "Exists":
		return held
	case "DoesNotExist":
		return !held
	}
	if !held {
		return false
	}
	vals := fcNums(req, "IntegerValues")
	if len(vals) == 0 {
		return false
	}
	x := float64(v)
	switch cmp {
	case "LessThan":
		return x < vals[0]
	case "LessThanOrEqualTo":
		return x <= vals[0]
	case "GreaterThan":
		return x > vals[0]
	case "GreaterThanOrEqualTo":
		return x >= vals[0]
	case "EqualTo":
		return x == vals[0]
	case "NotEqualTo":
		return x != vals[0]
	}
	in := false
	for _, n := range vals {
		in = in || x == n
	}
	return in == (cmp == "In")
}

func mtkQualHandlers() map[string]mtkHandler {
	return map[string]mtkHandler{
		"CreateQualificationType": (*mtkStore).createQualType, "GetQualificationType": (*mtkStore).getQualType,
		"UpdateQualificationType": (*mtkStore).updateQualType, "DeleteQualificationType": (*mtkStore).deleteQualType,
		"ListQualificationTypes":           (*mtkStore).listQualTypes,
		"AssociateQualificationWithWorker": (*mtkStore).associateQual, "DisassociateQualificationFromWorker": (*mtkStore).disassociateQual,
		"GetQualificationScore": (*mtkStore).qualScore, "ListWorkersWithQualificationType": (*mtkStore).listWorkersWithQual,
		"ListQualificationRequests":  (*mtkStore).listQualRequests,
		"AcceptQualificationRequest": (*mtkStore).acceptQualRequest, "RejectQualificationRequest": (*mtkStore).rejectQualRequest,
		"CreateWorkerBlock": (*mtkStore).createBlock, "DeleteWorkerBlock": (*mtkStore).deleteBlock, "ListWorkerBlocks": (*mtkStore).listBlocks,
		"NotifyWorkers": (*mtkStore).notifyWorkers,
	}
}

func (t *mtkQualType) view() map[string]any {
	_, system := mtkSystemQuals[t.id]
	out := map[string]any{
		"QualificationTypeId": t.id, "Name": t.name, "Description": t.desc, "QualificationTypeStatus": t.status,
		"IsRequestable": !system, "AutoGranted": t.autoGranted,
	}
	if !t.created.IsZero() {
		out["CreationTime"] = mtkEpoch(t.created)
	}
	if t.keywords != "" {
		out["Keywords"] = t.keywords
	}
	if t.test != "" {
		out["Test"], out["TestDurationInSeconds"] = t.test, t.testDuration
	}
	if t.answerKey != "" {
		out["AnswerKey"] = t.answerKey
	}
	if t.hasRetry {
		out["RetryDelayInSeconds"] = t.retryDelay
	}
	if t.autoGranted {
		out["AutoGrantedValue"] = t.autoValue
	}
	return out
}

// apply sets the updatable properties present in in.
func (t *mtkQualType) apply(in map[string]any) *mtkErr {
	if v, ok := in["Description"].(string); ok {
		t.desc = v
	}
	if v, ok := in["QualificationTypeStatus"].(string); ok {
		t.status = v
	}
	if v, ok := in["Test"].(string); ok {
		t.test = v
	}
	if v, ok := in["AnswerKey"].(string); ok {
		t.answerKey = v
	}
	if _, ok := in["TestDurationInSeconds"]; ok {
		t.testDuration = mtkInt(in, "TestDurationInSeconds", 0)
	}
	if _, ok := in["RetryDelayInSeconds"]; ok {
		t.retryDelay, t.hasRetry = mtkInt(in, "RetryDelayInSeconds", 0), true
	}
	if v, ok := in["AutoGranted"].(bool); ok {
		t.autoGranted = v
	}
	if _, ok := in["AutoGrantedValue"]; ok {
		t.autoValue = mtkInt(in, "AutoGrantedValue", 1)
	}
	switch {
	case strings.TrimSpace(t.desc) == "" || len(t.desc) > 2000:
		return mtkInvalid("Description must have 1-2000 characters")
	case len(t.test) > mtkMaxQuestion || len(t.answerKey) > mtkMaxQuestion:
		return mtkInvalid("Test and AnswerKey can have at most %d characters", mtkMaxQuestion)
	case t.test != "" && t.autoGranted:
		return mtkInvalid("a qualification type with a Test can't be AutoGranted")
	case t.answerKey != "" && t.test == "":
		return mtkInvalid("AnswerKey needs a Test")
	case t.test != "" && (t.testDuration < mtkMinDuration || t.testDuration > mtkMaxDuration):
		return mtkInvalid("TestDurationInSeconds must be between %d and %d when there is a Test", mtkMinDuration, mtkMaxDuration)
	case t.retryDelay < 0:
		return mtkInvalid("RetryDelayInSeconds can't be negative")
	}
	return nil
}

func (st *mtkStore) createQualType(q mtkReq) (map[string]any, *mtkErr) {
	qs := &st.quals
	name := strings.TrimSpace(fcStr(q.in, "Name"))
	if name == "" || len(name) > 255 {
		return nil, mtkInvalid("Name must have 1-255 characters")
	}
	if id, taken := qs.names[strings.ToLower(name)]; taken {
		return nil, mtkError("AWS.MechanicalTurk.QualificationTypeAlreadyExists", "You have already created a QualificationType with this name. A QualificationType's name must be unique among all of the QualificationTypes created by the same user. (%s)", id)
	}
	if len(qs.types) >= mtkMaxQualTypes {
		return nil, mtkError("AWS.MechanicalTurk.LimitExceeded", "Stackyard keeps at most %d qualification types", mtkMaxQualTypes)
	}
	qs.seq++
	t := &mtkQualType{id: mtkNewID(), name: name, keywords: fcStr(q.in, "Keywords"), created: st.now(), seq: qs.seq, autoValue: 1}
	if err := t.apply(q.in); err != nil {
		return nil, err
	}
	if err := st.chargeQualType(t, nil); err != nil {
		return nil, err
	}
	qs.types[t.id] = t
	qs.names[strings.ToLower(name)] = t.id
	return map[string]any{"QualificationType": t.view()}, nil
}

func (st *mtkStore) getQualType(q mtkReq) (map[string]any, *mtkErr) {
	t, err := st.quals.qualType(fcStr(q.in, "QualificationTypeId"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"QualificationType": t.view()}, nil
}

// ownType returns a qualification type the caller created.
func (qs *mtkQualState) ownType(id string) (*mtkQualType, *mtkErr) {
	if _, system := mtkSystemQuals[id]; system {
		return nil, mtkError("AWS.MechanicalTurk.PermissionDenied", "system qualification type %s can't be changed or assigned", id)
	}
	return qs.qualType(id)
}

func (st *mtkStore) updateQualType(q mtkReq) (map[string]any, *mtkErr) {
	t, err := st.quals.ownType(fcStr(q.in, "QualificationTypeId"))
	if err != nil {
		return nil, err
	}
	next := *t
	if err := next.apply(q.in); err != nil {
		return nil, err
	}
	if err := st.chargeQualType(&next, t); err != nil {
		return nil, err
	}
	*t = next
	return map[string]any{"QualificationType": t.view()}, nil
}

func (st *mtkStore) deleteQualType(q mtkReq) (map[string]any, *mtkErr) {
	qs := &st.quals
	t, err := qs.ownType(fcStr(q.in, "QualificationTypeId"))
	if err != nil {
		return nil, err
	}
	for id, r := range qs.requests { // bounded by mtkMaxQualRequests
		if r.typeID == t.id {
			_, _ = st.takeRequest(id)
		}
	}
	st.bytes -= t.size
	qs.heldCount -= len(qs.held[t.id])
	delete(qs.held, t.id)
	delete(qs.names, strings.ToLower(t.name))
	delete(qs.types, t.id)
	return nil, nil
}

func (st *mtkStore) listQualTypes(q mtkReq) (map[string]any, *mtkErr) {
	qs := &st.quals
	query := strings.ToLower(fcStr(q.in, "Query"))
	requestable, owned := fcBool(q.in, "MustBeRequestable"), fcBool(q.in, "MustBeOwnedByCaller")
	var list []*mtkQualType
	match := func(t *mtkQualType) bool {
		return query == "" || strings.Contains(strings.ToLower(t.name+" "+t.desc+" "+t.keywords), query)
	}
	if !owned && !requestable {
		ids := make([]string, 0, len(mtkSystemQuals))
		for id := range mtkSystemQuals {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			if t, _ := qs.qualType(id); match(t) {
				list = append(list, t)
			}
		}
	}
	own := make([]*mtkQualType, 0, len(qs.types))
	for _, t := range qs.types {
		if match(t) && (!requestable || t.status == "Active") {
			own = append(own, t)
		}
	}
	sort.Slice(own, func(i, j int) bool { return own[i].seq < own[j].seq })
	list = append(list, own...)
	start, end, next, err := mtkPage(len(list), q.in)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, t := range list[start:end] {
		items = append(items, t.view())
	}
	return mtkPaged("QualificationTypes", items, next), nil
}

// grant gives worker the qualification with value, counting new ones.
func (qs *mtkQualState) grant(typeID, worker string, value int64, at time.Time) *mtkErr {
	held := qs.held[typeID]
	if held == nil {
		held = map[string]*mtkQual{}
		qs.held[typeID] = held
	}
	if held[worker] == nil {
		if qs.heldCount >= mtkMaxQuals {
			return mtkError("AWS.MechanicalTurk.LimitExceeded", "Stackyard keeps at most %d worker qualifications", mtkMaxQuals)
		}
		qs.heldCount++
	}
	held[worker] = &mtkQual{value: value, status: "Granted", granted: at}
	return nil
}

func (st *mtkStore) associateQual(q mtkReq) (map[string]any, *mtkErr) {
	t, err := st.quals.ownType(fcStr(q.in, "QualificationTypeId"))
	if err != nil {
		return nil, err
	}
	return nil, st.quals.grant(t.id, fcStr(q.in, "WorkerId"), mtkInt(q.in, "IntegerValue", 1), st.now())
}

func (st *mtkStore) disassociateQual(q mtkReq) (map[string]any, *mtkErr) {
	t, err := st.quals.ownType(fcStr(q.in, "QualificationTypeId"))
	if err != nil {
		return nil, err
	}
	worker := fcStr(q.in, "WorkerId")
	held := st.quals.held[t.id][worker]
	if held == nil || held.status != "Granted" {
		return nil, mtkError("AWS.MechanicalTurk.QualificationDoesNotExist", "worker %s doesn't hold qualification %s", worker, t.id)
	}
	held.status = "Revoked"
	return nil, nil
}

func mtkViewQual(typeID, worker string, q *mtkQual) map[string]any {
	return map[string]any{"QualificationTypeId": typeID, "WorkerId": worker, "GrantTime": mtkEpoch(q.granted), "IntegerValue": q.value, "Status": q.status}
}

func (st *mtkStore) qualScore(q mtkReq) (map[string]any, *mtkErr) {
	qs := &st.quals
	typeID, worker := fcStr(q.in, "QualificationTypeId"), fcStr(q.in, "WorkerId")
	if _, err := qs.qualType(typeID); err != nil {
		return nil, err
	}
	if typeID == mtkQualLocale {
		if w := qs.workers[worker]; w != nil {
			locale := map[string]any{"Country": w.country}
			if w.subdivision != "" {
				locale["Subdivision"] = w.subdivision
			}
			return map[string]any{"Qualification": map[string]any{"QualificationTypeId": typeID, "WorkerId": worker, "LocaleValue": locale, "Status": "Granted"}}, nil
		}
	} else if _, system := mtkSystemQuals[typeID]; system {
		if w := qs.workers[worker]; w != nil {
			if v, ok := qs.value(w, typeID); ok {
				return map[string]any{"Qualification": map[string]any{"QualificationTypeId": typeID, "WorkerId": worker, "IntegerValue": v, "Status": "Granted"}}, nil
			}
		}
	} else if held := qs.held[typeID][worker]; held != nil {
		return map[string]any{"Qualification": mtkViewQual(typeID, worker, held)}, nil
	}
	return nil, mtkError("AWS.MechanicalTurk.QualificationDoesNotExist", "You requested a Qualification that does not exist. (worker %s, type %s)", worker, typeID)
}

func (st *mtkStore) listWorkersWithQual(q mtkReq) (map[string]any, *mtkErr) {
	t, err := st.quals.ownType(fcStr(q.in, "QualificationTypeId"))
	if err != nil {
		return nil, err
	}
	want := fcStr(q.in, "Status")
	held := st.quals.held[t.id]
	workers := make([]string, 0, len(held))
	for w, qual := range held {
		if want == "" || qual.status == want {
			workers = append(workers, w)
		}
	}
	sort.Strings(workers)
	start, end, next, perr := mtkPage(len(workers), q.in)
	if perr != nil {
		return nil, perr
	}
	items := make([]any, 0, end-start)
	for _, w := range workers[start:end] {
		items = append(items, mtkViewQual(t.id, w, held[w]))
	}
	return mtkPaged("Qualifications", items, next), nil
}

// requestQualification is a simulated worker asking for a qualification:
// auto-granted types grant at once, others queue a request for review.
func (st *mtkStore) requestQualification(req mtkWorkerRequest) (map[string]any, *mtkErr) {
	qs := &st.quals
	t, err := qs.ownType(req.QualificationTypeId)
	if err != nil {
		return nil, err
	}
	if t.status != "Active" {
		return nil, mtkError("AWS.MechanicalTurk.QualificationTypeNotActive", "qualification type %s is not active", t.id)
	}
	if !mtkWorkerID(req.WorkerId) {
		return nil, mtkInvalid("WorkerId must be A followed by capital letters and digits")
	}
	if _, werr := qs.ensureWorker(req.WorkerId, req.Country, req.Subdivision); werr != nil {
		return nil, werr
	}
	key := t.id + "/" + req.WorkerId
	if held := qs.held[t.id][req.WorkerId]; held != nil && held.status == "Granted" {
		return nil, mtkError("AWS.MechanicalTurk.QualificationAlreadyExists", "worker %s already holds qualification %s", req.WorkerId, t.id)
	}
	if last, ok := qs.rejected[key]; ok && (!t.hasRetry || st.now().Before(last.Add(time.Duration(t.retryDelay)*time.Second))) {
		return nil, mtkError("AWS.MechanicalTurk.QualificationRetryDelay", "worker %s can't request qualification %s again yet", req.WorkerId, t.id)
	}
	if t.autoGranted {
		if err := qs.grant(t.id, req.WorkerId, t.autoValue, st.now()); err != nil {
			return nil, err
		}
		return map[string]any{"Granted": true}, nil
	}
	if id, open := qs.pending[key]; open {
		return map[string]any{"QualificationRequestId": id}, nil
	}
	if len(qs.requests) >= mtkMaxQualRequests || len(req.Answer) > mtkMaxAnswer {
		return nil, mtkError("AWS.MechanicalTurk.LimitExceeded", "Stackyard keeps at most %d open qualification requests with answers of up to %d characters", mtkMaxQualRequests, mtkMaxAnswer)
	}
	qs.seq++
	r := &mtkQualRequest{id: mtkNewID(), typeID: t.id, worker: req.WorkerId, answer: req.Answer, submitted: st.now(), seq: qs.seq, size: len(req.Answer)}
	if st.bytes+r.size > mtkMaxBytes {
		return nil, mtkError("AWS.MechanicalTurk.LimitExceeded", "stored MTurk data is limited to %d MiB", mtkMaxBytes>>20)
	}
	st.bytes += r.size
	qs.requests[r.id] = r
	qs.pending[key] = r.id
	return map[string]any{"QualificationRequestId": r.id}, nil
}

func (st *mtkStore) listQualRequests(q mtkReq) (map[string]any, *mtkErr) {
	qs := &st.quals
	typeID := fcStr(q.in, "QualificationTypeId")
	if typeID != "" {
		if _, err := qs.ownType(typeID); err != nil {
			return nil, err
		}
	}
	list := make([]*mtkQualRequest, 0, len(qs.requests))
	for _, r := range qs.requests {
		if typeID == "" || r.typeID == typeID {
			list = append(list, r)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].seq < list[j].seq })
	start, end, next, err := mtkPage(len(list), q.in)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, r := range list[start:end] {
		item := map[string]any{"QualificationRequestId": r.id, "QualificationTypeId": r.typeID, "WorkerId": r.worker, "SubmitTime": mtkEpoch(r.submitted)}
		if t := qs.types[r.typeID]; t != nil && t.test != "" {
			item["Test"] = t.test
		}
		if r.answer != "" {
			item["Answer"] = r.answer
		}
		items = append(items, item)
	}
	return mtkPaged("QualificationRequests", items, next), nil
}

// chargeQualType charges a qualification type's text (test, answer key,
// description, keywords) against mtkMaxBytes, replacing old's charge.
func (st *mtkStore) chargeQualType(t, old *mtkQualType) *mtkErr {
	size := len(t.name) + len(t.desc) + len(t.keywords) + len(t.test) + len(t.answerKey)
	prev := 0
	if old != nil {
		prev = old.size
	}
	if st.bytes-prev+size > mtkMaxBytes {
		return mtkError("AWS.MechanicalTurk.LimitExceeded", "stored MTurk data is limited to %d MiB", mtkMaxBytes>>20)
	}
	st.bytes += size - prev
	t.size = size
	return nil
}

func (st *mtkStore) takeRequest(id string) (*mtkQualRequest, *mtkErr) {
	r, err := st.quals.takeRequest(id)
	if err == nil {
		st.bytes -= r.size
	}
	return r, err
}

func (qs *mtkQualState) takeRequest(id string) (*mtkQualRequest, *mtkErr) {
	r := qs.requests[id]
	if r == nil {
		return nil, mtkError("AWS.MechanicalTurk.QualificationRequestDoesNotExist", "Qualification request %s does not exist.", id)
	}
	delete(qs.requests, id)
	delete(qs.pending, r.typeID+"/"+r.worker)
	return r, nil
}

func (st *mtkStore) acceptQualRequest(q mtkReq) (map[string]any, *mtkErr) {
	qs := &st.quals
	r := qs.requests[fcStr(q.in, "QualificationRequestId")]
	if r == nil {
		return nil, mtkError("AWS.MechanicalTurk.QualificationRequestDoesNotExist", "Qualification request %s does not exist.", fcStr(q.in, "QualificationRequestId"))
	}
	if err := qs.grant(r.typeID, r.worker, mtkInt(q.in, "IntegerValue", 1), st.now()); err != nil {
		return nil, err
	}
	_, err := st.takeRequest(r.id)
	return nil, err
}

func (st *mtkStore) rejectQualRequest(q mtkReq) (map[string]any, *mtkErr) {
	qs := &st.quals
	r, err := st.takeRequest(fcStr(q.in, "QualificationRequestId"))
	if err != nil {
		return nil, err
	}
	key := r.typeID + "/" + r.worker
	if _, ok := qs.rejected[key]; !ok && len(qs.rejected) >= mtkMaxQuals {
		qs.rejected = map[string]time.Time{} // ponytail: forget old rejections past the cap
	}
	qs.rejected[key] = st.now()
	return nil, nil
}

// ---- worker blocks and notifications ----

func (st *mtkStore) createBlock(q mtkReq) (map[string]any, *mtkErr) {
	worker, reason := fcStr(q.in, "WorkerId"), fcStr(q.in, "Reason")
	if strings.TrimSpace(reason) == "" || len(reason) > 1024 {
		return nil, mtkInvalid("Reason must have 1-1024 characters")
	}
	if _, ok := st.quals.blocks[worker]; !ok && len(st.quals.blocks) >= mtkMaxBlocks {
		return nil, mtkError("AWS.MechanicalTurk.LimitExceeded", "Stackyard keeps at most %d worker blocks", mtkMaxBlocks)
	}
	st.quals.blocks[worker] = reason
	return nil, nil
}

func (st *mtkStore) deleteBlock(q mtkReq) (map[string]any, *mtkErr) {
	worker := fcStr(q.in, "WorkerId")
	if _, ok := st.quals.blocks[worker]; !ok {
		return nil, mtkError("AWS.MechanicalTurk.WorkerNotBlocked", "worker %s is not blocked", worker)
	}
	delete(st.quals.blocks, worker)
	return nil, nil
}

func (st *mtkStore) listBlocks(q mtkReq) (map[string]any, *mtkErr) {
	workers := make([]string, 0, len(st.quals.blocks))
	for w := range st.quals.blocks {
		workers = append(workers, w)
	}
	sort.Strings(workers)
	start, end, next, err := mtkPage(len(workers), q.in)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, w := range workers[start:end] {
		items = append(items, map[string]any{"WorkerId": w, "Reason": st.quals.blocks[w]})
	}
	return mtkPaged("WorkerBlocks", items, next), nil
}

// notifyWorkers checks each worker can be messaged; Stackyard sends no email.
func (st *mtkStore) notifyWorkers(q mtkReq) (map[string]any, *mtkErr) {
	subject, text, ids := fcStr(q.in, "Subject"), fcStr(q.in, "MessageText"), fcStrs(q.in, "WorkerIds")
	switch {
	case strings.TrimSpace(subject) == "" || len(subject) > 200:
		return nil, mtkInvalid("Subject must have 1-200 characters")
	case strings.TrimSpace(text) == "" || len(text) > 4096:
		return nil, mtkInvalid("MessageText must have 1-4096 characters")
	case len(ids) == 0 || len(ids) > mtkMaxNotify:
		return nil, mtkInvalid("WorkerIds must list 1-%d workers", mtkMaxNotify)
	}
	failures := []any{}
	for _, id := range ids {
		msg := ""
		switch {
		case st.quals.workers[id] == nil:
			msg = "Worker " + id + " has not worked for you and can't be notified."
		case st.quals.blocks[id] != "":
			msg = "Worker " + id + " is blocked."
		}
		if msg != "" {
			failures = append(failures, map[string]any{"NotifyWorkersFailureCode": "HardFailure", "NotifyWorkersFailureMessage": msg, "WorkerId": id})
		}
	}
	return map[string]any{"NotifyWorkersFailureStatuses": failures}, nil
}
