package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func elastiCacheRequest(t *testing.T, ts *httptest.Server, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded; charset=utf-8",
			"User-Agent":   "aws-cli/2.0 md/command#elasticache.describe-cache-clusters",
		},
		"elasticache",
	)
}

func TestElastiCacheStage0CatalogCoverage(t *testing.T) {
	if len(elastiCacheOperations) != 75 {
		t.Fatalf("expected 75 ElastiCache operations from docs, got %d", len(elastiCacheOperations))
	}
	if len(elastiCacheOperationByName) != len(elastiCacheOperations) {
		t.Fatalf("expected unique ElastiCache operation names")
	}

	requiredActions := []string{
		"CreateCacheCluster",
		"DescribeCacheClusters",
		"ModifyCacheCluster",
		"DeleteCacheCluster",
		"CreateReplicationGroup",
		"DescribeReplicationGroups",
		"ListTagsForResource",
		"StartMigration",
	}
	for _, action := range requiredActions {
		if _, ok := elastiCacheOperationByName[action]; !ok {
			t.Fatalf("missing documented operation %s", action)
		}
	}

	if len(elastiCacheDataTypes) != 72 {
		t.Fatalf("expected 72 ElastiCache data types from docs, got %d", len(elastiCacheDataTypes))
	}
	if len(elastiCacheDataTypeByName) != len(elastiCacheDataTypes) {
		t.Fatalf("expected unique ElastiCache data type names")
	}
}

func TestElastiCacheStage0UnknownActionReturnsInvalidAction(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := elastiCacheRequest(t, ts, []byte("Action=TotallyUnknownElastiCacheAction&Version=2015-02-02"))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "InvalidAction") {
		t.Fatalf("expected InvalidAction response body, got %q", body)
	}
}

func TestElastiCacheStage0KnownActionReturnsSuccess(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := elastiCacheRequest(t, ts, []byte("Action=DescribeCacheClusters&Version=2015-02-02"))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("did not expect NotImplemented response body, got %q", body)
	}
}

func TestElastiCacheStage0AllCatalogActionsDoNotReturnNotImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	for _, op := range elastiCacheOperations {
		body := []byte(fmt.Sprintf("Action=%s&Version=2015-02-02", op.Name))
		resp := elastiCacheRequest(t, ts, body)
		respBody := string(mustBody(t, resp))
		if resp.StatusCode == http.StatusNotImplemented || strings.Contains(respBody, "NotImplemented") {
			t.Fatalf("%s unexpectedly returned NotImplemented: status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
		if resp.StatusCode >= http.StatusBadRequest {
			t.Fatalf("%s unexpectedly returned status=%d body=%s", op.Name, resp.StatusCode, respBody)
		}
	}
}
