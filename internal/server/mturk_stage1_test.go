package server

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

const mtkTestQuestion = `<HTMLQuestion xmlns="http://mechanicalturk.amazonaws.com/AWSMechanicalTurkDataSchemas/2011-11-11/HTMLQuestion.xsd"><HTMLContent><![CDATA[<p>Tag it</p>]]></HTMLContent><FrameHeight>400</FrameHeight></HTMLQuestion>`

func mtkCreateHIT(t *testing.T, srvCall func(string, string) map[string]any, max int, extra string) string {
	t.Helper()
	out := srvCall("CreateHIT", fmt.Sprintf(`{"MaxAssignments":%d,"LifetimeInSeconds":3600,"AssignmentDurationInSeconds":600,"Reward":"0.25","Title":"Tag a photo","Description":"Tag the animal","Question":%q%s}`, max, mtkTestQuestion, extra))
	hit, _ := out["HIT"].(map[string]any)
	id, _ := hit["HITId"].(string)
	if id == "" || hit["HITStatus"] != "Assignable" {
		t.Fatalf("CreateHIT: %v", out)
	}
	return id
}

func TestMTurkHITLifecycle(t *testing.T) {
	srv, ts := newMessagingTestServer(t)
	call := func(op, body string) map[string]any { return mtkOK(t, ts, op, body) }
	hit := mtkCreateHIT(t, call, 2, "")
	status := func() string {
		h, _ := call("GetHIT", `{"HITId":"`+hit+`"}`)["HIT"].(map[string]any)
		s, _ := h["HITStatus"].(string)
		return s
	}
	// Two workers take both slots: Unassignable.
	_, a1 := mtkWorkerDo(t, ts, "accept", map[string]any{"HITId": hit, "WorkerId": "AWORKER1"})
	_, a2 := mtkWorkerDo(t, ts, "accept", map[string]any{"HITId": hit, "WorkerId": "AWORKER2"})
	if status() != "Unassignable" {
		t.Fatalf("after two accepts: %s", status())
	}
	if code, out := mtkWorkerDo(t, ts, "accept", map[string]any{"HITId": hit, "WorkerId": "AWORKER3"}); code != http.StatusBadRequest || out["TurkErrorCode"] != "AWS.MechanicalTurk.HITNotAssignable" {
		t.Fatalf("third accept: %d %v", code, out)
	}
	// One returns: Assignable again; the same worker can't take it twice.
	if code, _ := mtkWorkerDo(t, ts, "return", map[string]any{"AssignmentId": a2["AssignmentId"]}); code != http.StatusOK {
		t.Fatalf("return: %d", code)
	}
	if status() != "Assignable" {
		t.Fatalf("after return: %s", status())
	}
	if code, out := mtkWorkerDo(t, ts, "accept", map[string]any{"HITId": hit, "WorkerId": "AWORKER1"}); code != http.StatusBadRequest || out["TurkErrorCode"] != "AWS.MechanicalTurk.AlreadyWorkedOn" {
		t.Fatalf("repeat accept: %d %v", code, out)
	}
	_, a3 := mtkWorkerDo(t, ts, "accept", map[string]any{"HITId": hit, "WorkerId": "AWORKER3"})
	file := base64.StdEncoding.EncodeToString([]byte("<script>alert(1)</script>"))
	for _, a := range []map[string]any{a1, a3} {
		if code, out := mtkWorkerDo(t, ts, "submit", map[string]any{"AssignmentId": a["AssignmentId"], "Answer": "<QuestionFormAnswers/>", "Files": map[string]string{"photo": file}}); code != http.StatusOK {
			t.Fatalf("submit: %d %v", code, out)
		}
	}
	if status() != "Reviewable" {
		t.Fatalf("after submits: %s", status())
	}
	listed := call("ListAssignmentsForHIT", `{"HITId":"`+hit+`","AssignmentStatuses":["Submitted"]}`)
	if listed["NumResults"] != float64(2) {
		t.Fatalf("ListAssignmentsForHIT: %v", listed)
	}
	if code := mtkFail(t, ts, "DeleteHIT", `{"HITId":"`+hit+`"}`); code != "AWS.MechanicalTurk.InvalidHITState" {
		t.Fatalf("DeleteHIT with unreviewed work: %s", code)
	}
	id1, id3 := a1["AssignmentId"].(string), a3["AssignmentId"].(string)
	call("ApproveAssignment", `{"AssignmentId":"`+id1+`","RequesterFeedback":"Thanks"}`)
	if code := mtkFail(t, ts, "ApproveAssignment", `{"AssignmentId":"`+id1+`"}`); code != "AWS.MechanicalTurk.InvalidAssignmentState" {
		t.Fatalf("double approve: %s", code)
	}
	if code := mtkFail(t, ts, "RejectAssignment", `{"AssignmentId":"`+id3+`","RequesterFeedback":""}`); code == "" {
		t.Fatal("reject without feedback succeeded")
	}
	call("RejectAssignment", `{"AssignmentId":"`+id3+`","RequesterFeedback":"Wrong animal"}`)
	call("ApproveAssignment", `{"AssignmentId":"`+id3+`","OverrideRejection":true}`)
	got, _ := call("GetAssignment", `{"AssignmentId":"`+id3+`"}`)["Assignment"].(map[string]any)
	if got["AssignmentStatus"] != "Approved" || got["RejectionTime"] == nil {
		t.Fatalf("override: %v", got)
	}
	// Bonuses, and the uploaded file through its short-lived URL.
	call("SendBonus", `{"WorkerId":"AWORKER1","BonusAmount":"1.50","AssignmentId":"`+id1+`","Reason":"Great","UniqueRequestToken":"b1"}`)
	if code := mtkFail(t, ts, "SendBonus", `{"WorkerId":"AWORKER1","BonusAmount":"1.50","AssignmentId":"`+id1+`","Reason":"Great","UniqueRequestToken":"b1"}`); code != "AWS.MechanicalTurk.DuplicateRequest" {
		t.Fatalf("duplicate bonus: %s", code)
	}
	if b := call("ListBonusPayments", `{"HITId":"`+hit+`"}`); b["NumResults"] != float64(1) {
		t.Fatalf("ListBonusPayments: %v", b)
	}
	link, _ := call("GetFileUploadURL", `{"AssignmentId":"`+id1+`","QuestionIdentifier":"photo"}`)["FileUploadURL"].(string)
	resp, err := http.Get(link)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(body) != "<script>alert(1)</script>" || resp.Header.Get("Content-Type") != "application/octet-stream" || resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("file download: %d %q %v", resp.StatusCode, body, resp.Header)
	}
	if bad, _ := http.Get(link + "x"); bad.StatusCode != http.StatusForbidden {
		t.Fatalf("bad key: %d", bad.StatusCode)
	}
	// Reviewing round trip, then dispose.
	call("UpdateHITReviewStatus", `{"HITId":"`+hit+`"}`)
	if status() != "Reviewing" {
		t.Fatalf("UpdateHITReviewStatus: %s", status())
	}
	call("UpdateHITReviewStatus", `{"HITId":"`+hit+`","Revert":true}`)
	call("DeleteHIT", `{"HITId":"`+hit+`"}`)
	if code := mtkFail(t, ts, "GetHIT", `{"HITId":"`+hit+`"}`); code != "AWS.MechanicalTurk.HITDoesNotExist" {
		t.Fatalf("GetHIT after delete: %s", code)
	}
	srv.mturk.mu.Lock()
	bytes, bonuses := srv.mturk.bytes, srv.mturk.bonuses
	srv.mturk.mu.Unlock()
	if bytes != 0 || bonuses != 0 {
		t.Fatalf("storage not refunded: %d bytes, %d bonuses", bytes, bonuses)
	}
}

