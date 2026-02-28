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

type requestSpec struct {
	name string
	body map[string]any
}

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-west-2")
	routingControlARN := getenv(
		"STACKYARD_ROUTING_CONTROL_ARN",
		"arn:aws:route53-recovery-control::123456789012:routingcontrol/stackyard-routing-control",
	)

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Routing Control advanced client using %s\n", endpoint)

	calls := []requestSpec{
		{name: "ListRoutingControls", body: map[string]any{}},
		{name: "GetRoutingControlState", body: map[string]any{"RoutingControlArn": routingControlARN}},
		{
			name: "UpdateRoutingControlState",
			body: map[string]any{
				"RoutingControlArn":   routingControlARN,
				"RoutingControlState": "On",
			},
		},
		{
			name: "UpdateRoutingControlStates",
			body: map[string]any{
				"UpdateRoutingControlStateEntries": []map[string]any{
					{
						"RoutingControlArn":   routingControlARN,
						"RoutingControlState": "Off",
					},
				},
			},
		},
	}

	for _, call := range calls {
		status, body, err := routingControlRequest(ctx, endpoint, region, creds, call.name, call.body)
		if err != nil {
			exitf("%s request failed: %v", call.name, err)
		}
		trimmed := strings.TrimSpace(string(body))
		if status <= 0 || status >= 500 {
			exitf("%s returned unexpected HTTP %d: %s", call.name, status, trimmed)
		}
		logf("%s returned %d", call.name, status)
	}

	fmt.Println("Done.")
}

func routingControlRequest(
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
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "ToggleCustomerAPI."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "route53-recovery-cluster", region, time.Now()); err != nil {
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
