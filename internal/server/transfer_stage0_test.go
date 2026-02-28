package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func transferRequest(t *testing.T, ts *httptest.Server, action string, payload string) *http.Response {
	t.Helper()
	body := []byte(payload)
	headers := map[string]string{
		"Content-Type": "application/x-amz-json-1.1",
		"X-Amz-Target": "TransferService." + action,
	}
	return signedRequestWithService(t, http.MethodPost, ts.URL+"/", body, headers, "transfer")
}

func TestTransferStage0CatalogCoverage(t *testing.T) {
	if len(transferOperations) != 71 {
		t.Fatalf("expected 71 Transfer operations from docs, got %d", len(transferOperations))
	}
	if len(transferOperationByName) != len(transferOperations) {
		t.Fatalf("expected unique Transfer operation names")
	}

	requiredActions := []string{
		"CreateServer",
		"CreateUser",
		"CreateConnector",
		"ListServers",
		"ListUsers",
		"DescribeServer",
		"StartFileTransfer",
		"StartRemoteMove",
		"TagResource",
		"TestIdentityProvider",
	}
	for _, action := range requiredActions {
		if _, ok := transferOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(transferDataTypes) != 78 {
		t.Fatalf("expected 78 Transfer data types from docs, got %d", len(transferDataTypes))
	}
	if len(transferDataTypeByName) != len(transferDataTypes) {
		t.Fatalf("expected unique Transfer data type names")
	}

	requiredTypes := []string{
		"DescribedServer",
		"DescribedUser",
		"DescribedWorkflow",
		"ListedServer",
		"ListedUser",
		"ConnectorFileTransferResult",
	}
	for _, typeName := range requiredTypes {
		if _, ok := transferDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestTransferStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := transferRequest(t, ts, "UnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestTransferStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := transferRequest(t, ts, "ListServers", `{"MaxResults":10}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Servers") {
		t.Fatalf("expected ListServers response body to include Servers, got %q", body)
	}
}

func TestTransferStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range transferOperations {
		resp := transferRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
