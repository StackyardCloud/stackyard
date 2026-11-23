package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// tsqSeed creates "db1"."cpu" with six DOUBLE records over two hosts, one a
// minute apart, plus a MULTI record.
func tsqSeed(t *testing.T, ts *httptest.Server) time.Time {
	t.Helper()
	tsmOK(t, ts, "CreateDatabase", `{"DatabaseName":"db1"}`)
	tsmOK(t, ts, "CreateTable", `{"DatabaseName":"db1","TableName":"cpu"}`)
	base := time.Now().Add(-10 * time.Minute).Truncate(time.Minute)
	var recs []string
	for i := 0; i < 6; i++ {
		host := []string{"a", "b"}[i%2]
		recs = append(recs, fmt.Sprintf(`{"Dimensions":[{"Name":"host","Value":"%s"}],"MeasureName":"usage","MeasureValue":"%d","MeasureValueType":"DOUBLE","Time":"%d"}`, host, (i+1)*10, base.Add(time.Duration(i)*time.Minute).UnixMilli()))
	}
	recs = append(recs, fmt.Sprintf(`{"Dimensions":[{"Name":"host","Value":"a"}],"MeasureName":"mem","MeasureValueType":"MULTI","MeasureValues":[{"Name":"used","Value":"512","Type":"BIGINT"},{"Name":"ok","Value":"true","Type":"BOOLEAN"}],"Time":"%d"}`, base.UnixMilli()))
	tsmOK(t, ts, "WriteRecords", `{"DatabaseName":"db1","TableName":"cpu","Records":[`+strings.Join(recs, ",")+`]}`)
	return base
}

func tsqRun(t *testing.T, ts *httptest.Server, sql string) ([]any, []any) {
	t.Helper()
	out := tsmOK(t, ts, "Query", fmt.Sprintf(`{"QueryString":%q}`, sql))
	return out["ColumnInfo"].([]any), out["Rows"].([]any)
}

func TestTimestreamQueryStage1SelectFilterAggregate(t *testing.T) {
	_, ts := newTimestreamTestServer(t)
	tsqSeed(t, ts)
	cols, rows := tsqRun(t, ts, `SELECT host, avg(measure_value::double) AS avg_usage, count(*) AS n, max(measure_value::double) FROM "db1"."cpu" WHERE measure_name = 'usage' AND time > ago(1h) GROUP BY host ORDER BY host`)
	if len(rows) != 2 || tsmDatum(rows[0], 0) != "a" || tsmDatum(rows[0], 1) != "30.0" || tsmDatum(rows[0], 2) != "3" || tsmDatum(rows[1], 3) != "60.0" {
		t.Fatalf("grouped rows: %v", rows)
	}
	if cols[1].(map[string]any)["Type"].(map[string]any)["ScalarType"] != "DOUBLE" || cols[2].(map[string]any)["Type"].(map[string]any)["ScalarType"] != "BIGINT" || cols[3].(map[string]any)["Name"] != "_col3" {
		t.Fatalf("column info: %v", cols)
	}
	// bin() buckets, BETWEEN, IN, LIKE, ORDER BY DESC and LIMIT.
	_, rows = tsqRun(t, ts, `SELECT bin(time, 2m) AS b, sum(measure_value::double) FROM "db1"."cpu" WHERE measure_name IN ('usage') AND host LIKE '%' AND time BETWEEN ago(1h) AND now() GROUP BY bin(time, 2m) ORDER BY b DESC LIMIT 2`)
	if len(rows) != 2 {
		t.Fatalf("binned rows: %v", rows)
	}
	// Timestamps use Timestream's nanosecond text format, e.g. 2026-01-02 15:04:00.000000000.
	if b, _ := tsmDatum(rows[0], 0).(string); len(b) != len(tsqTimeLayout) || !strings.Contains(b, ":00.000000000") {
		t.Fatalf("bin timestamp format: %q", b)
	}
	// Multi-measure attributes are columns; a scalar column is NULL on MULTI rows.
	_, rows = tsqRun(t, ts, `SELECT used, ok, measure_value::double FROM "db1"."cpu" WHERE measure_name = 'mem'`)
	if len(rows) != 1 || tsmDatum(rows[0], 0) != "512" || tsmDatum(rows[0], 1) != "true" || tsmDatum(rows[0], 2) != nil {
		t.Fatalf("multi row: %v", rows)
	}
	// A global aggregate over no rows still returns one row.
	_, rows = tsqRun(t, ts, `SELECT count(*) FROM "db1"."cpu" WHERE host = 'nobody'`)
	if len(rows) != 1 || tsmDatum(rows[0], 0) != "0" {
		t.Fatalf("empty count: %v", rows)
	}
}

