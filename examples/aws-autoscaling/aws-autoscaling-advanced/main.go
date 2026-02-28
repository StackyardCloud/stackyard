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

	planName := "stackyard-example-plan"
	resourceID := "autoScalingGroup/stackyard-example-asg"

	calls := []apiCall{
		{
			Action: "CreateScalingPlan",
			Payload: map[string]any{
				"ScalingPlanName": planName,
				"ApplicationSource": map[string]any{
					"CloudFormationStackARN": "arn:aws:cloudformation:us-east-1:123456789012:stack/stackyard-example/1",
				},
				"ScalingInstructions": []any{
					map[string]any{
						"ServiceNamespace":  "autoscaling",
						"ResourceId":        resourceID,
						"ScalableDimension": "autoscaling:autoScalingGroup:DesiredCapacity",
						"MinCapacity":       1,
						"MaxCapacity":       6,
						"TargetTrackingConfigurations": []any{
							map[string]any{
								"PredefinedScalingMetricSpecification": map[string]any{
									"PredefinedScalingMetricType": "ASGAverageCPUUtilization",
								},
								"TargetValue": 50.0,
							},
						},
					},
				},
			},
		},
		{
			Action: "DescribeScalingPlans",
			Payload: map[string]any{
				"ScalingPlanNames": []any{planName},
			},
		},
		{
			Action: "DescribeScalingPlanResources",
			Payload: map[string]any{
				"ScalingPlanName":    planName,
				"ScalingPlanVersion": 1,
			},
		},
		{
			Action: "GetScalingPlanResourceForecastData",
			Payload: map[string]any{
				"ScalingPlanName":    planName,
				"ScalingPlanVersion": 1,
				"ServiceNamespace":   "autoscaling",
				"ResourceId":         resourceID,
				"ScalableDimension":  "autoscaling:autoScalingGroup:DesiredCapacity",
				"ForecastDataType":   "CapacityForecast",
				"StartTime":          time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339),
				"EndTime":            time.Now().UTC().Add(4 * time.Hour).Format(time.RFC3339),
			},
		},
		{
			Action: "UpdateScalingPlan",
			Payload: map[string]any{
				"ScalingPlanName":    planName,
				"ScalingPlanVersion": 1,
				"ScalingInstructions": []any{
					map[string]any{
						"ServiceNamespace":  "autoscaling",
						"ResourceId":        resourceID,
						"ScalableDimension": "autoscaling:autoScalingGroup:DesiredCapacity",
						"MinCapacity":       2,
						"MaxCapacity":       8,
					},
				},
			},
		},
		{
			Action: "DeleteScalingPlan",
			Payload: map[string]any{
				"ScalingPlanName":    planName,
				"ScalingPlanVersion": 2,
			},
		},
	}

	fmt.Printf("Stackyard AWS Auto Scaling advanced client using %s\n", endpoint)
	for _, call := range calls {
		status, body, err := autoScalingPlansRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
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

func autoScalingPlansRequest(
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

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(endpoint, "/")+"/",
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AnyScaleScalingPlannerFrontendService."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "autoscaling-plans", region, time.Now()); err != nil {
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
