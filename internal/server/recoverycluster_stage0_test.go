package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func recoveryClusterRequest(t *testing.T, ts *httptest.Server, method, path, payload string) *http.Response {
	t.Helper()
	var body []byte
	headers := map[string]string{}
	if strings.TrimSpace(payload) != "" {
		body = []byte(payload)
		headers["Content-Type"] = "application/json"
	}
	return signedRequestWithService(t, method, ts.URL+path, body, headers, "route53-recovery-control-config")
}

func TestRecoveryClusterStage0CatalogCoverage(t *testing.T) {
	if len(recoveryClusterOperations) != 25 {
		t.Fatalf("expected 25 Recovery Cluster operations from docs, got %d", len(recoveryClusterOperations))
	}
	if len(recoveryClusterOperationByName) != len(recoveryClusterOperations) {
		t.Fatalf("expected unique operation names")
	}

	requiredActions := []string{
		"CreateCluster",
		"DescribeCluster",
		"ListClusters",
		"CreateControlPanel",
		"ListRoutingControls",
		"CreateSafetyRule",
		"GetResourcePolicy",
		"TagResource",
		"UntagResource",
	}
	for _, action := range requiredActions {
		if _, ok := recoveryClusterOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(recoveryClusterDataTypes) != 22 {
		t.Fatalf("expected 22 Recovery Cluster data types from docs, got %d", len(recoveryClusterDataTypes))
	}
	if len(recoveryClusterDataTypeByName) != len(recoveryClusterDataTypes) {
		t.Fatalf("expected unique data type names")
	}
}

func TestRecoveryClusterStage0UnknownRouteReturnsValidationError(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := recoveryClusterRequest(t, ts, http.MethodGet, "/cluster/cluster-000001/unknown", "")
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "ValidationException") {
		t.Fatalf("expected ValidationException response body, got %q", body)
	}
}

func TestRecoveryClusterKnownRouteReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := recoveryClusterRequest(t, ts, http.MethodGet, "/cluster", "")
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
	if !strings.Contains(body, "Clusters") {
		t.Fatalf("expected ListClusters response body to include Clusters, got %q", body)
	}
}

func TestRecoveryClusterStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	replacer := strings.NewReplacer(
		"{ClusterArn}", "cluster-000001",
		"{ControlPanelArn}", "controlpanel-000001",
		"{RoutingControlArn}", "routingcontrol-000001",
		"{SafetyRuleArn}", "safetyrule-000001",
		"{ResourceArn}", "resource-000001",
	)

	for _, op := range recoveryClusterOperations {
		path := replacer.Replace(op.URI)
		payload := ""
		if op.Method == http.MethodPost || op.Method == http.MethodPut {
			payload = `{}`
		}
		resp := recoveryClusterRequest(t, ts, op.Method, path, payload)
		body := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(body, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, body)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, body)
		}
	}
}
