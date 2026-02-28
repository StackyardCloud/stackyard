package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func emrContainersRequest(t *testing.T, ts *httptest.Server, action, payload string) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		[]byte(payload),
		map[string]string{
			"Content-Type": "application/x-amz-json-1.1",
			"X-Amz-Target": "EMRContainers." + action,
		},
		"emr-containers",
	)
}

func TestEMRContainersStage0CatalogCoverage(t *testing.T) {
	if len(emrContainersOperations) != 23 {
		t.Fatalf("expected 23 EMR on EKS operations from docs, got %d", len(emrContainersOperations))
	}
	if len(emrContainersOperationByName) != len(emrContainersOperations) {
		t.Fatalf("expected unique EMR on EKS operation names")
	}

	requiredActions := []string{
		"CreateVirtualCluster",
		"DescribeVirtualCluster",
		"ListVirtualClusters",
		"StartJobRun",
		"DescribeJobRun",
		"ListJobRuns",
		"CreateManagedEndpoint",
		"DescribeManagedEndpoint",
		"ListManagedEndpoints",
		"ListTagsForResource",
	}
	for _, action := range requiredActions {
		if _, ok := emrContainersOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(emrContainersDataTypes) != 36 {
		t.Fatalf("expected 36 EMR on EKS data types from docs, got %d", len(emrContainersDataTypes))
	}
	if len(emrContainersDataTypeByName) != len(emrContainersDataTypes) {
		t.Fatalf("expected unique EMR on EKS data type names")
	}

	requiredTypes := []string{
		"VirtualCluster",
		"JobRun",
		"JobTemplate",
		"SecurityConfiguration",
		"Credentials",
	}
	for _, typeName := range requiredTypes {
		if _, ok := emrContainersDataTypeByName[typeName]; !ok {
			t.Fatalf("missing documented data type %s", typeName)
		}
	}
}

func TestEMRContainersStage0UnknownActionReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := emrContainersRequest(t, ts, "TotallyUnknownEMROnEKSAction", `{}`)
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestEMRContainersStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := emrContainersRequest(t, ts, "ListVirtualClusters", `{}`)
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "virtualClusters") {
		t.Fatalf("expected ListVirtualClusters response body to include virtualClusters, got %q", body)
	}
}

func TestEMRContainersStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range emrContainersOperations {
		resp := emrContainersRequest(t, ts, op.Name, `{}`)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
