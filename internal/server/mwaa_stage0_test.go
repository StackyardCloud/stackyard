package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func mwaaRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "airflow")
}

func TestMWAAStage0CatalogCoverage(t *testing.T) {
	if len(mwaaOperations) != 12 {
		t.Fatalf("expected 12 MWAA operations from docs, got %d", len(mwaaOperations))
	}
	if len(mwaaOperationByName) != len(mwaaOperations) {
		t.Fatalf("expected unique MWAA operation names")
	}

	requiredActions := []string{
		"CreateCliToken",
		"CreateEnvironment",
		"CreateWebLoginToken",
		"DeleteEnvironment",
		"GetEnvironment",
		"InvokeRestApi",
		"ListEnvironments",
		"ListTagsForResource",
		"PublishMetrics",
		"TagResource",
		"UntagResource",
		"UpdateEnvironment",
	}
	for _, action := range requiredActions {
		if _, ok := mwaaOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(mwaaDataTypes) != 13 {
		t.Fatalf("expected 13 MWAA data types from docs, got %d", len(mwaaDataTypes))
	}
	if len(mwaaDataTypeByName) != len(mwaaDataTypes) {
		t.Fatalf("expected unique MWAA data type names")
	}

	requiredTypes := []string{
		"Dimension",
		"Environment",
		"LastUpdate",
		"MetricDatum",
		"NetworkConfiguration",
		"StatisticSet",
		"UpdateEnvironment",
	}
	for _, typeName := range requiredTypes {
		if _, ok := mwaaDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestMWAAStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mwaaRequest(t, ts, http.MethodGet, "/mwaa-unknown", nil)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestMWAAStage0KnownActionReturnsListEnvironments(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mwaaRequest(t, ts, http.MethodGet, "/environments?MaxResults=10&NextToken=token-000001", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Environments") {
		t.Fatalf("expected ListEnvironments response body to include Environments, got %q", body)
	}
}

func TestMWAAStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"Name":            "stackyard-environment",
		"EnvironmentName": "stackyard-environment",
		"ResourceArn":     url.PathEscape("arn:aws:airflow:us-east-1:123456789012:environment/stackyard-environment"),
		"MaxResults":      "10",
		"NextToken":       "token-000001",
		"tagKeys":         "env",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range mwaaOperations {
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

		resp := mwaaRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
