package server

import (
	"net/url"
	"strings"
	"time"
)

// Assignments: simulated workers accept, submit and return them; requesters
// review them, pay bonuses and fetch uploaded files.

func (st *mtkStore) assignment(id string) (*mtkAssign, *mtkErr) {
	a := st.assigns[id]
	if a == nil || a.status == "Accepted" {
		return nil, mtkError("AWS.MechanicalTurk.AssignmentDoesNotExist", "Assignment %s does not exist.", id)
	}
	return a, nil
}

func (st *mtkStore) viewAssignment(a *mtkAssign) map[string]any {
	out := map[string]any{
		"AssignmentId": a.id, "WorkerId": a.worker, "HITId": a.hitID, "AssignmentStatus": a.status,
		"AcceptTime": mtkEpoch(a.accept), "SubmitTime": mtkEpoch(a.submit), "Deadline": mtkEpoch(a.deadline),
		"AutoApprovalTime": mtkEpoch(a.autoApprove), "ApprovalTime": mtkEpoch(a.approval), "RejectionTime": mtkEpoch(a.rejection),
		"Answer": a.answer,
	}
	if a.feedback != "" {
		out["RequesterFeedback"] = a.feedback
	}
	return out
}

func (st *mtkStore) getAssignment(q mtkReq) (map[string]any, *mtkErr) {
	a, err := st.assignment(fcStr(q.in, "AssignmentId"))
	if err != nil {
		return nil, err
	}
	out := map[string]any{"Assignment": st.viewAssignment(a)}
	if h := st.hits[a.hitID]; h != nil {
		out["HIT"] = st.viewHIT(h)
	}
	return out, nil
}

func (st *mtkStore) listAssignments(q mtkReq) (map[string]any, *mtkErr) {
	h, err := st.hit(fcStr(q.in, "HITId"))
	if err != nil {
		return nil, err
	}
	statuses := fcStrs(q.in, "AssignmentStatuses")
	var list []*mtkAssign
	for _, id := range h.assigns {
		a := st.assigns[id]
		if a != nil && a.status != "Accepted" && (len(statuses) == 0 || fcContains(statuses, a.status)) {
			list = append(list, a)
		}
	}
	start, end, next, perr := mtkPage(len(list), q.in)
	if perr != nil {
		return nil, perr
	}
	items := make([]any, 0, end-start)
	for _, a := range list[start:end] {
		items = append(items, st.viewAssignment(a))
	}
	return mtkPaged("Assignments", items, next), nil
}

func (st *mtkStore) approveAssignment(q mtkReq) (map[string]any, *mtkErr) {
	a, err := st.assignment(fcStr(q.in, "AssignmentId"))
	if err != nil {
		return nil, err
	}
	switch {
	case a.status == "Rejected" && fcBool(q.in, "OverrideRejection"):
		if st.now().Sub(a.rejection) > mtkDefaultAutoApp*time.Second {
			return nil, mtkError("AWS.MechanicalTurk.InvalidAssignmentState", "a rejection can only be overridden within 30 days")
		}
	case a.status != "Submitted":
		return nil, mtkError("AWS.MechanicalTurk.InvalidAssignmentState", "This operation can be called with a status of: Submitted (assignment %s is %s)", a.id, a.status)
	}
	if len(fcStr(q.in, "RequesterFeedback")) > 1024 {
		return nil, mtkInvalid("RequesterFeedback can have at most 1024 characters")
	}
	st.review(a, "Approved", fcStr(q.in, "RequesterFeedback"), st.now())
	return nil, nil
}

func (st *mtkStore) rejectAssignment(q mtkReq) (map[string]any, *mtkErr) {
	a, err := st.assignment(fcStr(q.in, "AssignmentId"))
	if err != nil {
		return nil, err
	}
	if a.status != "Submitted" {
		return nil, mtkError("AWS.MechanicalTurk.InvalidAssignmentState", "This operation can be called with a status of: Submitted (assignment %s is %s)", a.id, a.status)
	}
	feedback := fcStr(q.in, "RequesterFeedback")
	if strings.TrimSpace(feedback) == "" || len(feedback) > 1024 {
		return nil, mtkInvalid("RequesterFeedback must have 1-1024 characters")
	}
	st.review(a, "Rejected", feedback, st.now())
	return nil, nil
}

