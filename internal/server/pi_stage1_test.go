package server

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

const piT0 = 1767225600 // 2026-01-01T00:00:00Z

func piReq(id string, extra string) string {
	return `{"ServiceType":"RDS","Identifier":"` + id + `"` + extra + `}`
}

func TestPIStage1IdentifierChecks(t *testing.T) {
	srv, ts := newPITestServer(t)
	id := piInstance(t, srv, "orders-db", "postgres")
	doc := piInstance(t, srv, "docs-db", "docdb")
	if code := piErrCode(t, ts, "GetResourceMetadata", piReq("db-NOPE", "")); code != "NotAuthorizedException" {
		t.Fatalf("unknown identifier: %s", code)
	}
	if code := piErrCode(t, ts, "GetResourceMetadata", `{"ServiceType":"DOCDB","Identifier":"`+id+`"}`); code != "NotAuthorizedException" {
		t.Fatalf("RDS instance as DOCDB: %s", code)
	}
	out := piOK(t, ts, "GetResourceMetadata", `{"ServiceType":"DOCDB","Identifier":"`+doc+`"}`)
	if out["Identifier"] != doc {
		t.Fatalf("metadata: %v", out)
	}
}

func TestPIStage1GetResourceMetrics(t *testing.T) {
	srv, ts := newPITestServer(t)
	id := piInstance(t, srv, "orders-db", "postgres")
	window := fmt.Sprintf(`,"StartTime":%d,"EndTime":%d`, piT0, piT0+3600)
	q := piReq(id, window+`,"MetricQueries":[{"Metric":"db.load.avg","GroupBy":{"Group":"db.wait_event","Limit":3}},{"Metric":"os.cpuUtilization.total.avg"}]`)
	a, b := piOK(t, ts, "GetResourceMetrics", q), piOK(t, ts, "GetResourceMetrics", q)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("repeated calls disagree")
	}
	list := a["MetricList"].([]any)
	if len(list) != 5 { // load total + 3 wait events + cpu
		t.Fatalf("metric list: %d", len(list))
	}
	points := list[0].(map[string]any)["DataPoints"].([]any)
	if len(points) != 60 || a["AlignedStartTime"] != float64(piT0) { // 1 hour picks 60s periods
		t.Fatalf("points %d, aligned start %v", len(points), a["AlignedStartTime"])
	}
	if first := points[0].(map[string]any)["Timestamp"]; first != float64(piT0+60) {
		t.Fatalf("END_TIME alignment: first timestamp %v", first)
	}
	if k := list[1].(map[string]any)["Key"].(map[string]any); k["Dimensions"].(map[string]any)["db.wait_event.name"] == nil {
		t.Fatalf("grouped key: %v", k)
	}
	for _, v := range points {
		if cpu := v.(map[string]any)["Value"].(float64); cpu < 0 {
			t.Fatalf("negative value %v", cpu)
		}
	}
	page := piOK(t, ts, "GetResourceMetrics", strings.TrimSuffix(q, "}")+`,"MaxResults":2}`)
	if len(page["MetricList"].([]any)) != 2 || page["NextToken"] == nil {
		t.Fatalf("paging: %v", page["NextToken"])
	}
	rest := piOK(t, ts, "GetResourceMetrics", strings.TrimSuffix(q, "}")+`,"MaxResults":25,"NextToken":"`+page["NextToken"].(string)+`"}`)
	if len(rest["MetricList"].([]any)) != 3 {
		t.Fatalf("second page: %d", len(rest["MetricList"].([]any)))
	}

	for name, body := range map[string]string{
		"bad period":     piReq(id, window+`,"PeriodInSeconds":7,"MetricQueries":[{"Metric":"db.load.avg"}]`),
		"reversed":       piReq(id, fmt.Sprintf(`,"StartTime":%d,"EndTime":%d,"MetricQueries":[{"Metric":"db.load.avg"}]`, piT0+10, piT0)),
		"too many":       piReq(id, fmt.Sprintf(`,"StartTime":%d,"EndTime":%d,"PeriodInSeconds":1,"MetricQueries":[{"Metric":"db.load.avg"}]`, piT0, piT0+86400)),
		"unknown metric": piReq(id, window+`,"MetricQueries":[{"Metric":"db.nope.avg"}]`),
		"no statistic":   piReq(id, window+`,"MetricQueries":[{"Metric":"db.load"}]`),
		"group os":       piReq(id, window+`,"MetricQueries":[{"Metric":"os.memory.free.avg","GroupBy":{"Group":"db.user"}}]`),
		"bad group":      piReq(id, window+`,"MetricQueries":[{"Metric":"db.load.avg","GroupBy":{"Group":"db.query"}}]`),
		"bad token":      piReq(id, window+`,"MetricQueries":[{"Metric":"db.load.avg"}],"NextToken":"xyz"`),
	} {
		if code := piErrCode(t, ts, "GetResourceMetrics", body); code != "InvalidArgumentException" {
			t.Errorf("%s: %s", name, code)
		}
	}
}

