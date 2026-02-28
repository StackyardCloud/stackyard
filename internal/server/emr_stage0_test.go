package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func emrRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "ElasticMapReduce." + action,
		},
		"elasticmapreduce",
	)
}

func TestEMRStage0CatalogCoverage(t *testing.T) {
	if len(emrOperations) != 60 {
		t.Fatalf("expected 60 EMR operations from docs, got %d", len(emrOperations))
	}
	if len(emrOperationByName) != len(emrOperations) {
		t.Fatalf("expected unique EMR operation names")
	}

	requiredActions := []string{
		"RunJobFlow",
		"ListClusters",
		"DescribeCluster",
		"AddJobFlowSteps",
		"ListSteps",
		"CancelSteps",
		"CreateStudio",
		"ListStudios",
		"AddTags",
		"RemoveTags",
	}
	for _, action := range requiredActions {
		if _, ok := emrOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(emrDataTypes) != 106 {
		t.Fatalf("expected 106 EMR data types from docs, got %d", len(emrDataTypes))
	}
	if len(emrDataTypeByName) != len(emrDataTypes) {
		t.Fatalf("expected unique EMR data type names")
	}
}

func TestEMRStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := emrRequest(t, ts, "TotallyUnknownEMRAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestEMRStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := emrRequest(t, ts, "ListClusters", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
}

func TestEMRStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range emrOperations {
		resp := emrRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
