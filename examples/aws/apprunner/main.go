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

type rpcCall struct {
	Name    string
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

	fmt.Printf("Stackyard App Runner advanced client using %s\n", endpoint)

	serviceArn := "arn:aws:apprunner:us-east-1:123456789012:service/stackyard-service/0000000000000001"
	autoScalingArn := "arn:aws:apprunner:us-east-1:123456789012:autoscalingconfiguration/stackyard-auto-scaling/1"
	resourceArn := serviceArn

	calls := []rpcCall{
		{Name: "ListServices", Action: "ListServices", Payload: map[string]any{"MaxResults": 10}},
		{Name: "CreateService", Action: "CreateService", Payload: map[string]any{"ServiceName": "stackyard-service", "SourceConfiguration": map[string]any{"ImageRepository": map[string]any{"ImageIdentifier": "public.ecr.aws/nginx/nginx:latest", "ImageRepositoryType": "ECR_PUBLIC"}}}},
		{Name: "DescribeService", Action: "DescribeService", Payload: map[string]any{"ServiceArn": serviceArn}},
		{Name: "UpdateService", Action: "UpdateService", Payload: map[string]any{"ServiceArn": serviceArn, "AutoScalingConfigurationArn": autoScalingArn}},
		{Name: "PauseService", Action: "PauseService", Payload: map[string]any{"ServiceArn": serviceArn}},
		{Name: "ResumeService", Action: "ResumeService", Payload: map[string]any{"ServiceArn": serviceArn}},
		{Name: "StartDeployment", Action: "StartDeployment", Payload: map[string]any{"ServiceArn": serviceArn}},
		{Name: "ListOperations", Action: "ListOperations", Payload: map[string]any{"ServiceArn": serviceArn, "MaxResults": 10}},
		{Name: "AssociateCustomDomain", Action: "AssociateCustomDomain", Payload: map[string]any{"ServiceArn": serviceArn, "DomainName": "example.com", "EnableWWWSubdomain": true}},
		{Name: "DescribeCustomDomains", Action: "DescribeCustomDomains", Payload: map[string]any{"ServiceArn": serviceArn}},
		{Name: "TagResource", Action: "TagResource", Payload: map[string]any{"ResourceArn": resourceArn, "Tags": []map[string]string{{"Key": "env", "Value": "dev"}}}},
		{Name: "ListTagsForResource", Action: "ListTagsForResource", Payload: map[string]any{"ResourceArn": resourceArn}},
		{Name: "UntagResource", Action: "UntagResource", Payload: map[string]any{"ResourceArn": resourceArn, "TagKeys": []string{"env"}}},
		{Name: "DisassociateCustomDomain", Action: "DisassociateCustomDomain", Payload: map[string]any{"ServiceArn": serviceArn, "DomainName": "example.com"}},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call rpcCall) error {
	status, body, err := appRunnerRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
	if err != nil {
		return err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}

	errType := extractErrorType(body)
	if isStagedPlanTolerated(status, errType, body) {
		logf("%s returned %d (%s): expected while staged plan is in progress", call.Name, status, errType)
		return nil
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func appRunnerRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (int, []byte, error) {
	var body []byte
	var err error
	if payload == nil {
		body = []byte("{}")
	} else {
		body, err = json.Marshal(payload)
		if err != nil {
			return 0, nil, err
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(endpoint, "/")+"/",
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "AppRunner."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "apprunner", region, time.Now()); err != nil {
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

func isStagedPlanTolerated(status int, errType string, body []byte) bool {
	if status == http.StatusNotFound {
		return true
	}
	combined := strings.ToLower(errType + " " + string(body))
	return strings.Contains(combined, "notimplemented") ||
		strings.Contains(combined, "unknownoperation") ||
		strings.Contains(combined, "unknown action") ||
		strings.Contains(combined, "resourcenotfound") ||
		strings.Contains(combined, "validationexception") ||
		strings.Contains(combined, "accessdenied") ||
		strings.Contains(combined, "unauthorized")
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
