package server

import (
	"container/heap"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stackyard/stackyard/internal/awsmodels"
)

const (
	mtkMaxHITs          = 1000
	mtkMaxHITTypes      = 1000
	mtkMaxPerHIT        = 100   // MaxAssignments, including additional assignments
	mtkMaxAssignments   = 10000 // stored assignments
	mtkMaxBytes         = 64 << 20
	mtkMaxFileBytes     = 1 << 20
	mtkMaxQuestion      = 128 << 10 // characters of question XML, as MTurk allows
	mtkMaxAnswer        = 128 << 10
	mtkMaxBonuses       = 10000
	mtkMaxTokens        = 10000
	mtkMaxWorkers       = 10000
	mtkMaxQualTypes     = 1000
	mtkMaxQuals         = 50000 // worker qualifications across all types
	mtkMaxQualRequests  = 1000
	mtkMaxBlocks        = 10000
	mtkMaxNotify        = 100 // workers per NotifyWorkers call
	mtkMaxReqs          = 75  // qualification requirements per HIT type
	mtkDefaultPage      = 100
	mtkDefaultAutoApp   = 2592000 // 30 days
	mtkMinDuration      = 30
	mtkMaxDuration      = 31536000
	mtkFileLinkLifetime = time.Minute
	mtkTokenLifetime    = 24 * time.Hour
	mtkSandboxBalance   = "10000.00"
)

type mtkType struct {
	id                     string
	title, desc, keywords  string
	reward                 string
	autoApproval, duration int64
	quals                  []any // conformed QualificationRequirementList
	notify                 map[string]any
	notifyActive           bool
}

type mtkHIT struct {
	id, typeID, layoutID, question, annotation string
	layoutParams                               []any
	assignPolicy, hitPolicy                    map[string]any
	created, expiration                        time.Time
	seq                                        int
	max                                        int
	reviewStatus                               string
	reviewing                                  bool
	pending, submitted, reviewed               int // accepted; submitted; approved or rejected
	assigns                                    []string
	workers                                    map[string]bool // workers who hold or held an assignment
	bytes                                      int
}

type mtkFileLink struct {
	key     string
	expires time.Time
}

type mtkAssign struct {
	id, hitID, worker, status, answer, feedback string
	accept, submit, approval, rejection         time.Time
	deadline, autoApprove                       time.Time
	files                                       map[string][]byte
	links                                       map[string]mtkFileLink
	bonuses                                     []map[string]any
	bytes                                       int
}

type mtkToken struct {
	key, id string
	at      time.Time
}

type mtkStore struct {
	mu       sync.Mutex
	now      func() time.Time
	seq      int
	hits     map[string]*mtkHIT
	types    map[string]*mtkType
	assigns  map[string]*mtkAssign
	timers   mtkTimers
	bytes    int
	bonuses  int
	tokens   map[string]mtkToken
	tokenLog []string
	quals    mtkQualState
}

func newMTurkStore() *mtkStore {
	return &mtkStore{
		now:     func() time.Time { return time.Now().UTC() },
		hits:    map[string]*mtkHIT{},
		types:   map[string]*mtkType{},
		assigns: map[string]*mtkAssign{},
		tokens:  map[string]mtkToken{},
		quals:   newMTurkQualState(),
	}
}

// ---- timers: assignment deadlines and auto-approvals, settled on access ----

const (
	mtkTimerDeadline = iota
	mtkTimerAutoApprove
)

type mtkTimer struct {
	at     time.Time
	assign string
	kind   int
}

type mtkTimers []mtkTimer

func (t mtkTimers) Len() int           { return len(t) }
func (t mtkTimers) Less(i, j int) bool { return t[i].at.Before(t[j].at) }
func (t mtkTimers) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t *mtkTimers) Push(x any)        { *t = append(*t, x.(mtkTimer)) }
func (t *mtkTimers) Pop() any {
	old := *t
	item := old[len(old)-1]
	*t = old[:len(old)-1]
	return item
}

