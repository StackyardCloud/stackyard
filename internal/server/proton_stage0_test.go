package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func protonRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "AwsProton20200720." + action,
		},
		"proton",
	)
}

func TestProtonStage0CatalogCoverage(t *testing.T) {
	if len(protonOperations) != 87 {
		t.Fatalf("expected 87 Proton operations from docs, got %d", len(protonOperations))
	}
	if len(protonOperationByName) != len(protonOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateService",
		"GetService",
		"ListServices",
		"CreateEnvironmentTemplate",
		"ListRepositories",
		"TagResource",
	}
	for _, action := range requiredActions {
		if _, ok := protonOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(protonDataTypes) != 53 {
		t.Fatalf("expected 53 Proton data types from docs, got %d", len(protonDataTypes))
	}
	if len(protonDataTypeByName) != len(protonDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Service",
		"Environment",
		"ServiceTemplate",
		"EnvironmentTemplate",
		"Repository",
	}
	for _, typeName := range requiredTypes {
		if _, ok := protonDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestProtonStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := protonRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestProtonKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := protonRequest(t, ts, "ListServices", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "services") {
		t.Fatalf("expected ListServices response body to include services, got %q", body)
	}
}

func TestProtonAllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range protonOperations {
		resp := protonRequest(t, ts, op.Name, `{}`)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
