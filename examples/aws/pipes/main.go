// Command pipes walks an EventBridge Pipes lifecycle against Stackyard:
// create two SQS queues, connect them with a filtered pipe, stop and start
// it, update it, tag it, then delete everything.
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

const role = "arn:aws:iam::123456789012:role/stackyard-pipes"

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
	account := getenv("AWS_ACCOUNT_ID", "123456789012")
	fmt.Printf("Stackyard EventBridge Pipes client using %s\n", c.endpoint)

	suffix := fmt.Sprint(time.Now().UnixNano())
	source, target := "pipes-example-src-"+suffix, "pipes-example-dst-"+suffix
	for _, q := range []string{source, target} {
		c.sqs("CreateQueue", url.Values{"QueueName": {q}})
	}
	arn := func(q string) string { return fmt.Sprintf("arn:aws:sqs:%s:%s:%s", c.region, account, q) }

	name := "orders-" + suffix
	created := c.pipes("CreatePipe", http.MethodPost, "/v1/pipes/"+name, map[string]any{
		"Source": arn(source), "Target": arn(target), "RoleArn": role,
		"SourceParameters": map[string]any{
			"SqsQueueParameters": map[string]any{"BatchSize": 5},
			"FilterCriteria":     map[string]any{"Filters": []map[string]string{{"Pattern": `{"body":{"type":["order"]}}`}}},
		},
		"Tags": map[string]string{"team": "orders"},
	})
	fmt.Printf("created pipe %s (%v)\n", name, created["CurrentState"])
	waitFor(c, name, "RUNNING")

	c.pipes("StopPipe", http.MethodPost, "/v1/pipes/"+name+"/stop", nil)
	waitFor(c, name, "STOPPED")
	c.pipes("StartPipe", http.MethodPost, "/v1/pipes/"+name+"/start", nil)
	waitFor(c, name, "RUNNING")

	c.pipes("UpdatePipe", http.MethodPut, "/v1/pipes/"+name, map[string]any{
		"RoleArn": role, "Description": "orders only",
		"SourceParameters": map[string]any{"SqsQueueParameters": map[string]any{"MaximumBatchingWindowInSeconds": 10}},
	})
	waitFor(c, name, "RUNNING")

	pipeARN := created["Arn"].(string)
	c.pipes("TagResource", http.MethodPost, "/tags/"+url.PathEscape(pipeARN), map[string]any{"tags": map[string]string{"env": "dev"}})
	tags := c.pipes("ListTagsForResource", http.MethodGet, "/tags/"+url.PathEscape(pipeARN), nil)
	fmt.Printf("tags: %v\n", tags["tags"])

	listed := c.pipes("ListPipes", http.MethodGet, "/v1/pipes?NamePrefix="+url.QueryEscape(name), nil)
	fmt.Printf("ListPipes found %d pipe(s)\n", len(listed["Pipes"].([]any)))

	c.pipes("DeletePipe", http.MethodDelete, "/v1/pipes/"+name, nil)
	for _, q := range []string{source, target} {
		c.sqs("DeleteQueue", url.Values{"QueueUrl": {c.endpoint + "/" + account + "/" + q}})
	}
	fmt.Println("EventBridge Pipes example completed")
}

// waitFor reads the pipe until it reaches state (reads settle transitions).
func waitFor(c client, name, state string) {
	for i := 0; i < 5; i++ {
		got := c.pipes("DescribePipe", http.MethodGet, "/v1/pipes/"+name, nil)
		if got["CurrentState"] == state {
			fmt.Printf("pipe %s is %s\n", name, state)
			return
		}
	}
	exitf("pipe %s never reached %s", name, state)
}

func (c client) pipes(op, method, path string, payload any) map[string]any {
	var body []byte
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
	}
	raw := c.send(op, "pipes", method, c.endpoint+path, body, "application/json")
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

func (c client) sqs(action string, params url.Values) {
	params.Set("Action", action)
	params.Set("Version", "2012-11-05")
	c.send(action, "sqs", http.MethodPost, c.endpoint+"/", []byte(params.Encode()), "application/x-www-form-urlencoded")
}

// send signs a request for service and exits unless it returns 2xx.
func (c client) send(op, service, method, target string, body []byte, contentType string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", op, err)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", contentType)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	sum := sha256.Sum256(body)
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), service, c.region, time.Now()); err != nil {
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