func (st *mtkStore) schedule(at time.Time, assign string, kind int) {
	// Timers of deleted assignments linger until due; rebuild from the live
	// assignments when they outnumber the live ones (amortised O(1)).
	if len(st.timers) > 2*len(st.assigns)+64 {
		st.timers = st.timers[:0]
		for _, a := range st.assigns {
			switch a.status {
			case "Accepted":
				st.timers = append(st.timers, mtkTimer{a.deadline, a.id, mtkTimerDeadline})
			case "Submitted":
				st.timers = append(st.timers, mtkTimer{a.autoApprove, a.id, mtkTimerAutoApprove})
			}
		}
		heap.Init(&st.timers)
	}
	heap.Push(&st.timers, mtkTimer{at, assign, kind})
}

// settle fires every due timer; each assignment fires at most two, once.
func (st *mtkStore) settle() {
	now := st.now()
	for len(st.timers) > 0 && !st.timers[0].at.After(now) {
		t := heap.Pop(&st.timers).(mtkTimer)
		a := st.assigns[t.assign]
		if a == nil {
			continue
		}
		switch {
		case t.kind == mtkTimerDeadline && a.status == "Accepted":
			st.release(a) // abandoned: the slot goes back to the HIT
		case t.kind == mtkTimerAutoApprove && a.status == "Submitted":
			st.review(a, "Approved", "", t.at)
		}
	}
}

// release drops an accepted assignment that was returned or abandoned.
func (st *mtkStore) release(a *mtkAssign) {
	if h := st.hits[a.hitID]; h != nil {
		h.pending--
		for i, id := range h.assigns {
			if id == a.id {
				h.assigns = append(h.assigns[:i:i], h.assigns[i+1:]...)
				break
			}
		}
	}
	st.bytes -= a.bytes
	delete(st.assigns, a.id)
}

// review approves or rejects a submitted (or, overriding, rejected) assignment.
func (st *mtkStore) review(a *mtkAssign, status, feedback string, at time.Time) {
	h := st.hits[a.hitID]
	w := st.quals.worker(a.worker)
	switch a.status {
	case "Submitted":
		if h != nil {
			h.submitted--
			h.reviewed++
		}
	case "Rejected":
		if w != nil {
			w.rejected--
		}
	}
	a.status = status
	if feedback != "" {
		a.feedback = feedback
	}
	if status == "Approved" {
		a.approval = at
		if w != nil {
			w.approved++
		}
	} else {
		a.rejection = at
		if w != nil {
			w.rejected++
		}
	}
}

// ---- dispatch ----

type mtkReq struct {
	in   map[string]any
	base string
}

type mtkHandler func(st *mtkStore, q mtkReq) (map[string]any, *mtkErr)

var mtkHandlers map[string]mtkHandler

func init() {
	mtkHandlers = map[string]mtkHandler{
		"CreateHIT": (*mtkStore).createHIT, "CreateHITType": (*mtkStore).createHITType,
		"CreateHITWithHITType": (*mtkStore).createHITWithType, "GetHIT": (*mtkStore).getHIT,
		"ListHITs": (*mtkStore).listHITs, "ListReviewableHITs": (*mtkStore).listReviewableHITs,
		"ListHITsForQualificationType": (*mtkStore).listHITsForQualType,
		"DeleteHIT":                    (*mtkStore).deleteHIT, "UpdateExpirationForHIT": (*mtkStore).updateExpiration,
		"UpdateHITReviewStatus": (*mtkStore).updateReviewStatus, "UpdateHITTypeOfHIT": (*mtkStore).updateHITType,
		"CreateAdditionalAssignmentsForHIT": (*mtkStore).addAssignments,
		"ListReviewPolicyResultsForHIT":     (*mtkStore).reviewPolicyResults,
		"GetAssignment":                     (*mtkStore).getAssignment, "ListAssignmentsForHIT": (*mtkStore).listAssignments,
		"ApproveAssignment": (*mtkStore).approveAssignment, "RejectAssignment": (*mtkStore).rejectAssignment,
		"SendBonus": (*mtkStore).sendBonus, "ListBonusPayments": (*mtkStore).listBonuses,
		"GetFileUploadURL": (*mtkStore).fileUploadURL,
		"GetAccountBalance": func(*mtkStore, mtkReq) (map[string]any, *mtkErr) {
			return map[string]any{"AvailableBalance": mtkSandboxBalance, "OnHoldBalance": "0.00"}, nil
		},
		"UpdateNotificationSettings": (*mtkStore).updateNotification,
		"SendTestEventNotification":  (*mtkStore).testNotification,
	}
	for name, h := range mtkQualHandlers() {
		mtkHandlers[name] = h
	}
}

