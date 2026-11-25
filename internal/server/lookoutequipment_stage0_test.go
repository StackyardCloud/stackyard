package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func lkeCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	return jsonCall(t, ts, lkeTargetPrefix+op, "lookoutequipment", body)
}

func lkeOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := lkeCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func lkeErrCode(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := lkeCall(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s: want 400, got %d %v", op, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func TestLookoutEquipmentStage0CatalogMatchesModel(t *testing.T) {
	m := awsmodels.LookoutEquipmentModel()
	if len(lookoutEquipmentOperations) != len(m.Operations) {
		t.Fatalf("catalog has %d operations, model %d", len(lookoutEquipmentOperations), len(m.Operations))
	}
	for _, op := range lookoutEquipmentOperations {
		if _, ok := m.Operations[op.Name]; !ok {
			t.Errorf("%s is not in the model", op.Name)
		}
		if lkeHandlers[op.Name] == nil {
			t.Errorf("%s has no handler", op.Name)
		}
	}
}

func TestLookoutEquipmentStage0Routing(t *testing.T) {
	ts := newForecastTestServer(t)
	if out := lkeOK(t, ts, "ListDatasets", `{}`); out["DatasetSummaries"] == nil {
		t.Fatalf("ListDatasets: %v", out)
	}
	if status, _ := jsonCall(t, ts, lkeTargetPrefix+"ListDatasets", "machinelearning", `{}`); status != http.StatusForbidden {
		t.Fatalf("wrong signing service: %d", status)
	}
	// Model validation reports ValidationException (not Forecast's code).
	if code := lkeErrCode(t, ts, "CreateDataset", `{"DatasetName":"bad name!","ClientToken":"t"}`); code != "ValidationException" {
		t.Fatalf("bad name: %s", code)
	}
	if code := lkeErrCode(t, ts, "DescribeDataset", `{"DatasetName":"none"}`); code != "ResourceNotFoundException" {
		t.Fatalf("missing dataset: %s", code)
	}
	if code := lkeErrCode(t, ts, "NoSuchOperation", `{}`); code != "UnknownOperationException" {
		t.Fatalf("unknown op: %s", code)
	}
	// Neighbours keep their traffic.
	if out := amlOK(t, ts, "DescribeDataSources", `{}`); out["Results"] == nil {
		t.Fatalf("Machine Learning: %v", out)
	}
}
