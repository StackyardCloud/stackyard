package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stackyard/stackyard/internal/services/secretsmanager"
)

const rsdTestWorkgroup = "stackyard-workgroup"

// rsdRun submits one statement against the seeded workgroup and returns its Id.
func rsdRun(t *testing.T, ts *httptest.Server, sql string, extra map[string]any) string {
	t.Helper()
	payload := map[string]any{"Sql": sql, "Database": "dev", "WorkgroupName": rsdTestWorkgroup}
	for k, v := range extra {
		payload[k] = v
	}
	status, body := redshiftDataCall(t, ts, "ExecuteStatement", payload)
	if status != http.StatusOK {
		t.Fatalf("ExecuteStatement %q: %d %v", sql, status, body)
	}
	return body["Id"].(string)
}

func rsdDescribe(t *testing.T, ts *httptest.Server, id string) map[string]any {
	t.Helper()
	status, body := redshiftDataCall(t, ts, "DescribeStatement", map[string]any{"Id": id})
	if status != http.StatusOK {
		t.Fatalf("DescribeStatement %s: %d %v", id, status, body)
	}
	return body
}

func rsdMustFinish(t *testing.T, ts *httptest.Server, sql string) map[string]any {
	t.Helper()
	d := rsdDescribe(t, ts, rsdRun(t, ts, sql, nil))
	if d["Status"] != "FINISHED" {
		t.Fatalf("%q: %v", sql, d)
	}
	return d
}

func rsdRecords(t *testing.T, ts *httptest.Server, id string) [][]any {
	t.Helper()
	status, body := redshiftDataCall(t, ts, "GetStatementResult", map[string]any{"Id": id})
	if status != http.StatusOK {
		t.Fatalf("GetStatementResult: %d %v", status, body)
	}
	var out [][]any
	for _, r := range body["Records"].([]any) {
		out = append(out, r.([]any))
	}
	return out
}

