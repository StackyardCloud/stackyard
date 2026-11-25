package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestSageMakerMetricsStage1PutAndQuery(t *testing.T) {
	srv, ts := newSageMakerRuntimeTestServer(t)
	smOK(t, ts, "CreateTrialComponent", `{"TrialComponentName":"tc-1","Parameters":{"lr":{"NumberValue":0.1}}}`)
	put := func(name, points string) (int, map[string]any) {
		return smData(t, ts, http.MethodPut, "/BatchPutMetrics", fmt.Sprintf(`{"TrialComponentName":%q,"MetricData":[%s]}`, name, points))
	}
	// Steps 0..3 over two minutes.
	status, out := put("tc-1", `{"MetricName":"loss","Timestamp":1700000000,"Step":0,"Value":4},{"MetricName":"loss","Timestamp":1700000030,"Step":1,"Value":2},
		{"MetricName":"loss","Timestamp":1700000060,"Step":2,"Value":1},{"MetricName":"loss","Timestamp":1700000090,"Step":3,"Value":3}`)
	if status != http.StatusOK || len(out) != 0 {
		t.Fatalf("BatchPutMetrics: %d %v", status, out)
	}
	if status, out := put("missing", `{"MetricName":"loss","Timestamp":1,"Value":1}`); status != http.StatusBadRequest || out["__type"] != "ValidationError" {
		t.Fatalf("missing component: %d %v", status, out)
	}
	if status, _ := put("tc-1", `{"MetricName":"loss","Timestamp":1,"Step":-1,"Value":1}`); status != http.StatusBadRequest {
		t.Fatalf("negative step: %d", status)
	}

	arn := "arn:aws:sagemaker:us-east-1:123456789012:experiment-trial-component/tc-1"
	query := func(q string) map[string]any {
		_, out := smData(t, ts, http.MethodPost, "/BatchGetMetrics", fmt.Sprintf(`{"MetricQueries":[%s]}`, q))
		res, _ := out["MetricQueryResults"].([]any)
		if len(res) != 1 {
			t.Fatalf("BatchGetMetrics: %v", out)
		}
		return res[0].(map[string]any)
	}
	q := func(stat, period, axis string, extra string) string {
		return fmt.Sprintf(`{"MetricName":"loss","ResourceArn":%q,"MetricStat":%q,"Period":%q,"XAxisType":%q%s}`, arn, stat, period, axis, extra)
	}
	for _, c := range []struct{ q, want string }{
		{q("Avg", "OneMinute", "Timestamp", ""), `{"MetricValues":[3,2],"Status":"Complete","XAxisValues":[1699999980,1700000040]}`},
		{q("Max", "IterationNumber", "IterationNumber", `,"Start":1,"End":2`), `{"MetricValues":[2,1],"Status":"Complete","XAxisValues":[1,2]}`},
		{q("Count", "OneHour", "Timestamp", ""), `{"MetricValues":[4],"Status":"Complete","XAxisValues":[1699999200]}`},
		{q("Last", "FiveMinute", "IterationNumber", ""), `{"MetricValues":[3],"Status":"Complete","XAxisValues":[3]}`},
	} {
		if got := smJSON(query(c.q)); got != c.want {
			t.Errorf("%s:\n got %s\nwant %s", c.q, got, c.want)
		}
	}
	if got := query(q("Avg", "IterationNumber", "Timestamp", "")); got["Status"] != "ValidationError" {
		t.Fatalf("IterationNumber period on a time axis: %v", got)
	}
	// DescribeTrialComponent summarises the metric.
	out = smOK(t, ts, "DescribeTrialComponent", `{"TrialComponentName":"tc-1"}`)
	if got := smJSON(out["Metrics"]); !strings.Contains(got, `"Count":4`) || !strings.Contains(got, `"Last":3`) || !strings.Contains(got, `"Min":1`) {
		t.Fatalf("metric summary: %s", got)
	}

	// A training job's trial component works by name and by the job's ARN.
	smOK(t, ts, "CreateTrainingJob", `{"TrainingJobName":"job-a"}`)
	if status, out := put("job-a-aws-training-job", `{"MetricName":"acc","Timestamp":1700000000,"Step":1,"Value":0.9}`); status != http.StatusOK {
		t.Fatalf("training job metrics: %d %v", status, out)
	}
	jobQuery := `{"MetricName":"acc","ResourceArn":"arn:aws:sagemaker:us-east-1:123456789012:training-job/job-a","MetricStat":"Avg","Period":"IterationNumber","XAxisType":"IterationNumber"}`
	if got := smJSON(query(jobQuery)); got != `{"MetricValues":[0.9],"Status":"Complete","XAxisValues":[1]}` {
		t.Fatalf("query by training job ARN: %s", got)
	}
	out = smOK(t, ts, "ListTrialComponents", `{"SourceArn":"arn:aws:sagemaker:us-east-1:123456789012:training-job/job-a"}`)
	if got := smJSON(out["TrialComponentSummaries"]); !strings.Contains(got, `"job-a-aws-training-job"`) || strings.Contains(got, `"tc-1"`) {
		t.Fatalf("ListTrialComponents by source: %s", got)
	}

	// Deleting the component releases its points.
	smOK(t, ts, "DeleteTrialComponent", `{"TrialComponentName":"tc-1"}`)
	srv.sagemaker.mu.Lock()
	points := srv.sagemaker.smmPoints
	srv.sagemaker.mu.Unlock()
	if points != 1 {
		t.Fatalf("points after delete: %d", points)
	}
	if status, out := smCall(t, ts, "DescribeTrialComponent", `{"TrialComponentName":"tc-1"}`); status != http.StatusBadRequest || out["__type"] != "ResourceNotFound" {
		t.Fatalf("deleted component: %d %v", status, out)
	}
}

// Points per metric are capped; puts beyond the cap are reported per index.
func TestSageMakerMetricsStage1Limits(t *testing.T) {
	srv, ts := newSageMakerRuntimeTestServer(t)
	smOK(t, ts, "CreateTrialComponent", `{"TrialComponentName":"tc-1"}`)
	st := srv.sagemaker
	st.mu.Lock()
	series := &smmSeries{}
	for i := 0; i < smmMaxPointsPerMetric; i++ {
		series.add(smmPoint{at: float64(i), value: 1})
	}
	st.trialComponents["tc-1"].metrics["loss"] = series
	st.trialComponents["tc-1"].metricOrder = []string{"loss"}
	st.smmPoints += smmMaxPointsPerMetric
	st.mu.Unlock()
	status, out := smData(t, ts, http.MethodPut, "/BatchPutMetrics", `{"TrialComponentName":"tc-1","MetricData":[{"MetricName":"loss","Timestamp":1,"Value":1},{"MetricName":"other","Timestamp":1,"Value":1}]}`)
	var errs []struct {
		Code        string
		MetricIndex int
	}
	raw, _ := json.Marshal(out["Errors"])
	_ = json.Unmarshal(raw, &errs)
	if status != http.StatusOK || len(errs) != 1 || errs[0].Code != "METRIC_LIMIT_EXCEEDED" || errs[0].MetricIndex != 0 {
		t.Fatalf("limit: %d %v", status, out)
	}
}
