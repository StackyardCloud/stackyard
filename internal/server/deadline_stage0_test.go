package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func deadlineRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "deadline")
}

func deadlinePathForOperation(op deadlineOperation) string {
	path := op.URI
	replacements := map[string]string{
		"{farmId}":             "farm-00000001",
		"{fleetId}":            "fleet-00000001",
		"{queueId}":            "queue-00000001",
		"{jobId}":              "job-00000001",
		"{workerId}":           "worker-00000001",
		"{stepId}":             "step-00000001",
		"{taskId}":             "task-00000001",
		"{sessionId}":          "session-00000001",
		"{sessionActionId}":    "session-action-00000001",
		"{principalId}":        "AIDAEXAMPLE",
		"{monitorId}":          "monitor-00000001",
		"{budgetId}":           "budget-00000001",
		"{storageProfileId}":   "sp-00000001",
		"{licenseEndpointId}":  "lic-00000001",
		"{productId}":          "product-00000001",
		"{queueEnvironmentId}": "qenv-00000001",
		"{limitId}":            "limit-00000001",
		"{resourceArn}":        url.PathEscape("arn:aws:deadline:us-east-1:123456789012:farm/farm-00000001"),
	}
	for key, value := range replacements {
		path = strings.ReplaceAll(path, key, value)
	}
	return path
}

func TestDeadlineStage0CatalogCoverage(t *testing.T) {
	if len(deadlineOperations) != 113 {
		t.Fatalf("expected 113 Deadline operations from docs, got %d", len(deadlineOperations))
	}
	if len(deadlineOperationByName) != len(deadlineOperations) {
		t.Fatalf("expected unique Deadline operation names")
	}

	requiredActions := []string{
		"CreateFarm",
		"CreateFleet",
		"CreateQueue",
		"CreateJob",
		"GetFarm",
		"ListFarms",
		"SearchWorkers",
		"TagResource",
		"UntagResource",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := deadlineOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(deadlineDataTypes) != 132 {
		t.Fatalf("expected 132 Deadline data types from docs, got %d", len(deadlineDataTypes))
	}
	if len(deadlineDataTypeByName) != len(deadlineDataTypes) {
		t.Fatalf("expected unique Deadline data type names")
	}
}

func TestDeadlineStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := deadlineRequest(t, ts, http.MethodGet, "/2023-10-12/not-a-real-deadline-route", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestDeadlineStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := deadlineRequest(t, ts, http.MethodGet, "/2023-10-12/farms", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "farms") {
		t.Fatalf("expected ListFarms response body to include farms, got %q", body)
	}
}

func TestDeadlineStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range deadlineOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := deadlineRequest(t, ts, op.Method, deadlinePathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
