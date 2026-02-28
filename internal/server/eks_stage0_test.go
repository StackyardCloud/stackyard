package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEKSStage0OperationCoverage(t *testing.T) {
	if len(eksOperations) != 64 {
		t.Fatalf("expected 64 EKS operations from docs, got %d", len(eksOperations))
	}
	if len(eksOperationByName) != len(eksOperations) {
		t.Fatalf("expected unique operation names")
	}

	required := []string{
		"CreateCluster",
		"DescribeCluster",
		"ListClusters",
		"UpdateClusterConfig",
		"DeleteCluster",
		"CreateNodegroup",
		"DescribeNodegroup",
	}
	for _, name := range required {
		if _, ok := eksOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
	if _, ok := eksOperationByName["AssumeRoleForPodIdentity"]; ok {
		t.Fatalf("EKS Auth operation leaked into EKS operation catalog")
	}
}

func TestEKSStage0KnownRouteReturnsNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(
		t,
		http.MethodGet,
		ts.URL+"/addons",
		nil,
		map[string]string{"Content-Type": "application/json"},
		"eks",
	)
	if resp.StatusCode != http.StatusNotImplemented {
		t.Fatalf("expected %d, got %d", http.StatusNotImplemented, resp.StatusCode)
	}
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "NotImplementedException") {
		t.Fatalf("expected NotImplementedException response body, got %q", body)
	}
}
