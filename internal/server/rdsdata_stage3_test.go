package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"
)

type rdsDataHarness struct {
	t       *testing.T
	srv     *Server
	ts      *httptest.Server
	cluster string
	secret  string
}

func newRDSDataHarness(t *testing.T) *rdsDataHarness {
	srv, ts, cluster, secret := rdsDataFixture(t)
	return &rdsDataHarness{t: t, srv: srv, ts: ts, cluster: cluster, secret: secret}
}

func (h *rdsDataHarness) call(path string, body map[string]any) *http.Response {
	h.t.Helper()
	body["resourceArn"], body["secretArn"] = h.cluster, h.secret
	return rdsDataRequest(h.t, h.ts, path, body)
}

func (h *rdsDataHarness) exec(sql string, extra ...any) rdsDataExecOut {
	h.t.Helper()
	return rdsDataExec(h.t, h.call("/Execute", rdsDataBody(sql, extra...)))
}

func (h *rdsDataHarness) fail(code, sql string, extra ...any) string {
	h.t.Helper()
	return rdsDataErrCode(h.t, h.call("/Execute", rdsDataBody(sql, extra...)), http.StatusBadRequest, code)
}

func (h *rdsDataHarness) begin() string {
	h.t.Helper()
	resp := h.call("/BeginTransaction", map[string]any{})
	assertStatus(h.t, resp, http.StatusOK)
	var out struct {
		TransactionID string `json:"transactionId"`
	}
	if err := json.Unmarshal(mustBody(h.t, resp), &out); err != nil || out.TransactionID == "" {
		h.t.Fatalf("begin transaction: %v %+v", err, out)
	}
	return out.TransactionID
}

// rdsDataBody builds an ExecuteStatement body; extra is name/value pairs.
func rdsDataBody(sql string, extra ...any) map[string]any {
	body := map[string]any{"sql": sql}
	for i := 0; i+1 < len(extra); i += 2 {
		body[extra[i].(string)] = extra[i+1]
	}
	return body
}

func rdsDataParam(name string, value map[string]any) []map[string]any {
	return []map[string]any{{"name": name, "value": value}}
}

func TestRDSDataStage3TransactionConflicts(t *testing.T) {
	h := newRDSDataHarness(t)
	h.exec("CREATE TABLE t (id SERIAL PRIMARY KEY, name TEXT)")
	h.exec("CREATE TABLE u (id INT)")

	// Two transactions both generate id 1; the second commit must not
	// silently renumber its row.
	tx1, tx2 := h.begin(), h.begin()
	g1 := h.exec("INSERT INTO t (name) VALUES ('a')", "transactionId", tx1).GeneratedFields
	g2 := h.exec("INSERT INTO t (name) VALUES ('b')", "transactionId", tx2).GeneratedFields
	if g1[0]["longValue"] != float64(1) || g2[0]["longValue"] != float64(1) {
		t.Fatalf("expected both transactions to see id 1, got %v %v", g1, g2)
	}
	rdsDataErrCode(t, h.call("/CommitTransaction", map[string]any{"transactionId": tx1}), http.StatusOK, "")
	body := rdsDataErrCode(t, h.call("/CommitTransaction", map[string]any{"transactionId": tx2}), http.StatusBadRequest, "DatabaseErrorException")
	if !strings.Contains(body, "serialize") {
		t.Fatalf("expected a serialization conflict, got %s", body)
	}
	rows := h.exec("SELECT id, name FROM t").Records
	if len(rows) != 1 || rows[0][0]["longValue"] != float64(1) || rows[0][1]["stringValue"] != "a" {
		t.Fatalf("expected only the first transaction's row, got %+v", rows)
	}
	rdsDataErrCode(t, h.call("/RollbackTransaction", map[string]any{"transactionId": tx2}), http.StatusNotFound, "NotFoundException")

	// A non-transactional write to a table the transaction read conflicts.
	tx3 := h.begin()
	h.exec("SELECT * FROM t", "transactionId", tx3)
	h.exec("INSERT INTO t (name) VALUES ('c')")
	rdsDataErrCode(t, h.call("/CommitTransaction", map[string]any{"transactionId": tx3}), http.StatusBadRequest, "DatabaseErrorException")

	// Writes to tables the transaction never touched don't.
	tx4 := h.begin()
	h.exec("INSERT INTO u (id) VALUES (7)", "transactionId", tx4)
	h.exec("INSERT INTO t (name) VALUES ('d')")
	rdsDataErrCode(t, h.call("/CommitTransaction", map[string]any{"transactionId": tx4}), http.StatusOK, "")
	if rows := h.exec("SELECT * FROM u").Records; len(rows) != 1 {
		t.Fatalf("expected the committed row in u, got %+v", rows)
	}

	// A transaction that creates a table commits it; one that drops it too.
	tx5 := h.begin()
	h.exec("CREATE TABLE v (id INT)", "transactionId", tx5)
	h.exec("DROP TABLE u", "transactionId", tx5)
	rdsDataErrCode(t, h.call("/CommitTransaction", map[string]any{"transactionId": tx5}), http.StatusOK, "")
	h.exec("SELECT * FROM v")
	h.fail("DatabaseErrorException", "SELECT * FROM u")
}

