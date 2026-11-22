package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func rdsDataRequest(t *testing.T, ts *httptest.Server, path string, body any) *http.Response {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return signedRequestWithService(t, http.MethodPost, ts.URL+path, raw, map[string]string{"Content-Type": "application/json"}, "rds-data")
}

// rdsDataFixture starts a server with an Aurora cluster (HTTP endpoint on) and a
// secret, returning their ARNs.
func rdsDataFixture(t *testing.T) (*Server, *httptest.Server, string, string) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	status, body := rdsRequest(t, ts, url.Values{
		"Action":              []string{"CreateDBCluster"},
		"DBClusterIdentifier": []string{"data-api"},
		"Engine":              []string{"aurora-postgresql"},
		"MasterUsername":      []string{"admin"},
		"MasterUserPassword":  []string{"Secret1234"},
		"DatabaseName":        []string{"app"},
	})
	if status != http.StatusOK {
		t.Fatalf("create DB cluster: %d %s", status, body)
	}
	clusterARN := "arn:aws:rds:us-east-1:123456789012:cluster:data-api"
	status, body = rdsRequest(t, ts, url.Values{"Action": []string{"EnableHttpEndpoint"}, "ResourceArn": []string{clusterARN}})
	if status != http.StatusOK || !strings.Contains(string(body), "<HttpEndpointEnabled>true</HttpEndpointEnabled>") {
		t.Fatalf("enable HTTP endpoint: %d %s", status, body)
	}

	resp := secretsManagerRequest(t, ts, "CreateSecret", []byte(`{"Name":"data-api-creds","SecretString":"{\"username\":\"admin\",\"password\":\"Secret1234\"}"}`))
	assertStatus(t, resp, http.StatusOK)
	var secret struct {
		ARN string `json:"ARN"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &secret); err != nil || secret.ARN == "" {
		t.Fatalf("create secret: %v %+v", err, secret)
	}
	return srv, ts, clusterARN, secret.ARN
}

func TestRDSDataStage0OperationCoverage(t *testing.T) {
	if len(rdsdataOperations) != 6 || len(rdsdataOperationByPath) != 6 {
		t.Fatalf("expected 6 unique RDS Data API operations, got %d/%d", len(rdsdataOperations), len(rdsdataOperationByPath))
	}
	for _, path := range []string{"/Execute", "/BatchExecute", "/BeginTransaction", "/CommitTransaction", "/RollbackTransaction", "/ExecuteSql"} {
		if _, ok := rdsdataOperationByPath[path]; !ok {
			t.Fatalf("missing documented path %s", path)
		}
	}
}

func TestRDSDataStage0RoutingAndAuth(t *testing.T) {
	_, ts, _, _ := rdsDataFixture(t)

	resp := rdsDataRequest(t, ts, "/Unknown", map[string]any{})
	assertStatus(t, resp, http.StatusNotFound)
	if body := string(mustBody(t, resp)); !strings.Contains(resp.Header.Get("X-Amzn-ErrorType"), "UnknownOperationException") {
		t.Fatalf("expected UnknownOperationException, got %q %s", resp.Header.Get("X-Amzn-ErrorType"), body)
	}

	// The DSQL router sits in front of this one and must not claim rds-data calls.
	resp = rdsDataRequest(t, ts, "/Execute", map[string]any{})
	assertStatus(t, resp, http.StatusBadRequest)
	if got := resp.Header.Get("X-Amzn-ErrorType"); got != "BadRequestException" {
		t.Fatalf("expected BadRequestException for an empty body, got %q", got)
	}
}
