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
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type restCall struct {
	Name    string
	Method  string
	Path    string
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

	fmt.Printf("Stackyard Braket advanced client using %s\n", endpoint)

	jobArn := url.PathEscape("arn:aws:braket:us-east-1:123456789012:job/job-000001")
	taskArn := url.PathEscape("arn:aws:braket:us-east-1:123456789012:quantum-task/task-000001")
	limitArn := url.PathEscape("arn:aws:braket:us-east-1:123456789012:spending-limit/limit-000001")
	deviceArn := url.PathEscape("arn:aws:braket:us-east-1::device/qpu/test-device")
	resourceArn := url.PathEscape("arn:aws:braket:us-east-1:123456789012:job/job-000001")

	calls := []restCall{
		{Name: "SearchDevices", Method: http.MethodPost, Path: "/devices", Payload: map[string]any{}},
		{Name: "GetDevice", Method: http.MethodGet, Path: "/device/" + deviceArn, Payload: nil},
		{Name: "CreateJob", Method: http.MethodPost, Path: "/job", Payload: map[string]any{}},
		{Name: "GetJob", Method: http.MethodGet, Path: "/job/" + jobArn + "?additionalAttributeNames=queueInfo", Payload: nil},
		{Name: "CancelJob", Method: http.MethodPut, Path: "/job/" + jobArn + "/cancel", Payload: map[string]any{}},
		{Name: "SearchJobs", Method: http.MethodPost, Path: "/jobs", Payload: map[string]any{}},
		{Name: "CreateQuantumTask", Method: http.MethodPost, Path: "/quantum-task", Payload: map[string]any{}},
		{Name: "GetQuantumTask", Method: http.MethodGet, Path: "/quantum-task/" + taskArn + "?additionalAttributeNames=queueInfo", Payload: nil},
		{Name: "CancelQuantumTask", Method: http.MethodPut, Path: "/quantum-task/" + taskArn + "/cancel", Payload: map[string]any{}},
		{Name: "SearchQuantumTasks", Method: http.MethodPost, Path: "/quantum-tasks", Payload: map[string]any{}},
		{Name: "CreateSpendingLimit", Method: http.MethodPost, Path: "/spending-limit", Payload: map[string]any{}},
		{Name: "UpdateSpendingLimit", Method: http.MethodPatch, Path: "/spending-limit/" + limitArn + "/update", Payload: map[string]any{"amount": 500}},
		{Name: "SearchSpendingLimits", Method: http.MethodPost, Path: "/spending-limits", Payload: map[string]any{}},
		{Name: "TagResource", Method: http.MethodPost, Path: "/tags/" + resourceArn, Payload: map[string]any{"tags": map[string]string{"env": "local", "suite": "advanced"}}},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/tags/" + resourceArn, Payload: nil},
		{Name: "UntagResource", Method: http.MethodDelete, Path: "/tags/" + resourceArn + "?tagKeys=suite", Payload: nil},
		{Name: "DeleteSpendingLimit", Method: http.MethodDelete, Path: "/spending-limit/" + limitArn + "/delete", Payload: nil},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := braketRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
	if err != nil {
		return err
	}
	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func braketRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload == nil {
		body = []byte{}
	} else {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(endpoint, "/")+path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "braket", region, time.Now()); err != nil {
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