func TestRedshiftDataStage1StatementLifecycle(t *testing.T) {
	_, ts := newRedshiftDataTestServer(t)

	status, sub := redshiftDataCall(t, ts, "ExecuteStatement", map[string]any{
		"Sql":      "CREATE TABLE users (id BIGINT IDENTITY(1,1), name VARCHAR(10) NOT NULL, score DOUBLE PRECISION, active BOOLEAN DEFAULT true) DISTSTYLE AUTO",
		"Database": "dev", "WorkgroupName": rsdTestWorkgroup, "StatementName": "create-users",
	})
	if status != http.StatusOK || sub["Status"] != "SUBMITTED" || sub["WorkgroupName"] != rsdTestWorkgroup || sub["DbUser"] != "IAM:"+testAccessKey {
		t.Fatalf("submit: %d %v", status, sub)
	}
	if d := rsdDescribe(t, ts, sub["Id"].(string)); d["Status"] != "FINISHED" || d["HasResultSet"] != false || d["QueryString"] == nil || d["Duration"].(float64) <= 0 {
		t.Fatalf("describe: %v", d)
	}

	insert := rsdRun(t, ts, "INSERT INTO users (name, score) VALUES (:n, :s), ('bob', 2.5)", map[string]any{
		"Parameters": []map[string]string{{"name": "n", "value": "ann"}, {"name": "s", "value": "9"}},
	})
	if d := rsdDescribe(t, ts, insert); d["Status"] != "FINISHED" || d["ResultRows"].(float64) != 2 || d["QueryParameters"] == nil {
		t.Fatalf("insert: %v", d)
	}

	sel := rsdRun(t, ts, "SELECT id, name, score, active FROM users WHERE name = :n", map[string]any{
		"Parameters": []map[string]string{{"name": "n", "value": "ann"}},
	})
	status, res := redshiftDataCall(t, ts, "GetStatementResult", map[string]any{"Id": sel})
	if status != http.StatusOK || res["TotalNumRows"].(float64) != 1 {
		t.Fatalf("result: %d %v", status, res)
	}
	row := res["Records"].([]any)[0].([]any)
	if row[0].(map[string]any)["longValue"].(float64) != 1 || row[1].(map[string]any)["stringValue"] != "ann" ||
		row[2].(map[string]any)["doubleValue"].(float64) != 9 || row[3].(map[string]any)["booleanValue"] != true {
		t.Fatalf("typed fields: %v", row)
	}
	meta := res["ColumnMetadata"].([]any)
	if m := meta[1].(map[string]any); m["name"] != "name" || m["typeName"] != "varchar" || m["nullable"].(float64) != 0 || m["length"].(float64) != 10 || m["tableName"] != "users" {
		t.Fatalf("metadata: %v", m)
	}

	if got := rsdRecords(t, ts, rsdRun(t, ts, "select count(*) from users", nil)); got[0][0].(map[string]any)["longValue"].(float64) != 2 {
		t.Fatalf("count(*): %v", got)
	}
	ordered := rsdRecords(t, ts, rsdRun(t, ts, "SELECT name FROM users ORDER BY score DESC LIMIT 1", nil))
	if len(ordered) != 1 || ordered[0][0].(map[string]any)["stringValue"] != "ann" {
		t.Fatalf("order by desc limit: %v", ordered)
	}
	if got := rsdRecords(t, ts, rsdRun(t, ts, "SELECT 1 AS one, 'x', NULL", nil)); got[0][0].(map[string]any)["longValue"].(float64) != 1 || got[0][2].(map[string]any)["isNull"] != true {
		t.Fatalf("literal select: %v", got)
	}

	if got := rsdRecords(t, ts, rsdRun(t, ts, "SELECT id FROM users WHERE name = 'bob'", nil)); got[0][0].(map[string]any)["longValue"].(float64) != 2 {
		t.Fatalf("identity should increase per row: %v", got)
	}

	upd := rsdDescribe(t, ts, rsdRun(t, ts, "UPDATE users SET score = 1 WHERE score IS NOT NULL", nil))
	del := rsdDescribe(t, ts, rsdRun(t, ts, "DELETE FROM users WHERE name = 'bob'", nil))
	if upd["ResultRows"].(float64) != 2 || del["ResultRows"].(float64) != 1 {
		t.Fatalf("update/delete counts: %v %v", upd, del)
	}

	// A statement without a result set has nothing to fetch.
	if status, body := redshiftDataCall(t, ts, "GetStatementResult", map[string]any{"Id": insert}); status != http.StatusBadRequest || body["__type"] != "ResourceNotFoundException" {
		t.Fatalf("no result set: %d %v", status, body)
	}

	// ListStatements defaults to FINISHED, newest first, with name prefix filter and paging.
	status, list := redshiftDataCall(t, ts, "ListStatements", map[string]any{"StatementName": "create-"})
	if items := list["Statements"].([]any); status != http.StatusOK || len(items) != 1 || items[0].(map[string]any)["StatementName"] != "create-users" {
		t.Fatalf("list by name: %d %v", status, list)
	}
	_, page := redshiftDataCall(t, ts, "ListStatements", map[string]any{"MaxResults": 2})
	if len(page["Statements"].([]any)) != 2 || page["NextToken"] == nil {
		t.Fatalf("paging: %v", page)
	}
	_, page2 := redshiftDataCall(t, ts, "ListStatements", map[string]any{"MaxResults": 2, "NextToken": page["NextToken"]})
	if page2["Statements"].([]any)[0].(map[string]any)["Id"] == page["Statements"].([]any)[0].(map[string]any)["Id"] {
		t.Fatalf("second page repeated the first: %v", page2)
	}
}

