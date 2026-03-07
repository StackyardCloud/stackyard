package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGCPTraceV1Router_RESTRoutesRecognized(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceV1ContractServer(t)

	traceID := "0123456789abcdef0123456789abcdef"
	getPath := "/gcp/v1/projects/stackyard/traces/" + traceID

	assertGCPTraceV1Success(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/traces?pageSize=1&view=COMPLETE", nil, `"traces":[`)
	assertGCPTraceV1Success(t, ts, http.MethodGet, getPath, nil, `"traceId":"`+traceID+`"`)
	assertGCPTraceV1Success(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/traces", []byte(`{
		"traces":[{
			"projectId":"stackyard",
			"traceId":"0123456789abcdef0123456789abcdef",
			"spans":[{
				"spanId":"3001",
				"kind":1,
				"name":"/stackyard/tracev1/patch",
				"startTime":"2026-01-01T00:00:00Z",
				"endTime":"2026-01-01T00:00:01Z",
				"labels":{"/component":"trace_v1"}
			}]
		}]
	}`), `{}`)

	assertGCPTraceV1Success(t, ts, http.MethodPost, gcpTraceV1GRPCListTracesPath, []byte(`{
		"projectId":"stackyard",
		"pageSize":1,
		"view":"COMPLETE"
	}`), `"traces":[`)
	assertGCPTraceV1Success(t, ts, http.MethodPost, gcpTraceV1GRPCGetTracePath, []byte(`{
		"projectId":"stackyard",
		"traceId":"0123456789abcdef0123456789abcdef"
	}`), `"traceId":"0123456789abcdef0123456789abcdef"`)
	assertGCPTraceV1Success(t, ts, http.MethodPost, gcpTraceV1GRPCPatchTracesPath, []byte(`{
		"projectId":"stackyard",
		"traces":{
			"traces":[{
				"projectId":"stackyard",
				"traceId":"0123456789abcdef0123456789abcdef",
				"spans":[{
					"spanId":"3002",
					"kind":1,
					"name":"/stackyard/tracev1/grpc-patch",
					"startTime":"2026-01-01T00:00:00Z",
					"endTime":"2026-01-01T00:00:01Z"
				}]
			}]
		}
	}`), `{}`)
}

func TestGCPTraceV1Router_ListRejectsInvalidView(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceV1ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/traces?view=UNSUPPORTED", nil, map[string]string{
		"X-Stackyard-GCP-Service": "trace_v1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp trace_v1 list traces invalid view, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceV1Router_ListRejectsInvalidFilter(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceV1ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/traces?filter=not_supported_filter", nil, map[string]string{
		"X-Stackyard-GCP-Service": "trace_v1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp trace_v1 list traces invalid filter, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceV1Router_ListRejectsInvalidPageToken(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceV1ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/traces?pageToken=bad", nil, map[string]string{
		"X-Stackyard-GCP-Service": "trace_v1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp trace_v1 list traces invalid page token, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceV1Router_ListRejectsInvalidTimeWindow(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceV1ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/traces?startTime=2026-01-01T00:00:10Z&endTime=2026-01-01T00:00:00Z", nil, map[string]string{
		"X-Stackyard-GCP-Service": "trace_v1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp trace_v1 list traces invalid time window, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceV1Router_GetRejectsInvalidTraceID(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceV1ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/traces/not-a-trace-id", nil, map[string]string{
		"X-Stackyard-GCP-Service": "trace_v1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp trace_v1 get trace invalid trace id, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceV1Router_GetMissingTraceReturnsNotFound(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceV1ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/traces/ffffffffffffffffffffffffffffffff", nil, map[string]string{
		"X-Stackyard-GCP-Service": "trace_v1",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from gcp trace_v1 get trace missing, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"NotFound"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceV1Router_PatchRequiresTraces(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceV1ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/traces", []byte(`{}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace_v1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp trace_v1 patch traces missing traces, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceV1Router_PatchRejectsInvalidSpanID(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceV1ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/traces", []byte(`{
		"traces":[{
			"traceId":"0123456789abcdef0123456789abcdef",
			"spans":[{
				"spanId":"0",
				"kind":1,
				"name":"/stackyard/tracev1/patch",
				"startTime":"2026-01-01T00:00:00Z",
				"endTime":"2026-01-01T00:00:01Z"
			}]
		}]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace_v1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp trace_v1 patch traces invalid span id, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"InvalidArgument"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceV1Router_PatchSpanDurationFailedPrecondition(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceV1ContractServer(t)
	resp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/traces", []byte(`{
		"traces":[{
			"traceId":"0123456789abcdef0123456789abcdef",
			"spans":[{
				"spanId":"3001",
				"kind":1,
				"name":"/stackyard/tracev1/patch",
				"startTime":"2026-01-01T00:00:00Z",
				"endTime":"2026-01-02T12:00:00Z"
			}]
		}]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace_v1",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 from gcp trace_v1 patch traces duration precondition, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	if body := string(providerContractBody(t, resp)); !strings.Contains(body, `"error":"FailedPrecondition"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestGCPTraceV1Router_OutputShapes(t *testing.T) {
	t.Parallel()

	ts := newGCPTraceV1ContractServer(t)

	listResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/traces?pageSize=1&view=COMPLETE", nil, map[string]string{
		"X-Stackyard-GCP-Service": "trace_v1",
	})
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp trace_v1 list traces, got %d body=%s", listResp.StatusCode, string(providerContractBody(t, listResp)))
	}
	listBody := providerContractJSONMap(t, listResp)
	traces, _ := listBody["traces"].([]any)
	if len(traces) == 0 {
		t.Fatalf("expected traces array, got %#v", listBody["traces"])
	}
	firstTrace, _ := traces[0].(map[string]any)
	if _, ok := firstTrace["projectId"].(string); !ok {
		t.Fatalf("expected trace.projectId string, got %#v", firstTrace["projectId"])
	}
	if _, ok := firstTrace["traceId"].(string); !ok {
		t.Fatalf("expected trace.traceId string, got %#v", firstTrace["traceId"])
	}
	traceSpans, _ := firstTrace["spans"].([]any)
	if len(traceSpans) == 0 {
		t.Fatalf("expected trace.spans array, got %#v", firstTrace["spans"])
	}
	firstSpan, _ := traceSpans[0].(map[string]any)
	if _, ok := firstSpan["spanId"].(string); !ok {
		t.Fatalf("expected span.spanId string, got %#v", firstSpan["spanId"])
	}
	if _, ok := firstSpan["kind"].(float64); !ok {
		t.Fatalf("expected span.kind number, got %#v", firstSpan["kind"])
	}
	if _, ok := firstSpan["name"].(string); !ok {
		t.Fatalf("expected span.name string, got %#v", firstSpan["name"])
	}
	if _, ok := firstSpan["startTime"].(string); !ok {
		t.Fatalf("expected span.startTime string, got %#v", firstSpan["startTime"])
	}
	if _, ok := firstSpan["endTime"].(string); !ok {
		t.Fatalf("expected span.endTime string, got %#v", firstSpan["endTime"])
	}
	if _, ok := firstSpan["labels"].(map[string]any); !ok {
		t.Fatalf("expected span.labels object, got %#v", firstSpan["labels"])
	}
	if _, ok := listBody["nextPageToken"].(string); !ok {
		t.Fatalf("expected nextPageToken string, got %#v", listBody["nextPageToken"])
	}

	getResp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/traces/0123456789abcdef0123456789abcdef", nil, map[string]string{
		"X-Stackyard-GCP-Service": "trace_v1",
	})
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp trace_v1 get trace, got %d body=%s", getResp.StatusCode, string(providerContractBody(t, getResp)))
	}
	getBody := providerContractJSONMap(t, getResp)
	if _, ok := getBody["traceId"].(string); !ok {
		t.Fatalf("expected get trace.traceId string, got %#v", getBody["traceId"])
	}

	patchResp := providerContractRequest(t, ts, http.MethodPatch, "/gcp/v1/projects/stackyard/traces", []byte(`{
		"traces":[{
			"traceId":"0123456789abcdef0123456789abcdef",
			"spans":[{
				"spanId":"3001",
				"kind":1,
				"name":"/stackyard/tracev1/patch",
				"startTime":"2026-01-01T00:00:00Z",
				"endTime":"2026-01-01T00:00:01Z"
			}]
		}]
	}`), map[string]string{
		"Content-Type":            "application/json",
		"X-Stackyard-GCP-Service": "trace_v1",
	})
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp trace_v1 patch traces, got %d body=%s", patchResp.StatusCode, string(providerContractBody(t, patchResp)))
	}
	patchBody := providerContractJSONMap(t, patchResp)
	if len(patchBody) != 0 {
		t.Fatalf("expected empty object response from patch traces, got %#v", patchBody)
	}
}

