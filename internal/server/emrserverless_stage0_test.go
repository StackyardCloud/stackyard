package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func emrServerlessRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "emr-serverless")
}

func TestEMRServerlessStage0CatalogCoverage(t *testing.T) {
	if len(emrServerlessOperations) != 16 {
		t.Fatalf("expected 16 EMR Serverless operations from docs, got %d", len(emrServerlessOperations))
	}
	if len(emrServerlessOperationByName) != len(emrServerlessOperations) {
		t.Fatalf("expected unique EMR Serverless operation names")
	}

	requiredActions := []string{
		"CreateApplication",
		"GetApplication",
		"ListApplications",
		"StartApplication",
		"StartJobRun",
		"GetJobRun",
		"ListJobRuns",
		"ListTagsForResource",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := emrServerlessOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(emrServerlessDataTypes) != 36 {
		t.Fatalf("expected 36 EMR Serverless data types from docs, got %d", len(emrServerlessDataTypes))
	}
	if len(emrServerlessDataTypeByName) != len(emrServerlessDataTypes) {
		t.Fatalf("expected unique EMR Serverless data type names")
	}

	requiredTypes := []string{
		"Application",
		"ApplicationSummary",
		"JobRun",
		"JobRunSummary",
		"SparkSubmit",
	}
	for _, typeName := range requiredTypes {
		if _, ok := emrServerlessDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestEMRServerlessStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := emrServerlessRequest(t, ts, http.MethodPost, "/emr-serverless-unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestEMRServerlessStage0KnownActionReturnsListApplications(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := emrServerlessRequest(t, ts, http.MethodGet, "/applications?maxResults=10", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "applications") {
		t.Fatalf("expected ListApplications response body to include applications, got %q", body)
	}
}

func TestEMRServerlessStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"applicationId":                "app-000001",
		"jobRunId":                     "jobrun-000001",
		"resourceArn":                  url.PathEscape("arn:aws:emr-serverless:us-east-1:123456789012:/applications/app-000001"),
		"tagKeys":                      "env",
		"maxResults":                   "10",
		"nextToken":                    "token-000001",
		"states":                       "CREATED",
		"createdAtAfter":               "0",
		"createdAtBefore":              "9999999999",
		"mode":                         "BATCH",
		"accessSystemProfileLogs":      "false",
		"attempt":                      "1",
		"shutdownGracePeriodInSeconds": "60",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range emrServerlessOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			return value
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPatch || op.Method == http.MethodPut {
			body = []byte(`{}`)
		}

		resp := emrServerlessRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
