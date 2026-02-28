package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func accessAnalyzerRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "access-analyzer")
}

func accessAnalyzerPathForOperation(op accessAnalyzerOperation) string {
	path := strings.TrimSpace(strings.SplitN(op.URI, "?", 2)[0])
	re := regexp.MustCompile(`\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(path, func(match string) string {
		name := strings.Trim(match, "{}")
		switch name {
		case "analyzerName":
			return "stackyard-analyzer"
		case "ruleName":
			return "stackyard-rule"
		case "id":
			return "finding-000001"
		case "jobId":
			return "job-000001"
		case "accessPreviewId":
			return "ap-000001"
		case "resourceArn":
			return url.PathEscape("arn:aws:s3:::stackyard-bucket")
		default:
			return "stackyard"
		}
	})
}

func TestAccessAnalyzerStage0CatalogCoverage(t *testing.T) {
	if len(accessAnalyzerOperations) != 37 {
		t.Fatalf("expected 37 Access Analyzer operations from docs, got %d", len(accessAnalyzerOperations))
	}
	if len(accessAnalyzerOperationByName) != len(accessAnalyzerOperations) {
		t.Fatalf("expected unique Access Analyzer operation names")
	}

	requiredActions := []string{
		"CreateAnalyzer",
		"ListAnalyzers",
		"GetAnalyzer",
		"CreateArchiveRule",
		"ListFindings",
		"ValidatePolicy",
		"StartPolicyGeneration",
		"ListPolicyGenerations",
	}
	for _, action := range requiredActions {
		if _, ok := accessAnalyzerOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(accessAnalyzerDataTypes) != 91 {
		t.Fatalf("expected 91 Access Analyzer data types from docs, got %d", len(accessAnalyzerDataTypes))
	}
	if len(accessAnalyzerDataTypeByName) != len(accessAnalyzerDataTypes) {
		t.Fatalf("expected unique Access Analyzer data type names")
	}
}

func TestAccessAnalyzerStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := accessAnalyzerRequest(t, ts, http.MethodGet, "/definitely-not-accessanalyzer-route", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestAccessAnalyzerStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := accessAnalyzerRequest(t, ts, http.MethodGet, "/analyzer", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "analyzers") {
		t.Fatalf("expected ListAnalyzers response body to include analyzers, got %q", body)
	}
}

func TestAccessAnalyzerStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range accessAnalyzerOperations {
		payload := `{}`
		if strings.EqualFold(op.Method, http.MethodGet) || strings.EqualFold(op.Method, http.MethodDelete) {
			payload = ""
		}
		resp := accessAnalyzerRequest(t, ts, op.Method, accessAnalyzerPathForOperation(op), payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
