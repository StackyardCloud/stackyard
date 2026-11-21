package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCloudTrailDataStage0Catalog(t *testing.T) {
	if len(cloudtraildataOperations) != 1 || len(cloudtraildataOperationByName) != 1 {
		t.Fatalf("expected exactly one CloudTrail Data operation, got %d", len(cloudtraildataOperations))
	}
	if _, ok := cloudtraildataOperationByName["PutAuditEvents"]; !ok {
		t.Fatalf("missing PutAuditEvents")
	}
}

func TestCloudTrailDataStage0UnknownRouteReturnsNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := signedRequestWithService(t, http.MethodGet, ts.URL+"/GetAuditEvents", nil, nil, "cloudtrail-data")
	assertStatus(t, resp, http.StatusNotImplemented)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "NotImplementedException") {
		t.Fatalf("expected NotImplementedException, got %s", body)
	}
}

func TestCloudTrailDataStage0ControlPlaneStillRoutes(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := cloudTrailRequest(t, ts, "ListChannels", `{}`)
	assertStatus(t, resp, http.StatusOK)
	if body := string(mustBody(t, resp)); !strings.Contains(body, "Channels") {
		t.Fatalf("expected Channels in control-plane response, got %s", body)
	}
}
