package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func codeBuildRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "CodeBuild_20161006." + action,
		},
		"codebuild",
	)
}

func TestCodeBuildStage0CatalogCoverage(t *testing.T) {
	if len(codeBuildOperations) != 59 {
		t.Fatalf("expected 59 CodeBuild operations from docs, got %d", len(codeBuildOperations))
	}
	if len(codeBuildOperationByName) != len(codeBuildOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateProject",
		"StartBuild",
		"BatchGetBuilds",
		"ListProjects",
		"UpdateProject",
		"DeleteProject",
	}
	for _, action := range requiredActions {
		if _, ok := codeBuildOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(codeBuildDataTypes) != 73 {
		t.Fatalf("expected 73 CodeBuild data types from docs, got %d", len(codeBuildDataTypes))
	}
	if len(codeBuildDataTypeByName) != len(codeBuildDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Build",
		"Project",
		"ReportGroup",
		"Fleet",
		"Sandbox",
		"Webhook",
	}
	for _, typeName := range requiredTypes {
		if _, ok := codeBuildDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestCodeBuildStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeBuildRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCodeBuildKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeBuildRequest(t, ts, "ListProjects", `{"nextToken":""}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "projects") {
		t.Fatalf("expected ListProjects response body to include projects, got %q", body)
	}
}
