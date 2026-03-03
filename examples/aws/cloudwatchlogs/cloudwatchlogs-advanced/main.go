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
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type apiCall struct {
	Action  string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	logGroupName := "/stackyard/cloudwatchlogs-advanced"
	logStreamName := "stackyard-advanced-stream"

	calls := []apiCall{
		{Action: "CreateLogGroup", Payload: map[string]any{"logGroupName": logGroupName}},
		{Action: "CreateLogStream", Payload: map[string]any{"logGroupName": logGroupName, "logStreamName": logStreamName}},
		{Action: "PutLogEvents", Payload: map[string]any{
			"logGroupName":  logGroupName,
			"logStreamName": logStreamName,
			"logEvents": []map[string]any{{
				"timestamp": time.Now().UTC().UnixMilli(),
				"message":   "stackyard advanced log message",
			}},
		}},
		{Action: "GetLogEvents", Payload: map[string]any{"logGroupName": logGroupName, "logStreamName": logStreamName}},
		{Action: "FilterLogEvents", Payload: map[string]any{"logGroupName": logGroupName}},
		{Action: "DescribeLogGroups", Payload: map[string]any{}},
	}

	fmt.Printf("Stackyard CloudWatch Logs advanced client using %s\n", endpoint)
	for _, call := range calls {
		status, body, err := cloudWatchLogsRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
		if err != nil {
			exitf("%s failed: %v", call.Action, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s returned HTTP %d: %s", call.Action, status, strings.TrimSpace(string(body)))
		}
		fmt.Printf("%s succeeded\n", call.Action)
	}

	fmt.Println("Done.")
}

func cloudWatchLogsRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/", bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "Logs_20140328."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "logs", region, time.Now()); err != nil {
		return 0, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, respBody, nil
}

func hashSHA256(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
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