func TestMTurkTimersAndExpiry(t *testing.T) {
	srv, ts := newMessagingTestServer(t)
	call := func(op, body string) map[string]any { return mtkOK(t, ts, op, body) }
	base := time.Now().UTC()
	var offset atomic.Int64
	srv.mturk.mu.Lock()
	srv.mturk.now = func() time.Time { return base.Add(time.Duration(offset.Load())) }
	srv.mturk.mu.Unlock()
	hit := mtkCreateHIT(t, call, 1, `,"AutoApprovalDelayInSeconds":60`)
	_, a := mtkWorkerDo(t, ts, "accept", map[string]any{"HITId": hit, "WorkerId": "AWORKER1"})
	offset.Add(int64(601 * time.Second)) // past the assignment duration: abandoned
	h, _ := call("GetHIT", `{"HITId":"`+hit+`"}`)["HIT"].(map[string]any)
	if h["HITStatus"] != "Assignable" || h["NumberOfAssignmentsPending"] != float64(0) {
		t.Fatalf("after deadline: %v", h)
	}
	if code, _ := mtkWorkerDo(t, ts, "submit", map[string]any{"AssignmentId": a["AssignmentId"], "Answer": "<x/>"}); code != http.StatusBadRequest {
		t.Fatalf("submit after deadline: %d", code)
	}
	_, a = mtkWorkerDo(t, ts, "accept", map[string]any{"HITId": hit, "WorkerId": "AWORKER2"})
	mtkWorkerDo(t, ts, "submit", map[string]any{"AssignmentId": a["AssignmentId"], "Answer": "<x/>"})
	offset.Add(int64(61 * time.Second)) // auto-approval
	got, _ := call("GetAssignment", `{"AssignmentId":"`+a["AssignmentId"].(string)+`"}`)["Assignment"].(map[string]any)
	if got["AssignmentStatus"] != "Approved" {
		t.Fatalf("auto-approval: %v", got)
	}
	// Expiring a HIT with nothing pending makes it Reviewable at once.
	other := mtkCreateHIT(t, call, 3, "")
	call("UpdateExpirationForHIT", `{"HITId":"`+other+`","ExpireAt":0}`)
	if r := call("ListReviewableHITs", `{}`); r["NumResults"] != float64(2) {
		t.Fatalf("ListReviewableHITs: %v", r)
	}
	call("CreateAdditionalAssignmentsForHIT", `{"HITId":"`+hit+`","NumberOfAdditionalAssignments":2,"UniqueRequestToken":"x1"}`)
	if code := mtkFail(t, ts, "CreateAdditionalAssignmentsForHIT", `{"HITId":"`+hit+`","NumberOfAdditionalAssignments":2,"UniqueRequestToken":"x1"}`); code != "AWS.MechanicalTurk.DuplicateRequest" {
		t.Fatalf("duplicate extend: %s", code)
	}
	if code := mtkFail(t, ts, "CreateAdditionalAssignmentsForHIT", `{"HITId":"`+hit+`","NumberOfAdditionalAssignments":1000}`); code != "AWS.ParameterOutOfRange" {
		t.Fatalf("extend past cap: %s", code)
	}
}

