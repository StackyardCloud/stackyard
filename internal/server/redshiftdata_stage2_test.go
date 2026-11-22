package server

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestRedshiftDataStage2ComparisonsByColumnKind(t *testing.T) {
	_, ts := newRedshiftDataTestServer(t)
	rsdMustFinish(t, ts, "create table k (code varchar(40), amount numeric(38,0))")
	rsdMustFinish(t, ts, "insert into k values ('1', 123456789012345678901234567890), ('01', 123456789012345678901234567891), ('9', 1), ('10', 1.0)")

	count := func(sql string) float64 {
		t.Helper()
		return rsdRecords(t, ts, rsdRun(t, ts, sql, nil))[0][0].(map[string]any)["longValue"].(float64)
	}
	if n := count("select count(*) from k where code = '1'"); n != 1 {
		t.Fatalf("varchar '1' must not equal '01': %v", n)
	}
	if n := count("select count(*) from k where amount = 123456789012345678901234567890"); n != 1 {
		t.Fatalf("long NUMERIC values must compare exactly: %v", n)
	}
	if n := count("select count(*) from k where amount = '1.00'"); n != 2 {
		t.Fatalf("NUMERIC 1 and 1.0 are equal: %v", n)
	}
	got := rsdRecords(t, ts, rsdRun(t, ts, "select code from k order by code", nil))
	var codes []string
	for _, r := range got {
		codes = append(codes, r[0].(map[string]any)["stringValue"].(string))
	}
	if strings.Join(codes, ",") != "01,1,10,9" {
		t.Fatalf("varchar ORDER BY must sort as text: %v", codes)
	}
	got = rsdRecords(t, ts, rsdRun(t, ts, "select amount from k order by amount desc limit 1", nil))
	if got[0][0].(map[string]any)["stringValue"] != "123456789012345678901234567891" {
		t.Fatalf("NUMERIC ORDER BY: %v", got)
	}

	// WHERE is resolved before any row is read, so errors show on empty tables too.
	rsdMustFinish(t, ts, "create table empty (id int)")
	for sql, want := range map[string]string{
		"select * from empty where nosuch = 1":              `column "nosuch" does not exist`,
		"select * from empty where id = :missing":           `parameter ":missing"`,
		"delete from empty where id = 'x'":                  "invalid input syntax for integer",
		"update empty set id = 1 where id = 2 and nope = 3": `column "nope" does not exist`,
		"select * from k where code = '1' and nope is null": `column "nope" does not exist`,
	} {
		d := rsdDescribe(t, ts, rsdRun(t, ts, sql, nil))
		if d["Status"] != "FAILED" || !strings.Contains(d["Error"].(string), want) {
			t.Fatalf("%q: want FAILED with %q, got %v", sql, want, d)
		}
	}
}

func TestRedshiftDataStage2CSVAndRollback(t *testing.T) {
	_, ts := newRedshiftDataTestServer(t)
	rsdMustFinish(t, ts, "create table c (v varchar(10))")
	rsdMustFinish(t, ts, "insert into c values (''), (NULL), (' x'), ('a\\nb')")

	id := rsdRun(t, ts, "select v from c", map[string]any{"ResultFormat": "CSV"})
	_, body := redshiftDataCall(t, ts, "GetStatementResultV2", map[string]any{"Id": id})
	if got := body["Records"].([]any)[0].(map[string]any)["CSVRecords"]; got != "v\n\"\"\n\n\" x\"\n\"a\nb\"\n" {
		t.Fatalf("CSV must tell '' from NULL and quote edge cases: %q", got)
	}

	status, out := redshiftDataCall(t, ts, "BatchExecuteStatement", map[string]any{
		"Sqls": []string{"insert into c values ('t1')", "select * from c", "insert into c values (1, 2)"}, "Database": "dev", "WorkgroupName": rsdTestWorkgroup,
	})
	if status != http.StatusOK {
		t.Fatalf("batch: %d %v", status, out)
	}
	batch := out["Id"].(string)
	d := rsdDescribe(t, ts, batch)
	subs := d["SubStatements"].([]any)
	if d["Status"] != "FAILED" || d["HasResultSet"] != false || subs[0].(map[string]any)["Status"] != "ABORTED" || subs[1].(map[string]any)["Status"] != "ABORTED" {
		t.Fatalf("rolled-back sub-statements are ABORTED: %v", d)
	}
	if status, body := redshiftDataCall(t, ts, "GetStatementResult", map[string]any{"Id": batch + ":2"}); status != http.StatusBadRequest || body["__type"] != "ResourceNotFoundException" {
		t.Fatalf("rolled-back results must be gone: %d %v", status, body)
	}
}

