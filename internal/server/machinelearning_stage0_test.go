package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func amlCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	return jsonCall(t, ts, amlTargetPrefix+op, "machinelearning", body)
}

func amlOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := amlCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func amlErrCode(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := amlCall(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s: want 400, got %d %v", op, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func TestMachineLearningStage0CatalogMatchesModel(t *testing.T) {
	m := awsmodels.MachineLearningModel()
	if len(machineLearningOperations) != len(m.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(machineLearningOperations), len(m.Operations))
	}
	st := newMachineLearningStore()
	for _, op := range machineLearningOperations {
		if _, ok := m.Operations[op.Name]; !ok {
			t.Errorf("%s is not in the model", op.Name)
		}
		if _, _, err := st.dispatch(op.Name, map[string]any{}, "", &amlPre{}); err != nil && err.code == "UnknownOperationException" {
			t.Errorf("%s is not dispatched", op.Name)
		}
	}
}

func TestMachineLearningStage0Routing(t *testing.T) {
	ts := newForecastTestServer(t)
	if out := amlOK(t, ts, "DescribeMLModels", `{}`); out["Results"] == nil {
		t.Fatalf("DescribeMLModels: %v", out)
	}
	if status, _ := jsonCall(t, ts, amlTargetPrefix+"DescribeMLModels", "forecast", `{}`); status != http.StatusForbidden {
		t.Fatalf("wrong signing service: %d", status)
	}
	if code := amlErrCode(t, ts, "GetMLModel", `{"MLModelId":""}`); code != "InvalidInputException" {
		t.Fatalf("empty id: %s", code)
	}
	if code := amlErrCode(t, ts, "GetMLModel", `{"MLModelId":"none"}`); code != "ResourceNotFoundException" {
		t.Fatalf("missing model: %s", code)
	}
	// Neighbours keep their traffic.
	if out := fcOK(t, ts, "ListDatasets", `{}`); out["Datasets"] == nil {
		t.Fatalf("Forecast ListDatasets: %v", out)
	}
}