// run executes op under the store lock and marshals its conformed output there.
func (st *mtkStore) run(op string, in map[string]any, base string) ([]byte, *mtkErr) {
	h := mtkHandlers[op]
	if h == nil {
		return nil, &mtkErr{code: "UnknownOperationException", msg: "unknown operation " + op}
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	st.settle()
	out, err := h(st, mtkReq{in: in, base: base})
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	model := awsmodels.MTurkModel()
	body, _ := json.Marshal(model.Conform(model.Operations[op].Output, out))
	return body, nil
}

// ---- helpers ----

func mtkNewID() string {
	return strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))[:30]
}

func mtkEpoch(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return float64(t.UnixMilli()) / 1000
}

// mtkInt reads an integer member (the model checked it is integral),
// clamped to the int32 range.
func mtkInt(in map[string]any, key string, def int64) int64 {
	n, ok := fcNum(in, key)
	if !ok {
		return def
	}
	return int64(max(math.MinInt32, min(n, math.MaxInt32)))
}

func mtkTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) || math.Abs(t) > 1e11 {
			return time.Time{}, false
		}
		return time.UnixMilli(int64(t * 1000)).UTC(), true
	case string:
		if p, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return p.UTC(), true
		}
		if n, err := strconv.ParseFloat(t, 64); err == nil {
			return mtkTime(n)
		}
	}
	return time.Time{}, false
}

// mtkPage pages n items by an offset NextToken; MaxResults is bounded 1-100
// by the model.
func mtkPage(n int, in map[string]any) (int, int, string, *mtkErr) {
	start := 0
	if tok := fcStr(in, "NextToken"); tok != "" {
		v, err := strconv.ParseInt(tok, 10, 32)
		if err != nil || v < 0 || int(v) > n {
			return 0, 0, "", mtkError("AWS.MechanicalTurk.InvalidPaginationToken", "NextToken is not valid")
		}
		start = int(v)
	}
	size := int(mtkInt(in, "MaxResults", mtkDefaultPage))
	size = max(1, min(size, mtkDefaultPage))
	end := min(start+size, n)
	next := ""
	if end < n {
		next = strconv.Itoa(end)
	}
	return start, end, next, nil
}

func mtkPaged(key string, items []any, next string) map[string]any {
	out := map[string]any{key: items, "NumResults": len(items)}
	if next != "" {
		out["NextToken"] = next
	}
	return out
}

// token records a UniqueRequestToken (kept 24 hours, at most mtkMaxTokens)
// and reports the resource an earlier request with it created.
func (st *mtkStore) token(kind string, in map[string]any) (string, bool) {
	tok := fcStr(in, "UniqueRequestToken")
	if tok == "" {
		return "", false
	}
	key := kind + "/" + tok
	if t, ok := st.tokens[key]; ok && st.now().Sub(t.at) < mtkTokenLifetime {
		return t.id, true
	}
	return key, false
}

