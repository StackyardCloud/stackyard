package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPTraceRouter_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceContractServer(t)
	traceID := "0123456789abcdef0123456789abcdef"
	spanID := "1111111111111111"
	spanName := "projects/stackyard/traces/" + traceID + "/spans/" + spanID

	assertGCPTraceSuccess(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/traces:batchWrite", []byte(`{
		"spans":[{
			"name":"`+spanName+`",
			"spanId":"`+spanID+`",
			"displayName":{"value":"stackyard.trace.batch"},
			"startTime":"2026-01-01T00:00:00Z",
			"endTime":"2026-01-01T00:00:01Z"
		}]
	}`), `{}`)
	assertGCPTraceSuccess(t, ts, http.MethodPost, "/gcp/v2/"+spanName, []byte(`{
		"spanId":"`+spanID+`",
		"displayName":{"value":"stackyard.trace.create"},
		"startTime":"2026-01-01T00:00:00Z",
		"endTime":"2026-01-01T00:00:01Z",
		"spanKind":"SERVER"
	}`), `"name":"`+spanName+`"`)

	assertGCPTraceSuccess(t, ts, http.MethodPost, gcpTraceGRPCBatchWriteSpansPath, []byte(`{
		"name":"projects/stackyard",
		"spans":[{
			"name":"`+spanName+`",
			"spanId":"`+spanID+`",
			"displayName":{"value":"stackyard.trace.grpc.batch"},
			"startTime":"2026-01-01T00:00:00Z",
			"endTime":"2026-01-01T00:00:01Z"
		}]
	}`), `{}`)
	assertGCPTraceSuccess(t, ts, http.MethodPost, gcpTraceGRPCCreateSpanPath, []byte(`{
		"name":"`+spanName+`",
		"spanId":"`+spanID+`",
		"displayName":{"value":"stackyard.trace.grpc.create"},
		"startTime":"2026-01-01T00:00:00Z",
		"endTime":"2026-01-01T00:00:01Z"
	}`), `"name":"`+spanName+`"`)
}

func TestGCPTraceRouter_BatchWriteRequiresSpans(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/traces:batchWrite", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp trace batchWrite, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceRouter_BatchWriteRejectsPathBodyProjectMismatch(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/traces:batchWrite", []byte(`{
		"name":"projects/other-project",
		"spans":[{
			"name":"projects/stackyard/traces/0123456789abcdef0123456789abcdef/spans/1111111111111111",
			"spanId":"1111111111111111",
			"displayName":{"value":"stackyard.trace.batch"},
			"startTime":"2026-01-01T00:00:00Z",
			"endTime":"2026-01-01T00:00:01Z"
		}]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp trace batchWrite mismatch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceRouter_CreateSpanRequiresDisplayName(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/traces/0123456789abcdef0123456789abcdef/spans/1111111111111111", []byte(`{
		"spanId":"1111111111111111",
		"startTime":"2026-01-01T00:00:00Z",
		"endTime":"2026-01-01T00:00:01Z"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp trace create span, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceRouter_CreateSpanRejectsSpanIDMismatch(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/traces/0123456789abcdef0123456789abcdef/spans/1111111111111111", []byte(`{
		"spanId":"2222222222222222",
		"displayName":{"value":"stackyard.trace.mismatch"},
		"startTime":"2026-01-01T00:00:00Z",
		"endTime":"2026-01-01T00:00:01Z"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp trace create span mismatch, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceRouter_CreateSpanRejectsEndBeforeStart(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/traces/0123456789abcdef0123456789abcdef/spans/1111111111111111", []byte(`{
		"spanId":"1111111111111111",
		"displayName":{"value":"stackyard.trace.order"},
		"startTime":"2026-01-01T00:00:01Z",
		"endTime":"2026-01-01T00:00:00Z"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp trace create span timestamp order, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceRouter_CreateSpanDurationFailedPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/traces/0123456789abcdef0123456789abcdef/spans/1111111111111111", []byte(`{
		"spanId":"1111111111111111",
		"displayName":{"value":"stackyard.trace.duration"},
		"startTime":"2026-01-01T00:00:00Z",
		"endTime":"2026-01-02T12:00:00Z"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp trace create span duration precondition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceRouter_CreateSpanReturnsNotFoundForMissingProject(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/missing-project/traces/0123456789abcdef0123456789abcdef/spans/1111111111111111", []byte(`{
		"spanId":"1111111111111111",
		"displayName":{"value":"stackyard.trace.missing"},
		"startTime":"2026-01-01T00:00:00Z",
		"endTime":"2026-01-01T00:00:01Z"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp trace create span missing project, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceRouter_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceContractServer(t)
	traceID := "0123456789abcdef0123456789abcdef"
	spanID := "1111111111111111"
	spanName := "projects/stackyard/traces/" + traceID + "/spans/" + spanID

	createResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/"+spanName, []byte(`{
		"spanId":"`+spanID+`",
		"displayName":{"value":"stackyard.trace.output"},
		"startTime":"2026-01-01T00:00:00Z",
		"endTime":"2026-01-01T00:00:01Z"
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace",
	})
	if createResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp trace create span, got %d body=%s", createResp.StatusCode, string(providerContractBody(t, createResp)))
	}
	createBody := providerContractJSONMap(t, createResp)
	if _, ok := createBody["name"].(string); !ok {
		t.Fatalf("expected span.name string, got %#v", createBody["name"])
	}
	if _, ok := createBody["spanId"].(string); !ok {
		t.Fatalf("expected span.spanId string, got %#v", createBody["spanId"])
	}
	displayName, _ := createBody["displayName"].(map[string]any)
	if _, ok := displayName["value"].(string); !ok {
		t.Fatalf("expected displayName.value string, got %#v", displayName["value"])
	}
	if _, ok := createBody["startTime"].(string); !ok {
		t.Fatalf("expected startTime string, got %#v", createBody["startTime"])
	}
	if _, ok := createBody["endTime"].(string); !ok {
		t.Fatalf("expected endTime string, got %#v", createBody["endTime"])
	}
	if _, ok := createBody["sameProcessAsParentSpan"].(bool); !ok {
		t.Fatalf("expected sameProcessAsParentSpan bool, got %#v", createBody["sameProcessAsParentSpan"])
	}
	attributes, _ := createBody["attributes"].(map[string]any)
	if _, ok := attributes["attributeMap"].(map[string]any); !ok {
		t.Fatalf("expected attributes.attributeMap object, got %#v", attributes["attributeMap"])
	}

	batchResp := providerContractRequest(t, ts, http.MethodPost, "/gcp/v2/projects/stackyard/traces:batchWrite", []byte(`{
		"spans":[{
			"name":"`+spanName+`",
			"spanId":"`+spanID+`",
			"displayName":{"value":"stackyard.trace.batch.output"},
			"startTime":"2026-01-01T00:00:00Z",
			"endTime":"2026-01-01T00:00:01Z"
		}]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace",
	})
	if batchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp trace batchWrite, got %d body=%s", batchResp.StatusCode, string(providerContractBody(t, batchResp)))
	}
	batchBody := providerContractJSONMap(t, batchResp)
	if len(batchBody) != 0 {
		t.Fatalf("expected empty object response from batchWrite, got %#v", batchBody)
	}
}

func TestGCPTraceRouter_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/trace?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp trace contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "trace" {
		t.Fatalf("expected service=trace, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected name string in contract probe response, got %#v", body["name"])
	}
}

func newGCPTraceContractServer(t *testing.T) *httptest.Server {
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

func assertGCPTraceSuccess(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "trace",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp trace router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
