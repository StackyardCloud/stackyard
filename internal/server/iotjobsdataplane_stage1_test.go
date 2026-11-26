package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func iotjCreateJob(t *testing.T, ts *httptest.Server, id, body string) {
	t.Helper()
	iotdpOK(t, ts, "iot", http.MethodPut, "/jobs/"+id, body)
}

const iotjPumpTarget = `"targets":["arn:aws:iot:us-east-1:123456789012:thing/pump-1"]`

func TestIoTJobsDataPlaneStage1ExecutionLifecycle(t *testing.T) {
	srv, ts := iotdpServer(t)
	iotjCreateJob(t, ts, "fw-1", `{`+iotjPumpTarget+`,"document":"{\"v\":1}","timeoutConfig":{"inProgressTimeoutInMinutes":30}}`)
	iotjCreateJob(t, ts, "fw-2", `{`+iotjPumpTarget+`,"document":"{\"v\":2}"}`)
	iotjCreateJob(t, ts, "other", `{"targets":["arn:aws:iot:us-east-1:123456789012:thing/valve-9"]}`)

	pending := iotdpOK(t, ts, "iot-jobs-data", http.MethodGet, "/things/pump-1/jobs", "")
	if q := pending["queuedJobs"].([]any); len(q) != 2 || q[0].(map[string]any)["jobId"] != "fw-1" {
		t.Fatalf("pending: %v", pending)
	}
	iotdpErr(t, ts, "iot-jobs-data", http.MethodGet, "/things/pump-1/jobs/other", "", http.StatusNotFound, "ResourceNotFoundException")

	next := iotdpOK(t, ts, "iot-jobs-data", http.MethodPut, "/things/pump-1/jobs/$next", `{"statusDetails":{"step":"download"},"stepTimeoutInMinutes":5}`)["execution"].(map[string]any)
	if next["jobId"] != "fw-1" || next["status"] != "IN_PROGRESS" || next["versionNumber"] != float64(2) || next["jobDocument"] != `{"v":1}` {
		t.Fatalf("StartNext: %v", next)
	}
	if secs := next["approximateSecondsBeforeTimedOut"].(float64); secs < 290 || secs > 300 {
		t.Fatalf("step timeout: %v", secs)
	}
	// $next keeps returning the in-progress execution.
	if again := iotdpOK(t, ts, "iot-jobs-data", http.MethodPut, "/things/pump-1/jobs/$next", "")["execution"].(map[string]any); again["jobId"] != "fw-1" || again["versionNumber"] != float64(2) {
		t.Fatalf("StartNext again: %v", again)
	}
	iotdpErr(t, ts, "iot-jobs-data", http.MethodPost, "/things/pump-1/jobs/fw-1", `{"status":"SUCCEEDED","expectedVersion":1}`, http.StatusConflict, "InvalidStateTransitionException")
	iotdpErr(t, ts, "iot-jobs-data", http.MethodPost, "/things/pump-1/jobs/fw-1", `{"status":"CANCELED"}`, http.StatusBadRequest, "InvalidRequestException")
	iotdpErr(t, ts, "iot-jobs-data", http.MethodPost, "/things/pump-1/jobs/fw-1", `{"status":"IN_PROGRESS","stepTimeoutInMinutes":0}`, http.StatusBadRequest, "InvalidRequestException")
	up := iotdpOK(t, ts, "iot-jobs-data", http.MethodPost, "/things/pump-1/jobs/fw-1", `{"status":"SUCCEEDED","expectedVersion":2,"includeJobExecutionState":true,"includeJobDocument":true}`)
	if st := up["executionState"].(map[string]any); st["status"] != "SUCCEEDED" || st["versionNumber"] != float64(3) || up["jobDocument"] != `{"v":1}` {
		t.Fatalf("update: %v", up)
	}
	iotdpErr(t, ts, "iot-jobs-data", http.MethodPost, "/things/pump-1/jobs/fw-1", `{"status":"FAILED"}`, http.StatusConflict, "InvalidStateTransitionException")
	desc := iotdpOK(t, ts, "iot-jobs-data", http.MethodGet, "/things/pump-1/jobs/fw-1?includeJobDocument=true&executionNumber=1", "")["execution"].(map[string]any)
	if desc["status"] != "SUCCEEDED" || desc["statusDetails"].(map[string]any)["step"] != "download" {
		t.Fatalf("describe: %v", desc)
	}
	iotdpErr(t, ts, "iot-jobs-data", http.MethodGet, "/things/pump-1/jobs/fw-1?executionNumber=7", "", http.StatusNotFound, "ResourceNotFoundException")

	// fw-2 times out once its step timeout passes.
	iotdpOK(t, ts, "iot-jobs-data", http.MethodPost, "/things/pump-1/jobs/fw-2", `{"status":"IN_PROGRESS","stepTimeoutInMinutes":1}`)
	st := srv.iotJobsData
	st.mu.Lock()
	st.execs[iotjKey("pump-1", "fw-2")].stepDeadline = time.Now().Add(-time.Second)
	st.mu.Unlock()
	if d := iotdpOK(t, ts, "iot-jobs-data", http.MethodGet, "/things/pump-1/jobs/fw-2", "")["execution"].(map[string]any); d["status"] != "TIMED_OUT" {
		t.Fatalf("timeout: %v", d)
	}
	// Nothing pending: StartNext returns no execution.
	if out := iotdpOK(t, ts, "iot-jobs-data", http.MethodPut, "/things/pump-1/jobs/$next", ""); len(out) != 0 {
		t.Fatalf("nothing pending: %v", out)
	}
	// Deleting a job drops its stored executions.
	iotdpOK(t, ts, "iot", http.MethodDelete, "/jobs/fw-1", "")
	iotdpErr(t, ts, "iot-jobs-data", http.MethodGet, "/things/pump-1/jobs/fw-1", "", http.StatusNotFound, "ResourceNotFoundException")
	for _, k := range st.storedKeys() {
		if k == iotjKey("pump-1", "fw-1") {
			t.Fatalf("execution of a deleted job kept: %v", st.storedKeys())
		}
	}
}

