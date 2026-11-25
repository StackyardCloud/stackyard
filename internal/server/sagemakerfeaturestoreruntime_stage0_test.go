package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

// smCall sends a SageMaker control-plane request and decodes the reply.
func smCall(t *testing.T, ts *httptest.Server, action, payload string) (int, map[string]any) {
	t.Helper()
	resp := sagemakerRequest(t, ts, action, payload)
	return resp.StatusCode, smDecode(t, resp)
}

func smOK(t *testing.T, ts *httptest.Server, action, payload string) map[string]any {
	t.Helper()
	status, out := smCall(t, ts, action, payload)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", action, status, out)
	}
	return out
}

// smData sends a request to a SageMaker REST data plane (Feature Store
// Runtime, Metrics, Edge), signed for "sagemaker".
func smData(t *testing.T, ts *httptest.Server, method, path, body string) (int, map[string]any) {
	t.Helper()
	var raw []byte
	if body != "" {
		raw = []byte(body)
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, map[string]string{"Content-Type": "application/json"}, "sagemaker")
	return resp.StatusCode, smDecode(t, resp)
}

func smDecode(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("decode %q: %v", raw, err)
		}
	}
	return out
}

func TestSageMakerDataPlanesStage0CatalogsMatchModels(t *testing.T) {
	check := func(name string, model *awsmodels.Model, ops []string) {
		if len(ops) != len(model.Operations) {
			t.Fatalf("%s: catalog has %d operations, model %d", name, len(ops), len(model.Operations))
		}
		for _, op := range ops {
			if _, ok := model.Operations[op]; !ok {
				t.Errorf("%s: %s is not in the model", name, op)
			}
		}
	}
	var ops []string
	for _, op := range sagemakerfeaturestoreruntimeOperations {
		ops = append(ops, op.Name)
	}
	check("featurestore", awsmodels.SageMakerFeatureStoreRuntimeModel(), ops)
	ops = nil
	for _, op := range sagemakermetricsOperations {
		ops = append(ops, op.Name)
	}
	check("metrics", awsmodels.SageMakerMetricsModel(), ops)
	ops = nil
	for _, op := range sagemakeredgeOperations {
		ops = append(ops, op.Name)
	}
	check("edge", awsmodels.SageMakerEdgeModel(), ops)
	// Every stateful control-plane handler is in the slim model and the catalog.
	model := awsmodels.SageMakerResourceModel()
	if len(model.Operations) != len(sagemakerResourceHandlers) {
		t.Fatalf("resource model has %d operations, handlers %d", len(model.Operations), len(sagemakerResourceHandlers))
	}
	for op := range sagemakerResourceHandlers {
		if _, ok := model.Operations[op]; !ok {
			t.Errorf("%s is not in the resource model", op)
		}
		if _, ok := sagemakerOperationByName[op]; !ok {
			t.Errorf("%s is not in the SageMaker catalog", op)
		}
	}
}

// Each data-plane path reaches its service; the SageMaker control plane and
// SageMaker Runtime keep their own traffic, and other services' requests on
// the same paths are not claimed.
func TestSageMakerDataPlanesStage0Routing(t *testing.T) {
	_, ts := newSageMakerRuntimeTestServer(t)
	// Missing resources prove the request reached the right handler.
	cases := []struct {
		method, path, body, code string
		status                   int
	}{
		{http.MethodGet, "/FeatureGroup/nope?RecordIdentifierValueAsString=1", "", "ResourceNotFound", http.StatusNotFound},
		{http.MethodPut, "/FeatureGroup/nope", `{"Record":[{"FeatureName":"id","ValueAsString":"1"}]}`, "ResourceNotFound", http.StatusNotFound},
		{http.MethodDelete, "/FeatureGroup/nope?RecordIdentifierValueAsString=1&EventTime=1", "", "ResourceNotFound", http.StatusNotFound},
		{http.MethodPost, "/FeatureGroup/nope/ListRecords", `{}`, "ResourceNotFound", http.StatusNotFound},
		{http.MethodPost, "/FeatureGroup/nope/Record", `{"RecordIdentifierValueAsString":"1","Features":[{"FeatureName":"a"}]}`, "ResourceNotFound", http.StatusNotFound},
		{http.MethodPost, "/BatchWriteRecord", `{"Entries":[]}`, "ValidationError", http.StatusBadRequest},
		{http.MethodPost, "/BatchGetRecord", `{"Identifiers":[]}`, "ValidationError", http.StatusBadRequest},
		{http.MethodPut, "/BatchPutMetrics", `{"TrialComponentName":"nope","MetricData":[{"MetricName":"m","Timestamp":1,"Value":1}]}`, "ValidationError", http.StatusBadRequest},
		{http.MethodPost, "/GetDeviceRegistration", `{"DeviceName":"d","DeviceFleetName":"f"}`, "ValidationError", http.StatusBadRequest},
		{http.MethodPost, "/GetDeployments", `{"DeviceName":"d","DeviceFleetName":"f"}`, "ValidationError", http.StatusBadRequest},
		{http.MethodPost, "/SendHeartbeat", `{"DeviceName":"d","DeviceFleetName":"f","AgentVersion":"1.0"}`, "ValidationError", http.StatusBadRequest},
	}
	for _, c := range cases {
		status, out := smData(t, ts, c.method, c.path, c.body)
		if status != c.status || out["__type"] != c.code {
			t.Errorf("%s %s: %d %v", c.method, c.path, status, out)
		}
	}
	if status, out := smData(t, ts, http.MethodPost, "/BatchGetMetrics", `{"MetricQueries":[{"MetricName":"m","ResourceArn":"arn:aws:sagemaker:us-east-1:123456789012:experiment-trial-component/nope","MetricStat":"Avg","Period":"OneMinute","XAxisType":"Timestamp"}]}`); status != http.StatusOK ||
		!strings.Contains(smJSON(out), `"Status":"ValidationError"`) {
		t.Fatalf("BatchGetMetrics: %d %v", status, out)
	}
	// The control plane (X-Amz-Target) and SageMaker Runtime still answer.
	if out := smOK(t, ts, "ListFeatureGroups", `{}`); out["FeatureGroupSummaries"] == nil {
		t.Fatalf("ListFeatureGroups: %v", out)
	}
	resp := sagemakerRuntimeRequest(t, ts, "stackyard-endpoint", "invocations", []byte(`{}`), map[string]string{"Content-Type": "application/json"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("InvokeEndpoint: %d %s", resp.StatusCode, mustBody(t, resp))
	}
	resp.Body.Close()
	// A request to a data-plane path signed for another service isn't claimed.
	for _, path := range []string{"/BatchGetRecord", "/SendHeartbeat", "/BatchGetMetrics"} {
		resp := signedRequestWithService(t, http.MethodPost, ts.URL+path, []byte(`{}`), map[string]string{"Content-Type": "application/json"}, "s3")
		if body := string(mustBody(t, resp)); strings.Contains(body, `"ValidationError"`) {
			t.Fatalf("%s signed for s3 was claimed: %s", path, body)
		}
	}
	// Chunked bodies are refused before anything is read.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/BatchGetRecord", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=x/20240101/us-east-1/sagemaker/aws4_request, SignedHeaders=host, Signature=x")
	chunked, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer chunked.Body.Close()
	if chunked.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", chunked.StatusCode)
	}
}

func smJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