func TestRDSDataStage3SweepAndLifecycle(t *testing.T) {
	h := newRDSDataHarness(t)
	now := time.Now()
	h.srv.rdsData.now = func() time.Time { return now }
	h.begin()
	h.begin()
	now = now.Add(rdsDataTxIdleTimeout + time.Second)
	h.begin()
	h.srv.rdsData.mu.Lock()
	open := len(h.srv.rdsData.txs)
	h.srv.rdsData.mu.Unlock()
	if open != 1 {
		t.Fatalf("expected expired transactions to be swept, %d still open", open)
	}

	// Rollback still works with the endpoint disabled; commit on a stopped
	// cluster reports the outage and discards the transaction.
	h.srv.rdsData.now = time.Now
	h.exec("CREATE TABLE t (id INT)")
	tx := h.begin()
	h.exec("INSERT INTO t (id) VALUES (1)", "transactionId", tx)
	if status, raw := rdsRequest(t, h.ts, url.Values{"Action": {"DisableHttpEndpoint"}, "ResourceArn": {h.cluster}}); status != http.StatusOK {
		t.Fatalf("disable HTTP endpoint: %d %s", status, raw)
	}
	rdsDataErrCode(t, h.call("/RollbackTransaction", map[string]any{"transactionId": tx}), http.StatusOK, "")
	if status, raw := rdsRequest(t, h.ts, url.Values{"Action": {"EnableHttpEndpoint"}, "ResourceArn": {h.cluster}}); status != http.StatusOK {
		t.Fatalf("enable HTTP endpoint: %d %s", status, raw)
	}
	tx = h.begin()
	h.exec("INSERT INTO t (id) VALUES (2)", "transactionId", tx)
	if status, raw := rdsRequest(t, h.ts, url.Values{"Action": {"StopDBCluster"}, "DBClusterIdentifier": {"data-api"}}); status != http.StatusOK {
		t.Fatalf("stop cluster: %d %s", status, raw)
	}
	rdsDataErrCode(t, h.call("/CommitTransaction", map[string]any{"transactionId": tx}), http.StatusGatewayTimeout, "DatabaseUnavailableException")
	rdsDataErrCode(t, h.call("/RollbackTransaction", map[string]any{"transactionId": tx}), http.StatusNotFound, "NotFoundException")
	if status, raw := rdsRequest(t, h.ts, url.Values{"Action": {"StartDBCluster"}, "DBClusterIdentifier": {"data-api"}}); status != http.StatusOK {
		t.Fatalf("start cluster: %d %s", status, raw)
	}
	if rows := h.exec("SELECT * FROM t").Records; len(rows) != 0 {
		t.Fatalf("expected neither transaction applied, got %+v", rows)
	}

	// Deleting the cluster drops its data and transactions.
	h.exec("INSERT INTO t (id) VALUES (3)")
	stale := h.begin()
	if status, raw := rdsRequest(t, h.ts, url.Values{"Action": {"DeleteDBCluster"}, "DBClusterIdentifier": {"data-api"}, "SkipFinalSnapshot": {"true"}}); status != http.StatusOK {
		t.Fatalf("delete cluster: %d %s", status, raw)
	}
	if status, raw := rdsRequest(t, h.ts, url.Values{
		"Action": {"CreateDBCluster"}, "DBClusterIdentifier": {"data-api"}, "Engine": {"aurora-postgresql"},
		"MasterUsername": {"admin"}, "MasterUserPassword": {"Secret1234"}, "DatabaseName": {"app"}, "EnableHttpEndpoint": {"true"},
	}); status != http.StatusOK {
		t.Fatalf("recreate cluster: %d %s", status, raw)
	}
	h.fail("DatabaseErrorException", "SELECT * FROM t")
	rdsDataErrCode(t, h.call("/CommitTransaction", map[string]any{"transactionId": stale}), http.StatusNotFound, "NotFoundException")

	// Unknown cluster ARNs get the documented RDS fault.
	status, raw := rdsRequest(t, h.ts, url.Values{"Action": {"EnableHttpEndpoint"}, "ResourceArn": {"arn:aws:rds:us-east-1:123456789012:cluster:nope"}})
	if status != http.StatusNotFound || !strings.Contains(string(raw), "ResourceNotFoundFault") {
		t.Fatalf("expected ResourceNotFoundFault, got %d %s", status, raw)
	}
}

