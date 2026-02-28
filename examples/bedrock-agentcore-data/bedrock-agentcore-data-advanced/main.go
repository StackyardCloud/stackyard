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

	fmt.Printf("Stackyard Bedrock AgentCore Data Plane advanced client using %s\n", endpoint)

	runtimeARN := "arn:aws:bedrock-agentcore:us-east-1:123456789012:runtime/stackyard-runtime"
	runtimePath := url.PathEscape(runtimeARN)

	calls := []restCall{
		{Name: "ListMemoryRecords", Method: http.MethodPost, Path: "/memories/memory-000001/memoryRecords", Payload: map[string]any{}},
		{Name: "BatchCreateMemoryRecords", Method: http.MethodPost, Path: "/memories/memory-000001/memoryRecords/batchCreate", Payload: map[string]any{}},
		{Name: "RetrieveMemoryRecords", Method: http.MethodPost, Path: "/memories/memory-000001/retrieve", Payload: map[string]any{}},
		{Name: "StartBrowserSession", Method: http.MethodPut, Path: "/browsers/browser-000001/sessions/start", Payload: map[string]any{}},
		{Name: "GetBrowserSession", Method: http.MethodGet, Path: "/browsers/browser-000001/sessions/get?sessionId=session-000001", Payload: nil},
		{Name: "StopBrowserSession", Method: http.MethodPut, Path: "/browsers/browser-000001/sessions/stop?sessionId=session-000001", Payload: map[string]any{}},
		{Name: "StartCodeInterpreterSession", Method: http.MethodPut, Path: "/code-interpreters/code-000001/sessions/start", Payload: map[string]any{}},
		{Name: "InvokeCodeInterpreter", Method: http.MethodPost, Path: "/code-interpreters/code-000001/tools/invoke", Payload: map[string]any{}},
		{Name: "InvokeAgentRuntime", Method: http.MethodPost, Path: "/runtimes/" + runtimePath + "/invocations?accountId=123456789012&qualifier=LATEST", Payload: map[string]any{}},
		{Name: "StopRuntimeSession", Method: http.MethodPost, Path: "/runtimes/" + runtimePath + "/stopruntimesession?qualifier=LATEST", Payload: map[string]any{}},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := agentCoreRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
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

func agentCoreRequest(
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

	req, err := http.NewRequestWithContext(
		ctx,
		method,
		strings.TrimRight(endpoint, "/")+path,
		bytes.NewReader(body),
	)
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "bedrock-agentcore", region, time.Now()); err != nil {
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
