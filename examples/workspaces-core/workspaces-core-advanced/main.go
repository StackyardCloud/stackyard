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

	fmt.Printf("Stackyard WorkSpaces Core advanced client using %s\n", endpoint)

	for _, call := range []rpcCall{
		{Name: "DescribeAccount", Action: "DescribeAccount", Payload: map[string]any{}},
		{Name: "ListAvailableManagementCidrRanges", Action: "ListAvailableManagementCidrRanges", Payload: map[string]any{}},
		{Name: "RegisterWorkspaceDirectory", Action: "RegisterWorkspaceDirectory", Payload: map[string]any{"DirectoryId": "d-000001"}},
		{Name: "DescribeWorkspaceDirectories", Action: "DescribeWorkspaceDirectories", Payload: map[string]any{"Limit": 10}},
		{Name: "CreateWorkspaceBundle", Action: "CreateWorkspaceBundle", Payload: map[string]any{"BundleName": "core-advanced-bundle"}},
		{Name: "DescribeWorkspaceBundles", Action: "DescribeWorkspaceBundles", Payload: map[string]any{"Limit": 10}},
		{Name: "CreateWorkspaceImage", Action: "CreateWorkspaceImage", Payload: map[string]any{"Name": "core-advanced-image"}},
		{Name: "DescribeWorkspaceImages", Action: "DescribeWorkspaceImages", Payload: map[string]any{"MaxResults": 10}},
		{Name: "CreateWorkspaces", Action: "CreateWorkspaces", Payload: map[string]any{"Workspaces": []map[string]any{{"DirectoryId": "d-000001", "BundleId": "wsb-000001", "UserName": "core-advanced-user"}}}},
		{Name: "DescribeWorkspaces", Action: "DescribeWorkspaces", Payload: map[string]any{"Limit": 10}},
		{Name: "CreateTags", Action: "CreateTags", Payload: map[string]any{"ResourceId": "ws-000001", "Tags": []map[string]string{{"Key": "env", "Value": "advanced"}, {"Key": "owner", "Value": "qa"}}}},
		{Name: "DescribeTags", Action: "DescribeTags", Payload: map[string]any{"ResourceId": "ws-000001"}},
		{Name: "DeleteTags", Action: "DeleteTags", Payload: map[string]any{"ResourceId": "ws-000001", "TagKeys": []string{"owner"}}},
	} {
		if err := runCall(ctx, endpoint, region, creds, call); err != nil {
			exitf("%s failed: %v", call.Name, err)
		}
	}

	fmt.Println("Done.")
}

func runCall(ctx context.Context, endpoint, region string, creds aws.CredentialsProvider, call rpcCall) error {
	status, body, err := workspacesCoreRequest(ctx, endpoint, region, creds, call.Action, call.Payload)
	if err != nil {
		return err
	}
	if status >= 200 && status < 300 {
		logf("%s returned %d", call.Name, status)
		return nil
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		trimmed = "<empty body>"
	}
	return fmt.Errorf("HTTP %d: %s", status, trimmed)
}

func workspacesCoreRequest(
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
	req.Header.Set("X-Amz-Target", "WorkspacesService."+action)

	credValue, err := creds.Retrieve(ctx)
	if err != nil {
		return 0, nil, err
	}
	signer := v4.NewSigner()
	if err := signer.SignHTTP(ctx, credValue, req, hashSHA256(body), "workspaces", region, time.Now()); err != nil {
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
