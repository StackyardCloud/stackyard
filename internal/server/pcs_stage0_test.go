package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func pcsRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.0",
			"X-Amz-Target": "AWSParallelComputingService." + action,
		},
		"pcs",
	)
}

func TestPCSStage0CatalogCoverage(t *testing.T) {
	if len(pcsOperations) != 19 {
		t.Fatalf("expected 19 PCS operations from docs, got %d", len(pcsOperations))
	}
	if len(pcsOperationByName) != len(pcsOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateCluster",
		"GetCluster",
		"ListClusters",
		"CreateComputeNodeGroup",
		"ListComputeNodeGroups",
		"CreateQueue",
		"ListQueues",
		"TagResource",
		"UntagResource",
		"ListTagsForResource",
		"RegisterComputeNodeGroupInstance",
	}
	for _, action := range requiredActions {
		if _, ok := pcsOperationByName[action]; !ok {
			t.Fatalf("missing documented action %s", action)
		}
	}

	if len(pcsDataTypes) != 39 {
		t.Fatalf("expected 39 PCS data types from docs, got %d", len(pcsDataTypes))
	}
	if len(pcsDataTypeByName) != len(pcsDataTypes) {
		t.Fatalf("expected unique data type names")
	}

	requiredTypes := []string{
		"Cluster",
		"ClusterSummary",
		"ComputeNodeGroup",
		"ComputeNodeGroupSummary",
		"Queue",
		"QueueSummary",
		"ValidationExceptionField",
	}
	for _, typeName := range requiredTypes {
		if _, ok := pcsDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestPCSStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := pcsRequest(t, ts, "TotallyUnknownAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestPCSStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := pcsRequest(t, ts, "ListClusters", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "clusters") {
		t.Fatalf("expected ListClusters response body to include clusters, got %q", body)
	}
}

func TestPCSStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range pcsOperations {
		resp := pcsRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
