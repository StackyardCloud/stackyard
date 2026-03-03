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

type restCall struct {
	Name    string
	Method  string
	Path    string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	resourceArn := getenv(
		"STACKYARD_OBSERVABILITYADMIN_RESOURCE_ARN",
		"arn:aws:observabilityadmin:us-east-1:123456789012:telemetry-pipeline/stackyard",
	)

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard CloudWatch Observability Admin advanced client using %s\n", endpoint)

	calls := []restCall{
		{Name: "ListTelemetryPipelines", Method: http.MethodPost, Path: "/ListTelemetryPipelines", Payload: map[string]any{}},
		{Name: "ListTelemetryRules", Method: http.MethodPost, Path: "/ListTelemetryRules", Payload: map[string]any{}},
		{Name: "ListS3TableIntegrations", Method: http.MethodPost, Path: "/ListS3TableIntegrations", Payload: map[string]any{}},
		{Name: "GetTelemetryEvaluationStatus", Method: http.MethodPost, Path: "/GetTelemetryEvaluationStatus", Payload: map[string]any{}},
		{Name: "TagResource", Method: http.MethodPost, Path: "/TagResource", Payload: map[string]any{"ResourceArn": resourceArn, "Tags": map[string]any{"stackyard": "true"}}},
		{Name: "ListTagsForResource", Method: http.MethodPost, Path: "/ListTagsForResource", Payload: map[string]any{"ResourceArn": resourceArn}},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	call restCall,
) error {
	status, body, err := observabilityAdminRequest(ctx, endpoint, region, creds, call.Method, call.Path, call.Payload)
	if err != nil {
		return err
	}

	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}

	errType := extractErrorType(body)
	if isNotImplemented(errType, body) {
		logf("%s returned %d (%s): expected while staged plan is in progress", call.Name, status, errType)
		return nil
	}

	return fmt.Errorf("HTTP %d: %s", status, strings.TrimSpace(string(body)))
}

func observabilityAdminRequest(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	method, path string,
	payload map[string]any,
) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
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
	req.Header.Set("Content-Type", "application/json")

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "observabilityadmin", region, time.Now()); err != nil {
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

func isNotImplemented(errType string, body []byte) bool {
	if strings.Contains(strings.ToLower(errType), "notimplemented") {
		return true
	}
	return strings.Contains(strings.ToLower(string(body)), "notimplemented")
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
