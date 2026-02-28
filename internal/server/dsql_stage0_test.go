package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDSQLStage0OperationCoverage(t *testing.T) {
	if len(dsqlOperations) != 12 {
		t.Fatalf("expected 12 DSQL operations from docs, got %d", len(dsqlOperations))
	}
	if len(dsqlOperationByName) != len(dsqlOperations) {
		t.Fatalf("expected unique operation names")
	}

	required := []string{
		"CreateCluster",
		"DeleteCluster",
		"GetCluster",
		"ListClusters",
		"UpdateCluster",
		"GetVpcEndpointServiceName",
	}
	for _, name := range required {
		if _, ok := dsqlOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestDSQLStage0KnownRouteReturnsNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(
		t,
		http.MethodGet,
		ts.URL+"/clusters/dsql0000000000000000000001/unsupported",
		nil,
		map[string]string{"Content-Type": "application/json"},
		"dsql",
	)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected %d, got %d", http.StatusNotImplemented, resp.StatusCode)
	}
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "NotImplementedException") {
		t.Fatalf("expected NotImplementedException response body, got %q", body)
	}
}
