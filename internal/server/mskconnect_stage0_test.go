package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
)

func mskConnectRequest(t *testing.T, ts *httptest.Server, method, path string, body []byte) *http.Response {
	t.Helper()
	headers := map[string]string{}
	if len(body) > 0 {
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "kafkaconnect")
}

func TestMSKConnectStage0CatalogCoverage(t *testing.T) {
	if len(mskConnectOperations) != 18 {
		t.Fatalf("expected 18 MSK Connect actions from docs, got %d", len(mskConnectOperations))
	}
	if len(mskConnectOperationByName) != len(mskConnectOperations) {
		t.Fatalf("expected unique MSK Connect action names")
	}

	requiredActions := []string{
		"CreateConnector",
		"DeleteConnector",
		"DescribeConnector",
		"ListConnectors",
		"UpdateConnector",
		"CreateCustomPlugin",
		"ListCustomPlugins",
		"CreateWorkerConfiguration",
		"ListWorkerConfigurations",
		"ListConnectorOperations",
		"DescribeConnectorOperation",
		"TagResource",
		"ListTagsForResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := mskConnectOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(mskConnectDataTypes) != 56 {
		t.Fatalf("expected 56 MSK Connect data types from docs, got %d", len(mskConnectDataTypes))
	}
	if len(mskConnectDataTypeByName) != len(mskConnectDataTypes) {
		t.Fatalf("expected unique MSK Connect data type names")
	}

	requiredTypes := []string{
		"Capacity",
		"ConnectorSummary",
		"ConnectorOperationSummary",
		"CustomPluginSummary",
		"WorkerConfigurationSummary",
		"KafkaCluster",
		"KafkaClusterClientAuthentication",
		"WorkerLogDelivery",
	}
	for _, typeName := range requiredTypes {
		if _, ok := mskConnectDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestMSKConnectStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mskConnectRequest(t, ts, http.MethodPost, "/v1/mskconnect-unknown", []byte(`{}`))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestMSKConnectStage0KnownActionReturnsListConnectors(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := mskConnectRequest(t, ts, http.MethodGet, "/v1/connectors?maxResults=20", nil)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "connectors") {
		t.Fatalf("expected ListConnectors response body to include connectors, got %q", body)
	}
}

func TestMSKConnectStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacements := map[string]string{
		"connectorArn":           mskConnectSeedConnectorARN,
		"connectorOperationArn":  mskConnectSeedConnectorOperationARN,
		"customPluginArn":        mskConnectSeedCustomPluginARN,
		"workerConfigurationArn": mskConnectSeedWorkerConfigARN,
		"resourceArn":            mskConnectSeedConnectorARN,
		"currentVersion":         "1",
		"maxResults":             "20",
		"nextToken":              "token-000001",
		"connectorNamePrefix":    "stackyard",
		"namePrefix":             "stackyard",
		"tagKeys":                "env",
	}
	placeholder := regexp.MustCompile(`\{([^}]+)\}`)

	for _, op := range mskConnectOperations {
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

		resp := mskConnectRequest(t, ts, op.Method, path, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
