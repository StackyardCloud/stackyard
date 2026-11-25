package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const fcRole = "arn:aws:iam::123456789012:role/forecast"

// fcActive describes arn until it reports ACTIVE.
func fcActive(t *testing.T, ts *httptest.Server, op, key, arn string) map[string]any {
	t.Helper()
	for i := 0; i < 3; i++ {
		out := fcOK(t, ts, op, fmt.Sprintf(`{%q:%q}`, key, arn))
		if out["Status"] == "ACTIVE" {
			return out
		}
	}
	t.Fatalf("%s never became ACTIVE", arn)
	return nil
}

// fcPipeline creates a dataset, import job, dataset group, auto predictor
// and forecast, all ACTIVE, and returns their ARNs.
func fcPipeline(t *testing.T, ts *httptest.Server, name string) map[string]string {
	t.Helper()
	ds := fcOK(t, ts, "CreateDataset", fmt.Sprintf(`{"DatasetName":"%s_ds","Domain":"RETAIL","DatasetType":"TARGET_TIME_SERIES","DataFrequency":"D",
		"Schema":{"Attributes":[{"AttributeName":"item_id","AttributeType":"string"},{"AttributeName":"timestamp","AttributeType":"timestamp"},{"AttributeName":"demand","AttributeType":"float"}]}}`, name))["DatasetArn"].(string)
	job := fcOK(t, ts, "CreateDatasetImportJob", fmt.Sprintf(`{"DatasetImportJobName":"%s_job","DatasetArn":%q,
		"DataSource":{"S3Config":{"Path":"s3://bucket/demand.csv","RoleArn":%q}}}`, name, ds, fcRole))["DatasetImportJobArn"].(string)
	fcActive(t, ts, "DescribeDatasetImportJob", "DatasetImportJobArn", job)
	group := fcOK(t, ts, "CreateDatasetGroup", fmt.Sprintf(`{"DatasetGroupName":"%s_dsg","Domain":"RETAIL","DatasetArns":[%q]}`, name, ds))["DatasetGroupArn"].(string)
	pred := fcOK(t, ts, "CreateAutoPredictor", fmt.Sprintf(`{"PredictorName":"%s_pred","ForecastHorizon":7,"ForecastFrequency":"D",
		"DataConfig":{"DatasetGroupArn":%q}}`, name, group))["PredictorArn"].(string)
	fcActive(t, ts, "DescribeAutoPredictor", "PredictorArn", pred)
	fc := fcOK(t, ts, "CreateForecast", fmt.Sprintf(`{"ForecastName":"%s_fc","PredictorArn":%q}`, name, pred))["ForecastArn"].(string)
	fcActive(t, ts, "DescribeForecast", "ForecastArn", fc)
	return map[string]string{"dataset": ds, "job": job, "group": group, "predictor": pred, "forecast": fc}
}

