package server

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	rdssvc "github.com/stackyard/stackyard/internal/services/rds"
)

type rdsDataExecOut struct {
	Records                [][]map[string]any `json:"records"`
	ColumnMetadata         []map[string]any   `json:"columnMetadata"`
	NumberOfRecordsUpdated int64              `json:"numberOfRecordsUpdated"`
	GeneratedFields        []map[string]any   `json:"generatedFields"`
	FormattedRecords       string             `json:"formattedRecords"`
}

func rdsDataExec(t *testing.T, resp *http.Response) rdsDataExecOut {
	t.Helper()
	body := mustBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
	var out rdsDataExecOut
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return out
}

func rdsDataErrCode(t *testing.T, resp *http.Response, status int, code string) string {
	t.Helper()
	body := string(mustBody(t, resp))
	if resp.StatusCode != status || resp.Header.Get("X-Amzn-ErrorType") != code {
		t.Fatalf("expected %d %s, got %d %s: %s", status, code, resp.StatusCode, resp.Header.Get("X-Amzn-ErrorType"), body)
	}
	return body
}

func TestRDSDataStage1StatementLifecycle(t *testing.T) {
	_, ts, cluster, secret := rdsDataFixture(t)
	exec := func(sql string, extra map[string]any) *http.Response {
		body := map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": sql}
		for k, v := range extra {
			body[k] = v
		}
		return rdsDataRequest(t, ts, "/Execute", body)
	}

	rdsDataExec(t, exec("CREATE TABLE items (id SERIAL PRIMARY KEY, name VARCHAR(64) NOT NULL, price DECIMAL(10,2), qty INT, active BOOLEAN, data BYTEA)", nil))

	out := rdsDataExec(t, exec("INSERT INTO items (name, price, qty, active, data) VALUES (:name, :price, :qty, :active, :data)", map[string]any{
		"parameters": []map[string]any{
			{"name": "name", "value": map[string]any{"stringValue": "widget"}},
			{"name": "price", "value": map[string]any{"stringValue": "9.5"}, "typeHint": "DECIMAL"},
			{"name": "qty", "value": map[string]any{"longValue": 3}},
			{"name": "active", "value": map[string]any{"booleanValue": true}},
			{"name": "data", "value": map[string]any{"blobValue": "AQI="}},
		},
	}))
	if out.NumberOfRecordsUpdated != 1 || len(out.GeneratedFields) != 1 || out.GeneratedFields[0]["longValue"] != float64(1) {
		t.Fatalf("unexpected insert result: %+v", out)
	}
	rdsDataExec(t, exec("INSERT INTO items (name, qty) VALUES ('gadget', 7), ('it''s', NULL)", nil))

	out = rdsDataExec(t, exec("SELECT * FROM items WHERE name = :n", map[string]any{
		"includeResultMetadata": true,
		"parameters":            []map[string]any{{"name": "n", "value": map[string]any{"stringValue": "widget"}}},
	}))
	if len(out.Records) != 1 || len(out.ColumnMetadata) != 6 {
		t.Fatalf("unexpected select: %+v", out)
	}
	row := out.Records[0]
	if row[0]["longValue"] != float64(1) || row[1]["stringValue"] != "widget" || row[2]["stringValue"] != "9.5" ||
		row[3]["longValue"] != float64(3) || row[4]["booleanValue"] != true || row[5]["blobValue"] != "AQI=" {
		t.Fatalf("unexpected row: %+v", row)
	}
	if out.ColumnMetadata[0]["typeName"] != "SERIAL" || out.ColumnMetadata[0]["isAutoIncrement"] != true || out.ColumnMetadata[1]["type"] != float64(12) {
		t.Fatalf("unexpected metadata: %+v", out.ColumnMetadata)
	}

	out = rdsDataExec(t, exec("SELECT name, qty, price FROM items WHERE qty = 3 AND name = 'widget'", map[string]any{
		"resultSetOptions": map[string]any{"longReturnType": "STRING", "decimalReturnType": "DOUBLE_OR_LONG"},
	}))
	if len(out.Records) != 1 || out.Records[0][1]["stringValue"] != "3" || out.Records[0][2]["doubleValue"] != 9.5 {
		t.Fatalf("unexpected result-set options output: %+v", out.Records)
	}
	if out = rdsDataExec(t, exec("SELECT name FROM items WHERE qty = 3 AND name = 'gadget'", nil)); len(out.Records) != 0 {
		t.Fatalf("expected AND to narrow the match, got %+v", out.Records)
	}
	if out = rdsDataExec(t, exec("SELECT name FROM items WHERE qty = NULL", nil)); len(out.Records) != 0 {
		t.Fatalf("expected = NULL to match nothing, got %+v", out.Records)
	}

	out = rdsDataExec(t, exec("UPDATE items SET qty = :q WHERE name = 'gadget'", map[string]any{
		"parameters": []map[string]any{{"name": "q", "value": map[string]any{"longValue": 8}}},
	}))
	if out.NumberOfRecordsUpdated != 1 {
		t.Fatalf("expected 1 updated row, got %+v", out)
	}
	out = rdsDataExec(t, exec("SELECT id, name, qty FROM items WHERE qty = 8", map[string]any{"formatRecordsAs": "JSON"}))
	var formatted []map[string]any
	if err := json.Unmarshal([]byte(out.FormattedRecords), &formatted); err != nil || len(formatted) != 1 || formatted[0]["name"] != "gadget" || out.Records != nil {
		t.Fatalf("unexpected formatted records: %v %q %+v", err, out.FormattedRecords, out.Records)
	}

	// NULL never matches equality, so the row with qty NULL survives.
	out = rdsDataExec(t, exec("DELETE FROM items WHERE qty = 3", nil))
	if out.NumberOfRecordsUpdated != 1 {
		t.Fatalf("expected 1 deleted row, got %+v", out)
	}
	out = rdsDataExec(t, exec("SELECT * FROM items", nil))
	if len(out.Records) != 2 {
		t.Fatalf("expected 2 rows left, got %d", len(out.Records))
	}

	batch := rdsDataRequest(t, ts, "/BatchExecute", map[string]any{
		"resourceArn": cluster, "secretArn": secret, "sql": "INSERT INTO items (name) VALUES (:n)",
		"parameterSets": [][]map[string]any{
			{{"name": "n", "value": map[string]any{"stringValue": "a"}}},
			{{"name": "n", "value": map[string]any{"stringValue": "b"}}},
		},
	})
	assertStatus(t, batch, http.StatusOK)
	var batchOut struct {
		UpdateResults []struct {
			GeneratedFields []map[string]any `json:"generatedFields"`
		} `json:"updateResults"`
	}
	if err := json.Unmarshal(mustBody(t, batch), &batchOut); err != nil || len(batchOut.UpdateResults) != 2 || batchOut.UpdateResults[1].GeneratedFields[0]["longValue"] != float64(5) {
		t.Fatalf("unexpected batch output: %v %+v", err, batchOut)
	}

	legacy := rdsDataRequest(t, ts, "/ExecuteSql", map[string]any{
		"dbClusterOrInstanceArn": cluster, "awsSecretStoreArn": secret,
		"sqlStatements": "UPDATE items SET qty = 1 WHERE name = 'a'; SELECT name FROM items WHERE name = 'a;b' ;",
	})
	assertStatus(t, legacy, http.StatusOK)
	var legacyOut struct {
		Results []struct {
			NumberOfRecordsUpdated int64          `json:"numberOfRecordsUpdated"`
			ResultFrame            map[string]any `json:"resultFrame"`
		} `json:"sqlStatementResults"`
	}
	if err := json.Unmarshal(mustBody(t, legacy), &legacyOut); err != nil || len(legacyOut.Results) != 2 ||
		legacyOut.Results[0].NumberOfRecordsUpdated != 1 || legacyOut.Results[1].ResultFrame == nil {
		t.Fatalf("unexpected ExecuteSql output: %v %+v", err, legacyOut)
	}

	rdsDataExec(t, exec("DROP TABLE items", nil))
	rdsDataErrCode(t, exec("SELECT * FROM items", nil), http.StatusBadRequest, "DatabaseErrorException")
}

