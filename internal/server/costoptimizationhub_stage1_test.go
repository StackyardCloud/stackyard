package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestCostOptimizationHubStage1Enrollment(t *testing.T) {
	_, ts := newBudTestServer(t)
	status, out := budCallAPI(t, ts, cohAPI, "ListRecommendations", `{}`)
	if msg, _ := out["Message"].(string); status != http.StatusForbidden || out["__type"] != "AccessDeniedException" || !strings.Contains(msg, "UpdateEnrollmentStatus") {
		t.Fatalf("not enrolled: %d %v", status, out)
	}
	st := budOKAPI(t, ts, cohAPI, "ListEnrollmentStatuses", `{}`)["items"].([]any)
	if st[0].(map[string]any)["status"] != "Inactive" {
		t.Fatalf("initial status: %v", st)
	}
	if out := budOKAPI(t, ts, cohAPI, "UpdateEnrollmentStatus", `{"status":"Active","includeMemberAccounts":true}`); out["status"] != "Active" {
		t.Fatalf("enroll: %v", out)
	}
	out = budOKAPI(t, ts, cohAPI, "ListEnrollmentStatuses", `{}`)
	if out["includeMemberAccounts"] != true || out["items"].([]any)[0].(map[string]any)["status"] != "Active" {
		t.Fatalf("enrolled: %v", out)
	}
	if items := budOKAPI(t, ts, cohAPI, "ListEnrollmentStatuses", `{"accountId":"111122223333"}`)["items"].([]any); len(items) != 0 {
		t.Fatalf("other account: %v", items)
	}
	if code := budErrAPI(t, ts, cohAPI, "UpdateEnrollmentStatus", `{"status":"Maybe"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("bad status: %s", code)
	}
}

func TestCostOptimizationHubStage1Preferences(t *testing.T) {
	_, ts := newBudTestServer(t)
	if p := budOKAPI(t, ts, cohAPI, "GetPreferences", `{}`); p["savingsEstimationMode"] != "BeforeDiscounts" {
		t.Fatalf("defaults: %v", p)
	}
	budOKAPI(t, ts, cohAPI, "UpdatePreferences", `{"savingsEstimationMode":"AfterDiscounts","preferredCommitment":{"term":"ThreeYears"}}`)
	p := budOKAPI(t, ts, cohAPI, "GetPreferences", `{}`)
	pc := p["preferredCommitment"].(map[string]any)
	if p["savingsEstimationMode"] != "AfterDiscounts" || pc["term"] != "ThreeYears" || pc["paymentOption"] != "NoUpfront" {
		t.Fatalf("after update: %v", p)
	}
}

func TestCostOptimizationHubStage1Recommendations(t *testing.T) {
	_, ts := newBudTestServer(t)
	budOKAPI(t, ts, cohAPI, "UpdateEnrollmentStatus", `{"status":"Active"}`)
	items := budOKAPI(t, ts, cohAPI, "ListRecommendations", `{}`)["items"].([]any)
	all := budOKAPI(t, ts, cohAPI, "ListRecommendations", `{"includeAllRecommendations":true}`)["items"].([]any)
	if len(items) != len(cohSeeds)-1 || len(all) != len(cohSeeds) {
		t.Fatalf("deduped %d, all %d", len(items), len(all))
	}
	if first := items[0].(map[string]any); first["recommendationId"] != "coh-csp-purchase-0005" {
		t.Fatalf("default order is savings desc: %v", first["recommendationId"])
	}
	asc := budOKAPI(t, ts, cohAPI, "ListRecommendations", `{"orderBy":{"dimension":"EstimatedMonthlySavings","order":"Asc"}}`)["items"].([]any)
	if asc[0].(map[string]any)["recommendationId"] != "coh-ebs-upgrade-0002" {
		t.Fatalf("ascending: %v", asc[0])
	}
	if code := budErrAPI(t, ts, cohAPI, "ListRecommendations", `{"orderBy":{"dimension":"Colour"}}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("bad dimension: %s", code)
	}
	filtered := budOKAPI(t, ts, cohAPI, "ListRecommendations", `{"filter":{"resourceTypes":["Ec2Instance"],"tags":[{"key":"env","value":"prod"}]}}`)["items"].([]any)
	if len(filtered) != 1 {
		t.Fatalf("filter: %v", filtered)
	}
	page := budOKAPI(t, ts, cohAPI, "ListRecommendations", `{"maxResults":4}`)
	if len(page["items"].([]any)) != 4 || page["nextToken"] == nil {
		t.Fatalf("page: %v", page)
	}

	rec := budOKAPI(t, ts, cohAPI, "GetRecommendation", `{"recommendationId":"coh-ec2-rightsize-0001"}`)
	cur := rec["currentResourceDetails"].(map[string]any)["ec2Instance"].(map[string]any)
	if cur["configuration"].(map[string]any)["instance"].(map[string]any)["type"] != "m5.2xlarge" || rec["accountId"] != cohAccountID {
		t.Fatalf("recommendation: %v", rec)
	}
	if code := budErrAPI(t, ts, cohAPI, "GetRecommendation", `{"recommendationId":"nope"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("missing recommendation: %s", code)
	}

	sum := budOKAPI(t, ts, cohAPI, "ListRecommendationSummaries", `{"groupBy":"ResourceType","metrics":["SavingsPercentage"]}`)
	if sum["metrics"].(map[string]any)["savingsPercentage"] == "" || len(sum["items"].([]any)) != 5 {
		t.Fatalf("summaries: %v", sum)
	}
	if code := budErrAPI(t, ts, cohAPI, "ListRecommendationSummaries", `{"groupBy":"Colour"}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("bad groupBy: %s", code)
	}

	m := budOKAPI(t, ts, cohAPI, "ListEfficiencyMetrics", `{"granularity":"Daily","timePeriod":{"start":"2026-01-01","end":"2026-01-08"},"groupBy":"Region"}`)
	groups := m["efficiencyMetricsByGroup"].([]any)
	if len(groups) != 4 || len(groups[0].(map[string]any)["metricsByTime"].([]any)) != 7 {
		t.Fatalf("efficiency metrics: %v", m)
	}
	if code := budErrAPI(t, ts, cohAPI, "ListEfficiencyMetrics", `{"granularity":"Daily","timePeriod":{"start":"2026-01-08","end":"2026-01-01"}}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("reversed period: %s", code)
	}
	if code := budErrAPI(t, ts, cohAPI, "ListEfficiencyMetrics", `{"granularity":"Daily","timePeriod":{"start":"2020-01-01","end":"2026-01-01"}}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("too many periods: %s", code)
	}
}
