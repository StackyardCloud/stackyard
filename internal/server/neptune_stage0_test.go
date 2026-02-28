package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func neptuneRequest(t *testing.T, ts *httptest.Server, body []byte) *http.Response {
	t.Helper()
	return signedRequestWithService(
		t,
		http.MethodPost,
		ts.URL+"/",
		body,
		map[string]string{
			"Content-Type": "application/x-www-form-urlencoded; charset=utf-8",
			"User-Agent":   "aws-cli/2.0 md/command#neptune.describe-db-clusters",
		},
		"rds",
	)
}

func TestNeptuneStage0OperationCoverage(t *testing.T) {
	if len(neptuneOperations) != 70 {
		t.Fatalf("expected 70 Neptune operations from docs, got %d", len(neptuneOperations))
	}
	if len(neptuneOperationByName) != len(neptuneOperations) {
		t.Fatalf("expected unique operation names")
	}

	required := []string{
		"CreateDBCluster",
		"DescribeDBClusters",
		"ModifyDBCluster",
		"DeleteDBCluster",
		"ListTagsForResource",
		"SwitchoverGlobalCluster",
	}
	for _, name := range required {
		if _, ok := neptuneOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestNeptuneStage0CreateDBInstanceIsImplemented(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := neptuneRequest(t, ts, []byte("Action=CreateDBInstance&Version=2014-10-31"))
	assertStatus(t, resp, http.StatusOK)
	body := string(mustBody(t, resp))
	if strings.Contains(body, "NotImplemented") {
		t.Fatalf("expected CreateDBInstance to be implemented, got %q", body)
	}
}

func TestNeptuneStage0UnknownActionReturnsInvalidAction(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp := neptuneRequest(t, ts, []byte("Action=TotallyUnknownNeptuneAction&Version=2014-10-31"))
	assertStatus(t, resp, http.StatusBadRequest)
	body := string(mustBody(t, resp))
	if !strings.Contains(body, "InvalidAction") {
		t.Fatalf("expected InvalidAction response body, got %q", body)
	}
}
