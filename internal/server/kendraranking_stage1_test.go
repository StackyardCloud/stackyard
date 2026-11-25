package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestKendraRankingStage1PlansAndRescore(t *testing.T) {
	ts := newCSTestServer(t)
	created := kdrOK(t, ts, "CreateRescoreExecutionPlan", `{"Name":"plan-a","ClientToken":"tok-1","CapacityUnits":{"RescoreCapacityUnits":2},"Tags":[{"Key":"team","Value":"search"}]}`)
	id, arn := created["Id"].(string), created["Arn"].(string)
	if len(id) != 36 || !strings.HasSuffix(arn, "rescore-execution-plan/"+id) {
		t.Fatalf("create: %v", created)
	}
	// The client token makes create idempotent.
	if again := kdrOK(t, ts, "CreateRescoreExecutionPlan", `{"Name":"plan-a","ClientToken":"tok-1"}`); again["Id"] != id {
		t.Fatalf("idempotent create: %v", again)
	}
	idBody := `{"Id":"` + id + `"}`
	if d := kdrOK(t, ts, "DescribeRescoreExecutionPlan", idBody); d["Status"] != "CREATING" || d["CapacityUnits"].(map[string]any)["RescoreCapacityUnits"] != float64(2) {
		t.Fatalf("first describe: %v", d)
	}
	if d := kdrOK(t, ts, "DescribeRescoreExecutionPlan", idBody); d["Status"] != "ACTIVE" {
		t.Fatalf("second describe: %v", d)
	}
	kdrOK(t, ts, "UpdateRescoreExecutionPlan", `{"Id":"`+id+`","Name":"plan-b","Description":"renamed"}`)
	if d := kdrOK(t, ts, "DescribeRescoreExecutionPlan", idBody); d["Status"] != "UPDATING" || d["Name"] != "plan-b" || d["Description"] != "renamed" {
		t.Fatalf("after update: %v", d)
	}

	// Rescore: title matches weigh double, groups share their best score.
	out := kdrOK(t, ts, "Rescore", `{"RescoreExecutionPlanId":"`+id+`","SearchQuery":"cloud search","Documents":[
		{"Id":"body-only","Body":"a cloud platform","OriginalScore":9},
		{"Id":"title","Title":"Cloud Search","OriginalScore":1},
		{"Id":"none","TokenizedBody":["unrelated"],"OriginalScore":5},
		{"Id":"grouped","GroupId":"g","Body":"nothing","OriginalScore":0},
		{"Id":"leader","GroupId":"g","Title":"search","OriginalScore":0}]}`)
	items := out["ResultItems"].([]any)
	order := []string{}
	for _, it := range items {
		order = append(order, it.(map[string]any)["DocumentId"].(string))
	}
	if got := strings.Join(order, ","); got != "title,grouped,leader,body-only,none" {
		t.Fatalf("rescore order: %s (%v)", got, items)
	}
	if code := kdrErrCode(t, ts, "Rescore", `{"RescoreExecutionPlanId":"`+id+`","SearchQuery":"q","Documents":[{"Id":"x","Body":"a","TokenizedBody":["a"],"OriginalScore":1}]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("body and tokenized body: %s", code)
	}
	if code := kdrErrCode(t, ts, "Rescore", `{"RescoreExecutionPlanId":"`+id+`","SearchQuery":"q","Documents":[{"Id":"x","Body":"a","OriginalScore":1},{"Id":"x","Body":"b","OriginalScore":1}]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("duplicate ids: %s", code)
	}

	// Tags.
	kdrOK(t, ts, "TagResource", `{"ResourceARN":"`+arn+`","Tags":[{"Key":"env","Value":"dev"}]}`)
	kdrOK(t, ts, "UntagResource", `{"ResourceARN":"`+arn+`","TagKeys":["team"]}`)
	if tags := kdrOK(t, ts, "ListTagsForResource", `{"ResourceARN":"`+arn+`"}`)["Tags"].([]any); len(tags) != 1 || tags[0].(map[string]any)["Key"] != "env" {
		t.Fatalf("tags: %v", tags)
	}
	if code := kdrErrCode(t, ts, "TagResource", `{"ResourceARN":"`+arn+`","Tags":[{"Key":"aws:x","Value":"v"}]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("aws: tag: %s", code)
	}

	// Paging and delete.
	for i := 0; i < 3; i++ {
		kdrOK(t, ts, "CreateRescoreExecutionPlan", fmt.Sprintf(`{"Name":"p%d"}`, i))
	}
	page := kdrOK(t, ts, "ListRescoreExecutionPlans", `{"MaxResults":2}`)
	if len(page["SummaryItems"].([]any)) != 2 || page["NextToken"] == nil {
		t.Fatalf("page 1: %v", page)
	}
	page = kdrOK(t, ts, "ListRescoreExecutionPlans", `{"MaxResults":2,"NextToken":"`+page["NextToken"].(string)+`"}`)
	if len(page["SummaryItems"].([]any)) != 2 || page["NextToken"] != nil {
		t.Fatalf("page 2: %v", page)
	}
	kdrOK(t, ts, "DeleteRescoreExecutionPlan", idBody)
	if code := kdrErrCode(t, ts, "DescribeRescoreExecutionPlan", idBody, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("after delete: %s", code)
	}
	if code := kdrErrCode(t, ts, "ListTagsForResource", `{"ResourceARN":"`+arn+`"}`, http.StatusNotFound); code != "ResourceUnavailableException" {
		t.Fatalf("tags after delete: %s", code)
	}
	if code := kdrErrCode(t, ts, "DescribeRescoreExecutionPlan", `{"Id":"short"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("bad id: %s", code)
	}
}

func TestKendraRankingStage1PlanLimit(t *testing.T) {
	st := newKendraRankingStore()
	for i := 0; i < kdrMaxPlans; i++ {
		if _, err := st.createPlan(map[string]any{"Name": fmt.Sprintf("p%d", i)}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.createPlan(map[string]any{"Name": "over"}); err == nil || err.code != "ServiceQuotaExceededException" {
		t.Fatalf("plan limit: %v", err)
	}
}