func TestPIStage1DescribeDimensionKeys(t *testing.T) {
	srv, ts := newPITestServer(t)
	id := piInstance(t, srv, "orders-db", "postgres")
	window := fmt.Sprintf(`,"StartTime":%d,"EndTime":%d,"Metric":"db.load.avg"`, piT0, piT0+3600)
	out := piOK(t, ts, "DescribeDimensionKeys", piReq(id, window+`,"GroupBy":{"Group":"db.sql","Dimensions":["db.sql.id","db.sql.statement"],"Limit":3},"PartitionBy":{"Group":"db.wait_event_type"},"AdditionalMetrics":["db.sql.stats.calls_per_sec.avg"]`))
	keys := out["Keys"].([]any)
	if len(keys) != 3 || len(out["PartitionKeys"].([]any)) != 5 {
		t.Fatalf("keys %d, partition keys %d", len(keys), len(out["PartitionKeys"].([]any)))
	}
	prev := math.Inf(1)
	for _, k := range keys {
		key := k.(map[string]any)
		total := key["Total"].(float64)
		if total > prev {
			t.Fatal("keys aren't sorted by load")
		}
		prev = total
		if len(key["Dimensions"].(map[string]any)) != 2 || key["AdditionalMetrics"] == nil {
			t.Fatalf("key: %v", key)
		}
		sum := 0.0
		for _, p := range key["Partitions"].([]any) {
			sum += p.(float64)
		}
		if math.Abs(sum-total) > 0.01 {
			t.Fatalf("partitions sum %v, total %v", sum, total)
		}
	}
	filtered := piOK(t, ts, "DescribeDimensionKeys", piReq(id, window+`,"GroupBy":{"Group":"db.wait_event"},"Filter":{"db.wait_event.type":"IO"}`))
	if f := filtered["Keys"].([]any); len(f) != 1 || f[0].(map[string]any)["Dimensions"].(map[string]any)["db.wait_event.name"] != "IO:DataFileRead" {
		t.Fatalf("filter: %v", f)
	}
	for name, body := range map[string]string{
		"not load":   strings.Replace(piReq(id, window+`,"GroupBy":{"Group":"db.sql"}`), "db.load.avg", "os.memory.free.avg", 1),
		"bad group":  piReq(id, window+`,"GroupBy":{"Group":"db.query"}`),
		"bad dim":    piReq(id, window+`,"GroupBy":{"Group":"db.sql","Dimensions":["db.user.name"]}`),
		"bad filter": piReq(id, window+`,"GroupBy":{"Group":"db.sql"},"Filter":{"db.user.name":"app"}`),
	} {
		if code := piErrCode(t, ts, "DescribeDimensionKeys", body); code != "InvalidArgumentException" {
			t.Errorf("%s: %s", name, code)
		}
	}
}

func TestPIStage1CatalogsAndDetails(t *testing.T) {
	srv, ts := newPITestServer(t)
	id := piInstance(t, srv, "orders-db", "postgres")
	doc := piInstance(t, srv, "docs-db", "docdb")
	details := piOK(t, ts, "GetDimensionKeyDetails", piReq(id, `,"Group":"db.sql","GroupIdentifier":"A1B2C3D4E5F60718"`))
	if d := details["Dimensions"].([]any)[0].(map[string]any); d["Dimension"] != "db.sql.statement" || !strings.HasPrefix(d["Value"].(string), "SELECT") || d["Status"] != "AVAILABLE" {
		t.Fatalf("details: %v", d)
	}
	piOK(t, ts, "GetDimensionKeyDetails", `{"ServiceType":"DOCDB","Identifier":"`+doc+`","Group":"db.query","GroupIdentifier":"Q1A2B3C4D5E6F708"}`)
	if code := piErrCode(t, ts, "GetDimensionKeyDetails", piReq(id, `,"Group":"db.sql","GroupIdentifier":"NOPE"`)); code != "InvalidArgumentException" {
		t.Fatalf("unknown key: %s", code)
	}
	if code := piErrCode(t, ts, "GetDimensionKeyDetails", piReq(id, `,"Group":"db.user","GroupIdentifier":"app"`)); code != "InvalidArgumentException" {
		t.Fatalf("unsupported group: %s", code)
	}
	metrics := piOK(t, ts, "ListAvailableResourceMetrics", piReq(id, `,"MetricTypes":["os","db"],"MaxResults":5`))
	if len(metrics["Metrics"].([]any)) != 5 || metrics["NextToken"] == nil {
		t.Fatalf("metrics page: %v", metrics)
	}
	if code := piErrCode(t, ts, "ListAvailableResourceMetrics", piReq(id, `,"MetricTypes":["disk"]`)); code != "InvalidArgumentException" {
		t.Fatalf("bad metric type: %s", code)
	}
	dims := piOK(t, ts, "ListAvailableResourceDimensions", piReq(id, `,"Metrics":["db.load"]`))
	if g := dims["MetricDimensions"].([]any)[0].(map[string]any)["Groups"].([]any); len(g) != len(piRDSGroups) {
		t.Fatalf("groups: %d", len(g))
	}
}

