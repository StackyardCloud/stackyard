// Application Auto Scaling example: creates a provisioned DynamoDB table,
// registers its read capacity as a scalable target, adds a target tracking
// policy and a scheduled action, lists them, and cleans up.
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
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	table := getenv("STACKYARD_TABLE", fmt.Sprintf("aas-example-%d", time.Now().UnixNano()))
	creds := credentials.NewStaticCredentialsProvider(getenv("AWS_ACCESS_KEY_ID", "stackyard"), getenv("AWS_SECRET_ACCESS_KEY", "stackyard"), "")
	ctx := context.Background()
	fmt.Printf("Stackyard Application Auto Scaling client using %s\n", endpoint)

	call := func(service, target, contentType string, payload map[string]any) map[string]any {
		body, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/", bytes.NewReader(body))
		if err != nil {
			exitf("%s: %v", target, err)
		}
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("X-Amz-Target", target)
		sum := sha256.Sum256(body)
		hash := hex.EncodeToString(sum[:])
		req.Header.Set("X-Amz-Content-Sha256", hash)
		c, err := creds.Retrieve(ctx)
		if err != nil {
			exitf("credentials: %v", err)
		}
		if err := v4.NewSigner().SignHTTP(ctx, aws.Credentials(c), req, hash, service, region, time.Now()); err != nil {
			exitf("sign %s: %v", target, err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			exitf("%s: %v", target, err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode/100 != 2 {
			exitf("%s returned %d: %s", target, resp.StatusCode, raw)
		}
		out := map[string]any{}
		_ = json.Unmarshal(raw, &out)
		return out
	}
	aas := func(op string, payload map[string]any) map[string]any {
		return call("application-autoscaling", "AnyScaleFrontendService."+op, "application/x-amz-json-1.1", payload)
	}

	call("dynamodb", "DynamoDB_20120810.CreateTable", "application/x-amz-json-1.0", map[string]any{
		"TableName":             table,
		"AttributeDefinitions":  []any{map[string]any{"AttributeName": "id", "AttributeType": "S"}},
		"KeySchema":             []any{map[string]any{"AttributeName": "id", "KeyType": "HASH"}},
		"BillingMode":           "PROVISIONED",
		"ProvisionedThroughput": map[string]any{"ReadCapacityUnits": 5, "WriteCapacityUnits": 5},
	})
	key := map[string]any{"ServiceNamespace": "dynamodb", "ResourceId": "table/" + table, "ScalableDimension": "dynamodb:table:ReadCapacityUnits"}
	with := func(extra map[string]any) map[string]any {
		out := map[string]any{}
		for k, v := range key {
			out[k] = v
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}

	target := aas("RegisterScalableTarget", with(map[string]any{"MinCapacity": 5, "MaxCapacity": 50}))
	fmt.Printf("registered %v\n", target["ScalableTargetARN"])

	policy := aas("PutScalingPolicy", with(map[string]any{
		"PolicyName": "read-70",
		"PolicyType": "TargetTrackingScaling",
		"TargetTrackingScalingPolicyConfiguration": map[string]any{
			"TargetValue":                   70,
			"PredefinedMetricSpecification": map[string]any{"PredefinedMetricType": "DynamoDBReadCapacityUtilization"},
		},
	}))
	alarms, _ := policy["Alarms"].([]any)
	fmt.Printf("target tracking policy %v with %d CloudWatch alarms\n", policy["PolicyARN"], len(alarms))

	aas("PutScheduledAction", with(map[string]any{
		"ScheduledActionName":  "business-hours",
		"Schedule":             "cron(0 8 ? * MON-FRI *)",
		"Timezone":             "America/New_York",
		"ScalableTargetAction": map[string]any{"MinCapacity": 20, "MaxCapacity": 50},
	}))

	policies := aas("DescribeScalingPolicies", map[string]any{"ServiceNamespace": "dynamodb", "ResourceId": "table/" + table})["ScalingPolicies"].([]any)
	actions := aas("DescribeScheduledActions", map[string]any{"ServiceNamespace": "dynamodb", "ResourceId": "table/" + table})["ScheduledActions"].([]any)
	if len(policies) != 1 || len(actions) != 1 {
		exitf("expected 1 policy and 1 scheduled action, got %d and %d", len(policies), len(actions))
	}
	fmt.Printf("found %d policy and %d scheduled action\n", len(policies), len(actions))

	// Deregistering removes the target's policies and scheduled actions.
	aas("DeregisterScalableTarget", key)
	if left := aas("DescribeScalingPolicies", map[string]any{"ServiceNamespace": "dynamodb", "ResourceId": "table/" + table})["ScalingPolicies"].([]any); len(left) != 0 {
		exitf("policies left after deregister: %d", len(left))
	}
	fmt.Println("deregistered; Application Auto Scaling example complete")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
