package server

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAzureLoadTestingDataPlaneLifecycle(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	}
	jsonHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}
	fileHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/octet-stream",
	}

	base := "/azure/loadtesting-data-plane/2022-11-01"

	resp := providerContractRequest(t, ts, http.MethodPatch, base+"/tests/test-a?api-version=2022-11-01", []byte(`{
		"displayName":"Smoke test",
		"description":"stackyard load test"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for create test, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	testPayload := providerContractJSONMap(t, resp)
	if testPayload["testId"] != "test-a" || testPayload["displayName"] != "Smoke test" {
		t.Fatalf("unexpected test payload: %#v", testPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPut, base+"/tests/test-a/files/test-plan.jmx?api-version=2022-11-01&fileType=JMX_FILE", []byte("jmeter test plan"), fileHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for upload test file, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	filePayload := providerContractJSONMap(t, resp)
	if filePayload["fileName"] != "test-plan.jmx" || filePayload["fileType"] != "JMX_FILE" {
		t.Fatalf("unexpected test file payload: %#v", filePayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/tests/test-a/app-components?api-version=2022-11-01", []byte(`{
		"components":{
			"/subscriptions/000/resourceGroups/rg/providers/Microsoft.Web/sites/site-a":{"resourceId":"site-a"}
		}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for patch app components, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/tests/test-a/server-metrics-config?api-version=2022-11-01", []byte(`{
		"metrics":{
			"resourceId":"/subscriptions/000/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm-a",
			"metrics":["Percentage CPU"]
		}
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for patch server metrics, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/tests?api-version=2022-11-01&search=smoke", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list tests, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	listTests := providerContractJSONMap(t, resp)
	testsValue, ok := listTests["value"].([]any)
	if !ok || len(testsValue) != 1 {
		t.Fatalf("expected one test in list, got %#v", listTests)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/tests/test-a/app-components?api-version=2022-11-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get app components, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	componentsPayload := providerContractJSONMap(t, resp)
	if componentsPayload["components"] == nil {
		t.Fatalf("expected components payload, got %#v", componentsPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPatch, base+"/test-runs/run-a?api-version=2022-11-01", []byte(`{
		"testId":"test-a",
		"displayName":"Run A",
		"description":"initial run"
	}`), jsonHeaders)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for create test run, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	runPayload := providerContractJSONMap(t, resp)
	if runPayload["testRunId"] != "run-a" || runPayload["testId"] != "test-a" {
		t.Fatalf("unexpected run payload: %#v", runPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/test-runs?api-version=2022-11-01&search=run", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for list test runs, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	listRuns := providerContractJSONMap(t, resp)
	runsValue, ok := listRuns["value"].([]any)
	if !ok || len(runsValue) != 1 {
		t.Fatalf("expected one run in list, got %#v", listRuns)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/test-runs/run-a/files/results.csv?api-version=2022-11-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for get test run file, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	runFilePayload := providerContractJSONMap(t, resp)
	if runFilePayload["fileName"] != "results.csv" {
		t.Fatalf("unexpected run file payload: %#v", runFilePayload)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/test-runs/run-a/metric-namespaces?api-version=2022-11-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for metric namespaces, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	namespaces := providerContractJSONMap(t, resp)
	namespacesValue, ok := namespaces["value"].([]any)
	if !ok || len(namespacesValue) == 0 {
		t.Fatalf("expected metric namespaces, got %#v", namespaces)
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/test-runs/run-a/metric-definitions?api-version=2022-11-01&metricNamespace=LoadTestRunMetrics", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for metric definitions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodGet, base+"/test-runs/run-a/metric-dimensions/result?api-version=2022-11-01&metricName=response_time_ms&metricNamespace=LoadTestRunMetrics", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for metric dimensions, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/test-runs/run-a/metrics?api-version=2022-11-01&metricname=response_time_ms&metricNamespace=LoadTestRunMetrics&timespan=PT5M&aggregation=Average&interval=PT1M", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for metrics query, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	metricsPayload := providerContractJSONMap(t, resp)
	metricsValue, ok := metricsPayload["value"].([]any)
	if !ok || len(metricsValue) != 1 {
		t.Fatalf("expected one metrics series, got %#v", metricsPayload)
	}

	resp = providerContractRequest(t, ts, http.MethodPost, base+"/test-runs/run-a:stop?api-version=2022-11-01", nil, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for stop run, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	stoppedRun := providerContractJSONMap(t, resp)
	if stoppedRun["status"] != "CANCELLED" {
		t.Fatalf("expected cancelled run status, got %#v", stoppedRun)
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/test-runs/run-a?api-version=2022-11-01", nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete run, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/tests/test-a/files/test-plan.jmx?api-version=2022-11-01", nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete test file, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}

	resp = providerContractRequest(t, ts, http.MethodDelete, base+"/tests/test-a?api-version=2022-11-01", nil, headers)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for delete test, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
}

func TestAzureLoadTestingDataPlaneValidation(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:          "127.0.0.1:0",
		Providers:     []string{providerAzure},
		AzureAuthMode: "shared_key",
		AccessKey:     testAccessKey,
		SecretKey:     testSecretKey,
		LogLevel:      "error",
	})

	headers := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
	}
	jsonHeaders := map[string]string{
		"Authorization":             "SharedKey devstoreaccount1:signature",
		"Ocp-Apim-Subscription-Key": "stackyard-local-subscription-key",
		"Content-Type":              "application/json",
	}

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		headers    map[string]string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "invalid api version",
			method:     http.MethodGet,
			path:       "/azure/loadtesting-data-plane/2022-11-01/tests?api-version=",
			headers:    headers,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "invalid test id",
			method:     http.MethodPatch,
			path:       "/azure/loadtesting-data-plane/2022-11-01/tests/TestA?api-version=2022-11-01",
			body:       []byte(`{"displayName":"bad"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "run missing test id",
			method:     http.MethodPatch,
			path:       "/azure/loadtesting-data-plane/2022-11-01/test-runs/run-a?api-version=2022-11-01",
			body:       []byte(`{"displayName":"run"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusBadRequest,
			wantCode:   "InvalidRequest",
		},
		{
			name:       "run missing test",
			method:     http.MethodPatch,
			path:       "/azure/loadtesting-data-plane/2022-11-01/test-runs/run-a?api-version=2022-11-01",
			body:       []byte(`{"testId":"missing-test"}`),
			headers:    jsonHeaders,
			wantStatus: http.StatusNotFound,
			wantCode:   "TestNotFound",
		},
		{
			name:       "metrics missing params",
			method:     http.MethodPost,
			path:       "/azure/loadtesting-data-plane/2022-11-01/test-runs/run-a/metrics?api-version=2022-11-01",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "TestRunNotFound",
		},
		{
			name:       "file upload empty body",
			method:     http.MethodPut,
			path:       "/azure/loadtesting-data-plane/2022-11-01/tests/test-a/files/test-plan.jmx?api-version=2022-11-01",
			headers:    headers,
			wantStatus: http.StatusNotFound,
			wantCode:   "TestNotFound",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := providerContractRequest(t, ts, tt.method, tt.path, tt.body, tt.headers)
			body := providerContractBody(t, resp)
			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("expected %d, got %d body=%s", tt.wantStatus, resp.StatusCode, string(body))
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("unmarshal response JSON: %v body=%s", err, string(body))
			}
			errorPayload, _ := payload["error"].(map[string]any)
			if errorPayload["code"] != tt.wantCode {
				t.Fatalf("expected code %q, got %#v", tt.wantCode, payload)
			}
		})
	}
}