func TestPIStage1AnalysisReportsAndTags(t *testing.T) {
	srv, ts := newPITestServer(t)
	id := piInstance(t, srv, "orders-db", "postgres")
	if code := piErrCode(t, ts, "CreatePerformanceAnalysisReport", piReq(id, fmt.Sprintf(`,"StartTime":%d,"EndTime":%d`, piT0, piT0+60))); code != "InvalidArgumentException" {
		t.Fatalf("short report window: %s", code)
	}
	if code := piErrCode(t, ts, "CreatePerformanceAnalysisReport", piReq(id, fmt.Sprintf(`,"StartTime":%d,"Tags":[{"Key":"aws:x","Value":"y"}]`, piT0))); code != "InvalidArgumentException" {
		t.Fatalf("aws: tag: %s", code)
	}
	rid := piOK(t, ts, "CreatePerformanceAnalysisReport", piReq(id, fmt.Sprintf(`,"StartTime":%d,"EndTime":%d,"Tags":[{"Key":"team","Value":"db"}]`, piT0, piT0+7200)))["AnalysisReportId"].(string)
	get := piReq(id, `,"AnalysisReportId":"`+rid+`","TextFormat":"MARKDOWN"`)
	if r := piOK(t, ts, "GetPerformanceAnalysisReport", get)["AnalysisReport"].(map[string]any); r["Status"] != "RUNNING" || r["Insights"] != nil {
		t.Fatalf("first read: %v", r)
	}
	r := piOK(t, ts, "GetPerformanceAnalysisReport", get)["AnalysisReport"].(map[string]any)
	insight := r["Insights"].([]any)[0].(map[string]any)
	if r["Status"] != "SUCCEEDED" || !strings.HasPrefix(insight["Description"].(string), "**") {
		t.Fatalf("second read: %v", r)
	}
	plain := piOK(t, ts, "GetPerformanceAnalysisReport", piReq(id, `,"AnalysisReportId":"`+rid+`","TextFormat":"PLAIN_TEXT"`))["AnalysisReport"].(map[string]any)
	if strings.Contains(plain["Insights"].([]any)[0].(map[string]any)["Description"].(string), "**") {
		t.Fatal("plain text report has markdown")
	}
	list := piOK(t, ts, "ListPerformanceAnalysisReports", piReq(id, `,"ListTags":true`))["AnalysisReports"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["Tags"] == nil {
		t.Fatalf("list: %v", list)
	}
	recs := piOK(t, ts, "ListPerformanceAnalysisReportRecommendations", piReq(id, `,"AnalysisReportId":"`+rid+`","RecommendationIds":["`+rid+`-rec-2"]`))["Recommendations"].([]any)
	if len(recs) != 1 {
		t.Fatalf("recommendations filter: %v", recs)
	}

	arn := fmt.Sprintf("arn:aws:pi:us-east-1:123456789012:perf-reports/RDS/%s/%s", id, rid)
	tagBody := func(extra string) string {
		b, _ := json.Marshal(map[string]any{"ServiceType": "RDS", "ResourceARN": arn})
		return strings.TrimSuffix(string(b), "}") + extra + "}"
	}
	piOK(t, ts, "TagResource", tagBody(`,"Tags":[{"Key":"env","Value":"dev"}]`))
	if tags := piOK(t, ts, "ListTagsForResource", tagBody(""))["Tags"].([]any); len(tags) != 2 {
		t.Fatalf("tags: %v", tags)
	}
	piOK(t, ts, "UntagResource", tagBody(`,"TagKeys":["team"]`))
	if tags := piOK(t, ts, "ListTagsForResource", tagBody(""))["Tags"].([]any); len(tags) != 1 {
		t.Fatalf("after untag: %v", tags)
	}
	if code := piErrCode(t, ts, "ListTagsForResource", strings.Replace(tagBody(""), `"RDS"`, `"DOCDB"`, 1)); code != "InvalidArgumentException" {
		t.Fatalf("service type mismatch: %s", code)
	}

	piOK(t, ts, "DeletePerformanceAnalysisReport", piReq(id, `,"AnalysisReportId":"`+rid+`"`))
	if code := piErrCode(t, ts, "GetPerformanceAnalysisReport", get); code != "InvalidArgumentException" {
		t.Fatalf("deleted report: %s", code)
	}
}
