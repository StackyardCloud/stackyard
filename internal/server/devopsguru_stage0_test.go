package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func devOpsGuruRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "DevOpsGuru_20200331." + action,
		},
		"devops-guru",
	)
}

func TestDevOpsGuruStage0CatalogCoverage(t *testing.T) {
	if len(devOpsGuruOperations) != 31 {
		t.Fatalf("expected 31 DevOps Guru operations from docs, got %d", len(devOpsGuruOperations))
	}
	if len(devOpsGuruOperationByName) != len(devOpsGuruOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"DescribeAccountHealth",
		"DescribeInsight",
		"ListInsights",
		"SearchInsights",
		"AddNotificationChannel",
		"UpdateServiceIntegration",
	}
	for _, action := range requiredActions {
		if _, ok := devOpsGuruOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(devOpsGuruDataTypes) != 93 {
		t.Fatalf("expected 93 DevOps Guru data types from docs, got %d", len(devOpsGuruDataTypes))
	}
	if len(devOpsGuruDataTypeByName) != len(devOpsGuruDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"AccountHealth",
		"AnomalyResource",
		"Event",
		"Recommendation",
		"ServiceIntegrationConfig",
		"ValidationExceptionField",
	}
	for _, typeName := range requiredTypes {
		if _, ok := devOpsGuruDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestDevOpsGuruStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := devOpsGuruRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestDevOpsGuruKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := devOpsGuruRequest(t, ts, "ListInsights", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "ProactiveInsights") {
		t.Fatalf("expected ListInsights response body to include ProactiveInsights, got %q", body)
	}
}
