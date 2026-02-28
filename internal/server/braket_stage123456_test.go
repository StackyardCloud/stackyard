package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBraketStage1DeviceReadAndSearch(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := braketRequest(t, ts, http.MethodPost, "/devices", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	searchPayload := decodeBraketPayload(t, resp)
	devices, ok := searchPayload["devices"].([]any)
	if !ok || len(devices) == 0 {
		t.Fatalf("expected SearchDevices to return devices")
	}

	deviceArn := url.PathEscape("arn:aws:braket:us-east-1::device/qpu/test-device")
	resp = braketRequest(t, ts, http.MethodGet, "/device/"+deviceArn, nil)
	assertStatus(t, resp, http.StatusOK)
	getPayload := decodeBraketPayload(t, resp)
	if got := braketPayloadString(getPayload, "deviceArn"); got == "" {
		t.Fatalf("expected GetDevice response to include deviceArn")
	}
}

func TestBraketStage2JobLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := braketRequest(t, ts, http.MethodPost, "/job", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeBraketPayload(t, resp)
	jobArn := braketPayloadString(createPayload, "jobArn")
	if jobArn == "" {
		t.Fatalf("expected CreateJob to return jobArn")
	}

	resp = braketRequest(t, ts, http.MethodGet, "/job/"+url.PathEscape(jobArn)+"?additionalAttributeNames=queueInfo", nil)
	assertStatus(t, resp, http.StatusOK)
	getPayload := decodeBraketPayload(t, resp)
	if got := braketPayloadString(getPayload, "status"); got == "" {
		t.Fatalf("expected GetJob response to include status")
	}

	resp = braketRequest(t, ts, http.MethodPost, "/jobs", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	searchPayload := decodeBraketPayload(t, resp)
	jobs, ok := searchPayload["jobs"].([]any)
	if !ok || len(jobs) == 0 {
		t.Fatalf("expected SearchJobs to return jobs")
	}

	resp = braketRequest(t, ts, http.MethodPut, "/job/"+url.PathEscape(jobArn)+"/cancel", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	cancelPayload := decodeBraketPayload(t, resp)
	if got := braketPayloadString(cancelPayload, "cancellationStatus"); got != "CANCELLING" {
		t.Fatalf("expected CancelJob cancellationStatus CANCELLING, got %q", got)
	}
}

func TestBraketStage3QuantumTaskLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := braketRequest(t, ts, http.MethodPost, "/quantum-task", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeBraketPayload(t, resp)
	taskArn := braketPayloadString(createPayload, "quantumTaskArn")
	if taskArn == "" {
		t.Fatalf("expected CreateQuantumTask to return quantumTaskArn")
	}

	resp = braketRequest(t, ts, http.MethodGet, "/quantum-task/"+url.PathEscape(taskArn)+"?additionalAttributeNames=queueInfo", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = braketRequest(t, ts, http.MethodPost, "/quantum-tasks", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	searchPayload := decodeBraketPayload(t, resp)
	tasks, ok := searchPayload["quantumTasks"].([]any)
	if !ok || len(tasks) == 0 {
		t.Fatalf("expected SearchQuantumTasks to return quantumTasks")
	}

	resp = braketRequest(t, ts, http.MethodPut, "/quantum-task/"+url.PathEscape(taskArn)+"/cancel", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	cancelPayload := decodeBraketPayload(t, resp)
	if got := braketPayloadString(cancelPayload, "cancellationStatus"); got != "CANCELLING" {
		t.Fatalf("expected CancelQuantumTask cancellationStatus CANCELLING, got %q", got)
	}
}

func TestBraketStage4SpendingLimitLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := braketRequest(t, ts, http.MethodPost, "/spending-limit", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	createPayload := decodeBraketPayload(t, resp)
	limitArn := braketPayloadString(createPayload, "spendingLimitArn")
	if limitArn == "" {
		t.Fatalf("expected CreateSpendingLimit to return spendingLimitArn")
	}

	resp = braketRequest(t, ts, http.MethodPatch, "/spending-limit/"+url.PathEscape(limitArn)+"/update", []byte(`{"amount":500}`))
	assertStatus(t, resp, http.StatusOK)
	updatePayload := decodeBraketPayload(t, resp)
	if got := braketPayloadString(updatePayload, "spendingLimitArn"); got == "" {
		t.Fatalf("expected UpdateSpendingLimit response to include spendingLimitArn")
	}

	resp = braketRequest(t, ts, http.MethodPost, "/spending-limits", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	searchPayload := decodeBraketPayload(t, resp)
	limits, ok := searchPayload["spendingLimits"].([]any)
	if !ok || len(limits) == 0 {
		t.Fatalf("expected SearchSpendingLimits response to include spendingLimits")
	}

	resp = braketRequest(t, ts, http.MethodDelete, "/spending-limit/"+url.PathEscape(limitArn)+"/delete", nil)
	assertStatus(t, resp, http.StatusOK)
}

func TestBraketStage5TaggingLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resourceArn := url.PathEscape("arn:aws:braket:us-east-1:123456789012:job/job-000001")

	resp := braketRequest(t, ts, http.MethodPost, "/tags/"+resourceArn, []byte(`{"tags":{"team":"platform","env":"dev"}}`))
	assertStatus(t, resp, http.StatusOK)

	resp = braketRequest(t, ts, http.MethodGet, "/tags/"+resourceArn, nil)
	assertStatus(t, resp, http.StatusOK)
	tagPayload := decodeBraketPayload(t, resp)
	tags, ok := tagPayload["tags"].(map[string]any)
	if !ok || len(tags) == 0 {
		t.Fatalf("expected ListTagsForResource to return tags")
	}
	if got, ok := tags["team"].(string); !ok || strings.TrimSpace(got) != "platform" {
		t.Fatalf("expected ListTagsForResource to include team=platform")
	}

	resp = braketRequest(t, ts, http.MethodDelete, "/tags/"+resourceArn+"?tagKeys=team", nil)
	assertStatus(t, resp, http.StatusOK)

	resp = braketRequest(t, ts, http.MethodGet, "/tags/"+resourceArn, nil)
	assertStatus(t, resp, http.StatusOK)
	afterPayload := decodeBraketPayload(t, resp)
	afterTags, ok := afterPayload["tags"].(map[string]any)
	if !ok {
		t.Fatalf("expected ListTagsForResource response to include tags map")
	}
	if _, exists := afterTags["team"]; exists {
		t.Fatalf("expected team tag to be removed")
	}
}

func TestBraketStage6ValidationAndIdempotency(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := braketRequest(t, ts, http.MethodPost, "/unknown-braket-route", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException for unknown route, got %q", body)
	}

	resp = signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/job",
		[]byte(`{"broken":`),
		map[string]string{"Content-Type": "application/json"},
		"braket",
	)
	assertStatus(t, resp, http.StatusBadRequest)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "invalid JSON body") {
		t.Fatalf("expected invalid JSON body error, got %q", body)
	}

	limitArn := url.PathEscape("arn:aws:braket:us-east-1:123456789012:spending-limit/limit-000001")
	resp = braketRequest(t, ts, http.MethodDelete, "/spending-limit/"+limitArn+"/delete", nil)
	assertStatus(t, resp, http.StatusOK)
	resp = braketRequest(t, ts, http.MethodDelete, "/spending-limit/"+limitArn+"/delete", nil)
	assertStatus(t, resp, http.StatusOK)
}

func decodeBraketPayload(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	payload := map[string]any{}
	if err := json.Unmarshal(mustBody(t, resp), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func braketPayloadString(payload map[string]any, key string) string {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
