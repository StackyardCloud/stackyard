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
