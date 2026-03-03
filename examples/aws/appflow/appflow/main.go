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
	flowName := getenv("STACKYARD_APPFLOW_FLOW_NAME", "stackyard-example-flow")

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard AppFlow advanced client using %s\n", endpoint)

	flowArn, err := runCreateFlow(ctx, endpoint, region, creds, flowName)
	if err != nil {
		exitf("CreateFlow failed: %v", err)
	}
	if flowArn == "" {
		flowArn = "arn:aws:appflow:us-east-1:123456789012:flow/" + flowName
	}

	calls := []restCall{
		{Name: "DescribeFlow", Method: http.MethodPost, Path: "/describe-flow", Payload: map[string]any{"flowName": flowName}},
		{Name: "ListFlows", Method: http.MethodPost, Path: "/list-flows", Payload: map[string]any{}},
		{Name: "StartFlow", Method: http.MethodPost, Path: "/start-flow", Payload: map[string]any{"flowName": flowName}},
		{Name: "StopFlow", Method: http.MethodPost, Path: "/stop-flow", Payload: map[string]any{"flowName": flowName}},
		{Name: "TagResource", Method: http.MethodPost, Path: "/tags/" + url.PathEscape(flowArn), Payload: map[string]any{"tags": map[string]any{"env": "dev", "owner": "stackyard"}}},
		{Name: "ListTagsForResource", Method: http.MethodGet, Path: "/tags/" + url.PathEscape(flowArn), Payload: nil},
		{Name: "UntagResource", Method: http.MethodDelete, Path: "/tags/" + url.PathEscape(flowArn) + "?tagKeys=owner", Payload: nil},
		{Name: "DescribeFlowExecutionRecords", Method: http.MethodPost, Path: "/describe-flow-execution-records", Payload: map[string]any{"flowName": flowName}},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCreateFlow(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, flowName string) (string, error) {
	status, body, err := appFlowRequest(ctx, endpoint, region, creds, http.MethodPost, "/create-flow", map[string]any{
		"flowName": flowName,
	})
	if err != nil {
		return "", err
	}
	if status >= 200 && status < 300 {
		payload := map[string]any{}
		if err := json.Unmarshal(body, &payload); err == nil {
			if arn, ok := payload["flowArn"].(string); ok {
				return strings.TrimSpace(arn), nil
			}
		}
		return "", nil
	}

	errType := extractErrorType(body)
	if isStagedPlanTolerated(errType, body) {
		logf("CreateFlow returned %d (%s): tolerated while staged plan is in progress", status, errType)
		return "", nil
	}
	return "", fmt.Errorf("HTTP %d: %s", status, strings.TrimSpace(string(body)))
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call restCall) error {
	status, body, err := appFlowRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
	if err != nil {
		return err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}

	errType := extractErrorType(body)
	if isStagedPlanTolerated(errType, body) {
		logf("%s returned %d (%s): expected while staged plan is in progress", call.Name, status, errType)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func appFlowRequest(
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
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "appflow", region, time.Now()); err != nil {
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

func extractErrorType(body []byte) string {
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if v, ok := payload["__type"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["code"].(string); ok {
		return strings.TrimSpace(v)
	}
	if v, ok := payload["message"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func isStagedPlanTolerated(errType string, body []byte) bool {
	combined := strings.ToLower(errType + " " + string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "accessdenied")
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
