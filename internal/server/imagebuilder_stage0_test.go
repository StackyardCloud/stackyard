package server

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func imageBuilderRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "imagebuilder")
}

func TestImageBuilderStage0CatalogCoverage(t *testing.T) {
	if len(imageBuilderOperations) != 77 {
		t.Fatalf("expected 77 Image Builder actions from docs, got %d", len(imageBuilderOperations))
	}
	if len(imageBuilderOperationByName) != len(imageBuilderOperations) {
		t.Fatalf("expected unique Image Builder action names")
	}

	requiredActions := []string{
		"CreateImagePipeline",
		"StartImagePipelineExecution",
		"ListImages",
		"GetImage",
		"CreateComponent",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := imageBuilderOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(imageBuilderDataTypes) != 103 {
		t.Fatalf("expected 103 Image Builder data types from docs, got %d", len(imageBuilderDataTypes))
	}
	if len(imageBuilderDataTypeByName) != len(imageBuilderDataTypes) {
		t.Fatalf("expected unique Image Builder data type names")
	}

	requiredTypes := []string{
		"Image",
		"ImagePipeline",
		"ImageRecipe",
		"InfrastructureConfiguration",
		"LifecyclePolicy",
		"Workflow",
		"Component",
	}
	for _, typeName := range requiredTypes {
		if _, ok := imageBuilderDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestImageBuilderStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := imageBuilderRequest(t, ts, http.MethodPost, "/DefinitelyUnknownImageBuilderAction", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestImageBuilderStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := imageBuilderRequest(t, ts, http.MethodPost, "/ListImages", []byte(`{}`))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "imageVersionList") {
		t.Fatalf("expected ListImages response body to include imageVersionList, got %q", body)
	}
}

func TestImageBuilderStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	placeholder := regexp.MustCompile(`\{([^}]+)\}`)
	for _, op := range imageBuilderOperations {
		path := placeholder.ReplaceAllString(op.URI, "stackyard")

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := imageBuilderRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