func TestRDSDataStage3Values(t *testing.T) {
	h := newRDSDataHarness(t)
	h.exec("CREATE TABLE v (id BIGINT PRIMARY KEY, d DOUBLE, amount DECIMAL(10,2), b BYTEA)")

	h.fail("DatabaseErrorException", "INSERT INTO v (id, d) VALUES (1, :d)", "parameters", rdsDataParam("d", map[string]any{"stringValue": "NaN"}))
	h.fail("DatabaseErrorException", "INSERT INTO v (id, d) VALUES (1, :d)", "parameters", rdsDataParam("d", map[string]any{"stringValue": "Inf"}))
	h.fail("DatabaseErrorException", "INSERT INTO v (id) VALUES (:id)", "parameters", rdsDataParam("id", map[string]any{"doubleValue": 1e19}))
	h.fail("DatabaseErrorException", "INSERT INTO v (id) VALUES (:id)", "parameters", rdsDataParam("id", map[string]any{"doubleValue": 1.5}))

	// An explicit id is not a generated field; DECIMAL keeps its text; an
	// empty blob round-trips.
	out := h.exec("INSERT INTO v (id, amount, b) VALUES (:id, 12.50, :b)", "parameters", []map[string]any{
		{"name": "id", "value": map[string]any{"doubleValue": 2}},
		{"name": "b", "value": map[string]any{"blobValue": ""}},
	})
	if len(out.GeneratedFields) != 0 {
		t.Fatalf("expected no generated fields for an explicit key, got %+v", out.GeneratedFields)
	}
	resp := h.call("/Execute", rdsDataBody("SELECT amount, b FROM v"))
	raw := string(mustBody(t, resp))
	if !strings.Contains(raw, `"stringValue":"12.50"`) || !strings.Contains(raw, `"blobValue":""`) {
		t.Fatalf("expected DECIMAL 12.50 and an empty blob, got %s", raw)
	}
	out = h.exec("SELECT amount FROM v", "resultSetOptions", map[string]any{"decimalReturnType": "DOUBLE_OR_LONG"})
	if out.Records[0][0]["doubleValue"] != 12.5 {
		t.Fatalf("expected DOUBLE_OR_LONG to return a double, got %+v", out.Records)
	}

	// Reads use the live table directly and leave its version alone.
	h.srv.rdsData.mu.Lock()
	before := h.srv.rdsData.dbs[rdsDataKey(h.cluster, "app")]["v"]
	h.srv.rdsData.mu.Unlock()
	h.exec("SELECT * FROM v")
	h.srv.rdsData.mu.Lock()
	after := h.srv.rdsData.dbs[rdsDataKey(h.cluster, "app")]["v"]
	h.srv.rdsData.mu.Unlock()
	if before != after || before.Version != after.Version {
		t.Fatalf("expected a read to leave the table untouched")
	}
	h.exec("UPDATE v SET d = 1.5 WHERE id = 2")
	h.srv.rdsData.mu.Lock()
	written := h.srv.rdsData.dbs[rdsDataKey(h.cluster, "app")]["v"]
	h.srv.rdsData.mu.Unlock()
	if written == before || written.Version <= before.Version {
		t.Fatalf("expected a write to swap in a new table version")
	}
}

