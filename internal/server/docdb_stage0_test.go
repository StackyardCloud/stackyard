package server

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newDocDBTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

// docdbCall sends a DocumentDB Query action the way the CLI does (rds
// signing, docdb User-Agent) and returns the status and body.
func docdbCall(t *testing.T, ts *httptest.Server, action string, params map[string]string) (int, string) {
	t.Helper()
	return queryCall(t, ts, "aws-cli/2.41.0 md/command#docdb.x", action, params)
}

func queryCall(t *testing.T, ts *httptest.Server, userAgent, action string, params map[string]string) (int, string) {
	t.Helper()
	form := url.Values{"Action": {action}, "Version": {"2014-10-31"}}
	for k, v := range params {
		form.Set(k, v)
	}
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(form.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded; charset=utf-8",
		"User-Agent":   userAgent,
	}, "rds")
	return resp.StatusCode, string(mustBody(t, resp))
}

func docdbWant(t *testing.T, ts *httptest.Server, status int, action string, params map[string]string) string {
	t.Helper()
	got, body := docdbCall(t, ts, action, params)
	if got != status {
		t.Fatalf("%s %v: status %d, want %d: %s", action, params, got, status, body)
	}
	return body
}

func docdbWantError(t *testing.T, ts *httptest.Server, status int, code, action string, params map[string]string) {
	t.Helper()
	body := docdbWant(t, ts, status, action, params)
	if !strings.Contains(body, "<Code>"+code+"</Code>") {
		t.Fatalf("%s: want error code %s, got %s", action, code, body)
	}
}

func TestDocDBStage0OperationCatalog(t *testing.T) {
	if len(docdbOperations) != 56 || len(docdbOperationByName) != len(docdbOperations) {
		t.Fatalf("expected 56 unique DocumentDB operations, got %d (%d unique)", len(docdbOperations), len(docdbOperationByName))
	}
	for _, op := range docdbOperations {
		_, own := docdbHandlers[op.Name]
		delegated := false
		for _, name := range docdbDelegatedToRDS {
			delegated = delegated || name == op.Name
		}
		if own == delegated {
			t.Errorf("%s: want exactly one of a docdb handler or RDS delegation (own=%v delegated=%v)", op.Name, own, delegated)
		}
		if delegated {
			if _, ok := rdsQueryHandlers[op.Name]; !ok {
				t.Errorf("%s is delegated but RDS has no handler", op.Name)
			}
		}
	}
}

func TestDocDBStage0RoutingKeepsRDSAndNeptuneTraffic(t *testing.T) {
	ts := newDocDBTestServer(t)
	// An rds-CLI call with Engine=mysql still reaches the RDS handler.
	status, body := queryCall(t, ts, "aws-cli/2.41.0 md/command#rds.create-db-cluster", "CreateDBCluster", map[string]string{
		"DBClusterIdentifier": "plain-rds", "Engine": "aurora-mysql", "MasterUsername": "admin", "MasterUserPassword": "Passw0rd!123",
	})
	if status != http.StatusOK || !strings.Contains(body, "<Engine>aurora-mysql</Engine>") {
		t.Fatalf("rds CreateDBCluster: %d %s", status, body)
	}
	// Docdb-only actions aren't claimed for rds clients: RDS answers
	// DescribeEventCategories with its own (no-op) handler, not ours.
	if _, body := queryCall(t, ts, "aws-cli/2.41.0 md/command#rds.x", "DescribeEventCategories", nil); strings.Contains(body, "db-cluster-snapshot") {
		t.Fatalf("rds client got the docdb catalog: %s", body)
	}
	// The docdb-elastic CLI is a different service and must not match.
	if hasDocDBRequestHint(&http.Request{Header: http.Header{"User-Agent": {"aws-cli/2 md/command#docdb-elastic.list-clusters"}}, URL: &url.URL{Path: "/"}}) {
		t.Fatalf("docdb-elastic matched the docdb hint")
	}
	// Neptune keeps its own router.
	status, _ = queryCall(t, ts, "aws-cli/2.41.0 md/command#neptune.describe-db-clusters", "DescribeDBClusters", nil)
	if status != http.StatusOK {
		t.Fatalf("neptune DescribeDBClusters: %d", status)
	}
	// DocumentDB sees the RDS cluster (shared control plane) and can filter it out.
	body = docdbWant(t, ts, http.StatusOK, "DescribeDBClusters", nil)
	if !strings.Contains(body, "plain-rds") {
		t.Fatalf("docdb DescribeDBClusters should list every engine unfiltered: %s", body)
	}
	body = docdbWant(t, ts, http.StatusOK, "DescribeDBClusters", map[string]string{"Filters.Filter.1.Name": "engine", "Filters.Filter.1.Values.Value.1": "docdb"})
	if strings.Contains(body, "plain-rds") {
		t.Fatalf("engine=docdb filter kept an aurora cluster: %s", body)
	}
}

func TestDocDBStage0RejectsBadRequests(t *testing.T) {
	ts := newDocDBTestServer(t)
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidAction", "CreateDBProxy", nil)
	// Signed for another service: rejected.
	form := url.Values{"Action": {"DescribeDBClusters"}, "Version": {"2014-10-31"}}
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(form.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded", "User-Agent": "aws-cli/2 md/command#docdb.x",
	}, "s3")
	if body := string(mustBody(t, resp)); resp.StatusCode == http.StatusOK {
		t.Fatalf("s3-signed docdb call accepted: %s", body)
	}
	var out struct {
		XMLName xml.Name
	}
	body := docdbWant(t, ts, http.StatusOK, "DescribeDBEngineVersions", map[string]string{"Engine": "docdb"})
	if err := xml.Unmarshal([]byte(body), &out); err != nil || out.XMLName.Local != "DescribeDBEngineVersionsResponse" {
		t.Fatalf("engine versions: %v %s", err, body)
	}
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterValue", "DescribeDBEngineVersions", map[string]string{"Engine": "mysql"})
	docdbWantError(t, ts, http.StatusBadRequest, "MissingParameter", "DescribeOrderableDBInstanceOptions", nil)
	docdbWantError(t, ts, http.StatusBadRequest, "InvalidParameterValue", "DescribeDBClusters", map[string]string{"MaxRecords": "5"})
}
