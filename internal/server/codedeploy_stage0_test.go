package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func codeDeployRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "CodeDeploy_20141006." + action,
		},
		"codedeploy",
	)
}

func TestCodeDeployStage0CatalogCoverage(t *testing.T) {
	if len(codeDeployOperations) != 47 {
		t.Fatalf("expected 47 CodeDeploy operations from docs, got %d", len(codeDeployOperations))
	}
	if len(codeDeployOperationByName) != len(codeDeployOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateApplication",
		"CreateDeployment",
		"CreateDeploymentGroup",
		"ListDeployments",
		"GetDeployment",
		"UpdateDeploymentGroup",
	}
	for _, action := range requiredActions {
		if _, ok := codeDeployOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(codeDeployDataTypes) != 57 {
		t.Fatalf("expected 57 CodeDeploy data types from docs, got %d", len(codeDeployDataTypes))
	}
	if len(codeDeployDataTypeByName) != len(codeDeployDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"ApplicationInfo",
		"DeploymentInfo",
		"DeploymentGroupInfo",
		"DeploymentConfigInfo",
		"RevisionLocation",
		"TargetInstances",
	}
	for _, typeName := range requiredTypes {
		if _, ok := codeDeployDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestCodeDeployStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeDeployRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCodeDeployStage0KnownActionDoesNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeDeployRequest(t, ts, "ListApplications", `{"nextToken":"","sortBy":"name"}`)
	if resp.StatusCode == http.StatusNotImplemented {
		t.Fatalf("expected ListApplications to be implemented")
	}
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("expected non-NotImplemented response body, got %q", body)
	}
}

func TestCodeDeployStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range codeDeployOperations {
		resp := codeDeployRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
