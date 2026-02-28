package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func dataPipelineRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "DataPipeline." + action,
		},
		"datapipeline",
	)
}

func TestDataPipelineStage0CatalogCoverage(t *testing.T) {
	if len(dataPipelineOperations) != 19 {
		t.Fatalf("expected 19 Data Pipeline operations from docs, got %d", len(dataPipelineOperations))
	}
	if len(dataPipelineOperationByName) != len(dataPipelineOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreatePipeline",
		"PutPipelineDefinition",
		"ValidatePipelineDefinition",
		"PollForTask",
		"SetTaskStatus",
		"ListPipelines",
		"DeletePipeline",
	}
	for _, action := range requiredActions {
		if _, ok := dataPipelineOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(dataPipelineDataTypes) != 15 {
		t.Fatalf("expected 15 Data Pipeline data types from docs, got %d", len(dataPipelineDataTypes))
	}
	if len(dataPipelineDataTypeByName) != len(dataPipelineDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"PipelineObject",
		"PipelineDescription",
		"TaskObject",
		"Tag",
		"ValidationError",
		"ValidationWarning",
	}
	for _, typeName := range requiredTypes {
		if _, ok := dataPipelineDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestDataPipelineStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dataPipelineRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestDataPipelineStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dataPipelineRequest(t, ts, "ListPipelines", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "pipelineIdList") {
		t.Fatalf("expected ListPipelines response body to include pipelineIdList, got %q", body)
	}
}

func TestDataPipelineStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range dataPipelineOperations {
		resp := dataPipelineRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
