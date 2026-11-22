// Command datasync copies objects between two S3 buckets with AWS DataSync
// against Stackyard: create the buckets and locations, run a task that skips
// *.log files, and wait for the execution to finish.
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
	fmt.Printf("Stackyard DataSync client using %s\n", c.endpoint)
	suffix := time.Now().UnixNano()
	src, dst := fmt.Sprintf("datasync-src-%d", suffix), fmt.Sprintf("datasync-dst-%d", suffix)
	c.s3(http.MethodPut, "/"+src, nil)
	c.s3(http.MethodPut, "/"+dst, nil)
	for _, key := range []string{"reports/q1.csv", "reports/q2.csv", "reports/debug.log"} {
		c.s3(http.MethodPut, "/"+src+"/data/"+key, []byte("contents of "+key))
	}

	role := map[string]string{"BucketAccessRoleArn": "arn:aws:iam::123456789012:role/datasync"}
	srcLoc := c.call("CreateLocationS3", map[string]any{"S3BucketArn": "arn:aws:s3:::" + src, "Subdirectory": "/data", "S3Config": role})["LocationArn"]
	dstLoc := c.call("CreateLocationS3", map[string]any{"S3BucketArn": "arn:aws:s3:::" + dst, "Subdirectory": "/copy", "S3Config": role})["LocationArn"]
	task := c.call("CreateTask", map[string]any{
		"SourceLocationArn": srcLoc, "DestinationLocationArn": dstLoc, "Name": "stackyard-example",
		"Excludes": []map[string]string{{"FilterType": "SIMPLE_PATTERN", "Value": "*.log"}},
	})["TaskArn"]
	fmt.Printf("created task %v\n", task)

	exec := c.call("StartTaskExecution", map[string]any{"TaskArn": task})["TaskExecutionArn"]
	var got map[string]any
	for i := 0; i < 10; i++ {
		got = c.call("DescribeTaskExecution", map[string]any{"TaskExecutionArn": exec})
		fmt.Printf("execution status %v\n", got["Status"])
		if got["Status"] == "SUCCESS" || got["Status"] == "ERROR" {
			break
		}
	}
	if got["Status"] != "SUCCESS" || got["FilesTransferred"] != float64(2) {
		exitf("expected 2 files transferred successfully, got %v", got)
	}
	fmt.Printf("transferred %v files (%v bytes), *.log skipped\n", got["FilesTransferred"], got["BytesTransferred"])

	c.call("DeleteTask", map[string]any{"TaskArn": task})
	c.call("DeleteLocation", map[string]any{"LocationArn": srcLoc})
	c.call("DeleteLocation", map[string]any{"LocationArn": dstLoc})
	fmt.Println("DataSync example completed")
}

// call sends a signed DataSync JSON 1.1 request and exits unless it returns 200.
func (c client) call(op string, payload any) map[string]any {
	body, err := json.Marshal(payload)
	if err != nil {
		exitf("%s: %v", op, err)
	}
	raw := c.do(op, "datasync", http.MethodPost, "/", body, map[string]string{
		"Content-Type": "application/x-amz-json-1.1", "X-Amz-Target": "FmrsService." + op,
	})
	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			exitf("%s: decode: %v", op, err)
		}
	}
	return out
}

func (c client) s3(method, path string, body []byte) {
	c.do("S3 "+method+" "+path, "s3", method, path, body, nil)
}

func (c client) do(label, service, method, path string, body []byte, headers map[string]string) []byte {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint+path, bytes.NewReader(body))
	if err != nil {
		exitf("%s: %v", label, err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	creds, err := c.creds.Retrieve(ctx)
	if err != nil {
		exitf("%s: %v", label, err)
	}
	sum := sha256.Sum256(body)
	req.Header.Set("X-Amz-Content-Sha256", hex.EncodeToString(sum[:]))
	if err := v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(sum[:]), service, c.region, time.Now()); err != nil {
		exitf("%s: %v", label, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		exitf("%s: %v", label, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		exitf("%s: HTTP %d: %s", label, resp.StatusCode, raw)
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
