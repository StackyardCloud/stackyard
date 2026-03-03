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

	fmt.Printf("Stackyard Shield Advanced advanced client using %s\n", endpoint)

	calls := []apiCall{
		{Action: "CreateSubscription", Payload: map[string]any{}},
		{Action: "GetSubscriptionState", Payload: map[string]any{}},
		{Action: "CreateProtection", Payload: map[string]any{
			"Name":        "stackyard-protection",
			"ResourceArn": "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/stackyard/0000000000000001",
		}},
		{Action: "DescribeProtection", Payload: map[string]any{"ProtectionId": "protection-00000001"}},
		{Action: "ListProtections", Payload: map[string]any{}},
		{Action: "CreateProtectionGroup", Payload: map[string]any{
			"ProtectionGroupId": "stackyard-protection-group",
			"Pattern":           "ARBITRARY",
			"Aggregation":       "SUM",
			"Members":           []string{"arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/stackyard/0000000000000001"},
		}},
		{Action: "ListProtectionGroups", Payload: map[string]any{}},
		{Action: "DescribeAttack", Payload: map[string]any{"AttackId": "attack-00000001"}},
		{Action: "TagResource", Payload: map[string]any{
			"ResourceArn": "arn:aws:shield:us-east-1:123456789012:protection/protection-00000001",
			"Tags": []map[string]string{
				{"Key": "env", "Value": "coverage"},
				{"Key": "stackyard", "Value": "true"},
			},
		}},
		{Action: "ListTagsForResource", Payload: map[string]any{
			"ResourceArn": "arn:aws:shield:us-east-1:123456789012:protection/protection-00000001",
		}},
		{Action: "UntagResource", Payload: map[string]any{
			"ResourceArn": "arn:aws:shield:us-east-1:123456789012:protection/protection-00000001",
			"TagKeys":     []string{"stackyard"},
		}},
	}

	for _, call := range calls {
		status, body, err := shieldRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
		if err != nil {
			exitf("%s request failed: %v", call.Action, err)
		}
		if status < 200 || status >= 300 {
			exitf("%s returned HTTP %d: %s", call.Action, status, strings.TrimSpace(string(body)))
		}
		logf("%s returned %d", call.Action, status)
	}

	fmt.Println("Done.")
}

func shieldRequest(
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
	req.Header.Set("X-Amz-Target", "AWSShield_20160616."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "shield", region, time.Now()); err != nil {
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
