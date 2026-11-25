// Command bcm-data-exports runs AWS Billing and Cost Management Data Exports
// against Stackyard: create an S3 bucket, look up the CUR 2.0 table, create a
// daily export of a few columns, and read back its execution and the files it
// delivered.
//
// Stackyard records no billing data, so delivered CSV files hold only the
// header row. The SDK has no Data Exports module in this repository's go.mod,
// so requests are signed by hand.
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

const (
	signingName  = "bcm-data-exports"
	targetPrefix = "AWSBillingAndCostManagementDataExports."
	contentType  = "application/x-amz-json-1.1"
)

func main() {
	c := newClient()
	fmt.Printf("Stackyard Data Exports client using %s\n", c.endpoint)

	bucket := fmt.Sprintf("billing-exports-%d", time.Now().UnixNano())
	c.send("CreateBucket", http.MethodPut, "s3", c.endpoint+"/"+bucket, nil, nil)
	fmt.Println("bucket:", bucket)

	table := c.call("GetTable", map[string]any{"TableName": "COST_AND_USAGE_REPORT", "TableProperties": map[string]any{"INCLUDE_RESOURCES": "TRUE"}})
	fmt.Printf("table %v: %d columns\n", table["TableName"], len(table["Schema"].([]any)))

	name := fmt.Sprintf("daily-cur-%d", time.Now().UnixNano())
	arn := c.call("CreateExport", map[string]any{"Export": map[string]any{
		"Name": name,
		"DataQuery": map[string]any{
			"QueryStatement":      "SELECT line_item_usage_account_id, line_item_product_code, line_item_unblended_cost, line_item_resource_id FROM COST_AND_USAGE_REPORT",
			"TableConfigurations": map[string]any{"COST_AND_USAGE_REPORT": map[string]any{"TIME_GRANULARITY": "DAILY", "INCLUDE_RESOURCES": "TRUE"}},
		},
		"DestinationConfigurations": map[string]any{"S3Destination": map[string]any{
			"S3Bucket": bucket, "S3Prefix": "cur", "S3Region": "us-east-1",
			"S3OutputConfigurations": map[string]any{"OutputType": "CUSTOM", "Format": "TEXT_OR_CSV", "Compression": "GZIP", "Overwrite": "OVERWRITE_REPORT"},
		}},
		"RefreshCadence": map[string]any{"Frequency": "SYNCHRONOUS"},
	}})["ExportArn"].(string)
	fmt.Println("export:", arn)

	status := c.call("GetExport", map[string]any{"ExportArn": arn})["ExportStatus"].(map[string]any)
	fmt.Println("export status:", status["StatusCode"])
	execs := c.call("ListExecutions", map[string]any{"ExportArn": arn})["Executions"].([]any)
	exec := execs[0].(map[string]any)
	fmt.Printf("execution %v: %v\n", exec["ExecutionId"], exec["ExecutionStatus"].(map[string]any)["StatusCode"])

	manifest := c.send("GetObject", http.MethodGet, "s3",
		c.endpoint+"/"+bucket+"/cur/"+name+"/metadata/BILLING_PERIOD="+time.Now().UTC().Format("2006-01")+"/"+name+"-Manifest.json", nil, nil)
	var m map[string]any
	if err := json.Unmarshal(manifest, &m); err != nil {
		exitf("manifest: %v", err)
	}
	fmt.Println("delivered:", m["dataFiles"])

	c.call("DeleteExport", map[string]any{"ExportArn": arn})
	fmt.Println("deleted export", name)
}

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func newClient() client {
	return client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
}

// call sends one awsJson operation and exits unless it returns 2xx.
func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(op, http.MethodPost, signingName, c.endpoint+"/", body, map[string]string{
		"Content-Type": contentType, "X-Amz-Target": targetPrefix + op,
	})
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(op, method, service, target string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, service, c.region, time.Now()); err != nil {
		exitf("%s: %v", op, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s: HTTP %d: %s", op, resp.StatusCode, raw)
	}
	return raw
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