func (st *mtkStore) remember(key, id string) {
	if key == "" {
		return
	}
	if len(st.tokenLog) >= mtkMaxTokens {
		delete(st.tokens, st.tokenLog[0])
		st.tokenLog = append(st.tokenLog[:0:0], st.tokenLog[1:]...)
	}
	if _, ok := st.tokens[key]; !ok {
		st.tokenLog = append(st.tokenLog, key)
	}
	st.tokens[key] = mtkToken{key: key, id: id, at: st.now()}
}

// ---- HIT types ----

func mtkReward(s string) *mtkErr {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v < 0.01 || v > 10000 {
		return mtkInvalid("Reward must be between 0.01 and 10000.00 USD")
	}
	return nil
}

// typeFrom validates HIT type properties and returns the (possibly existing)
// type with those properties; MTurk derives the type ID from them.
func (st *mtkStore) typeFrom(in map[string]any) (*mtkType, *mtkErr) {
	t := &mtkType{
		title: fcStr(in, "Title"), desc: fcStr(in, "Description"), keywords: fcStr(in, "Keywords"), reward: fcStr(in, "Reward"),
		autoApproval: mtkInt(in, "AutoApprovalDelayInSeconds", mtkDefaultAutoApp), duration: mtkInt(in, "AssignmentDurationInSeconds", 0),
	}
	quals, _ := in["QualificationRequirements"].([]any)
	t.quals = quals
	switch {
	case strings.TrimSpace(t.title) == "" || len(t.title) > 128:
		return nil, mtkInvalid("Title must have 1-128 characters")
	case strings.TrimSpace(t.desc) == "" || len(t.desc) > 2000:
		return nil, mtkInvalid("Description must have 1-2000 characters")
	case len(t.keywords) > 1000:
		return nil, mtkInvalid("Keywords can have at most 1000 characters")
	case t.duration < mtkMinDuration || t.duration > mtkMaxDuration:
		return nil, mtkInvalid("AssignmentDurationInSeconds must be between %d and %d", mtkMinDuration, mtkMaxDuration)
	case t.autoApproval < 0 || t.autoApproval > mtkDefaultAutoApp:
		return nil, mtkInvalid("AutoApprovalDelayInSeconds must be between 0 and %d", mtkDefaultAutoApp)
	case len(quals) > mtkMaxReqs:
		return nil, mtkInvalid("a HIT type can have at most %d qualification requirements", mtkMaxReqs)
	}
	if err := mtkReward(t.reward); err != nil {
		return nil, err
	}
	for _, q := range fcObjList(quals) {
		if err := st.quals.checkRequirement(q); err != nil {
			return nil, err
		}
	}
	raw, _ := json.Marshal([]any{t.title, t.desc, t.keywords, t.reward, t.autoApproval, t.duration, t.quals})
	sum := sha1.Sum(raw)
	t.id = strings.ToUpper(hex.EncodeToString(sum[:]))[:30]
	if old := st.types[t.id]; old != nil {
		return old, nil
	}
	if len(st.types) >= mtkMaxHITTypes {
		return nil, mtkError("AWS.MechanicalTurk.LimitExceeded", "Stackyard keeps at most %d HIT types", mtkMaxHITTypes)
	}
	st.types[t.id] = t
	return t, nil
}

func (st *mtkStore) createHITType(q mtkReq) (map[string]any, *mtkErr) {
	t, err := st.typeFrom(q.in)
	if err != nil {
		return nil, err
	}
	return map[string]any{"HITTypeId": t.id}, nil
}

func (st *mtkStore) hitType(id string) (*mtkType, *mtkErr) {
	t := st.types[id]
	if t == nil {
		return nil, mtkError("AWS.MechanicalTurk.HITTypeDoesNotExist", "HIT type %s does not exist", id)
	}
	return t, nil
}

func mtkPolicy(p map[string]any) *mtkErr {
	if p == nil {
		return nil
	}
	switch fcStr(p, "PolicyName") {
	case "ScoreMyKnownAnswers/2011-09-01", "SimplePlurality/2011-09-01":
		return nil
	}
	return mtkInvalid("PolicyName must be ScoreMyKnownAnswers/2011-09-01 or SimplePlurality/2011-09-01")
}

