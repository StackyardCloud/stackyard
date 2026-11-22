// Command backup walks the AWS Backup lifecycle against Stackyard: it backs up
// a DynamoDB table into a vault, restores the recovery point into a new table,
// checks the restored item is there, and cleans up.
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

const roleArn = "arn:aws:iam::123456789012:role/backup-example"

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
	signer           *v4.Signer
}

func main() {
	c := client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds:    credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
		signer:   v4.NewSigner(),
	}
	suffix := fmt.Sprint(time.Now().UnixNano())
	table, restored, vault := "orders"+suffix, "orders"+suffix+"_restored", "vault"+suffix
	fmt.Printf("AWS Backup example using %s\n", c.endpoint)

	desc := c.dynamo("CreateTable", map[string]any{
		"TableName":            table,
		"AttributeDefinitions": []any{map[string]any{"AttributeName": "id", "AttributeType": "S"}},
		"KeySchema":            []any{map[string]any{"AttributeName": "id", "KeyType": "HASH"}},
		"BillingMode":          "PAY_PER_REQUEST",
	})["TableDescription"].(map[string]any)
	tableArn := desc["TableArn"].(string)
	c.dynamo("PutItem", map[string]any{"TableName": table, "Item": map[string]any{"id": map[string]any{"S": "order-1"}}})

	c.backup("CreateBackupVault", http.MethodPut, "/backup-vaults/"+vault, map[string]any{})
	job := c.backup("StartBackupJob", http.MethodPut, "/backup-jobs", map[string]any{
		"BackupVaultName": vault, "ResourceArn": tableArn, "IamRoleArn": roleArn,
		"Lifecycle": map[string]any{"DeleteAfterDays": 7},
	})
	rpArn := job["RecoveryPointArn"].(string)
	c.wait("backup job", "/backup-jobs/"+job["BackupJobId"].(string), "State")

	rp := c.backup("DescribeRecoveryPoint", http.MethodGet, "/backup-vaults/"+vault+"/recovery-points/"+url.PathEscape(rpArn), nil)
	fmt.Printf("recovery point %s is %v\n", rpArn, rp["Status"])

	restore := c.backup("StartRestoreJob", http.MethodPut, "/restore-jobs", map[string]any{
		"RecoveryPointArn": rpArn, "IamRoleArn": roleArn, "Metadata": map[string]any{"targetTableName": restored},
	})
	done := c.wait("restore job", "/restore-jobs/"+restore["RestoreJobId"].(string), "Status")
	fmt.Printf("restored table %v\n", done["CreatedResourceArn"])

	scan := c.dynamo("Scan", map[string]any{"TableName": restored})
	if scan["Count"] != float64(1) {
		exitf("restored table should hold 1 item, got %v", scan["Count"])
	}

	c.backup("DeleteRecoveryPoint", http.MethodDelete, "/backup-vaults/"+vault+"/recovery-points/"+url.PathEscape(rpArn), nil)
	c.backup("DeleteBackupVault", http.MethodDelete, "/backup-vaults/"+vault, nil)
	for _, name := range []string{table, restored} {
		c.dynamo("DeleteTable", map[string]any{"TableName": name})
	}
	fmt.Println("done")
}

// wait reads a job until its state member is COMPLETED (each read advances it).
func (c client) wait(what, path, stateKey string) map[string]any {
	for range 10 {
		out := c.backup("Describe "+what, http.MethodGet, path, nil)
		switch out[stateKey] {
		case "COMPLETED":
			return out
		case "FAILED", "ABORTED":
			exitf("%s ended %v: %v", what, out[stateKey], out["StatusMessage"])
		}
	}
	exitf("%s did not complete", what)
	return nil
}

func (c client) backup(op, method, path string, body any) map[string]any {
	return c.call(op, method, path, body, "backup", nil)
}

func (c client) dynamo(op string, body any) map[string]any {
	return c.call(op, http.MethodPost, "/", body, "dynamodb", map[string]string{
		"Content-Type": "application/x-amz-json-1.0", "X-Amz-Target": "DynamoDB_20120810." + op,
	})
}

// call sends one signed request and exits unless it succeeds.
func (c client) call(op, method, path string, body any, service string, headers map[string]string) map[string]any {
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	req, err := http.NewRequest(method, c.endpoint+path, bytes.NewReader(raw))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", hash)
	creds, err := c.creds.Retrieve(context.Background())
	if err != nil {
		exitf("%s: credentials: %v", op, err)
	}
	if err := c.signer.SignHTTP(context.Background(), creds, req, hash, service, c.region, time.Now()); err != nil {
		exitf("%s: sign: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, data)
	}
	out := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	fmt.Printf("%s: HTTP %d\n", op, resp.StatusCode)
	return out
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
