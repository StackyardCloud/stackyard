package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newRedshiftDataTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Config{Addr: "127.0.0.1:0", AccessKey: testAccessKey, SecretKey: testSecretKey, LogLevel: "error"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func redshiftDataCall(t *testing.T, ts *httptest.Server, action string, payload any) (int, map[string]any) {
	t.Helper()
	return redshiftDataCallAs(t, ts, action, payload, "redshift-data")
}

func redshiftDataCallAs(t *testing.T, ts *httptest.Server, action string, payload any, service string) (int, map[string]any) {
	t.Helper()
	body, ok := payload.([]byte)
	if !ok {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			t.Fatal(err)
		}
	}
	resp := signedRequestWithService(t, http.MethodPost, ts.URL+"/", body, map[string]string{
		"Content-Type": "application/x-amz-json-1.1",
		"X-Amz-Target": "RedshiftData." + action,
	}, service)
	raw := mustBody(t, resp)
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s: non-JSON response %q", action, raw)
	}
	return resp.StatusCode, out
}

func TestRedshiftDataStage0Operations(t *testing.T) {
	if len(redshiftdataOperations) != 12 || len(redshiftdataOperationByName) != 12 {
		t.Fatalf("expected 12 unique operations, got %d/%d", len(redshiftdataOperations), len(redshiftdataOperationByName))
	}
	for _, op := range []string{"ExecuteStatement", "BatchExecuteStatement", "GetStatementResultV2", "ListSessions"} {
		if _, ok := redshiftdataOperationByName[op]; !ok {
			t.Fatalf("missing operation %s", op)
		}
	}
}

func TestRedshiftDataStage0RequestErrors(t *testing.T) {
	_, ts := newRedshiftDataTestServer(t)

	status, body := redshiftDataCall(t, ts, "Nope", map[string]any{})
	if status != http.StatusBadRequest || body["__type"] != "UnknownOperationException" {
		t.Fatalf("unknown action: %d %v", status, body)
	}
	status, body = redshiftDataCall(t, ts, "ListStatements", []byte("{not json"))
	if status != http.StatusBadRequest || body["__type"] != "SerializationException" {
		t.Fatalf("bad json: %d %v", status, body)
	}
	status, body = redshiftDataCallAs(t, ts, "ListStatements", map[string]any{}, "redshift")
	if status == http.StatusOK || body["__type"] == nil {
		t.Fatalf("wrong signing service should be rejected: %d %v", status, body)
	}
	status, body = redshiftDataCall(t, ts, "DescribeStatement", map[string]any{"Id": "not-a-uuid"})
	if status != http.StatusBadRequest || body["__type"] != "ValidationException" {
		t.Fatalf("bad id: %d %v", status, body)
	}
	status, body = redshiftDataCall(t, ts, "DescribeStatement", map[string]any{"Id": "00000000-0000-0000-0000-000000000000"})
	if status != http.StatusBadRequest || body["__type"] != "ResourceNotFoundException" || body["ResourceId"] == nil {
		t.Fatalf("missing statement: %d %v", status, body)
	}
	status, body = redshiftDataCall(t, ts, "ExecuteStatement", map[string]any{"Sql": "select 1", "Database": "dev", "WorkgroupName": "no-such-workgroup"})
	if status != http.StatusBadRequest || !strings.Contains(body["message"].(string), "doesn't exist") {
		t.Fatalf("missing workgroup: %d %v", status, body)
	}
	for _, payload := range []map[string]any{
		{"Sql": "select 1", "Database": "dev"},
		{"Sql": "select 1", "Database": "dev", "WorkgroupName": "stackyard-workgroup", "ClusterIdentifier": "c1"},
		{"Sql": "select 1", "WorkgroupName": "stackyard-workgroup"},
		{"Sql": " ", "Database": "dev", "WorkgroupName": "stackyard-workgroup"},
		{"Sql": "select 1", "Database": "dev", "WorkgroupName": "stackyard-workgroup", "ResultFormat": "XML"},
		{"Sql": "select 1", "Database": "dev", "WorkgroupName": "stackyard-workgroup", "Parameters": []any{}},
		{"Sql": "select 1", "Database": "dev", "WorkgroupName": "stackyard-workgroup", "Parameters": []any{map[string]any{"name": "a-b", "value": "1"}}},
		{"Sql": "select 1", "Database": "dev", "WorkgroupName": "stackyard-workgroup", "WaitTimeSeconds": 31},
		{"Sql": "select 1", "Database": "dev", "WorkgroupName": "stackyard-workgroup", "Parameters": []any{map[string]any{"name": "a", "value": "1"}, map[string]any{"name": "a", "value": "2"}}},
		{"Sql": "select 1", "Database": "dev", "WorkgroupName": "stackyard-workgroup", "SessionKeepAliveSeconds": 86401},
	} {
		if status, body := redshiftDataCall(t, ts, "ExecuteStatement", payload); status != http.StatusBadRequest || body["__type"] != "ValidationException" {
			t.Fatalf("payload %v: %d %v", payload, status, body)
		}
	}
	if status, body := redshiftDataCall(t, ts, "BatchExecuteStatement", map[string]any{"Sqls": []string{}, "Database": "dev", "WorkgroupName": "stackyard-workgroup"}); status != http.StatusBadRequest {
		t.Fatalf("empty batch: %d %v", status, body)
	}
	if status, body := redshiftDataCall(t, ts, "ListStatements", map[string]any{"Status": "DONE"}); status != http.StatusBadRequest {
		t.Fatalf("bad status filter: %d %v", status, body)
	}
}