func TestRedshiftDataStage2LowLevelRules(t *testing.T) {
	srv, ts := newRedshiftDataTestServer(t)

	rsdMustFinish(t, ts, "create table kc (count int)")
	rsdMustFinish(t, ts, "insert into kc values (4)")
	if got := rsdRecords(t, ts, rsdRun(t, ts, "select count from kc", nil)); got[0][0].(map[string]any)["longValue"].(float64) != 4 {
		t.Fatalf("count as a column name: %v", got)
	}
	if got := rsdRecords(t, ts, rsdRun(t, ts, "select count(*)", nil)); got[0][0].(map[string]any)["longValue"].(float64) != 1 {
		t.Fatalf("COUNT(*) without FROM: %v", got)
	}
	if got := rsdRecords(t, ts, rsdRun(t, ts, `select 'tab\there', 'it\'s', 'back\\slash'`, nil)); got[0][0].(map[string]any)["stringValue"] != "tab\there" ||
		got[0][1].(map[string]any)["stringValue"] != "it's" || got[0][2].(map[string]any)["stringValue"] != `back\slash` {
		t.Fatalf("backslash escapes: %v", got)
	}
	for _, sql := range []string{"create table v1 (s varchar(70000))", "create table v2 (s varchar(x))", "create table v3 (n numeric(40,2))", "create table v4 (id int identity(a, 1))"} {
		if d := rsdDescribe(t, ts, rsdRun(t, ts, sql, nil)); d["Status"] != "FAILED" {
			t.Fatalf("%q should fail: %v", sql, d)
		}
	}
	if status, body := redshiftDataCall(t, ts, "ExecuteStatement", map[string]any{"Sql": "select '" + strings.Repeat("x", 100001) + "'", "Database": "dev", "WorkgroupName": rsdTestWorkgroup}); status != http.StatusBadRequest || body["__type"] != "ValidationException" {
		t.Fatalf("oversized SQL: %d %v", status, body["__type"])
	}

	rsdMustFinish(t, ts, "create table a_b (id int)")
	rsdMustFinish(t, ts, "create table axb (id int)")
	_, tables := redshiftDataCall(t, ts, "ListTables", map[string]any{"Database": "dev", "WorkgroupName": rsdTestWorkgroup, "TablePattern": `a\_b`})
	if tl := tables["Tables"].([]any); len(tl) != 1 || tl[0].(map[string]any)["name"] != "a_b" {
		t.Fatalf(`\_ matches a literal underscore: %v`, tables)
	}

	// A token whose statement was pruned starts a new request.
	tok := map[string]any{"Sql": "select 1", "Database": "dev", "WorkgroupName": rsdTestWorkgroup, "ClientToken": "pruned"}
	_, first := redshiftDataCall(t, ts, "ExecuteStatement", tok)
	srv.redshiftData.mu.Lock()
	delete(srv.redshiftData.statements, first["Id"].(string))
	srv.redshiftData.mu.Unlock()
	tok["Sql"] = "select 2"
	if status, again := redshiftDataCall(t, ts, "ExecuteStatement", tok); status != http.StatusOK || again["Id"] == first["Id"] {
		t.Fatalf("pruned ClientToken: %d %v", status, again)
	}

	// A session that expires while its statement is queued fails the statement.
	_, started := redshiftDataCall(t, ts, "ExecuteStatement", map[string]any{"Sql": "select 1", "Database": "dev", "WorkgroupName": rsdTestWorkgroup, "SessionKeepAliveSeconds": 30, "WaitTimeSeconds": 1})
	_, queued := redshiftDataCall(t, ts, "ExecuteStatement", map[string]any{"Sql": "select 2", "SessionId": started["SessionId"]})
	srv.redshiftData.mu.Lock()
	srv.redshiftData.now = func() time.Time { return time.Now().UTC().Add(time.Minute) }
	srv.redshiftData.mu.Unlock()
	if d := rsdDescribe(t, ts, queued["Id"].(string)); d["Status"] != "FAILED" || !strings.Contains(d["Error"].(string), "is closed") {
		t.Fatalf("expired session at run time: %v", d)
	}
}

