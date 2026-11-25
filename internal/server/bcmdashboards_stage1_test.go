package server

import (
	"net/http"
	"strings"
	"testing"
)

const bcmdWidgetID = "0123456789abcdef0123456789abcdef"

func bcmdWidgetJSON(id, query string) string {
	return `[{"id":"` + id + `","title":"Cost by service","width":4,"configs":[{"queryParameters":` + query +
		`,"displayConfig":{"graph":{"default":{"visualType":"BAR"}}}}]}]`
}

const bcmdQuery = `{"costAndUsage":{"metrics":["UnblendedCost"],"granularity":"MONTHLY",` +
	`"timeRange":{"startTime":{"type":"RELATIVE","value":"-P3M"},"endTime":{"type":"RELATIVE","value":"P0D"}},` +
	`"groupBy":[{"key":"SERVICE","type":"DIMENSION"}],"filter":{"dimensions":{"key":"REGION","values":["us-east-1"]}}}}`

func bcmdReportJSON(name, dashboard string) string {
	return `{"name":"` + name + `","dashboardArn":"` + dashboard + `","scheduledReportExecutionRoleArn":"arn:aws:iam::123456789012:role/reports",` +
		`"scheduleConfig":{"scheduleExpression":"rate(1 day)"},"widgetIds":["` + bcmdWidgetID + `"]}`
}

