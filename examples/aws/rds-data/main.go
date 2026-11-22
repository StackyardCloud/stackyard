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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type client struct {
	endpoint string
	region   string
	creds    aws.CredentialsProvider
}

func main() {
	c := client{
		endpoint: getenv("STACKYARD_ENDPOINT", "http://localhost:4566"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"),
			getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
			"",
		),
	}
	ctx := context.Background()
	clusterID := getenv("STACKYARD_CLUSTER_ID", "rds-data-example")
	fmt.Printf("Stackyard RDS Data API client using %s\n", c.endpoint)

	// An Aurora cluster with the HTTP endpoint enabled, plus a secret for it.
	form := url.Values{
		"Action":              {"CreateDBCluster"},
		"Version":             {"2014-10-31"},
		"DBClusterIdentifier": {clusterID},
		"Engine":              {"aurora-postgresql"},
		"MasterUsername":      {"admin"},
		"MasterUserPassword":  {"Example1234"},
		"DatabaseName":        {"app"},
		"EnableHttpEndpoint":  {"true"},
	}
	body := c.must(ctx, "CreateDBCluster", "rds", "/", "application/x-www-form-urlencoded", "", []byte(form.Encode()))
	if !strings.Contains(string(body), "<HttpEndpointEnabled>true</HttpEndpointEnabled>") {
		exitf("CreateDBCluster did not enable the HTTP endpoint: %s", body)
	}
	clusterARN := fmt.Sprintf("arn:aws:rds:%s:%s:cluster:%s", c.region, getenv("AWS_ACCOUNT_ID", "123456789012"), clusterID)

	var secret struct {
		ARN string `json:"ARN"`
	}
	body = c.must(ctx, "CreateSecret", "secretsmanager", "/", "application/x-amz-json-1.1", "secretsmanager.CreateSecret",
		mustJSON(map[string]string{"Name": clusterID + "-creds", "SecretString": `{"username":"admin","password":"Example1234"}`}))
	_ = json.Unmarshal(body, &secret)

	target := map[string]any{"resourceArn": clusterARN, "secretArn": secret.ARN}
	execute := func(action, path string, extra map[string]any) []byte {
		payload := map[string]any{}
		for k, v := range target {
			payload[k] = v
		}
		for k, v := range extra {
			payload[k] = v
		}
		return c.must(ctx, action, "rds-data", path, "application/json", "", mustJSON(payload))
	}

	execute("ExecuteStatement", "/Execute", map[string]any{"sql": "CREATE TABLE notes (id SERIAL PRIMARY KEY, title TEXT, pinned BOOLEAN)"})
	body = execute("ExecuteStatement", "/Execute", map[string]any{
		"sql": "INSERT INTO notes (title, pinned) VALUES (:title, :pinned)",
		"parameters": []map[string]any{
			{"name": "title", "value": map[string]any{"stringValue": "hello"}},
			{"name": "pinned", "value": map[string]any{"booleanValue": true}},
		},
	})
	logf("insert: %s", strings.TrimSpace(string(body)))

	var tx struct {
		TransactionID string `json:"transactionId"`
	}
	_ = json.Unmarshal(execute("BeginTransaction", "/BeginTransaction", nil), &tx)
	execute("BatchExecuteStatement", "/BatchExecute", map[string]any{
		"sql":           "INSERT INTO notes (title, pinned) VALUES (:title, false)",
		"transactionId": tx.TransactionID,
		"parameterSets": [][]map[string]any{
			{{"name": "title", "value": map[string]any{"stringValue": "second"}}},
			{{"name": "title", "value": map[string]any{"stringValue": "third"}}},
		},
	})
	execute("CommitTransaction", "/CommitTransaction", map[string]any{"transactionId": tx.TransactionID})

	var selected struct {
		Records [][]map[string]any `json:"records"`
	}
	_ = json.Unmarshal(execute("ExecuteStatement", "/Execute", map[string]any{"sql": "SELECT id, title FROM notes WHERE pinned = false"}), &selected)
	if len(selected.Records) != 2 {
		exitf("expected 2 committed unpinned notes, got %d", len(selected.Records))
	}
	logf("unpinned notes after commit: %d", len(selected.Records))

	execute("ExecuteStatement", "/Execute", map[string]any{"sql": "DROP TABLE notes"})
	c.must(ctx, "DeleteSecret", "secretsmanager", "/", "application/x-amz-json-1.1", "secretsmanager.DeleteSecret",
		mustJSON(map[string]any{"SecretId": secret.ARN, "ForceDeleteWithoutRecovery": true}))
	form = url.Values{"Action": {"DeleteDBCluster"}, "Version": {"2014-10-31"}, "DBClusterIdentifier": {clusterID}, "SkipFinalSnapshot": {"true"}}
	c.must(ctx, "DeleteDBCluster", "rds", "/", "application/x-www-form-urlencoded", "", []byte(form.Encode()))
	logf("RDS Data API example finished")
}

// must sends a SigV4-signed POST and exits unless it returns 2xx.
func (c client) must(ctx context.Context, action, service, path, contentType, target string, body []byte) []byte {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.endpoint, "/")+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", action, err)
	}
	req.Header.Set("Content-Type", contentType)
	if target != "" {
		req.Header.Set("X-Amz-Target", target)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), service, c.region, time.Now()); err != nil {
		exitf("%s: %v", action, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	if err != nil {
		exitf("%s: %v", action, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		exitf("expected %s to return 2xx, got %d: %s", action, resp.StatusCode, strings.TrimSpace(string(out)))
	}
	logf("%s returned %d", action, resp.StatusCode)
	return out
}

func mustJSON(v any) []byte {
	out, err := json.Marshal(v)
	if err != nil {
		exitf("marshal: %v", err)
	}
	return out
}

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func logf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
