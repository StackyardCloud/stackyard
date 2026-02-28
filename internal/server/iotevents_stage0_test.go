package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func iotEventsRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		method,
		ts.URL+path,
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/json",
		},
		"iotevents",
	)
}

func TestIoTEventsStage0CatalogCoverage(t *testing.T) {
	if len(iotEventsOperations) != 26 {
		t.Fatalf("expected 26 IoT Events operations from docs, got %d", len(iotEventsOperations))
	}
	if len(iotEventsOperationByName) != len(iotEventsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateInput",
		"CreateDetectorModel",
		"CreateAlarmModel",
		"ListInputs",
		"DescribeDetectorModel",
		"PutLoggingOptions",
	}
	for _, action := range requiredActions {
		if _, ok := iotEventsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(iotEventsDataTypes) != 62 {
		t.Fatalf("expected 62 IoT Events data types from docs, got %d", len(iotEventsDataTypes))
	}
	if len(iotEventsDataTypeByName) != len(iotEventsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"InputSummary",
		"DetectorModelSummary",
		"AlarmModelSummary",
		"LoggingOptions",
		"RoutedResource",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := iotEventsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestIoTEventsStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotEventsRequest(t, ts, http.MethodPost, "/unknown-iotevents-route", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestIoTEventsKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := iotEventsRequest(t, ts, http.MethodGet, "/inputs?maxResults=1", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "inputSummaries") {
		t.Fatalf("expected ListInputs response body to include inputSummaries, got %q", body)
	}
}
