// Command iot-jobs-data runs the AWS IoT Jobs Data Plane against Stackyard:
// create a firmware job and a command through the IoT control plane, then act
// as the device: list pending executions, start the next one, report
// progress and success (guarded by expectedVersion), describe the result and
// start a command execution.
//
// The SDK has no IoT Jobs Data module in this repository's go.mod, so
// requests are signed by hand.
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

const signingName = "iot-jobs-data"

func main() {
	c := newClient()
	fmt.Printf("Stackyard IoT Jobs Data client using %s\n", c.endpoint)
	run := time.Now().UnixNano()
	thing := fmt.Sprintf("gateway-%d", run)
	thingArn := "arn:aws:iot:us-east-1:123456789012:thing/" + thing
	job := fmt.Sprintf("firmware-%d", run)
	c.call("CreateJob", "iot", http.MethodPut, "/jobs/"+job, mustJSON(map[string]any{
		"targets": []any{thingArn}, "document": `{"url":"s3://firmware/1.4.2.bin"}`,
		"timeoutConfig": map[string]any{"inProgressTimeoutInMinutes": 30},
	}), map[string]string{"Content-Type": "application/json"})

	pending := c.call("GetPendingJobExecutions", signingName, http.MethodGet, "/things/"+thing+"/jobs", nil, nil)
	fmt.Println("queued jobs:", len(pending["queuedJobs"].([]any)))
	exec := c.call("StartNextPendingJobExecution", signingName, http.MethodPut, "/things/"+thing+"/jobs/$next",
		mustJSON(map[string]any{"statusDetails": map[string]any{"step": "download"}, "stepTimeoutInMinutes": 10}), nil)["execution"].(map[string]any)
	fmt.Printf("started %v: %v (version %v), document %v\n", exec["jobId"], exec["status"], exec["versionNumber"], exec["jobDocument"])
	state := c.call("UpdateJobExecution", signingName, http.MethodPost, "/things/"+thing+"/jobs/"+job, mustJSON(map[string]any{
		"status": "SUCCEEDED", "statusDetails": map[string]any{"step": "installed"},
		"expectedVersion": exec["versionNumber"], "includeJobExecutionState": true,
	}), nil)["executionState"].(map[string]any)
	fmt.Printf("now %v (version %v)\n", state["status"], state["versionNumber"])
	desc := c.call("DescribeJobExecution", signingName, http.MethodGet, "/things/"+thing+"/jobs/"+job, nil, nil)["execution"].(map[string]any)
	fmt.Println("described:", desc["status"], desc["statusDetails"])

	command := fmt.Sprintf("reboot-%d", run)
	c.call("CreateCommand", "iot", http.MethodPut, "/commands/"+command, mustJSON(map[string]any{
		"payload": map[string]any{"content": "eyJhY3Rpb24iOiJyZWJvb3QifQ==", "contentType": "application/json"},
	}), map[string]string{"Content-Type": "application/json"})
	started := c.call("StartCommandExecution", signingName, http.MethodPost, "/command-executions", mustJSON(map[string]any{
		"targetArn": thingArn, "commandArn": "arn:aws:iot:us-east-1:123456789012:command/" + command,
		"executionTimeoutSeconds": 60, "clientToken": fmt.Sprintf("token-%d", run),
	}), nil)
	fmt.Println("command execution", started["executionId"])
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

// call sends one signed request and exits unless it returns 2xx; a JSON
// reply is decoded.
func (c client) call(op, service, method, path string, body []byte, headers map[string]string) map[string]any {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
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
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		exitf("marshal: %v", err)
	}
	return b
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
