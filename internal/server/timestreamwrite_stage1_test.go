package server

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTimestreamWriteStage1BatchLoad(t *testing.T) {
	srv, ts := newTimestreamTestServer(t)
	if _, err := srv.s3.CreateBucket("ts-load"); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	csv := fmt.Sprintf("time,host,cpu\n%d,h1,1.5\n%d,h2,2.5\n1000,h3,3.5\n", now, now)
	if _, err := srv.s3.PutObject("ts-load", "in/data.csv", "text/csv", []byte(csv), nil, "", ""); err != nil {
		t.Fatal(err)
	}
	tsmOK(t, ts, "CreateDatabase", `{"DatabaseName":"db1"}`)
	tsmOK(t, ts, "CreateTable", `{"DatabaseName":"db1","TableName":"metrics"}`)
	task := func(bucket, token string) (int, map[string]any) {
		return tsmCallAPI(t, ts, "CreateBatchLoadTask", fmt.Sprintf(`{"ClientToken":%q,"TargetDatabaseName":"db1","TargetTableName":"metrics",
			"DataSourceConfiguration":{"DataSourceS3Configuration":{"BucketName":%q,"ObjectKeyPrefix":"in/"},"DataFormat":"CSV"},
			"ReportConfiguration":{"ReportS3Configuration":{"BucketName":"ts-load","ObjectKeyPrefix":"reports"}},
			"DataModelConfiguration":{"DataModel":{"TimeColumn":"time","TimeUnit":"MILLISECONDS","DimensionMappings":[{"SourceColumn":"host"}],
			"MultiMeasureMappings":{"TargetMultiMeasureName":"m","MultiMeasureAttributeMappings":[{"SourceColumn":"cpu","MeasureValueType":"DOUBLE"}]}}}}`, token, bucket))
	}
	if status, out := task("no-such-bucket", "tok-0"); status != http.StatusBadRequest || out["__type"] != "ValidationException" {
		t.Fatalf("missing source bucket: %d %v", status, out)
	}
	status, out := task("ts-load", "tok-1")
	if status != http.StatusOK {
		t.Fatalf("create: %d %v", status, out)
	}
	id := out["TaskId"].(string)
	if _, again := task("ts-load", "tok-1"); again["TaskId"] != id {
		t.Fatalf("ClientToken reuse made a second task: %v", again)
	}
	// Tasks settle on read: IN_PROGRESS, then PROGRESS_STOPPED for the row
	// outside the retention window.
	var desc map[string]any
	for _, want := range []string{"IN_PROGRESS", "PROGRESS_STOPPED"} {
		desc = tsmOK(t, ts, "DescribeBatchLoadTask", fmt.Sprintf(`{"TaskId":%q}`, id))["BatchLoadTaskDescription"].(map[string]any)
		if desc["TaskStatus"] != want {
			t.Fatalf("status %v, want %s", desc["TaskStatus"], want)
		}
	}
	progress := desc["ProgressReport"].(map[string]any)
	if progress["RecordsIngested"] != float64(2) || progress["RecordIngestionFailures"] != float64(1) {
		t.Fatalf("progress: %v", progress)
	}
	report, err := srv.s3.GetObject("ts-load", "reports/"+id+"/error-report.csv")
	if err != nil || !strings.Contains(string(report.Body), "data.csv,4,") {
		t.Fatalf("error report: %v %q", err, report.Body)
	}
	rows := tsmOK(t, ts, "Query", `{"QueryString":"SELECT host, cpu FROM \"db1\".\"metrics\" ORDER BY host"}`)["Rows"].([]any)
	if len(rows) != 2 || tsmDatum(rows[1], 1) != "2.5" {
		t.Fatalf("loaded rows: %v", rows)
	}
	tsmOK(t, ts, "ResumeBatchLoadTask", fmt.Sprintf(`{"TaskId":%q}`, id))
	if code, _ := tsmFail(t, ts, "ResumeBatchLoadTask", fmt.Sprintf(`{"TaskId":%q}`, id)); code != "ValidationException" {
		t.Fatalf("resume a finished task: %s", code)
	}
	listed := tsmOK(t, ts, "ListBatchLoadTasks", `{"TaskStatus":"SUCCEEDED"}`)["BatchLoadTasks"].([]any)
	if len(listed) != 1 {
		t.Fatalf("list: %v", listed)
	}
}
