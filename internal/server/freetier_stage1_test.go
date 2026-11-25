package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestFreeTierStage1PlanAndActivities(t *testing.T) {
	_, ts := newBudTestServer(t)
	plan := budOKAPI(t, ts, ftierAPI, "GetAccountPlanState", `{}`)
	credits := plan["accountPlanRemainingCredits"].(map[string]any)
	if plan["accountPlanType"] != "FREE" || plan["accountPlanStatus"] != "ACTIVE" || credits["amount"] != 100.0 {
		t.Fatalf("plan: %v", plan)
	}
	list := budOKAPI(t, ts, ftierAPI, "ListAccountActivities", `{}`)["activities"].([]any)
	if len(list) != len(ftierActivities) {
		t.Fatalf("activities: %d", len(list))
	}
	var budgetID string
	for _, a := range ftierActivities {
		if a.budgets {
			budgetID = a.id
		}
	}
	if got := budOKAPI(t, ts, ftierAPI, "GetAccountActivity", `{"activityId":"`+budgetID+`"}`); got["status"] != "NOT_STARTED" || got["instructionsUrl"] == "" {
		t.Fatalf("activity: %v", got)
	}
	if code := budErrAPI(t, ts, ftierAPI, "GetAccountActivity", `{"activityId":"`+strings.Repeat("0", 32)+`"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("unknown activity: %s", code)
	}

	// Creating a budget completes the Budgets activity and earns its credit.
	budOK(t, ts, "CreateBudget", `{"AccountId":"123456789012","Budget":{"BudgetName":"ft","BudgetLimit":{"Amount":"10","Unit":"USD"},"TimeUnit":"MONTHLY","BudgetType":"COST"}}`)
	if got := budOKAPI(t, ts, ftierAPI, "GetAccountActivity", `{"activityId":"`+budgetID+`"}`); got["status"] != "COMPLETED" || got["completedAt"] == nil {
		t.Fatalf("activity after budget: %v", got)
	}
	done := budOKAPI(t, ts, ftierAPI, "ListAccountActivities", `{"filterActivityStatuses":["COMPLETED"]}`)["activities"].([]any)
	if len(done) != 1 {
		t.Fatalf("completed: %v", done)
	}
	plan = budOKAPI(t, ts, ftierAPI, "GetAccountPlanState", `{}`)
	if plan["accountPlanRemainingCredits"].(map[string]any)["amount"] != 120.0 {
		t.Fatalf("credits after activity: %v", plan)
	}

	if code := budErrAPI(t, ts, ftierAPI, "UpgradeAccountPlan", `{"accountPlanType":"FREE"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("downgrade: %s", code)
	}
	if got := budOKAPI(t, ts, ftierAPI, "UpgradeAccountPlan", `{"accountPlanType":"PAID"}`); got["accountPlanType"] != "PAID" {
		t.Fatalf("upgrade: %v", got)
	}
	budOKAPI(t, ts, ftierAPI, "UpgradeAccountPlan", `{"accountPlanType":"PAID"}`)
	if got := budOKAPI(t, ts, ftierAPI, "GetAccountPlanState", `{}`); got["accountPlanType"] != "PAID" {
		t.Fatalf("plan after upgrade: %v", got)
	}
}

func TestFreeTierStage1UsageFilters(t *testing.T) {
	_, ts := newBudTestServer(t)
	count := func(body string) int {
		return len(budOKAPI(t, ts, ftierAPI, "GetFreeTierUsage", body)["freeTierUsages"].([]any))
	}
	if n := count(`{}`); n != len(ftierUsages) {
		t.Fatalf("all: %d", n)
	}
	if n := count(`{"filter":{"Dimensions":{"Key":"SERVICE","Values":["AWS Lambda"],"MatchOptions":["EQUALS"]}}}`); n != 2 {
		t.Fatalf("lambda: %d", n)
	}
	if n := count(`{"filter":{"Not":{"Dimensions":{"Key":"FREE_TIER_TYPE","Values":["Always"],"MatchOptions":["STARTS_WITH"]}}}}`); n != 5 {
		t.Fatalf("not always free: %d", n)
	}
	and := `{"filter":{"And":[{"Dimensions":{"Key":"REGION","Values":["us-east-1"],"MatchOptions":["EQUALS"]}},{"Dimensions":{"Key":"USAGE_PERCENTAGE","Values":["20"],"MatchOptions":["GREATER_THAN_OR_EQUAL"]}}]}}`
	if n := count(and); n != 2 { // EBS 27%, S3 storage 24%
		t.Fatalf("and: %d", n)
	}
	if status, out := budCallAPI(t, ts, ftierAPI, "GetFreeTierUsage", `{"filter":{}}`); status != http.StatusBadRequest ||
		out["__type"] != "ValidationException" || out["message"] == nil {
		t.Fatalf("empty expression: %d %v", status, out)
	}
	deep := `{"Dimensions":{"Key":"SERVICE","Values":["x"],"MatchOptions":["EQUALS"]}}`
	for i := 0; i <= ftierMaxDepth+1; i++ {
		deep = `{"Not":` + deep + `}`
	}
	budErrAPI(t, ts, ftierAPI, "GetFreeTierUsage", `{"filter":`+deep+`}`, http.StatusBadRequest)
	many := `"x"` + strings.Repeat(`,"x"`, ftierMaxValues)
	budErrAPI(t, ts, ftierAPI, "GetFreeTierUsage", `{"filter":{"Dimensions":{"Key":"SERVICE","Values":[`+many+`],"MatchOptions":["EQUALS"]}}}`, http.StatusBadRequest)
	page := budOKAPI(t, ts, ftierAPI, "GetFreeTierUsage", `{"maxResults":4}`)
	if len(page["freeTierUsages"].([]any)) != 4 || page["nextToken"] == nil {
		t.Fatalf("page: %v", page)
	}
}
