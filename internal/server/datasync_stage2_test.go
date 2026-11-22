package server

import (
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
)

// dsyS3Pair makes source and destination S3 locations over two buckets and
// fills the source with objects.
func dsyS3Pair(t *testing.T, srv *Server, ts *httptest.Server, keys ...string) (src, dst string) {
	t.Helper()
	for _, b := range []string{"dsy-src", "dsy-dst"} {
		if _, err := srv.s3.CreateBucket(b); err != nil {
			t.Fatal(err)
		}
	}
	for _, k := range keys {
		if _, err := srv.s3.PutObject("dsy-src", "in/"+k, "text/plain", []byte("data-"+k), nil, "", ""); err != nil {
			t.Fatal(err)
		}
	}
	src = dsyOK(t, ts, "CreateLocationS3", `{"S3BucketArn":"arn:aws:s3:::dsy-src","Subdirectory":"in",`+dsyTestRole+`}`)["LocationArn"].(string)
	dst = dsyOK(t, ts, "CreateLocationS3", `{"S3BucketArn":"arn:aws:s3:::dsy-dst","Subdirectory":"out",`+dsyTestRole+`}`)["LocationArn"].(string)
	return src, dst
}

// dsyRun starts an execution and reads it until it finishes.
func dsyRun(t *testing.T, ts *httptest.Server, task, body string) map[string]any {
	t.Helper()
	arn := dsyOK(t, ts, "StartTaskExecution", `{"TaskArn":"`+task+`"`+body+`}`)["TaskExecutionArn"].(string)
	var got map[string]any
	for i := 0; i < 10; i++ {
		got = dsyOK(t, ts, "DescribeTaskExecution", `{"TaskExecutionArn":"`+arn+`"}`)
		if got["Status"] == "SUCCESS" || got["Status"] == "ERROR" {
			return got
		}
	}
	t.Fatalf("execution never finished: %v", got)
	return nil
}

func dsyKeys(t *testing.T, srv *Server, bucket string) []string {
	t.Helper()
	objs, err := srv.s3.ListObjects(bucket, "")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, o := range objs {
		out = append(out, o.Key)
	}
	sort.Strings(out)
	return out
}

func TestDataSyncStage2S3CopyWithFilters(t *testing.T) {
	srv, ts := newDataSyncTestServer(t)
	src, dst := dsyS3Pair(t, srv, ts, "a.txt", "dir/b.txt", "dir/c.log", "skip/d.txt")
	task := dsyOK(t, ts, "CreateTask", `{"SourceLocationArn":"`+src+`","DestinationLocationArn":"`+dst+`","Excludes":[{"FilterType":"SIMPLE_PATTERN","Value":"/skip|*.log"}]}`)["TaskArn"].(string)
	if got := dsyOK(t, ts, "DescribeTask", `{"TaskArn":"`+task+`"}`); got["Status"] != "AVAILABLE" || got["Options"].(map[string]any)["TransferMode"] != "CHANGED" {
		t.Fatalf("describe task defaults: %v", got)
	}

	got := dsyRun(t, ts, task, "")
	if got["Status"] != "SUCCESS" || got["FilesTransferred"] != float64(2) || got["FilesVerified"] != float64(2) {
		t.Fatalf("first run: %v", got)
	}
	if keys := dsyKeys(t, srv, "dsy-dst"); strings.Join(keys, ",") != "out/a.txt,out/dir/b.txt" {
		t.Fatalf("copied keys: %v", keys)
	}
	// Unchanged files are skipped on the next run (TransferMode CHANGED).
	if got := dsyRun(t, ts, task, ""); got["FilesTransferred"] != float64(0) || got["FilesSkipped"] != float64(2) {
		t.Fatalf("second run: %v", got)
	}
	// Includes narrow the run; REMOVE deletes destination files no longer in
	// the source.
	if _, err := srv.s3.PutObject("dsy-dst", "out/stale.txt", "text/plain", []byte("old"), nil, "", ""); err != nil {
		t.Fatal(err)
	}
	got = dsyRun(t, ts, task, `,"OverrideOptions":{"TransferMode":"ALL","PreserveDeletedFiles":"REMOVE"},"Includes":[{"FilterType":"SIMPLE_PATTERN","Value":"/dir|/stale.txt"}]`)
	if got["FilesTransferred"] != float64(1) || got["FilesDeleted"] != float64(1) {
		t.Fatalf("include + remove run: %v", got)
	}
	if keys := dsyKeys(t, srv, "dsy-dst"); strings.Join(keys, ",") != "out/a.txt,out/dir/b.txt" {
		t.Fatalf("after remove: %v", keys)
	}
}