func TestIoTJobsDataPlaneStage1Cancel(t *testing.T) {
	_, ts := iotdpServer(t)
	iotjCreateJob(t, ts, "a", `{`+iotjPumpTarget+`}`)
	iotjCreateJob(t, ts, "b", `{`+iotjPumpTarget+`}`)
	iotdpOK(t, ts, "iot-jobs-data", http.MethodPost, "/things/pump-1/jobs/a", `{"status":"IN_PROGRESS"}`)
	iotdpOK(t, ts, "iot", http.MethodPut, "/jobs/a/cancel", `{}`)
	iotdpOK(t, ts, "iot", http.MethodPut, "/jobs/b/cancel", `{}`)
	// Without force, the in-progress execution continues; the queued one is canceled.
	if s := iotdpOK(t, ts, "iot-jobs-data", http.MethodGet, "/things/pump-1/jobs/a", "")["execution"].(map[string]any)["status"]; s != "IN_PROGRESS" {
		t.Fatalf("a after cancel: %v", s)
	}
	if s := iotdpOK(t, ts, "iot-jobs-data", http.MethodGet, "/things/pump-1/jobs/b", "")["execution"].(map[string]any)["status"]; s != "CANCELED" {
		t.Fatalf("b after cancel: %v", s)
	}
	iotdpOK(t, ts, "iot", http.MethodPut, "/jobs/a/cancel?force=true", `{}`)
	if s := iotdpOK(t, ts, "iot-jobs-data", http.MethodGet, "/things/pump-1/jobs/a", "")["execution"].(map[string]any)["status"]; s != "CANCELED" {
		t.Fatalf("a after forced cancel: %v", s)
	}
	iotdpErr(t, ts, "iot-jobs-data", http.MethodPost, "/things/pump-1/jobs/a", `{"status":"SUCCEEDED"}`, http.StatusConflict, "InvalidStateTransitionException")
	// An id in use is refused; once deleted it can be created again, and
	// the new job's execution starts over.
	iotdpErr(t, ts, "iot", http.MethodPut, "/jobs/a", `{`+iotjPumpTarget+`}`, http.StatusConflict, "ResourceAlreadyExistsException")
	if s := iotdpOK(t, ts, "iot-jobs-data", http.MethodGet, "/things/pump-1/jobs/a", "")["execution"].(map[string]any)["status"]; s != "CANCELED" {
		t.Fatalf("refused duplicate changed the job: %v", s)
	}
	iotdpOK(t, ts, "iot", http.MethodDelete, "/jobs/a", "")
	iotjCreateJob(t, ts, "a", `{`+iotjPumpTarget+`}`)
	if s := iotdpOK(t, ts, "iot-jobs-data", http.MethodGet, "/things/pump-1/jobs/a", "")["execution"].(map[string]any)["status"]; s != "QUEUED" {
		t.Fatalf("re-created job: %v", s)
	}
	// The IoT control plane refuses oversized jobs.
	iotdpErr(t, ts, "iot", http.MethodPut, "/jobs/huge", `{`+iotjPumpTarget+`,"document":"`+strings.Repeat("x", iotMaxJobDocument+1)+`"}`, http.StatusBadRequest, "InvalidRequestException")
}

