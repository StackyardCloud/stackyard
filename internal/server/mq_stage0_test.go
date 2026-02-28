package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func mqRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "mq")
}

func TestMQStage0CatalogCoverage(t *testing.T) {
	if len(mqOperations) != 24 {
		t.Fatalf("expected 24 MQ operations from docs, got %d", len(mqOperations))
	}
	if len(mqOperationByName) != len(mqOperations) {
		t.Fatalf("expected unique MQ operation names")
	}

	requiredActions := []string{
		"CreateBroker",
		"DeleteBroker",
		"DescribeBroker",
		"ListBrokers",
		"CreateConfiguration",
		"ListConfigurations",
		"CreateUser",
		"ListUsers",
		"CreateTags",
		"ListTags",
	}
	for _, action := range requiredActions {
		if _, ok := mqOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(mqDataTypes) != 60 {
		t.Fatalf("expected 60 MQ data types from docs, got %d", len(mqDataTypes))
	}
	if len(mqDataTypeByName) != len(mqDataTypes) {
		t.Fatalf("expected unique MQ data type names")
	}

	requiredTypes := []string{
		"CreateBrokerInput",
		"DescribeBrokerOutput",
		"CreateConfigurationOutput",
		"ListConfigurationsOutput",
		"ListUsersOutput",
		"Tags",
	}
	for _, typeName := range requiredTypes {
		if _, ok := mqDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestMQStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mqRequest(t, ts, http.MethodPost, "/v1/mq-unknown-action", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestMQStage0KnownActionReturnsListBrokers(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mqRequest(t, ts, http.MethodGet, "/v1/brokers", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "BrokerSummaries") {
		t.Fatalf("expected ListBrokers response body to include BrokerSummaries, got %q", body)
	}
}

func TestMQStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"broker-id":              "b-000001",
		"configuration-id":       "c-000001",
		"configuration-revision": "1",
		"username":               "admin",
		"resource-arn":           "arn:aws:mq:us-east-1:123456789012:broker:stackyard-broker:b-000001",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range mqOperations {
		path := placeholder.ReplaceAllStringFunc(op.URI, func(token string) string {
			name := strings.Trim(token, "{}")
			value := replacements[name]
			if value == "" {
				value = "stackyard"
			}
			return url.PathEscape(value)
		})

		var body []byte
		if op.Method == http.MethodPost || op.Method == http.MethodPut || op.Method == http.MethodPatch {
			body = []byte(`{}`)
		}

		resp := mqRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
