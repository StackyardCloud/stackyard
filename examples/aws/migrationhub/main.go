// Command migrationhub runs AWS Migration Hub against Stackyard: create a
// progress update stream, import a migration task, report its progress, link
// the discovered server and the created instance, then record the
// application's migration state.
//
// The discovered server and the application come from Application Discovery
// Service, which Stackyard seeds with srv-000001 and app-000001. The SDK has
// no Migration Hub module in this repository's go.mod, so requests are
// signed by hand.
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

type client struct {
	endpoint, region string
	creds            aws.CredentialsProvider
}

func main() {
	c := client{
		endpoint: strings.TrimRight(getenv("STACKYARD_ENDPOINT", "http://localhost:4566"), "/"),
		region:   getenv("AWS_REGION", "us-east-1"),
		creds: credentials.NewStaticCredentialsProvider(
			getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), ""),
	}
	fmt.Printf("Stackyard Migration Hub client using %s\n", c.endpoint)

	stream := fmt.Sprintf("stackyard-tool-%d", time.Now().UnixNano()%1_000_000_000)
	task := map[string]any{"ProgressUpdateStream": stream, "MigrationTaskName": "web-server"}
	c.call("CreateProgressUpdateStream", map[string]any{"ProgressUpdateStreamName": stream})
	c.call("ImportMigrationTask", task)

	start := time.Now().Add(-time.Minute)
	for i, pct := range []int{25, 75, 100} {
		status := "IN_PROGRESS"
		if pct == 100 {
			status = "COMPLETED"
		}
		c.call("NotifyMigrationTaskState", with(task, map[string]any{
			"Task":              map[string]any{"Status": status, "ProgressPercent": pct, "StatusDetail": "replicating disks"},
			"UpdateDateTime":    float64(start.Add(time.Duration(i) * 10 * time.Second).Unix()),
			"NextUpdateSeconds": 30,
		}))
	}
	c.call("PutResourceAttributes", with(task, map[string]any{"ResourceAttributeList": []map[string]any{
		{"Type": "IPV4_ADDRESS", "Value": "10.0.0.5"}, {"Type": "FQDN", "Value": "web.example.com"},
	}}))
	c.call("AssociateDiscoveredResource", with(task, map[string]any{
		"DiscoveredResource": map[string]any{"ConfigurationId": "srv-000001", "Description": "on-premises web server"},
	}))
	c.call("AssociateCreatedArtifact", with(task, map[string]any{
		"CreatedArtifact": map[string]any{"Name": "arn:aws:ec2:us-east-1:123456789012:instance/i-0123456789abcdef0", "Description": "migrated instance"},
	}))

	described := c.call("DescribeMigrationTask", task)["MigrationTask"].(map[string]any)
	fmt.Printf("task %v: %v\n", described["MigrationTaskName"], described["Task"])
	fmt.Printf("updates: %d\n", len(c.call("ListMigrationTaskUpdates", task)["MigrationTaskUpdateList"].([]any)))
	fmt.Printf("tasks for web.example.com: %d\n", len(c.call("ListMigrationTasks", map[string]any{"ResourceName": "web.example.com"})["MigrationTaskSummaryList"].([]any)))
	fmt.Printf("created artifacts: %v\n", c.call("ListCreatedArtifacts", task)["CreatedArtifactList"])

	c.call("NotifyApplicationState", map[string]any{"ApplicationId": "app-000001", "Status": "COMPLETED"})
	fmt.Printf("application state: %v\n", c.call("DescribeApplicationState", map[string]any{"ApplicationId": "app-000001"})["ApplicationStatus"])

	c.call("DeleteProgressUpdateStream", map[string]any{"ProgressUpdateStreamName": stream})
	fmt.Println("deleted progress update stream", stream)
}

func with(base map[string]any, extra map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(op, body)
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

// send signs an AWSMigrationHub.* request and exits unless it returns 2xx.
func (c client) send(op string, body []byte) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/", bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSMigrationHub."+op)
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, payloadHash, "mgh", c.region, time.Now()); err != nil {
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