func TestRedshiftDataStage1FailuresAndCancel(t *testing.T) {
	_, ts := newRedshiftDataTestServer(t)
	rsdMustFinish(t, ts, "create table t (id int not null, v varchar(3))")

	for sql, want := range map[string]string{
		"SELECT * FROM t JOIN u ON t.id = u.id":  "syntax error",
		"GRANT ALL ON t TO bob":                  "unsupported statement",
		"SELECT * FROM missing":                  `relation "public.missing" does not exist`,
		"INSERT INTO t (v) VALUES ('a')":         "NULL value into column id",
		"INSERT INTO t VALUES (1, 'toolong')":    "value too long",
		"INSERT INTO t VALUES ('x', 'a')":        "invalid input syntax for integer",
		"SELECT * FROM t WHERE id = 1 OR id = 2": "syntax error",
		"select 1; select 2":                     "multiple commands",
		"select :missing":                        `parameter ":missing" is not provided`,
	} {
		d := rsdDescribe(t, ts, rsdRun(t, ts, sql, nil))
		if d["Status"] != "FAILED" || !strings.Contains(d["Error"].(string), want) || d["ResultRows"].(float64) != -1 {
			t.Fatalf("%q: want FAILED containing %q, got %v", sql, want, d)
		}
	}

	// Cancel aborts a still-queued statement, so its write never happens.
	id := rsdRun(t, ts, "INSERT INTO t VALUES (7, 'a')", nil)
	if status, body := redshiftDataCall(t, ts, "CancelStatement", map[string]any{"Id": id}); status != http.StatusOK || body["Status"] != true {
		t.Fatalf("cancel: %d %v", status, body)
	}
	if d := rsdDescribe(t, ts, id); d["Status"] != "ABORTED" {
		t.Fatalf("cancelled status: %v", d)
	}
	if got := rsdRecords(t, ts, rsdRun(t, ts, "select count(*) from t", nil)); got[0][0].(map[string]any)["longValue"].(float64) != 0 {
		t.Fatalf("cancelled insert ran: %v", got)
	}
	if status, body := redshiftDataCall(t, ts, "CancelStatement", map[string]any{"Id": id}); status != http.StatusBadRequest || !strings.Contains(body["message"].(string), "ABORTED") {
		t.Fatalf("cancel twice: %d %v", status, body)
	}

	// ListStatements shows only FINISHED statements unless asked for others.
	_, listed := redshiftDataCall(t, ts, "ListStatements", map[string]any{})
	for _, item := range listed["Statements"].([]any) {
		if item.(map[string]any)["Status"] != "FINISHED" {
			t.Fatalf("default listing included %v", item)
		}
	}
	if _, all := redshiftDataCall(t, ts, "ListStatements", map[string]any{"Status": "ALL"}); len(all["Statements"].([]any)) <= len(listed["Statements"].([]any)) {
		t.Fatalf("Status ALL should include failed and aborted statements")
	}

	// WaitTimeSeconds returns the final status in the submit response.
	status, body := redshiftDataCall(t, ts, "ExecuteStatement", map[string]any{"Sql": "select 1", "Database": "dev", "WorkgroupName": rsdTestWorkgroup, "WaitTimeSeconds": 5})
	if status != http.StatusOK || body["Status"] != "FINISHED" || body["HasResultSet"] != true {
		t.Fatalf("wait: %d %v", status, body)
	}
	// An unknown database fails the statement.
	d := rsdDescribe(t, ts, rsdRun(t, ts, "select 1", map[string]any{"Database": "nodb"}))
	if d["Status"] != "FAILED" || !strings.Contains(d["Error"].(string), `database "nodb" does not exist`) {
		t.Fatalf("unknown database: %v", d)
	}
}

