package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRDSStage0OperationCoverage(t *testing.T) {
	if len(rdsOperations) != 163 {
		t.Fatalf("expected 163 RDS operations from docs, got %d", len(rdsOperations))
	}
	if len(rdsOperationByName) != len(rdsOperations) {
		t.Fatalf("expected unique operation names")
	}

	required := []string{
		"CreateDBInstance",
		"DescribeDBInstances",
		"ModifyDBInstance",
		"DeleteDBInstance",
		"CreateDBSnapshot",
		"DescribeDBSnapshots",
		"StartExportTask",
	}
	for _, name := range required {
		if _, ok := rdsOperationByName[name]; !ok {
			t.Fatalf("missing documented operation %s", name)
		}
	}
}

func TestRDSStage0KnownActionRoutesThroughRDS(t *testing.T) {
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := []byte("Action=CreateCustomDBEngineVersion&Version=2014-10-31")
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", body, map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}, "rds")
	if got := string(mustBody(t, resp)); strings.Contains(got, "InvalidBucketName") {
		t.Fatalf("expected RDS router handling, got S3-style response %q", got)
	}
}