// ---- HITs ----

func (st *mtkStore) createHIT(q mtkReq) (map[string]any, *mtkErr) {
	if q.in["MaxAssignments"] == nil {
		q.in["MaxAssignments"] = float64(1)
	}
	if id, dup := st.token("hit", q.in); dup {
		return nil, mtkError("AWS.MechanicalTurk.HitAlreadyExists", "The HIT with ID %s already exists for this UniqueRequestToken", id)
	}
	if err := st.checkHIT(q.in); err != nil {
		return nil, err
	}
	t, err := st.typeFrom(q.in)
	if err != nil {
		return nil, err
	}
	return st.newHIT(t, q.in)
}

func (st *mtkStore) createHITWithType(q mtkReq) (map[string]any, *mtkErr) {
	if q.in["MaxAssignments"] == nil {
		q.in["MaxAssignments"] = float64(1)
	}
	if id, dup := st.token("hit", q.in); dup {
		return nil, mtkError("AWS.MechanicalTurk.HitAlreadyExists", "The HIT with ID %s already exists for this UniqueRequestToken", id)
	}
	t, err := st.hitType(fcStr(q.in, "HITTypeId"))
	if err != nil {
		return nil, err
	}
	if err := st.checkHIT(q.in); err != nil {
		return nil, err
	}
	return st.newHIT(t, q.in)
}

func (st *mtkStore) checkHIT(in map[string]any) *mtkErr {
	question, layout := fcStr(in, "Question"), fcStr(in, "HITLayoutId")
	switch {
	case (question == "") == (layout == ""):
		return mtkInvalid("set exactly one of Question and HITLayoutId")
	case len(question) > mtkMaxQuestion:
		return mtkInvalid("Question can have at most %d characters", mtkMaxQuestion)
	case question != "" && !strings.Contains(question, "<QuestionForm") && !strings.Contains(question, "<ExternalQuestion") && !strings.Contains(question, "<HTMLQuestion"):
		return mtkError("AWS.MechanicalTurk.XMLParseError", "Question must be a QuestionForm, ExternalQuestion or HTMLQuestion document")
	case len(fcStr(in, "RequesterAnnotation")) > 255:
		return mtkInvalid("RequesterAnnotation can have at most 255 characters")
	}
	life := mtkInt(in, "LifetimeInSeconds", 0)
	if life < mtkMinDuration || life > mtkMaxDuration {
		return mtkInvalid("LifetimeInSeconds must be between %d and %d", mtkMinDuration, mtkMaxDuration)
	}
	if n := mtkInt(in, "MaxAssignments", 1); n < 1 || n > mtkMaxPerHIT {
		return mtkInvalid("MaxAssignments must be between 1 and %d", mtkMaxPerHIT)
	}
	if err := mtkPolicy(fcObj(in, "AssignmentReviewPolicy")); err != nil {
		return err
	}
	return mtkPolicy(fcObj(in, "HITReviewPolicy"))
}

func (st *mtkStore) newHIT(t *mtkType, in map[string]any) (map[string]any, *mtkErr) {
	if len(st.hits) >= mtkMaxHITs {
		return nil, mtkError("AWS.MechanicalTurk.LimitExceeded", "Stackyard keeps at most %d HITs", mtkMaxHITs)
	}
	size := len(fcStr(in, "Question")) + len(fcStr(in, "RequesterAnnotation"))
	if st.bytes+size > mtkMaxBytes {
		return nil, mtkError("AWS.MechanicalTurk.LimitExceeded", "stored MTurk data is limited to %d MiB", mtkMaxBytes>>20)
	}
	now := st.now()
	st.seq++
	params, _ := in["HITLayoutParameters"].([]any)
	h := &mtkHIT{
		id: mtkNewID(), typeID: t.id, layoutID: fcStr(in, "HITLayoutId"), question: fcStr(in, "Question"),
		annotation: fcStr(in, "RequesterAnnotation"), layoutParams: params,
		assignPolicy: fcObj(in, "AssignmentReviewPolicy"), hitPolicy: fcObj(in, "HITReviewPolicy"),
		created: now, expiration: now.Add(time.Duration(mtkInt(in, "LifetimeInSeconds", 0)) * time.Second),
		seq: st.seq, max: int(mtkInt(in, "MaxAssignments", 1)), reviewStatus: "NotReviewed",
		workers: map[string]bool{}, bytes: size,
	}
	st.hits[h.id] = h
	st.bytes += size
	st.quals.indexHIT(h.id, nil, t.quals)
	if key, _ := st.token("hit", in); key != "" {
		st.remember(key, h.id)
	}
	return map[string]any{"HIT": st.viewHIT(h)}, nil
}

