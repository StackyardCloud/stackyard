package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	rdssvc "github.com/stackyard/stackyard/internal/services/rds"
)

func TestRDSDataStage2Transactions(t *testing.T) {
	srv, ts, cluster, secret := rdsDataFixture(t)
	call := func(path string, body map[string]any) *http.Response {
		body["resourceArn"], body["secretArn"] = cluster, secret
		return rdsDataRequest(t, ts, path, body)
	}
	begin := func() string {
		resp := call("/BeginTransaction", map[string]any{})
		assertStatus(t, resp, http.StatusOK)
		var out struct {
			TransactionID string `json:"transactionId"`
		}
		if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || out.TransactionID == "" {
			t.Fatalf("begin transaction: %v %+v", err, out)
		}
		return out.TransactionID
	}
	count := func(tx string) int {
		body := map[string]any{"sql": "SELECT * FROM t"}
		if tx != "" {
			body["transactionId"] = tx
		}
		return len(rdsDataExec(t, call("/Execute", body)).Records)
	}
	finish := func(path, tx, status string) {
		resp := call(path, map[string]any{"transactionId": tx})
		assertStatus(t, resp, http.StatusOK)
		var out struct {
			TransactionStatus string `json:"transactionStatus"`
		}
		if err := json.Unmarshal(mustBody(t, resp), &out); err != nil || out.TransactionStatus != status {
			t.Fatalf("%s: %v %+v", path, err, out)
		}
	}

	rdsDataExec(t, call("/Execute", map[string]any{"sql": "CREATE TABLE t (id INT, name TEXT)"}))

	// Writes inside a transaction are visible to it, not to others, until commit.
	tx := begin()
	rdsDataExec(t, call("/Execute", map[string]any{"sql": "INSERT INTO t (id) VALUES (1)", "transactionId": tx}))
	if count(tx) != 1 || count("") != 0 {
		t.Fatalf("expected isolation before commit: tx=%d live=%d", count(tx), count(""))
	}
	finish("/CommitTransaction", tx, "Transaction Committed")
	if count("") != 1 {
		t.Fatalf("expected the committed row, got %d", count(""))
	}
	rdsDataErrCode(t, call("/CommitTransaction", map[string]any{"transactionId": tx}), http.StatusNotFound, "NotFoundException")

	tx = begin()
	rdsDataExec(t, call("/Execute", map[string]any{"sql": "DELETE FROM t", "transactionId": tx}))
	finish("/RollbackTransaction", tx, "Rollback Complete")
	if count("") != 1 {
		t.Fatalf("expected rollback to keep the row, got %d", count(""))
	}
	rdsDataErrCode(t, call("/RollbackTransaction", map[string]any{"transactionId": tx}), http.StatusNotFound, "NotFoundException")
	rdsDataErrCode(t, call("/CommitTransaction", map[string]any{}), http.StatusBadRequest, "BadRequestException")

	// A commit whose replay fails leaves the live data untouched.
	tx = begin()
	rdsDataExec(t, call("/Execute", map[string]any{"sql": "INSERT INTO t (id) VALUES (2)", "transactionId": tx}))
	rdsDataExec(t, call("/Execute", map[string]any{"sql": "DROP TABLE t"}))
	rdsDataErrCode(t, call("/CommitTransaction", map[string]any{"transactionId": tx}), http.StatusBadRequest, "DatabaseErrorException")
	rdsDataErrCode(t, call("/Execute", map[string]any{"sql": "SELECT * FROM t"}), http.StatusBadRequest, "DatabaseErrorException")
	rdsDataErrCode(t, call("/Execute", map[string]any{"sql": "SELECT * FROM t", "transactionId": tx}), http.StatusNotFound, "TransactionNotFoundException")

	// Transactions expire after three idle minutes; each use resets the clock.
	rdsDataExec(t, call("/Execute", map[string]any{"sql": "CREATE TABLE t (id INT)"}))
	now := time.Now()
	srv.rdsData.now = func() time.Time { return now }
	tx = begin()
	now = now.Add(2 * time.Minute)
	count(tx)
	now = now.Add(2 * time.Minute)
	count(tx)
	now = now.Add(3*time.Minute + time.Second)
	rdsDataErrCode(t, call("/Execute", map[string]any{"sql": "SELECT * FROM t", "transactionId": tx}), http.StatusNotFound, "TransactionNotFoundException")

	// A transaction is bound to the cluster it started on.
	srv.rdsData.now = time.Now
	other, err := srv.rds.CreateDBCluster(rdssvc.CreateDBClusterInput{
		Identifier: "other", Engine: "aurora-mysql", MasterUsername: "admin", MasterUserPassword: "Secret1234", EnableHTTPEndpoint: true,
	})
	if err != nil {
		t.Fatalf("create second cluster: %v", err)
	}
	tx = begin()
	resp := rdsDataRequest(t, ts, "/CommitTransaction", map[string]any{"resourceArn": other.ARN, "secretArn": secret, "transactionId": tx})
	rdsDataErrCode(t, resp, http.StatusNotFound, "NotFoundException")
	finish("/CommitTransaction", tx, "Transaction Committed")
}