func TestMTurkQualifications(t *testing.T) {
	_, ts := newMessagingTestServer(t)
	call := func(op, body string) map[string]any { return mtkOK(t, ts, op, body) }
	qt, _ := call("CreateQualificationType", `{"Name":"Photo skill","Description":"Can tag photos","QualificationTypeStatus":"Active","RetryDelayInSeconds":3600}`)["QualificationType"].(map[string]any)
	qid := qt["QualificationTypeId"].(string)
	if code := mtkFail(t, ts, "CreateQualificationType", `{"Name":"photo SKILL","Description":"dup","QualificationTypeStatus":"Active"}`); code != "AWS.MechanicalTurk.QualificationTypeAlreadyExists" {
		t.Fatalf("duplicate name: %s", code)
	}
	req := `,"QualificationRequirements":[{"QualificationTypeId":"` + qid + `","Comparator":"GreaterThanOrEqualTo","IntegerValues":[80]},{"QualificationTypeId":"00000000000000000071","Comparator":"EqualTo","LocaleValues":[{"Country":"US"}]}]`
	hit := mtkCreateHIT(t, call, 5, req)
	if l := call("ListHITsForQualificationType", `{"QualificationTypeId":"`+qid+`"}`); l["NumResults"] != float64(1) {
		t.Fatalf("ListHITsForQualificationType: %v", l)
	}
	if code, out := mtkWorkerDo(t, ts, "accept", map[string]any{"HITId": hit, "WorkerId": "AWORKER1"}); code != http.StatusBadRequest || out["TurkErrorCode"] != "AWS.MechanicalTurk.QualificationRequirementNotMet" {
		t.Fatalf("unqualified accept: %d %v", code, out)
	}
	// The worker asks, the requester rejects, the retry delay applies.
	_, r := mtkWorkerDo(t, ts, "request-qualification", map[string]any{"QualificationTypeId": qid, "WorkerId": "AWORKER1", "Answer": "<a/>"})
	call("RejectQualificationRequest", `{"QualificationRequestId":"`+r["QualificationRequestId"].(string)+`","Reason":"no"}`)
	if code, out := mtkWorkerDo(t, ts, "request-qualification", map[string]any{"QualificationTypeId": qid, "WorkerId": "AWORKER1"}); code != http.StatusBadRequest || out["TurkErrorCode"] != "AWS.MechanicalTurk.QualificationRetryDelay" {
		t.Fatalf("retry too soon: %d %v", code, out)
	}
	_, r = mtkWorkerDo(t, ts, "request-qualification", map[string]any{"QualificationTypeId": qid, "WorkerId": "AWORKER2"})
	if l := call("ListQualificationRequests", `{"QualificationTypeId":"`+qid+`"}`); l["NumResults"] != float64(1) {
		t.Fatalf("ListQualificationRequests: %v", l)
	}
	call("AcceptQualificationRequest", `{"QualificationRequestId":"`+r["QualificationRequestId"].(string)+`","IntegerValue":90}`)
	if code, out := mtkWorkerDo(t, ts, "accept", map[string]any{"HITId": hit, "WorkerId": "AWORKER2"}); code != http.StatusOK {
		t.Fatalf("qualified accept: %d %v", code, out)
	}
	if code, _ := mtkWorkerDo(t, ts, "accept", map[string]any{"HITId": hit, "WorkerId": "AWORKER9", "Country": "DE"}); code != http.StatusBadRequest {
		t.Fatal("locale requirement not enforced")
	}
	call("AssociateQualificationWithWorker", `{"QualificationTypeId":"`+qid+`","WorkerId":"AWORKER1","IntegerValue":50}`)
	score, _ := call("GetQualificationScore", `{"QualificationTypeId":"`+qid+`","WorkerId":"AWORKER1"}`)["Qualification"].(map[string]any)
	if score["IntegerValue"] != float64(50) || score["Status"] != "Granted" {
		t.Fatalf("GetQualificationScore: %v", score)
	}
	call("DisassociateQualificationFromWorker", `{"QualificationTypeId":"`+qid+`","WorkerId":"AWORKER1"}`)
	if l := call("ListWorkersWithQualificationType", `{"QualificationTypeId":"`+qid+`","Status":"Revoked"}`); l["NumResults"] != float64(1) {
		t.Fatalf("revoked list: %v", l)
	}
	if code := mtkFail(t, ts, "AssociateQualificationWithWorker", `{"QualificationTypeId":"00000000000000000071","WorkerId":"AWORKER1"}`); code != "AWS.MechanicalTurk.PermissionDenied" {
		t.Fatalf("system qualification: %s", code)
	}
	// Blocks stop accepts; notifications report workers who never worked.
	call("CreateWorkerBlock", `{"WorkerId":"AWORKER2","Reason":"spam"}`)
	if l := call("ListWorkerBlocks", `{}`); l["NumResults"] != float64(1) {
		t.Fatalf("ListWorkerBlocks: %v", l)
	}
	n := call("NotifyWorkers", `{"Subject":"Hi","MessageText":"New HITs","WorkerIds":["AWORKER1","ANEVERSEEN"]}`)
	if fails, _ := n["NotifyWorkersFailureStatuses"].([]any); len(fails) != 1 {
		t.Fatalf("NotifyWorkers: %v", n)
	}
	call("DeleteWorkerBlock", `{"WorkerId":"AWORKER2"}`)
	if code := mtkFail(t, ts, "DeleteWorkerBlock", `{"WorkerId":"AWORKER2"}`); code != "AWS.MechanicalTurk.WorkerNotBlocked" {
		t.Fatalf("unblock twice: %s", code)
	}
	call("DeleteQualificationType", `{"QualificationTypeId":"`+qid+`"}`)
	if code := mtkFail(t, ts, "GetQualificationType", `{"QualificationTypeId":"`+qid+`"}`); code != "AWS.MechanicalTurk.QualificationTypeDoesNotExist" {
		t.Fatalf("after delete: %s", code)
	}
}

