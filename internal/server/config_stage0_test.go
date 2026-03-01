package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func configRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "StarlingDoveService." + action,
		},
		"config",
	)
}

func TestConfigStage0CatalogCoverage(t *testing.T) {
	if len(configOperations) != 97 {
		t.Fatalf("expected 97 Config actions from docs, got %d", len(configOperations))
	}
	if len(configOperationByName) != len(configOperations) {
		t.Fatalf("expected unique action names")
	}

	requiredActions := []string{
		"PutConfigurationRecorder",
		"DescribeConfigurationRecorders",
		"PutConfigRule",
		"StartConfigurationRecorder",
		"ListStoredQueries",
		"SelectResourceConfig",
	}
	for _, action := range requiredActions {
		if _, ok := configOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(configDataTypes) != 112 {
		t.Fatalf("expected 112 Config data types from docs, got %d", len(configDataTypes))
	}
	if len(configDataTypeByName) != len(configDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"ConfigurationRecorder",
		"DeliveryChannel",
		"ConfigRule",
		"ConformancePackDetail",
		"ResourceEvaluation",
		"StoredQuery",
	}
	for _, typeName := range requiredTypes {
		if _, ok := configDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestConfigStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := configRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestConfigKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := configRequest(t, ts, "DescribeConfigurationRecorders", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplementedException") {
		t.Fatalf("did not expect NotImplementedException response body, got %q", body)
	}
}

func TestConfigStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range configOperations {
		resp := configRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplementedException") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}

func TestConfigRecorderLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := configRequest(t, ts, "PutConfigurationRecorder", `{
		"ConfigurationRecorder": {
			"Name": "stage1-recorder",
			"RoleARN": "arn:aws:iam::123456789012:role/stackyard-config-recorder"
		}
	}`)
	assertStatus(t, resp, http.StatusOK)

	resp = configRequest(t, ts, "DescribeConfigurationRecorders", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "stage1-recorder") {
		t.Fatalf("expected created recorder in DescribeConfigurationRecorders output, got %s", body)
	}

	resp = configRequest(t, ts, "StartConfigurationRecorder", `{"ConfigurationRecorderName":"stage1-recorder"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = configRequest(t, ts, "DescribeConfigurationRecorderStatus", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if !strings.Contains(body, `"Recording":true`) {
		t.Fatalf("expected recorder to be started, got %s", body)
	}

	resp = configRequest(t, ts, "StopConfigurationRecorder", `{"ConfigurationRecorderName":"stage1-recorder"}`)
	assertStatus(t, resp, http.StatusOK)

	resp = configRequest(t, ts, "DescribeConfigurationRecorderStatus", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if !strings.Contains(body, `"Recording":false`) {
		t.Fatalf("expected recorder to be stopped, got %s", body)
	}

	resp = configRequest(t, ts, "DeleteConfigurationRecorder", `{"ConfigurationRecorderName":"stage1-recorder"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = configRequest(t, ts, "DescribeConfigurationRecorders", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if strings.Contains(body, "stage1-recorder") {
		t.Fatalf("expected deleted recorder to be absent, got %s", body)
	}
}

func TestConfigStoredQueryLifecycle(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := configRequest(t, ts, "PutStoredQuery", `{
		"StoredQuery": {
			"QueryName": "stage1-query",
			"Description": "query for tests",
			"Expression": "SELECT resourceId WHERE resourceType = 'AWS::S3::Bucket'"
		}
	}`)
	assertStatus(t, resp, http.StatusOK)

	resp = configRequest(t, ts, "GetStoredQuery", `{"QueryName":"stage1-query"}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "stage1-query") {
		t.Fatalf("expected stored query to be returned, got %s", body)
	}
	if !strings.Contains(body, "AWS::S3::Bucket") {
		t.Fatalf("expected stored query expression to be returned, got %s", body)
	}

	resp = configRequest(t, ts, "ListStoredQueries", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if !strings.Contains(body, "stage1-query") {
		t.Fatalf("expected stored query metadata to include stage1-query, got %s", body)
	}

	resp = configRequest(t, ts, "DeleteStoredQuery", `{"QueryName":"stage1-query"}`)
	assertStatus(t, resp, http.StatusOK)
	resp = configRequest(t, ts, "ListStoredQueries", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body = string(mustBody(t, resp))
	if strings.Contains(body, "stage1-query") {
		t.Fatalf("expected deleted stored query to be absent, got %s", body)
	}
}
