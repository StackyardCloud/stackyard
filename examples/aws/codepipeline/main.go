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

func main() {
	endpoint := getenv("STACKYARD_ENDPOINT", "http://localhost:4566")
	region := getenv("AWS_REGION", "us-east-1")
	unique := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000)
	pipelineName := "stackyard-pipeline-" + unique
	resourceARN := "arn:aws:codepipeline:us-east-1:123456789012:" + pipelineName
	webhookName := "stackyard-webhook-" + unique

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard CodePipeline advanced client using %s\n", endpoint)

	mustRun(ctx, endpoint, region, creds, "CreatePipeline", map[string]any{"pipeline": defaultPipeline(pipelineName)})
	mustRun(ctx, endpoint, region, creds, "GetPipelineState", map[string]any{"name": pipelineName})
	mustRun(ctx, endpoint, region, creds, "ListActionTypes", map[string]any{"actionOwnerFilter": "AWS"})
	mustRun(ctx, endpoint, region, creds, "ListRuleTypes", map[string]any{"ruleOwnerFilter": "AWS"})

	mustRun(ctx, endpoint, region, creds, "PutWebhook", map[string]any{
		"webhook": map[string]any{
			"name":           webhookName,
			"targetPipeline": pipelineName,
			"targetAction":   "Source",
			"filters": []map[string]any{{
				"jsonPath":    "$.ref",
				"matchEquals": "refs/heads/main",
			}},
		},
	})
	mustRun(ctx, endpoint, region, creds, "RegisterWebhookWithThirdParty", map[string]any{"webhookName": webhookName})
	mustRun(ctx, endpoint, region, creds, "ListWebhooks", map[string]any{})
	mustRun(ctx, endpoint, region, creds, "DeregisterWebhookWithThirdParty", map[string]any{"webhookName": webhookName})

	mustRun(ctx, endpoint, region, creds, "TagResource", map[string]any{
		"resourceArn": resourceARN,
		"tags":        []map[string]any{{"key": "env", "value": "test"}, {"key": "owner", "value": "stackyard"}},
	})
	mustRun(ctx, endpoint, region, creds, "ListTagsForResource", map[string]any{"resourceArn": resourceARN})
	mustRun(ctx, endpoint, region, creds, "UntagResource", map[string]any{"resourceArn": resourceARN, "tagKeys": []string{"owner"}})

	mustRun(ctx, endpoint, region, creds, "PollForJobs", map[string]any{})
	mustRun(ctx, endpoint, region, creds, "PollForThirdPartyJobs", map[string]any{})
	mustRun(ctx, endpoint, region, creds, "DeleteWebhook", map[string]any{"name": webhookName})
	mustRun(ctx, endpoint, region, creds, "DeletePipeline", map[string]any{"name": pipelineName})

	fmt.Println("Done.")
}

func mustRun(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) {
	if _, err := runCodePipelineAction(ctx, endpoint, region, creds, action, payload); err != nil {
		exitf("%s failed: %v", action, err)
	}
}

func runCodePipelineAction(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (map[string]any, error) {
	status, body, err := codePipelineRequest(ctx, endpoint, region, creds, action, payload)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		trimmed := strings.TrimSpace(string(body))
		if trimmed == "" {
			trimmed = "<empty body>"
		}
		return nil, fmt.Errorf("HTTP %d: %s", status, trimmed)
	}
	logf("%s returned %d", action, status)
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	out := map[string]any{}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode %s response: %w", action, err)
	}
	return out, nil
}

func codePipelineRequest(
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
	req.Header.Set("X-Amz-Target", "CodePipeline_20150709."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "codepipeline", region, time.Now()); err != nil {
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

func defaultPipeline(name string) map[string]any {
	return map[string]any{
		"name":    name,
		"roleArn": "arn:aws:iam::123456789012:role/stackyard-codepipeline",
		"artifactStore": map[string]any{
			"type":     "S3",
			"location": "stackyard-codepipeline-artifacts",
		},
		"stages": []map[string]any{
			{
				"name": "Source",
				"actions": []map[string]any{
					{
						"name": "Source",
						"actionTypeId": map[string]any{
							"category": "Source",
							"owner":    "AWS",
							"provider": "S3",
							"version":  "1",
						},
						"outputArtifacts": []map[string]any{{"name": "SourceArtifact"}},
						"configuration": map[string]any{
							"S3Bucket":    "stackyard-source",
							"S3ObjectKey": "source.zip",
						},
					},
				},
			},
		},
	}
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