func TestRDSDataStage3Constraints(t *testing.T) {
	h := newRDSDataHarness(t)
	h.exec(`CREATE TABLE users (
		id INT PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		team TEXT DEFAULT 'core',
		org INT, seat INT,
		CONSTRAINT seat_key UNIQUE (org, seat)
	)`)
	h.exec("INSERT INTO users (id, email, org, seat) VALUES (1, 'a@x', 1, 1)")
	if body := h.fail("DatabaseErrorException", "INSERT INTO users (id, email) VALUES (1, 'b@x')"); !strings.Contains(body, "duplicate key") || !strings.Contains(body, "users_pkey") {
		t.Fatalf("expected a primary key violation, got %s", body)
	}
	h.fail("DatabaseErrorException", "INSERT INTO users (id, email) VALUES (2, 'a@x')")
	h.fail("DatabaseErrorException", "INSERT INTO users (id, email) VALUES (2, NULL)")
	h.fail("DatabaseErrorException", "INSERT INTO users (email) VALUES ('c@x')")
	if body := h.fail("DatabaseErrorException", "INSERT INTO users (id, email, org, seat) VALUES (2, 'b@x', 1, 1)"); !strings.Contains(body, "seat_key") {
		t.Fatalf("expected the composite key named, got %s", body)
	}
	// NULLs never collide in a unique key.
	h.exec("INSERT INTO users (id, email, org) VALUES (2, 'b@x', 1), (3, 'c@x', 1)")
	if out := h.exec("SELECT team FROM users WHERE id = 2"); out.Records[0][0]["stringValue"] != "core" {
		t.Fatalf("expected the default team, got %+v", out.Records)
	}
	// A violating UPDATE changes nothing.
	h.fail("DatabaseErrorException", "UPDATE users SET email = 'a@x' WHERE id = 2")
	h.fail("DatabaseErrorException", "UPDATE users SET email = NULL WHERE id = 3")
	if out := h.exec("SELECT email FROM users WHERE id = 2"); out.Records[0][0]["stringValue"] != "b@x" {
		t.Fatalf("expected the failed update to change nothing, got %+v", out.Records)
	}
	h.fail("BadRequestException", "CREATE TABLE z (created TIMESTAMP DEFAULT CURRENT_TIMESTAMP)")

	// An explicit SERIAL value is kept and isn't reported as generated; the
	// sequence continues after it.
	h.exec("CREATE TABLE s (id SERIAL PRIMARY KEY, n TEXT)")
	if out := h.exec("INSERT INTO s (id, n) VALUES (10, 'x')"); len(out.GeneratedFields) != 0 {
		t.Fatalf("expected no generated field for an explicit id, got %+v", out.GeneratedFields)
	}
	if out := h.exec("INSERT INTO s (n) VALUES ('y')"); len(out.GeneratedFields) != 1 || out.GeneratedFields[0]["longValue"] != float64(11) {
		t.Fatalf("expected the next generated id to be 11, got %+v", out.GeneratedFields)
	}
	if out := h.exec("SELECT n FROM s WHERE id = 10"); len(out.Records) != 1 {
		t.Fatalf("expected the explicit id stored, got %+v", out.Records)
	}
}

