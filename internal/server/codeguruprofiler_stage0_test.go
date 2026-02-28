package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func codeGuruProfilerRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
	}
	headers := map[string]string{}
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "codeguru-profiler")
}

func TestCodeGuruProfilerStage0CatalogCoverage(t *testing.T) {
	if len(codeGuruProfilerOperations) != 23 {
		t.Fatalf("expected 23 CodeGuru Profiler operations from docs, got %d", len(codeGuruProfilerOperations))
	}
	if len(codeGuruProfilerOperationByName) != len(codeGuruProfilerOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"AddNotificationChannels",
		"BatchGetFrameMetricData",
		"GetProfile",
		"GetRecommendations",
		"PutPermission",
		"SubmitFeedback",
	}
	for _, action := range requiredActions {
		if _, ok := codeGuruProfilerOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(codeGuruProfilerDataTypes) != 19 {
		t.Fatalf("expected 19 CodeGuru Profiler data types from docs, got %d", len(codeGuruProfilerDataTypes))
	}
	if len(codeGuruProfilerDataTypeByName) != len(codeGuruProfilerDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"AgentConfiguration",
		"FindingsReportSummary",
		"ProfilingGroupDescription",
		"Recommendation",
		"UserFeedback",
	}
	for _, typeName := range requiredTypes {
		if _, ok := codeGuruProfilerDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestCodeGuruProfilerStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeGuruProfilerRequest(t, ts, http.MethodGet, "/unknown-codeguruprofiler-route", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCodeGuruProfilerKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeGuruProfilerRequest(t, ts, http.MethodGet, "/profilingGroups", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "profilingGroupNames") {
		t.Fatalf("expected response body to include profilingGroupNames, got %q", body)
	}
}
