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

	fmt.Printf("Stackyard Translate advanced client using %s\n", endpoint)

	calls := []rpcCall{
		{Name: "ListLanguages", Action: "ListLanguages", Payload: map[string]any{}},
		{Name: "TranslateText", Action: "TranslateText", Payload: map[string]any{"Text": "hello world", "SourceLanguageCode": "en", "TargetLanguageCode": "es"}},
		{Name: "TranslateDocument", Action: "TranslateDocument", Payload: map[string]any{"Document": map[string]any{"Content": "aGVsbG8=", "ContentType": "text/plain"}, "SourceLanguageCode": "en", "TargetLanguageCode": "fr"}},
		{Name: "ImportTerminology", Action: "ImportTerminology", Payload: map[string]any{"Name": "advanced-terminology", "Description": "advanced terminology"}},
		{Name: "GetTerminology", Action: "GetTerminology", Payload: map[string]any{"Name": "advanced-terminology"}},
		{Name: "ListTerminologies", Action: "ListTerminologies", Payload: map[string]any{}},
		{Name: "CreateParallelData", Action: "CreateParallelData", Payload: map[string]any{"Name": "advanced-parallel-data", "Description": "advanced parallel data"}},
		{Name: "GetParallelData", Action: "GetParallelData", Payload: map[string]any{"Name": "advanced-parallel-data"}},
		{Name: "ListParallelData", Action: "ListParallelData", Payload: map[string]any{}},
		{Name: "UpdateParallelData", Action: "UpdateParallelData", Payload: map[string]any{"Name": "advanced-parallel-data", "Description": "advanced parallel data updated"}},
	}

	for _, call := range calls {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	jobID := ""
	startPayload, err := runCallWithPayload(ctx, endpoint, region, creds, rpcCall{
		Name:   "StartTextTranslationJob",
		Action: "StartTextTranslationJob",
		Payload: map[string]any{
			"JobName":            "advanced-translate-job",
			"ClientToken":        "advanced-token-0001",
			"SourceLanguageCode": "en",
			"TargetLanguageCode": "de",
			"InputDataConfig":    map[string]any{"S3Uri": "s3://stackyard/input/"},
			"OutputDataConfig":   map[string]any{"S3Uri": "s3://stackyard/output/"},
		},
	})
	if err != nil {
		exitf("StartTextTranslationJob failed: %v", err)
	}
	jobID = payloadString(startPayload, "JobId")
	if strings.TrimSpace(jobID) == "" {
		exitf("StartTextTranslationJob did not return JobId")
	}

	for _, call := range []rpcCall{
		{Name: "DescribeTextTranslationJob", Action: "DescribeTextTranslationJob", Payload: map[string]any{"JobId": jobID}},
		{Name: "ListTextTranslationJobs", Action: "ListTextTranslationJobs", Payload: map[string]any{}},
		{Name: "StopTextTranslationJob", Action: "StopTextTranslationJob", Payload: map[string]any{"JobId": jobID}},
		{Name: "TagResource", Action: "TagResource", Payload: map[string]any{"ResourceArn": "arn:aws:translate:us-east-1:123456789012:terminology/advanced-terminology", "Tags": []map[string]string{{"Key": "env", "Value": "advanced"}, {"Key": "owner", "Value": "qa"}}}},
		{Name: "ListTagsForResource", Action: "ListTagsForResource", Payload: map[string]any{"ResourceArn": "arn:aws:translate:us-east-1:123456789012:terminology/advanced-terminology"}},
		{Name: "UntagResource", Action: "UntagResource", Payload: map[string]any{"ResourceArn": "arn:aws:translate:us-east-1:123456789012:terminology/advanced-terminology", "TagKeys": []string{"owner"}}},
		{Name: "DeleteParallelData", Action: "DeleteParallelData", Payload: map[string]any{"Name": "advanced-parallel-data"}},
		{Name: "DeleteTerminology", Action: "DeleteTerminology", Payload: map[string]any{"Name": "advanced-terminology"}},
	} {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call rpcCall) error {
	_, err := runCallWithPayload(ctx, endpoint, region, creds, call)
	return err
}

func runCallWithPayload(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call rpcCall) (map[string]any, error) {
	status, body, err := translateRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
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
	logf("%s returned %d", call.Name, status)

	payload := map[string]any{}
	if len(strings.TrimSpace(string(body))) == 0 {
		return payload, nil
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}
	return payload, nil
}

func translateRequest(
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
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "AWSShineFrontendService_20170701."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "translate", region, time.Now()); err != nil {
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

func payloadString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
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
