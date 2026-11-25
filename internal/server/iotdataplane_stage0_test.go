package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stackyard/stackyard/internal/awsmodels"
)

func iotdpServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// iotdpCall sends a signed request and returns the status, the decoded JSON
// body and the X-Amzn-ErrorType header.
func iotdpCall(t *testing.T, ts *httptest.Server, service, method, path, body string, headers map[string]string) (int, map[string]any, string) {
	t.Helper()
	var raw []byte
	if body != "" {
		raw = []byte(body)
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, service)
	data := mustBody(t, resp)
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("%s %s: decode %q: %v", method, path, data, err)
		}
	}
	return resp.StatusCode, out, resp.Header.Get("X-Amzn-ErrorType")
}

func iotdpOK(t *testing.T, ts *httptest.Server, service, method, path, body string) map[string]any {
	t.Helper()
	status, out, code := iotdpCall(t, ts, service, method, path, body, nil)
	if status != http.StatusOK {
		t.Fatalf("%s %s: %d %s %v", method, path, status, code, out)
	}
	return out
}

func iotdpErr(t *testing.T, ts *httptest.Server, service, method, path, body string, wantStatus int, wantCode string) {
	t.Helper()
	status, out, code := iotdpCall(t, ts, service, method, path, body, nil)
	if status != wantStatus || code != wantCode {
		t.Fatalf("%s %s: got %d %s %v, want %d %s", method, path, status, code, out, wantStatus, wantCode)
	}
	if msg, _ := out["message"].(string); msg == "" {
		t.Fatalf("%s %s: error without message: %v", method, path, out)
	}
}

func kssCall(t *testing.T, ts *httptest.Server, op, body string) (int, map[string]any) {
	t.Helper()
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(body), map[string]string{
		"Content-Type": kssContentType, "X-Amz-Target": kssTargetPrefix + op,
	}, "cassandra")
	data := mustBody(t, resp)
	if ct := resp.Header.Get("Content-Type"); ct != kssContentType {
		t.Fatalf("%s: content type %q", op, ct)
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("%s: decode %q: %v", op, data, err)
	}
	return resp.StatusCode, out
}

func TestIoTDataPlaneBatchStage0CatalogsMatchModels(t *testing.T) {
	jobs := awsmodels.IoTJobsDataPlaneModel()
	if len(iotJobsDataPlaneOperations) != len(jobs.Operations) {
		t.Fatalf("iot-jobs-data: catalog %d, model %d", len(iotJobsDataPlaneOperations), len(jobs.Operations))
	}
	for _, op := range iotJobsDataPlaneOperations {
		if _, ok := jobs.Operations[op.Name]; !ok {
			t.Errorf("iot-jobs-data: %s is not in the model", op.Name)
		}
	}
	kss := awsmodels.KeyspacesStreamsModel()
	if len(keyspacesStreamsOperations) != len(kss.Operations) {
		t.Fatalf("keyspacesstreams: catalog %d, model %d", len(keyspacesStreamsOperations), len(kss.Operations))
	}
	for _, op := range keyspacesStreamsOperations {
		if _, ok := kss.Operations[op.Name]; !ok {
			t.Errorf("keyspacesstreams: %s is not in the model", op.Name)
		}
	}
	if len(iotDataPlaneOperations) != 11 {
		t.Fatalf("iot-data: catalog has %d operations, the CLI 2.42.1 model has 11", len(iotDataPlaneOperations))
	}
}

// TestIoTDataPlaneBatchStage0Routing proves the data planes claim only their
// own signing names and IoT, Keyspaces and Device Advisor keep their traffic.
func TestIoTDataPlaneBatchStage0Routing(t *testing.T) {
	_, ts := iotdpServer(t)
	// The same path answers differently per signing name.
	iotdpErr(t, ts, "iotdata", http.MethodGet, "/things/pump/shadow", "", http.StatusNotFound, "ResourceNotFoundException")
	if out := iotdpOK(t, ts, "iot", http.MethodGet, "/things/pump", ""); out["thingName"] != "pump" {
		t.Fatalf("iot DescribeThing: %v", out)
	}
	iotdpErr(t, ts, "iot-jobs-data", http.MethodGet, "/things/pump/jobs/j1", "", http.StatusNotFound, "ResourceNotFoundException")
	if out := iotdpOK(t, ts, "iot", http.MethodGet, "/things/pump/jobs/j1", ""); out["jobId"] == nil {
		t.Fatalf("iot DescribeJobExecution: %v", out)
	}
	if out := iotdpOK(t, ts, "iot", http.MethodGet, "/tags?resourceArn=arn:aws:iot:us-east-1:123456789012:thing/stackyard-thing", ""); out["tags"] == nil {
		t.Fatalf("iot ListTagsForResource: %v", out)
	}
	if status, out, _ := iotdpCall(t, ts, "iotdeviceadvisor", http.MethodGet, "/suiteDefinitions", "", nil); status != http.StatusOK {
		t.Fatalf("Device Advisor: %d %v", status, out)
	}
	// Keyspaces keeps its target; Keyspaces Streams takes its own.
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(`{}`), map[string]string{
		"Content-Type": kssContentType, "X-Amz-Target": "KeyspacesService.ListKeyspaces",
	}, "cassandra")
	if body := mustBody(t, resp); resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "keyspaces") {
		t.Fatalf("keyspaces ListKeyspaces: %d %s", resp.StatusCode, body)
	}
	if status, out := kssCall(t, ts, "ListStreams", `{}`); status != http.StatusOK || out["streams"] == nil {
		t.Fatalf("ListStreams: %d %v", status, out)
	}
	// Unknown operations and members are refused.
	if status, out := kssCall(t, ts, "Nope", `{}`); status != http.StatusBadRequest || out["__type"] != "UnknownOperationException" {
		t.Fatalf("unknown op: %d %v", status, out)
	}
	if status, out := kssCall(t, ts, "ListStreams", `{"bogus":1}`); status != http.StatusBadRequest || out["__type"] != "ValidationException" || out["errorCode"] != "InvalidFormat" {
		t.Fatalf("unknown member: %d %v", status, out)
	}
	iotdpErr(t, ts, "iot-jobs-data", http.MethodPost, "/things/pump/jobs/j1", `{"status":"IN_PROGRESS","bogus":1}`, http.StatusBadRequest, "InvalidRequestException")
	iotdpErr(t, ts, "iotdata", http.MethodPatch, "/things/pump/shadow", "", http.StatusMethodNotAllowed, "MethodNotAllowedException")
	// A data-plane request signed for another service is not served by it.
	iotdpErr(t, ts, "iotdata", http.MethodGet, "/command-executions", "", http.StatusMethodNotAllowed, "MethodNotAllowedException")
}

func TestIoTDataPlaneStage0RefusesChunkedBodies(t *testing.T) {
	_, ts := iotdpServer(t)
	for _, svc := range []string{"iotdata", "iot-jobs-data"} {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/topics/a", strings.NewReader("x"))
		req.ContentLength = -1 // chunked
		req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/us-east-1/"+svc+"/aws4_request, SignedHeaders=host, Signature=x")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body := mustBody(t, resp)
		if resp.StatusCode == http.StatusOK {
			t.Fatalf("%s accepted a chunked body: %s", svc, body)
		}
	}
}