func TestForecastStage1PipelineAndQuery(t *testing.T) {
	ts := newForecastTestServer(t)
	// A predictor needs a target time series in its group.
	empty := fcOK(t, ts, "CreateDatasetGroup", `{"DatasetGroupName":"empty","Domain":"RETAIL"}`)["DatasetGroupArn"].(string)
	if code := fcErrCode(t, ts, "CreateAutoPredictor", fmt.Sprintf(`{"PredictorName":"p","ForecastHorizon":7,"ForecastFrequency":"D","DataConfig":{"DatasetGroupArn":%q}}`, empty)); code != "InvalidInputException" {
		t.Fatalf("predictor without data: %s", code)
	}
	if code := fcErrCode(t, ts, "CreateDataset", `{"DatasetName":"nodemand","Domain":"RETAIL","DatasetType":"TARGET_TIME_SERIES","Schema":{"Attributes":[{"AttributeName":"timestamp","AttributeType":"timestamp"}]}}`); code != "InvalidInputException" {
		t.Fatalf("target without its field: %s", code)
	}

	p := fcPipeline(t, ts, "retail")
	// A predictor in training can't be forecast from.
	pred2 := fcOK(t, ts, "CreateAutoPredictor", fmt.Sprintf(`{"PredictorName":"busy","ReferencePredictorArn":%q}`, p["predictor"]))["PredictorArn"].(string)
	if code := fcErrCode(t, ts, "CreateForecast", fmt.Sprintf(`{"ForecastName":"early","PredictorArn":%q}`, pred2)); code != "ResourceInUseException" {
		t.Fatalf("forecast from a training predictor: %s", code)
	}
	if out := fcOK(t, ts, "DescribeAutoPredictor", fmt.Sprintf(`{"PredictorArn":%q}`, pred2)); out["Status"] != "CREATE_IN_PROGRESS" ||
		out["ReferencePredictorSummary"].(map[string]any)["Arn"] != p["predictor"] || out["ForecastHorizon"] != 7.0 {
		t.Fatalf("retrained predictor: %v", out)
	}
	if code := fcErrCode(t, ts, "CreateDatasetGroup", fmt.Sprintf(`{"DatasetGroupName":"retail_dsg","Domain":"RETAIL","DatasetArns":[%q]}`, p["dataset"])); code != "ResourceAlreadyExistsException" {
		t.Fatalf("duplicate name: %s", code)
	}

	desc := fcOK(t, ts, "DescribeAutoPredictor", fmt.Sprintf(`{"PredictorArn":%q}`, p["predictor"]))
	if jobs := desc["DatasetImportJobArns"].([]any); len(jobs) != 1 || jobs[0] != p["job"] {
		t.Fatalf("predictor import jobs: %v", desc)
	}
	metrics := fcOK(t, ts, "GetAccuracyMetrics", fmt.Sprintf(`{"PredictorArn":%q}`, p["predictor"]))
	if r := metrics["PredictorEvaluationResults"].([]any); len(r) != 1 || metrics["IsAutoPredictor"] != true {
		t.Fatalf("accuracy metrics: %v", metrics)
	}

	q := func(body string) (int, map[string]any) {
		return jsonCall(t, ts, fcqTargetPrefix+"QueryForecast", "forecast", body)
	}
	status, out := q(fmt.Sprintf(`{"ForecastArn":%q,"Filters":{"item_id":"sku1"}}`, p["forecast"]))
	if status != http.StatusOK {
		t.Fatalf("query: %d %v", status, out)
	}
	preds := out["Forecast"].(map[string]any)["Predictions"].(map[string]any)
	if len(preds) != 3 || len(preds["p50"].([]any)) != 7 {
		t.Fatalf("predictions: %v", preds)
	}
	p10 := preds["p10"].([]any)[0].(map[string]any)["Value"].(float64)
	p90 := preds["p90"].([]any)[0].(map[string]any)["Value"].(float64)
	if p10 >= p90 {
		t.Fatalf("p10 %v >= p90 %v", p10, p90)
	}
	_, again := q(fmt.Sprintf(`{"ForecastArn":%q,"Filters":{"item_id":"sku1"}}`, p["forecast"]))
	if fmt.Sprint(again) != fmt.Sprint(out) {
		t.Fatal("queries are not deterministic")
	}
	first := preds["p50"].([]any)[2].(map[string]any)["Timestamp"].(string)
	_, window := q(fmt.Sprintf(`{"ForecastArn":%q,"Filters":{"item_id":"sku1"},"StartDate":%q}`, p["forecast"], first))
	if n := len(window["Forecast"].(map[string]any)["Predictions"].(map[string]any)["p50"].([]any)); n != 5 {
		t.Fatalf("StartDate window: %d points", n)
	}
	if status, out := q(fmt.Sprintf(`{"ForecastArn":%q,"Filters":{"store":"x"}}`, p["forecast"])); status != http.StatusBadRequest || out["__type"] != "InvalidInputException" {
		t.Fatalf("unknown filter key: %d %v", status, out)
	}

	// What-if analysis and forecast, queried through forecastquery.
	wa := fcOK(t, ts, "CreateWhatIfAnalysis", fmt.Sprintf(`{"WhatIfAnalysisName":"promo","ForecastArn":%q}`, p["forecast"]))["WhatIfAnalysisArn"].(string)
	fcActive(t, ts, "DescribeWhatIfAnalysis", "WhatIfAnalysisArn", wa)
	wf := fcOK(t, ts, "CreateWhatIfForecast", fmt.Sprintf(`{"WhatIfForecastName":"half","WhatIfAnalysisArn":%q,
		"TimeSeriesTransformations":[{"Action":{"AttributeName":"price","Operation":"MULTIPLY","Value":0.5}}]}`, wa))["WhatIfForecastArn"].(string)
	fcActive(t, ts, "DescribeWhatIfForecast", "WhatIfForecastArn", wf)
	status, out = jsonCall(t, ts, fcqTargetPrefix+"QueryWhatIfForecast", "forecast", fmt.Sprintf(`{"WhatIfForecastArn":%q,"Filters":{"item_id":"sku1"}}`, wf))
	if status != http.StatusOK || len(out["Forecast"].(map[string]any)["Predictions"].(map[string]any)["p50"].([]any)) != 7 {
		t.Fatalf("what-if query: %d %v", status, out)
	}

	// Deleting needs children gone first; DeleteResourceTree removes them.
	if code := fcErrCode(t, ts, "DeletePredictor", fmt.Sprintf(`{"PredictorArn":%q}`, p["predictor"])); code != "ResourceInUseException" {
		t.Fatalf("delete predictor with a forecast: %s", code)
	}
	fcOK(t, ts, "DeleteResourceTree", fmt.Sprintf(`{"ResourceArn":%q}`, p["forecast"]))
	if code := fcErrCode(t, ts, "DescribeWhatIfForecast", fmt.Sprintf(`{"WhatIfForecastArn":%q}`, wf)); code != "ResourceNotFoundException" {
		t.Fatalf("tree delete left the what-if forecast: %s", code)
	}
}

