package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func braketRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "braket")
}

func TestBraketStage0CatalogCoverage(t *testing.T) {
	if len(braketOperations) != 17 {
		t.Fatalf("expected 17 Braket operations from docs, got %d", len(braketOperations))
	}
	if len(braketOperationByName) != len(braketOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateJob",
		"GetJob",
		"CancelJob",
		"CreateQuantumTask",
		"SearchDevices",
		"TagResource",
		"UpdateSpendingLimit",
	}
	for _, action := range requiredActions {
		if _, ok := braketOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(braketTypes) != 29 {
		t.Fatalf("expected 29 Braket data types from docs, got %d", len(braketTypes))
	}
	if len(braketTypeByName) != len(braketTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"AlgorithmSpecification",
		"DeviceSummary",
		"JobSummary",
		"QuantumTaskSummary",
		"SpendingLimitSummary",
	}
	for _, typeName := range requiredTypes {
		if _, ok := braketTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestBraketStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := braketRequest(t, ts, http.MethodPost, "/braket-unknown-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestBraketStage0KnownActionReturnsSearchDevices(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := braketRequest(t, ts, http.MethodPost, "/devices", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "devices") {
		t.Fatalf("expected SearchDevices response body to include devices, got %q", body)
	}
}

func TestBraketStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"jobArn":                   "arn:aws:braket:us-east-1:123456789012:job/job-000001",
		"quantumTaskArn":           "arn:aws:braket:us-east-1:123456789012:quantum-task/task-000001",
		"spendingLimitArn":         "arn:aws:braket:us-east-1:123456789012:spending-limit/limit-000001",
		"deviceArn":                "arn:aws:braket:us-east-1::device/qpu/test-device",
		"resourceArn":              "arn:aws:braket:us-east-1:123456789012:job/job-000001",
		"tagKeys":                  "env",
		"additionalAttributeNames": "queueInfo",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range braketOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			return url.QueryEscape(value)
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := braketRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