// status derives the HIT status from its expiration and assignment counts.
func (st *mtkStore) status(h *mtkHIT) string {
	expired := !st.now().Before(h.expiration)
	completed := h.submitted + h.reviewed
	switch {
	case completed >= h.max || expired && h.pending == 0:
		if h.reviewing {
			return "Reviewing"
		}
		return "Reviewable"
	case expired || h.pending+completed >= h.max:
		return "Unassignable"
	}
	return "Assignable"
}

func (st *mtkStore) viewHIT(h *mtkHIT) map[string]any {
	t := st.types[h.typeID]
	out := map[string]any{
		"HITId": h.id, "HITTypeId": h.typeID, "HITGroupId": "G" + h.typeID[1:], "CreationTime": mtkEpoch(h.created),
		"Expiration": mtkEpoch(h.expiration), "HITStatus": st.status(h), "MaxAssignments": h.max,
		"HITReviewStatus": h.reviewStatus, "NumberOfAssignmentsPending": h.pending,
		"NumberOfAssignmentsAvailable": max(0, h.max-h.pending-h.submitted-h.reviewed),
		"NumberOfAssignmentsCompleted": h.submitted + h.reviewed,
	}
	if t != nil {
		out["Title"], out["Description"], out["Reward"] = t.title, t.desc, t.reward
		out["AutoApprovalDelayInSeconds"], out["AssignmentDurationInSeconds"] = t.autoApproval, t.duration
		if t.keywords != "" {
			out["Keywords"] = t.keywords
		}
		if t.quals != nil {
			out["QualificationRequirements"] = t.quals
		}
	}
	if h.question != "" {
		out["Question"] = h.question
	}
	if h.layoutID != "" {
		out["HITLayoutId"] = h.layoutID
	}
	if h.annotation != "" {
		out["RequesterAnnotation"] = h.annotation
	}
	return out
}

func (st *mtkStore) hit(id string) (*mtkHIT, *mtkErr) {
	h := st.hits[id]
	if h == nil {
		return nil, mtkError("AWS.MechanicalTurk.HITDoesNotExist", "Hit %s does not exist.", id)
	}
	return h, nil
}

func (st *mtkStore) getHIT(q mtkReq) (map[string]any, *mtkErr) {
	h, err := st.hit(fcStr(q.in, "HITId"))
	if err != nil {
		return nil, err
	}
	return map[string]any{"HIT": st.viewHIT(h)}, nil
}

func (st *mtkStore) sortedHITs(keep func(*mtkHIT) bool) []*mtkHIT {
	list := make([]*mtkHIT, 0, len(st.hits))
	for _, h := range st.hits {
		if keep == nil || keep(h) {
			list = append(list, h)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].seq < list[j].seq })
	return list
}

func (st *mtkStore) pageHITs(list []*mtkHIT, in map[string]any) (map[string]any, *mtkErr) {
	start, end, next, err := mtkPage(len(list), in)
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, h := range list[start:end] {
		items = append(items, st.viewHIT(h))
	}
	return mtkPaged("HITs", items, next), nil
}

