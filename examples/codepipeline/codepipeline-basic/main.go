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
	pipelineName := "stackyard-basic-pipeline-" + unique

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard CodePipeline basic client using %s\n", endpoint)

	if _, err := runCodePipelineAction(ctx, endpoint, region, creds, "CreatePipeline", map[string]any{
		"pipeline": defaultPipeline(pipelineName),
	}); err != nil {
		exitf("CreatePipeline failed: %v", err)
	}

	if _, err := runCodePipelineAction(ctx, endpoint, region, creds, "ListPipelines", map[string]any{"maxResults": 10}); err != nil {
		exitf("ListPipelines failed: %v", err)
	}

	startOut, err := runCodePipelineAction(ctx, endpoint, region, creds, "StartPipelineExecution", map[string]any{
		"name": pipelineName,
	})
	if err != nil {
		exitf("StartPipelineExecution failed: %v", err)
	}

	executionID := stringField(startOut, "pipelineExecutionId")
	if executionID == "" {
		executionID = "exec-missing"
	}

	if _, err := runCodePipelineAction(ctx, endpoint, region, creds, "GetPipelineExecution", map[string]any{
		"pipelineName":        pipelineName,
		"pipelineExecutionId": executionID,
	}); err != nil {
		exitf("GetPipelineExecution failed: %v", err)
	}

	if _, err := runCodePipelineAction(ctx, endpoint, region, creds, "StopPipelineExecution", map[string]any{
		"pipelineName":        pipelineName,
		"pipelineExecutionId": executionID,
	}); err != nil {
		exitf("StopPipelineExecution failed: %v", err)
	}

	if _, err := runCodePipelineAction(ctx, endpoint, region, creds, "DeletePipeline", map[string]any{
		"name": pipelineName,
	}); err != nil {
		exitf("DeletePipeline failed: %v", err)
	}

	fmt.Println("Done.")
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
		return nil, fmt.Errorf("%s returned HTTP %d: %s", action, status, trimmed)
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

func stringField(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok {
		return ""
	}
	asString, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(asString)
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