func TestTimestreamQueryStage1PaginationAndCancel(t *testing.T) {
	_, ts := newTimestreamTestServer(t)
	tsqSeed(t, ts)
	sql := `SELECT time, host FROM "db1"."cpu" WHERE measure_name = 'usage' ORDER BY time`
	page := tsmOK(t, ts, "Query", fmt.Sprintf(`{"QueryString":%q,"MaxRows":4}`, sql))
	tok, _ := page["NextToken"].(string)
	if len(page["Rows"].([]any)) != 4 || tok == "" {
		t.Fatalf("first page: %v", page)
	}
	if code, _ := tsmFail(t, ts, "Query", fmt.Sprintf(`{"QueryString":"SELECT 1 FROM \"db1\".\"cpu\"","NextToken":%q}`, tok)); code != "ValidationException" {
		t.Fatalf("token with a different query: %s", code)
	}
	second := tsmOK(t, ts, "Query", fmt.Sprintf(`{"QueryString":%q,"MaxRows":4,"NextToken":%q}`, sql, tok))
	if len(second["Rows"].([]any)) != 2 || second["NextToken"] != nil || second["QueryId"] != page["QueryId"] {
		t.Fatalf("second page: %v", second)
	}
	third := tsmOK(t, ts, "Query", fmt.Sprintf(`{"QueryString":%q,"MaxRows":4}`, sql))
	tsmOK(t, ts, "CancelQuery", fmt.Sprintf(`{"QueryId":%q}`, third["QueryId"]))
	if code, _ := tsmFail(t, ts, "Query", fmt.Sprintf(`{"QueryString":%q,"NextToken":%q}`, sql, third["NextToken"])); code != "ValidationException" {
		t.Fatalf("token after cancel: %s", code)
	}
}

func TestTimestreamQueryStage1RejectsUnsupportedSQL(t *testing.T) {
	_, ts := newTimestreamTestServer(t)
	tsqSeed(t, ts)
	for name, sql := range map[string]string{
		"join":           `SELECT * FROM "db1"."cpu" JOIN "db1"."cpu" ON 1 = 1`,
		"subquery":       `SELECT * FROM (SELECT 1)`,
		"unknown column": `SELECT nope FROM "db1"."cpu"`,
		"missing table":  `SELECT * FROM "db1"."nope"`,
		"not grouped":    `SELECT host, measure_name, count(*) FROM "db1"."cpu" GROUP BY host`,
		"unknown func":   `SELECT approx_percentile(measure_value::double, 0.5) FROM "db1"."cpu"`,
		"unbound param":  `SELECT * FROM "db1"."cpu" WHERE time > @scheduled_runtime`,
		"insert":         `INSERT INTO "db1"."cpu" VALUES (1)`,
	} {
		if code, _ := tsmFail(t, ts, "Query", fmt.Sprintf(`{"QueryString":%q}`, sql)); code != "ValidationException" {
			t.Fatalf("%s: %s", name, code)
		}
	}
	prep := tsmOK(t, ts, "PrepareQuery", `{"QueryString":"SELECT host AS h, time FROM \"db1\".\"cpu\" WHERE time > @scheduled_runtime - 1h","ValidateOnly":true}`)
	cols := prep["Columns"].([]any)
	if len(cols) != 2 || cols[0].(map[string]any)["Aliased"] != true || cols[1].(map[string]any)["Type"].(map[string]any)["ScalarType"] != "TIMESTAMP" || len(prep["Parameters"].([]any)) != 1 {
		t.Fatalf("prepare: %v", prep)
	}
}