func TestRedshiftDataStage1BatchModes(t *testing.T) {
	_, ts := newRedshiftDataTestServer(t)
	rsdMustFinish(t, ts, "create table b (id int)")
	batch := func(mode string, sqls ...string) map[string]any {
		payload := map[string]any{"Sqls": sqls, "Database": "dev", "WorkgroupName": rsdTestWorkgroup}
		if mode != "" {
			payload["ExecutionMode"] = mode
		}
		status, body := redshiftDataCall(t, ts, "BatchExecuteStatement", payload)
		if status != http.StatusOK {
			t.Fatalf("batch: %d %v", status, body)
		}
		return rsdDescribe(t, ts, body["Id"].(string))
	}
	count := func() float64 {
		return rsdRecords(t, ts, rsdRun(t, ts, "select count(*) from b", nil))[0][0].(map[string]any)["longValue"].(float64)
	}

	d := batch("", "insert into b values (1)", "insert into b values ('x')", "insert into b values (3)")
	subs := d["SubStatements"].([]any)
	if d["Status"] != "FAILED" || d["ExecutionMode"] != "TRANSACTION" || subs[1].(map[string]any)["Status"] != "FAILED" || subs[2].(map[string]any)["Status"] != "ABORTED" {
		t.Fatalf("transaction batch: %v", d)
	}
	if count() != 0 {
		t.Fatalf("failed transaction should roll back everything")
	}
	d = batch("AUTO_COMMIT", "insert into b values (1)", "insert into b values ('x')", "insert into b values (3)")
	if d["Status"] != "FAILED" || count() != 1 {
		t.Fatalf("auto-commit keeps statements before the failure: %v", d)
	}
	if d := batch("", "create database other", "select 1"); d["Status"] != "FAILED" || !strings.Contains(d["Error"].(string), "transaction block") {
		t.Fatalf("CREATE DATABASE inside a transaction: %v", d)
	}

	d = batch("", "insert into b values (5)", "select * from b where id = 5")
	if d["Status"] != "FINISHED" || d["HasResultSet"] != true {
		t.Fatalf("good batch: %v", d)
	}
	id := d["Id"].(string)
	if status, body := redshiftDataCall(t, ts, "GetStatementResult", map[string]any{"Id": id}); status != http.StatusBadRequest || body["__type"] != "ValidationException" {
		t.Fatalf("batch parent results: %d %v", status, body)
	}
	if got := rsdRecords(t, ts, id+":2"); len(got) != 1 {
		t.Fatalf("sub-statement results: %v", got)
	}
	if sub := rsdDescribe(t, ts, id+":1"); sub["Id"] != id+":1" || sub["QueryString"] != "insert into b values (5)" {
		t.Fatalf("describe sub-statement: %v", sub)
	}
	if status, _ := redshiftDataCall(t, ts, "DescribeStatement", map[string]any{"Id": id + ":3"}); status != http.StatusBadRequest {
		t.Fatalf("out-of-range sub-statement should be missing")
	}
}

func TestRedshiftDataStage1CSVAndMetadata(t *testing.T) {
	_, ts := newRedshiftDataTestServer(t)
	rsdMustFinish(t, ts, "create schema sales")
	rsdMustFinish(t, ts, "create table sales.orders (id int, note varchar(20))")
	rsdMustFinish(t, ts, "insert into sales.orders values (1, 'a,b'), (2, NULL)")
	rsdMustFinish(t, ts, "create database analytics")

	id := rsdRun(t, ts, "select * from sales.orders order by id", map[string]any{"ResultFormat": "CSV"})
	status, body := redshiftDataCall(t, ts, "GetStatementResultV2", map[string]any{"Id": id})
	if status != http.StatusOK || body["ResultFormat"] != "CSV" {
		t.Fatalf("v2: %d %v", status, body)
	}
	if got := body["Records"].([]any)[0].(map[string]any)["CSVRecords"]; got != "id,note\n1,\"a,b\"\n2,\n" {
		t.Fatalf("csv records: %q", got)
	}
	if status, _ := redshiftDataCall(t, ts, "GetStatementResult", map[string]any{"Id": id}); status != http.StatusBadRequest {
		t.Fatalf("JSON fetch of a CSV result should fail")
	}

	conn := map[string]any{"Database": "dev", "WorkgroupName": rsdTestWorkgroup}
	with := func(kv ...any) map[string]any {
		out := map[string]any{}
		for k, v := range conn {
			out[k] = v
		}
		for i := 0; i < len(kv); i += 2 {
			out[kv[i].(string)] = kv[i+1]
		}
		return out
	}
	_, dbs := redshiftDataCall(t, ts, "ListDatabases", conn)
	if strings.Join(rsdStrings(dbs["Databases"]), ",") != "analytics,dev" {
		t.Fatalf("databases: %v", dbs)
	}
	_, schemas := redshiftDataCall(t, ts, "ListSchemas", with("SchemaPattern", "s%"))
	if strings.Join(rsdStrings(schemas["Schemas"]), ",") != "sales" {
		t.Fatalf("schemas: %v", schemas)
	}
	_, tables := redshiftDataCall(t, ts, "ListTables", with("TablePattern", "ord_rs"))
	if tl := tables["Tables"].([]any); len(tl) != 1 || tl[0].(map[string]any)["schema"] != "sales" || tl[0].(map[string]any)["type"] != "TABLE" {
		t.Fatalf("tables: %v", tables)
	}
	_, desc := redshiftDataCall(t, ts, "DescribeTable", with("Schema", "sales", "Table", "orders"))
	if cols := desc["ColumnList"].([]any); desc["TableName"] != "orders" || len(cols) != 2 || cols[1].(map[string]any)["typeName"] != "varchar" {
		t.Fatalf("describe table: %v", desc)
	}
	if got := rsdRecords(t, ts, rsdRun(t, ts, "select id from sales.orders where note is null", nil)); len(got) != 1 || got[0][0].(map[string]any)["longValue"].(float64) != 2 {
		t.Fatalf("IS NULL: %v", got)
	}
	status, body = redshiftDataCall(t, ts, "ListTables", with("Database", "nodb"))
	if status != http.StatusInternalServerError || body["__type"] != "DatabaseConnectionException" {
		t.Fatalf("unknown database: %d %v", status, body)
	}
}

