package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func codeGuruRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
	}
	headers := map[string]string{}
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "codeguru-reviewer")
}

func TestCodeGuruStage0CatalogCoverage(t *testing.T) {
	if len(codeGuruOperations) != 18 {
		t.Fatalf("expected 18 CodeGuru Reviewer operations from docs, got %d", len(codeGuruOperations))
	}
	if len(codeGuruOperationByName) != len(codeGuruOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"AssociateRepository",
		"CreateCodeReview",
		"CreateCodeReviewInternal",
		"CreateConnectionToken",
		"GetMetricsData",
		"ListThirdPartyRepositories",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := codeGuruOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(codeGuruDataTypes) != 30 {
		t.Fatalf("expected 30 CodeGuru Reviewer data types from docs, got %d", len(codeGuruDataTypes))
	}
	if len(codeGuruDataTypeByName) != len(codeGuruDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"CodeReview",
		"RepositoryAssociation",
		"RecommendationSummary",
		"Metrics",
		"AuthorizationToken",
	}
	for _, typeName := range requiredTypes {
		if _, ok := codeGuruDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestCodeGuruStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeGuruRequest(t, ts, http.MethodGet, "/unknown-codeguru-route", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestCodeGuruKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := codeGuruRequest(t, ts, http.MethodGet, "/associations", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "RepositoryAssociationSummaries") {
		t.Fatalf("expected ListRepositoryAssociations response body to include RepositoryAssociationSummaries, got %q", body)
	}
}

func TestCodeGuruExtendedOperationsReturnSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	tests := []struct {
		method        string
		path          string
		payload       string
		expectedField string
	}{
		{method: http.MethodPost, path: "/createCodeReviewInternal", payload: `{}`, expectedField: "CodeReview"},
		{method: http.MethodPost, path: "/token", payload: `{}`, expectedField: "ConnectionToken"},
		{method: http.MethodGet, path: "/metrics", payload: "", expectedField: "MetricQueryResults"},
		{method: http.MethodGet, path: "/thirdPartyRepositories", payload: "", expectedField: "RepositorySummaries"},
	}

	for _, tc := range tests {
		resp := codeGuruRequest(t, ts, tc.method, tc.path, tc.payload)
		assertStatus(t, resp, http.StatusOK)
		body := string(mustBody(t, resp))
		if strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s %s returned NotImplementedException: %q", tc.method, tc.path, body)
		}
		if !strings.Contains(body, tc.expectedField) {
			t.Fatalf("%s %s expected response to include %q, got %q", tc.method, tc.path, tc.expectedField, body)
		}
	}
}