func (st *mtkStore) sendBonus(q mtkReq) (map[string]any, *mtkErr) {
	a, err := st.assignment(fcStr(q.in, "AssignmentId"))
	if err != nil {
		return nil, err
	}
	if a.worker != fcStr(q.in, "WorkerId") {
		return nil, mtkInvalid("assignment %s was not done by worker %s", a.id, fcStr(q.in, "WorkerId"))
	}
	amount := fcStr(q.in, "BonusAmount")
	if err := mtkReward(amount); err != nil {
		return nil, mtkInvalid("BonusAmount must be between 0.01 and 10000.00 USD")
	}
	reason := fcStr(q.in, "Reason")
	if strings.TrimSpace(reason) == "" || len(reason) > 4096 {
		return nil, mtkInvalid("Reason must have 1-4096 characters")
	}
	key, dup := st.token("bonus", q.in)
	if dup {
		return nil, mtkError("AWS.MechanicalTurk.DuplicateRequest", "a bonus with this UniqueRequestToken was already paid")
	}
	if st.bonuses >= mtkMaxBonuses || st.bytes+len(reason) > mtkMaxBytes {
		return nil, mtkError("AWS.MechanicalTurk.LimitExceeded", "Stackyard keeps at most %d bonus payments and %d MiB of MTurk data", mtkMaxBonuses, mtkMaxBytes>>20)
	}
	a.bonuses = append(a.bonuses, map[string]any{
		"WorkerId": a.worker, "BonusAmount": amount, "AssignmentId": a.id, "Reason": reason, "GrantTime": mtkEpoch(st.now()),
	})
	st.bonuses++
	a.bytes += len(reason)
	st.bytes += len(reason)
	st.remember(key, a.id)
	return nil, nil
}

func (st *mtkStore) listBonuses(q mtkReq) (map[string]any, *mtkErr) {
	hitID, assignID := fcStr(q.in, "HITId"), fcStr(q.in, "AssignmentId")
	if (hitID == "") == (assignID == "") {
		return nil, mtkInvalid("set exactly one of HITId and AssignmentId")
	}
	var items []any
	if assignID != "" {
		a, err := st.assignment(assignID)
		if err != nil {
			return nil, err
		}
		for _, b := range a.bonuses {
			items = append(items, b)
		}
	} else {
		h, err := st.hit(hitID)
		if err != nil {
			return nil, err
		}
		for _, id := range h.assigns {
			if a := st.assigns[id]; a != nil {
				for _, b := range a.bonuses {
					items = append(items, b)
				}
			}
		}
	}
	start, end, next, err := mtkPage(len(items), q.in)
	if err != nil {
		return nil, err
	}
	return mtkPaged("BonusPayments", append([]any{}, items[start:end]...), next), nil
}

func (st *mtkStore) fileUploadURL(q mtkReq) (map[string]any, *mtkErr) {
	a, err := st.assignment(fcStr(q.in, "AssignmentId"))
	if err != nil {
		return nil, err
	}
	qid := fcStr(q.in, "QuestionIdentifier")
	if _, ok := a.files[qid]; !ok {
		return nil, mtkInvalid("assignment %s has no file upload answer for question %s", a.id, qid)
	}
	key := strings.ReplaceAll(mtkNewID(), "-", "")
	a.links[qid] = mtkFileLink{key: key, expires: st.now().Add(mtkFileLinkLifetime)}
	return map[string]any{"FileUploadURL": q.base + mtkFilePath + url.PathEscape(a.id) + "/" + url.PathEscape(qid) + "?key=" + key}, nil
}

// ---- simulated workers ----

func (st *mtkStore) worker(action string, req mtkWorkerRequest, files map[string][]byte) (map[string]any, *mtkErr) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.settle()
	switch action {
	case "accept":
		return st.accept(req)
	case "submit":
		return st.submit(req, files)
	case "return":
		a := st.assigns[req.AssignmentId]
		if a == nil || a.status != "Accepted" {
			return nil, mtkError("AWS.MechanicalTurk.AssignmentDoesNotExist", "no accepted assignment %s", req.AssignmentId)
		}
		st.release(a)
		return map[string]any{}, nil
	}
	return st.requestQualification(req)
}