func TestTimestreamQueryStage1ScheduledQueryWritesTarget(t *testing.T) {
	srv, ts := newTimestreamTestServer(t)
	tsqSeed(t, ts)
	topic, err := srv.sns.CreateTopic("ts-notify", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.s3.CreateBucket("ts-errors"); err != nil {
		t.Fatal(err)
	}
	tsmOK(t, ts, "CreateTable", `{"DatabaseName":"db1","TableName":"rollup"}`)
	create := func(name, query string) (int, map[string]any) {
		return tsmCallAPI(t, ts, "CreateScheduledQuery", fmt.Sprintf(`{"Name":%q,"QueryString":%q,"ScheduleConfiguration":{"ScheduleExpression":"rate(1 hour)"},
			"NotificationConfiguration":{"SnsConfiguration":{"TopicArn":%q}},"ScheduledQueryExecutionRoleArn":"arn:aws:iam::123456789012:role/r",
			"ErrorReportConfiguration":{"S3Configuration":{"BucketName":"ts-errors"}},
			"TargetConfiguration":{"TimestreamConfiguration":{"DatabaseName":"db1","TableName":"rollup","TimeColumn":"t","DimensionMappings":[{"Name":"host","DimensionValueType":"VARCHAR"}],
			"MultiMeasureMappings":{"TargetMultiMeasureName":"avg","MultiMeasureAttributeMappings":[{"SourceColumn":"v","MeasureValueType":"DOUBLE"}]}}}}`, name, query, topic.ARN))
	}
	query := `SELECT host, bin(time, 1h) AS t, avg(measure_value::double) AS v FROM "db1"."cpu" WHERE measure_name = 'usage' AND time <= @scheduled_runtime GROUP BY host, bin(time, 1h)`
	status, out := create("hourly", query)
	if status != http.StatusOK {
		t.Fatalf("create: %d %v", status, out)
	}
	arn := out["Arn"].(string)
	if status, _ := create("hourly", query); status != http.StatusBadRequest {
		t.Fatalf("duplicate name: %d", status)
	}
	if status, out := create("bad-param", `SELECT * FROM "db1"."cpu" WHERE time > @other`); status != http.StatusBadRequest {
		t.Fatalf("unknown parameter: %d %v", status, out)
	}
	tsmOK(t, ts, "ExecuteScheduledQuery", fmt.Sprintf(`{"ScheduledQueryArn":%q,"InvocationTime":%d}`, arn, time.Now().Unix()))
	desc := tsmOK(t, ts, "DescribeScheduledQuery", fmt.Sprintf(`{"ScheduledQueryArn":%q}`, arn))["ScheduledQuery"].(map[string]any)
	last := desc["LastRunSummary"].(map[string]any)
	// One row per host per hour bin: 2 or 4 depending on whether the seeded
	// ten minutes cross an hour boundary.
	stats := last["ExecutionStats"].(map[string]any)
	if last["RunStatus"] != "MANUAL_TRIGGER_SUCCESS" || stats["RecordsIngested"] != stats["QueryResultRows"] || stats["RecordsIngested"].(float64) < 2 {
		t.Fatalf("last run: %v", desc)
	}
	_, rows := tsqRun(t, ts, `SELECT host, v FROM "db1"."rollup" ORDER BY host`)
	if float64(len(rows)) != stats["RecordsIngested"] || tsmDatum(rows[0], 0) != "a" {
		t.Fatalf("rollup rows: %v", rows)
	}
	tsmOK(t, ts, "UpdateScheduledQuery", fmt.Sprintf(`{"ScheduledQueryArn":%q,"State":"DISABLED"}`, arn))
	list := tsmOK(t, ts, "ListScheduledQueries", `{}`)["ScheduledQueries"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["State"] != "DISABLED" {
		t.Fatalf("list: %v", list)
	}
	tsmOK(t, ts, "DeleteScheduledQuery", fmt.Sprintf(`{"ScheduledQueryArn":%q}`, arn))
	if code, _ := tsmFail(t, ts, "DescribeScheduledQuery", fmt.Sprintf(`{"ScheduledQueryArn":%q}`, arn)); code != "ResourceNotFoundException" {
		t.Fatalf("after delete: %s", code)
	}
}

func TestTimestreamQueryStage1AccountSettings(t *testing.T) {
	_, ts := newTimestreamTestServer(t)
	if code, _ := tsmFail(t, ts, "UpdateAccountSettings", `{"QueryCompute":{"ComputeMode":"PROVISIONED","ProvisionedCapacity":{"TargetQueryTCU":8}}}`); code != "ValidationException" {
		t.Fatalf("provisioned without COMPUTE_UNITS: %s", code)
	}
	out := tsmOK(t, ts, "UpdateAccountSettings", `{"QueryPricingModel":"COMPUTE_UNITS","MaxQueryTCU":16,"QueryCompute":{"ComputeMode":"PROVISIONED","ProvisionedCapacity":{"TargetQueryTCU":8}}}`)
	if out["QueryCompute"].(map[string]any)["ProvisionedCapacity"].(map[string]any)["ActiveQueryTCU"] != float64(8) {
		t.Fatalf("settings: %v", out)
	}
	if got := tsmOK(t, ts, "DescribeAccountSettings", `{}`)["MaxQueryTCU"]; got != float64(16) {
		t.Fatalf("describe: %v", got)
	}
}

func TestTimestreamQueryStage1BoundsHostileQueries(t *testing.T) {
	_, ts := newTimestreamTestServer(t)
	tsqSeed(t, ts)
	// Deep nesting is refused instead of exhausting the stack.
	deep := `SELECT * FROM "db1"."cpu" WHERE ` + strings.Repeat("(", 500) + "1 = 1" + strings.Repeat(")", 500)
	if code, _ := tsmFail(t, ts, "Query", fmt.Sprintf(`{"QueryString":%q}`, deep)); code != "ValidationException" {
		t.Fatalf("deep nesting: %s", code)
	}
	if code, _ := tsmFail(t, ts, "Query", `{"QueryString":"SELECT * FROM \"db1\".\"cpu\" WHERE time > ago(99999999999d)"}`); code != "ValidationException" {
		t.Fatalf("interval overflow: %s", code)
	}
	if _, err := tsqParse(`SELECT * FROM "a"."b" WHERE x > ago(9999999d)`); err == nil {
		t.Fatal("an interval that overflows int64 was accepted")
	}
	// LIKE work is budgeted per query.
	env := &tsqEnv{types: map[string]string{"host": "VARCHAR"}}
	like := &tsqNode{op: "like", args: []*tsqNode{{op: "lit", val: tsqVal{typ: "VARCHAR", s: strings.Repeat("a", 4000)}}, {op: "lit", val: tsqVal{typ: "VARCHAR", s: strings.Repeat("%a", 2000)}}}}
	for i := 0; i < 10 && !env.over; i++ {
		env.eval(like, nil)
	}
	if !env.over {
		t.Fatal("the LIKE work budget never tripped")
	}
	if !tsqLike("web-01", "web-__") || tsqLike("web-1", "api%") || !tsqLike("anything", "%") {
		t.Fatal("LIKE semantics")
	}
}
