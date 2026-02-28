package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func transcribeRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "Transcribe." + action,
		},
		"transcribe",
	)
}

func TestTranscribeStage0CatalogCoverage(t *testing.T) {
	if len(transcribeOperations) != 43 {
		t.Fatalf("expected 43 Transcribe operations from docs, got %d", len(transcribeOperations))
	}
	if len(transcribeOperationByName) != len(transcribeOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"StartTranscriptionJob",
		"GetTranscriptionJob",
		"ListTranscriptionJobs",
		"CreateVocabulary",
		"CreateLanguageModel",
		"StartCallAnalyticsJob",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := transcribeOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(transcribeDataTypes) != 45 {
		t.Fatalf("expected 45 Transcribe data types from docs, got %d", len(transcribeDataTypes))
	}
	if len(transcribeDataTypeByName) != len(transcribeDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"TranscriptionJob",
		"MedicalTranscriptionJob",
		"MedicalScribeJob",
		"CallAnalyticsJob",
		"LanguageModel",
		"VocabularyInfo",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := transcribeDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestTranscribeStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := transcribeRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestTranscribeStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := transcribeRequest(t, ts, "ListTranscriptionJobs", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "TranscriptionJobSummaries") {
		t.Fatalf("expected ListTranscriptionJobs response body to include TranscriptionJobSummaries, got %q", body)
	}
}

func TestTranscribeStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range transcribeOperations {
		resp := transcribeRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
