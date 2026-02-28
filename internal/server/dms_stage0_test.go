package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func dmsRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "AmazonDMSv20160101." + action,
		},
		"dms",
	)
}

func TestDMSStage0CatalogCoverage(t *testing.T) {
	if len(dmsOperations) != 119 {
		t.Fatalf("expected 119 DMS operations from docs, got %d", len(dmsOperations))
	}
	if len(dmsOperationByName) != len(dmsOperations) {
		t.Fatalf("expected unique DMS operation names")
	}

	requiredActions := []string{
		"CreateReplicationInstance",
		"CreateEndpoint",
		"CreateReplicationTask",
		"DescribeReplicationInstances",
		"DescribeReplicationTasks",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := dmsOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(dmsDataTypes) != 114 {
		t.Fatalf("expected 114 DMS data types from docs, got %d", len(dmsDataTypes))
	}
	if len(dmsDataTypeByName) != len(dmsDataTypes) {
		t.Fatalf("expected unique DMS data type names")
	}

	requiredTypes := []string{
		"ReplicationInstance",
		"ReplicationTask",
		"Endpoint",
		"Tag",
		"Event",
		"Connection",
	}
	for _, typeName := range requiredTypes {
		if _, ok := dmsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestDMSStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dmsRequest(t, ts, "UnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestDMSStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := dmsRequest(t, ts, "DescribeReplicationInstances", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "ReplicationInstances") {
		t.Fatalf("expected response body to include ReplicationInstances, got %q", body)
	}
}

func TestDMSStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range dmsOperations {
		resp := dmsRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
