package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func sagemakerRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "SageMaker." + action,
		},
		"sagemaker",
	)
}

func TestSageMakerStage0CatalogCoverage(t *testing.T) {
	if len(sagemakerOperations) != 379 {
		t.Fatalf("expected 379 SageMaker operations from docs, got %d", len(sagemakerOperations))
	}
	if len(sagemakerOperationByName) != len(sagemakerOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateTrainingJob",
		"DescribeTrainingJob",
		"ListTrainingJobs",
		"CreateModel",
		"CreateEndpoint",
		"CreateNotebookInstance",
		"AddTags",
	}
	for _, action := range requiredActions {
		if _, ok := sagemakerOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(sagemakerDataTypes) != 728 {
		t.Fatalf("expected 728 SageMaker data types from docs, got %d", len(sagemakerDataTypes))
	}
	if len(sagemakerDataTypeByName) != len(sagemakerDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"AlgorithmSpecification",
		"TrainingJobSummary",
		"Model",
		"Endpoint",
		"NotebookInstanceSummary",
		"Tag",
	}
	for _, typeName := range requiredTypes {
		if _, ok := sagemakerDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestSageMakerStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := sagemakerRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestSageMakerStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := sagemakerRequest(t, ts, "ListTrainingJobs", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "TrainingJobSummaries") {
		t.Fatalf("expected ListTrainingJobs response body to include TrainingJobSummaries, got %q", body)
	}
}

func TestSageMakerStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range sagemakerOperations {
		resp := sagemakerRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
