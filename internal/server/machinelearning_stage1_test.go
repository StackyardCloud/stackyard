package server

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

const amlTestSchema = `{\"version\":\"1.0\",\"targetAttributeName\":\"y\",\"dataFormat\":\"CSV\",\"dataFileContainsHeader\":true,\"attributes\":[{\"attributeName\":\"x\",\"attributeType\":\"NUMERIC\"},{\"attributeName\":\"y\",\"attributeType\":\"BINARY\"}]}`

func amlAwait(t *testing.T, ts *httptest.Server, kind, id, want string) map[string]any {
	t.Helper()
	for i := 0; i < 4; i++ {
		out := amlOK(t, ts, "Get"+kind, fmt.Sprintf(`{%q:%q}`, amlIDMember[kind], id))
		if out["Status"] == want {
			return out
		}
	}
	t.Fatalf("%s %s never reached %s", kind, id, want)
	return nil
}

func TestMachineLearningLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	if _, err := srv.s3.CreateBucket("aml-data"); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.s3.PutObject("aml-data", "train.csv", "text/csv", []byte("x,y\n1,0\n2,1\n3,1\n"), nil, "", ""); err != nil {
		t.Fatal(err)
	}

	ds := `{"DataSourceId":"ds1","DataSourceName":"train","DataSpec":{"DataLocationS3":"s3://aml-data/train.csv","DataSchema":"` + amlTestSchema + `"},"ComputeStatistics":true}`
	amlOK(t, ts, "CreateDataSourceFromS3", ds)
	// The same request again is an idempotent retry; different parameters are not.
	amlOK(t, ts, "CreateDataSourceFromS3", ds)
	if code := amlErrCode(t, ts, "CreateDataSourceFromS3", strings.Replace(ds, `"train"`, `"other"`, 1)); code != "IdempotentParameterMismatchException" {
		t.Fatalf("mismatch: %s", code)
	}
	if code := amlErrCode(t, ts, "CreateDataSourceFromS3", `{"DataSourceId":"ds-bad","DataSpec":{"DataLocationS3":"s3://aml-data/train.csv","DataSchema":"not json"}}`); code != "InvalidInputException" {
		t.Fatalf("bad schema: %s", code)
	}
	got := amlAwait(t, ts, "DataSource", "ds1", "COMPLETED")
	if got["DataSizeInBytes"] != float64(16) || got["NumberOfFiles"] != float64(1) || got["DataSourceSchema"] != nil {
		t.Fatalf("data source: %v", got)
	}
	if v := amlOK(t, ts, "GetDataSource", `{"DataSourceId":"ds1","Verbose":true}`); v["DataSourceSchema"] == nil {
		t.Fatalf("verbose: %v", v)
	}
	// Missing data settles FAILED.
	amlOK(t, ts, "CreateDataSourceFromS3", `{"DataSourceId":"ds-missing","DataSpec":{"DataLocationS3":"s3://aml-data/none.csv","DataSchema":"`+amlTestSchema+`"}}`)
	if failed := amlAwait(t, ts, "DataSource", "ds-missing", "FAILED"); failed["Message"] == nil {
		t.Fatalf("failed data source: %v", failed)
	}

	amlOK(t, ts, "CreateMLModel", `{"MLModelId":"m1","MLModelName":"model","MLModelType":"BINARY","TrainingDataSourceId":"ds1"}`)
	if code := amlErrCode(t, ts, "Predict", `{"MLModelId":"m1","Record":{"x":"1"},"PredictEndpoint":"http://x"}`); code != "PredictorNotMountedException" {
		t.Fatalf("predict without endpoint: %s", code)
	}
	amlAwait(t, ts, "MLModel", "m1", "COMPLETED")
	ep := amlOK(t, ts, "CreateRealtimeEndpoint", `{"MLModelId":"m1"}`)["RealtimeEndpointInfo"].(map[string]any)
	if ep["EndpointStatus"] != "UPDATING" || !strings.HasPrefix(ep["EndpointUrl"].(string), "http://127.0.0.1") {
		t.Fatalf("endpoint: %v", ep)
	}
	pred := amlOK(t, ts, "Predict", `{"MLModelId":"m1","Record":{"x":"2"},"PredictEndpoint":"`+ep["EndpointUrl"].(string)+`"}`)["Prediction"].(map[string]any)
	again := amlOK(t, ts, "Predict", `{"MLModelId":"m1","Record":{"x":"2"},"PredictEndpoint":"x"}`)["Prediction"].(map[string]any)
	if pred["predictedLabel"] == nil || fmt.Sprint(pred) != fmt.Sprint(again) {
		t.Fatalf("prediction: %v / %v", pred, again)
	}

	amlOK(t, ts, "CreateEvaluation", `{"EvaluationId":"e1","MLModelId":"m1","EvaluationDataSourceId":"ds1"}`)
	ev := amlAwait(t, ts, "Evaluation", "e1", "COMPLETED")
	if props := ev["PerformanceMetrics"].(map[string]any)["Properties"].(map[string]any); props["BinaryAUC"] == nil {
		t.Fatalf("evaluation: %v", ev)
	}

	amlOK(t, ts, "CreateBatchPrediction", `{"BatchPredictionId":"b1","MLModelId":"m1","BatchPredictionDataSourceId":"ds1","OutputUri":"s3://aml-data/out/"}`)
	bp := amlAwait(t, ts, "BatchPrediction", "b1", "COMPLETED")
	if bp["TotalRecordCount"] != float64(3) {
		t.Fatalf("batch: %v", bp)
	}
	obj, err := srv.s3.GetObject("aml-data", "out/batch-prediction/result/b1-train.csv.gz")
	if err != nil {
		t.Fatal(err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(obj.Body))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(zr)
	if lines := strings.Split(strings.TrimSpace(string(raw)), "\n"); len(lines) != 4 || lines[0] != "bestAnswer,score" {
		t.Fatalf("batch output: %q", raw)
	}
	// No output bucket: the batch prediction settles FAILED.
	amlOK(t, ts, "CreateBatchPrediction", `{"BatchPredictionId":"b2","MLModelId":"m1","BatchPredictionDataSourceId":"ds1","OutputUri":"s3://nowhere/out/"}`)
	amlAwait(t, ts, "BatchPrediction", "b2", "FAILED")

	// Filters, sorting and paging.
	out := amlOK(t, ts, "DescribeBatchPredictions", `{"FilterVariable":"Status","EQ":"COMPLETED"}`)
	if res := out["Results"].([]any); len(res) != 1 {
		t.Fatalf("filtered: %v", out)
	}
	page := amlOK(t, ts, "DescribeDataSources", `{"Limit":1,"SortOrder":"asc","FilterVariable":"Name","Prefix":""}`)
	if len(page["Results"].([]any)) != 1 || page["NextToken"] == nil {
		t.Fatalf("page: %v", page)
	}
	if code := amlErrCode(t, ts, "DescribeDataSources", `{"NextToken":"999"}`); code != "InvalidInputException" {
		t.Fatalf("bad token: %s", code)
	}

	// Tags, with the 10-tag limit.
	amlOK(t, ts, "AddTags", `{"ResourceId":"m1","ResourceType":"MLModel","Tags":[{"Key":"env","Value":"test"}]}`)
	var many []string
	for i := 0; i < 11; i++ {
		many = append(many, fmt.Sprintf(`{"Key":"k%d","Value":"v"}`, i))
	}
	if code := amlErrCode(t, ts, "AddTags", `{"ResourceId":"m1","ResourceType":"MLModel","Tags":[`+strings.Join(many, ",")+`]}`); code != "TagLimitExceededException" {
		t.Fatalf("tag limit: %s", code)
	}
	if tags := amlOK(t, ts, "DescribeTags", `{"ResourceId":"m1","ResourceType":"MLModel"}`)["Tags"].([]any); len(tags) != 1 {
		t.Fatalf("tags: %v", tags)
	}
	amlOK(t, ts, "DeleteTags", `{"ResourceId":"m1","ResourceType":"MLModel","TagKeys":["env"]}`)

	// Updates, then deletion marks DELETED.
	amlOK(t, ts, "UpdateMLModel", `{"MLModelId":"m1","MLModelName":"renamed","ScoreThreshold":0.9}`)
	if m := amlOK(t, ts, "GetMLModel", `{"MLModelId":"m1"}`); m["Name"] != "renamed" || m["ScoreThreshold"] != 0.9 {
		t.Fatalf("update: %v", m)
	}
	amlOK(t, ts, "UpdateDataSource", `{"DataSourceId":"ds1","DataSourceName":"n"}`)
	amlOK(t, ts, "UpdateEvaluation", `{"EvaluationId":"e1","EvaluationName":"n"}`)
	amlOK(t, ts, "UpdateBatchPrediction", `{"BatchPredictionId":"b1","BatchPredictionName":"n"}`)
	amlOK(t, ts, "DeleteRealtimeEndpoint", `{"MLModelId":"m1"}`)
	amlOK(t, ts, "DeleteMLModel", `{"MLModelId":"m1"}`)
	// A DELETED model stays readable.
	if m := amlOK(t, ts, "GetMLModel", `{"MLModelId":"m1"}`); m["Status"] != "DELETED" {
		t.Fatalf("deleted: %v", m)
	}
	if code := amlErrCode(t, ts, "DeleteMLModel", `{"MLModelId":"m1"}`); code != "ResourceNotFoundException" {
		t.Fatalf("delete twice: %s", code)
	}
	for _, op := range []string{`DeleteEvaluation","EvaluationId":"e1`, `DeleteBatchPrediction","BatchPredictionId":"b1`, `DeleteDataSource","DataSourceId":"ds1`} {
		parts := strings.SplitN(op, `","`, 2)
		amlOK(t, ts, parts[0], `{"`+parts[1]+`"}`)
	}
}

func TestMachineLearningUpdateRechargesBytes(t *testing.T) {
	st := newMachineLearningStore()
	pre := &amlPre{}
	if _, _, err := st.dispatch("CreateDataSourceFromRedshift", map[string]any{"DataSourceId": "r", "DataSpec": map[string]any{}}, "", pre); err != nil {
		t.Fatal(err)
	}
	st.bytes = amlMaxBytes - 10
	long := strings.Repeat("n", 1000)
	if _, _, err := st.dispatch("UpdateDataSource", map[string]any{"DataSourceId": "r", "DataSourceName": long}, "", pre); err == nil || err.code != "LimitExceededException" {
		t.Fatalf("update past the byte cap: %v", err)
	}
	if st.res["DataSource"]["r"].rec["Name"] == long {
		t.Fatal("refused update was applied")
	}
}