func TestRDSDataStage1Errors(t *testing.T) {
	srv, ts, cluster, secret := rdsDataFixture(t)
	exec := func(body map[string]any) *http.Response { return rdsDataRequest(t, ts, "/Execute", body) }

	// resourceArn must be the cluster ARN, not its identifier.
	if _, err := srv.rds.CreateDBCluster(rdssvc.CreateDBClusterInput{
		Identifier: "identifier-only-cluster", Engine: "aurora-mysql", MasterUsername: "admin", MasterUserPassword: "Secret1234", EnableHTTPEndpoint: true,
	}); err != nil {
		t.Fatalf("create cluster: %v", err)
	}
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": "identifier-only-cluster", "secretArn": secret, "sql": "SELECT * FROM t"}), http.StatusBadRequest, "BadRequestException")

	rdsDataErrCode(t, exec(map[string]any{"secretArn": secret, "sql": "SELECT 1"}), http.StatusBadRequest, "BadRequestException")
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "sql": "SELECT 1"}), http.StatusBadRequest, "BadRequestException")
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret}), http.StatusBadRequest, "BadRequestException")
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": "arn:aws:rds:us-east-1:123456789012:cluster:missing", "secretArn": secret, "sql": "SELECT 1"}), http.StatusBadRequest, "BadRequestException")
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret + "-gone", "sql": "SELECT 1"}), http.StatusBadRequest, "SecretsErrorException")
	// Valid SQL, so only the option check can reject these.
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "CREATE TABLE opts (id INT)", "formatRecordsAs": "CSV"}), http.StatusBadRequest, "BadRequestException")
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "CREATE TABLE opts (id INT)", "resultSetOptions": map[string]any{"longReturnType": "INT"}}), http.StatusBadRequest, "BadRequestException")
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "CREATE TABLE opts (id INT)", "resultSetOptions": map[string]any{"decimalReturnType": "FLOAT"}}), http.StatusBadRequest, "BadRequestException")
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "DROP TABLE opts"}), http.StatusBadRequest, "DatabaseErrorException")
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "DROP TABLE IF EXISTS opts"}), http.StatusOK, "")

	body := rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "ALTER TABLE t ADD c INT"}), http.StatusBadRequest, "BadRequestException")
	if !strings.Contains(body, "ALTER") {
		t.Fatalf("expected the unsupported statement named, got %s", body)
	}
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "CREATE TABLE t (id INT)"}), http.StatusOK, "")
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "INSERT INTO t (id) VALUES (:missing)"}), http.StatusBadRequest, "DatabaseErrorException")
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "INSERT INTO t (id) VALUES ('abc')"}), http.StatusBadRequest, "DatabaseErrorException")
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "CREATE TABLE t (id INT)"}), http.StatusBadRequest, "DatabaseErrorException")
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "CREATE TABLE IF NOT EXISTS t (id INT)"}), http.StatusOK, "")
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "INSERT INTO t (id) VALUES (:v)",
		"parameters": []map[string]any{{"name": "v", "value": map[string]any{}}}}), http.StatusBadRequest, "BadRequestException")
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "INSERT INTO t (id) VALUES (:v)",
		"parameters": []map[string]any{{"name": "v", "value": map[string]any{"isNull": true}}}}), http.StatusOK, "")
	if out := rdsDataExec(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "SELECT * FROM t"})); len(out.Records) != 1 || out.Records[0][0]["isNull"] != true {
		t.Fatalf("expected one NULL row, got %+v", out.Records)
	}
	rdsDataExec(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "DELETE FROM t"}))
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "SELECT nope FROM t"}), http.StatusBadRequest, "DatabaseErrorException")
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "SELECT * FROM t", "transactionId": "nope"}), http.StatusNotFound, "TransactionNotFoundException")

	// A failed batch applies nothing.
	resp := rdsDataRequest(t, ts, "/BatchExecute", map[string]any{
		"resourceArn": cluster, "secretArn": secret, "sql": "INSERT INTO t (id) VALUES (:v)",
		"parameterSets": [][]map[string]any{
			{{"name": "v", "value": map[string]any{"longValue": 1}}},
			{{"name": "v", "value": map[string]any{"stringValue": "x"}}},
		},
	})
	rdsDataErrCode(t, resp, http.StatusBadRequest, "DatabaseErrorException")
	if out := rdsDataExec(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "SELECT * FROM t"})); len(out.Records) != 0 {
		t.Fatalf("expected the failed batch to apply nothing, got %+v", out.Records)
	}

	// Disabling the endpoint, then stopping the cluster, block the Data API.
	status, raw := rdsRequest(t, ts, url.Values{"Action": []string{"DisableHttpEndpoint"}, "ResourceArn": []string{cluster}})
	if status != http.StatusOK {
		t.Fatalf("disable HTTP endpoint: %d %s", status, raw)
	}
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "SELECT * FROM t"}), http.StatusBadRequest, "HttpEndpointNotEnabledException")
	status, raw = rdsRequest(t, ts, url.Values{"Action": []string{"ModifyDBCluster"}, "DBClusterIdentifier": []string{"data-api"}, "EnableHttpEndpoint": []string{"true"}})
	if status != http.StatusOK || !strings.Contains(string(raw), "<HttpEndpointEnabled>true</HttpEndpointEnabled>") {
		t.Fatalf("modify cluster HTTP endpoint: %d %s", status, raw)
	}
	status, raw = rdsRequest(t, ts, url.Values{"Action": []string{"StopDBCluster"}, "DBClusterIdentifier": []string{"data-api"}})
	if status != http.StatusOK {
		t.Fatalf("stop cluster: %d %s", status, raw)
	}
	rdsDataErrCode(t, exec(map[string]any{"resourceArn": cluster, "secretArn": secret, "sql": "SELECT * FROM t"}), http.StatusGatewayTimeout, "DatabaseUnavailableException")
}
