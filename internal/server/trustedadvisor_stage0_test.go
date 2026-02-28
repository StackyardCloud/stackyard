package server

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func trustedAdvisorRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		method,
		ts.URL+path,
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/json",
		},
		"trustedadvisor",
	)
}

func trustedAdvisorConcretePath(path string) string {
	re := regexp.MustCompile(`\{[^}]+\}`)
	return re.ReplaceAllString(path, "stackyard-id")
}

func TestTrustedAdvisorStage0CatalogCoverage(t *testing.T) {
	if len(trustedAdvisorOperations) != 11 {
		t.Fatalf("expected 11 Trusted Advisor operations from docs, got %d", len(trustedAdvisorOperations))
	}
	if len(trustedAdvisorOperationByName) != len(trustedAdvisorOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"ListChecks",
		"ListRecommendations",
		"GetRecommendation",
		"UpdateRecommendationLifecycle",
		"BatchUpdateRecommendationResourceExclusion",
	}
	for _, action := range requiredActions {
		if _, ok := trustedAdvisorOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(trustedAdvisorDataTypes) != 13 {
		t.Fatalf("expected 13 Trusted Advisor data types from docs, got %d", len(trustedAdvisorDataTypes))
	}
	if len(trustedAdvisorDataTypeByName) != len(trustedAdvisorDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"CheckSummary",
		"Recommendation",
		"OrganizationRecommendation",
	}
	for _, typeName := range requiredTypes {
		if _, ok := trustedAdvisorDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestTrustedAdvisorStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := trustedAdvisorRequest(t, ts, http.MethodGet, "/v1/totally-unknown-action", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestTrustedAdvisorKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := trustedAdvisorRequest(t, ts, http.MethodGet, "/v1/checks", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
	if !strings.Contains(body, "checkSummaries") {
		t.Fatalf("expected ListChecks response body to include checkSummaries, got %q", body)
	}
}

func TestTrustedAdvisorStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range trustedAdvisorOperations {
		payload := ""
		if op.Method == http.MethodPut {
			payload = `{}`
		}
		resp := trustedAdvisorRequest(t, ts, op.Method, trustedAdvisorConcretePath(op.URI), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