func (st *mtkStore) listHITs(q mtkReq) (map[string]any, *mtkErr) {
	return st.pageHITs(st.sortedHITs(nil), q.in)
}

func (st *mtkStore) listReviewableHITs(q mtkReq) (map[string]any, *mtkErr) {
	want := fcStr(q.in, "Status")
	if want == "" {
		want = "Reviewable"
	}
	typeID := fcStr(q.in, "HITTypeId")
	return st.pageHITs(st.sortedHITs(func(h *mtkHIT) bool {
		return (typeID == "" || h.typeID == typeID) && st.status(h) == want
	}), q.in)
}

func (st *mtkStore) listHITsForQualType(q mtkReq) (map[string]any, *mtkErr) {
	id := fcStr(q.in, "QualificationTypeId")
	if _, err := st.quals.qualType(id); err != nil {
		return nil, err
	}
	ids := st.quals.hitsByQual[id]
	list := make([]*mtkHIT, 0, len(ids))
	for hid := range ids {
		if h := st.hits[hid]; h != nil {
			list = append(list, h)
		}
	}
	sort.Slice(list, func(i, j int) bool { return list[i].seq < list[j].seq })
	return st.pageHITs(list, q.in)
}

func (st *mtkStore) deleteHIT(q mtkReq) (map[string]any, *mtkErr) {
	h, err := st.hit(fcStr(q.in, "HITId"))
	if err != nil {
		return nil, err
	}
	status := st.status(h)
	if status != "Reviewable" && status != "Reviewing" || h.submitted > 0 {
		return nil, mtkError("AWS.MechanicalTurk.InvalidHITState", "This HIT is currently in the state '%s'. This operation can be called with a status of: Reviewable, with every submitted assignment approved or rejected.", status)
	}
	for _, id := range h.assigns {
		if a := st.assigns[id]; a != nil {
			st.bonuses -= len(a.bonuses)
			st.bytes -= a.bytes
			delete(st.assigns, id)
		}
	}
	if t := st.types[h.typeID]; t != nil {
		st.quals.indexHIT(h.id, t.quals, nil)
	}
	st.bytes -= h.bytes
	delete(st.hits, h.id)
	return nil, nil
}

func (st *mtkStore) updateExpiration(q mtkReq) (map[string]any, *mtkErr) {
	h, err := st.hit(fcStr(q.in, "HITId"))
	if err != nil {
		return nil, err
	}
	at, ok := mtkTime(q.in["ExpireAt"])
	if !ok {
		return nil, mtkInvalid("ExpireAt is not a valid timestamp")
	}
	now := st.now()
	if at.Before(now) {
		at = now // a time in the past expires the HIT immediately
	}
	if at.After(h.created.Add(mtkMaxDuration * time.Second)) {
		return nil, mtkInvalid("a HIT can't stay open more than a year after it was created")
	}
	h.expiration = at
	return nil, nil
}

func (st *mtkStore) updateReviewStatus(q mtkReq) (map[string]any, *mtkErr) {
	h, err := st.hit(fcStr(q.in, "HITId"))
	if err != nil {
		return nil, err
	}
	revert := fcBool(q.in, "Revert")
	from := "Reviewable"
	if revert {
		from = "Reviewing"
	}
	if status := st.status(h); status != from {
		return nil, mtkError("AWS.MechanicalTurk.InvalidHITState", "This HIT is currently in the state '%s'. This operation can be called with a status of: %s", status, from)
	}
	h.reviewing = !revert
	return nil, nil
}

func (st *mtkStore) updateHITType(q mtkReq) (map[string]any, *mtkErr) {
	h, err := st.hit(fcStr(q.in, "HITId"))
	if err != nil {
		return nil, err
	}
	t, err := st.hitType(fcStr(q.in, "HITTypeId"))
	if err != nil {
		return nil, err
	}
	if old := st.types[h.typeID]; old != nil {
		st.quals.indexHIT(h.id, old.quals, t.quals)
	}
	h.typeID = t.id
	return nil, nil
}

