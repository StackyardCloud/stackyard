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
	resourceName := "stackyard-cloudcontrol-" + unique
	desiredState := fmt.Sprintf(`{"BucketName":"%s"}`, resourceName)

	ctx := context.Background()
	creds := credentials.NewStaticCredentialsProvider(
		getenv("AWS_ACCESS_KEY_ID", "stackyard"),
		getenv("AWS_SECRET_ACCESS_KEY", "stackyard"),
		"",
	)

	fmt.Printf("Stackyard Cloud Control API advanced client using %s\n", endpoint)

	createOut, err := runCloudControlAction(ctx, endpoint, region, creds, "CreateResource", map[string]any{
		"TypeName":     "AWS::S3::Bucket",
		"DesiredState": desiredState,
		"ClientToken":  "advanced-create-" + unique,
	})
	if err != nil {
		exitf("CreateResource failed: %v", err)
	}

	identifier, err := progressValue(createOut, "Identifier")
	if err != nil {
		exitf("CreateResource missing ProgressEvent.Identifier: %v", err)
	}
	createRequestToken, err := progressValue(createOut, "RequestToken")
	if err != nil {
		exitf("CreateResource missing ProgressEvent.RequestToken: %v", err)
	}

	patchDoc := fmt.Sprintf(`[{"op":"replace","path":"/BucketName","value":"%s-updated"}]`, resourceName)
	updateOut, err := runCloudControlAction(ctx, endpoint, region, creds, "UpdateResource", map[string]any{
		"TypeName":      "AWS::S3::Bucket",
		"Identifier":    identifier,
		"PatchDocument": patchDoc,
		"ClientToken":   "advanced-update-" + unique,
	})
	if err != nil {
		exitf("UpdateResource failed: %v", err)
	}
	updateRequestToken, err := progressValue(updateOut, "RequestToken")
	if err != nil {
		exitf("UpdateResource missing ProgressEvent.RequestToken: %v", err)
	}

	if _, err := runCloudControlAction(ctx, endpoint, region, creds, "GetResource", map[string]any{
		"TypeName":   "AWS::S3::Bucket",
		"Identifier": identifier,
	}); err != nil {
		exitf("GetResource failed: %v", err)
	}

	if _, err := runCloudControlAction(ctx, endpoint, region, creds, "GetResourceRequestStatus", map[string]any{
		"RequestToken": createRequestToken,
	}); err != nil {
		exitf("GetResourceRequestStatus (create token) failed: %v", err)
	}

	if _, err := runCloudControlAction(ctx, endpoint, region, creds, "ListResourceRequests", map[string]any{
		"MaxResults": 20,
		"ResourceRequestStatusFilter": map[string]any{
			"TypeName":          "AWS::S3::Bucket",
			"Operations":        []string{"CREATE", "UPDATE"},
			"OperationStatuses": []string{"SUCCESS"},
		},
	}); err != nil {
		exitf("ListResourceRequests failed: %v", err)
	}

	if _, err := runCloudControlAction(ctx, endpoint, region, creds, "CancelResourceRequest", map[string]any{
		"RequestToken": updateRequestToken,
	}); err != nil {
		exitf("CancelResourceRequest failed: %v", err)
	}

	if _, err := runCloudControlAction(ctx, endpoint, region, creds, "ListResources", map[string]any{
		"TypeName":   "AWS::S3::Bucket",
		"MaxResults": 20,
	}); err != nil {
		exitf("ListResources failed: %v", err)
	}

	if _, err := runCloudControlAction(ctx, endpoint, region, creds, "DeleteResource", map[string]any{
		"TypeName":    "AWS::S3::Bucket",
		"Identifier":  identifier,
		"ClientToken": "advanced-delete-" + unique,
	}); err != nil {
		exitf("DeleteResource failed: %v", err)
	}

	fmt.Println("Done.")
}

func runCloudControlAction(
	ctx context.Context,
	endpoint, region string,
	creds aws.CredentialsProvider,
	action string,
	payload map[string]any,
) (map[string]any, error) {
	status, body, err := cloudControlRequest(ctx, endpoint, region, creds, action, payload)
	if err != nil {
		return nil, err
	}
	if err := expectOK(action, status, body); err != nil {
		return nil, err
	}
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

func cloudControlRequest(
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
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")
	req.Header.Set("X-Amz-Target", "CloudApiService."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "cloudcontrolapi", region, time.Now()); err != nil {
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

func progressValue(payload map[string]any, key string) (string, error) {
	progressRaw, ok := payload["ProgressEvent"]
	if !ok {
		return "", fmt.Errorf("missing ProgressEvent")
	}
	progress, ok := progressRaw.(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid ProgressEvent")
	}
	value, ok := progress[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("missing ProgressEvent.%s", key)
	}
	return value, nil
}

func expectOK(action string, status int, body []byte) error {
	if status >= 200 && status < 300 {
		logf("%s returned %d", action, status)
		return nil
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("%s returned HTTP %d: %s", action, status, trimmed)
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
