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

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Data Pipeline advanced client using %s\n", endpoint)

	pipelineID := runJSONCall(ctx, endpoint, region, creds, "CreatePipeline", map[string]any{
		"name":     "advanced-datapipeline",
		"uniqueId": "advanced-datapipeline-unique",
	})["pipelineId"].(string)

	callAndCheck(ctx, endpoint, region, creds, "AddTags", map[string]any{
		"pipelineId": pipelineID,
		"tags": []map[string]any{
			{"key": "env", "value": "advanced"},
			{"key": "owner", "value": "qa"},
		},
	})
	callAndCheck(ctx, endpoint, region, creds, "PutPipelineDefinition", map[string]any{
		"pipelineId": pipelineID,
		"pipelineObjects": []map[string]any{
			{
				"id":   "DefaultActivity",
				"name": "DefaultActivity",
				"fields": []map[string]any{
					{"key": "type", "stringValue": "ShellCommandActivity"},
				},
			},
		},
		"parameterObjects": []map[string]any{
			{
				"id": "myParam",
				"attributes": []map[string]any{
					{"key": "type", "stringValue": "String"},
				},
			},
		},
		"parameterValues": []map[string]any{
			{"id": "myParam", "stringValue": "advanced"},
		},
	})
	callAndCheck(ctx, endpoint, region, creds, "ValidatePipelineDefinition", map[string]any{"pipelineId": pipelineID})
	callAndCheck(ctx, endpoint, region, creds, "GetPipelineDefinition", map[string]any{"pipelineId": pipelineID})
	callAndCheck(ctx, endpoint, region, creds, "QueryObjects", map[string]any{"pipelineId": pipelineID})
	callAndCheck(ctx, endpoint, region, creds, "DescribeObjects", map[string]any{
		"pipelineId": pipelineID,
		"objectIds":  []string{"DefaultActivity"},
	})
	callAndCheck(ctx, endpoint, region, creds, "EvaluateExpression", map[string]any{
		"pipelineId": pipelineID,
		"expression": "#{@scheduledStartTime}",
	})
	callAndCheck(ctx, endpoint, region, creds, "ActivatePipeline", map[string]any{"pipelineId": pipelineID})
	callAndCheck(ctx, endpoint, region, creds, "PollForTask", map[string]any{
		"workerGroup": "default-worker-group",
		"hostname":    "advanced-worker",
		"instanceIdentity": map[string]any{
			"document":  "advanced",
			"signature": "advanced",
		},
	})
	callAndCheck(ctx, endpoint, region, creds, "ReportTaskRunnerHeartbeat", map[string]any{
		"pipelineId":  pipelineID,
		"taskId":      "task-000001",
		"workerGroup": "default-worker-group",
	})
	callAndCheck(ctx, endpoint, region, creds, "ReportTaskProgress", map[string]any{
		"pipelineId": pipelineID,
		"taskId":     "task-000001",
		"fields": []map[string]any{
			{"key": "percentComplete", "stringValue": "100"},
		},
	})
	callAndCheck(ctx, endpoint, region, creds, "SetTaskStatus", map[string]any{
		"pipelineId": pipelineID,
		"taskId":     "task-000001",
		"taskStatus": "FINISHED",
	})
	callAndCheck(ctx, endpoint, region, creds, "SetStatus", map[string]any{
		"pipelineId": pipelineID,
		"objectIds":  []string{"DefaultActivity"},
		"status":     "FINISHED",
	})
	callAndCheck(ctx, endpoint, region, creds, "DescribePipelines", map[string]any{
		"pipelineIds": []string{pipelineID},
	})
	callAndCheck(ctx, endpoint, region, creds, "ListPipelines", map[string]any{})
	callAndCheck(ctx, endpoint, region, creds, "RemoveTags", map[string]any{
		"pipelineId": pipelineID,
		"tagKeys":    []string{"owner"},
	})
	callAndCheck(ctx, endpoint, region, creds, "DeactivatePipeline", map[string]any{"pipelineId": pipelineID})
	callAndCheck(ctx, endpoint, region, creds, "DeletePipeline", map[string]any{"pipelineId": pipelineID})

	fmt.Println("Done.")
}

func runJSONCall(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) map[string]any {
	status, body, err := dataPipelineRequest(ctx, endpoint, region, creds, action, payload)
	if err != nil {
		exitf("%s request failed: %v", action, err)
	}
	if status != http.StatusOK {
		exitf("%s expected 200, got %d: %s", action, status, strings.TrimSpace(string(body)))
	}
	if strings.Contains(string(body), "NotImplemented") {
		exitf("%s returned NotImplemented: %s", action, strings.TrimSpace(string(body)))
	}

	var out map[string]any
	if len(bytes.TrimSpace(body)) == 0 {
		return map[string]any{}
	}
	if err := json.Unmarshal(body, &out); err != nil {
		exitf("%s returned invalid JSON: %v (body=%s)", action, err, strings.TrimSpace(string(body)))
	}
	if out == nil {
		out = map[string]any{}
	}
	logf("%s returned %d", action, status)
	return out
}

func callAndCheck(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) {
	_ = runJSONCall(ctx, endpoint, region, creds, action, payload)
}

func dataPipelineRequest(
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
	req.Header.Set("X-Amz-Target", "DataPipeline."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "datapipeline", region, time.Now()); err != nil {
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