func TestBCMDashboardsStage1DashboardLifecycle(t *testing.T) {
	_, ts := newBudTestServer(t)
	arn := budOKAPI(t, ts, bcmdAPI, "CreateDashboard", `{"name":"Spend","widgets":`+bcmdWidgetJSON(bcmdWidgetID, bcmdQuery)+
		`,"resourceTags":[{"key":"team","value":"finance"}]}`)["arn"].(string)
	if !strings.HasPrefix(arn, "arn:aws:bcm-dashboards::123456789012:dashboard/") {
		t.Fatalf("arn %s", arn)
	}
	got := budOKAPI(t, ts, bcmdAPI, "GetDashboard", `{"arn":"`+arn+`"}`)
	w := got["widgets"].([]any)[0].(map[string]any)
	if got["type"] != "CUSTOM" || w["id"] != bcmdWidgetID || w["height"] != 4.0 || w["horizontalOffset"] != 0.0 {
		t.Fatalf("dashboard: %v", got)
	}

	// Widgets without ids get one; Cost Explorer rules are enforced.
	upd := budOKAPI(t, ts, bcmdAPI, "UpdateDashboard", `{"arn":"`+arn+`","name":"Spend v2","widgets":`+
		`[{"title":"Auto id","configs":[{"queryParameters":`+bcmdQuery+`,"displayConfig":{"table":{}}}]}]}`)
	if upd["arn"] != arn {
		t.Fatalf("update: %v", upd)
	}
	got = budOKAPI(t, ts, bcmdAPI, "GetDashboard", `{"arn":"`+arn+`"}`)
	if id := got["widgets"].([]any)[0].(map[string]any)["id"].(string); len(id) != 32 || got["name"] != "Spend v2" {
		t.Fatalf("auto id: %v", got)
	}
	for name, query := range map[string]string{
		"two query kinds":   `{"costAndUsage":{"metrics":["UnblendedCost"],"granularity":"DAILY","timeRange":{"startTime":{"type":"RELATIVE","value":"-P1M"},"endTime":{"type":"RELATIVE","value":"P0D"}}},"reservationUtilization":{"timeRange":{"startTime":{"type":"RELATIVE","value":"-P1M"},"endTime":{"type":"RELATIVE","value":"P0D"}}}}`,
		"no metrics":        `{"costAndUsage":{"metrics":[],"granularity":"DAILY","timeRange":{"startTime":{"type":"RELATIVE","value":"-P1M"},"endTime":{"type":"RELATIVE","value":"P0D"}}}}`,
		"reversed range":    `{"costAndUsage":{"metrics":["UnblendedCost"],"granularity":"DAILY","timeRange":{"startTime":{"type":"ABSOLUTE","value":"2026-03-01"},"endTime":{"type":"ABSOLUTE","value":"2026-01-01"}}}}`,
		"bad relative":      `{"costAndUsage":{"metrics":["UnblendedCost"],"granularity":"DAILY","timeRange":{"startTime":{"type":"RELATIVE","value":"last week"},"endTime":{"type":"RELATIVE","value":"P0D"}}}}`,
		"single-member and": `{"costAndUsage":{"metrics":["UnblendedCost"],"granularity":"DAILY","timeRange":{"startTime":{"type":"RELATIVE","value":"-P1M"},"endTime":{"type":"RELATIVE","value":"P0D"}},"filter":{"and":[{"dimensions":{"key":"REGION","values":["us-east-1"]}}]}}}`,
		"three groups":      `{"costAndUsage":{"metrics":["UnblendedCost"],"granularity":"DAILY","timeRange":{"startTime":{"type":"RELATIVE","value":"-P1M"},"endTime":{"type":"RELATIVE","value":"P0D"}},"groupBy":[{"key":"SERVICE"},{"key":"REGION"},{"key":"AZ"}]}}`,
	} {
		if code := budErrAPI(t, ts, bcmdAPI, "CreateDashboard", `{"name":"Bad","widgets":`+bcmdWidgetJSON(bcmdWidgetID, query)+`}`, http.StatusBadRequest); code != "ValidationException" {
			t.Fatalf("%s: %s", name, code)
		}
	}
	if code := budErrAPI(t, ts, bcmdAPI, "CreateDashboard", `{"name":"Nulls","widgets":[null]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("null widget: %s", code)
	}
	if code := budErrAPI(t, ts, bcmdAPI, "CreateDashboard", `{"name":"Two  spaces","widgets":[]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("double space: %s", code)
	}
	wide := strings.Replace(bcmdWidgetJSON(bcmdWidgetID, bcmdQuery), `"width":4`, `"width":4,"horizontalOffset":3`, 1)
	if code := budErrAPI(t, ts, bcmdAPI, "CreateDashboard", `{"name":"Wide","widgets":`+wide+`}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("offset: %s", code)
	}

	// Policy, tags, list, delete.
	pol := budOKAPI(t, ts, bcmdAPI, "GetResourcePolicy", `{"resourceArn":"`+arn+`"}`)
	if !strings.Contains(pol["policyDocument"].(string), arn) {
		t.Fatalf("policy: %v", pol)
	}
	budOKAPI(t, ts, bcmdAPI, "TagResource", `{"resourceArn":"`+arn+`","resourceTags":[{"key":"env","value":"dev"}]}`)
	budOKAPI(t, ts, bcmdAPI, "UntagResource", `{"resourceArn":"`+arn+`","resourceTagKeys":["team"]}`)
	tags := budOKAPI(t, ts, bcmdAPI, "ListTagsForResource", `{"resourceArn":"`+arn+`"}`)["resourceTags"].([]any)
	if len(tags) != 1 || tags[0].(map[string]any)["key"] != "env" {
		t.Fatalf("tags: %v", tags)
	}
	if code := budErrAPI(t, ts, bcmdAPI, "TagResource", `{"resourceArn":"`+arn+`","resourceTags":[{"key":"aws:x","value":"1"}]}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("aws: tag: %s", code)
	}
	budOKAPI(t, ts, bcmdAPI, "CreateDashboard", `{"name":"Another","widgets":[]}`)
	page := budOKAPI(t, ts, bcmdAPI, "ListDashboards", `{"maxResults":1}`)
	if len(page["dashboards"].([]any)) != 1 || page["nextToken"] == nil {
		t.Fatalf("page: %v", page)
	}
	if _, ok := page["dashboards"].([]any)[0].(map[string]any)["widgets"]; ok {
		t.Fatalf("list returned widgets")
	}
	budOKAPI(t, ts, bcmdAPI, "DeleteDashboard", `{"arn":"`+arn+`"}`)
	if code := budErrAPI(t, ts, bcmdAPI, "GetDashboard", `{"arn":"`+arn+`"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("deleted: %s", code)
	}
}

func TestBCMDashboardsStage1ScheduledReports(t *testing.T) {
	_, ts := newBudTestServer(t)
	dash := budOKAPI(t, ts, bcmdAPI, "CreateDashboard", `{"name":"Spend","widgets":`+bcmdWidgetJSON(bcmdWidgetID, bcmdQuery)+`}`)["arn"].(string)
	body := `{"scheduledReport":` + bcmdReportJSON("Weekly", dash) + `,"clientToken":"tok-1"}`
	arn := budOKAPI(t, ts, bcmdAPI, "CreateScheduledReport", body)["arn"].(string)
	if again := budOKAPI(t, ts, bcmdAPI, "CreateScheduledReport", body)["arn"]; again != arn {
		t.Fatalf("idempotent create: %v vs %s", again, arn)
	}
	if code := budErrAPI(t, ts, bcmdAPI, "CreateScheduledReport", `{"scheduledReport":`+bcmdReportJSON("Other", dash)+`,"clientToken":"tok-1"}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("token reuse: %s", code)
	}
	if code := budErrAPI(t, ts, bcmdAPI, "CreateScheduledReport", `{"scheduledReport":`+bcmdReportJSON("Weekly", dash)+`}`, http.StatusConflict); code != "ConflictException" {
		t.Fatalf("duplicate name: %s", code)
	}
	bad := strings.Replace(bcmdReportJSON("Bad", dash), "rate(1 day)", "every day", 1)
	if code := budErrAPI(t, ts, bcmdAPI, "CreateScheduledReport", `{"scheduledReport":`+bad+`}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("schedule: %s", code)
	}
	missing := strings.Replace(bcmdReportJSON("Bad", dash), bcmdWidgetID, strings.Repeat("f", 32), 1)
	if code := budErrAPI(t, ts, bcmdAPI, "CreateScheduledReport", `{"scheduledReport":`+missing+`}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("unknown widget: %s", code)
	}

	rep := budOKAPI(t, ts, bcmdAPI, "GetScheduledReport", `{"arn":"`+arn+`"}`)["scheduledReport"].(map[string]any)
	cfg := rep["scheduleConfig"].(map[string]any)
	if cfg["state"] != "ENABLED" || cfg["scheduleExpressionTimeZone"] != "UTC" || rep["healthStatus"].(map[string]any)["statusCode"] != "HEALTHY" {
		t.Fatalf("report: %v", rep)
	}
	dry := budOKAPI(t, ts, bcmdAPI, "ExecuteScheduledReport", `{"arn":"`+arn+`","dryRun":true}`)
	if dry["executionTriggered"] != false {
		t.Fatalf("dry run: %v", dry)
	}
	if run := budOKAPI(t, ts, bcmdAPI, "ExecuteScheduledReport", `{"arn":"`+arn+`"}`); run["executionTriggered"] != true {
		t.Fatalf("run: %v", run)
	}
	rep = budOKAPI(t, ts, bcmdAPI, "GetScheduledReport", `{"arn":"`+arn+`"}`)["scheduledReport"].(map[string]any)
	if rep["lastExecutionAt"] == nil {
		t.Fatalf("lastExecutionAt: %v", rep)
	}

	// Partial update merges the schedule; clearWidgetIds drops the widget filter.
	budOKAPI(t, ts, bcmdAPI, "UpdateScheduledReport", `{"arn":"`+arn+`","scheduleConfig":{"state":"DISABLED"},"clearWidgetIds":true}`)
	rep = budOKAPI(t, ts, bcmdAPI, "GetScheduledReport", `{"arn":"`+arn+`"}`)["scheduledReport"].(map[string]any)
	if rep["scheduleConfig"].(map[string]any)["state"] != "DISABLED" || rep["scheduleConfig"].(map[string]any)["scheduleExpression"] != "rate(1 day)" || rep["widgetIds"] != nil {
		t.Fatalf("updated: %v", rep)
	}
	if code := budErrAPI(t, ts, bcmdAPI, "UpdateScheduledReport", `{"arn":"`+arn+`","widgetIds":["`+bcmdWidgetID+`"],"clearWidgetIds":true}`, http.StatusBadRequest); code != "ValidationException" {
		t.Fatalf("set and clear: %s", code)
	}

	// Once the dashboard is gone the report is unhealthy and is not triggered.
	budOKAPI(t, ts, bcmdAPI, "DeleteDashboard", `{"arn":"`+dash+`"}`)
	run := budOKAPI(t, ts, bcmdAPI, "ExecuteScheduledReport", `{"arn":"`+arn+`"}`)
	health := run["healthStatus"].(map[string]any)
	if run["executionTriggered"] != false || health["statusCode"] != "UNHEALTHY" || health["statusReasons"].([]any)[0] != "DASHBOARD_NOT_FOUND" {
		t.Fatalf("unhealthy run: %v", run)
	}
	list := budOKAPI(t, ts, bcmdAPI, "ListScheduledReports", `{}`)["scheduledReports"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["state"] != "DISABLED" {
		t.Fatalf("list: %v", list)
	}
	budOKAPI(t, ts, bcmdAPI, "TagResource", `{"resourceArn":"`+arn+`","resourceTags":[{"key":"k","value":"v"}]}`)
	budOKAPI(t, ts, bcmdAPI, "DeleteScheduledReport", `{"arn":"`+arn+`"}`)
	if code := budErrAPI(t, ts, bcmdAPI, "ListTagsForResource", `{"resourceArn":"`+arn+`"}`, http.StatusNotFound); code != "ResourceNotFoundException" {
		t.Fatalf("deleted report tags: %s", code)
	}
}

func TestBCMDashboardsStage1ByteBudget(t *testing.T) {
	srv, ts := newBudTestServer(t)
	arn := budOKAPI(t, ts, bcmdAPI, "CreateDashboard", `{"name":"Spend","widgets":[]}`)["arn"].(string)
	srv.bcmDashboards.mu.Lock()
	srv.bcmDashboards.bytes = bcmdMaxStoreBytes - 10
	srv.bcmDashboards.mu.Unlock()
	// Updates and tag changes are charged, not just creates.
	if code := budErrAPI(t, ts, bcmdAPI, "UpdateDashboard", `{"arn":"`+arn+`","name":"Spend","widgets":`+bcmdWidgetJSON(bcmdWidgetID, bcmdQuery)+`}`, http.StatusPaymentRequired); code != "ServiceQuotaExceededException" {
		t.Fatalf("update over budget: %s", code)
	}
	if code := budErrAPI(t, ts, bcmdAPI, "TagResource", `{"resourceArn":"`+arn+`","resourceTags":[{"key":"team","value":"finance-and-operations"}]}`, http.StatusPaymentRequired); code != "ServiceQuotaExceededException" {
		t.Fatalf("tag over budget: %s", code)
	}
	if code := budErrAPI(t, ts, bcmdAPI, "CreateDashboard", `{"name":"More","widgets":[]}`, http.StatusPaymentRequired); code != "ServiceQuotaExceededException" {
		t.Fatalf("create over budget: %s", code)
	}
}