func TestGCPTraceV1Router_OutputShapeContractProbe(t *testing.T) {
	t.Parallel()

	ts := newProviderContractServer(t, Config{
		Addr:        "127.0.0.1:0",
		Providers:   []string{providerGCP},
		GCPAuthMode: "emulator",
		AccessKey:   testAccessKey,
		SecretKey:   testSecretKey,
		LogLevel:    "error",
	})

	resp := providerContractRequest(t, ts, http.MethodGet, "/gcp/v1/projects/stackyard/locations/us-central1/trace_v1?stackyard_contract_probe=1&typedSuccess=1", nil, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp trace_v1 contract probe, got %d body=%s", resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := providerContractJSONMap(t, resp)
	if got, _ := body["service"].(string); got != "trace_v1" {
		t.Fatalf("expected service=trace_v1, got %#v", body["service"])
	}
	if got, _ := body["provider"].(string); got != providerGCP {
		t.Fatalf("expected provider=%s, got %#v", providerGCP, body["provider"])
	}
	if _, ok := body["name"].(string); !ok {
		t.Fatalf("expected name string in contract probe response, got %#v", body["name"])
	}
}

func newGCPTraceV1ContractServer(t *testing.T) *httptest.Server {
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

func assertGCPTraceV1Success(t *testing.T, ts *httptest.Server, method, path string, payload []byte, expectBodyFragment string) {
	t.Helper()

	headers := map[string]string{
		"X-Stackyard-GCP-Service": "trace_v1",
	}
	if payload != nil {
		headers["Content-Type"] = "application/json"
	}
	resp := providerContractRequest(t, ts, method, path, payload, headers)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from gcp trace_v1 router for %s %s, got %d body=%s", method, path, resp.StatusCode, string(providerContractBody(t, resp)))
	}
	body := string(providerContractBody(t, resp))
	if !strings.Contains(body, expectBodyFragment) {
		t.Fatalf("unexpected response body for %s %s: %s", method, path, body)
	}
}
