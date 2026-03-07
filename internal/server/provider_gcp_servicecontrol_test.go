package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPServiceControlRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceControlContractServer(t)
	base := "/gcp/v1/services/stackyard.googleapis.com"

	assertGCPServiceControlSuccess(t, ts, http.MethodPost, base+":check", []byte(`{
		"operation": {
			"operationId": "check-op-1",
			"consumerId": "project:stackyard",
			"startTime": "2026-01-01T00:00:00Z"
		}
	}`), "operationId")
	assertGCPServiceControlSuccess(t, ts, http.MethodPost, base+":report", []byte(`{
		"operations": [{
			"operationId": "report-op-1",
			"consumerId": "project:stackyard",
			"startTime": "2026-01-01T00:00:00Z",
			"endTime": "2026-01-01T00:01:00Z"
		}]
	}`), "serviceRolloutId")
	assertGCPServiceControlSuccess(t, ts, http.MethodPost, base+":allocateQuota", []byte(`{
		"allocateOperation": {
			"operationId": "quota-op-1",
			"consumerId": "project:stackyard",
			"methodName": "google.example.v1.Service/Call"
		}
	}`), "quotaMetrics")
}

func TestGCPServiceControlRouter_CheckRequiresOperation(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceControlContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services/stackyard.googleapis.com:check", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicecontrol",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicecontrol router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceControlRouter_CheckRequiresStartTime(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceControlContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services/stackyard.googleapis.com:check", []byte(`{
		"operation": {
			"operationId": "check-op-1",
			"consumerId": "project:stackyard"
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicecontrol",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicecontrol router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceControlRouter_ReportRequiresOperations(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceControlContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services/stackyard.googleapis.com:report", []byte(`{"operations":[]}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicecontrol",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicecontrol router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceControlRouter_ReportOperationRequiresConsumerID(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceControlContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services/stackyard.googleapis.com:report", []byte(`{
		"operations": [{
			"operationId": "report-op-1",
			"startTime": "2026-01-01T00:00:00Z"
		}]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicecontrol",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicecontrol router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceControlRouter_AllocateQuotaRequiresAllocateOperation(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceControlContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services/stackyard.googleapis.com:allocateQuota", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicecontrol",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicecontrol router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceControlRouter_AllocateQuotaRequiresConsumerID(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceControlContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services/stackyard.googleapis.com:allocateQuota", []byte(`{
		"allocateOperation": {
			"operationId": "quota-op-1",
			"methodName": "google.example.v1.Service/Call"
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicecontrol",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicecontrol router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceControlRouter_AllocateQuotaRejectsMethodNameAndQuotaMetrics(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceControlContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services/stackyard.googleapis.com:allocateQuota", []byte(`{
		"allocateOperation": {
			"operationId": "quota-op-1",
			"consumerId": "project:stackyard",
			"methodName": "google.example.v1.Service/Call",
			"quotaMetrics": [{
				"metricName": "serviceruntime.googleapis.com/api/consumer/quota_used_count",
				"metricValues": [{"int64Value": "1"}]
			}]
		}
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicecontrol",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp servicecontrol router, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPServiceControlRouter_TypedOutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPServiceControlContractServer(t)
	base := "/gcp/v1/services/stackyard.googleapis.com"
	headers := map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "servicecontrol",
	}

	checkResp := providerContractRequest(t, ts, http.MethodPost, base+":check", []byte(`{
		"operation": {
			"operationId": "check-op-typed",
			"consumerId": "project:stackyard",
			"startTime": "2026-01-01T00:00:00Z"
		}
	}`), headers)
	if checkResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicecontrol check, got %d body=%s", checkResp.StatusCode, string(providerContractBody(t, checkResp)))
	}
	checkBody := providerContractJSONMap(t, checkResp)
	if _, ok := checkBody["operationId"].(string); !ok {
		t.Fatalf("expected check operationId string, got %#v", checkBody["operationId"])
	}
	if _, ok := checkBody["serviceConfigId"].(string); !ok {
		t.Fatalf("expected check serviceConfigId string, got %#v", checkBody["serviceConfigId"])
	}
	checkInfo, ok := checkBody["checkInfo"].(map[string]any)
	if !ok {
		t.Fatalf("expected checkInfo object, got %#v", checkBody["checkInfo"])
	}
	if _, ok := checkInfo["consumerInfo"].(map[string]any); !ok {
		t.Fatalf("expected checkInfo.consumerInfo object, got %#v", checkInfo["consumerInfo"])
	}

	reportResp := providerContractRequest(t, ts, http.MethodPost, base+":report", []byte(`{
		"operations": [{
			"operationId": "report-op-typed",
			"consumerId": "project:stackyard",
			"startTime": "2026-01-01T00:00:00Z"
		}]
	}`), headers)
	if reportResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicecontrol report, got %d body=%s", reportResp.StatusCode, string(providerContractBody(t, reportResp)))
	}
	reportBody := providerContractJSONMap(t, reportResp)
	if _, ok := reportBody["serviceRolloutId"].(string); !ok {
		t.Fatalf("expected report serviceRolloutId string, got %#v", reportBody["serviceRolloutId"])
	}
	if _, ok := reportBody["reportErrors"].([]any); !ok {
		t.Fatalf("expected reportErrors array, got %#v", reportBody["reportErrors"])
	}

	quotaResp := providerContractRequest(t, ts, http.MethodPost, base+":allocateQuota", []byte(`{
		"allocateOperation": {
			"operationId": "quota-op-typed",
			"consumerId": "project:stackyard",
			"methodName": "google.example.v1.Service/Call"
		}
	}`), headers)
	if quotaResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicecontrol allocateQuota, got %d body=%s", quotaResp.StatusCode, string(providerContractBody(t, quotaResp)))
	}
	quotaBody := providerContractJSONMap(t, quotaResp)
	quotaMetrics, ok := quotaBody["quotaMetrics"].([]any)
	if !ok || len(quotaMetrics) == 0 {
		t.Fatalf("expected quotaMetrics array, got %#v", quotaBody["quotaMetrics"])
	}
	quotaMetric, _ := quotaMetrics[0].(map[string]any)
	if _, ok := quotaMetric["metricName"].(string); !ok {
		t.Fatalf("expected quotaMetrics[0].metricName string, got %#v", quotaMetric["metricName"])
	}
}

func TestGCPServiceControlRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v1/services/stackyard.googleapis.com:check?stackyard_contract_probe=1&typedSuccess=1", []byte(`{}`), map[string]string{
		"Content-Type": "application/json",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicecontrol contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "servicecontrol" {
		t.Fatalf("expected service=servicecontrol, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["operationId"].(string); !ok {
		t.Fatalf("expected typed operationId in contract probe response, got %#v", body["operationId"])
	}
}

func newGCPServiceControlContractServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})
}

func assertGCPServiceControlSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "servicecontrol",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp servicecontrol router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
