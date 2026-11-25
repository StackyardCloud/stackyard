// Command snow-device-management runs AWS Snow Device Management against
// Stackyard: order a Snowball Edge job (which becomes a managed device), list
// and describe the device and its EC2 instance, unlock it with a task, follow
// the execution to SUCCEEDED, then tag the task.
//
// The SDK has no Snow Device Management module in this repository's go.mod,
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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

const signingName = "snow-device-management"

func main() {
	c := newClient()
	fmt.Printf("Stackyard Snow Device Management client using %s\n", c.endpoint)

	job := c.snowball("CreateJob", map[string]any{"JobType": "LOCAL_USE", "SnowballType": "EDGE", "Description": "field kit"})["JobId"].(string)
	fmt.Println("created Snowball job", job)

	devices := c.rest("ListDevices", http.MethodGet, "/managed-devices?jobId="+url.QueryEscape(job), nil)["devices"].([]any)
	dev := devices[0].(map[string]any)["managedDeviceId"].(string)
	d := c.rest("DescribeDevice", http.MethodPost, "/managed-device/"+dev+"/describe", nil)
	fmt.Printf("device %s (%v) is %v\n", dev, d["deviceType"], d["deviceState"])

	res := c.rest("ListDeviceResources", http.MethodGet, "/managed-device/"+dev+"/resources", nil)["resources"].([]any)
	inst := res[0].(map[string]any)["id"].(string)
	ec2 := c.rest("DescribeDeviceEc2Instances", http.MethodPost, "/managed-device/"+dev+"/resources/ec2/describe",
		map[string]any{"instanceIds": []any{inst}})["instances"].([]any)
	fmt.Printf("instance %s is %v\n", inst, ec2[0].(map[string]any)["instance"].(map[string]any)["state"].(map[string]any)["name"])

	task := c.rest("CreateTask", http.MethodPost, "/task", map[string]any{
		"command": map[string]any{"unlock": map[string]any{}}, "targets": []any{dev}, "description": "unlock on arrival",
	})
	taskID := task["taskId"].(string)
	for {
		e := c.rest("DescribeExecution", http.MethodPost, "/task/"+taskID+"/execution/"+dev, nil)
		fmt.Printf("execution on %s: %v\n", dev, e["state"])
		if e["state"] != "QUEUED" && e["state"] != "IN_PROGRESS" {
			break
		}
	}
	fmt.Println("task:", c.rest("DescribeTask", http.MethodPost, "/task/"+taskID, nil)["state"])
	fmt.Println("device now:", c.rest("DescribeDevice", http.MethodPost, "/managed-device/"+dev+"/describe", nil)["deviceState"])

	arn := url.PathEscape(task["taskArn"].(string))
	c.rest("TagResource", http.MethodPost, "/tags/"+arn, map[string]any{"tags": map[string]any{"site": "field"}})
	fmt.Println("task tags:", c.rest("ListTagsForResource", http.MethodGet, "/tags/"+arn, nil)["tags"])
}

// snowball sends one Snowball JSON operation.
func (c client) snowball(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.send(op, http.MethodPost, "snowball", c.endpoint+"/", body, map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "AWSIESnowballJobManagementService." + op,
	})
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		exitf("%s: decode: %v", op, err)
	}
	return out
}

// rest sends one REST JSON operation and exits unless it returns 2xx.
func (c client) rest(op, method, path string, payload any) map[string]any {
	var body []byte
	headers := map[string]string{}
	if payload != nil {
		var err error
		if body, err = json.Marshal(payload); err != nil {
			exitf("%s: %v", op, err)
		}
		headers["Content-Type"] = "application/json"
	}
	raw := c.send(op, method, signingName, c.endpoint+path, body, headers)
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
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
