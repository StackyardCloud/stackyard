package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newSchedulerTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

// schedulerCall sends a scheduler-signed REST call and decodes the reply.
func schedulerCall(t *testing.T, ts *httptest.Server, method, path, body string, want int) map[string]any {
	t.Helper()
	headers := map[string]string{}
	var raw []byte
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "scheduler")
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		t.Fatalf("%s %s: status %d (want %d) %s", method, path, resp.StatusCode, want, data)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("%s %s: Content-Type %q", method, path, ct)
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("%s %s: decode %s: %v", method, path, data, err)
	}
	return out
}

// schedulerCallError expects an error status and returns the error code.
func schedulerCallError(t *testing.T, ts *httptest.Server, method, path, body string, want int) string {
	t.Helper()
	headers := map[string]string{}
	var raw []byte
	if body != "" {
		raw = []byte(body)
		headers["Content-Type"] = "application/json"
	}
	resp := signedRequestWithService(t, method, ts.URL+path, raw, headers, "scheduler")
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != want {
		t.Fatalf("%s %s: status %d (want %d) %s", method, path, resp.StatusCode, want, data)
	}
	if !strings.Contains(string(data), `"Message"`) {
		t.Fatalf("%s %s: error body %s", method, path, data)
	}
	return resp.Header.Get("X-Amzn-ErrorType")
}

func TestSchedulerStage0Operations(t *testing.T) {
	if len(schedulerOperations) != 12 || len(schedulerOperationByName) != 12 {
		t.Fatalf("operations: %d / %d", len(schedulerOperations), len(schedulerOperationByName))
	}
	for _, name := range []string{"CreateSchedule", "UpdateSchedule", "ListScheduleGroups", "UntagResource"} {
		if _, ok := schedulerOperationByName[name]; !ok {
			t.Fatalf("missing %s", name)
		}
	}
}

func TestSchedulerStage0Routing(t *testing.T) {
	_, ts := newSchedulerTestServer(t)
	if code := schedulerCallError(t, ts, http.MethodPatch, "/schedules/x", "", http.StatusNotImplemented); code != "NotImplementedException" {
		t.Fatalf("unknown route: %s", code)
	}
	// A request signed for another service is not claimed: an S3 bucket
	// called schedules keeps working.
	resp := signedRequestWithService(t, http.MethodPut, ts.URL+"/schedules", nil, nil, "s3")
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("S3 bucket schedules: %d", resp.StatusCode)
	}
	// A request whose body length isn't declared is refused before signing.
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/schedule-groups/g1", io.MultiReader(strings.NewReader(`{}`)))
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+testAccessKey+"/20260101/"+testRegion+"/scheduler/aws4_request, SignedHeaders=host, Signature=00")
	chunked, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	chunked.Body.Close()
	if chunked.StatusCode != http.StatusBadRequest {
		t.Fatalf("chunked body: %d", chunked.StatusCode)
	}
}