func TestDataSyncStage2TaskRulesAndExecutionLifecycle(t *testing.T) {
	srv, ts := newDataSyncTestServer(t)
	src, dst := dsyS3Pair(t, srv, ts, "a.txt")
	pair := `"SourceLocationArn":"` + src + `","DestinationLocationArn":"` + dst + `"`
	for _, body := range map[string]string{
		"same location":   `{"SourceLocationArn":"` + src + `","DestinationLocationArn":"` + src + `"}`,
		"atime vs mtime":  `{` + pair + `,"Options":{"Atime":"NONE","Mtime":"PRESERVE"}}`,
		"bad pattern":     `{` + pair + `,"Excludes":[{"FilterType":"SIMPLE_PATTERN","Value":"no-slash"}]}`,
		"too frequent":    `{` + pair + `,"Schedule":{"ScheduleExpression":"rate(5 minutes)"}}`,
		"bad expression":  `{` + pair + `,"Schedule":{"ScheduleExpression":"every day"}}`,
		"unknown enum":    `{` + pair + `,"Options":{"VerifyMode":"SOMETIMES"}}`,
		"missing source":  `{"SourceLocationArn":"arn:aws:datasync:us-east-1:123456789012:location/loc-0123456789abcdef0","DestinationLocationArn":"` + dst + `"}`,
		"bad log group":   `{` + pair + `,"CloudWatchLogGroupArn":"nope"}`,
		"manifest source": `{` + pair + `,"ManifestConfig":{"Action":"TRANSFER"}}`,
	} {
		dsyFail(t, ts, "CreateTask", body)
	}
	task := dsyOK(t, ts, "CreateTask", `{`+pair+`,"Schedule":{"ScheduleExpression":"rate(1 hour)"}}`)["TaskArn"].(string)

	// A second start queues behind the first; with queueing off it's refused.
	first := dsyOK(t, ts, "StartTaskExecution", `{"TaskArn":"`+task+`"}`)["TaskExecutionArn"].(string)
	second := dsyOK(t, ts, "StartTaskExecution", `{"TaskArn":"`+task+`"}`)["TaskExecutionArn"].(string)
	if got := dsyOK(t, ts, "DescribeTask", `{"TaskArn":"`+task+`"}`); got["Status"] != "RUNNING" || got["CurrentTaskExecutionArn"] != first {
		t.Fatalf("running task: %v", got)
	}
	dsyFail(t, ts, "StartTaskExecution", `{"TaskArn":"`+task+`","OverrideOptions":{"TaskQueueing":"DISABLED"}}`)

	// Only the bandwidth limit can change mid-run.
	dsyFail(t, ts, "UpdateTaskExecution", `{"TaskExecutionArn":"`+first+`","Options":{"VerifyMode":"NONE"}}`)
	dsyOK(t, ts, "UpdateTaskExecution", `{"TaskExecutionArn":"`+first+`","Options":{"BytesPerSecond":1048576}}`)

	// Cancelling ends the execution with OpCancelled and starts the queued one.
	dsyOK(t, ts, "CancelTaskExecution", `{"TaskExecutionArn":"`+first+`"}`)
	got := dsyOK(t, ts, "DescribeTaskExecution", `{"TaskExecutionArn":"`+first+`"}`)
	if got["Status"] != "ERROR" || got["Result"].(map[string]any)["ErrorCode"] != "OpCancelled" {
		t.Fatalf("cancelled execution: %v", got)
	}
	dsyFail(t, ts, "CancelTaskExecution", `{"TaskExecutionArn":"`+first+`"}`)
	if got := dsyOK(t, ts, "DescribeTask", `{"TaskArn":"`+task+`"}`); got["CurrentTaskExecutionArn"] != second {
		t.Fatalf("queued execution should run next: %v", got)
	}
	if list := dsyOK(t, ts, "ListTaskExecutions", `{"TaskArn":"`+task+`"}`)["TaskExecutions"].([]any); len(list) != 2 {
		t.Fatalf("executions: %v", list)
	}
	dsyOK(t, ts, "UpdateTask", `{"TaskArn":"`+task+`","Name":"renamed","Options":{"LogLevel":"BASIC"}}`)
	got = dsyOK(t, ts, "DescribeTask", `{"TaskArn":"`+task+`"}`)
	if got["Name"] != "renamed" || got["Options"].(map[string]any)["LogLevel"] != "BASIC" || got["Options"].(map[string]any)["VerifyMode"] != "POINT_IN_TIME_CONSISTENT" {
		t.Fatalf("updated task: %v", got)
	}
	dsyOK(t, ts, "DeleteTask", `{"TaskArn":"`+task+`"}`)
	dsyFail(t, ts, "DescribeTaskExecution", `{"TaskExecutionArn":"`+second+`"}`)
}
