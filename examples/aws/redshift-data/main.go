package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

// Runs SQL through the Redshift Data API against the Redshift Serverless
// workgroup it creates: create a table, insert with parameters, query it,
// list metadata, and cancel a queued statement.
func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	workgroup := getenv("STACKYARD_REDSHIFT_WORKGROUP", "redshift-data-example")
	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), "")
	c := client{ctx: ctx, endpoint: endpoint, region: region, creds: creds}
	conn := map[string]any{"Database": "dev", "WorkgroupName": workgroup}
	fmt.Printf("Stackyard Redshift Data API client using %s, workgroup %s\n", endpoint, workgroup)

	// The Data API needs a compute target: create a Serverless workgroup first.
	c.callService("redshift-serverless", "RedshiftServerless.CreateNamespace", map[string]any{"namespaceName": "redshift-data-example", "dbName": "dev"})
	c.callService("redshift-serverless", "RedshiftServerless.CreateWorkgroup", map[string]any{"workgroupName": workgroup, "namespaceName": "redshift-data-example"})

	run := func(sql string, extra map[string]any) string {
		payload := map[string]any{"Sql": sql}
		for k, v := range conn {
			payload[k] = v
		}
		for k, v := range extra {
			payload[k] = v
		}
		out := c.call("ExecuteStatement", payload)
		id, _ := out["Id"].(string)
		desc := c.call("DescribeStatement", map[string]any{"Id": id})
		if desc["Status"] != "FINISHED" {
			exitf("%q: status %v error %v", sql, desc["Status"], desc["Error"])
		}
		logf("%-60s -> %v rows", sql, desc["ResultRows"])
		return id
	}

	run("DROP TABLE IF EXISTS example_events", nil)
	run("CREATE TABLE example_events (id BIGINT IDENTITY(1,1), kind VARCHAR(20) NOT NULL, amount DOUBLE PRECISION)", nil)
	run("INSERT INTO example_events (kind, amount) VALUES (:kind, :amount), ('refund', -3.5)", map[string]any{
		"Parameters": []map[string]string{{"name": "kind", "value": "purchase"}, {"name": "amount", "value": "12.25"}},
	})
	query := run("SELECT id, kind, amount FROM example_events ORDER BY id", nil)
	result := c.call("GetStatementResult", map[string]any{"Id": query})
	records, _ := result["Records"].([]any)
	if len(records) != 2 {
		exitf("expected 2 records, got %v", result)
	}
	for _, rec := range records {
		logf("record %v", rec)
	}

	tables := c.call("ListTables", map[string]any{"Database": "dev", "WorkgroupName": workgroup, "TablePattern": "example%"})
	logf("tables %v", tables["Tables"])
	columns := c.call("DescribeTable", map[string]any{"Database": "dev", "WorkgroupName": workgroup, "Table": "example_events"})
	logf("columns %d", len(columns["ColumnList"].([]any)))

	queued := c.call("ExecuteStatement", map[string]any{"Sql": "DELETE FROM example_events", "Database": "dev", "WorkgroupName": workgroup})
	cancel := c.call("CancelStatement", map[string]any{"Id": queued["Id"]})
	if cancel["Status"] != true {
		exitf("cancel: %v", cancel)
	}
	logf("cancelled queued statement %v", queued["Id"])

	run("DROP TABLE example_events", nil)
	fmt.Println("Redshift Data API example finished")
}

type client struct {
	ctx      context.Context
	endpoint string
	region   string
	creds    aws.CredentialsProvider
}

func (c client) call(action string, payload map[string]any) map[string]any {
	return c.callService("redshift-data", "RedshiftData."+action, payload)
}

func (c client) callService(signingName, target string, payload map[string]any) map[string]any {
	action := target
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	req, err := http.NewRequestWithContext(c.ctx, http.MethodPost, strings.TrimRight(c.endpoint, "/")+"/", bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", action, err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", target)
	cred, err := c.creds.Retrieve(c.ctx)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(c.ctx, cred, req, hex.EncodeToString(sum[:]), signingName, c.region, time.Now()); err != nil {
		exitf("%s: %v", action, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		exitf("%s: HTTP %d: %s", action, resp.StatusCode, raw)
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		exitf("%s: %v", action, err)
	}
	return out
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func logf(format string, args ...any) { fmt.Printf(format+"\n", args...) }

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
