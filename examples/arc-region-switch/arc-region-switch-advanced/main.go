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

type actionCall struct {
	Name    string
	Payload map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	planArn := getenv(
		"STACKYARD_ARC_PLAN_ARN",
		"arn:aws:arc-region-switch:us-east-1:123456789012:plan/stackyard-plan",
	)

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard ARC Region Switch advanced client using %s\n", endpoint)

	calls := []actionCall{
		{Name: "ListPlans", Payload: map[string]any{"maxResults": 5}},
		{Name: "GetPlan", Payload: map[string]any{"planArn": planArn}},
		{Name: "ListPlanExecutions", Payload: map[string]any{"planArn": planArn, "maxResults": 5}},
		{Name: "ListTagsForResource", Payload: map[string]any{"resourceArn": planArn}},
	}

	for _, call := range calls {
		status, body, err := arcRegionSwitchRequest(ctx, endpoint, region, creds, call.Name, call.Payload)
		if err != nil {
			exitf("%s request failed: %v", call.Name, err)
		}
		trimmed := strings.TrimSpace(string(body))
		if status <= 0 || status >= 500 {
			exitf("%s returned unexpected HTTP %d: %s", call.Name, status, trimmed)
		}
		logf("%s returned %d", call.Name, status)
	}

	fmt.Println("Done.")
}

func arcRegionSwitchRequest(
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

	url := strings.TrimRight(endpoint, "/") + "/"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "ArcRegionSwitchService."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "arc-region-switch", region, time.Now()); err != nil {
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