func TestRDSDataStage3Grammar(t *testing.T) {
	h := newRDSDataHarness(t)
	out := h.exec("SELECT 1", "includeResultMetadata", true)
	if len(out.Records) != 1 || out.Records[0][0]["longValue"] != float64(1) {
		t.Fatalf("expected SELECT 1 to return 1, got %+v", out.Records)
	}
	out = h.exec("SELECT 'x' AS name, :p AS p", "includeResultMetadata", true, "parameters", rdsDataParam("p", map[string]any{"longValue": 5}))
	if out.ColumnMetadata[0]["name"] != "name" || out.Records[0][1]["longValue"] != float64(5) {
		t.Fatalf("unexpected literal select: %+v %+v", out.ColumnMetadata, out.Records)
	}
	h.fail("DatabaseErrorException", "SELECT nope")

	h.exec(`CREATE TABLE "select" (id INT, "from" TEXT, note TEXT) -- a comment`)
	h.exec(`/* leading */ INSERT INTO "select" (id, "from", note) VALUES (1, 'b', NULL), (2, 'a', 'it\'s'), (3, 'c', 'x''y'), (4, 'a0', NULL)`)
	if out = h.exec(`SELECT COUNT(*) AS n FROM "select" WHERE note IS NOT NULL`); out.Records[0][0]["longValue"] != float64(2) {
		t.Fatalf("expected COUNT(*) of 2, got %+v", out.Records)
	}
	if out = h.exec(`SELECT id FROM "select" WHERE note IS NULL`); len(out.Records) != 2 || out.Records[0][0]["longValue"] != float64(1) {
		t.Fatalf("expected IS NULL to match ids 1 and 4, got %+v", out.Records)
	}
	out = h.exec(`SELECT id, note FROM "select" ORDER BY "from" DESC LIMIT 2`)
	if len(out.Records) != 2 || out.Records[0][0]["longValue"] != float64(3) || out.Records[1][0]["longValue"] != float64(1) {
		t.Fatalf("expected ORDER BY DESC with LIMIT, got %+v", out.Records)
	}
	if out = h.exec(`SELECT note FROM "select" WHERE id = 2`); out.Records[0][0]["stringValue"] != "it's" {
		t.Fatalf("expected the backslash escape decoded, got %+v", out.Records)
	}
	ids := func(out rdsDataExecOut) []float64 {
		var got []float64
		for _, rec := range out.Records {
			got = append(got, rec[0]["longValue"].(float64))
		}
		return got
	}
	if got := ids(h.exec(`SELECT id FROM "select" ORDER BY note`)); !slices.Equal(got, []float64{1, 4, 2, 3}) {
		t.Fatalf("expected NULLs first ascending, got %v", got)
	}
	if got := ids(h.exec(`SELECT id FROM "select" ORDER BY note DESC`)); !slices.Equal(got, []float64{3, 2, 1, 4}) {
		t.Fatalf("expected NULLs last descending, got %v", got)
	}
	// A quoted "from" is a column name, not the FROM keyword.
	if out = h.exec(`SELECT "from" FROM "select" WHERE id = 2`); len(out.Records) != 1 || out.Records[0][0]["stringValue"] != "a" {
		t.Fatalf("expected the quoted column, got %+v", out.Records)
	}
	h.fail("BadRequestException", `SELECT id FROM "select" WHERE id = 1 OR id = 2`)

	resp := h.call("/ExecuteSql", map[string]any{})
	rdsDataErrCode(t, resp, http.StatusBadRequest, "BadRequestException")
	resp = rdsDataRequest(t, h.ts, "/ExecuteSql", map[string]any{
		"dbClusterOrInstanceArn": h.cluster, "awsSecretStoreArn": h.secret,
		"sqlStatements": `UPDATE "select" SET note = 'a;b\';c' WHERE id = 1; -- trailing; comment
			/* ; */ SELECT note FROM "select" WHERE id = 1`,
	})
	assertStatus(t, resp, http.StatusOK)
	var legacy struct {
		Results []struct {
			ResultFrame struct {
				Records []struct {
					Values []map[string]any `json:"values"`
				} `json:"records"`
			} `json:"resultFrame"`
		} `json:"sqlStatementResults"`
	}
	if err := json.Unmarshal(mustBody(t, resp), &legacy); err != nil || len(legacy.Results) != 2 ||
		legacy.Results[1].ResultFrame.Records[0].Values[0]["stringValue"] != "a;b';c" {
		t.Fatalf("expected the splitter to respect quotes, escapes and comments: %v %+v", err, legacy)
	}
}