func TestRedshiftDataStage2StoreLimits(t *testing.T) {
	st := newRedshiftDataStore()
	now := st.now()
	for i := 0; i < rsdMaxStatements+5; i++ {
		stmt := &rsdStatement{id: fmt.Sprintf("s%d", i), created: now, status: "FINISHED"}
		st.statements[stmt.id] = stmt
		st.order = append(st.order, stmt)
	}
	st.settleLocked()
	if len(st.order) != rsdMaxStatements || st.statements["s0"] != nil || st.statements[fmt.Sprintf("s%d", rsdMaxStatements+4)] == nil {
		t.Fatalf("expected the newest %d statements, got %d", rsdMaxStatements, len(st.order))
	}

	// A panic while running becomes a FAILED statement.
	broken := &rsdStatement{batch: true, execMode: "TRANSACTION", subs: []*rsdStatement{nil}}
	st.pending = broken
	st.settleLocked()
	if broken.status != "FAILED" || !strings.Contains(broken.errText, "internal error") {
		t.Fatalf("panic should fail the statement: %+v", broken)
	}

	// Data of a deleted cluster is dropped.
	srv, ts := newRedshiftDataTestServer(t)
	srv.redshiftMu.Lock()
	srv.redshift.clusters["gone"] = &redshiftCluster{ID: "gone", Status: "available", CreatedAt: time.Unix(5, 0)}
	srv.redshiftMu.Unlock()
	if status, _ := redshiftDataCall(t, ts, "ListTables", map[string]any{"Database": "dev", "ClusterIdentifier": "gone"}); status != http.StatusOK {
		t.Fatalf("cluster target should work")
	}
	srv.redshiftMu.Lock()
	delete(srv.redshift.clusters, "gone")
	srv.redshiftMu.Unlock()
	redshiftDataCall(t, ts, "ListStatements", map[string]any{})
	srv.redshiftData.mu.Lock()
	_, kept := srv.redshiftData.targets[rsdClusterKey("gone", time.Unix(5, 0))]
	srv.redshiftData.mu.Unlock()
	if kept {
		t.Fatalf("a deleted cluster's data should be pruned")
	}
}

func TestRedshiftDataStage2ParserNeverPanics(t *testing.T) {
	inputs := []string{"'", `"`, "/*", "select 1 --", "--", ";", ";;", ":", "::", "1e", "1.2.3", "select 1e", "select 1.2.3",
		"select '" + strings.Repeat("y", 100000) + "'", "create table t (v varchar(", "create table t (id int identity(",
		"create table t (id int, primary key (", "(((((", "create table t (", "insert into t values (", "select * from",
		"update t set", "delete from t where", "select count(", "select count(*", "create table t (v varchar(10,", "select -", "drop table",
		"create table t (id int default", "select * from t where id is", "select * from t order by", "select 1 limit -1"}
	for _, sql := range inputs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("%.40q panicked: %v", sql, r)
				}
			}()
			td := &rsdTargetData{dbs: map[string]*rsdDatabase{"dev": newRSDDatabase()}}
			_, _ = rsdRunSQL(td, "dev", "create table t (id int, v varchar(10))", nil, false)
			_, _ = rsdRunSQL(td, "dev", sql, nil, false)
		}()
	}
}

func TestRedshiftDataStage2RouterPrecedence(t *testing.T) {
	_, ts := newRedshiftDataTestServer(t)
	resp := redshiftServerlessRequest(t, ts, "ListWorkgroups", "{}")
	if raw := string(mustBody(t, resp)); resp.StatusCode != http.StatusOK || !strings.Contains(raw, "workgroups") {
		t.Fatalf("RedshiftServerless.* must reach the serverless router: %d %s", resp.StatusCode, raw)
	}
	form := url.Values{"Action": {"DescribeClusters"}, "Version": {"2012-12-01"}}.Encode()
	resp = signedRequestWithService(t, http.MethodPost, ts.URL+"/", []byte(form), map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, "redshift")
	if raw := string(mustBody(t, resp)); resp.StatusCode != http.StatusOK || !strings.Contains(raw, "DescribeClustersResponse") {
		t.Fatalf("a Redshift Query request must not be claimed by the Data API: %d %s", resp.StatusCode, raw)
	}
}