func TestForecastStage1MonitorsTagsAndPaging(t *testing.T) {
	ts := newForecastTestServer(t)
	p := fcPipeline(t, ts, "mon")
	m := fcOK(t, ts, "CreateMonitor", fmt.Sprintf(`{"MonitorName":"m1","ResourceArn":%q,"Tags":[{"Key":"team","Value":"ml"}]}`, p["predictor"]))["MonitorArn"].(string)
	if code := fcErrCode(t, ts, "CreateMonitor", fmt.Sprintf(`{"MonitorName":"m2","ResourceArn":%q}`, p["predictor"])); code != "ResourceAlreadyExistsException" {
		t.Fatalf("second monitor: %s", code)
	}
	fcActive(t, ts, "DescribeMonitor", "MonitorArn", m)
	fcOK(t, ts, "StopResource", fmt.Sprintf(`{"ResourceArn":%q}`, m))
	if s := fcOK(t, ts, "DescribeMonitor", fmt.Sprintf(`{"MonitorArn":%q}`, m))["Status"]; s != "ACTIVE_STOPPED" {
		t.Fatalf("stopped monitor: %v", s)
	}
	fcOK(t, ts, "ResumeResource", fmt.Sprintf(`{"ResourceArn":%q}`, m))
	if code := fcErrCode(t, ts, "ResumeResource", fmt.Sprintf(`{"ResourceArn":%q}`, m)); code != "ResourceInUseException" {
		t.Fatalf("resume an active monitor: %s", code)
	}
	// New data after the monitor was created yields an evaluation.
	job := fcOK(t, ts, "CreateDatasetImportJob", fmt.Sprintf(`{"DatasetImportJobName":"more","DatasetArn":%q,"ImportMode":"INCREMENTAL",
		"DataSource":{"S3Config":{"Path":"s3://bucket/more.csv","RoleArn":%q}}}`, p["dataset"], fcRole))["DatasetImportJobArn"].(string)
	fcActive(t, ts, "DescribeDatasetImportJob", "DatasetImportJobArn", job)
	evals := fcOK(t, ts, "ListMonitorEvaluations", fmt.Sprintf(`{"MonitorArn":%q}`, m))["PredictorMonitorEvaluations"].([]any)
	if len(evals) != 1 || evals[0].(map[string]any)["MonitorDataSource"].(map[string]any)["DatasetImportJobArn"] != job {
		t.Fatalf("evaluations: %v", evals)
	}

	tags := fcOK(t, ts, "ListTagsForResource", fmt.Sprintf(`{"ResourceArn":%q}`, m))["Tags"].([]any)
	if len(tags) != 1 {
		t.Fatalf("tags: %v", tags)
	}
	if code := fcErrCode(t, ts, "TagResource", fmt.Sprintf(`{"ResourceArn":%q,"Tags":[{"Key":"aws:x","Value":"y"}]}`, m)); code != "InvalidInputException" {
		t.Fatalf("aws: tag: %s", code)
	}
	fcOK(t, ts, "UntagResource", fmt.Sprintf(`{"ResourceArn":%q,"TagKeys":["team"]}`, m))
	if tags := fcOK(t, ts, "ListTagsForResource", fmt.Sprintf(`{"ResourceArn":%q}`, m))["Tags"].([]any); len(tags) != 0 {
		t.Fatalf("after untag: %v", tags)
	}

	// Filters and paging on list operations.
	for i := 0; i < 3; i++ {
		fcOK(t, ts, "CreateDatasetGroup", fmt.Sprintf(`{"DatasetGroupName":"g%d","Domain":"CUSTOM"}`, i))
	}
	page := fcOK(t, ts, "ListDatasetGroups", `{"MaxResults":2}`)
	if len(page["DatasetGroups"].([]any)) != 2 || page["NextToken"] == nil {
		t.Fatalf("page 1: %v", page)
	}
	rest := fcOK(t, ts, "ListDatasetGroups", fmt.Sprintf(`{"NextToken":%q}`, page["NextToken"]))
	if len(rest["DatasetGroups"].([]any)) != 2 {
		t.Fatalf("page 2: %v", rest)
	}
	if code := fcErrCode(t, ts, "ListDatasetGroups", `{"NextToken":"bogus"}`); code != "InvalidNextTokenException" {
		t.Fatalf("bad token: %s", code)
	}
	jobs := fcOK(t, ts, "ListDatasetImportJobs", fmt.Sprintf(`{"Filters":[{"Key":"DatasetArn","Value":%q,"Condition":"IS"},{"Key":"Status","Value":"ACTIVE","Condition":"IS"}]}`, p["dataset"]))
	if len(jobs["DatasetImportJobs"].([]any)) != 2 {
		t.Fatalf("filtered jobs: %v", jobs)
	}
	if code := fcErrCode(t, ts, "ListDatasetImportJobs", `{"Filters":[{"Key":"Colour","Value":"x","Condition":"IS"}]}`); code != "InvalidInputException" {
		t.Fatalf("bad filter key: %s", code)
	}

	// Stopping an in-progress export job; a stopped job can be deleted.
	exp := fcOK(t, ts, "CreateForecastExportJob", fmt.Sprintf(`{"ForecastExportJobName":"out","ForecastArn":%q,"Destination":{"S3Config":{"Path":"s3://bucket/out/","RoleArn":%q}}}`, p["forecast"], fcRole))["ForecastExportJobArn"].(string)
	if code := fcErrCode(t, ts, "DeleteForecastExportJob", fmt.Sprintf(`{"ForecastExportJobArn":%q}`, exp)); code != "ResourceInUseException" {
		t.Fatalf("delete in-progress job: %s", code)
	}
	fcOK(t, ts, "StopResource", fmt.Sprintf(`{"ResourceArn":%q}`, exp))
	fcOK(t, ts, "DeleteForecastExportJob", fmt.Sprintf(`{"ForecastExportJobArn":%q}`, exp))
	if code := fcErrCode(t, ts, "StopResource", fmt.Sprintf(`{"ResourceArn":%q}`, p["dataset"])); code != "InvalidInputException" {
		t.Fatalf("stop a dataset: %s", code)
	}
}
