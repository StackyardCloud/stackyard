package server

import (
	"math"
	"net/http"
	"strings"
	"testing"
	"time"
)

func bcmpcMonthStart() string {
	start, _ := bcmpcInterval(time.Now().UTC())
	return start.Format(time.RFC3339)
}

func bcmpcUsageJSON(key, usageType string, amount string) string {
	return `{"serviceCode":"AmazonEC2","usageType":"` + usageType + `","operation":"RunInstances","key":"` + key +
		`","usageAccountId":"123456789012","amounts":[{"startHour":"` + bcmpcMonthStart() + `","amount":` + amount + `}]}`
}

func bcmpcAmount(t *testing.T, v any) float64 {
	t.Helper()
	f, ok := v.(map[string]any)["amount"].(float64)
	if !ok {
		t.Fatalf("no amount in %v", v)
	}
	return f
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-4 }

func TestBCMPricingCalculatorStage1BillEstimate(t *testing.T) {
	_, ts := newBudTestServer(t)
	sc := budOKAPI(t, ts, bcmpcAPI, "CreateBillScenario", `{"name":"growth","tags":{"team":"finance"}}`)
	id := sc["id"].(string)
	if sc["status"] != "READY" || sc["groupSharingPreference"] != "OPEN" {
		t.Fatalf("scenario: %v", sc)
	}
	start, end := bcmpcInterval(time.Now().UTC())
	hours := end.Sub(start).Hours()

	// Two m5.large instances all month; one bad line of each kind.
	usage := budOKAPI(t, ts, bcmpcAPI, "BatchCreateBillScenarioUsageModification", `{"billScenarioId":"`+id+`","usageModifications":[`+
		bcmpcUsageJSON("web", "BoxUsage:m5.large", "2")+`,`+bcmpcUsageJSON("web", "BoxUsage:m5.large", "1")+`,`+
		bcmpcUsageJSON("nope", "BoxUsage:z9.huge", "1")+`,`+strings.Replace(bcmpcUsageJSON("acct", "BoxUsage:t3.micro", "1"), "123456789012", "999999999999", 1)+`]}`)
	if items := usage["items"].([]any); len(items) != 1 || items[0].(map[string]any)["location"] != bcmpcUSEast {
		t.Fatalf("usage items: %v", usage)
	}
	codes := map[string]string{}
	for _, e := range usage["errors"].([]any) {
		codes[e.(map[string]any)["key"].(string)] = e.(map[string]any)["errorCode"].(string)
	}
	if codes["web"] != "CONFLICT" || codes["nope"] != "NOT_FOUND" || codes["acct"] != "BAD_REQUEST" {
		t.Fatalf("batch errors: %v", codes)
	}
	// One RI covering one instance.
	commits := budOKAPI(t, ts, bcmpcAPI, "BatchCreateBillScenarioCommitmentModification", `{"billScenarioId":"`+id+`","commitmentModifications":[`+
		`{"key":"ri","usageAccountId":"123456789012","commitmentAction":{"addReservedInstanceAction":{"reservedInstancesOfferingId":"7d1e2f3a-4b5c-4d6e-8f9a-0b1c2d3e4f5a","instanceCount":1}}}]}`)
	if len(commits["items"].([]any)) != 1 {
		t.Fatalf("commitments: %v", commits)
	}
	if code := budErrAPI(t, ts, bcmpcAPI, "BatchCreateBillScenarioCommitmentModification", `{"billScenarioId":"`+id+`","commitmentModifications":[`+
		`{"key":"x","usageAccountId":"123456789012","commitmentAction":{"addReservedInstanceAction":{"reservedInstancesOfferingId":"00000000-0000-0000-0000-000000000000","instanceCount":1}}}]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("unknown offering: %s", code)
	}

	if code := budErrAPI(t, ts, bcmpcAPI, "BatchCreateBillScenarioCommitmentModification", `{"billScenarioId":"`+id+`","commitmentModifications":[null]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("null commitment: %s", code)
	}
	nulls := budOKAPI(t, ts, bcmpcAPI, "BatchCreateBillScenarioUsageModification", `{"billScenarioId":"`+id+`","usageModifications":[null]}`)
	if len(nulls["items"].([]any)) != 0 || len(nulls["errors"].([]any)) != 1 {
		t.Fatalf("null usage entry: %v", nulls)
	}

	// The estimate starts IN_PROGRESS and locks the scenario until the next request.
	est := budOKAPI(t, ts, bcmpcAPI, "CreateBillEstimate", `{"billScenarioId":"`+id+`","name":"q3"}`)
	estID := est["id"].(string)
	if est["status"] != "IN_PROGRESS" || est["costSummary"] != nil {
		t.Fatalf("new estimate: %v", est)
	}
	srvStatus, out := budCallAPI(t, ts, bcmpcAPI, "GetBillScenario", `{"identifier":"`+id+`"}`)
	if srvStatus != http.StatusOK || out["status"] != "READY" {
		t.Fatalf("scenario after estimate: %d %v", srvStatus, out)
	}
	est = budOKAPI(t, ts, bcmpcAPI, "GetBillEstimate", `{"identifier":"`+estID+`"}`)
	total := bcmpcAmount(t, est["costSummary"].(map[string]any)["totalCostDifference"].(map[string]any)["estimatedCost"])
	// On demand 2 x 0.096 x hours, minus one covered instance, plus the RI fee 0.060 x hours.
	want := 2*0.096*hours - 0.096*hours + 0.060*hours
	if est["status"] != "COMPLETE" || !near(total, want) {
		t.Fatalf("estimate total %v, want %v: %v", total, want, est)
	}
	lines := budOKAPI(t, ts, bcmpcAPI, "ListBillEstimateLineItems", `{"billEstimateId":"`+estID+`","filters":[{"name":"LINE_ITEM_TYPE","values":["DiscountedUsage"]}]}`)["items"].([]any)
	if len(lines) != 1 || !near(bcmpcAmount(t, lines[0].(map[string]any)["estimatedCost"]), -0.096*hours) {
		t.Fatalf("discount lines: %v", lines)
	}
	cs := budOKAPI(t, ts, bcmpcAPI, "ListBillEstimateCommitments", `{"billEstimateId":"`+estID+`"}`)["items"].([]any)
	if len(cs) != 1 || cs[0].(map[string]any)["purchaseAgreementType"] != "RESERVED_INSTANCE" || !near(bcmpcAmount(t, cs[0].(map[string]any)["monthlyPayment"]), 0.060*730) {
		t.Fatalf("estimate commitments: %v", cs)
	}
	inputs := budOKAPI(t, ts, bcmpcAPI, "ListBillEstimateInputUsageModifications", `{"billEstimateId":"`+estID+`"}`)["items"].([]any)
	if len(inputs) != 1 {
		t.Fatalf("inputs: %v", inputs)
	}

	// The estimate is a snapshot: later scenario changes don't touch it.
	mod := inputs[0].(map[string]any)["id"].(string)
	budOKAPI(t, ts, bcmpcAPI, "BatchDeleteBillScenarioUsageModification", `{"billScenarioId":"`+id+`","ids":["`+mod+`"]}`)
	if n := len(budOKAPI(t, ts, bcmpcAPI, "ListBillEstimateInputUsageModifications", `{"billEstimateId":"`+estID+`"}`)["items"].([]any)); n != 1 {
		t.Fatalf("snapshot changed: %d", n)
	}
	del := budOKAPI(t, ts, bcmpcAPI, "BatchDeleteBillScenarioUsageModification", `{"billScenarioId":"`+id+`","ids":["`+mod+`"]}`)
	if len(del["errors"].([]any)) != 1 {
		t.Fatalf("second delete: %v", del)
	}

	// Update, list filters, delete; deleting an in-progress estimate conflicts.
	if code := budErrAPI(t, ts, bcmpcAPI, "UpdateBillEstimate", `{"identifier":"`+estID+`","expiresAt":1}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("past expiry: %s", code)
	}
	budOKAPI(t, ts, bcmpcAPI, "UpdateBillEstimate", `{"identifier":"`+estID+`","name":"q4"}`)
	found := budOKAPI(t, ts, bcmpcAPI, "ListBillEstimates", `{"filters":[{"name":"NAME","values":["q"],"matchOption":"STARTS_WITH"}]}`)["items"].([]any)
	if len(found) != 1 || found[0].(map[string]any)["name"] != "q4" {
		t.Fatalf("list estimates: %v", found)
	}
	// A new estimate locks the scenario only until the next request, so
	// this delete (which completes it first) succeeds.
	budOKAPI(t, ts, bcmpcAPI, "CreateBillEstimate", `{"billScenarioId":"`+id+`","name":"again"}`)
	budOKAPI(t, ts, bcmpcAPI, "DeleteBillScenario", `{"identifier":"`+id+`"}`)
	if code := budErrAPI(t, ts, bcmpcAPI, "GetBillScenario", `{"identifier":"`+id+`"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("deleted scenario: %s", code)
	}
}

func TestBCMPricingCalculatorStage1Locking(t *testing.T) {
	srv, ts := newBudTestServer(t)
	id := budOKAPI(t, ts, bcmpcAPI, "CreateBillScenario", `{"name":"lock"}`)["id"].(string)
	est := budOKAPI(t, ts, bcmpcAPI, "CreateBillEstimate", `{"billScenarioId":"`+id+`","name":"e"}`)["id"].(string)
	// While the estimate is in progress (checked without advancing), the
	// scenario is LOCKED and edits conflict.
	st := srv.bcmPricingCalculator
	st.mu.Lock()
	sc, err := st.scenario(id, true, time.Now().UTC())
	locked := st.scenarioStatus(st.scenarios[id], time.Now().UTC())
	_, delErr := st.estimate(est)
	st.mu.Unlock()
	if sc != nil || err == nil || err.code != "ConflictException" || err.extra["resourceType"] != bcmpcScenarioKind || locked != "LOCKED" || delErr != nil {
		t.Fatalf("locked scenario: %v %v %s", sc, err, locked)
	}
	status, out := budCallAPI(t, ts, bcmpcAPI, "GetBillEstimate", `{"identifier":"`+est+`"}`)
	if status != http.StatusOK || out["status"] != "COMPLETE" {
		t.Fatalf("estimate: %v", out)
	}
	status, out = budCallAPI(t, ts, bcmpcAPI, "GetBillScenario", `{"identifier":"00000000-0000-0000-0000-000000000000"}`)
	if msg, _ := out["message"].(string); status != http.StatusNotFound || !strings.Contains(msg, "does not exist") ||
		out["resourceId"] != "00000000-0000-0000-0000-000000000000" || out["resourceType"] != bcmpcScenarioKind {
		t.Fatalf("not found shape: %d %v", status, out)
	}
	if code := budErrAPI(t, ts, bcmpcAPI, "CreateBillScenario", `{"name":"p","groupSharingPreference":"PRIORITIZED"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("sharing without cost category: %s", code)
	}
}

func TestBCMPricingCalculatorStage1WorkloadsAndPreferences(t *testing.T) {
	srv, ts := newBudTestServer(t)
	w := budOKAPI(t, ts, bcmpcAPI, "CreateWorkloadEstimate", `{"name":"app","clientToken":"tok"}`)
	id := w["id"].(string)
	if again := budOKAPI(t, ts, bcmpcAPI, "CreateWorkloadEstimate", `{"name":"app","clientToken":"tok"}`)["id"]; again != id {
		t.Fatalf("idempotent create: %v", again)
	}
	res := budOKAPI(t, ts, bcmpcAPI, "BatchCreateWorkloadEstimateUsage", `{"workloadEstimateId":"`+id+`","usage":[`+
		`{"serviceCode":"AmazonS3","usageType":"TimedStorage-ByteHrs","operation":"StandardStorage","key":"s3","usageAccountId":"123456789012","amount":500},`+
		`{"serviceCode":"AmazonS3","usageType":"Nope","operation":"StandardStorage","key":"bad","usageAccountId":"123456789012","amount":1}]}`)
	items := res["items"].([]any)
	if len(items) != 1 || !near(items[0].(map[string]any)["cost"].(float64), 11.5) || len(res["errors"].([]any)) != 1 {
		t.Fatalf("usage: %v", res)
	}
	// UPDATING until the next request, then VALID with the new total.
	st := srv.bcmPricingCalculator
	st.mu.Lock()
	status := st.workloads[id].status
	st.mu.Unlock()
	if status != "UPDATING" {
		t.Fatalf("status after change: %s", status)
	}
	got := budOKAPI(t, ts, bcmpcAPI, "GetWorkloadEstimate", `{"identifier":"`+id+`"}`)
	if got["status"] != "VALID" || !near(got["totalCost"].(float64), 11.5) {
		t.Fatalf("workload: %v", got)
	}
	line := items[0].(map[string]any)["id"].(string)
	budOKAPI(t, ts, bcmpcAPI, "BatchUpdateWorkloadEstimateUsage", `{"workloadEstimateId":"`+id+`","usage":[{"id":"`+line+`","amount":1000}]}`)
	if got := budOKAPI(t, ts, bcmpcAPI, "GetWorkloadEstimate", `{"identifier":"`+id+`"}`); !near(got["totalCost"].(float64), 23) {
		t.Fatalf("updated total: %v", got)
	}
	listed := budOKAPI(t, ts, bcmpcAPI, "ListWorkloadEstimateUsage", `{"workloadEstimateId":"`+id+`","filters":[{"name":"SERVICE_CODE","values":["AmazonS3"]}]}`)["items"].([]any)
	if len(listed) != 1 {
		t.Fatalf("listed usage: %v", listed)
	}

	cost := budOKAPI(t, ts, bcmpcAPI, "GetUsageLinesEstimatedCost", `{"rateType":"BEFORE_DISCOUNTS","usage":[`+
		`{"serviceCode":"AWSLambda","usageType":"Request","operation":"Invoke","key":"l","amount":1000000},`+
		`{"serviceCode":"AWSLambda","usageType":"Nope","operation":"Invoke","key":"n","amount":1}]}`)
	if !near(cost["totalCost"].(float64), 0.2) || len(cost["errors"].([]any)) != 1 {
		t.Fatalf("usage lines: %v", cost)
	}

	// Disallowed rate types are refused with reason disallowedRate.
	budOKAPI(t, ts, bcmpcAPI, "UpdatePreferences", `{"standaloneAccountRateTypeSelections":["BEFORE_DISCOUNTS"]}`)
	code, out := budCallAPI(t, ts, bcmpcAPI, "GetUsageLinesEstimatedCost", `{"rateType":"AFTER_DISCOUNTS","usage":[{"serviceCode":"AWSLambda","usageType":"Request","operation":"Invoke","key":"l","amount":1}]}`)
	if code != http.StatusBadRequest || out["reason"] != "disallowedRate" {
		t.Fatalf("disallowed rate: %d %v", code, out)
	}
	prefs := budOKAPI(t, ts, bcmpcAPI, "GetPreferences", `{}`)
	if len(prefs["standaloneAccountRateTypeSelections"].([]any)) != 1 || len(prefs["memberAccountRateTypeSelections"].([]any)) != 3 {
		t.Fatalf("prefs: %v", prefs)
	}

	// Tags and the byte budget, which update paths are charged against too.
	arn := "arn:aws:bcm-pricing-calculator::123456789012:workload-estimate/" + id
	budOKAPI(t, ts, bcmpcAPI, "TagResource", `{"arn":"`+arn+`","tags":{"env":"dev"}}`)
	if tags := budOKAPI(t, ts, bcmpcAPI, "ListTagsForResource", `{"arn":"`+arn+`"}`)["tags"].(map[string]any); tags["env"] != "dev" {
		t.Fatalf("tags: %v", tags)
	}
	st.mu.Lock()
	st.bytes = bcmpcMaxStoreBytes - 5
	st.mu.Unlock()
	if code := budErrAPI(t, ts, bcmpcAPI, "UpdateWorkloadEstimate", `{"identifier":"`+id+`","name":"a-much-longer-workload-name"}`, http.StatusPaymentRequired); code != "ServiceQuotaExceededException" {
		t.Fatalf("update over budget: %s", code)
	}
	if code := budErrAPI(t, ts, bcmpcAPI, "BatchUpdateWorkloadEstimateUsage", `{"workloadEstimateId":"`+id+`","usage":[{"id":"`+line+`","group":"a-long-group-name"}]}`, http.StatusPaymentRequired); code != "ServiceQuotaExceededException" {
		t.Fatalf("usage update over budget: %s", code)
	}
	budOKAPI(t, ts, bcmpcAPI, "DeleteWorkloadEstimate", `{"identifier":"`+id+`"}`)
	if code := budErrAPI(t, ts, bcmpcAPI, "GetWorkloadEstimate", `{"identifier":"`+id+`"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("deleted: %s", code)
	}
}
