package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func redshiftServerlessRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "RedshiftServerless." + action,
		},
		"redshift-serverless",
	)
}

func TestRedshiftServerlessStage0CatalogCoverage(t *testing.T) {
	if len(redshiftServerlessOperations) != 65 {
		t.Fatalf("expected 65 Redshift Serverless operations from docs, got %d", len(redshiftServerlessOperations))
	}
	if len(redshiftServerlessOperationByName) != len(redshiftServerlessOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateNamespace",
		"CreateWorkgroup",
		"ListWorkgroups",
		"GetCredentials",
		"TagResource",
		"ListTagsForResource",
		"PutResourcePolicy",
		"GetTrack",
	}
	for _, action := range requiredActions {
		if _, ok := redshiftServerlessOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(redshiftServerlessDataTypes) != 27 {
		t.Fatalf("expected 27 Redshift Serverless data types from docs, got %d", len(redshiftServerlessDataTypes))
	}
	if len(redshiftServerlessDataTypeByName) != len(redshiftServerlessDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Namespace",
		"Workgroup",
		"Snapshot",
		"RecoveryPoint",
		"UsageLimit",
	}
	for _, typeName := range requiredTypes {
		if _, ok := redshiftServerlessDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestRedshiftServerlessStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := redshiftServerlessRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestRedshiftServerlessKnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := redshiftServerlessRequest(t, ts, "ListWorkgroups", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "workgroups") {
		t.Fatalf("expected ListWorkgroups response body to include workgroups, got %q", body)
	}
}

func TestRedshiftServerlessStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range redshiftServerlessOperations {
		resp := redshiftServerlessRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