func (st *mtkStore) accept(req mtkWorkerRequest) (map[string]any, *mtkErr) {
	h, err := st.hit(req.HITId)
	if err != nil {
		return nil, err
	}
	if !mtkWorkerID(req.WorkerId) {
		return nil, mtkInvalid("WorkerId must be A followed by capital letters and digits")
	}
	if _, blocked := st.quals.blocks[req.WorkerId]; blocked {
		return nil, mtkError("AWS.MechanicalTurk.WorkerBlocked", "worker %s is blocked by this requester", req.WorkerId)
	}
	if status := st.status(h); status != "Assignable" {
		return nil, mtkError("AWS.MechanicalTurk.HITNotAssignable", "HIT %s is %s", h.id, status)
	}
	if h.workers[req.WorkerId] {
		return nil, mtkError("AWS.MechanicalTurk.AlreadyWorkedOn", "worker %s already worked on HIT %s", req.WorkerId, h.id)
	}
	if len(st.assigns) >= mtkMaxAssignments {
		return nil, mtkError("AWS.MechanicalTurk.LimitExceeded", "Stackyard keeps at most %d assignments", mtkMaxAssignments)
	}
	w, werr := st.quals.ensureWorker(req.WorkerId, req.Country, req.Subdivision)
	if werr != nil {
		return nil, werr
	}
	if t := st.types[h.typeID]; t != nil {
		for _, q := range fcObjList(t.quals) {
			if !st.quals.meets(w, q) {
				return nil, mtkError("AWS.MechanicalTurk.QualificationRequirementNotMet", "worker %s doesn't meet the requirement on qualification type %s", w.id, fcStr(q, "QualificationTypeId"))
			}
		}
	}
	now := st.now()
	dur := int64(mtkMinDuration)
	if t := st.types[h.typeID]; t != nil {
		dur = t.duration
	}
	a := &mtkAssign{
		id: mtkNewID(), hitID: h.id, worker: w.id, status: "Accepted", accept: now,
		deadline: now.Add(time.Duration(dur) * time.Second), files: map[string][]byte{}, links: map[string]mtkFileLink{},
	}
	st.assigns[a.id] = a
	h.assigns = append(h.assigns, a.id)
	h.workers[w.id] = true
	h.pending++
	st.schedule(a.deadline, a.id, mtkTimerDeadline)
	return map[string]any{"AssignmentId": a.id, "Deadline": a.deadline.Format(time.RFC3339)}, nil
}

func (st *mtkStore) submit(req mtkWorkerRequest, files map[string][]byte) (map[string]any, *mtkErr) {
	a := st.assigns[req.AssignmentId]
	if a == nil || a.status != "Accepted" {
		return nil, mtkError("AWS.MechanicalTurk.AssignmentDoesNotExist", "no accepted assignment %s", req.AssignmentId)
	}
	if strings.TrimSpace(req.Answer) == "" || len(req.Answer) > mtkMaxAnswer {
		return nil, mtkInvalid("Answer must be QuestionFormAnswers XML of 1-%d characters", mtkMaxAnswer)
	}
	size := len(req.Answer)
	for qid, data := range files {
		if qid == "" || len(qid) > 256 || len(data) > mtkMaxFileBytes {
			return nil, mtkInvalid("each file needs a question identifier and at most 1 MiB")
		}
		size += len(qid) + len(data)
	}
	if st.bytes+size > mtkMaxBytes {
		return nil, mtkError("AWS.MechanicalTurk.LimitExceeded", "stored MTurk data is limited to %d MiB", mtkMaxBytes>>20)
	}
	h := st.hits[a.hitID]
	auto := int64(mtkDefaultAutoApp)
	if h != nil {
		if t := st.types[h.typeID]; t != nil {
			auto = t.autoApproval
		}
		h.pending--
		h.submitted++
	}
	now := st.now()
	a.status, a.answer, a.submit, a.files, a.bytes = "Submitted", req.Answer, now, files, size
	a.autoApprove = now.Add(time.Duration(auto) * time.Second)
	st.bytes += size
	if w := st.quals.worker(a.worker); w != nil {
		w.submitted++
	}
	st.schedule(a.autoApprove, a.id, mtkTimerAutoApprove)
	return map[string]any{"AssignmentId": a.id, "AutoApprovalTime": a.autoApprove.Format(time.RFC3339)}, nil
}

func mtkWorkerID(id string) bool {
	if len(id) < 2 || len(id) > 64 || id[0] != 'A' {
		return false
	}
	for _, c := range id[1:] {
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
