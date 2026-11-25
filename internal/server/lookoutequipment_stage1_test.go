package server

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

const lkeTestRole = "arn:aws:iam::123456789012:role/lookout"

func lkeAwait(t *testing.T, ts *httptest.Server, op, body, key, want string) map[string]any {
	t.Helper()
	for i := 0; i < 4; i++ {
		out := lkeOK(t, ts, op, body)
		if out[key] == want {
			return out
		}
	}
	t.Fatalf("%s %s never reached %s=%s", op, body, key, want)
	return nil
}

func TestLookoutEquipmentLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	if _, err := srv.s3.CreateBucket("lke-data"); err != nil {
		t.Fatal(err)
	}
	csv := "Timestamp,temp,pressure\n2024-01-01T00:00:00Z,1.5,3\n2024-01-01T00:05:00Z,,4\n2024-01-01T00:05:00Z,abc,5\n"
	if _, err := srv.s3.PutObject("lke-data", "pump/pump1/sensors.csv", "text/csv", []byte(csv), nil, "", ""); err != nil {
		t.Fatal(err)
	}

	lkeOK(t, ts, "CreateDataset", `{"DatasetName":"pump","ClientToken":"a","Tags":[{"Key":"env","Value":"dev"}]}`)
	if code := lkeErrCode(t, ts, "CreateDataset", `{"DatasetName":"pump","ClientToken":"b"}`); code != "ConflictException" {
		t.Fatalf("duplicate: %s", code)
	}
	// Training needs ingested data.
	if code := lkeErrCode(t, ts, "CreateModel", `{"ModelName":"m","DatasetName":"pump","ClientToken":"c"}`); code != "ValidationException" {
		t.Fatalf("model before ingestion: %s", code)
	}
	ingest := `{"DatasetName":"pump","IngestionInputConfiguration":{"S3InputConfiguration":{"Bucket":"lke-data","Prefix":"pump/"}},"RoleArn":"` + lkeTestRole + `","ClientToken":"d"}`
	job := lkeOK(t, ts, "StartDataIngestionJob", ingest)["JobId"].(string)
	if code := lkeErrCode(t, ts, "StartDataIngestionJob", ingest); code != "ConflictException" {
		t.Fatalf("second ingestion while one runs: %s", code)
	}
	done := lkeAwait(t, ts, "DescribeDataIngestionJob", `{"JobId":"`+job+`"}`, "Status", "SUCCESS")
	q := done["DataQualitySummary"].(map[string]any)
	if q["MissingSensorData"].(map[string]any)["TotalNumberOfMissingValues"] != float64(1) ||
		q["InvalidSensorData"].(map[string]any)["TotalNumberOfInvalidValues"] != float64(1) ||
		q["DuplicateTimestamps"].(map[string]any)["TotalNumberOfDuplicateTimestamps"] != float64(1) {
		t.Fatalf("quality: %v", q)
	}
	if ds := lkeOK(t, ts, "DescribeDataset", `{"DatasetName":"pump"}`); ds["Status"] != "ACTIVE" {
		t.Fatalf("dataset: %v", ds)
	}
	stats := lkeOK(t, ts, "ListSensorStatistics", `{"DatasetName":"pump"}`)["SensorStatisticsSummaries"].([]any)
	if len(stats) != 2 || stats[1].(map[string]any)["ComponentName"] != "pump1" || stats[1].(map[string]any)["SensorName"] != "temp" {
		t.Fatalf("sensors: %v", stats)
	}
	// No CSVs under the prefix: the job fails and the dataset keeps its data.
	bad := lkeOK(t, ts, "StartDataIngestionJob", strings.Replace(ingest, "pump/", "none/", 1))["JobId"].(string)
	if failed := lkeAwait(t, ts, "DescribeDataIngestionJob", `{"JobId":"`+bad+`"}`, "Status", "FAILED"); failed["FailedReason"] == nil {
		t.Fatalf("failed job: %v", failed)
	}
	if ds := lkeOK(t, ts, "DescribeDataset", `{"DatasetName":"pump"}`); ds["Status"] != "ACTIVE" {
		t.Fatalf("dataset after failed ingestion: %v", ds)
	}

	lkeOK(t, ts, "CreateLabelGroup", `{"LabelGroupName":"faults","FaultCodes":["F1"],"ClientToken":"e"}`)
	if code := lkeErrCode(t, ts, "CreateLabel", `{"LabelGroupName":"faults","StartTime":10,"EndTime":5,"Rating":"ANOMALY","ClientToken":"f"}`); code != "ValidationException" {
		t.Fatalf("reversed label: %s", code)
	}
	if code := lkeErrCode(t, ts, "CreateLabel", `{"LabelGroupName":"faults","StartTime":1,"EndTime":5,"Rating":"ANOMALY","FaultCode":"F9","ClientToken":"f"}`); code != "ValidationException" {
		t.Fatalf("unknown fault code: %s", code)
	}
	label := lkeOK(t, ts, "CreateLabel", `{"LabelGroupName":"faults","StartTime":1704067200,"EndTime":1704070800,"Rating":"ANOMALY","FaultCode":"F1","ClientToken":"g"}`)["LabelId"].(string)
	if l := lkeOK(t, ts, "ListLabels", `{"LabelGroupName":"faults","FaultCode":"F1"}`)["LabelSummaries"].([]any); len(l) != 1 {
		t.Fatalf("labels: %v", l)
	}

	model := lkeOK(t, ts, "CreateModel", `{"ModelName":"m","DatasetName":"pump","ClientToken":"h","LabelsInputConfiguration":{"LabelGroupName":"faults"},"RoleArn":"`+lkeTestRole+`"}`)
	if model["Status"] != "IN_PROGRESS" {
		t.Fatalf("model: %v", model)
	}
	trained := lkeAwait(t, ts, "DescribeModel", `{"ModelName":"m"}`, "Status", "SUCCESS")
	if trained["ActiveModelVersion"] != float64(1) || !strings.Contains(trained["ModelMetrics"].(string), `"labels_count":1`) {
		t.Fatalf("trained: %v", trained)
	}
	if code := lkeErrCode(t, ts, "DescribeModelVersion", `{"ModelName":"m","ModelVersion":7}`); code != "ResourceNotFoundException" {
		t.Fatalf("missing version: %s", code)
	}
	v1 := lkeOK(t, ts, "DescribeModelVersion", `{"ModelName":"m","ModelVersion":1}`)["ModelVersionArn"].(string)
	imp := lkeOK(t, ts, "ImportModelVersion", `{"SourceModelVersionArn":"`+v1+`","ModelName":"m","DatasetName":"pump","ClientToken":"i"}`)
	if imp["ModelVersion"] != float64(2) {
		t.Fatalf("import: %v", imp)
	}
	lkeAwait(t, ts, "DescribeModelVersion", `{"ModelName":"m","ModelVersion":2}`, "Status", "SUCCESS")
	act := lkeOK(t, ts, "UpdateActiveModelVersion", `{"ModelName":"m","ModelVersion":1}`)
	if act["CurrentActiveVersion"] != float64(1) || act["PreviousActiveVersion"] != float64(2) {
		t.Fatalf("activate: %v", act)
	}
	if vs := lkeOK(t, ts, "ListModelVersions", `{"ModelName":"m","SourceType":"IMPORT"}`)["ModelVersionSummaries"].([]any); len(vs) != 1 {
		t.Fatalf("versions: %v", vs)
	}
	lkeOK(t, ts, "UpdateModel", `{"ModelName":"m","RoleArn":"`+lkeTestRole+`"}`)

	// Resource policies carry revisions.
	policy := `{\"Version\":\"2012-10-17\",\"Statement\":[]}`
	rev := lkeOK(t, ts, "PutResourcePolicy", `{"ResourceArn":"`+model["ModelArn"].(string)+`","ResourcePolicy":"`+policy+`","ClientToken":"j"}`)["PolicyRevisionId"].(string)
	if code := lkeErrCode(t, ts, "PutResourcePolicy", `{"ResourceArn":"`+model["ModelArn"].(string)+`","ResourcePolicy":"`+policy+`","PolicyRevisionId":"abc","ClientToken":"k"}`); code != "ConflictException" {
		t.Fatalf("stale revision: %s", code)
	}
	lkeOK(t, ts, "PutResourcePolicy", `{"ResourceArn":"`+model["ModelArn"].(string)+`","ResourcePolicy":"`+policy+`","PolicyRevisionId":"`+rev+`","ClientToken":"l"}`)
	lkeOK(t, ts, "DescribeResourcePolicy", `{"ResourceArn":"`+model["ModelArn"].(string)+`"}`)
	lkeOK(t, ts, "DeleteResourcePolicy", `{"ResourceArn":"`+model["ModelArn"].(string)+`"}`)

	// Retraining scheduler: PENDING -> RUNNING, stop before delete.
	lkeOK(t, ts, "CreateRetrainingScheduler", `{"ModelName":"m","RetrainingFrequency":"P30D","LookbackWindow":"P360D","ClientToken":"m"}`)
	lkeAwait(t, ts, "DescribeRetrainingScheduler", `{"ModelName":"m"}`, "Status", "RUNNING")
	if code := lkeErrCode(t, ts, "DeleteRetrainingScheduler", `{"ModelName":"m"}`); code != "ConflictException" {
		t.Fatalf("delete running retrainer: %s", code)
	}
	lkeOK(t, ts, "UpdateRetrainingScheduler", `{"ModelName":"m","PromoteMode":"MANUAL"}`)
	lkeOK(t, ts, "StopRetrainingScheduler", `{"ModelName":"m"}`)
	lkeAwait(t, ts, "DescribeRetrainingScheduler", `{"ModelName":"m"}`, "Status", "STOPPED")
	lkeOK(t, ts, "StartRetrainingScheduler", `{"ModelName":"m"}`)
	lkeOK(t, ts, "StopRetrainingScheduler", `{"ModelName":"m"}`)
	lkeAwait(t, ts, "DescribeRetrainingScheduler", `{"ModelName":"m"}`, "Status", "STOPPED")
	lkeOK(t, ts, "DeleteRetrainingScheduler", `{"ModelName":"m"}`)

	// Inference scheduler: executions accrue while RUNNING.
	sched := `{"ModelName":"m","InferenceSchedulerName":"s","DataUploadFrequency":"PT5M","DataInputConfiguration":{"S3InputConfiguration":{"Bucket":"lke-data","Prefix":"in/"}},"DataOutputConfiguration":{"S3OutputConfiguration":{"Bucket":"lke-data","Prefix":"out/"}},"RoleArn":"` + lkeTestRole + `","ClientToken":"n"}`
	lkeOK(t, ts, "CreateInferenceScheduler", sched)
	lkeAwait(t, ts, "DescribeInferenceScheduler", `{"InferenceSchedulerName":"s"}`, "Status", "RUNNING")
	execs := lkeOK(t, ts, "ListInferenceExecutions", `{"InferenceSchedulerName":"s"}`)["InferenceExecutionSummaries"].([]any)
	if len(execs) != 1 || execs[0].(map[string]any)["Status"] != "SUCCESS" {
		t.Fatalf("executions: %v", execs)
	}
	lkeOK(t, ts, "ListInferenceEvents", `{"InferenceSchedulerName":"s","IntervalStartTime":0,"IntervalEndTime":4102444800}`)
	if code := lkeErrCode(t, ts, "DeleteModel", `{"ModelName":"m"}`); code != "ConflictException" {
		t.Fatalf("delete model with scheduler: %s", code)
	}
	if code := lkeErrCode(t, ts, "DeleteInferenceScheduler", `{"InferenceSchedulerName":"s"}`); code != "ConflictException" {
		t.Fatalf("delete running scheduler: %s", code)
	}
	lkeOK(t, ts, "UpdateInferenceScheduler", `{"InferenceSchedulerName":"s","DataDelayOffsetInMinutes":5}`)
	lkeOK(t, ts, "StopInferenceScheduler", `{"InferenceSchedulerName":"s"}`)
	lkeAwait(t, ts, "DescribeInferenceScheduler", `{"InferenceSchedulerName":"s"}`, "Status", "STOPPED")
	lkeOK(t, ts, "StartInferenceScheduler", `{"InferenceSchedulerName":"s"}`)
	lkeOK(t, ts, "StopInferenceScheduler", `{"InferenceSchedulerName":"s"}`)
	lkeAwait(t, ts, "DescribeInferenceScheduler", `{"InferenceSchedulerName":"s"}`, "Status", "STOPPED")
	if l := lkeOK(t, ts, "ListInferenceSchedulers", `{"ModelName":"m"}`)["InferenceSchedulerSummaries"].([]any); len(l) != 1 {
		t.Fatalf("schedulers: %v", l)
	}
	lkeOK(t, ts, "DeleteInferenceScheduler", `{"InferenceSchedulerName":"s"}`)

	// Tags and listings.
	dsArn := lkeOK(t, ts, "DescribeDataset", `{"DatasetName":"pump"}`)["DatasetArn"].(string)
	lkeOK(t, ts, "TagResource", `{"ResourceArn":"`+dsArn+`","Tags":[{"Key":"team","Value":"ops"}]}`)
	lkeOK(t, ts, "UntagResource", `{"ResourceArn":"`+dsArn+`","TagKeys":["env"]}`)
	if tags := lkeOK(t, ts, "ListTagsForResource", `{"ResourceArn":"`+dsArn+`"}`)["Tags"].([]any); len(tags) != 1 {
		t.Fatalf("tags: %v", tags)
	}
	imported := lkeOK(t, ts, "ImportDataset", `{"SourceDatasetArn":"`+dsArn+`","DatasetName":"pump-copy","ClientToken":"o"}`)
	lkeAwait(t, ts, "DescribeDataIngestionJob", `{"JobId":"`+imported["JobId"].(string)+`"}`, "Status", "SUCCESS")
	if ds := lkeOK(t, ts, "DescribeDataset", `{"DatasetName":"pump-copy"}`); ds["Status"] != "ACTIVE" || ds["SourceDatasetArn"] != dsArn {
		t.Fatalf("imported dataset: %v", ds)
	}
	for _, op := range []string{"ListDatasets", "ListModels", "ListLabelGroups", "ListRetrainingSchedulers"} {
		lkeOK(t, ts, op, `{}`)
	}
	if jobs := lkeOK(t, ts, "ListDataIngestionJobs", `{"DatasetName":"pump","MaxResults":1}`); jobs["NextToken"] == nil {
		t.Fatalf("jobs page: %v", jobs)
	}

	// Cleanup in dependency order.
	lkeOK(t, ts, "DeleteLabel", fmt.Sprintf(`{"LabelGroupName":"faults","LabelId":%q}`, label))
	lkeOK(t, ts, "DeleteLabelGroup", `{"LabelGroupName":"faults"}`)
	lkeOK(t, ts, "DeleteModel", `{"ModelName":"m"}`)
	lkeOK(t, ts, "DeleteDataset", `{"DatasetName":"pump"}`)
	if code := lkeErrCode(t, ts, "DescribeModel", `{"ModelName":"m"}`); code != "ResourceNotFoundException" {
		t.Fatalf("deleted model: %s", code)
	}
}

func TestLookoutEquipmentUpdateRechargesBytes(t *testing.T) {
	st := newLookoutEquipmentStore()
	if _, err := st.dispatch("CreateLabelGroup", map[string]any{"LabelGroupName": "g"}, nil); err != nil {
		t.Fatal(err)
	}
	st.bytes = lkeMaxBytes - 10
	codes := make([]any, 50)
	for i := range codes {
		codes[i] = strings.Repeat("c", 100)
	}
	if _, err := st.dispatch("UpdateLabelGroup", map[string]any{"LabelGroupName": "g", "FaultCodes": codes}, nil); err == nil || err.code != "ServiceQuotaExceededException" {
		t.Fatalf("update past the byte cap: %v", err)
	}
	if got := st.groups["g"].doc["FaultCodes"].([]any); len(got) != 0 {
		t.Fatalf("refused update was applied: %v", got)
	}
}
