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

func newForecastTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// jsonCall sends a signed awsJson1_1 request and decodes the reply.
func jsonCall(t *testing.T, ts *httptest.Server, target, service, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": target,
	}, service)
	raw := mustBody(t, resp)
	out := map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatalf("%s: decode %q: %v", target, raw, err)
		}
	}
	return resp.StatusCode, out
}

func fcCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	return jsonCall(t, ts, fcTargetPrefix+op, "forecast", body)
}

func fcOK(t *testing.T, ts *httptest.Server, op, body string) map[string]any {
	t.Helper()
	status, out := fcCall(t, ts, op, body)
	if status != http.StatusOK {
		t.Fatalf("%s: %d %v", op, status, out)
	}
	return out
}

func fcErrCode(t *testing.T, ts *httptest.Server, op, body string) string {
	t.Helper()
	status, out := fcCall(t, ts, op, body)
	if status != http.StatusBadRequest {
		t.Fatalf("%s: want 400, got %d %v", op, status, out)
	}
	code, _ := out["__type"].(string)
	return code
}

func TestForecastStage0CatalogMatchesModel(t *testing.T) {
	for _, c := range []struct {
		model *awsmodels.Model
		ops   []string
	}{
		{awsmodels.ForecastModel(), func() (o []string) {
			for _, op := range forecastOperations {
				o = append(o, op.Name)
			}
			return
		}()},
		{awsmodels.ForecastQueryModel(), func() (o []string) {
			for _, op := range forecastQueryOperations {
				o = append(o, op.Name)
			}
			return
		}()},
		{awsmodels.FraudDetectorModel(), func() (o []string) {
			for _, op := range fraudDetectorOperations {
				o = append(o, op.Name)
				if frdHandlers[op.Name] == nil {
					t.Errorf("frauddetector %s has no handler", op.Name)
				}
			}
			return
		}()},
	} {
		if len(c.ops) != len(c.model.Operations) {
			t.Fatalf("catalog has %d operations, model %d", len(c.ops), len(c.model.Operations))
		}
		for _, op := range c.ops {
			if _, ok := c.model.Operations[op]; !ok {
				t.Errorf("%s is not in the model", op)
			}
		}
	}
	// Every Forecast operation dispatches somewhere.
	st := newForecastStore()
	for _, op := range forecastOperations {
		if _, err := st.dispatch(op.Name, map[string]any{}); err != nil && err.code == "UnknownOperationException" {
			t.Errorf("forecast %s is not dispatched", op.Name)
		}
	}
}

func TestForecastStage0Routing(t *testing.T) {
	ts := newForecastTestServer(t)
	if out := fcOK(t, ts, "ListDatasets", `{}`); out["Datasets"] == nil {
		t.Fatalf("ListDatasets: %v", out)
	}
	// Forecast Query shares the signing name; its target prefix routes it.
	status, out := jsonCall(t, ts, fcqTargetPrefix+"QueryForecast", "forecast",
		`{"ForecastArn":"arn:aws:forecast:us-east-1:123456789012:forecast/none","Filters":{"item_id":"a"}}`)
	if status != http.StatusBadRequest || out["__type"] != "ResourceNotFoundException" {
		t.Fatalf("QueryForecast: %d %v", status, out)
	}
	if status, out := jsonCall(t, ts, frdTargetPrefix+"GetLabels", "frauddetector", `{}`); status != http.StatusOK || out["labels"] == nil {
		t.Fatalf("GetLabels: %d %v", status, out)
	}
	// Signed for another service: refused.
	if status, _ := jsonCall(t, ts, fcTargetPrefix+"ListDatasets", "frauddetector", `{}`); status != http.StatusForbidden {
		t.Fatalf("wrong signing service: %d", status)
	}
	if status, _ := jsonCall(t, ts, frdTargetPrefix+"GetLabels", "forecast", `{}`); status != http.StatusForbidden {
		t.Fatalf("wrong signing service for frauddetector: %d", status)
	}
	if code := fcErrCode(t, ts, "NoSuchOperation", `{}`); code != "UnknownOperationException" {
		t.Fatalf("unknown op: %s", code)
	}
	// Neighbours keep their traffic.
	if status, out := jsonCall(t, ts, "AWSWAF_Regional_20161128.ListIPSets", "waf-regional", `{}`); status != http.StatusOK || out["IPSets"] == nil {
		t.Fatalf("WAF Regional ListIPSets: %d %v", status, out)
	}
	if status, out := jsonCall(t, ts, "Textract.ListAdapters", "textract", `{}`); status != http.StatusOK || out["Adapters"] == nil {
		t.Fatalf("Textract ListAdapters: %d %v", status, out)
	}
	// Invalid input is refused against the model.
	if code := fcErrCode(t, ts, "CreateDatasetGroup", `{"DatasetGroupName":"1bad","Domain":"RETAIL"}`); code != "InvalidInputException" {
		t.Fatalf("bad name: %s", code)
	}
	// Chunked bodies of unknown length are refused.
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/", io.MultiReader(strings.NewReader(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Amz-Target", fcTargetPrefix+"ListDatasets")
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	chunked, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer chunked.Body.Close()
	if chunked.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", chunked.StatusCode)
	}
}