func TestMTurkTypesAndNotifications(t *testing.T) {
	_, ts := newMessagingTestServer(t)
	call := func(op, body string) map[string]any { return mtkOK(t, ts, op, body) }
	props := `"AssignmentDurationInSeconds":300,"Reward":"0.10","Title":"Transcribe","Description":"Type the text"`
	t1 := call("CreateHITType", `{`+props+`}`)["HITTypeId"]
	if t2 := call("CreateHITType", `{`+props+`}`)["HITTypeId"]; t1 != t2 {
		t.Fatalf("same properties gave %v and %v", t1, t2)
	}
	out := call("CreateHITWithHITType", fmt.Sprintf(`{"HITTypeId":%q,"LifetimeInSeconds":600,"Question":%q,"UniqueRequestToken":"tok-1"}`, t1, mtkTestQuestion))
	if code := mtkFail(t, ts, "CreateHITWithHITType", fmt.Sprintf(`{"HITTypeId":%q,"LifetimeInSeconds":600,"Question":%q,"UniqueRequestToken":"tok-1"}`, t1, mtkTestQuestion)); code != "AWS.MechanicalTurk.HitAlreadyExists" {
		t.Fatalf("duplicate token: %s", code)
	}
	if code := mtkFail(t, ts, "CreateHIT", `{"LifetimeInSeconds":600,`+props+`,"Question":"<p>not xml</p>"}`); code != "AWS.MechanicalTurk.XMLParseError" {
		t.Fatalf("bad question: %s", code)
	}
	if code := mtkFail(t, ts, "CreateHIT", `{"LifetimeInSeconds":600,`+props+`,"Reward":"abc","Question":"<QuestionForm/>"}`); code == "" {
		t.Fatal("bad reward accepted")
	}
	call("UpdateNotificationSettings", fmt.Sprintf(`{"HITTypeId":%q,"Notification":{"Destination":"arn:aws:sns:us-east-1:123456789012:turk","Transport":"SNS","Version":"2006-05-05","EventTypes":["AssignmentSubmitted"]},"Active":true}`, t1))
	if code := mtkFail(t, ts, "SendTestEventNotification", `{"Notification":{"Destination":"not-a-queue","Transport":"SQS","Version":"2006-05-05","EventTypes":["Ping"]},"TestEventType":"Ping"}`); code != "AWS.ParameterOutOfRange" {
		t.Fatalf("bad destination: %s", code)
	}
	hit, _ := out["HIT"].(map[string]any)
	p := call("ListReviewPolicyResultsForHIT", `{"HITId":"`+hit["HITId"].(string)+`"}`)
	if p["HITId"] != hit["HITId"] {
		t.Fatalf("ListReviewPolicyResultsForHIT: %v", p)
	}
	if l := call("ListHITs", `{"MaxResults":1}`); l["NumResults"] != float64(1) {
		t.Fatalf("ListHITs: %v", l)
	}
}