func TestIoTJobsDataPlaneStage1CommandExecutions(t *testing.T) {
	_, ts := iotdpServer(t)
	const cmd = "arn:aws:iot:us-east-1:123456789012:command/blink"
	const target = "arn:aws:iot:us-east-1:123456789012:thing/pump-1"
	iotdpErr(t, ts, "iot-jobs-data", http.MethodPost, "/command-executions", `{"targetArn":"`+target+`","commandArn":"`+cmd+`"}`, http.StatusNotFound, "ResourceNotFoundException")
	iotdpOK(t, ts, "iot", http.MethodPut, "/commands/blink", `{"payload":{"content":"aGk="}}`)
	body := `{"targetArn":"` + target + `","commandArn":"` + cmd + `","parameters":{"n":{"I":3}},"clientToken":"tok-1"}`
	first := iotdpOK(t, ts, "iot-jobs-data", http.MethodPost, "/command-executions", body)
	if again := iotdpOK(t, ts, "iot-jobs-data", http.MethodPost, "/command-executions", body); again["executionId"] != first["executionId"] {
		t.Fatalf("idempotency: %v vs %v", first, again)
	}
	iotdpErr(t, ts, "iot-jobs-data", http.MethodPost, "/command-executions", `{"targetArn":"`+target+`","commandArn":"`+cmd+`","clientToken":"tok-1"}`, http.StatusConflict, "ConflictException")
	iotdpErr(t, ts, "iot-jobs-data", http.MethodPost, "/command-executions", `{"targetArn":"arn:aws:s3:::bucket","commandArn":"`+cmd+`"}`, http.StatusBadRequest, "ValidationException")
	iotdpErr(t, ts, "iot-jobs-data", http.MethodPost, "/command-executions", `{"targetArn":"`+target+`","commandArn":"`+cmd+`","parameters":{"n":{"I":3,"S":"x"}}}`, http.StatusBadRequest, "ValidationException")
	iotdpOK(t, ts, "iot", http.MethodDelete, "/commands/blink", "")
	iotdpErr(t, ts, "iot-jobs-data", http.MethodPost, "/command-executions", `{"targetArn":"`+target+`","commandArn":"`+cmd+`"}`, http.StatusNotFound, "ResourceNotFoundException")
}

// TestIoTJobsDataPlaneStage1ErrorShape checks the REST JSON error: the type
// in X-Amzn-ErrorType and a message naming the missing execution.
func TestIoTJobsDataPlaneStage1ErrorShape(t *testing.T) {
	_, ts := iotdpServer(t)
	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/things/pump-9/jobs/missing-job", nil, nil, "iot-jobs-data")
	raw := mustBody(t, resp)
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil || resp.StatusCode != http.StatusNotFound ||
		resp.Header.Get("X-Amzn-ErrorType") != "ResourceNotFoundException" || resp.Header.Get("Content-Type") != "application/json" ||
		!strings.Contains(out["message"], "missing-job") {
		t.Fatalf("error shape: %d %v %s", resp.StatusCode, resp.Header, raw)
	}
}