func (st *mtkStore) addAssignments(q mtkReq) (map[string]any, *mtkErr) {
	h, err := st.hit(fcStr(q.in, "HITId"))
	if err != nil {
		return nil, err
	}
	if _, dup := st.token("extend/"+h.id, q.in); dup {
		return nil, mtkError("AWS.MechanicalTurk.DuplicateRequest", "This request with UniqueRequestToken was already processed")
	}
	n := mtkInt(q.in, "NumberOfAdditionalAssignments", 0)
	if n < 1 || int64(h.max)+n > mtkMaxPerHIT {
		return nil, mtkInvalid("NumberOfAdditionalAssignments must be at least 1, and a HIT can have at most %d assignments", mtkMaxPerHIT)
	}
	if st.status(h) == "Reviewing" {
		return nil, mtkError("AWS.MechanicalTurk.InvalidHITState", "a HIT in the Reviewing state can't be extended")
	}
	h.max += int(n)
	if key, _ := st.token("extend/"+h.id, q.in); key != "" {
		st.remember(key, h.id)
	}
	return nil, nil
}

func (st *mtkStore) reviewPolicyResults(q mtkReq) (map[string]any, *mtkErr) {
	h, err := st.hit(fcStr(q.in, "HITId"))
	if err != nil {
		return nil, err
	}
	levels := fcStrs(q.in, "PolicyLevels")
	want := func(level string) bool { return len(levels) == 0 || fcContains(levels, level) }
	out := map[string]any{"HITId": h.id}
	empty := map[string]any{"ReviewResults": []any{}, "ReviewActions": []any{}}
	if want("Assignment") && h.assignPolicy != nil {
		out["AssignmentReviewPolicy"], out["AssignmentReviewReport"] = h.assignPolicy, empty
	}
	if want("HIT") && h.hitPolicy != nil {
		out["HITReviewPolicy"], out["HITReviewReport"] = h.hitPolicy, empty
	}
	return out, nil
}

// ---- notifications ----

func mtkCheckNotification(n map[string]any) *mtkErr {
	dest := fcStr(n, "Destination")
	if fcStr(n, "Version") != "2006-05-05" {
		return mtkInvalid("Notification.Version must be 2006-05-05")
	}
	ok := false
	switch fcStr(n, "Transport") {
	case "Email":
		ok = strings.Contains(dest, "@")
	case "SQS":
		ok = strings.HasPrefix(dest, "https://") && strings.Contains(dest, "sqs")
	case "SNS":
		ok = strings.HasPrefix(dest, "arn:aws:sns:")
	}
	if !ok {
		return mtkInvalid("Notification.Destination %q doesn't match its %s transport", dest, fcStr(n, "Transport"))
	}
	if len(fcStrs(n, "EventTypes")) == 0 {
		return mtkInvalid("Notification.EventTypes needs at least one event type")
	}
	return nil
}

func (st *mtkStore) updateNotification(q mtkReq) (map[string]any, *mtkErr) {
	t, err := st.hitType(fcStr(q.in, "HITTypeId"))
	if err != nil {
		return nil, err
	}
	n := fcObj(q.in, "Notification")
	if n != nil {
		if err := mtkCheckNotification(n); err != nil {
			return nil, err
		}
	}
	active, hasActive := q.in["Active"].(bool)
	if n == nil && t.notify == nil {
		return nil, mtkInvalid("this HIT type has no notification specification yet; set Notification")
	}
	if n != nil {
		t.notify = n
		t.notifyActive = true
	}
	if hasActive {
		t.notifyActive = active
	}
	return nil, nil
}

// testNotification validates the specification; Stackyard delivers nothing.
func (st *mtkStore) testNotification(q mtkReq) (map[string]any, *mtkErr) {
	return nil, mtkCheckNotification(fcObj(q.in, "Notification"))
}