func rsdStrings(v any) []string {
	var out []string
	for _, item := range v.([]any) {
		out = append(out, item.(string))
	}
	return out
}

func TestRedshiftDataStage1SessionsTokensAndClusters(t *testing.T) {
	srv, ts := newRedshiftDataTestServer(t)

	status, body := redshiftDataCall(t, ts, "ExecuteStatement", map[string]any{"Sql": "create table s (id int)", "Database": "dev", "WorkgroupName": rsdTestWorkgroup, "SessionKeepAliveSeconds": 60})
	sessionID, _ := body["SessionId"].(string)
	if status != http.StatusOK || sessionID == "" {
		t.Fatalf("session start: %d %v", status, body)
	}
	status, body = redshiftDataCall(t, ts, "ExecuteStatement", map[string]any{"Sql": "insert into s values (1)", "SessionId": sessionID})
	if status != http.StatusOK || body["Database"] != "dev" || body["WorkgroupName"] != rsdTestWorkgroup {
		t.Fatalf("session reuse: %d %v", status, body)
	}
	_, sessions := redshiftDataCall(t, ts, "ListSessions", map[string]any{})
	if items := sessions["Sessions"].([]any); len(items) != 1 || items[0].(map[string]any)["Status"] != "AVAILABLE" {
		t.Fatalf("list sessions: %v", sessions)
	}
	if status, _ := redshiftDataCall(t, ts, "ListSessions", map[string]any{"SessionId": sessionID, "Status": "BUSY"}); status != http.StatusBadRequest {
		t.Fatalf("SessionId with Status should be rejected")
	}
	if status, _ := redshiftDataCall(t, ts, "ExecuteStatement", map[string]any{"Sql": "select 1", "SessionId": sessionID, "Database": "other"}); status != http.StatusBadRequest {
		t.Fatalf("a different Database with SessionId should be rejected")
	}
	srv.redshiftData.mu.Lock()
	srv.redshiftData.now = func() time.Time { return time.Now().UTC().Add(2 * time.Minute) }
	srv.redshiftData.mu.Unlock()
	if status, body := redshiftDataCall(t, ts, "ExecuteStatement", map[string]any{"Sql": "select 1", "SessionId": sessionID}); status != http.StatusBadRequest || !strings.Contains(body["message"].(string), "closed") {
		t.Fatalf("expired session: %d %v", status, body)
	}
	_, closed := redshiftDataCall(t, ts, "ListSessions", map[string]any{"Status": "CLOSED"})
	if len(closed["Sessions"].([]any)) != 1 {
		t.Fatalf("closed sessions: %v", closed)
	}
	if _, open := redshiftDataCall(t, ts, "ListSessions", map[string]any{}); len(open["Sessions"].([]any)) != 0 {
		t.Fatalf("closed sessions are hidden by default: %v", open)
	}

	// ClientToken makes a submit idempotent; reuse with other parameters fails.
	tok := map[string]any{"Sql": "select 1", "Database": "dev", "WorkgroupName": rsdTestWorkgroup, "ClientToken": "tok-1"}
	_, first := redshiftDataCall(t, ts, "ExecuteStatement", tok)
	_, again := redshiftDataCall(t, ts, "ExecuteStatement", tok)
	if first["Id"] != again["Id"] {
		t.Fatalf("ClientToken retry gave a new statement: %v %v", first, again)
	}
	tok["Sql"] = "select 2"
	if status, _ := redshiftDataCall(t, ts, "ExecuteStatement", tok); status != http.StatusBadRequest {
		t.Fatalf("ClientToken reuse with other parameters should fail")
	}

	// Provisioned cluster targets, with a secret's username and cluster check.
	srv.redshiftMu.Lock()
	srv.redshift.clusters["analytics"] = &redshiftCluster{ID: "analytics", Status: "available", DBName: "warehouse", CreatedAt: time.Unix(100, 0)}
	srv.redshiftMu.Unlock()
	good, err := srv.secretsmgr.CreateSecret(secretsmanager.CreateSecretInput{Name: "rs-good", SecretString: `{"username":"awsuser","dbClusterIdentifier":"analytics"}`})
	if err != nil {
		t.Fatal(err)
	}
	other, _ := srv.secretsmgr.CreateSecret(secretsmanager.CreateSecretInput{Name: "rs-other", SecretString: `{"username":"x","dbClusterIdentifier":"elsewhere"}`})
	status, body = redshiftDataCall(t, ts, "ExecuteStatement", map[string]any{"Sql": "create table c (id int)", "Database": "warehouse", "ClusterIdentifier": "analytics", "SecretArn": good.ARN})
	if status != http.StatusOK || body["DbUser"] != "awsuser" || body["ClusterIdentifier"] != "analytics" || body["SecretArn"] != good.ARN {
		t.Fatalf("cluster submit: %d %v", status, body)
	}
	if d := rsdDescribe(t, ts, body["Id"].(string)); d["Status"] != "FINISHED" {
		t.Fatalf("cluster statement: %v", d)
	}
	for _, secret := range []string{other.ARN, "arn:aws:secretsmanager:us-east-1:123456789012:secret:missing"} {
		if status, _ := redshiftDataCall(t, ts, "ExecuteStatement", map[string]any{"Sql": "select 1", "Database": "warehouse", "ClusterIdentifier": "analytics", "SecretArn": secret}); status != http.StatusBadRequest {
			t.Fatalf("secret %s should be rejected", secret)
		}
	}
	// A workgroup being deleted is no longer a valid target.
	srv.redshiftserverless.mu.Lock()
	srv.redshiftserverless.workgroups[rsdTestWorkgroup].Status = "DELETING"
	srv.redshiftserverless.mu.Unlock()
	if status, _ := redshiftDataCall(t, ts, "ListDatabases", map[string]any{"Database": "dev", "WorkgroupName": rsdTestWorkgroup}); status != http.StatusBadRequest {
		t.Fatalf("deleting workgroup should be rejected")
	}

	// A recreated cluster (new creation time) starts with empty databases.
	srv.redshiftMu.Lock()
	srv.redshift.clusters["analytics"].CreatedAt = time.Unix(200, 0)
	srv.redshiftMu.Unlock()
	_, tables := redshiftDataCall(t, ts, "ListTables", map[string]any{"Database": "warehouse", "ClusterIdentifier": "analytics"})
	if len(tables["Tables"].([]any)) != 0 {
		t.Fatalf("recreated cluster kept old tables: %v", tables)
	}
}
